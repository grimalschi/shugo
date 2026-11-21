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
-- components/Frame.shugo --
<div><g-slot /></div>
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
-- pages/local.shugo --
---
const item = Shugo.query("products").get("p1")
const cost = (row) => row.price + 1
console.log("local")
---
<p>{cost(item)}</p>
-- pages/component.shugo --
---
import Row from "../components/Row.shugo"
const item = Shugo.query("products").get("p1")
console.log("component")
---
<ul><Row row={item} /></ul>
-- pages/fill.shugo --
---
import Frame from "../components/Frame.shugo"
const item = Shugo.query("products").get("p1")
console.log("fill")
---
<Frame>{item.title}</Frame>
-- pages/asking.shugo --
<p>{Shugo.query("products").get("p1").title}{console.log("asking")}</p>
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

// A page depends on the columns that are read of the rows it asked for
// while it is computed — by its script, its template, the functions they
// call and the components the template gives a row to — and on no others.
func TestColumnsThatAreRead(t *testing.T) {
	s := newSite(t, readSite)
	s.sql(readRows)
	s.build()
	s.computed()
	expect(t, s.page("/script"), `<p>20</p>`)
	expect(t, s.page("/function"), `<p>JUG</p>`)
	expect(t, s.page("/local"), `<p>11</p>`)
	expect(t, s.page("/component"), `<ul><li>Jug</li></ul>`)
	expect(t, s.page("/asking"), `<p>Jug</p>`)
	expect(t, s.page("/joined"), `<p>Jug in Kitchen</p>`)
	expect(t, s.page("/handed/vase"), `<p>Vase</p>`)

	// A column nobody reads of the product: only who took the product as a
	// whole depends on it, and a spread and JSON.stringify do. A function
	// and a component that are given the row read the title of it.
	s.changed(`UPDATE products SET note = 'checked' WHERE id = 'p1'`, "spread", "written")
	// A column the script read, one it only asked about, and one that a
	// function of the script reads when the template calls it.
	s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "asked", "local", "script", "spread", "written")
	expect(t, s.page("/script"), `<p>24</p>`)
	expect(t, s.page("/local"), `<p>13</p>`)
	// A column that is shown: by a template, by a function it calls, by a
	// component it gives the row to, by what it puts into a shared block,
	// by a template that asks the database itself.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`,
		"alone", "asking", "component", "fill", "function", "joined", "spread", "template", "together", "written")
	expect(t, s.page("/template"), `<p>Big jug</p>`)
	expect(t, s.page("/function"), `<p>BIG JUG</p>`)
	expect(t, s.page("/component"), `<ul><li>Big jug</li></ul>`)
	expect(t, s.page("/fill"), `<div>Big jug</div>`)
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
	s.changed(`UPDATE products SET category = 'c2' WHERE id = 'p1'`, "handed jug", "handed vase", "joined", "spread", "together", "written")
	expect(t, s.page("/joined"), `<p>Big jug in Garden</p>`)

	// A list of pages that reads the addresses alone does not depend on a
	// title. One that hands its rows to its pages depends on all of them:
	// what a page reads of a row is read after the list is made.
	s.changed(`UPDATE products SET title = 'Tall vase' WHERE id = 'p4'`, "handed jug", "handed vase")
	expect(t, s.page("/handed/vase"), `<p>Tall vase</p>`)
	s.changed(`UPDATE products SET note = 'fragile' WHERE id = 'p4'`, "handed jug", "handed vase")
	// A product that gets another address: its page moves.
	s.changed(`UPDATE products SET slug = 'pitcher' WHERE id = 'p1'`, "handed pitcher", "handed vase", "listed pitcher", "spread", "written")
	if s.has("/listed/jug") || !s.has("/listed/pitcher") {
		t.Errorf("pages: %v", s.addresses())
	}

	// The row goes: everyone who asked for it by its key is told.
	s.changed(`DELETE FROM products WHERE id = 'p2'`, "alone", "together")
	s.sameAsBuilt()
}

const pathRows = `
CREATE TABLE products (id TEXT PRIMARY KEY, title TEXT, price INTEGER, available INTEGER, arrival TEXT, note TEXT, position INTEGER);
INSERT INTO products VALUES
  ('p1', 'Jug', 10, 1, 'in May', 'plain', 1), ('p2', 'Cup', 5, 1, 'in June', 'plain', 2), ('p3', 'Tray', 7, 0, 'in July', 'plain', 3);
`

