// Package store keeps what the runtime shows pages from: the program of the
// site, the record of every page and of every shared block, and the values
// of blocks by their content. All of it is one SQLite file, so that a server
// reads one consistent state while an update writes the next one.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/record"
)

const schema = `
CREATE TABLE IF NOT EXISTS meta   (name TEXT PRIMARY KEY, value BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS page   (path TEXT PRIMARY KEY, hash BLOB NOT NULL, record BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS shared (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, hash BLOB NOT NULL, record BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS value  (hash BLOB PRIMARY KEY, data BLOB NOT NULL) WITHOUT ROWID;
`

// Store is an open store.
type Store struct {
	// Reading and writing go through separate connections: a read sees the
	// state at its start, and an update takes the right to write at its
	// start, so that two updates go one after the other and neither fails
	// halfway for the other's sake.
	read, write *sql.DB
	reads       [2]queries // the statements of reading, for either connection
}

// queries are the statements that read one thing by its key.
type queries struct{ meta, page, shared, value *sql.Stmt }

func prepare(db *sql.DB) (q queries, err error) {
	for statement, query := range map[**sql.Stmt]string{
		&q.meta:   "SELECT value FROM meta WHERE name = ?",
		&q.page:   "SELECT record FROM page WHERE path = ?",
		&q.shared: "SELECT record FROM shared WHERE id = ?",
		&q.value:  "SELECT data FROM value WHERE hash = ?",
	} {
		if *statement, err = db.Prepare(query); err != nil {
			return q, err
		}
	}
	return q, nil
}

