package data

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
)

const wares = `
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT, price INTEGER, category TEXT, position INTEGER, note TEXT);
CREATE TABLE "a_b" (id INTEGER PRIMARY KEY, c TEXT);
CREATE TABLE a (id INTEGER PRIMARY KEY, b TEXT, c TEXT);
INSERT INTO products VALUES ('p1', 'jug', 'Jug', 100, 'c1', 1, ''), ('p2', 'cup', 'Cup', 50, 'c1', 2, '');
`

// made returns the indexes that Shugo has made, as "name: the statement
// that made it", in the order of their names.
func (l *lab) made() []string {
	l.t.Helper()
	db, err := sql.Open("sqlite", "file:"+l.path+"?mode=ro")
	if err != nil {
		l.t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT sql FROM sqlite_master WHERE type = 'index' AND name LIKE '\_shugo\_%' ESCAPE '\' ORDER BY name`)
	if err != nil {
		l.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var statement string
		if err := rows.Scan(&statement); err != nil {
			l.t.Fatal(err)
		}
		out = append(out, strings.NewReplacer(`"`, "", "CREATE INDEX ", "").Replace(statement))
	}
	return out
}

// index asks for an index and returns the name of the one that was made.
func (l *lab) index(table string, columns ...string) string {
	l.t.Helper()
	name, _, err := l.db.Index(table, columns)
	if err != nil {
		l.t.Fatal(err)
	}
	return name
}

func TestIndex(t *testing.T) {
	l := newLab(t, wares)
	id := l.db.ID

	// An index is made for columns that have none, once.
	if name := l.index("products", "category"); name != "_shugo_products_category" {
		t.Errorf("made %q", name)
	}
	if name := l.index("products", "category"); name != "" {
		t.Errorf("made %q again", name)
	}
	// The key and a unique column have their index by themselves.
	for _, column := range []string{"id", "slug"} {
		if name := l.index("products", column); name != "" {
			t.Errorf("made %q for %s", name, column)
		}
	}
	// An index that starts with the columns asked for serves them; one that
	// is longer takes the place of ours that it starts with.
	if name := l.index("products", "category", "position"); name != "_shugo_products_category_position" {
		t.Errorf("made %q", name)
	}
	if name := l.index("products", "category"); name != "" {
		t.Errorf("made %q, which the longer index serves", name)
	}
	l.index("products", "position")
	if want := []string{"_shugo_products_category_position ON products (category, position)", "_shugo_products_position ON products (position)"}; !reflect.DeepEqual(l.made(), want) {
		t.Errorf("indexes: %q, want %q", l.made(), want)
	}

	// The columns of an index count in their order.
	if name := l.index("products", "position", "category"); name != "_shugo_products_position_category" {
		t.Errorf("made %q", name)
	}
	if got := l.made(); len(got) != 2 || !strings.HasPrefix(got[1], "_shugo_products_position_category ") {
		t.Errorf("the index of the position alone was to go: %q", got)
	}

	// Another connection sees the same.
	l.open()
	if name := l.index("products", "category", "position"); name != "" {
		t.Errorf("made %q after the database was opened anew", name)
	}
	// An index is not a change of the tables: the journal keeps its name,
	// and nothing of the data has moved.
	if l.db.ID != id {
		t.Errorf("the journal got another name")
	}
	if moved, err := l.db.Moved(0); err != nil || moved {
		t.Errorf("the data moved: %v %v", moved, err)
	}
}

// An index made by hand serves if it starts with the columns asked for and
// finds rows by them as the queries compare them.
func TestIndexMadeByHand(t *testing.T) {
	l := newLab(t, wares+`
CREATE INDEX wide ON products (category, position, title);
CREATE INDEX part ON products (price) WHERE price > 10;
CREATE INDEX loose ON products (title COLLATE NOCASE);
CREATE INDEX reckoned ON products (lower(note));
`)
	if name := l.index("products", "category", "position"); name != "" {
		t.Errorf("made %q beside an index made by hand", name)
	}
	if name := l.index("products", "category"); name != "" {
		t.Errorf("made %q beside an index made by hand", name)
	}
	// One over a part of the rows, one that compares text in another way
	// and one over an expression do not find every row the queries look for.
	for _, column := range []string{"price", "title", "note"} {
		if name := l.index("products", column); name != "_shugo_products_"+column {
			t.Errorf("%s: made %q", column, name)
		}
	}
}

func TestIndexNames(t *testing.T) {
	l := newLab(t, wares)
	// Two tables may come to the same name; the later index gets a number.
	if name := l.index("a_b", "c"); name != "_shugo_a_b_c" {
		t.Errorf("made %q", name)
	}
	if name := l.index("a", "b", "c"); name != "_shugo_a_b_c_2" {
		t.Errorf("made %q", name)
	}
	if want := []string{"_shugo_a_b_c ON a_b (c)", "_shugo_a_b_c_2 ON a (b, c)"}; !reflect.DeepEqual(l.made(), want) {
		t.Errorf("indexes: %q, want %q", l.made(), want)
	}
}

// What the project's code asks for is checked.
func TestIndexRejects(t *testing.T) {
	l := newLab(t, wares)
	for name, ask := range map[string][]string{
		"a table that is not there":    {"orders", "id"},
		"a table of the journal":       {"_shugo_journal", "seq"},
		"a column that is not there":   {"products", "colour"},
		"no columns":                   {"products"},
		"a column twice":               {"products", "title", "title"},
		"something that is not a name": {"products", `title"); DROP TABLE products; --`},
	} {
		if made, _, err := l.db.Index(ask[0], ask[1:]); err == nil {
			t.Errorf("%s: made %q", name, made)
		}
	}
	if got := l.made(); len(got) != 0 {
		t.Errorf("indexes: %q", got)
	}
}

// The data has moved when the journal has new lines or is no longer in
// place; not when Shugo wrote to the database for itself.
func TestMoved(t *testing.T) {
	l := newLab(t, wares)
	moved := func() bool {
		t.Helper()
		moved, err := l.db.Moved(l.at)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	l.changes()
	if moved() {
		t.Error("moved, and nothing was written")
	}
	l.index("products", "category")
	if moved() {
		t.Error("moved by an index")
	}
	l.write(`UPDATE products SET price = 120 WHERE id = 'p1'`)
	if !moved() {
		t.Error("a changed row moved nothing")
	}
	// Reading the changes drops their lines from the journal: still nothing new.
	l.changes()
	l.changes()
	if moved() {
		t.Error("moved by the lines that were dropped")
	}
	l.write(`CREATE TABLE more (id INTEGER PRIMARY KEY)`)
	if !moved() {
		t.Error("a new table moved nothing")
	}
}
