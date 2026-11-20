package test

import (
	"strings"
	"testing"
)

const layouts = `
-- pages/+layout.shugo --
<html><head><g-slot name="head"><title>Site</title></g-slot></head><body><g-slot /></body></html>
-- pages/index.shugo --
<title slot="head">Home</title>
<h1>home</h1>
-- pages/about.shugo --
<h1>about</h1>
-- pages/docs/+layout.shugo --
<nav>docs</nav><main><g-slot /></main>
-- pages/docs/index.shugo --
<p>docs home</p>
-- pages/docs/(guides)/+layout.shugo --
<aside>guides</aside><g-slot />
-- pages/docs/(guides)/start.shugo --
<meta slot="head" name="x" content="y">
<p>start</p>
-- pages/_drafts/secret.shugo --
<p>never a page</p>
-- pages/blog/_card.shugo --
<p>a component among pages</p>
`

func TestLayouts(t *testing.T) {
	s := newSite(t, layouts)
	report := s.build()
	expect(t, s.page("/"), `<html><head><title>Home</title></head><body>
<h1>home</h1></body></html>`)
	expect(t, s.page("/about"), `<html><head><title>Site</title></head><body><h1>about</h1></body></html>`)
	expect(t, s.page("/docs"), `<html><head><title>Site</title></head><body><nav>docs</nav><main><p>docs home</p></main></body></html>`)
	// A folder in parentheses adds a layout and no part of the address; what
	// the page gives to "head" goes through both inner layouts to the outer one.
	expect(t, s.page("/docs/start"), `<html><head><meta name="x" content="y"></head><body><nav>docs</nav><main><aside>guides</aside>
<p>start</p></main></body></html>`)
	if s.has("/_drafts/secret") || s.has("/blog/_card") || s.has("/docs/(guides)/start") {
		t.Error("a file that is not a page got an address")
	}
	if report.Pages.Total != 4 {
		t.Errorf("pages: %d", report.Pages.Total)
	}
}

const catalogue = `
-- lib/data.ts --
import { readFileSync } from "node:fs"
export interface Product { slug: string; title: string; price: number; category: string }
export const products: Product[] = JSON.parse(readFileSync("data.json", "utf8"))
-- data.json --
[{"slug":"jug","title":"Jug","price":10,"category":"clay"},
 {"slug":"cup","title":"Cup","price":5,"category":"clay"},
 {"slug":"tray","title":"Tray","price":7,"category":"wood"}]
-- pages/index.shugo --
<h1>shop</h1>
-- pages/products/[slug].shugo --
---
import { products, type Product } from "../../lib/data"
export function getStaticPaths() {
  return products.map((product) => ({ params: { slug: product.slug }, props: { product } }))
}
const { product } = Shugo.props as { product: Product }
---
<h1>{product.title}</h1><p>{Shugo.params.slug} {Shugo.url.pathname} {Shugo.url.href}</p>
-- pages/products/new.shugo --
<h1>the newest</h1>
-- pages/files/[...path].shugo --
---
export const getStaticPaths = () => [{ params: { path: undefined } }, { params: { path: "a" } }, { params: { path: "a/b c" } }]
---
<p>[{Shugo.params.path}]</p>
-- pages/list/[...page].shugo --
---
import { products } from "../../lib/data"
export async function getStaticPaths({ paginate }) {
  return paginate(products, { pageSize: 2 })
}
const { page } = Shugo.props
---
<p>{page.currentPage}/{page.lastPage}: {page.data.map((p) => p.slug).join(",")} prev={page.url.prev} next={page.url.next}</p>
`

func TestDynamicPages(t *testing.T) {
	s := newSite(t, catalogue+`
-- shugo.json --
{"site": "https://shop.example"}`)
	report := s.build()
	expect(t, s.page("/products/jug"), `<h1>Jug</h1><p>jug /products/jug https://shop.example/products/jug</p>`)
	expect(t, s.page("/products/tray"), `<h1>Tray</h1><p>tray /products/tray https://shop.example/products/tray</p>`)
	expect(t, s.page("/products/new"), `<h1>the newest</h1>`)
	expect(t, s.page("/files"), `<p>[]</p>`)
	expect(t, s.page("/files/a/b c"), `<p>[a/b c]</p>`)
	expect(t, s.page("/list"), `<p>1/2: jug,cup prev= next=/list/2</p>`)
	expect(t, s.page("/list/2"), `<p>2/2: tray prev=/list next=</p>`)
	if s.has("/products/none") {
		t.Error("a page that getStaticPaths did not return")
	}
	if want := 1 + 3 + 1 + 3 + 2; report.Pages.Total != want {
		t.Errorf("pages: %d, want %d", report.Pages.Total, want)
	}
}

