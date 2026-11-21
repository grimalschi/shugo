package test

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/evaluate"
	"github.com/grimalschi/shugo/internal/store"
)

// A shop that keeps its data in a database. Every page and shared block
// prints its name when it is computed, so a test sees exactly what an update
// computed.
const dataShop = `
-- shugo.json --
{"data": "data/shop.db"}
-- components/Card.shugo --
---
const { id } = Shugo.props as { id: string }
const item = Shugo.query("products").select("slug", "title", "price").get(id)
if (!item) throw new Error("there is no product " + id)
console.log("card " + id)
---
<a href={"/products/" + item.slug}>{item.title} — {item.price}</a>
-- components/Header.shugo --
---
const shop = Shugo.query("shop").select("name").get(1)
const count = Shugo.query("products").count()
console.log("header")
---
<header>{shop.name} · {count} products</header>
-- pages/+layout.shugo --
---
import Header from "../components/Header.shugo"
---
<html><body><Header /><main><g-slot /></main></body></html>
-- pages/index.shugo --
---
import Card from "../components/Card.shugo"
const top = Shugo.query("products").select("id").order("position").limit(2).all()
console.log("index")
---
<ul><g-for each={top} as="p"><li><Card id={p.id} /></li></g-for></ul>
-- pages/products/[slug].shugo --
---
import Card from "../../components/Card.shugo"
export const getStaticPaths = ({ query }) => query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
const item = Shugo.query("products").select("title", "price")
  .where({ slug: Shugo.params.slug }).join("categories", "category", { select: ["title"], as: "group" }).first()
const reviews = Shugo.query("reviews").select("text").where({ product: item.id, approved: true }).all()
const related = Shugo.query("related").select("other").where({ product: item.id }).order("position").all()
console.log("page " + item.id)
---
<h1>{item.title}</h1><p>{item.price} · {item.group?.title}</p>
<ul><g-for each={reviews} as="r"><li>{r.text}</li></g-for></ul>
<nav><g-for each={related} as="r"><Card id={r.other} /></g-for></nav>
-- pages/categories/[id]/[...page].shugo --
---
import Card from "../../../components/Card.shugo"
export const getStaticPaths = ({ query, paginate }) =>
  query("categories").select("id").all().flatMap(({ id }) => paginate(query("products").where({ category: id }).count(), { pageSize: 2, params: { id } }))
const category = Shugo.query("categories").get(Shugo.params.id)
const page = Shugo.paginate(Shugo.query("products").select("id").where({ category: category.id }).order("position"), { pageSize: 2 })
console.log("list " + Shugo.url.pathname)
---
<h1>{category.title}: {page.total}, page {page.currentPage} of {page.lastPage}</h1>
<g-for each={page.data} as="p"><Card id={p.id} /></g-for>
<g-if cond={page.url.next}><a href={page.url.next}>next</a></g-if>
`

const dataShopRows = `
CREATE TABLE shop (id INTEGER PRIMARY KEY, name TEXT);
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT UNIQUE, title TEXT, price INTEGER, note TEXT, category TEXT, position INTEGER);
CREATE TABLE reviews (id INTEGER PRIMARY KEY, product TEXT, text TEXT, approved INTEGER);
CREATE TABLE related (id INTEGER PRIMARY KEY, product TEXT, other TEXT, position INTEGER);
INSERT INTO shop VALUES (1, 'Clay');
INSERT INTO categories VALUES ('c1', 'Kitchen'), ('c2', 'Garden');
INSERT INTO products VALUES
  ('p1', 'jug', 'Jug', 10, '', 'c1', 1), ('p2', 'cup', 'Cup', 5, '', 'c1', 2),
  ('p3', 'tray', 'Tray', 7, '', 'c1', 3), ('p4', 'vase', 'Vase', 9, '', 'c2', 4);
INSERT INTO reviews (product, text, approved) VALUES ('p1', 'Solid', 1), ('p1', 'Spam', 0);
INSERT INTO related (product, other, position) VALUES ('p1', 'p2', 1), ('p1', 'p3', 2);
`

