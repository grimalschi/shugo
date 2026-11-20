package data

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// A lab is a data database with someone else writing to it.
type lab struct {
	t    *testing.T
	path string
	db   *DB
	at   int64 // where the journal was read last
}

const shop = `
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT, price INTEGER, weight REAL, picture);
CREATE TABLE reviews (id INTEGER PRIMARY KEY, product TEXT, text TEXT);
INSERT INTO products VALUES ('p1', 'jug', 'Jug', 100, 1.5, NULL), ('p2', 'cup', 'Cup', 50, 0.3, NULL);
`

func newLab(t *testing.T, tables string) *lab {
	t.Helper()
	l := &lab{t: t, path: filepath.Join(t.TempDir(), "data.db")}
	l.write(tables)
	l.open()
	return l
}

// write changes the database the way any program would: over a connection
// of its own that knows nothing of the journal.
func (l *lab) write(statements string, arguments ...any) {
	l.t.Helper()
	db, err := sql.Open("sqlite", "file:"+l.path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		l.t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(statements, arguments...); err != nil {
		l.t.Fatalf("%s: %v", statements, err)
	}
}

func (l *lab) open() {
	l.t.Helper()
	if l.db != nil {
		l.db.Close()
	}
	db, err := Open(l.path)
	if err != nil {
		l.t.Fatal(err)
	}
	l.db = db
	l.t.Cleanup(func() { db.Close() })
}

// changes reads what has changed since the last reading, as lines like
// "products p1: price 100 → 120", sorted.
func (l *lab) changes() []string {
	l.t.Helper()
	changes, now, lost, err := l.db.Changes(l.at)
	if err != nil {
		l.t.Fatal(err)
	}
	if lost {
		l.t.Fatalf("the journal is lost at %d", l.at)
	}
	l.at = now
	var out []string
	for _, c := range changes {
		switch {
		case c.Old == nil:
			out = append(out, fmt.Sprintf("%s %v: added", c.Table.Name, c.New[c.Table.Key]))
		case c.New == nil:
			out = append(out, fmt.Sprintf("%s %v: removed", c.Table.Name, c.Old[c.Table.Key]))
		default:
			var differs []string
			for at, column := range c.Table.Columns {
				if c.Differs(at) {
					differs = append(differs, fmt.Sprintf("%s %v → %v", column, c.Old[at], c.New[at]))
				}
			}
			out = append(out, fmt.Sprintf("%s %v: %s", c.Table.Name, c.Old[c.Table.Key], strings.Join(differs, ", ")))
		}
	}
	sort.Strings(out)
	return out
}

func (l *lab) expect(want ...string) {
	l.t.Helper()
	if got := l.changes(); !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
		l.t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

func TestChanges(t *testing.T) {
	l := newLab(t, shop)
	// What was there before the journal is not a change.
	l.expect()

	l.write(`UPDATE products SET price = 120 WHERE id = 'p1'`)
	l.expect("products p1: price 100 → 120")

	// Several changes of one row are one change from what it was first.
	l.write(`UPDATE products SET price = 130, title = 'Big jug' WHERE id = 'p1'; UPDATE products SET price = 140 WHERE id = 'p1'`)
	l.expect("products p1: title Jug → Big jug, price 120 → 140")

	// A row written with what it holds, or changed and changed back, has not changed.
	l.write(`UPDATE products SET price = 140 WHERE id = 'p1'; UPDATE products SET title = 'x' WHERE id = 'p2'; UPDATE products SET title = 'Cup' WHERE id = 'p2'`)
	l.expect()

	l.write(`INSERT INTO products VALUES ('p3', 'pot', 'Pot', 70, 2, NULL); DELETE FROM products WHERE id = 'p2'`)
	l.expect("products p2: removed", "products p3: added")

	// Added and removed in between: nothing to tell.
	l.write(`INSERT INTO reviews (product, text) VALUES ('p1', 'fine'); DELETE FROM reviews`)
	l.expect()

	// Removed and put back as it was: nothing to tell either.
	l.write(`DELETE FROM products WHERE id = 'p3'; INSERT INTO products VALUES ('p3', 'pot', 'Pot', 70, 2, NULL)`)
	l.expect()

	// A row that got another key is one row removed and one added.
	l.write(`UPDATE products SET id = 'p9' WHERE id = 'p3'`)
	l.expect("products p3: removed", "products p9: added")

	// A whole number is the same number however it is written.
	l.write(`UPDATE products SET weight = 2.0 WHERE id = 'p9'`)
	l.expect()
	l.write(`UPDATE products SET weight = 2.5, picture = x'CAFE' WHERE id = 'p9'`)
	l.expect("products p9: weight 2 → 2.5, picture <nil> → \x01CAFE")
	l.write(`UPDATE products SET picture = x'BEEF' WHERE id = 'p9'`)
	l.expect("products p9: picture \x01CAFE → \x01BEEF")
}

// INSERT OR REPLACE and its kin remove rows without firing the triggers of
// removal. The journal still knows what those rows were.
func TestReplacedRows(t *testing.T) {
	l := newLab(t, shop)

	l.write(`INSERT OR REPLACE INTO products VALUES ('p1', 'jug', 'Jug', 110, 1.5, NULL)`)
	l.expect("products p1: price 100 → 110")

	// The same row again changes nothing.
	l.write(`INSERT OR REPLACE INTO products VALUES ('p1', 'jug', 'Jug', 110, 1.5, NULL)`)
	l.expect()

	// A new row that takes the slug of another one pushes that one out.
	l.write(`INSERT OR REPLACE INTO products VALUES ('p5', 'cup', 'Mug', 60, 0.4, NULL)`)
	l.expect("products p2: removed", "products p5: added")

	// So does a row that is changed to hold the slug of another one.
	l.write(`UPDATE OR REPLACE products SET slug = 'jug' WHERE id = 'p5'`)
	l.expect("products p1: removed", "products p5: slug cup → jug")

	l.write(`INSERT INTO products VALUES ('p5', 'jug', 'Mug', 65, 0.4, NULL) ON CONFLICT (id) DO UPDATE SET price = excluded.price`)
	l.expect("products p5: price 60 → 65")
	l.write(`INSERT INTO products VALUES ('p5', 'jug', 'Mug', 65, 0.4, NULL) ON CONFLICT (id) DO UPDATE SET price = excluded.price`)
	l.expect()
}

func TestJournalIsKept(t *testing.T) {
	l := newLab(t, shop)
	first := l.db.ID
	if first == "" {
		t.Fatal("the journal has no name")
	}

	// Opened again, the journal goes on under its name, and what was
	// written in between is in it.
	l.write(`UPDATE products SET price = 1 WHERE id = 'p1'`)
	l.open()
	if l.db.ID != first {
		t.Errorf("the journal got another name: %s after %s", l.db.ID, first)
	}
	l.expect("products p1: price 100 → 1")

	// The lines that were read are dropped, the count of lines goes on.
	l.write(`UPDATE products SET price = 2 WHERE id = 'p1'`)
	before := l.at
	l.expect("products p1: price 1 → 2")
	if l.at <= before {
		t.Errorf("the journal stands at %d after %d", l.at, before)
	}
	var lines int
	l.read(`SELECT count(*) FROM _shugo_journal`, &lines)
	l.expect()
	if l.read(`SELECT count(*) FROM _shugo_journal`, &lines); lines != 0 {
		t.Errorf("%d lines stay in the journal after they were read", lines)
	}
	if at, err := l.db.Position(); err != nil || at != l.at {
		t.Errorf("the journal stands at %d, %v; it was read up to %d", at, err, l.at)
	}

	// A new table: rows may have changed while it had no triggers, so the
	// journal starts under another name.
	l.write(`CREATE TABLE notes (id INTEGER PRIMARY KEY, text TEXT)`)
	l.open()
	second := l.db.ID
	if second == first {
		t.Error("the journal kept its name after a table was added")
	}
	l.at, _ = l.db.Position()
	l.write(`INSERT INTO notes (text) VALUES ('a')`)
	l.expect("notes 1: added")

	// Someone removed a trigger: the same.
	l.write(`DROP TRIGGER "_shugo_notes_insert"`)
	l.open()
	if l.db.ID == second {
		t.Error("the journal kept its name after a trigger was removed")
	}
	l.at, _ = l.db.Position()
	l.write(`INSERT INTO notes (text) VALUES ('b')`)
	l.expect("notes 2: added")
}

func (l *lab) read(query string, into any) {
	l.t.Helper()
	db, err := sql.Open("sqlite", "file:"+l.path)
	if err != nil {
		l.t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(query).Scan(into); err != nil {
		l.t.Fatal(err)
	}
}

// Someone who read the journal earlier than the lines that are left cannot
// learn from it what has changed.
func TestJournalIsLost(t *testing.T) {
	l := newLab(t, shop)
	l.write(`UPDATE products SET price = 1`)
	l.expect("products p1: price 100 → 1", "products p2: price 50 → 1")
	l.write(`UPDATE products SET price = 2`)
	l.expect("products p1: price 1 → 2", "products p2: price 1 → 2")
	l.expect() // drops the lines of the second reading

	for _, after := range []int64{0, 1, l.at + 5} {
		if _, _, lost, err := l.db.Changes(after); err != nil || !lost {
			t.Errorf("reading from %d of %d: lost=%v, %v", after, l.at, lost, err)
		}
	}
	if _, _, lost, err := l.db.Changes(l.at); err != nil || lost {
		t.Errorf("reading from where it stands: lost=%v, %v", lost, err)
	}
}

func TestTablesNeedOneKey(t *testing.T) {
	for name, tables := range map[string]string{
		"no key":   `CREATE TABLE notes (text TEXT)`,
		"two keys": `CREATE TABLE links (a TEXT, b TEXT, PRIMARY KEY (a, b))`,
	} {
		l := &lab{t: t, path: filepath.Join(t.TempDir(), "data.db")}
		l.write(tables)
		if _, err := Open(l.path); err == nil || !strings.Contains(err.Error(), "primary key of one column") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := Open(filepath.Join(t.TempDir(), "none.db")); err == nil || !strings.Contains(err.Error(), "is not there") {
		t.Errorf("a database that is not there: %v", err)
	}
}

func TestKey(t *testing.T) {
	same := [][2][]any{
		{{int64(5)}, {5.0}},
		{{"a", int64(1)}, {"a", true}},
		{{nil}, {nil}},
	}
	for _, pair := range same {
		if string(Key(pair[0])) != string(Key(pair[1])) {
			t.Errorf("%v and %v are written apart", pair[0], pair[1])
		}
	}
	apart := [][2][]any{
		{{int64(5)}, {"5"}},
		{{"ab", "c"}, {"a", "bc"}},
		{{nil}, {""}},
		{{1.5}, {int64(1)}},
		{{"a"}, {"a", nil}},
	}
	for _, pair := range apart {
		if string(Key(pair[0])) == string(Key(pair[1])) {
			t.Errorf("%v and %v are written alike", pair[0], pair[1])
		}
	}
}