// What is read is what was read on the way that was taken. A template that
// does not go into a branch does not read what the branch shows; when the
// data sends it there, the change that did so is one it had read.
func TestWaysThatAreTaken(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- lib/text.ts --
// What it reads of the row depends on the row.
export const label = (row) => (row.price > 6 ? row.title : row.note)
-- pages/branch.shugo --
---
const item = Shugo.query("products").get("p1")
console.log("branch")
---
<g-if cond={item.available}><p>in stock</p></g-if><g-else><p>comes {item.arrival}</p></g-else>
-- pages/either.shugo --
---
const item = Shugo.query("products").get("p1")
console.log("either")
---
<p>{item.note || item.title}</p>
-- pages/function.shugo --
---
import { label } from "../lib/text"
const item = Shugo.query("products").get("p2")
console.log("function")
---
<p>{label(item)}</p>
-- pages/loop.shugo --
---
const rows = Shugo.query("products").where({ available: true }).order("position").all()
console.log("loop")
---
<ul><g-for each={rows} as="row"><li>{row.title}</li></g-for></ul>
-- pages/first.shugo --
---
const rows = Shugo.query("products").order("position").limit(2).all()
console.log("first")
---
<ul><g-for each={rows} as="row"><li><g-if cond={row.price > 6}>{row.title}</g-if><g-else>cheap</g-else></li></g-for></ul>
`)
	s.sql(pathRows)
	s.build()
	s.computed()
	expect(t, s.page("/branch"), `<p>in stock</p>`)
	expect(t, s.page("/either"), `<p>plain</p>`)
	expect(t, s.page("/function"), `<p>plain</p>`)
	expect(t, s.page("/first"), `<ul><li>Jug</li><li>cheap</li></ul>`)

	// The branch that shows the arrival is not taken: the arrival reaches
	// nobody. The product stops being available: the page is computed, goes
	// the other way and reads the arrival, and from then on depends on it.
	s.changed(`UPDATE products SET arrival = 'in August' WHERE id = 'p1'`)
	s.changed(`UPDATE products SET available = 0 WHERE id = 'p1'`, "branch", "loop")
	expect(t, s.page("/branch"), `<p>comes in August</p>`)
	s.changed(`UPDATE products SET arrival = 'in September' WHERE id = 'p1'`, "branch")
	expect(t, s.page("/branch"), `<p>comes in September</p>`)
	// And back: the arrival is let go again.
	s.changed(`UPDATE products SET available = 1 WHERE id = 'p1'`, "branch", "loop")
	s.changed(`UPDATE products SET arrival = 'in October' WHERE id = 'p1'`)

	// a || b reads b only when a is nothing.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "first", "loop")
	s.changed(`UPDATE products SET note = '' WHERE id = 'p1'`, "either")
	expect(t, s.page("/either"), `<p>Big jug</p>`)
	s.changed(`UPDATE products SET title = 'Jug' WHERE id = 'p1'`, "either", "first", "loop")

	// A function reads the note of a cheap product and the title of a dear
	// one; which of them it is, it knows from the price it read.
	s.changed(`UPDATE products SET title = 'Mug' WHERE id = 'p2'`, "loop")
	s.changed(`UPDATE products SET note = 'small' WHERE id = 'p2'`, "function")
	s.changed(`UPDATE products SET price = 9 WHERE id = 'p2'`, "first", "function")
	expect(t, s.page("/function"), `<p>Mug</p>`)
	s.changed(`UPDATE products SET note = 'tiny' WHERE id = 'p2'`)
	s.changed(`UPDATE products SET title = 'Cup' WHERE id = 'p2'`, "first", "function", "loop")

	// A loop gets longer: the row that comes is one the question looked for.
	// What the loop reads of it, it reads when it is computed again.
	s.changed(`UPDATE products SET note = 'new' WHERE id = 'p3'`)
	s.changed(`UPDATE products SET available = 1 WHERE id = 'p3'`, "loop")
	expect(t, s.page("/loop"), `<ul><li>Jug</li><li>Cup</li><li>Tray</li></ul>`)
	s.changed(`UPDATE products SET title = 'Big tray' WHERE id = 'p3'`, "loop")
	s.changed(`INSERT INTO products VALUES ('p0', 'Pot', 3, 1, 'in May', '', 0)`, "first", "loop")
	expect(t, s.page("/first"), `<ul><li>cheap</li><li>Jug</li></ul>`)
	// Rows behind a limit count alone, and each in what was read of it: of
	// the cheap one the price, of the dear one the price and the title.
	s.changed(`UPDATE products SET title = 'Bowl' WHERE id = 'p0'`, "loop")
	s.changed(`UPDATE products SET price = 8 WHERE id = 'p0'`, "first")
	expect(t, s.page("/first"), `<ul><li>Bowl</li><li>Jug</li></ul>`)
	s.changed(`UPDATE products SET title = 'Pot' WHERE id = 'p0'`, "first", "loop")
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

// A row is an object like any other, on Bun and on Node. Shown as a whole
// it is what every object is, and reads nothing; only its columns tell of
// the data, whatever else is asked of it.
func TestRowIsAnOrdinaryObject(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/index.shugo --
---
const item = Shugo.query("products").get("p1")
const priced = item.hasOwnProperty("price")
const same = item.then === undefined && item.toString === Object.prototype.toString && item.length === undefined
console.log("index")
---
<p title={item}>{priced} {same} {item} {[item, item]}</p>
-- pages/attributes.shugo --
---
const item = Shugo.query("products").select("title", "note").get("p1")
console.log("attributes")
---
<p {...item}>x</p>
`)
	s.sql(readRows)
	s.build()
	s.computed()
	expect(t, s.page("/"), `<p title="[object Object]">true true [object Object] [object Object][object Object]</p>`)
	expect(t, s.page("/attributes"), `<p title="Jug" note="" id="p1">x</p>`)
	// Putting a row among attributes shows every column of it.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "attributes")
	s.changed(`UPDATE products SET note = 'x' WHERE id = 'p1'`, "attributes")
	s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "index")
}