// sql changes the data database of the site the way any program would: over
// a connection of its own.
func (s *site) sql(statements string) {
	s.t.Helper()
	path := filepath.Join(s.root, "data", "shop.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		s.t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		s.t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(statements); err != nil {
		s.t.Fatalf("%s: %v", statements, err)
	}
}

// computed returns what the project's code printed since it was asked last,
// sorted: the names of the pages and shared blocks that were computed.
func (s *site) computed() []string {
	var lines []string
	for _, line := range strings.Split(s.printed.String(), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	s.printed.Reset()
	sort.Strings(lines)
	return lines
}

// changed runs statements against the data, updates the site and checks
// what the update computed.
func (s *site) changed(statements string, want ...string) {
	s.t.Helper()
	s.sql(statements)
	s.update()
	if got := s.computed(); !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
		s.t.Errorf("%s\n computed: %q\n     want: %q", strings.TrimSpace(statements), got, want)
	}
}

// addresses returns the addresses of all pages the store holds.
func (s *site) addresses() []string {
	s.t.Helper()
	var out []string
	err := s.store.View(func(v *store.View) error {
		return v.Pages(func(path string, _ []byte) error {
			out = append(out, path)
			return nil
		})
	})
	if err != nil {
		s.t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// sameAsBuilt checks that the site, updated step by step, is what a build
// from nothing makes of the same files and the same data.
func (s *site) sameAsBuilt() {
	s.t.Helper()
	fresh := openSite(s.t, s.t.TempDir())
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, err error) error {
		rel, _ := filepath.Rel(s.root, path)
		if err != nil || entry.IsDir() || strings.HasPrefix(rel, "_build") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(fresh.root, rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(fresh.root, rel), data, 0o644)
	})
	if err != nil {
		s.t.Fatal(err)
	}
	fresh.build()
	if got, want := s.addresses(), fresh.addresses(); !reflect.DeepEqual(got, want) {
		s.t.Fatalf("pages: %v\n built anew: %v", got, want)
	}
	for _, address := range fresh.addresses() {
		if got, want := s.page(address), fresh.page(address); got != want {
			s.t.Errorf("%s\n  updated: %s\nbuilt anew: %s", address, got, want)
		}
	}
}

