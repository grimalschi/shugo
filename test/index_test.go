package test

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/grimalschi/shugo/internal/evaluate"
)

// indexes returns the indexes that Shugo has made in the data database of
// the site, sorted.
func (s *site) indexes() []string {
	s.t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(s.root, "data", "shop.db")+"?mode=ro")
	if err != nil {
		s.t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'index' AND name LIKE '\_shugo\_%' ESCAPE '\' ORDER BY name`)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			s.t.Fatal(err)
		}
		out = append(out, name)
	}
	return out
}

// Every page asks one question, so a test sees which questions get an
// index. The tables have none of their own beside the key and the address.
const indexSite = `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/equal.shugo --
<p>{Shugo.query("products").where({ category: "c1" }).all().length}</p>
-- pages/counted.shugo --
<p>{Shugo.query("products").where({ category: "c2", price: 9 }).count()}</p>
-- pages/ordered.shugo --
<p>{Shugo.query("products").where({ note: "" }).order("position", "desc").all().map((row) => row.title)}</p><p>{Shugo.query("products").where({ note: "" }).count()}</p>
-- pages/sorted.shugo --
<p>{Shugo.query("products").order("price").limit(2).all().map((row) => row.title)}</p>
-- pages/among.shugo --
<p>{Shugo.query("products").where("title", "in", ["Jug", "Cup"]).all().length}</p>
-- pages/both.shugo --
<p>{Shugo.query("categories").where({ title: "Garden" }).where("position", "in", [1, 2]).order("id").all().length}</p>
-- pages/key.shugo --
<p>{Shugo.query("products").get("p1").title}{Shugo.query("products").where("id", "in", ["p1", "p2"]).order("price").all().length}</p>
-- pages/unique.shugo --
<p>{Shugo.query("products").where({ slug: "jug" }).first().title}</p>
-- pages/all.shugo --
<p>{Shugo.query("categories").all().length}{Shugo.query("categories").count()}</p>
-- pages/bykey.shugo --
<p>{Shugo.query("products").order("id").limit(1).all().length}</p>
-- pages/other.shugo --
<p>{Shugo.query("products").where("price", "<", 8).all().length}</p>
-- pages/joined.shugo --
<p>{Shugo.query("products").join("categories", "category").get("p1").categories.title}</p>
`