// A page with a name of its own takes its address from a page with a
// parameter that would give the same one.
func TestStaticPageWins(t *testing.T) {
	s := newSite(t, strings.Replace(catalogue, `{"slug":"cup"`, `{"slug":"new"`, 1))
	report := s.build()
	expect(t, s.page("/products/new"), `<h1>the newest</h1>`)
	if report.Shadowed != 1 {
		t.Errorf("shadowed: %d", report.Shadowed)
	}
}

func TestProjectMistakes(t *testing.T) {
	cases := []struct{ name, files, want string }{
		{"a parameter without getStaticPaths", "-- pages/[id].shugo --\n<p>x</p>", "pages/[id].shugo:1:1: the name of this page has a parameter"},
		{"two pages of one address", "-- pages/a.shugo --\n<p>1</p>\n-- pages/a/index.shugo --\n<p>2</p>", "give the same addresses"},
		{"an unknown + file", "-- pages/+page.shugo --\n<p>x</p>", "names that start with + are kept for the framework"},
		{"a slot in a page", "-- pages/index.shugo --\n<g-slot />", "a page is not called by anything"},
		{"a missing parameter", "-- pages/[id].shugo --\n---\nexport const getStaticPaths = () => [{ params: {} }]\n---\n<p>x</p>", `returned params without "id"`},
		{"an address twice", "-- pages/[id].shugo --\n---\nexport const getStaticPaths = () => [{ params: { id: \"a\" } }, { params: { id: \"a\" } }]\n---\n<p>x</p>", "returned the address /a twice"},
		{"not a list", "-- pages/[id].shugo --\n---\nexport const getStaticPaths = () => ({ id: 1 })\n---\n<p>x</p>", "must return a list of { params, props }"},
		{"a slot name inside a condition", "-- pages/index.shugo --\n---\nimport C from \"../c.shugo\"\n---\n<C><g-if cond={1}><b slot=\"x\">t</b></g-if></C>\n-- c.shugo --\n<g-slot name=\"x\" />", `slot="x" works only on a tag that stands right inside the component`},
		{"a loop over a number", "-- pages/index.shugo --\n<g-for each={5} as=\"x\">{x}</g-for>", "<g-for each={…}> expects a list, got number 5"},
		{"a tag that is not a component", "-- pages/index.shugo --\n---\nconst Card = \"div\"\n---\n<Card />", "this tag is not a component"},
		{"a component that shows itself forever", "-- pages/index.shugo --\n---\nimport Loop from \"../loop.shugo\"\n---\n<Loop />\n-- loop.shugo --\n<Shugo.self />", "shows itself again with the same parameters"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, c.files).tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("\n got: %v\nwant: %s", err, c.want)
			}
		})
	}
}

// A mistake found by the bundler or at run time is reported at its place in
// the .shugo file, not in the code generated from it.
func TestMistakesPointAtTheTemplate(t *testing.T) {
	cases := []struct{ name, page, want string }{
		{"a module that is not there", "---\nimport { helper } from \"../lib/missing\"\n---\n<h1>{helper()}</h1>", `pages/index.shugo:2:24: Could not resolve "../lib/missing"`},
		{"an expression cut short", "<p>ok</p>\n<h1 class={a +}>x</h1>", `pages/index.shugo:2:15: Unexpected ")"`},
		{"a statement in braces", "<p>\n  {const a = 1}\n</p>", `pages/index.shugo:2:4: Unexpected "const"`},
		{"a name that is not a name", `<g-for each={[1]} as="1x">{x}</g-for>`, `pages/index.shugo:1:24: Syntax error "x"`},
		{"a mistake in the script", "---\nconst a = 1\nconst b = (a +\n---\n<p>{a}</p>", "pages/index.shugo:3:15: the script ends in the middle of a statement"},
		{"markup inside braces", "<ul>\n  {[1].map(n => <li>{n}</li>)}\n</ul>", "pages/index.shugo:2:3: markup cannot stand inside {…}"},
		{"a failure while computing", "---\nconst product = { title: \"x\" }\n---\n<p>\n  {product.missing.field}\n</p>", "at pages/index.shugo:5:"},
		{"a failure in a component", "---\nimport Card from \"../card.shugo\"\n---\n<Card />\n-- card.shugo --\n<p>ok</p>\n<b>{Shugo.props.item.title}</b>", "at card.shugo:2:"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, "-- pages/index.shugo --\n"+c.page).tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("\n got: %v\nwant: %s", err, c.want)
			}
		})
	}
}
