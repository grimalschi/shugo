// Package store keeps what the runtime shows pages from: the skeleton of the
// site and the record of every page and of every shared block, with the HTML
// of its blocks inside. All of it is one SQLite file, so that a server reads
// one consistent state while an update writes the next one.
//
// The store also knows who shows what: which pages and shared blocks show a
// shared block. An update that computed a part of the site can therefore say
// on which pages the HTML differs and drop what nothing shows any more,
// without reading the rest.
//
// And it knows who read what from the data database of the project: see
// Read. A row of the data that has changed finds there the pages and shared
// blocks to compute again.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/record"
)

// layout names the set of tables below. A store laid out in another way is
// emptied: it holds nothing that a build cannot make again.
//
// A record of a page is HTML, a kilobyte or a few. The table of pages has
// rowids so that a record stays on the page of the file with its row: a
// table without them keeps a quarter of a page of the file there, and the
// rest of a row on pages of its own, mostly empty.
const layout = "8"

const schema = `
CREATE TABLE meta   (name TEXT PRIMARY KEY, value BLOB NOT NULL) WITHOUT ROWID;
CREATE TABLE page   (path TEXT PRIMARY KEY, hash BLOB NOT NULL, record BLOB NOT NULL, route INTEGER NOT NULL, reads BLOB NOT NULL, version INTEGER NOT NULL);
CREATE TABLE shared (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, hash BLOB NOT NULL, record BLOB NOT NULL, reads BLOB NOT NULL, version INTEGER NOT NULL);
CREATE TABLE gone   (path TEXT PRIMARY KEY, route INTEGER NOT NULL, version INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE shows  (shared INTEGER NOT NULL, user TEXT NOT NULL, PRIMARY KEY (shared, user)) WITHOUT ROWID;
CREATE TABLE route  (id INTEGER PRIMARY KEY, reads BLOB NOT NULL);
CREATE TABLE shape  (id INTEGER PRIMARY KEY, spec TEXT NOT NULL UNIQUE);
CREATE TABLE reads  (shape INTEGER NOT NULL, key BLOB NOT NULL, reader TEXT NOT NULL, PRIMARY KEY (shape, key, reader)) WITHOUT ROWID;
CREATE TABLE plan   (question TEXT PRIMARY KEY) WITHOUT ROWID;
CREATE INDEX page_route ON page (route);
CREATE INDEX page_version ON page (version);
CREATE INDEX shared_version ON shared (version);
CREATE INDEX gone_version ON gone (version);
`

// Store is an open store.
type Store struct {
	// Reading and writing go through separate connections: a read sees the
	// state at its start, and an update takes the right to write at its
	// start, so that two updates go one after the other and neither fails
	// halfway for the other's sake.
	read, write *sql.DB
	// readers are connections of reading, each with its statements ready:
	// a page is read with a few lookups by key, and preparing them anew for
	// every read would cost more than the lookups.
	readers chan *pooled
	opened  chan struct{} // a token for every connection that may still be opened
	writes  queries       // the statements of reading, for the connection of writing
}

// pooled is a connection of reading with its statements.
type pooled struct {
	conn *sql.Conn
	queries
}

// queries are the statements that read one thing by its key: all that a
// page is read with.
type queries struct{ meta, page, shared *sql.Stmt }

// preparer is a connection or a pool of them that statements are prepared on.
type preparer interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func prepare(db preparer) (q queries, err error) {
	for statement, query := range map[**sql.Stmt]string{
		&q.meta:   "SELECT value FROM meta WHERE name = ?",
		&q.page:   "SELECT record, route FROM page WHERE path = ?",
		&q.shared: "SELECT record FROM shared WHERE id = ?",
	} {
		if *statement, err = db.PrepareContext(context.Background(), query); err != nil {
			return q, err
		}
	}
	return q, nil
}

