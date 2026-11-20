// Package store keeps what the runtime shows pages from: the program of the
// site, the record of every page and of every shared block, and the values
// of blocks by their content. All of it is one SQLite file, so that a server
// reads one consistent state while an update writes the next one.
//
// The store also knows who shows what: which pages and shared blocks show a
// shared block, and how many records use the values kept under a name. An
// update that computed a part of the site can therefore say on which pages
// the HTML differs and drop what nothing shows any more, without reading
// the rest.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/record"
)

// layout names the set of tables below. A store laid out in another way is
// emptied: it holds nothing that a build cannot make again.
const layout = "2"

const schema = `
CREATE TABLE meta   (name TEXT PRIMARY KEY, value BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE page   (path TEXT PRIMARY KEY, hash BLOB NOT NULL, record BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE shared (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, hash BLOB NOT NULL, record BLOB NOT NULL);
CREATE TABLE value  (hash BLOB PRIMARY KEY, data BLOB NOT NULL, uses INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE shows  (shared INTEGER NOT NULL, user TEXT NOT NULL, PRIMARY KEY (shared, user)) WITHOUT ROWID;
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
	if err = lay(s.write); err != nil {
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

// lay makes the tables of the store, in place of tables laid out in another way.
func lay(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var tables int
	if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'").Scan(&tables); err != nil {
		return err
	}
	if tables == 1 {
		var kept []byte
		switch err := tx.QueryRow("SELECT value FROM meta WHERE name = ?", metaLayout).Scan(&kept); {
		case err == nil && string(kept) == layout:
			return nil
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return err
		}
		if _, err := tx.Exec("DROP TABLE IF EXISTS meta; DROP TABLE IF EXISTS page; DROP TABLE IF EXISTS shared; DROP TABLE IF EXISTS value; DROP TABLE IF EXISTS shows"); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(schema); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO meta (name, value) VALUES (?, ?)", metaLayout, []byte(layout)); err != nil {
		return err
	}
	return tx.Commit()
}

// Close closes the store.
func (s *Store) Close() error {
	return errors.Join(s.read.Close(), s.write.Close())
}

// Names of what meta holds.
const (
	metaLayout  = "layout"  // how the tables are laid out
	metaBuild   = "build"   // which compiled project the records belong to
	metaSite    = "site"    // the program of that project
	metaVersion = "version" // grows with every update
	metaTotals  = "totals"  // how many pages, shared blocks and values there are
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

// A user is who shows a shared block: a page, named by its address, or
// another shared block, named by its number.

// PageUser names a page as the user of a shared block.
func PageUser(path string) string { return path }

// SharedUser names a shared block as the user of another one.
func SharedUser(id int64) string { return strconv.FormatInt(id, 10) }

// UserShared tells whether a user is a shared block, and which. Every
// address of a page starts with a slash, and no number does.
func UserShared(user string) (id int64, shared bool) {
	if strings.HasPrefix(user, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(user, 10, 64)
	return id, err == nil
}

// Users calls visit with everyone who shows a shared block.
func (r *reader) Users(shared int64, visit func(user string)) error {
	rows, err := r.tx.Query("SELECT user FROM shows WHERE shared = ?", shared)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return err
		}
		visit(user)
	}
	return rows.Err()
}

// Totals counts what the store holds.
type Totals struct{ Pages, Shared, Values int }

// Totals returns how much the store held after its latest update.
func (r *reader) Totals() (Totals, error) {
	var text []byte
	var t Totals
	err := r.meta.QueryRow(metaTotals).Scan(&text)
	if errors.Is(err, sql.ErrNoRows) {
		return t, nil
	} else if err != nil {
		return t, err
	}
	_, err = fmt.Sscanf(string(text), "%d %d %d", &t.Pages, &t.Shared, &t.Values)
	return t, err
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
	hashPage, named, putPage, putShared    *sql.Stmt
	show, unshow, shown                    *sql.Stmt
	useKept, newValues, dropUses, dropKept *sql.Stmt
}

// Update starts a change. It waits for an update that is under way.
func (s *Store) Update() (*Update, error) {
	tx, err := s.write.Begin()
	if err != nil {
		return nil, err
	}
	u := &Update{reader: s.reads[1].within(tx)}
	for statement, query := range map[**sql.Stmt]string{
		&u.hashPage:  "SELECT hash FROM page WHERE path = ?",
		&u.named:     "SELECT id, hash FROM shared WHERE name = ?",
		&u.putPage:   "INSERT INTO page (path, hash, record) VALUES (?, ?, ?) ON CONFLICT (path) DO UPDATE SET hash = excluded.hash, record = excluded.record",
		&u.putShared: "UPDATE shared SET hash = ?, record = ? WHERE id = ?",
		&u.show:      "INSERT OR IGNORE INTO shows (shared, user) VALUES (?, ?)",
		&u.unshow:    "DELETE FROM shows WHERE shared = ? AND user = ?",
		&u.shown:     "SELECT 1 FROM shows WHERE shared = ? LIMIT 1",
		&u.useKept:   "UPDATE value SET uses = uses + ? WHERE hash = ?",
		&u.newValues: "INSERT INTO value (hash, data, uses) VALUES (?, ?, ?)",
		&u.dropUses:  "UPDATE value SET uses = uses - ? WHERE hash = ?",
		&u.dropKept:  "DELETE FROM value WHERE hash = ? AND uses <= 0",
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

// KeptPages returns the content of the record of every page. It reads the
// whole table: an update that computes a few pages asks PageHash instead.
func (u *Update) KeptPages() (map[string]record.Hash, error) {
	out := map[string]record.Hash{}
	err := u.each("SELECT path, 0, hash FROM page", func(path string, _ int64, hash record.Hash) { out[path] = hash })
	return out, err
}

// KeptShared returns the number and the content of every shared block by
// its name. It reads the whole table: see SharedNamed.
func (u *Update) KeptShared() (map[string]Kept, error) {
	out := map[string]Kept{}
	err := u.each("SELECT name, id, hash FROM shared", func(name string, id int64, hash record.Hash) { out[name] = Kept{id, hash} })
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
		kept, err := hashOf(hash)
		if err != nil {
			return err
		}
		visit(name, id, kept)
	}
	return rows.Err()
}

// hashOf reads the content of a record as the store keeps it. A shared
// block that has a number and no record yet has no content.
func hashOf(kept []byte) (hash record.Hash, err error) {
	if len(kept) != 0 && len(kept) != len(hash) {
		return hash, fmt.Errorf("store: a name of %d bytes where one of %d is kept", len(kept), len(hash))
	}
	copy(hash[:], kept)
	return hash, nil
}

// PageHash returns the content of the record of one page.
func (u *Update) PageHash(path string) (hash record.Hash, found bool, err error) {
	var kept []byte
	if err := u.hashPage.QueryRow(path).Scan(&kept); errors.Is(err, sql.ErrNoRows) {
		return hash, false, nil
	} else if err != nil {
		return hash, false, err
	}
	hash, err = hashOf(kept)
	return hash, err == nil, err
}

// SharedNamed returns the number and the content of one shared block.
func (u *Update) SharedNamed(name string) (kept Kept, found bool, err error) {
	var hash []byte
	if err := u.named.QueryRow(name).Scan(&kept.ID, &hash); errors.Is(err, sql.ErrNoRows) {
		return kept, false, nil
	} else if err != nil {
		return kept, false, err
	}
	kept.Hash, err = hashOf(hash)
	return kept, err == nil, err
}

// SharedName returns the name of a shared block.
func (u *Update) SharedName(id int64) (name string, err error) {
	err = u.tx.QueryRow("SELECT name FROM shared WHERE id = ?", id).Scan(&name)
	return name, err
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

// DeletePage and DeleteShared drop a record. What the record showed is told
// apart with Unshow and DropValues.
func (u *Update) DeletePage(path string) error {
	_, err := u.tx.Exec("DELETE FROM page WHERE path = ?", path)
	return err
}

func (u *Update) DeleteShared(id int64) error {
	_, err := u.tx.Exec("DELETE FROM shared WHERE id = ?", id)
	return err
}

// Show notes that a user shows a shared block; Unshow, that it no longer does.
func (u *Update) Show(shared int64, user string) error {
	_, err := u.show.Exec(shared, user)
	return err
}

func (u *Update) Unshow(shared int64, user string) error {
	_, err := u.unshow.Exec(shared, user)
	return err
}

// Shown tells whether anyone shows a shared block.
func (u *Update) Shown(shared int64) (bool, error) {
	var one int
	err := u.shown.QueryRow(shared).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// UseValues notes that more records use the values kept under a name, and
// keeps the values if the store did not have them. It tells whether it did
// not. Many values are kept fastest in the order of their names.
func (u *Update) UseValues(hash record.Hash, data []byte, more int) (fresh bool, err error) {
	result, err := u.useKept.Exec(more, hash[:])
	if err != nil {
		return false, err
	}
	if kept, err := result.RowsAffected(); err != nil || kept > 0 {
		return false, err
	}
	_, err = u.newValues.Exec(hash[:], data, more)
	return true, err
}

// DropValues notes that fewer records use the values kept under a name, and
// drops the values that none uses. It tells whether it dropped them.
func (u *Update) DropValues(hash record.Hash, fewer int) (gone bool, err error) {
	result, err := u.dropUses.Exec(fewer, hash[:])
	if err != nil {
		return false, err
	}
	if kept, err := result.RowsAffected(); err != nil {
		return false, err
	} else if kept == 0 {
		return false, errors.New("store: values that a record names are not kept")
	}
	if result, err = u.dropKept.Exec(hash[:]); err != nil {
		return false, err
	}
	dropped, err := result.RowsAffected()
	return dropped > 0, err
}

// ClearRecords drops the records of all pages and shared blocks: those
// computed by another compiled project say nothing to this one. Values
// stay, used by nothing: most of them will be computed again as they were.
// SweepValues drops the rest once the records are back.
func (u *Update) ClearRecords() error {
	_, err := u.tx.Exec("DELETE FROM page; DELETE FROM shared; DELETE FROM shows; UPDATE value SET uses = 0")
	return err
}

// SweepValues drops the values that no record uses after ClearRecords, and
// tells how many values went and how many are left.
func (u *Update) SweepValues() (dropped, left int, err error) {
	result, err := u.tx.Exec("DELETE FROM value WHERE uses = 0")
	if err != nil {
		return 0, 0, err
	}
	gone, err := result.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	err = u.tx.QueryRow("SELECT count(*) FROM value").Scan(&left)
	return int(gone), left, err
}

const put = "INSERT INTO meta (name, value) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"

// SetTotals keeps how much the store holds.
func (u *Update) SetTotals(t Totals) error {
	_, err := u.tx.Exec(put, metaTotals, fmt.Appendf(nil, "%d %d %d", t.Pages, t.Shared, t.Values))
	return err
}

// SetBuild names the compiled project the records belong to and keeps its program.
func (u *Update) SetBuild(id string, site []byte) error {
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
	if _, err := u.tx.Exec(put, metaVersion, version+1); err != nil {
		u.tx.Rollback()
		return err
	}
	return u.tx.Commit()
}

// Rollback drops the change. After Commit it does nothing.
func (u *Update) Rollback() { u.tx.Rollback() }