// A module that keeps a row, and one that keeps Shugo, for whoever asks next.
const keptSite = `
-- shugo.json --
{"data": "data/shop.db"}
-- lib/kept.ts --
let row: { title: string } | undefined
export const jug = (Shugo: any) => (row ??= Shugo.query("products").get("p1"))
let asker: any
export const count = (Shugo: any) => (asker ??= Shugo).query("products").count()
`

// A row notes what is read of it for the one who asked for it. Kept in a
// module and read by another page, it would leave that page depending on
// nothing: the row refuses, and says what to do. So does Shugo itself, kept
// for another page to ask with. The page at / is computed first.
func TestRowKeptForAnotherScript(t *testing.T) {
	cases := []struct {
		name, files string
		want        []string
	}{
		{"a row read after its page is computed", keptSite + `
-- pages/index.shugo --
---
import { jug } from "../lib/kept"
const item = jug(Shugo)
---
<p>{item.title}</p>
-- pages/about.shugo --
---
import { jug } from "../lib/kept"
const item = jug(Shugo)
---
<p>{item.title}</p>
`, []string{
			"a row of the table products is read when the page, shared block or list of pages it belongs to is already computed",
			"let every script ask for the rows it uses",
			"while computing the page /about",
		}},
		{"a row that a list of pages keeps for its pages", `
-- shugo.json --
{"data": "data/shop.db"}
-- lib/kept.ts --
let first: { title: string } | undefined
export const keep = (row: { title: string }) => (first ??= row)
export const kept = () => first
-- pages/[slug].shugo --
---
import { keep, kept } from "../lib/kept"
export const getStaticPaths = ({ query }) => query("products").all().map((row) => (keep(row), { params: { slug: row.slug } }))
---
<p>{kept().title}</p>
`, []string{"a row of the table products is read when the page, shared block or list of pages it belongs to is already computed"}},
		{"Shugo asked after its page is computed", keptSite + `
-- pages/index.shugo --
---
import { count } from "../lib/kept"
---
<p>{count(Shugo)}</p>
-- pages/about.shugo --
---
import { count } from "../lib/kept"
---
<p>{count(Shugo)}</p>
`, []string{
			"the table products is asked when the page, shared block or list of pages it belongs to is already computed",
			"do not keep Shugo, or a query made of it, in a variable of a module",
			"while computing the page /about",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSite(t, c.files)
			s.sql(readRows)
			_, err := s.tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			for _, want := range c.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%v\nwant: %s", err, want)
				}
			}
		})
	}
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
-- lib/kept.ts --
// A value that is not a row, kept for whoever comes next: nothing tells
// that the second page shows what the first one read.
let kept: string | undefined
export const title = (Shugo: any) => (kept ??= Shugo.query("products").get("p2").title)
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
-- pages/one.shugo --
---
import { title } from "../lib/kept"
---
<p>{title(Shugo)}</p>
-- pages/two.shugo --
---
import { title } from "../lib/kept"
---
<p>{title(Shugo)}</p>
`)
	s.sql(readRows)
	s.build()

	// Nothing differs: the check finds nothing.
	report, err := s.run(evaluate.Options{Check: true})
	if err != nil || !report.Checked || len(report.Unseen) != 0 || report.Computed.Pages != 4 {
		t.Fatalf("a check of an unchanged site: %v, %+v", err, report)
	}

	// Rows change and a file does. The update sees the rows that were read.
	s.write("-- note.txt --\nnew\n")
	s.sql(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'; UPDATE products SET title = 'Mug' WHERE id = 'p2'`)
	if report := s.update(); report.Computed.Pages != 2 {
		t.Errorf("a plain update computed %+v", report.Computed)
	}
	expect(t, s.page("/"), `<p>old: 4</p>`)
	expect(t, s.page("/about"), `<p>Big jug</p>`)
	expect(t, s.page("/one"), `<p>Mug</p>`)
	expect(t, s.page("/two"), `<p>Cup</p>`)

	// The check sees the file and the kept value too, and tells which pages
	// showed them. A page that changes with a row it read is no finding.
	s.sql(`UPDATE products SET title = 'Jug' WHERE id = 'p1'`)
	report, err = s.run(evaluate.Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"the page /", "the page /two"}; !reflect.DeepEqual(report.Unseen, want) {
		t.Errorf("unseen: %v, want %v", report.Unseen, want)
	}
	expect(t, s.page("/"), `<p>new: 4</p>`)
	expect(t, s.page("/about"), `<p>Jug</p>`)
	expect(t, s.page("/two"), `<p>Mug</p>`)
}