// Open opens the store at path and creates it if it is not there.
func Open(path string) (*Store, error) {
	open := func(lock string) (*sql.DB, error) {
		options := url.Values{"_pragma": {"busy_timeout(10000)", "journal_mode(WAL)", "synchronous(NORMAL)"}, "_txlock": {lock}}
		return sql.Open("sqlite", "file:"+path+"?"+options.Encode())
	}
	s := &Store{}
	var err error
	if s.write, err = open("immediate"); err != nil {
		return nil, err
	}
	s.write.SetMaxOpenConns(1)
	if _, err = s.write.Exec(schema); err != nil {
		s.write.Close()
		return nil, fmt.Errorf("store %s: %w", path, err)
	}
	if s.read, err = open("deferred"); err != nil {
		s.write.Close()
		return nil, err
	}
	for i, db := range []*sql.DB{s.read, s.write} {
		if s.reads[i], err = prepare(db); err != nil {
			s.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close closes the store.
func (s *Store) Close() error {
	return errors.Join(s.read.Close(), s.write.Close())
}

// Names of what meta holds.
const (
	metaBuild   = "build"   // which compiled project the records belong to
	metaSite    = "site"    // the program of that project
	metaVersion = "version" // grows with every update
)

// reader reads through one transaction, so everything it returns belongs to
// one state of the store.
type reader struct {
	tx *sql.Tx
	queries
}

func (r *reader) blob(statement *sql.Stmt, key any) ([]byte, bool, error) {
	var data []byte
	err := statement.QueryRow(key).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return data, err == nil, err
}

// Build returns the name of the compiled project the records were computed
// by, and its program. The name is empty for a store that holds nothing yet.
func (r *reader) Build() (id string, site []byte, err error) {
	build, found, err := r.blob(r.meta, metaBuild)
	if err != nil || !found {
		return "", nil, err
	}
	site, _, err = r.blob(r.meta, metaSite)
	return string(build), site, err
}

// Version returns a number that grows with every update of the store.
func (r *reader) Version() (int64, error) {
	var version int64
	err := r.meta.QueryRow(metaVersion).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return version, err
}

// Page returns the record of the page at path.
func (r *reader) Page(path string) (data []byte, found bool, err error) {
	return r.blob(r.page, path)
}

// Shared returns the record of a shared block.
func (r *reader) Shared(id int64) (data []byte, found bool, err error) {
	return r.blob(r.shared, id)
}

// Values returns the values kept under a name.
func (r *reader) Values(hash record.Hash) (data []byte, found bool, err error) {
	return r.blob(r.value, hash[:])
}

// Pages calls visit with the record of every page.
func (r *reader) Pages(visit func(path string, record []byte)) error {
	rows, err := r.tx.Query("SELECT path, record FROM page")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		var data []byte
		if err := rows.Scan(&path, &data); err != nil {
			return err
		}
		visit(path, data)
	}
	return rows.Err()
}

// SharedBlocks calls visit with the record of every shared block.
func (r *reader) SharedBlocks(visit func(id int64, record []byte)) error {
	rows, err := r.tx.Query("SELECT id, record FROM shared")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		visit(id, data)
	}
	return rows.Err()
}

// within returns statements that run inside a transaction.
func (q queries) within(tx *sql.Tx) reader {
	return reader{tx, queries{tx.Stmt(q.meta), tx.Stmt(q.page), tx.Stmt(q.shared), tx.Stmt(q.value)}}
}

// Version returns the number of the latest update, as of now.
func (s *Store) Version() (int64, error) {
	return (&reader{queries: s.reads[0]}).Version()
}

// Values returns the values kept under a name, as of now. Values never
// change under their name; they only go away when no record names them.
func (s *Store) Values(hash record.Hash) (data []byte, found bool, err error) {
	return (&reader{queries: s.reads[0]}).Values(hash)
}

// View is one state of the store being read.
type View struct{ reader }

// View runs read with one state of the store: an update that ends meanwhile
// is not seen by it.
func (s *Store) View(read func(*View) error) error {
	tx, err := s.read.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return read(&View{s.reads[0].within(tx)})
}

// Update is a change of the store being made. Nothing of it is seen by
// anyone else until Commit, and nothing remains of it after Rollback.
type Update struct {
	reader
	putValue, putPage, putShared *sql.Stmt
}

// Update starts a change. It waits for an update that is under way.
func (s *Store) Update() (*Update, error) {
	tx, err := s.write.Begin()
	if err != nil {
		return nil, err
	}
	u := &Update{reader: s.reads[1].within(tx)}
	for statement, query := range map[**sql.Stmt]string{
		&u.putValue:  "INSERT OR IGNORE INTO value (hash, data) VALUES (?, ?)",
		&u.putPage:   "INSERT INTO page (path, hash, record) VALUES (?, ?, ?) ON CONFLICT (path) DO UPDATE SET hash = excluded.hash, record = excluded.record",
		&u.putShared: "UPDATE shared SET hash = ?, record = ? WHERE id = ?",
	} {
		if *statement, err = tx.Prepare(query); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	return u, nil
}

// Kept is a record as the store has it: under which number, for a shared
// block, and with which content.
type Kept struct {
	ID   int64
	Hash record.Hash
}

// KeptPages returns the content of the record of every page.
func (u *Update) KeptPages() (map[string]record.Hash, error) {
	out := map[string]record.Hash{}
	err := u.each("SELECT path, 0, hash FROM page", func(path string, _ int64, hash record.Hash) { out[path] = hash })
	return out, err
}

// KeptShared returns the number and the content of every shared block by its name.
func (u *Update) KeptShared() (map[string]Kept, error) {
	out := map[string]Kept{}
	err := u.each("SELECT name, id, hash FROM shared", func(name string, id int64, hash record.Hash) { out[name] = Kept{id, hash} })
	return out, err
}

// KeptValues returns the names of all values.
func (u *Update) KeptValues() (map[record.Hash]struct{}, error) {
	out := map[record.Hash]struct{}{}
	err := u.each("SELECT '', 0, hash FROM value", func(_ string, _ int64, hash record.Hash) { out[hash] = struct{}{} })
	return out, err
}

func (u *Update) each(query string, visit func(name string, id int64, hash record.Hash)) error {
	rows, err := u.tx.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var id int64
		var hash []byte
		if err := rows.Scan(&name, &id, &hash); err != nil {
			return err
		}
		if len(hash) != len(record.Hash{}) {
			return fmt.Errorf("store: a name of %d bytes where one of %d is kept", len(hash), len(record.Hash{}))
		}
		visit(name, id, record.Hash(hash))
	}
	return rows.Err()
}

// PutValues keeps values under their name.
func (u *Update) PutValues(hash record.Hash, data []byte) error {
	_, err := u.putValue.Exec(hash[:], data)
	return err
}

// PutPage keeps the record of a page in place of the one it had.
func (u *Update) PutPage(path string, hash record.Hash, data []byte) error {
	_, err := u.putPage.Exec(path, hash[:], data)
	return err
}

// NewShared gives a number to a shared block that the store has not had.
// Its record follows with PutShared.
func (u *Update) NewShared(name string) (int64, error) {
	result, err := u.tx.Exec("INSERT INTO shared (name, hash, record) VALUES (?, x'', x'')", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// PutShared keeps the record of a shared block in place of the one it had.
func (u *Update) PutShared(id int64, hash record.Hash, data []byte) error {
	_, err := u.putShared.Exec(hash[:], data, id)
	return err
}

// DeletePage, DeleteShared and DeleteValues drop what nothing shows any more.
func (u *Update) DeletePage(path string) error {
	_, err := u.tx.Exec("DELETE FROM page WHERE path = ?", path)
	return err
}

func (u *Update) DeleteShared(id int64) error {
	_, err := u.tx.Exec("DELETE FROM shared WHERE id = ?", id)
	return err
}

func (u *Update) DeleteValues(hash record.Hash) error {
	_, err := u.tx.Exec("DELETE FROM value WHERE hash = ?", hash[:])
	return err
}

// ClearRecords drops the records of all pages and shared blocks: those
// computed by another compiled project say nothing to this one. Values stay.
func (u *Update) ClearRecords() error {
	_, err := u.tx.Exec("DELETE FROM page; DELETE FROM shared")
	return err
}

// SetBuild names the compiled project the records belong to and keeps its program.
func (u *Update) SetBuild(id string, site []byte) error {
	const put = "INSERT INTO meta (name, value) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"
	if _, err := u.tx.Exec(put, metaBuild, []byte(id)); err != nil {
		return err
	}
	_, err := u.tx.Exec(put, metaSite, site)
	return err
}

// Commit makes the change, all of it at once.
func (u *Update) Commit() error {
	version, err := u.Version()
	if err != nil {
		u.tx.Rollback()
		return err
	}
	if _, err := u.tx.Exec("INSERT INTO meta (name, value) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value", metaVersion, version+1); err != nil {
		u.tx.Rollback()
		return err
	}
	return u.tx.Commit()
}

// Rollback drops the change. After Commit it does nothing.
func (u *Update) Rollback() { u.tx.Rollback() }
