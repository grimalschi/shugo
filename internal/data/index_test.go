package data

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
)

const wares = `
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT, price INTEGER, category TEXT, position INTEGER, note TEXT, maker TEXT COLLATE NOCASE);
CREATE TABLE "a_b" (id INTEGER PRIMARY KEY, c TEXT);
CREATE TABLE a (id INTEGER PRIMARY KEY, b TEXT, c TEXT);
INSERT INTO products VALUES ('p1', 'jug', 'Jug', 100, 'c1', 1, '', 'Anna'), ('p2', 'cup', 'Cup', 50, 'c1', 2, '', 'anna');
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
		out = append(out, strings.NewReplacer(`"`, "", "CREATE INDEX ", "", " COLLATE BINARY", "").Replace(statement))
	}
	return out
}

// by are the columns rows are found by with given values.
type by = []string

// rows asks for an index for rows that are found by given values of some
// columns and taken in the order of others, and returns the name of the one
// that was made. The project's code asks for an order down to the key.
func (l *lab) rows(table string, equal by, order ...string) string {
	l.t.Helper()
	name, err := l.db.Index(table, equal, order, true)
	if err != nil {
		l.t.Fatal(err)
	}
	return name
}

// count asks for an index for rows that are found and counted.
func (l *lab) count(table string, equal by) string {
	l.t.Helper()
	name, err := l.db.Index(table, equal, nil, false)
	if err != nil {
		l.t.Fatal(err)
	}
	return name
}

// The table of products keeps its rows as they come: an index of it ends
// with the key only if it names the key. The table a has a key of whole
// numbers and keeps its rows by it: every index of it goes on with the key.
func TestIndex(t *testing.T) {
	l := newLab(t, wares)
	id := l.db.ID
	none := func(what string, names ...string) {
		t.Helper()
		for _, name := range names {
			if name != "" {
				t.Errorf("%s: made %q", what, name)
			}
		}
	}
	made := func(name, want string) {
		t.Helper()
		if name != want {
			t.Errorf("made %q, want %q", name, want)
		}
	}

	// An index is made for columns that have none, once.
	made(l.rows("products", by{"category"}, "id"), "_shugo_products_category_id")
	none("the same again", l.rows("products", by{"category"}, "id"), l.count("products", by{"category"}))
	// The key and a unique column have their index by themselves, for
	// finding a row and for the order. The rows of a table with a key of
	// whole numbers lie in its order with no index at all.
	none("the key", l.rows("products", by{"id"}), l.rows("products", nil, "id"), l.rows("a", by{"id"}), l.rows("a", nil, "id"))
	none("a unique column", l.rows("products", by{"slug"}), l.rows("products", nil, "slug"), l.count("products", by{"slug"}))
	// One row is found by them, so whatever else is asked of it adds nothing.
	none("one row", l.rows("products", by{"category", "id"}), l.rows("products", by{"slug"}, "position", "id"), l.rows("a", by{"b", "id"}, "c"))

	// Rows that are taken come in the order asked for, down to their keys.
	// An index gives that when it ends where the order ends.
	made(l.rows("products", by{"category"}, "position", "id"), "_shugo_products_category_position_id")
	// The columns of the order count in their order.
	made(l.rows("products", nil, "position", "category", "id"), "_shugo_products_position_category_id")
	// The columns rows are found by with given values do not: an index that
	// starts with them whichever way round finds the rows at once.
	none("two columns the other way round", l.rows("products", by{"position", "category"}, "id"))
	// An index that goes on after the columns asked for does not give the
	// rows in their order.
	made(l.rows("products", nil, "position", "id"), "_shugo_products_position_id")
	made(l.rows("products", by{"title", "category"}, "position", "id"), "_shugo_products_title_category_position_id")
	made(l.rows("products", by{"title"}, "id"), "_shugo_products_title_id")
	// A count asks for no order: any index that starts with its columns does.
	none("a count", l.count("products", by{"category"}), l.count("products", by{"position"}), l.count("products", by{"title", "category"}), l.count("products", by{"category", "position"}))
	made(l.count("products", by{"price"}), "_shugo_products_price")
	// The index of the count does not end with the key.
	made(l.rows("products", by{"price"}, "id"), "_shugo_products_price_id")

	// Where the table keeps its rows by the key, the key is not named in
	// an index: rows that are the same in its columns follow by their keys.
	made(l.rows("a", by{"b"}, "c", "id"), "_shugo_a_b_c")
	made(l.rows("a", by{"b"}, "id"), "_shugo_a_b")
	made(l.count("a", by{"c"}), "_shugo_a_c")
	none("rows by the column of a count", l.rows("a", by{"c"}, "id"), l.rows("a", nil, "c", "id"))

	// A column that was declared to compare text loosely gets an index
	// that compares it exactly, as the queries do.
	made(l.rows("products", by{"maker"}, "id"), "_shugo_products_maker_id")

	if want := []string{
		"_shugo_a_b ON a (b)",
		"_shugo_a_b_c ON a (b, c)",
		"_shugo_a_c ON a (c)",
		"_shugo_products_category_id ON products (category, id)",
		"_shugo_products_category_position_id ON products (category, position, id)",
		"_shugo_products_maker_id ON products (maker, id)",
		"_shugo_products_position_category_id ON products (position, category, id)",
		"_shugo_products_position_id ON products (position, id)",
		"_shugo_products_price ON products (price)",
		"_shugo_products_price_id ON products (price, id)",
		"_shugo_products_title_category_position_id ON products (title, category, position, id)",
		"_shugo_products_title_id ON products (title, id)",
	}; !reflect.DeepEqual(l.made(), want) {
		t.Errorf("indexes: %q\n    want: %q", l.made(), want)
	}

	// Another connection sees the same.
	l.open()
	none("after the database was opened anew",
		l.rows("products", by{"category"}, "position", "id"), l.rows("products", by{"title"}, "id"), l.count("products", by{"price"}),
		l.rows("products", by{"maker"}, "id"), l.rows("a", by{"b"}, "c", "id"), l.rows("a", by{"c"}, "id"))
	// An index is not a change of the tables: the journal keeps its name,
	// and nothing of the data has moved.
	if l.db.ID != id {
		t.Errorf("the journal got another name")
	}
	if moved, err := l.db.Moved(0); err != nil || moved {
		t.Errorf("the data moved: %v %v", moved, err)
	}
}

