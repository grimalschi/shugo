package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/evaluate"
)

// A small shop. The header and the cards do not depend on the page they
// stand on, so each of them is computed and kept once for the whole site.
const shop = `
-- shugo.json --
{"data": "data/shop.db"}
-- src/components/Header.shugo --
<script>
import { query } from "shugo"
const motto = query("settings").get(1).motto
const count = query("products").count()
console.log("header computed")
</script>
<header>{motto} · {count} products</header>
-- src/components/Card.shugo --
<script>
import { query } from "shugo"
const { slug } = $props() as { slug: string }
const { title, price } = query("products").where({ slug }).first()
</script>
<a href={"/products/" + slug}>{title} — {price}</a>
-- src/pages/+layout.shugo --
<script>
let { children } = $props()
import Header from "../components/Header.shugo"
</script>
<Header /><main>{@render children()}</main>
-- src/pages/index.shugo --
<script>
import { query } from "shugo"
import Card from "../components/Card.shugo"
const products = query("products").select("slug").order("position").all()
</script>
<ul>{#each products as p}<li><Card slug={p.slug} /></li>{/each}</ul>
-- src/pages/about.shugo --
<h1>About</h1>
-- src/pages/products/[slug].shugo --
<script module>
import { query, page } from "shugo"
export const getStaticPaths = () => query("products").select("slug").all().map((p) => ({ params: { slug: p.slug } }))
</script>
<script>
import Card from "../../components/Card.shugo"
const p = query("products").where({ slug: page.params.slug }).first()
const related = query("related").where({ product: p.slug }).order("position").all()
</script>
<h1>{p.title}</h1><p>{p.price}</p>{#each related as r}<Card slug={r.other} />{/each}
`

const shopRows = `
CREATE TABLE settings (id INTEGER PRIMARY KEY, motto TEXT NOT NULL);
CREATE TABLE products (slug TEXT PRIMARY KEY, title TEXT NOT NULL, price INTEGER NOT NULL, note TEXT NOT NULL, position INTEGER NOT NULL);
CREATE TABLE related (id INTEGER PRIMARY KEY, product TEXT NOT NULL, other TEXT NOT NULL, position INTEGER NOT NULL);
INSERT INTO settings VALUES (1, 'Clay and wood');
INSERT INTO products VALUES ('jug', 'Jug', 10, 'internal', 1), ('cup', 'Cup', 5, '', 2), ('tray', 'Tray', 7, '', 3);
INSERT INTO related (product, other, position) VALUES ('jug', 'cup', 1), ('jug', 'tray', 2), ('cup', 'jug', 1);
`

// newShop makes the shop with its data.
func newShop(t testing.TB) *site {
	s := newSite(t, shop)
	s.sql(shopRows)
	return s
}

// expectChanges compares what an update reports with what should have changed.
func expectChanges(t *testing.T, r *evaluate.Report, pages, shared evaluate.Changes) {
	t.Helper()
	strip := func(c evaluate.Changes) evaluate.Changes {
		c.Total = 0
		return c
	}
	if got := strip(r.Pages); !reflect.DeepEqual(got, pages) {
		t.Errorf("pages:\n got %+v\nwant %+v", got, pages)
	}
	if got := strip(r.Shared); !reflect.DeepEqual(got, shared) {
		t.Errorf("shared blocks:\n got %+v\nwant %+v", got, shared)
	}
}

const (
	header   = "src/components/Header.shugo{}"
	cardJug  = `src/components/Card.shugo{"slug":"jug"}`
	cardCup  = `src/components/Card.shugo{"slug":"cup"}`
	cardTray = `src/components/Card.shugo{"slug":"tray"}`
	cardPot  = `src/components/Card.shugo{"slug":"pot"}`
)

