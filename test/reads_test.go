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
-- src/lib/text.ts --
export const shout = (row: { title: string }) => row.title.toUpperCase()
-- src/components/Row.svelte --
<li>{$props().row.title}</li>
-- src/components/Frame.svelte --
<script>
let { children } = $props()
</script>
<div>{@render children?.()}</div>
-- src/pages/script.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
const cost = item.price * 2
console.log("script")
</script>
<p>{cost}</p>
-- src/pages/template.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
console.log("template")
</script>
<p>{item.title}</p>
-- src/pages/neither.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
console.log("neither")
</script>
<p>{#if item}there{:else}gone{/if}</p>
-- src/pages/asked.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
const priced = "price" in item
console.log("asked")
</script>
<p>{priced}</p>
-- src/pages/spread.svelte --
<script>
import { query } from "shugo"
const copy = { ...query("products").get("p1") }
console.log("spread")
</script>
<p>{copy.title}</p>
-- src/pages/written.svelte --
<script>
import { query } from "shugo"
const text = JSON.stringify(query("products").get("p1")).length > 0
console.log("written")
</script>
<p>{text}</p>
-- src/pages/function.svelte --
<script>
import { query } from "shugo"
import { shout } from "../lib/text"
const item = query("products").get("p1")
console.log("function")
</script>
<p>{shout(item)}</p>
-- src/pages/local.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
const cost = (row) => row.price + 1
console.log("local")
</script>
<p>{cost(item)}</p>
-- src/pages/component.svelte --
<script>
import { query } from "shugo"
import Row from "../components/Row.svelte"
const item = query("products").get("p1")
console.log("component")
</script>
<ul><Row row={item} /></ul>
-- src/pages/fill.svelte --
<script>
import { query } from "shugo"
import Frame from "../components/Frame.svelte"
const item = query("products").get("p1")
console.log("fill")
</script>
<Frame>{item.title}</Frame>
-- src/pages/asking.svelte --
<script>
import { query } from "shugo"
</script>
<p>{query("products").get("p1").title}{console.log("asking")}</p>
-- src/pages/together.svelte --
<script>
import { query } from "shugo"
const rows = query("products").where({ category: "c1" }).order("position").all()
console.log("together")
</script>
<ul>{#each rows as row}<li>{row.title}</li>{/each}</ul>
-- src/pages/alone.svelte --
<script>
import { query } from "shugo"
const rows = query("products").order("position").limit(2).all()
console.log("alone")
</script>
<ul>{#each rows as row}<li>{row.title}</li>{/each}</ul>
-- src/pages/joined.svelte --
<script>
import { query } from "shugo"
const item = query("products").join("categories", "category").get("p1")
console.log("joined")
</script>
<p>{item.title} in {item.categories.title}</p>
-- src/pages/listed/[slug].svelte --
<script module>
import { query, page } from "shugo"
export const getStaticPaths = () => query("products").all().map(({ slug }) => ({ params: { slug } }))
</script>
<script>
console.log("listed " + page.params.slug)
</script>
<p>{page.params.slug}</p>
-- src/pages/garden/[slug].svelte --
<script module>
import { query, page } from "shugo"
export const getStaticPaths = () => query("products").select("slug").where({ category: "c2" }).all().map(({ slug }) => ({ params: { slug } }))
</script>
<script>
const row = query("products").where({ slug: page.params.slug }).first()
console.log("garden " + page.params.slug)
</script>
<p>{row.title}</p>
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
	expect(t, s.page("/garden/vase"), `<p>Vase</p>`)

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
	// The product moves into the category a list of pages reads: the list
	// gains a page, and the other pages of the list are not computed.
	s.changed(`UPDATE products SET category = 'c2' WHERE id = 'p1'`, "garden jug", "joined", "spread", "together", "written")
	expect(t, s.page("/joined"), `<p>Big jug in Garden</p>`)

	// A list of pages reads the addresses alone and does not depend on a
	// title; the page of the row does.
	s.changed(`UPDATE products SET title = 'Tall vase' WHERE id = 'p4'`, "garden vase")
	expect(t, s.page("/garden/vase"), `<p>Tall vase</p>`)
	s.changed(`UPDATE products SET note = 'fragile' WHERE id = 'p4'`)
	// A product that gets another address: its pages move.
	s.changed(`UPDATE products SET slug = 'pitcher' WHERE id = 'p1'`, "garden pitcher", "listed pitcher", "spread", "written")
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
-- src/lib/text.ts --
// What it reads of the row depends on the row.
export const label = (row) => (row.price > 6 ? row.title : row.note)
-- src/pages/branch.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
console.log("branch")
</script>
{#if item.available}<p>in stock</p>{:else}<p>comes {item.arrival}</p>{/if}
-- src/pages/either.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
console.log("either")
</script>
<p>{item.note || item.title}</p>
-- src/pages/function.svelte --
<script>
import { query } from "shugo"
import { label } from "../lib/text"
const item = query("products").get("p2")
console.log("function")
</script>
<p>{label(item)}</p>
-- src/pages/loop.svelte --
<script>
import { query } from "shugo"
const rows = query("products").where({ available: true }).order("position").all()
console.log("loop")
</script>
<ul>{#each rows as row}<li>{row.title}</li>{/each}</ul>
-- src/pages/first.svelte --
<script>
import { query } from "shugo"
const rows = query("products").order("position").limit(2).all()
console.log("first")
</script>
<ul>{#each rows as row}<li>{#if row.price > 6}{row.title}{:else}cheap{/if}</li>{/each}</ul>
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
-- src/pages/index.svelte --
<script>
import { query } from "shugo"
const item = query("products").select("title", "price", "note").get("p1")
console.log("index")
</script>
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
-- src/pages/index.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
const priced = item.hasOwnProperty("price")
const same = item.then === undefined && item.toString === Object.prototype.toString && item.length === undefined
console.log("index")
</script>
<p title={item}>{priced} {same} {item} {[item, item]}</p>
-- src/pages/attributes.svelte --
<script>
import { query } from "shugo"
const item = query("products").select("title", "note").get("p1")
console.log("attributes")
</script>
<p {...item}>x</p>
`)
	s.sql(readRows)
	s.build()
	s.computed()
	expect(t, s.page("/"), `<p title="[object Object]">true true [object Object] [object Object],[object Object]</p>`)
	expect(t, s.page("/attributes"), `<p title="Jug" note="" id="p1">x</p>`)
	// Putting a row among attributes shows every column of it.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "attributes")
	s.changed(`UPDATE products SET note = 'x' WHERE id = 'p1'`, "attributes")
	s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "index")
}

// A module that keeps a row, and one that keeps a query, for whoever asks
// next.
const keptSite = `
-- shugo.json --
{"data": "data/shop.db"}
-- src/lib/kept.ts --
import { query } from "shugo"
let row: { title: string } | undefined
export const jug = () => (row ??= query("products").get("p1"))
let asked: any
export const count = () => (asked ??= query("products")).count()
`

// A row notes what is read of it for the one who asked for it. Kept in a
// module and read by another page, it would leave that page depending on
// nothing: the row refuses, and says what to do. So does a query, kept for
// another page to ask. The page at / is computed first.
func TestRowKeptForAnotherScript(t *testing.T) {
	cases := []struct {
		name, files string
		want        []string
	}{
		{"a row read after its page is computed", keptSite + `
-- src/pages/index.svelte --
<script>
import { jug } from "../lib/kept"
const item = jug()
</script>
<p>{item.title}</p>
-- src/pages/about.svelte --
<script>
import { jug } from "../lib/kept"
const item = jug()
</script>
<p>{item.title}</p>
`, []string{
			"a row of the table products is read when the page, shared block or list of pages it belongs to is already computed",
			"let every script ask for the rows it uses",
			"while computing the page /about",
		}},
		{"a row that a list of pages keeps for its pages", `
-- shugo.json --
{"data": "data/shop.db"}
-- src/lib/kept.ts --
let first: { title: string } | undefined
export const keep = (row: { title: string }) => (first ??= row)
export const kept = () => first
-- src/pages/[slug].svelte --
<script module>
import { query } from "shugo"
import { keep, kept } from "../lib/kept"
export const getStaticPaths = () => query("products").all().map((row) => (keep(row), { params: { slug: row.slug } }))
</script>
<p>{kept().title}</p>
`, []string{"a row of the table products is read when the page, shared block or list of pages it belongs to is already computed"}},
		{"a query asked after its page is computed", keptSite + `
-- src/pages/index.svelte --
<script>
import { count } from "../lib/kept"
</script>
<p>{count()}</p>
-- src/pages/about.svelte --
<script>
import { count } from "../lib/kept"
</script>
<p>{count()}</p>
`, []string{
			"the table products is asked when the page, shared block or list of pages it belongs to is already computed",
			"do not keep a query in a variable of a module for others",
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
-- src/lib/kept.ts --
// A value that is not a row, kept for whoever comes next: nothing tells
// that the second page shows what the first one read.
import { query } from "shugo"
let kept: string | undefined
export const title = () => (kept ??= query("products").get("p2").title)
-- src/pages/index.svelte --
<script>
import { query } from "shugo"
import { readFileSync } from "node:fs"
const note = readFileSync("note.txt", "utf8").trim()
const count = query("products").count()
</script>
<p>{note}: {count}</p>
-- src/pages/about.svelte --
<script>
import { query } from "shugo"
const item = query("products").get("p1")
</script>
<p>{item.title}</p>
-- src/pages/one.svelte --
<script>
import { title } from "../lib/kept"
</script>
<p>{title()}</p>
-- src/pages/two.svelte --
<script>
import { title } from "../lib/kept"
</script>
<p>{title()}</p>
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
-- src/pages/price/[price].svelte --
<script module>
import { query, page } from "shugo"
export const getStaticPaths = () => query("products").select("price").all().map(({ price }) => ({ params: { price, more: "no" } }))
</script>
<script>
const item = query("products").where({ price: Number(page.params.price) }).first()
console.log("price " + page.params.price)
</script>
<p>{typeof page.params.price} {String(page.params.more)} {item.title}</p>
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
-- src/pages/index.svelte --
<script>
import { query } from "shugo"
import { readFileSync } from "node:fs"
const note = readFileSync("note.txt", "utf8").trim()
const item = query("products").get("p1")
</script>
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
		"Put that data into the database and ask for it with query from \"shugo\"",
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