func TestDataDatabase(t *testing.T) {
	s := newSite(t, dataShop)
	s.sql(dataShopRows)
	first := s.build()
	if first.Data == nil || !first.Data.Whole || first.Pages.Total != 8 || first.Shared.Total != 6 {
		t.Fatalf("the first build: %+v, data %+v", first, first.Data)
	}
	s.computed()
	expect(t, s.page("/products/jug"), `<html><body><header>Clay · 4 products</header><main><h1>Jug</h1><p>10 · Kitchen</p>
<ul><li>Solid</li></ul>
<nav><a href="/products/cup">Cup — 5</a><a href="/products/tray">Tray — 7</a></nav></main></body></html>`)
	expect(t, s.page("/categories/c1"), `<html><body><header>Clay · 4 products</header><main><h1>Kitchen: 3, page 1 of 2</h1>
<a href="/products/jug">Jug — 10</a><a href="/products/cup">Cup — 5</a>
<a href="/categories/c1/2">next</a></main></body></html>`)

	t.Run("nothing changes", func(t *testing.T) {
		report := s.update()
		if report.Computed.Pages+report.Computed.Shared != 0 || report.Data.Whole || report.Data.Rows != 0 {
			t.Errorf("%+v, data %+v", report.Computed, report.Data)
		}
		s.changed(`UPDATE products SET price = 10 WHERE id = 'p1'`)
	})

	t.Run("a column nobody took", func(t *testing.T) {
		s.changed(`UPDATE products SET note = 'checked'`)
	})

	t.Run("the price of a product", func(t *testing.T) {
		s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "card p1", "page p1")
		if !strings.Contains(s.page("/"), "Jug — 12") || !strings.Contains(s.page("/categories/c1"), "Jug — 12") {
			t.Errorf("the card is not seen where it stands: %s", s.page("/"))
		}
	})

	t.Run("rows a filter leaves out", func(t *testing.T) {
		// A review that is not approved is not what the page asked for.
		s.changed(`INSERT INTO reviews (product, text, approved) VALUES ('p1', 'More spam', 0)`)
		s.changed(`UPDATE reviews SET text = 'Spam!' WHERE text = 'Spam'`)
		s.changed(`UPDATE reviews SET approved = 1 WHERE text = 'Spam!'`, "page p1")
		s.changed(`UPDATE reviews SET text = 'Sturdy' WHERE text = 'Solid'`, "page p1")
		s.changed(`DELETE FROM reviews WHERE text = 'Spam!'`, "page p1")
		// A review of another product.
		s.changed(`INSERT INTO reviews (product, text, approved) VALUES ('p4', 'Tall', 1)`, "page p4")
	})

	t.Run("a joined row", func(t *testing.T) {
		// The title of a category stands on the pages of its products and of itself.
		s.changed(`UPDATE categories SET title = 'Kitchenware' WHERE id = 'c1'`, "list /categories/c1", "list /categories/c1/2", "page p1", "page p2", "page p3")
		// A product that points at another category.
		s.changed(`UPDATE products SET category = 'c2' WHERE id = 'p3'`, "list /categories/c1", "list /categories/c2", "page p3")
		if s.has("/categories/c1/2") || !strings.Contains(s.page("/products/tray"), "Garden") {
			t.Errorf("after the move: %v, %s", s.addresses(), s.page("/products/tray"))
		}
		// A category that is gone leaves its products without one.
		s.changed(`UPDATE products SET category = 'c9' WHERE id = 'p3'`, "list /categories/c2", "page p3")
		s.changed(`INSERT INTO categories VALUES ('c9', 'Shelf')`, "list /categories/c9", "page p3")
	})

	t.Run("rows behind a limit", func(t *testing.T) {
		// The first page shows two products. A change of the third is not seen there…
		s.changed(`UPDATE products SET title = 'Big tray' WHERE id = 'p3'`, "card p3", "page p3")
		// …until the order brings it in.
		s.changed(`UPDATE products SET position = 0 WHERE id = 'p3'`, "index", "list /categories/c9")
		if !strings.Contains(s.page("/"), "Big tray") {
			t.Errorf("the first page: %s", s.page("/"))
		}
	})

	t.Run("a product is added", func(t *testing.T) {
		s.changed(`INSERT INTO products VALUES ('p5', 'bowl', 'Bowl', 4, '', 'c1', 5)`, "card p5", "header", "index", "list /categories/c1", "list /categories/c1/2", "page p5")
		if !s.has("/products/bowl") {
			t.Errorf("pages: %v", s.addresses())
		}
	})

	t.Run("a product gets another address", func(t *testing.T) {
		s.changed(`UPDATE products SET slug = 'mug' WHERE id = 'p2'`, "card p2", "page p2")
		if s.has("/products/cup") || !s.has("/products/mug") || !strings.Contains(s.page("/products/jug"), `href="/products/mug"`) {
			t.Errorf("pages: %v", s.addresses())
		}
	})

	t.Run("a product is removed", func(t *testing.T) {
		s.sql(`DELETE FROM related WHERE other = 'p3'; DELETE FROM products WHERE id = 'p3'`)
		report := s.update()
		s.computed()
		if s.has("/products/tray") || report.Pages.Removed != 1 || report.Shared.Removed != 1 {
			t.Errorf("pages %v, report %+v %+v", s.addresses(), report.Pages, report.Shared)
		}
	})

	t.Run("the same as built from nothing", func(t *testing.T) { s.sameAsBuilt() })
}

// tryUpdate brings the site up to date, as update does, and returns the
// failure instead of ending the test.
func (s *site) tryUpdate() (*evaluate.Report, error) {
	s.t.Helper()
	return s.run(evaluate.Options{})
}