func TestUpdates(t *testing.T) {
	s := newShop(t)
	first := s.build()
	if first.Pages.Total != 5 || first.Pages.Added != 5 {
		t.Fatalf("first build: %+v", first.Pages)
	}
	// The header, the layout around it and one card for every product.
	if first.Shared.Total != 5 {
		t.Errorf("shared blocks: %+v", first.Shared)
	}
	if n := strings.Count(s.printed.String(), "header computed"); n != 1 {
		t.Errorf("the script of the header ran %d times for 5 pages", n)
	}
	expect(t, s.page("/products/jug"), `<header>Clay and wood · 3 products</header><main><h1>Jug</h1><p>10</p><a href="/products/cup">Cup — 5</a><a href="/products/tray">Tray — 7</a></main>`)

	t.Run("nothing changed", func(t *testing.T) {
		r := s.update()
		expectChanges(t, r, evaluate.Changes{}, evaluate.Changes{})
		if r.Computed.Pages+r.Computed.Shared != 0 || r.Render != 0 {
			t.Errorf("computed %+v, pages with new HTML: %d", r.Computed, r.Render)
		}
	})

	t.Run("a field no page shows", func(t *testing.T) {
		s.sql(`UPDATE products SET note = 'changed' WHERE slug = 'jug'`)
		r := s.update()
		expectChanges(t, r, evaluate.Changes{}, evaluate.Changes{})
		if r.Computed.Pages+r.Computed.Shared != 0 {
			t.Errorf("computed %+v", r.Computed)
		}
	})

	// The header stands on every page, and no page is touched when it changes.
	t.Run("the text in the header", func(t *testing.T) {
		s.sql(`UPDATE settings SET motto = 'Handmade'`)
		r := s.update()
		expectChanges(t, r, evaluate.Changes{},
			evaluate.Changes{Changed: 1, ChangedNames: []string{header}})
		// No record of a page has changed, and the HTML of every page has:
		// the layout of each shows the header.
		if r.Render != 5 {
			t.Errorf("pages with new HTML: %d %v", r.Render, r.RenderNames)
		}
		if !strings.Contains(s.page("/about"), "<header>Handmade · 3 products</header>") {
			t.Errorf("the page shows the old header: %s", s.page("/about"))
		}
	})

	// A card is shown on the main page and on the pages of other products;
	// only the card itself and the page of the product change.
	t.Run("the price of one product", func(t *testing.T) {
		s.sql(`UPDATE products SET price = 6 WHERE slug = 'cup'`)
		r := s.update()
		expectChanges(t, r,
			evaluate.Changes{Changed: 1, ChangedNames: []string{"/products/cup"}},
			evaluate.Changes{Changed: 1, ChangedNames: []string{cardCup}})
		// The HTML differs on the page of the cup and on the two pages that
		// show its card: the main page and the page of the jug.
		if want := []string{"/", "/products/cup", "/products/jug"}; !reflect.DeepEqual(r.RenderNames, want) {
			t.Errorf("pages with new HTML: %v, want %v", r.RenderNames, want)
		}
		if !strings.Contains(s.page("/products/jug"), "Cup — 6") || !strings.Contains(s.page("/"), "Cup — 6") {
			t.Error("a page shows the old card")
		}
	})

	t.Run("two products change places", func(t *testing.T) {
		s.sql(`UPDATE products SET position = 3 - position WHERE slug IN ('jug', 'cup')`)
		expectChanges(t, s.update(), evaluate.Changes{Changed: 1, ChangedNames: []string{"/"}}, evaluate.Changes{})
	})

	t.Run("a product is removed", func(t *testing.T) {
		s.sql(`DELETE FROM products WHERE slug = 'tray'; DELETE FROM related WHERE other = 'tray'`)
		expectChanges(t, s.update(),
			evaluate.Changes{Changed: 2, Removed: 1, ChangedNames: []string{"/", "/products/jug"}, RemovedNames: []string{"/products/tray"}},
			evaluate.Changes{Changed: 1, Removed: 1, ChangedNames: []string{header}, RemovedNames: []string{cardTray}})
		if s.has("/products/tray") {
			t.Error("the page of the removed product is still there")
		}
	})

	t.Run("a product is added", func(t *testing.T) {
		s.sql(`INSERT INTO products VALUES ('pot', 'Pot', 12, '', 4)`)
		expectChanges(t, s.update(),
			evaluate.Changes{Added: 1, Changed: 1, AddedNames: []string{"/products/pot"}, ChangedNames: []string{"/"}},
			evaluate.Changes{Added: 1, Changed: 1, AddedNames: []string{cardPot}, ChangedNames: []string{header}})
		expect(t, s.page("/products/pot"), `<header>Handmade · 3 products</header><main><h1>Pot</h1><p>12</p></main>`)
	})

	// After a template changes every record is written anew.
	t.Run("a template changes", func(t *testing.T) {
		s.write("-- src/pages/about.shugo --\n<h1>About us</h1>")
		r := s.build()
		if !r.Rebuilt || r.Pages.Added != r.Pages.Total {
			t.Errorf("rebuilt=%v pages=%+v", r.Rebuilt, r.Pages)
		}
		if !strings.Contains(s.page("/about"), "<h1>About us</h1>") {
			t.Error("the page shows the old template")
		}
	})
}

// A failed update leaves the store as it was.
func TestFailedUpdateChangesNothing(t *testing.T) {
	s := newShop(t)
	s.build()
	before := s.page("/products/jug")
	// The card of a product that is not there fails.
	s.sql(`UPDATE products SET price = 11 WHERE slug = 'jug'; UPDATE related SET other = 'missing' WHERE product = 'jug' AND other = 'cup'`)
	if _, err := s.tryBuild(); err == nil {
		t.Fatal("the update did not fail")
	}
	expect(t, s.page("/products/jug"), before)
}

// A project without a data database has no data that could change: an
// update computes nothing. Its pages change with the project, and shugo
// update compiles and builds a project that has changed.
func TestUpdateWithoutData(t *testing.T) {
	s := newSite(t, `
-- src/lib/names.ts --
export const names = ["Ann", "Bob"]
-- src/pages/index.shugo --
<script>
import { names } from "../lib/names"
</script>
<ul>{#each names as name}<li>{name}</li>{/each}</ul>
`)
	s.build()
	if r := s.update(); r.Computed.Pages+r.Computed.Shared != 0 || r.Data != nil {
		t.Errorf("an update without data computed %+v", r.Computed)
	}
	expect(t, s.page("/"), "<ul><li>Ann</li><li>Bob</li></ul>")
}
