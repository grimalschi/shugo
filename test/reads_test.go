package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/evaluate"
)

const readRows = `
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT UNIQUE, title TEXT, price INTEGER, note TEXT, category TEXT, position INTEGER);
INSERT INTO categories VALUES ('c1', 'Kitchen', 1), ('c2', 'Garden', 2);
INSERT INTO products VALUES
  ('p1', 'jug', 'Jug', 10, '', 'c1', 1), ('p2', 'cup', 'Cup', 5, '', 'c1', 2),
  ('p3', 'tray', 'Tray', 7, '', 'c1', 3), ('p4', 'vase', 'Vase', 9, '', 'c2', 4);
`

// No query here names the columns it takes. Every page asks for the same
// product and does one thing with it, so a test sees what that thing makes
// the page depend on. A page prints its name when it is computed.
const readSite = `
-- shugo.json --
{"data": "data/shop.db"}
-- lib/text.ts --
export const shout = (row: { title: string }) => row.title.toUpperCase()
-- components/Row.shugo --
<li>{Shugo.props.row.title}</li>
-- pages/script.shugo --
---
const item = Shugo.query("products").get("p1")
const cost = item.price * 2
console.log("script")
---
<p>{cost}</p>
-- pages/template.shugo --
---
const item = Shugo.query("products").get("p1")
console.log("template")
---
<p>{item.title}</p>
-- pages/neither.shugo --
---
const item = Shugo.query("products").get("p1")
console.log("neither")
---
<p><g-if cond={item}>there</g-if><g-else>gone</g-else></p>
-- pages/asked.shugo --
---
const item = Shugo.query("products").get("p1")
const priced = "price" in item
console.log("asked")
---
<p>{priced}</p>
-- pages/spread.shugo --
---
const copy = { ...Shugo.query("products").get("p1") }
console.log("spread")
---
<p>{copy.title}</p>
-- pages/written.shugo --
---
const text = JSON.stringify(Shugo.query("products").get("p1")).length > 0
console.log("written")
---
<p>{text}</p>
-- pages/function.shugo --
---
import { shout } from "../lib/text"
const item = Shugo.query("products").get("p1")
console.log("function")
---
<p>{shout(item)}</p>
-- pages/component.shugo --
---
import Row from "../components/Row.shugo"
const item = Shugo.query("products").get("p1")
console.log("component")
---
<ul><Row row={item} /></ul>
-- pages/together.shugo --
---
const rows = Shugo.query("products").where({ category: "c1" }).order("position").all()
console.log("together")
---
<ul><g-for each={rows} as="row"><li>{row.title}</li></g-for></ul>
-- pages/alone.shugo --
---
const rows = Shugo.query("products").order("position").limit(2).all()
console.log("alone")
---
<ul><g-for each={rows} as="row"><li>{row.title}</li></g-for></ul>
-- pages/joined.shugo --
---
const item = Shugo.query("products").join("categories", "category").get("p1")
console.log("joined")
---
<p>{item.title} in {item.categories.title}</p>
-- pages/listed/[slug].shugo --
---
export const getStaticPaths = ({ query }) => query("products").all().map(({ slug }) => ({ params: { slug } }))
console.log("listed " + Shugo.params.slug)
---
<p>{Shugo.params.slug}</p>
-- pages/handed/[slug].shugo --
---
export const getStaticPaths = ({ query }) => query("products").where({ category: "c2" }).all().map((row) => ({ params: { slug: row.slug }, props: { row } }))
console.log("handed " + Shugo.params.slug)
---
<p>{Shugo.props.row.title}</p>
`

// A page depends on the columns that are read of the rows it asked for, by
// its script or by its template, and on no others.
func TestColumnsThatAreRead(t *testing.T) {
	s := newSite(t, readSite)
	s.sql(readRows)
	s.build()
	s.computed()
	expect(t, s.page("/script"), `<p>20</p>`)
	expect(t, s.page("/function"), `<p>JUG</p>`)
	expect(t, s.page("/component"), `<ul><li>Jug</li></ul>`)
	expect(t, s.page("/joined"), `<p>Jug in Kitchen</p>`)
	expect(t, s.page("/handed/vase"), `<p>Vase</p>`)

	// A column nobody reads of the product: only who took the product as a
	// whole depends on it. A spread and JSON.stringify list its keys; a
	// function in a template and a component that only shows what it is
	// given are handed the row, and what they do with it is not followed.
	s.changed(`UPDATE products SET note = 'checked' WHERE id = 'p1'`, "component", "function", "spread", "written")
	// A column the script read, and one it only asked about.
	s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "asked", "component", "function", "script", "spread", "written")
	expect(t, s.page("/script"), `<p>24</p>`)
	// A column the template shows.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "alone", "component", "function", "joined", "spread", "template", "together", "written")
	expect(t, s.page("/template"), `<p>Big jug</p>`)
	expect(t, s.page("/joined"), `<p>Big jug in Kitchen</p>`)

	// Rows that are read together: any of them, in the columns read of any.
	s.changed(`UPDATE products SET title = 'Big tray' WHERE id = 'p3'`, "together")
	s.changed(`UPDATE products SET price = 8 WHERE id = 'p3'`)
	// Rows behind a limit count alone: the third is not read, until the
	// order brings it in.
	s.changed(`UPDATE products SET title = 'Mug' WHERE id = 'p2'`, "alone", "together")
	s.changed(`UPDATE products SET position = 0 WHERE id = 'p3'`, "alone", "together")
	expect(t, s.page("/alone"), `<ul><li>Big tray</li><li>Big jug</li></ul>`)

	// A joined row is read in the columns that are read of it; the column
	// that points at it is read by the join.
	s.changed(`UPDATE categories SET position = 5 WHERE id = 'c1'`)
	s.changed(`UPDATE categories SET title = 'Kitchenware' WHERE id = 'c1'`, "joined")
	// The product moves into the category whose rows a list of pages hands
	// to its pages: the list gains a page.
	s.changed(`UPDATE products SET category = 'c2' WHERE id = 'p1'`, "component", "function", "handed jug", "handed vase", "joined", "spread", "together", "written")
	expect(t, s.page("/joined"), `<p>Big jug in Garden</p>`)

	// A list of pages that reads the addresses alone does not depend on a
	// title. One that hands its rows to its pages depends on all of them:
	// what a page reads of a row is read after the list is made.
	s.changed(`UPDATE products SET title = 'Tall vase' WHERE id = 'p4'`, "handed jug", "handed vase")
	expect(t, s.page("/handed/vase"), `<p>Tall vase</p>`)
	s.changed(`UPDATE products SET note = 'fragile' WHERE id = 'p4'`, "handed jug", "handed vase")
	// A product that gets another address: its page moves.
	s.changed(`UPDATE products SET slug = 'pitcher' WHERE id = 'p1'`, "component", "function", "handed pitcher", "handed vase", "listed pitcher", "spread", "written")
	if s.has("/listed/jug") || !s.has("/listed/pitcher") {
		t.Errorf("pages: %v", s.addresses())
	}

	// The row goes: everyone who asked for it by its key is told.
	s.changed(`DELETE FROM products WHERE id = 'p2'`, "alone", "together")
	s.sameAsBuilt()
}

