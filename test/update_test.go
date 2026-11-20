package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/evaluate"
)

// A small shop. The header and the cards do not depend on the page they
// stand on, so each of them is computed and kept once for the whole site.
const shop = `
-- data.json --
{"motto": "Clay and wood", "products": [
  {"slug": "jug", "title": "Jug", "price": 10, "note": "internal", "related": ["cup", "tray"]},
  {"slug": "cup", "title": "Cup", "price": 5, "note": "", "related": ["jug"]},
  {"slug": "tray", "title": "Tray", "price": 7, "note": "", "related": []}
]}
-- lib/data.ts --
import { readFileSync } from "node:fs"
export interface Product { slug: string; title: string; price: number; related: string[] }
const data = JSON.parse(readFileSync("data.json", "utf8"))
export const motto: string = data.motto
export const products: Product[] = data.products
export const product = (slug: string): Product => products.find((p) => p.slug === slug)!
-- components/Header.shugo --
---
import { motto, products } from "../lib/data"
console.log("header computed")
---
<header>{motto} · {products.length} products</header>
-- components/Card.shugo --
---
import { product } from "../lib/data"
const { slug } = Shugo.props as { slug: string }
const { title, price } = product(slug)
---
<a href={"/products/" + slug}>{title} — {price}</a>
-- pages/+layout.shugo --
---
import Header from "../components/Header.shugo"
---
<html><body><Header /><main><g-slot /></main></body></html>
-- pages/index.shugo --
---
import Card from "../components/Card.shugo"
import { products } from "../lib/data"
---
<ul><g-for each={products} as="p"><li><Card slug={p.slug} /></li></g-for></ul>
-- pages/about.shugo --
<h1>About</h1>
-- pages/products/[slug].shugo --
---
import Card from "../../components/Card.shugo"
import { products, type Product } from "../../lib/data"
export const getStaticPaths = () => products.map((p) => ({ params: { slug: p.slug }, props: { p } }))
const { p } = Shugo.props as { p: Product }
---
<h1>{p.title}</h1><p>{p.price}</p><g-for each={p.related} as="slug"><Card slug={slug} /></g-for>
`

// change rewrites data.json.
func (s *site) change(edit func(data map[string]any, products []map[string]any) []map[string]any) {
	s.t.Helper()
	path := filepath.Join(s.root, "data.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		s.t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		s.t.Fatal(err)
	}
	var products []map[string]any
	for _, p := range data["products"].([]any) {
		products = append(products, p.(map[string]any))
	}
	data["products"] = edit(data, products)
	raw, _ = json.Marshal(data)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		s.t.Fatal(err)
	}
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
	header   = "components/Header.shugo{}"
	cardJug  = `components/Card.shugo{"slug":"jug",}`
	cardCup  = `components/Card.shugo{"slug":"cup",}`
	cardTray = `components/Card.shugo{"slug":"tray",}`
	cardPot  = `components/Card.shugo{"slug":"pot",}`
	layout   = "pages/+layout.shugo{|default}"
)

func TestUpdates(t *testing.T) {
	s := newSite(t, shop)
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
	expect(t, s.page("/products/jug"), `<html><body><header>Clay and wood · 3 products</header><main><h1>Jug</h1><p>10</p><a href="/products/cup">Cup — 5</a><a href="/products/tray">Tray — 7</a></main></body></html>`)

	t.Run("nothing changed", func(t *testing.T) {
		r := s.update()
		expectChanges(t, r, evaluate.Changes{}, evaluate.Changes{})
		if r.Values.Added != 0 || r.Values.Removed != 0 {
			t.Errorf("values: %+v", r.Values)
		}
	})

	t.Run("a field no page shows", func(t *testing.T) {
		s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
			products[0]["note"] = "changed"
			return products
		})
		expectChanges(t, s.update(), evaluate.Changes{}, evaluate.Changes{})
	})

	// The header stands on every page, and no page is touched when it changes.
	t.Run("the text in the header", func(t *testing.T) {
		s.change(func(data map[string]any, products []map[string]any) []map[string]any {
			data["motto"] = "Handmade"
			return products
		})
		expectChanges(t, s.update(), evaluate.Changes{},
			evaluate.Changes{Changed: 1, ChangedNames: []string{header}})
		if !strings.Contains(s.page("/about"), "<header>Handmade · 3 products</header>") {
			t.Errorf("the page shows the old header: %s", s.page("/about"))
		}
	})

	// A card is shown on the main page and on the pages of other products;
	// only the card itself and the page of the product change.
	t.Run("the price of one product", func(t *testing.T) {
		s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
			products[1]["price"] = 6
			return products
		})
		expectChanges(t, s.update(),
			evaluate.Changes{Changed: 1, ChangedNames: []string{"/products/cup"}},
			evaluate.Changes{Changed: 1, ChangedNames: []string{cardCup}})
		if !strings.Contains(s.page("/products/jug"), "Cup — 6") || !strings.Contains(s.page("/"), "Cup — 6") {
			t.Error("a page shows the old card")
		}
	})

	t.Run("two products change places", func(t *testing.T) {
		s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
			products[0], products[1] = products[1], products[0]
			return products
		})
		expectChanges(t, s.update(), evaluate.Changes{Changed: 1, ChangedNames: []string{"/"}}, evaluate.Changes{})
	})

	t.Run("a product is removed", func(t *testing.T) {
		s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
			products[1]["related"] = []string{"cup"} // the jug no longer points at the tray
			return products[:2]
		})
		expectChanges(t, s.update(),
			evaluate.Changes{Changed: 2, Removed: 1, ChangedNames: []string{"/", "/products/jug"}, RemovedNames: []string{"/products/tray"}},
			evaluate.Changes{Changed: 1, Removed: 1, ChangedNames: []string{header}, RemovedNames: []string{cardTray}})
		if s.has("/products/tray") {
			t.Error("the page of the removed product is still there")
		}
	})

	t.Run("a product is added", func(t *testing.T) {
		s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
			return append(products, map[string]any{"slug": "pot", "title": "Pot", "price": 12, "related": []string{}})
		})
		expectChanges(t, s.update(),
			evaluate.Changes{Added: 1, Changed: 1, AddedNames: []string{"/products/pot"}, ChangedNames: []string{"/"}},
			evaluate.Changes{Added: 1, Changed: 1, AddedNames: []string{cardPot}, ChangedNames: []string{header}})
		expect(t, s.page("/products/pot"), `<html><body><header>Handmade · 3 products</header><main><h1>Pot</h1><p>12</p></main></body></html>`)
	})

	// After a template changes every record is written anew. The markup of
	// a template is no value, so no values are added for it, and those that
	// came out as before stay where they are.
	t.Run("a template changes", func(t *testing.T) {
		s.write("-- pages/about.shugo --\n<h1>About us</h1>")
		r := s.build()
		if !r.Rebuilt || r.Pages.Added != r.Pages.Total || r.Values.Added != 0 {
			t.Errorf("rebuilt=%v pages=%+v values=%+v", r.Rebuilt, r.Pages, r.Values)
		}
		if !strings.Contains(s.page("/about"), "<h1>About us</h1>") {
			t.Error("the page shows the old template")
		}
	})
}

// A failed update leaves the store as it was.
func TestFailedUpdateChangesNothing(t *testing.T) {
	s := newSite(t, shop)
	s.build()
	before := s.page("/products/jug")
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		products[0]["price"] = 11
		products[0]["related"] = []string{"missing"} // the card of a product that is not there fails
		return products
	})
	if _, err := s.tryBuild(); err == nil {
		t.Fatal("the update did not fail")
	}
	expect(t, s.page("/products/jug"), before)
}