// run updates the site the way the options say.
func (s *site) run(options evaluate.Options) (*evaluate.Report, error) {
	s.t.Helper()
	p := s.project()
	compiled, err := compile.Load(p)
	if err != nil {
		s.t.Fatal(err)
	}
	options.Log = &s.printed
	return evaluate.Run(p, compiled, s.store, options)
}

const queryRows = `
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT UNIQUE, title TEXT, price INTEGER, category TEXT);
CREATE VIEW cheap AS SELECT * FROM products WHERE price < 8;
INSERT INTO categories VALUES ('c1', 'Kitchen'), ('c2', 'Garden');
INSERT INTO products VALUES ('p1', 'jug', 'Jug', 10, 'c1'), ('p2', 'cup', 'Cup', 5, 'c1'), ('p3', 'tray', 'Tray', 7, 'c1'), ('p4', 'vase', 'Vase', 9, 'c2');
`

// Every page asks one question, so a test sees which questions a change of
// the data reaches.
func TestDataQueries(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/among.shugo --
---
const rows = Shugo.query("products").select("title").where("id", "in", ["p1", "p3"]).order("title").all()
console.log("among")
---
<p>{rows.map((row) => row.title).join(", ")}</p>
-- pages/cheap.shugo --
---
const rows = Shugo.query("products").select("title", "price").where("price", "<", 8).order("price", "desc").all()
console.log("cheap")
---
<p>{rows.map((row) => row.title + " " + row.price).join(", ")}</p>
-- pages/dearest.shugo --
---
const row = Shugo.query("products").select("title").where({ category: "c1" }).order("price", "desc").first()
console.log("dearest")
---
<p>{row.title}</p>
-- pages/joined.shugo --
---
const row = Shugo.query("products").select("title").join("categories", "category", { select: ["title"], as: "group" }).get("p4")
const other = Shugo.query("categories").join("products", "id", { on: "slug", select: ["title"], as: "namesake" }).get("c1")
console.log("joined")
---
<p>{row.title} in {row.group.title}; {String(other.namesake)}</p>
-- pages/count.shugo --
---
const count = Shugo.query("products").where("price", ">=", 7).where("category", "!=", null).count()
console.log("count")
---
<p>{count}</p>
`)
	s.sql(queryRows)
	s.build()
	s.computed()
	expect(t, s.page("/among"), `<p>Jug, Tray</p>`)
	expect(t, s.page("/cheap"), `<p>Tray 7, Cup 5</p>`)
	expect(t, s.page("/dearest"), `<p>Jug</p>`)
	expect(t, s.page("/joined"), `<p>Vase in Garden; null</p>`)
	expect(t, s.page("/count"), `<p>3</p>`)

	// A row that only one question took a column of.
	s.changed(`UPDATE products SET title = 'Mug' WHERE id = 'p2'`, "cheap")
	// A column that decides which rows two questions find.
	s.changed(`UPDATE products SET price = 6 WHERE id = 'p4'`, "cheap", "count")
	expect(t, s.page("/cheap"), `<p>Tray 7, Vase 6, Mug 5</p>`)
	expect(t, s.page("/count"), `<p>2</p>`)
	// The first of several rows is a reading of its own; a question without
	// a limit reads every row its filter may find.
	s.changed(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`, "among", "cheap", "dearest")
	// A row a join found.
	s.changed(`UPDATE categories SET title = 'Yard' WHERE id = 'c2'`, "joined")
	expect(t, s.page("/joined"), `<p>Vase in Yard; null</p>`)
	// A row a join looked for and did not find. A question that is not
	// asked by equal values may find any new row, so it is asked again.
	s.changed(`INSERT INTO products VALUES ('p5', 'c1', 'Namesake', 1, 'c2')`, "cheap", "count", "joined")
	expect(t, s.page("/joined"), `<p>Vase in Yard; [object Object]</p>`)
	// The order of the rows one question cuts.
	s.changed(`UPDATE products SET price = 11 WHERE id = 'p3'`, "cheap", "count", "dearest")
	expect(t, s.page("/dearest"), `<p>Tray</p>`)
	// A row of a list of keys goes.
	s.changed(`DELETE FROM products WHERE id = 'p3'`, "among", "cheap", "count", "dearest")
	expect(t, s.page("/among"), `<p>Big jug</p>`)
	expect(t, s.page("/dearest"), `<p>Big jug</p>`)
	s.sameAsBuilt()
}