// A query may still name its columns, to take less from the database. What
// the page depends on is what is read of the rows all the same.
func TestSelectTakesLess(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/index.shugo --
---
const item = Shugo.query("products").select("title", "price", "note").get("p1")
console.log("index")
---
<p>{item.title} {String(item.slug)}</p>
`)
	s.sql(readRows)
	s.build()
	s.computed()
	// A column that was not taken is not there.
	expect(t, s.page("/"), `<p>Jug undefined</p>`)
	s.changed(`UPDATE products SET price = 12, note = 'x', slug = 'pot' WHERE id = 'p1'`)
	s.changed(`UPDATE products SET title = 'Pot' WHERE id = 'p1'`, "index")
}

// A check computes everything and names the pages that came out different
// although no row they read has changed: they read something past the
// database, which a plain update cannot see.
func TestCheckFindsWhatIsReadPastTheDatabase(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- note.txt --
old
-- pages/index.shugo --
---
import { readFileSync } from "node:fs"
const note = readFileSync("note.txt", "utf8").trim()
const count = Shugo.query("products").count()
---
<p>{note}: {count}</p>
-- pages/about.shugo --
---
const item = Shugo.query("products").get("p1")
---
<p>{item.title}</p>
`)
	s.sql(readRows)
	s.build()

	// Nothing differs: the check finds nothing.
	report, err := s.run(evaluate.Options{Check: true})
	if err != nil || !report.Checked || len(report.Unseen) != 0 || report.Computed.Pages != 2 {
		t.Fatalf("a check of an unchanged site: %v, %+v", err, report)
	}

	// A row changes and a file does. The update sees the row.
	s.write("-- note.txt --\nnew\n")
	s.sql(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 1 {
		t.Errorf("a plain update computed %+v", report.Computed)
	}
	expect(t, s.page("/"), `<p>old: 4</p>`)
	expect(t, s.page("/about"), `<p>Big jug</p>`)

	// The check sees the file too, and tells which page read it. A page
	// that changes with a row it read is no finding.
	s.sql(`UPDATE products SET title = 'Jug' WHERE id = 'p1'`)
	report, err = s.run(evaluate.Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"the page /"}; !reflect.DeepEqual(report.Unseen, want) {
		t.Errorf("unseen: %v, want %v", report.Unseen, want)
	}
	expect(t, s.page("/"), `<p>new: 4</p>`)
	expect(t, s.page("/about"), `<p>Jug</p>`)
}

// The data database is asked where data is loaded: in the script of a page,
// a layout or a shared block. Elsewhere the mistake is told with what to do.
func TestWhoMayAskTheData(t *testing.T) {
	cases := []struct{ name, files, want string }{
		{"a template asks", `-- pages/index.shugo --
<p>{Shugo.query("products").count()}</p>`, "a template cannot ask the data database: it shows what its script gives it"},

		{"a component that asks is given an object", `-- components/More.shugo --
---
const { item } = Shugo.props
const more = Shugo.query("products").where({ category: item.category }).count()
---
<p>{more}</p>
-- pages/index.shugo --
---
import More from "../components/More.shugo"
const item = Shugo.query("products").get("p1")
---
<More item={item} />`, "components/More.shugo loads data: its script asks the data database (Shugo.query). So it is computed once for the whole site and takes only plain values: text, numbers, true, false, null. Here item is an object"},

		{"a component that asks reads the page", `-- components/Here.shugo --
---
const page = Shugo.paginate(Shugo.query("products"), { pageSize: 2 })
---
<p>{page.total}</p>
-- pages/index.shugo --
---
import Here from "../components/Here.shugo"
---
<Here />`, "this component loads data — its script asks the data database (Shugo.query) — so it is computed once for the whole site; and it reads the page it is shown on"},

		{"a function that a template calls asks", `-- lib/ask.ts --
export const count = (Shugo: any) => Shugo.query("products").count()
-- pages/index.shugo --
---
import { count } from "../lib/ask"
---
<p>{count(Shugo)}</p>`, "Shugo.query is called in a template, or in code that a template calls"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSite(t, "-- shugo.json --\n{\"data\": \"data/shop.db\"}\n"+c.files)
			s.sql(readRows)
			_, err := s.tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("%v\nwant: %s", err, c.want)
			}
		})
	}
}
