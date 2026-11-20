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
		return v.Pages(func(path string, _ []byte) { out = append(out, path) })
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