func TestDataQueryMistakes(t *testing.T) {
	for name, mistake := range map[string][2]string{
		"no table":                 {`Shugo.query("goods").all()`, `there is no table "goods" in the data database; it has: categories, products`},
		"a view":                   {`Shugo.query("cheap").all()`, `cheap is a view`},
		"the journal":              {`Shugo.query("_shugo_journal").all()`, `there is no table "_shugo_journal"`},
		"no column":                {`Shugo.query("products").select("name").all()`, `there is no column "name" in the table products (select); it has: id, slug, title, price, category`},
		"a number for a text":      {`Shugo.query("products").where({ category: 5 }).all()`, `products.category holds text, and it is compared with the number 5: write "5"`},
		"a text for a number":      {`Shugo.query("products").where({ price: "5" }).all()`, `products.price holds numbers, and it is compared with the text "5": write 5`},
		"a join to many rows":      {`Shugo.query("categories").join("products", "id", { on: "category" }).all()`, `products.category may hold the same value in several rows`},
		"a column compared twice":  {`Shugo.query("products").where({ id: "p1" }).where({ id: "p2" }).all()`, `products.id is already compared with a value`},
		"an unknown comparison":    {`Shugo.query("products").where("price", "~", 5).all()`, `the comparisons are =, !=, <, <=, >, >=, like, in`},
		"a join that hides a name": {`Shugo.query("products").join("categories", "category", { as: "title" }).all()`, `a row already has "title"`},
	} {
		s := newSite(t, "-- shugo.json --\n{\"data\": \"data/shop.db\"}\n-- pages/index.shugo --\n---\nconst rows = "+mistake[0]+"\n---\n<p>{rows.length}</p>\n")
		s.sql(queryRows)
		if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), mistake[1]) {
			t.Errorf("%s: %v", name, err)
		}
	}

	s := newSite(t, "-- pages/index.shugo --\n---\nconst rows = Shugo.query(\"products\").all()\n---\n<p>{rows.length}</p>\n")
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "this project names none") {
		t.Errorf("no data database: %v", err)
	}
	s = newSite(t, "-- shugo.json --\n{\"data\": \"data/shop.db\"}\n-- pages/index.shugo --\n<p>Hi</p>\n")
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "is not there") {
		t.Errorf("a data database that is not there: %v", err)
	}
	s.sql(`CREATE TABLE links (a TEXT, b TEXT, PRIMARY KEY (a, b))`)
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "the table links needs a primary key of one column") {
		t.Errorf("a table without one key: %v", err)
	}
}

