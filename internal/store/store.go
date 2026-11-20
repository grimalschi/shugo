// Package store keeps what the runtime shows pages from: the program of the
// site, the record of every page and of every shared block, and the values
// of blocks by their content. All of it is one SQLite file, so that a server
// reads one consistent state while an update writes the next one.
//
// The store keeps the fingerprint of what every page and every shared block
// was computed from: an update computes a page again only when that
// differs. Every update compares all of them, so they are kept together, in
// a few hundred values that are read at once, and not with the records.
//
// The store also knows who shows what: which pages and shared blocks show a
// shared block, and how many records use the values kept under a name. An
// update that computed a part of the site can therefore say on which pages
// the HTML differs and drop what nothing shows any more, without reading
// the rest.
//
// And it knows who read what from the data database of the project: see
// Read. A row of the data that has changed finds there the pages and shared
// blocks to compute again.
package store

import (
	"context"
	"database/sql"
	"encoding/binary"
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
const layout = "6"

const schema = `
CREATE TABLE meta   (name TEXT PRIMARY KEY, value BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE page   (path TEXT PRIMARY KEY, hash BLOB NOT NULL, record BLOB NOT NULL, route INTEGER NOT NULL, reads BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE shared (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, hash BLOB NOT NULL, record BLOB NOT NULL, reads BLOB NOT NULL);
CREATE TABLE value  (hash BLOB PRIMARY KEY, data BLOB NOT NULL, uses INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE shows  (shared INTEGER NOT NULL, user TEXT NOT NULL, PRIMARY KEY (shared, user)) WITHOUT ROWID;
CREATE TABLE route  (id INTEGER PRIMARY KEY, props INTEGER NOT NULL, reads BLOB NOT NULL);
CREATE TABLE shape  (id INTEGER PRIMARY KEY, spec TEXT NOT NULL UNIQUE);
CREATE TABLE reads  (shape INTEGER NOT NULL, key BLOB NOT NULL, reader TEXT NOT NULL, PRIMARY KEY (shape, key, reader)) WITHOUT ROWID;
CREATE INDEX page_route ON page (route);
`

// tables are the tables of the schema.
var tables = []string{"meta", "page", "shared", "value", "shows", "route", "shape", "reads"}

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
type queries struct{ meta, page, shared, value, users, readers *sql.Stmt }

func prepare(db *sql.DB) (q queries, err error) {
	for statement, query := range map[**sql.Stmt]string{
		&q.meta:    "SELECT value FROM meta WHERE name = ?",
		&q.page:    "SELECT record FROM page WHERE path = ?",
		&q.shared:  "SELECT record FROM shared WHERE id = ?",
		&q.value:   "SELECT data FROM value WHERE hash = ?",
		&q.users:   "SELECT user FROM shows WHERE shared = ?",
		&q.readers: "SELECT reader FROM reads WHERE shape = ? AND key = ?",
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
	var laid int
	if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'").Scan(&laid); err != nil {
		return err
	}
	if laid == 1 {
		var kept []byte
		switch err := tx.QueryRow("SELECT value FROM meta WHERE name = ?", metaLayout).Scan(&kept); {
		case err == nil && string(kept) == layout:
			return nil
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return err
		}
		for _, table := range tables {
			if _, err := tx.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				return err
			}
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
	metaData    = "data"    // which state of the data database the records were computed from
)

// What the fingerprints are of: the names of their values in meta start so.
const (
	PagePrints   = "prints of pages "
	SharedPrints = "prints of shared blocks "
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

// A reader is who read something from the data database: a page or a
// shared block, named as a user is, or the list of pages of a route.

// RouteReader names the list of pages of a route as a reader of data.
func RouteReader(route int) string { return "@" + strconv.Itoa(route) }

// ReaderRoute tells whether a reader is the list of pages of a route, and
// of which.
func ReaderRoute(reader string) (route int, is bool) {
	number, is := strings.CutPrefix(reader, "@")
	if !is {
		return 0, false
	}
	route, err := strconv.Atoi(number)
	return route, err == nil
}

// Users calls visit with everyone who shows a shared block.
func (r *reader) Users(shared int64, visit func(user string)) error {
	rows, err := r.users.Query(shared)
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

// Readers calls visit with everyone who made a reading: see Read.
func (r *reader) Readers(shape int64, key []byte, visit func(reader string)) error {
	rows, err := r.readers.Query(shape, orEmpty(key))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var reader string
		if err := rows.Scan(&reader); err != nil {
			return err
		}
		visit(reader)
	}
	return rows.Err()
}

// Shapes calls visit with every shape of readings the store knows: see Read.
func (r *reader) Shapes(visit func(id int64, spec string)) error {
	rows, err := r.tx.Query("SELECT id, spec FROM shape")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var spec string
		if err := rows.Scan(&id, &spec); err != nil {
			return err
		}
		visit(id, spec)
	}
	return rows.Err()
}

// RoutePages calls visit with the address of every page a route gave.
func (r *reader) RoutePages(route int, visit func(path string)) error {
	rows, err := r.tx.Query("SELECT path FROM page WHERE route = ?", route)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		visit(path)
	}
	return rows.Err()
}

// Data returns what an update noted about the state of the data database
// its records were computed from; nothing, if none did.
func (r *reader) Data() (string, error) {
	state, _, err := r.blob(r.meta, metaData)
	return string(state), err
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

// Prints are the fingerprints of what pages, or shared blocks, were computed
// from, by the address of the page or the name of the block. They are kept
// in parts, by a hash of the name, so that an update which changes a few of
// them writes a few kilobytes and not all of them.
type Prints struct {
	parts [printParts]map[string]uint64
	moved [printParts]bool
	count int
}

const printParts = 256

// part tells which part a name belongs to.
func part(name string) int {
	hash := uint32(2166136261)
	for i := 0; i < len(name); i++ {
		hash = (hash ^ uint32(name[i])) * 16777619
	}
	return int(hash % printParts)
}

// Get returns the fingerprint kept under a name.
func (p *Prints) Get(name string) (print uint64, found bool) {
	print, found = p.parts[part(name)][name]
	return print, found
}

// Set keeps a fingerprint under a name.
func (p *Prints) Set(name string, print uint64) {
	at := part(name)
	if p.parts[at] == nil {
		p.parts[at] = map[string]uint64{}
	}
	if kept, found := p.parts[at][name]; !found {
		p.count++
	} else if kept == print {
		return
	}
	p.parts[at][name], p.moved[at] = print, true
}

// Delete drops the fingerprint kept under a name.
func (p *Prints) Delete(name string) {
	at := part(name)
	if _, found := p.parts[at][name]; found {
		delete(p.parts[at], name)
		p.moved[at] = true
		p.count--
	}
}

// Len tells how many fingerprints there are.
func (p *Prints) Len() int { return p.count }

// Names returns the names that fingerprints are kept under.
func (p *Prints) Names() []string {
	names := make([]string, 0, p.count)
	for _, part := range p.parts {
		for name := range part {
			names = append(names, name)
		}
	}
	return names
}

// Prints returns the fingerprints the store keeps: of is PagePrints or
// SharedPrints.
func (r *reader) Prints(of string) (*Prints, error) {
	rows, err := r.tx.Query("SELECT name, value FROM meta WHERE name >= ? AND name < ?", of, of+"\xff")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &Prints{}
	for rows.Next() {
		var name string
		var data []byte
		if err := rows.Scan(&name, &data); err != nil {
			return nil, err
		}
		at, err := strconv.Atoi(strings.TrimPrefix(name, of))
		if err != nil || at < 0 || at >= printParts {
			return nil, fmt.Errorf("store: there is no part %q of the fingerprints", name)
		}
		// A part is written as the length of a name, the name and eight
		// bytes, over and over.
		list := make(map[string]uint64, len(data)/24)
		for len(data) > 0 {
			length, size := binary.Uvarint(data)
			if size <= 0 || uint64(len(data)-size) < length+8 {
				return nil, fmt.Errorf("store: the %s are damaged", name)
			}
			data = data[size:]
			list[string(data[:length])] = binary.LittleEndian.Uint64(data[length:])
			data = data[length+8:]
		}
		out.parts[at] = list
		out.count += len(list)
	}
	return out, rows.Err()
}

// within returns statements that run inside a transaction.
func (q queries) within(tx *sql.Tx) reader {
	return reader{tx, queries{tx.Stmt(q.meta), tx.Stmt(q.page), tx.Stmt(q.shared), tx.Stmt(q.value), tx.Stmt(q.users), tx.Stmt(q.readers)}}
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
	keptPage, named, name, putPage, putShared *sql.Stmt
	show, unshow, shown                       *sql.Stmt
	useKept, newValues, dropUses, dropKept    *sql.Stmt
	read, unread                              *sql.Stmt
}

// Update starts a change. It waits for an update that is under way.
func (s *Store) Update() (*Update, error) {
	tx, err := s.write.Begin()
	if err != nil {
		return nil, err
	}
	u := &Update{reader: s.reads[1].within(tx)}
	for statement, query := range map[**sql.Stmt]string{
		&u.keptPage:  "SELECT hash, route, reads FROM page WHERE path = ?",
		&u.named:     "SELECT id, hash, reads FROM shared WHERE name = ?",
		&u.name:      "SELECT name FROM shared WHERE id = ?",
		&u.putPage:   "INSERT INTO page (path, hash, record, route, reads) VALUES (?, ?, ?, ?, ?) ON CONFLICT (path) DO UPDATE SET hash = excluded.hash, record = excluded.record, route = excluded.route, reads = excluded.reads",
		&u.putShared: "UPDATE shared SET hash = ?, record = ?, reads = ? WHERE id = ?",
		&u.read:      "INSERT OR IGNORE INTO reads (shape, key, reader) VALUES (?, ?, ?)",
		&u.unread:    "DELETE FROM reads WHERE shape = ? AND key = ? AND reader = ?",
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

// Kept is a record as the store has it: with which content, and what it
// read from the data to come to it. A shared block has a number, a page the
// route that gave it.
type Kept struct {
	ID    int64
	Route int
	Hash  record.Hash
	Reads []byte
}

// KeptPages returns what the store has of every page. It reads the whole
// table: an update that computes a few pages asks PageKept instead.
func (u *Update) KeptPages() (map[string]Kept, error) {
	return u.each("SELECT path, route, hash, reads FROM page")
}

// KeptShared returns what the store has of every shared block by its name.
// It reads the whole table: see SharedNamed.
func (u *Update) KeptShared() (map[string]Kept, error) {
	return u.each("SELECT name, id, hash, reads FROM shared")
}

func (u *Update) each(query string) (map[string]Kept, error) {
	rows, err := u.tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Kept{}
	for rows.Next() {
		var name string
		var kept Kept
		var hash []byte
		if err := rows.Scan(&name, &kept.ID, &hash, &kept.Reads); err != nil {
			return nil, err
		}
		if kept.Hash, err = hashOf(hash); err != nil {
			return nil, err
		}
		// Of a page the number is its route.
		kept.Route = int(kept.ID)
		out[name] = kept
	}
	return out, rows.Err()
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

// PageKept returns what the store has of one page.
func (u *Update) PageKept(path string) (kept Kept, found bool, err error) {
	var hash []byte
	if err := u.keptPage.QueryRow(path).Scan(&hash, &kept.Route, &kept.Reads); errors.Is(err, sql.ErrNoRows) {
		return kept, false, nil
	} else if err != nil {
		return kept, false, err
	}
	kept.Hash, err = hashOf(hash)
	return kept, err == nil, err
}

// SharedNamed returns what the store has of one shared block.
func (u *Update) SharedNamed(name string) (kept Kept, found bool, err error) {
	var hash []byte
	if err := u.named.QueryRow(name).Scan(&kept.ID, &hash, &kept.Reads); errors.Is(err, sql.ErrNoRows) {
		return kept, false, nil
	} else if err != nil {
		return kept, false, err
	}
	kept.Hash, err = hashOf(hash)
	return kept, err == nil, err
}

// SharedName returns the name of a shared block, and SharedReads what it
// read from the data.
func (u *Update) SharedName(id int64) (name string, err error) {
	err = u.name.QueryRow(id).Scan(&name)
	return name, err
}

func (u *Update) SharedReads(id int64) (reads []byte, err error) {
	err = u.tx.QueryRow("SELECT reads FROM shared WHERE id = ?", id).Scan(&reads)
	return reads, err
}

// PutPage keeps the record of a page in place of the one it had, with the
// route that gave the page and what it read from the data.
func (u *Update) PutPage(path string, hash record.Hash, data []byte, route int, reads []byte) error {
	_, err := u.putPage.Exec(path, hash[:], data, route, orEmpty(reads))
	return err
}

// SetPageReads keeps what a page read and which route gave it, when its
// record is what it was.
func (u *Update) SetPageReads(path string, route int, reads []byte) error {
	_, err := u.tx.Exec("UPDATE page SET route = ?, reads = ? WHERE path = ?", route, orEmpty(reads), path)
	return err
}

// orEmpty gives a column that takes no NULL an empty value for none.
func orEmpty(data []byte) []byte {
	if data == nil {
		return []byte{}
	}
	return data
}

// NewShared gives a number to a shared block that the store has not had.
// Its record follows with PutShared.
func (u *Update) NewShared(name string) (int64, error) {
	result, err := u.tx.Exec("INSERT INTO shared (name, hash, record, reads) VALUES (?, x'', x'', x'')", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// PutShared keeps the record of a shared block in place of the one it had,
// with what it read from the data.
func (u *Update) PutShared(id int64, hash record.Hash, data []byte, reads []byte) error {
	_, err := u.putShared.Exec(hash[:], data, orEmpty(reads), id)
	return err
}

// SetSharedReads keeps what a shared block read, when its record is what it was.
func (u *Update) SetSharedReads(id int64, reads []byte) error {
	_, err := u.tx.Exec("UPDATE shared SET reads = ? WHERE id = ?", orEmpty(reads), id)
	return err
}

// Route returns what the store has of the list of pages of a route: whether
// the route gave its pages more than their addresses, and what the listing
// read from the data.
func (u *Update) Route(route int) (props bool, reads []byte, found bool, err error) {
	err = u.tx.QueryRow("SELECT props, reads FROM route WHERE id = ?", route).Scan(&props, &reads)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, false, nil
	}
	return props, reads, err == nil, err
}

// PutRoute keeps the same of a route.
func (u *Update) PutRoute(route int, props bool, reads []byte) error {
	_, err := u.tx.Exec("INSERT INTO route (id, props, reads) VALUES (?, ?, ?) ON CONFLICT (id) DO UPDATE SET props = excluded.props, reads = excluded.reads", route, props, orEmpty(reads))
	return err
}

// A reading is what a reader took from the data database: the rows of a
// table found by given values of given columns. Its shape says which table
// and columns, and which columns of those rows matter to the reader; its key
// holds the values. Readings of one shape differ in their keys alone, so a
// row that has changed finds its readers with one lookup for every shape of
// its table.

// NewShape gives a number to a shape the store has not had.
func (u *Update) NewShape(spec string) (int64, error) {
	result, err := u.tx.Exec("INSERT INTO shape (spec) VALUES (?)", spec)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Read notes a reading of a reader; Unread, that the reader no longer makes it.
func (u *Update) Read(shape int64, key []byte, reader string) error {
	_, err := u.read.Exec(shape, orEmpty(key), reader)
	return err
}

func (u *Update) Unread(shape int64, key []byte, reader string) error {
	_, err := u.unread.Exec(shape, orEmpty(key), reader)
	return err
}

// SetData notes the state of the data database the records were computed from.
func (u *Update) SetData(state string) error {
	_, err := u.tx.Exec(put, metaData, []byte(state))
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

// ClearRecords drops the records of all pages and shared blocks with what
// they read and with their fingerprints: those computed by another compiled
// project say nothing to this one. Values stay, used by nothing: most of
// them will be computed again as they were. SweepValues drops the rest once
// the records are back.
func (u *Update) ClearRecords() error {
	if _, err := u.tx.Exec("DELETE FROM page; DELETE FROM shared; DELETE FROM shows; DELETE FROM route; DELETE FROM shape; DELETE FROM reads; UPDATE value SET uses = 0"); err != nil {
		return err
	}
	_, err := u.tx.Exec("DELETE FROM meta WHERE name >= ? AND name < ?", "prints ", "prints \xff")
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

// SetPrints keeps the fingerprints that have changed since they were read.
func (u *Update) SetPrints(of string, prints *Prints) error {
	for at, list := range prints.parts {
		if !prints.moved[at] {
			continue
		}
		data := make([]byte, 0, len(list)*40)
		for name, print := range list {
			data = binary.AppendUvarint(data, uint64(len(name)))
			data = append(data, name...)
			data = binary.LittleEndian.AppendUint64(data, print)
		}
		if _, err := u.tx.Exec(put, of+strconv.Itoa(at), data); err != nil {
			return err
		}
		prints.moved[at] = false
	}
	return nil
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