// Shugo makes the indexes that the questions of a project need: by the
// columns the rows are found by with given values, then by the columns of
// the order, down to the key. The author of the project makes none.
func TestIndexesAreMade(t *testing.T) {
	s := newSite(t, indexSite)
	s.sql(readRows)
	report := s.build()
	want := []string{
		"_shugo_categories_position_title_id", // both: the columns with values in the order of their names, then the order
		"_shugo_products_category_id",         // equal
		"_shugo_products_category_price",      // counted: a count has no order
		"_shugo_products_note_position_id",    // ordered; the count beside it is served by it
		"_shugo_products_price_id",            // sorted
		"_shugo_products_title_id",            // among
	}
	if got := s.indexes(); !reflect.DeepEqual(got, want) {
		t.Errorf("indexes: %q\n    want: %q", got, want)
	}
	// The report names what was made.
	made := append([]string{}, report.Data.Indexes...)
	sort.Strings(made)
	if !reflect.DeepEqual(made, want) {
		t.Errorf("the report names %q", made)
	}
	expect(t, s.page("/ordered"), `<p>VaseTrayCupJug</p><p>4</p>`)
	expect(t, s.page("/sorted"), `<p>CupTray</p>`)
	expect(t, s.page("/both"), `<p>1</p>`)
	expect(t, s.page("/counted"), `<p>1</p>`)

	// The indexes are a part of neither the tables nor the data: an update
	// after them follows the journal, and a second build makes none.
	s.sql(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`)
	if report := s.update(); report.Data.Whole || len(report.Data.Indexes) != 0 || report.Computed.Pages == 0 || report.Computed.Pages > 6 {
		t.Errorf("the update after the indexes were made: %+v, computed %+v", report.Data, report.Computed)
	}
	expect(t, s.page("/key"), `<p>Big jug2</p>`)
	if report := s.build(); len(report.Data.Indexes) != 0 {
		t.Errorf("a second build made %q", report.Data.Indexes)
	}
	if got := s.indexes(); !reflect.DeepEqual(got, want) {
		t.Errorf("indexes after a second build: %q", got)
	}
	s.sameAsBuilt()
}

// A database that has indexes made by hand keeps them, and Shugo makes none
// beside one that serves a question. The columns the rows are found by with
// given values may stand in it whichever way round. For rows that are taken
// it has to end where the order of the question ends, with the key; for a
// count it only has to start with the columns of the count.
func TestIndexesMadeByHand(t *testing.T) {
	s := newSite(t, indexSite)
	s.sql(readRows + `
CREATE INDEX by_category ON products (category, id);
CREATE INDEX by_note ON products (note, position, id);
CREATE INDEX by_price ON products (price);
CREATE INDEX by_title ON categories (title, position, id);
CREATE INDEX by_three ON products (price, category, note);
`)
	s.build()
	// The index of the price does not end with the key: products of the
	// same price would come from it as they happen to lie in the table.
	if got, want := s.indexes(), []string{"_shugo_products_price_id", "_shugo_products_title_id"}; !reflect.DeepEqual(got, want) {
		t.Errorf("indexes: %q\n    want: %q", got, want)
	}
	expect(t, s.page("/ordered"), `<p>VaseTrayCupJug</p><p>4</p>`)
	expect(t, s.page("/counted"), `<p>1</p>`)
	s.changed(`UPDATE products SET position = 9 WHERE id = 'p1'`)
	expect(t, s.page("/ordered"), `<p>JugVaseTrayCup</p><p>4</p>`)
	s.sameAsBuilt()
}

const orderSite = `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/a.shugo --
---
const titles = (query) => query.all().map((row) => row.title)
console.log("a")
---
<p>{titles(Shugo.query("products").where({ category: "c1" }))}</p><p>{titles(Shugo.query("products").order("price"))} {titles(Shugo.query("products").order("price", "desc"))} {titles(Shugo.query("products").order("price").limit(2))}</p><p>{titles(Shugo.query("tags").where({ kind: "x" }))} {titles(Shugo.query("odd").where({ kind: "x" }))}</p><p>{titles(Shugo.query("products").where({ note: "" }))} {titles(Shugo.query("products").order("slug"))} {titles(Shugo.query("products").order("alias"))}</p>
-- pages/b.shugo --
<p>{Shugo.query("products").where({ category: "c1" }).order("title").all().map((row) => row.title)}</p><p>{Shugo.query("products").order("price").order("title", "desc").all().map((row) => row.title)}</p>
`

// Three products cost the same, and were put into their table in another
// order than that of their keys. An address tells any two products apart;
// a second name does not, for three have none; and a note may not repeat
// only where there is one. One table keeps its rows by a key of text, one
// by a key of whole numbers.
const orderRows = `
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, alias TEXT UNIQUE, title TEXT, price INTEGER, note TEXT, category TEXT, stock INTEGER);
INSERT INTO products VALUES
  ('p3', 'tray', NULL, 'Tray', 5, '', 'c1', 1), ('p1', 'jug', NULL, 'Jug', 5, '', 'c1', 1),
  ('p4', 'vase', 'urn', 'Vase', 9, '', 'c2', 1), ('p2', 'cup', NULL, 'Cup', 5, '', 'c1', 1);
CREATE UNIQUE INDEX one_note ON products (note) WHERE note <> '';
CREATE TABLE tags (title TEXT PRIMARY KEY, kind TEXT) WITHOUT ROWID;
INSERT INTO tags VALUES ('new', 'x'), ('big', 'x'), ('old', 'y');
CREATE TABLE odd (id INTEGER PRIMARY KEY, title TEXT, kind TEXT);
INSERT INTO odd VALUES (2, 'second', 'x'), (1, 'first', 'x');
`

// Rows that nothing orders, and rows that the order asked for does not tell
// apart, come in the order of their keys — whichever index finds them and
// wherever they lie in their table. So a page shows the same after an index
// is made for another page, and after a row is removed and put back as it
// was; and the same as a site built anew.
func TestRowsComeInTheOrderOfTheirKeys(t *testing.T) {
	const want = `<p>JugCupTray</p><p>JugCupTrayVase VaseTrayCupJug JugCup</p><p>bignew firstsecond</p><p>JugCupTrayVase CupJugTrayVase JugCupTrayVase</p>`
	s := newSite(t, orderSite)
	s.sql(orderRows)
	s.build()
	s.computed()
	expect(t, s.page("/a"), want)
	expect(t, s.page("/b"), `<p>CupJugTray</p><p>TrayJugCupVase</p>`)
	// An order by the address needs no index beside the one the table has:
	// no two rows are the same in it. An order by the second name does.
	if got, want := s.indexes(), []string{
		"_shugo_odd_kind", "_shugo_products_alias_id", "_shugo_products_category_id", "_shugo_products_category_title_id",
		"_shugo_products_note_id", "_shugo_products_price_id", "_shugo_products_price_title_id", "_shugo_tags_kind",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("indexes: %q\n    want: %q", got, want)
	}
	s.sameAsBuilt()

	// A row is removed and put back as it was, and one is written over
	// with what no page reads of it: both lie at the end of the table now,
	// and nothing has changed for the pages.
	s.changed(`DELETE FROM products WHERE id = 'p1'; INSERT INTO products VALUES ('p1', 'jug', NULL, 'Jug', 5, '', 'c1', 1)`)
	s.changed(`INSERT OR REPLACE INTO products VALUES ('p3', 'tray', NULL, 'Tray', 5, '', 'c1', 2)`)
	expect(t, s.page("/a"), want)
	s.sameAsBuilt()

	// The same where indexes made by hand go on after the columns of the
	// questions, and were there before the site was first built.
	s = newSite(t, orderSite)
	s.sql(orderRows + `
CREATE INDEX wide ON products (category, title);
CREATE INDEX dear ON products (price, title);
`)
	s.build()
	expect(t, s.page("/a"), want)
	s.sameAsBuilt()
}

// A question that is first asked by an update gets its index then: the
// rows that the update reads before and after it are of one state for all
// that the journal tells.
func TestIndexMadeByAnUpdate(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/index.shugo --
---
const item = Shugo.query("products").get("p1")
// The second question is asked only of a dear product.
const others = item.price > 10 ? Shugo.query("products").where({ category: item.category }).order("position").all() : []
console.log("index")
---
<p>{item.title}: {others.map((row) => row.title)}</p>
`)
	s.sql(readRows)
	s.build()
	s.computed()
	if got := s.indexes(); len(got) != 0 {
		t.Fatalf("indexes after the build: %q", got)
	}
	s.sql(`UPDATE products SET price = 12 WHERE id = 'p1'`)
	report, err := s.run(evaluate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := report.Data.Indexes, []string{"_shugo_products_category_position_id"}; !reflect.DeepEqual(got, want) || report.Data.Whole {
		t.Errorf("the update made %q, want %q; %+v", got, want, report.Data)
	}
	expect(t, s.page("/"), `<p>Jug: JugCupTray</p>`)
	s.computed()
	// What the page read after the index was made is followed like the rest.
	s.changed(`UPDATE products SET title = 'Mug' WHERE id = 'p2'`, "index")
	expect(t, s.page("/"), `<p>Jug: JugMugTray</p>`)
	s.sameAsBuilt()
}