// A route may give its pages their data. Then only the route knows what a
// page shows, and all its pages are computed again when its list is.
func TestDataFromGetStaticPaths(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/items/[id].shugo --
---
export const getStaticPaths = ({ query }) => query("products").select("title").all().map((item) => ({ params: { id: item.id }, props: { item } }))
const { item } = Shugo.props
console.log("item " + item.id)
---
<h1>{item.title}</h1>
`)
	s.sql(queryRows)
	s.build()
	s.computed()
	s.sql(`UPDATE products SET title = 'Carafe' WHERE id = 'p1'`)
	report := s.update()
	if got := s.computed(); len(got) != 4 || report.Data.Routes != 1 || report.Pages.Changed != 1 {
		t.Errorf("computed %q, routes listed %d, pages changed %d", got, report.Data.Routes, report.Pages.Changed)
	}
	expect(t, s.page("/items/p1"), `<h1>Carafe</h1>`)
	// A column the route did not take.
	s.changed(`UPDATE products SET price = 99`)
	s.changed(`DELETE FROM products WHERE id = 'p2'`, "item p1", "item p3", "item p4")
	if s.has("/items/p2") {
		t.Errorf("pages: %v", s.addresses())
	}
	s.sameAsBuilt()
}

// A shared block that cannot be computed any more is an error as long as a
// page shows it, and the site stays as it was.
func TestSharedBlockThatFails(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- components/Card.shugo --
---
const item = Shugo.query("products").select("title").get(Shugo.props.id)
if (!item) throw new Error("there is no product " + Shugo.props.id)
---
<b>{item.title}</b>
-- pages/index.shugo --
---
import Card from "../components/Card.shugo"
const rows = Shugo.query("products").select("id").where({ category: "c2" }).all()
---
<Card id="p1" /><g-for each={rows} as="row"><Card id={row.id} /></g-for>
`)
	s.sql(queryRows)
	s.build()
	expect(t, s.page("/"), `<b>Jug</b><b>Vase</b>`)

	// The page stops showing the card of a product that is gone: no error.
	s.sql(`DELETE FROM products WHERE id = 'p4'`)
	if report := s.update(); report.Shared.Removed != 1 {
		t.Errorf("shared blocks: %+v", report.Shared)
	}
	expect(t, s.page("/"), `<b>Jug</b>`)

	// The page still shows the other card, whatever the data says.
	s.sql(`DELETE FROM products WHERE id = 'p1'`)
	if _, err := s.tryUpdate(); err == nil || !strings.Contains(err.Error(), "there is no product p1") || !strings.Contains(err.Error(), "components/Card.shugo") {
		t.Errorf("the update: %v", err)
	}
	expect(t, s.page("/"), `<b>Jug</b>`)
	// The change is not forgotten: the next update meets it again.
	if _, err := s.tryUpdate(); err == nil {
		t.Error("the second update went through")
	}
	s.sql(`INSERT INTO products VALUES ('p1', 'jug', 'New jug', 10, 'c1')`)
	s.update()
	expect(t, s.page("/"), `<b>New jug</b>`)
}

// When the journal cannot tell what has changed, everything is computed.
func TestDataJournalStartsAnew(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/index.shugo --
---
const rows = Shugo.query("products").select("title").order("title").all()
---
<p>{rows.map((row) => row.title).join(", ")}</p>
`)
	s.sql(queryRows)
	s.build()
	if report := s.update(); report.Data.Whole || report.Computed.Pages != 0 {
		t.Errorf("an update without changes: %+v", report.Data)
	}

	// A table is added: rows may have changed before it got its triggers.
	s.sql(`CREATE TABLE notes (id INTEGER PRIMARY KEY, text TEXT); UPDATE products SET title = 'Carafe' WHERE id = 'p1'`)
	if report := s.update(); !report.Data.Whole || report.Computed.Pages != 1 {
		t.Errorf("after a table was added: %+v, computed %+v", report.Data, report.Computed)
	}
	expect(t, s.page("/"), `<p>Carafe, Cup, Tray, Vase</p>`)
	s.sql(`UPDATE products SET title = 'Beaker' WHERE id = 'p2'`)
	if report := s.update(); report.Data.Whole || report.Data.Rows != 1 || report.Computed.Pages != 1 {
		t.Errorf("the update after: %+v, computed %+v", report.Data, report.Computed)
	}

	// Another database stands in place of the one the site was computed from.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		os.Remove(filepath.Join(s.root, "data", "shop.db"+suffix))
	}
	s.sql(queryRows)
	if report := s.update(); !report.Data.Whole {
		t.Errorf("after the database was replaced: %+v", report.Data)
	}
	expect(t, s.page("/"), `<p>Cup, Jug, Tray, Vase</p>`)
}

// What a page reads from elsewhere than the data database is not tracked:
// an update does not see it change, a build reads it again.
func TestDataOutsideTheDatabase(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- note.txt --
old
-- pages/index.shugo --
---
import { readFileSync } from "node:fs"
const note = readFileSync("note.txt", "utf8")
const count = Shugo.query("products").count()
---
<p>{note}: {count}</p>
`)
	s.sql(queryRows)
	s.build()
	s.write("-- note.txt --\nnew\n")
	if report := s.update(); report.Computed.Pages != 0 {
		t.Errorf("computed %+v", report.Computed)
	}
	expect(t, s.page("/"), `<p>old: 4</p>`)
	s.build()
	expect(t, s.page("/"), `<p>new: 4</p>`)
}