// The parameters of a page are read from its address, whether the page is
// computed from the list of its route or by its address alone: the two
// must give the same page.
func TestParametersComeFromTheAddress(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/price/[price].shugo --
---
export const getStaticPaths = ({ query }) => query("products").select("price").all().map(({ price }) => ({ params: { price, more: "no" } }))
const item = Shugo.query("products").where({ price: Number(Shugo.params.price) }).first()
console.log("price " + Shugo.params.price)
---
<p>{typeof Shugo.params.price} {String(Shugo.params.more)} {item.title}</p>
`)
	s.sql(readRows)
	s.build()
	s.computed()
	expect(t, s.page("/price/10"), `<p>string undefined Jug</p>`)
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "price 10")
	expect(t, s.page("/price/10"), `<p>string undefined Big jug</p>`)
	s.sameAsBuilt()
}

// shugo update -check ends with an error when a page shows what Shugo did
// not see it read, and says where to look.
func TestCheckCommand(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- note.txt --
old
-- pages/index.shugo --
---
import { readFileSync } from "node:fs"
const note = readFileSync("note.txt", "utf8").trim()
const item = Shugo.query("products").get("p1")
---
<p>{note}: {item.title}</p>
`)
	s.sql(readRows)
	s.store.Close() // the commands open the store themselves
	if out, errs, status := shugo(t, "build", s.root); status != 0 {
		t.Fatalf("build: %d\n%s\n%s", status, out, errs)
	}

	s.sql(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`)
	out, errs, status := shugo(t, "update", "-check", s.root)
	if status != 0 || !strings.Contains(out, "checked: everything was computed") || !strings.Contains(out, "pages: 0 added, 1 changed") {
		t.Errorf("a check with nothing unseen: %d\n%s\n%s", status, out, errs)
	}

	s.write("-- note.txt --\nnew\n")
	out, errs, status = shugo(t, "update", "-check", s.root)
	for _, want := range []string{
		"1 pages or shared blocks came out different although no row they read has changed; the first: the page /.",
		"Put that data into the database and ask for it with Shugo.query",
	} {
		if status != 1 || !strings.Contains(errs, want) {
			t.Errorf("a check with a page that read a file: %d\n%s\n%s\nwant: %s", status, out, errs, want)
		}
	}
	// The store is right after the check all the same.
	if out, _, status := shugo(t, "update", "-check", s.root); status != 0 || !strings.Contains(out, "nothing has changed") {
		t.Errorf("the check again: %d\n%s", status, out)
	}
}
