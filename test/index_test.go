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
<p>{Shugo.query("products").where({ note: "" }).count()}</p>
-- pages/ordered.shugo --
<p>{Shugo.query("products").where({ note: "" }).order("position", "desc").all().map((row) => row.title)}</p>
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
// the order. The author of the project makes none.
func TestIndexesAreMade(t *testing.T) {
	s := newSite(t, indexSite)
	s.sql(readRows)
	report := s.build()
	want := []string{
		"_shugo_categories_position_title_id", // both: the columns with values in the order of their names, then the order
		"_shugo_products_category",            // equal
		"_shugo_products_note_position",       // ordered; counted is served by it
		"_shugo_products_price",               // sorted
		"_shugo_products_title",               // among
	}
	if got := s.indexes(); !reflect.DeepEqual(got, want) {
		t.Errorf("indexes: %q\n    want: %q", got, want)
	}
	// The report names what was made and is there. The index of the note
	// alone was made for the count, or was not, as the pages came; if it
	// was, it went when the longer one came.
	made := append([]string{}, report.Data.Indexes...)
	sort.Strings(made)
	if !reflect.DeepEqual(made, want) {
		t.Errorf("the report names %q", made)
	}
	expect(t, s.page("/ordered"), `<p>VaseTrayCupJug</p>`)
	expect(t, s.page("/sorted"), `<p>CupTray</p>`)
	expect(t, s.page("/both"), `<p>1</p>`)

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
// beside one that starts with the columns a question needs. The columns the
// rows are found by with given values may stand in it whichever way round.
func TestIndexesMadeByHand(t *testing.T) {
	s := newSite(t, indexSite)
	s.sql(readRows + `
CREATE INDEX by_category ON products (category, position);
CREATE INDEX by_note ON products (note, position, title);
CREATE INDEX by_price ON products (price);
CREATE INDEX by_title ON categories (title, position, id);
`)
	s.build()
	if got, want := s.indexes(), []string{"_shugo_products_title"}; !reflect.DeepEqual(got, want) {
		t.Errorf("indexes: %q\n    want: %q", got, want)
	}
	expect(t, s.page("/ordered"), `<p>VaseTrayCupJug</p>`)
	s.changed(`UPDATE products SET position = 9 WHERE id = 'p1'`)
	expect(t, s.page("/ordered"), `<p>JugVaseTrayCup</p>`)
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
	if got, want := report.Data.Indexes, []string{"_shugo_products_category_position"}; !reflect.DeepEqual(got, want) || report.Data.Whole {
		t.Errorf("the update made %q, want %q; %+v", got, want, report.Data)
	}
	expect(t, s.page("/"), `<p>Jug: JugCupTray</p>`)
	s.computed()
	// What the page read after the index was made is followed like the rest.
	s.changed(`UPDATE products SET title = 'Mug' WHERE id = 'p2'`, "index")
	expect(t, s.page("/"), `<p>Jug: JugMugTray</p>`)
	s.sameAsBuilt()
}