// An index made by hand serves if it is over the columns asked for and
// finds rows by them as the queries compare them.
func TestIndexMadeByHand(t *testing.T) {
	l := newLab(t, wares+`
CREATE TABLE tags (title TEXT PRIMARY KEY, kind TEXT, colour TEXT) WITHOUT ROWID;
CREATE INDEX wide ON products (category, position, id);
CREATE INDEX keyless ON products (note, position);
CREATE INDEX part ON products (price) WHERE price > 10;
CREATE INDEX loose ON products (title COLLATE NOCASE);
CREATE INDEX reckoned ON products (lower(category));
CREATE INDEX whole ON a (c, b);
CREATE INDEX kinds ON tags (kind);
`)
	for _, name := range []string{
		l.rows("products", by{"category"}, "position", "id"),
		l.rows("products", by{"position", "category"}, "id"),
		l.count("products", by{"category"}),
		l.count("products", by{"position", "category"}),
		l.count("products", by{"note"}),
		// Tables that keep their rows by the key: an index of theirs goes
		// on with the key, though it does not name it.
		l.rows("a", by{"c"}, "b", "id"),
		l.rows("a", by{"b", "c"}, "id"),
		l.rows("tags", by{"kind"}, "title"),
	} {
		if name != "" {
			t.Errorf("made %q beside an index made by hand", name)
		}
	}
	// It does not end with the key: rows of the same note and position
	// would come from it as they happen to lie in the table.
	if name := l.rows("products", by{"note"}, "position", "id"); name != "_shugo_products_note_position_id" {
		t.Errorf("made %q", name)
	}
	// It does not serve what it does not start with.
	if name := l.count("products", by{"position"}); name != "_shugo_products_position" {
		t.Errorf("made %q", name)
	}
	// One over a part of the rows, one that compares text in another way
	// and one over an expression do not find every row the queries look for.
	for _, column := range []string{"price", "title", "category"} {
		if name := l.rows("products", by{column}, "id"); name != "_shugo_products_"+column+"_id" {
			t.Errorf("%s: made %q", column, name)
		}
	}
	if name := l.rows("tags", by{"colour"}, "title"); name != "_shugo_tags_colour" {
		t.Errorf("made %q", name)
	}
}

func TestIndexNames(t *testing.T) {
	l := newLab(t, wares)
	// Two tables may come to the same name; the later index gets a number.
	if name := l.rows("a_b", by{"c"}, "id"); name != "_shugo_a_b_c" {
		t.Errorf("made %q", name)
	}
	if name := l.rows("a", by{"b"}, "c", "id"); name != "_shugo_a_b_c_2" {
		t.Errorf("made %q", name)
	}
	if want := []string{"_shugo_a_b_c ON a_b (c)", "_shugo_a_b_c_2 ON a (b, c)"}; !reflect.DeepEqual(l.made(), want) {
		t.Errorf("indexes: %q, want %q", l.made(), want)
	}
}

// What the project's code asks for is checked.
func TestIndexRejects(t *testing.T) {
	l := newLab(t, wares)
	for name, ask := range map[string]struct {
		table        string
		equal, order []string
		rows         bool
	}{
		"a table that is not there":        {"orders", by{"id"}, nil, true},
		"a table of the journal":           {"_shugo_journal", by{"seq"}, nil, true},
		"a column that is not there":       {"products", by{"colour"}, nil, true},
		"an order that is not there":       {"products", by{"title"}, []string{"colour"}, true},
		"no columns":                       {"products", nil, nil, true},
		"a column twice":                   {"products", by{"title", "title"}, nil, true},
		"a column found by and ordered by": {"products", by{"title"}, []string{"title"}, true},
		"a count in an order":              {"products", by{"title"}, []string{"position"}, false},
		"something that is not a name":     {"products", by{`title"); DROP TABLE products; --`}, nil, true},
	} {
		if made, err := l.db.Index(ask.table, ask.equal, ask.order, ask.rows); err == nil {
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
	l.rows("products", by{"category"}, "id")
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