// Open opens the store at path and creates it if it is not there.
func Open(path string) (*Store, error) {
	open := func(lock string, pragmas ...string) (*sql.DB, error) {
		options := url.Values{"_pragma": append([]string{"busy_timeout(10000)", "journal_mode(WAL)", "synchronous(NORMAL)"}, pragmas...), "_txlock": {lock}}
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
	if s.writes, err = prepare(s.write); err != nil {
		s.write.Close()
		return nil, err
	}
	// Reading maps the file into memory: SQLite then reads its pages where
	// the system keeps them, without copying them in first.
	if s.read, err = open("deferred", "mmap_size(1073741824)"); err != nil {
		s.write.Close()
		return nil, err
	}
	readers := 2 * runtime.GOMAXPROCS(0)
	s.readers, s.opened = make(chan *pooled, readers), make(chan struct{}, readers)
	for range readers {
		s.opened <- struct{}{}
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
		// Every table of the other layout goes, whatever it was.
		rows, err := tx.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\'")
		if err != nil {
			return err
		}
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				rows.Close()
				return err
			}
			tables = append(tables, name)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, table := range tables {
			if _, err := tx.Exec(`DROP TABLE "` + strings.ReplaceAll(table, `"`, `""`) + `"`); err != nil {
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
	if _, err := tx.Exec("INSERT INTO meta (name, value) VALUES (?, ?)", metaStore, []byte(rand.Text())); err != nil {
		return err
	}
	return tx.Commit()
}

// Close closes the store.
func (s *Store) Close() error {
	var errs []error
	for drained := false; !drained; {
		select {
		case p := <-s.readers:
			errs = append(errs, p.conn.Close())
		default:
			drained = true
		}
	}
	return errors.Join(append(errs, s.read.Close(), s.write.Close())...)
}

// reading takes a connection of reading: a free one, a new one while there
// may be more, or the next one that is let go.
func (s *Store) reading() (*pooled, error) {
	select {
	case p := <-s.readers:
		return p, nil
	default:
	}
	select {
	case p := <-s.readers:
		return p, nil
	case <-s.opened:
		conn, err := s.read.Conn(context.Background())
		if err != nil {
			s.opened <- struct{}{}
			return nil, err
		}
		q, err := prepare(conn)
		if err != nil {
			conn.Close()
			s.opened <- struct{}{}
			return nil, err
		}
		return &pooled{conn, q}, nil
	}
}

// Names of what meta holds.
const (
	metaLayout  = "layout"  // how the tables are laid out
	metaStore   = "store"   // which store this is: chosen at random when the tables are laid out
	metaBuild   = "build"   // which compiled project the records belong to
	metaSite    = "site"    // the skeleton of that project
	metaVersion = "version" // grows with every update
	metaOrigin  = "origin"  // the version from which the store knows every change: see Mark
	metaTotals  = "totals"  // how many pages and shared blocks there are
	metaData    = "data"    // which state of the data database the records were computed from
)

// reader reads through one transaction, so everything it returns belongs to
// one state of the store.
type reader struct {
	db querier
	queries
}

// querier runs a statement written out: a transaction or a connection in one.
type querier interface {
	QueryContext(ctx context.Context, query string, arguments ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, arguments ...any) *sql.Row
}

func (r *reader) query(text string, arguments ...any) (*sql.Rows, error) {
	return r.db.QueryContext(context.Background(), text, arguments...)
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
// by, and its skeleton. The name is empty for a store that holds nothing yet.
func (r *reader) Build() (id string, site []byte, err error) {
	if id, err = r.BuildID(); err != nil || id == "" {
		return id, nil, err
	}
	site, _, err = r.blob(r.meta, metaSite)
	return id, site, err
}

// BuildID returns the name of the compiled project alone.
func (r *reader) BuildID() (string, error) {
	build, _, err := r.blob(r.meta, metaBuild)
	return string(build), err
}

// Mark names one state of one store. Versions are counted within a store,
// and a store laid out anew is another one, even in the same file.
type Mark struct {
	Store   string
	Version int64 // grows with every update; 0 before the first
	// Origin is the version from which the store knows every change of its
	// records and every page that went away: Changes can tell what changed
	// after any version from Origin on.
	Origin int64
}

// Mark returns which store this is and its version.
func (r *reader) Mark() (Mark, error) {
	store, _, err := r.blob(r.meta, metaStore)
	if err != nil {
		return Mark{}, err
	}
	mark := Mark{Store: string(store)}
	for name, number := range map[string]*int64{metaVersion: &mark.Version, metaOrigin: &mark.Origin} {
		if err := r.meta.QueryRow(name).Scan(number); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return mark, err
		}
	}
	return mark, nil
}

// Page returns the record of the page at path, and its route.
func (r *reader) Page(path string) (data []byte, route int, found bool, err error) {
	err = r.page.QueryRow(path).Scan(&data, &route)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, false, nil
	}
	return data, route, err == nil, err
}

// Shared returns the record of a shared block.
func (r *reader) Shared(id int64) (data []byte, found bool, err error) {
	return r.blob(r.shared, id)
}

// Pages calls visit with the address, the route and the record of every
// page. It stops at the first error of visit.
func (r *reader) Pages(visit func(path string, route int, record []byte) error) error {
	return r.each("SELECT path, route, record FROM page", visit)
}

// Changes calls visit with every page whose HTML may have changed after
// version since, with its route and its record, and with every page that
// went away since, with the route it had and nil. A page has changed when
// its own record has, or the record of a shared block it shows, directly or
// through other shared blocks. Since must not be below the origin of the
// store: see Mark. Only those pages are read; visit is called once for
// each, and an error of it stops the walk.
func (r *reader) Changes(since int64, visit func(path string, route int, record []byte) error) error {
	seen := map[string]struct{}{}
	err := r.each("SELECT path, route, record FROM page WHERE version > ?", func(path string, route int, record []byte) error {
		seen[path] = struct{}{}
		return visit(path, route, record)
	}, since)
	if err != nil {
		return err
	}

	// The pages that show a shared block that has changed. Every list is
	// read whole before its pages are, so that one query runs at a time.
	var queue []int64
	rows, err := r.query("SELECT id FROM shared WHERE version > ?", since)
	if err := column(rows, err, func(id int64) { queue = append(queue, id) }); err != nil {
		return err
	}
	met := map[int64]struct{}{}
	for _, id := range queue {
		met[id] = struct{}{}
	}
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		var users []string
		rows, err := r.query("SELECT user FROM shows WHERE shared = ?", id)
		if err := column(rows, err, func(user string) { users = append(users, user) }); err != nil {
			return err
		}
		for _, user := range users {
			if shared, is := UserShared(user); is {
				if _, already := met[shared]; !already {
					met[shared] = struct{}{}
					queue = append(queue, shared)
				}
				continue
			}
			if _, already := seen[user]; already {
				continue
			}
			seen[user] = struct{}{}
			record, route, found, err := r.Page(user)
			if err != nil {
				return err
			}
			if found {
				if err := visit(user, route, record); err != nil {
					return err
				}
			}
		}
	}

	// The pages that went away, unless they came back. They are read whole
	// before they are visited, so that one query runs at a time.
	type left struct {
		path  string
		route int
	}
	var gone []left
	err = r.each("SELECT gone.path, gone.route, NULL FROM gone LEFT JOIN page ON page.path = gone.path WHERE gone.version > ? AND page.path IS NULL", func(path string, route int, _ []byte) error {
		gone = append(gone, left{path, route})
		return nil
	}, since)
	if err != nil {
		return err
	}
	for _, page := range gone {
		if err := visit(page.path, page.route, nil); err != nil {
			return err
		}
	}
	return nil
}

// each calls visit with the address, the route and the record of every
// page a query gives.
func (r *reader) each(query string, visit func(path string, route int, record []byte) error, arguments ...any) error {
	rows, err := r.query(query, arguments...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		var route int
		var data []byte
		if err := rows.Scan(&path, &route, &data); err != nil {
			return err
		}
		if err := visit(path, route, data); err != nil {
			return err
		}
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

// Totals counts what the store holds.
type Totals struct{ Pages, Shared int }

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
	_, err = fmt.Sscanf(string(text), "%d %d", &t.Pages, &t.Shared)
	return t, err
}

// within returns statements that run inside a transaction.
func (q queries) within(tx *sql.Tx) reader {
	return reader{tx, queries{tx.Stmt(q.meta), tx.Stmt(q.page), tx.Stmt(q.shared)}}
}

// View is one state of the store being read.
type View struct{ reader }

// View runs read with one state of the store: an update that ends meanwhile
// is not seen by it.
func (s *Store) View(read func(*View) error) error {
	p, err := s.reading()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if _, err := p.conn.ExecContext(ctx, "BEGIN"); err != nil {
		p.conn.Close()
		s.opened <- struct{}{}
		return err
	}
	err = read(&View{reader{p.conn, p.queries}})
	if _, done := p.conn.ExecContext(ctx, "ROLLBACK"); done != nil {
		// A connection in an unknown state is not given out again.
		p.conn.Close()
		s.opened <- struct{}{}
		return errors.Join(err, done)
	}
	s.readers <- p
	return err
}

// Update is a change of the store being made. Nothing of it is seen by
// anyone else until Commit, and nothing remains of it after Rollback.
type Update struct {
	reader
	write                                     *sql.Tx
	number                                    int64 // the version of the store it makes
	keptPage, named, name, putPage, putShared *sql.Stmt
	show, unshow, shown, users                *sql.Stmt
	read, unread, readers                     *sql.Stmt
}

// Update starts a change. It waits for an update that is under way.
func (s *Store) Update() (*Update, error) {
	tx, err := s.write.Begin()
	if err != nil {
		return nil, err
	}
	u := &Update{reader: s.writes.within(tx), write: tx}
	mark, err := u.Mark()
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	u.number = mark.Version + 1
	for statement, query := range map[**sql.Stmt]string{
		&u.keptPage:  "SELECT hash, route, reads FROM page WHERE path = ?",
		&u.named:     "SELECT id, hash, reads FROM shared WHERE name = ?",
		&u.name:      "SELECT name FROM shared WHERE id = ?",
		&u.putPage:   "INSERT INTO page (path, hash, record, route, reads, version) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (path) DO UPDATE SET hash = excluded.hash, record = excluded.record, route = excluded.route, reads = excluded.reads, version = excluded.version",
		&u.putShared: "UPDATE shared SET hash = ?, record = ?, reads = ?, version = ? WHERE id = ?",
		&u.read:      "INSERT OR IGNORE INTO reads (shape, key, reader) VALUES (?, ?, ?)",
		&u.unread:    "DELETE FROM reads WHERE shape = ? AND key = ? AND reader = ?",
		&u.show:      "INSERT OR IGNORE INTO shows (shared, user) VALUES (?, ?)",
		&u.unshow:    "DELETE FROM shows WHERE shared = ? AND user = ?",
		&u.shown:     "SELECT 1 FROM shows WHERE shared = ? LIMIT 1",
		&u.users:     "SELECT user FROM shows WHERE shared = ?",
		&u.readers:   "SELECT reader FROM reads WHERE shape = ? AND key = ?",
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

// SharedUsers calls visit with the name of every shared block and how many
// users show it.
func (u *Update) SharedUsers(visit func(name string, users int)) error {
	rows, err := u.write.Query("SELECT shared.name, COUNT(shows.user) FROM shared LEFT JOIN shows ON shows.shared = shared.id GROUP BY shared.id")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var users int
		if err := rows.Scan(&name, &users); err != nil {
			return err
		}
		visit(name, users)
	}
	return rows.Err()
}

// KeptShared returns what the store has of every shared block by its name.
// It reads the whole table: see SharedNamed.
func (u *Update) KeptShared() (map[string]Kept, error) {
	return u.each("SELECT name, id, hash, reads FROM shared")
}

func (u *Update) each(query string) (map[string]Kept, error) {
	rows, err := u.write.Query(query)
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
	err = u.write.QueryRow("SELECT reads FROM shared WHERE id = ?", id).Scan(&reads)
	return reads, err
}

// PutPage keeps the record of a page in place of the one it had, with the
// route that gave the page and what it read from the data.
func (u *Update) PutPage(path string, hash record.Hash, data []byte, route int, reads []byte) error {
	_, err := u.putPage.Exec(path, hash[:], data, route, orEmpty(reads), u.number)
	return err
}

// SetPageReads keeps what a page read and which route gave it, when its
// record is what it was.
func (u *Update) SetPageReads(path string, route int, reads []byte) error {
	_, err := u.write.Exec("UPDATE page SET route = ?, reads = ? WHERE path = ?", route, orEmpty(reads), path)
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
	result, err := u.write.Exec("INSERT INTO shared (name, hash, record, reads, version) VALUES (?, x'', x'', x'', ?)", name, u.number)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// PutShared keeps the record of a shared block in place of the one it had,
// with what it read from the data.
func (u *Update) PutShared(id int64, hash record.Hash, data []byte, reads []byte) error {
	_, err := u.putShared.Exec(hash[:], data, orEmpty(reads), u.number, id)
	return err
}

// SetSharedReads keeps what a shared block read, when its record is what it was.
func (u *Update) SetSharedReads(id int64, reads []byte) error {
	_, err := u.write.Exec("UPDATE shared SET reads = ? WHERE id = ?", orEmpty(reads), id)
	return err
}

// Route returns what the listing of the pages of a route read from the data.
func (u *Update) Route(route int) (reads []byte, found bool, err error) {
	err = u.write.QueryRow("SELECT reads FROM route WHERE id = ?", route).Scan(&reads)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return reads, err == nil, err
}

// PutRoute keeps the same of a route.
func (u *Update) PutRoute(route int, reads []byte) error {
	_, err := u.write.Exec("INSERT INTO route (id, reads) VALUES (?, ?) ON CONFLICT (id) DO UPDATE SET reads = excluded.reads", route, orEmpty(reads))
	return err
}

// A reading is what a reader took from the data database: the rows of a
// table found by given values of given columns. Its shape says which table
// and columns, and which columns of those rows matter to the reader; its key
// holds the values. Readings of one shape differ in their keys alone, so a
// row that has changed finds its readers with one lookup for every shape of
// its table.

// NewShape gives a number to a shape the store has not had.
// PutPlan keeps a question the project's code asked an index for; DropPlan
// forgets one that can no longer be asked.
func (u *Update) PutPlan(question string) error {
	_, err := u.write.Exec("INSERT OR IGNORE INTO plan (question) VALUES (?)", question)
	return err
}

func (u *Update) DropPlan(question string) error {
	_, err := u.write.Exec("DELETE FROM plan WHERE question = ?", question)
	return err
}

func (u *Update) NewShape(spec string) (int64, error) {
	result, err := u.write.Exec("INSERT INTO shape (spec) VALUES (?)", spec)
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

// Data returns what an update noted about the state of the data database
// its records were computed from; nothing, if none did.
func (u *Update) Data() (string, error) {
	state, _, err := u.blob(u.meta, metaData)
	return string(state), err
}

// SetData notes the state of the data database the records were computed from.
func (u *Update) SetData(state string) error {
	_, err := u.write.Exec(put, metaData, []byte(state))
	return err
}

// DeletePage and DeleteShared drop a record. Who no longer shows a shared
// block is told with Unshow.
func (u *Update) DeletePage(path string) error {
	// An export removes its file, named as the route of the page names it.
	if _, err := u.write.Exec("INSERT INTO gone (path, route, version) SELECT path, route, ? FROM page WHERE path = ? ON CONFLICT (path) DO UPDATE SET route = excluded.route, version = excluded.version", u.number, path); err != nil {
		return err
	}
	_, err := u.write.Exec("DELETE FROM page WHERE path = ?", path)
	return err
}

func (u *Update) DeleteShared(id int64) error {
	_, err := u.write.Exec("DELETE FROM shared WHERE id = ?", id)
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

// Users calls visit with everyone who shows a shared block.
func (u *Update) Users(shared int64, visit func(user string)) error {
	rows, err := u.users.Query(shared)
	return column(rows, err, visit)
}

// Readers calls visit with everyone who made a reading: see Read.
func (u *Update) Readers(shape int64, key []byte, visit func(reader string)) error {
	rows, err := u.readers.Query(shape, orEmpty(key))
	return column(rows, err, visit)
}

// Plans calls visit with every question the project's code has asked an
// index for.
func (u *Update) Plans(visit func(question string)) error {
	rows, err := u.write.Query("SELECT question FROM plan")
	return column(rows, err, visit)
}

// RoutePages calls visit with the address of every page a route gave.
func (u *Update) RoutePages(route int, visit func(path string)) error {
	rows, err := u.write.Query("SELECT path FROM page WHERE route = ?", route)
	return column(rows, err, visit)
}

// Shapes calls visit with every shape of readings the store knows: see Read.
func (u *Update) Shapes(visit func(id int64, spec string)) error {
	rows, err := u.write.Query("SELECT id, spec FROM shape")
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

// column calls visit with the one column of every row a query gave.
func column[T any](rows *sql.Rows, err error, visit func(T)) error {
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var value T
		if err := rows.Scan(&value); err != nil {
			return err
		}
		visit(value)
	}
	return rows.Err()
}

// ClearRecords drops the records of all pages and shared blocks with what
// they read: those computed by another compiled project say nothing to this
// one. Which pages went away before goes too, so the store knows changes
// from this update on.
func (u *Update) ClearRecords() error {
	if _, err := u.write.Exec("DELETE FROM page; DELETE FROM gone; DELETE FROM shared; DELETE FROM shows; DELETE FROM route; DELETE FROM shape; DELETE FROM reads; DELETE FROM plan"); err != nil {
		return err
	}
	_, err := u.write.Exec(put, metaOrigin, u.number)
	return err
}

const put = "INSERT INTO meta (name, value) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"

// SetTotals keeps how much the store holds.
func (u *Update) SetTotals(t Totals) error {
	_, err := u.write.Exec(put, metaTotals, fmt.Appendf(nil, "%d %d", t.Pages, t.Shared))
	return err
}

// SetBuild names the compiled project the records belong to and keeps its skeleton.
func (u *Update) SetBuild(id string, site []byte) error {
	if _, err := u.write.Exec(put, metaBuild, []byte(id)); err != nil {
		return err
	}
	_, err := u.write.Exec(put, metaSite, site)
	return err
}

// Commit makes the change, all of it at once.
func (u *Update) Commit() error {
	if _, err := u.write.Exec(put, metaVersion, u.number); err != nil {
		u.write.Rollback()
		return err
	}
	return u.write.Commit()
}

// Rollback drops the change. After Commit it does nothing.
func (u *Update) Rollback() { u.write.Rollback() }