// Reading starts anew when an index is made in the middle of a run, so what
// is read after it may be of a later state of the data than what was read
// before. The page here is written to show it: something writes to the data
// between its two questions, and the second is new to the run.
//
// Nothing is lost by it. The run notes where the journal stood before it
// read anything, so the change is past that place, and the next update
// computes again whoever read the row — as it does for any change that
// comes while a site is built.
func TestChangeWhileAnIndexIsMade(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- lib/outside.ts --
// Whoever fills the database writes to it while the site is built.
import { createRequire } from "node:module"
const require = createRequire(import.meta.url)
export function rename(): void {
  const change = "UPDATE products SET title = 'Mug' WHERE id = 'p2' AND title = 'Cup'"
  if (typeof Bun !== "undefined") {
    const { Database } = require("bun:sqlite")
    const db = new Database(process.env.SHUGO_DATA)
    db.exec(change)
    db.close()
  } else {
    const { DatabaseSync } = require("node:sqlite")
    const db = new DatabaseSync(process.env.SHUGO_DATA)
    db.exec(change)
    db.close()
  }
}
-- pages/index.shugo --
---
import { rename } from "../lib/outside"
const before = Shugo.query("products").get("p2").title
rename()
const after = Shugo.query("products").where({ category: "c1" }).order("position").all().map((row) => row.title)
console.log("index")
---
<p>{before}: {after}</p>
`)
	s.sql(readRows)
	if report := s.build(); !reflect.DeepEqual(report.Data.Indexes, []string{"_shugo_products_category_position_id"}) {
		t.Fatalf("the build made %q", report.Data.Indexes)
	}
	expect(t, s.page("/"), `<p>Cup: JugMugTray</p>`)
	s.computed()

	report := s.update()
	if report.Data.Whole || report.Data.Rows != 1 {
		t.Errorf("the update after it: %+v", report.Data)
	}
	if got := s.computed(); !reflect.DeepEqual(got, []string{"index"}) {
		t.Errorf("computed %q", got)
	}
	expect(t, s.page("/"), `<p>Mug: JugMugTray</p>`)
	s.sameAsBuilt()
}