// A page that stands at an address of its own hides the page a route with
// a parameter would give there, also when that page comes later.
func TestDataPageBehindAnother(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- pages/products/special.shugo --
<h1>Special</h1>
-- pages/products/[slug].shugo --
---
export const getStaticPaths = ({ query }) => query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
const item = Shugo.query("products").select("title").where({ slug: Shugo.params.slug }).first()
console.log("page " + item.id)
---
<h1>{item.title}</h1>
`)
	s.sql(queryRows)
	s.build()
	s.computed()
	s.sql(`INSERT INTO products VALUES ('p9', 'special', 'Not special', 1, 'c1')`)
	report := s.update()
	if got := s.computed(); len(got) != 0 || report.Shadowed != 1 || report.Pages.Added != 0 {
		t.Errorf("computed %q, shadowed %d, pages %+v", got, report.Shadowed, report.Pages)
	}
	expect(t, s.page("/products/special"), `<h1>Special</h1>`)
	s.changed(`UPDATE products SET slug = 'plain' WHERE id = 'p9'`, "page p9")
	expect(t, s.page("/products/plain"), `<h1>Not special</h1>`)
	s.sameAsBuilt()
}

// A page that one route stops giving may at once be given by a less exact
// route. It is dropped and made anew within one update, and what it shows
// and what it read stays on the books.
func TestDataPageChangesRoutes(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data": "data/shop.db"}
-- components/Card.shugo --
---
const item = Shugo.query("products").select("title").get(Shugo.props.id)
console.log("card " + Shugo.props.id)
---
<b>{item.title}</b>
-- pages/x/[slug].shugo --
---
import Card from "../../components/Card.shugo"
export const getStaticPaths = ({ query }) => query("products").select("slug").where({ category: "c1" }).all().map(({ slug }) => ({ params: { slug } }))
const item = Shugo.query("products").select("price").where({ slug: Shugo.params.slug }).first()
console.log("exact " + item.id)
---
<p>exact {item.price} <Card id={item.id} /></p>
-- pages/[...all].shugo --
---
import Card from "../components/Card.shugo"
export const getStaticPaths = ({ query }) => query("products").select("slug").all().map(({ slug }) => ({ params: { all: "x/" + slug } }))
const item = Shugo.query("products").select("price").where({ slug: Shugo.params.all.slice(2) }).first()
console.log("loose " + item.id)
---
<p>loose {item.price} <Card id={item.id} /></p>
`)
	s.sql(queryRows)
	if report := s.build(); report.Shadowed != 3 || report.Pages.Total != 4 {
		t.Fatalf("shadowed %d, pages %d", report.Shadowed, report.Pages.Total)
	}
	s.computed()
	expect(t, s.page("/x/jug"), `<p>exact 10 <b>Jug</b></p>`)
	expect(t, s.page("/x/vase"), `<p>loose 9 <b>Vase</b></p>`)

	// The exact route lets the page go; the loose one has it from now on.
	s.changed(`UPDATE products SET category = 'c2' WHERE id = 'p1'`, "loose p1")
	expect(t, s.page("/x/jug"), `<p>loose 10 <b>Jug</b></p>`)
	// The page still shows its card, and still reads its row.
	s.changed(`UPDATE products SET title = 'Carafe' WHERE id = 'p1'`, "card p1")
	expect(t, s.page("/x/jug"), `<p>loose 10 <b>Carafe</b></p>`)
	s.changed(`UPDATE products SET price = 11 WHERE id = 'p1'`, "loose p1")
	// And back: the exact route takes the page over.
	s.changed(`UPDATE products SET category = 'c1' WHERE id = 'p1'`, "exact p1")
	expect(t, s.page("/x/jug"), `<p>exact 11 <b>Carafe</b></p>`)
	s.changed(`UPDATE products SET price = 12 WHERE id = 'p1'`, "exact p1")
	s.sameAsBuilt()
}
