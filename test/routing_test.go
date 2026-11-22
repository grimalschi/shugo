package test

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/skeleton"
)

const layouts = `
-- src/site.html --
<html><head><meta charset="utf-8">%shugo.head%</head><body>%shugo.body%</body></html>
-- src/pages/+layout.svelte --
<script>
let { children } = $props()
</script>
<svelte:head><title>Shop</title><meta name="generator" content="shugo"></svelte:head>
{@render children()}
-- src/pages/index.svelte --
<svelte:head><title>Home</title></svelte:head>
<h1>home</h1>
-- src/pages/about.svelte --
<h1>about</h1>
-- src/pages/docs/+layout.svelte --
<script>
let { children } = $props()
</script>
<nav>docs</nav><main>{@render children()}</main>
-- src/pages/docs/index.svelte --
<p>docs home</p>
-- src/pages/docs/(guides)/+layout.svelte --
<script>
let { children } = $props()
</script>
<svelte:head><link rel="next" href="/docs"></svelte:head>
<aside>guides</aside>{@render children()}
-- src/pages/docs/(guides)/start.svelte --
<svelte:head><meta name="x" content="y"></svelte:head>
<p>start</p>
-- src/pages/_drafts/secret.svelte --
<p>never a page</p>
-- src/pages/blog/_card.svelte --
<p>a component among pages</p>
`

func TestLayouts(t *testing.T) {
	s := newSite(t, layouts)
	report := s.build()
	// What <svelte:head> gives goes to the place %shugo.head% of the shell,
	// in the order of the document, the outer layouts first; of the titles
	// the page keeps the last, its own, and it stands after the rest: so
	// Svelte does.
	expect(t, s.page("/"), `<html><head><meta charset="utf-8"><meta name="generator" content="shugo"/><title>Home</title></head><body><h1>home</h1></body></html>`)
	expect(t, s.page("/about"), `<html><head><meta charset="utf-8"><meta name="generator" content="shugo"/><title>Shop</title></head><body><h1>about</h1></body></html>`)
	expect(t, s.page("/docs"), `<html><head><meta charset="utf-8"><meta name="generator" content="shugo"/><title>Shop</title></head><body><nav>docs</nav><main><p>docs home</p></main></body></html>`)
	// A folder in parentheses adds a layout and no part of the address.
	expect(t, s.page("/docs/start"), `<html><head><meta charset="utf-8"><meta name="generator" content="shugo"/><link rel="next" href="/docs"/><meta name="x" content="y"/><title>Shop</title></head><body><nav>docs</nav><main><aside>guides</aside><p>start</p></main></body></html>`)
	if s.has("/_drafts/secret") || s.has("/blog/_card") || s.has("/docs/(guides)/start") {
		t.Error("a file that is not a page got an address")
	}
	if report.Pages.Total != 4 {
		t.Errorf("pages: %d", report.Pages.Total)
	}
}

const catalogue = `
-- src/lib/data.ts --
import { readFileSync } from "node:fs"
export interface Product { slug: string; title: string; price: number; category: string }
export const products: Product[] = JSON.parse(readFileSync("data.json", "utf8"))
-- data.json --
[{"slug":"jug","title":"Jug","price":10,"category":"clay"},
 {"slug":"cup","title":"Cup","price":5,"category":"clay"},
 {"slug":"tray","title":"Tray","price":7,"category":"wood"}]
-- src/pages/index.svelte --
<h1>shop</h1>
-- src/pages/products/[slug].svelte --
<script module>
import { page } from "shugo"
import { products } from "../../lib/data"
export function getStaticPaths() {
  return products.map((product) => ({ params: { slug: product.slug } }))
}
</script>
<script lang="ts">
const product = products.find((product) => product.slug === page.params.slug)
</script>
<h1>{product.title}</h1><p>{page.params.slug} {page.url.pathname} {page.url.href}</p>
-- src/pages/products/new.svelte --
<h1>the newest</h1>
-- src/pages/files/[...path].svelte --
<script module>
import { page } from "shugo"
export const getStaticPaths = () => [{ params: { path: undefined } }, { params: { path: "a" } }, { params: { path: "a/b c" } }]
</script>
<p>[{page.params.path}]</p>
-- src/pages/list/[...page].svelte --
<script module>
import { paginate } from "shugo"
import { products } from "../../lib/data"
export function getStaticPaths() {
  return paginate(products.length, { pageSize: 2 })
}
</script>
<script>
const list = paginate(products, { pageSize: 2 })
</script>
<p>{list.currentPage}/{list.lastPage}: {list.data.map((p) => p.slug).join(",")} prev={list.url.prev} next={list.url.next}</p>
-- src/pages/categories/[category].svelte --
<script module>
import { page } from "shugo"
import { products } from "../../lib/data"
// A category for every product: an address given twice is one page.
export const getStaticPaths = () => products.map((product) => ({ params: { category: product.category } }))
</script>
<p>{page.params.category}</p>
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
	expect(t, s.page("/categories/clay"), `<p>clay</p>`)
	if s.has("/products/none") {
		t.Error("a page that getStaticPaths did not return")
	}
	if want := 1 + 3 + 1 + 3 + 2 + 2; report.Pages.Total != want {
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
		{"a parameter without getStaticPaths", "-- src/pages/[id].svelte --\n<p>x</p>", "src/pages/[id].svelte:1:1: the name of this page has a parameter"},
		{"two pages of one address", "-- src/pages/a.svelte --\n<p>1</p>\n-- src/pages/a/index.svelte --\n<p>2</p>", "give the same addresses"},
		{"an unknown + file", "-- src/pages/+page.svelte --\n<p>x</p>", "names that start with + are kept for the framework"},
		{"a snippet nobody gave", "-- src/pages/index.svelte --\n<script>\nlet { children } = $props()\n</script>\n{@render children()}", "children is not a function"},
		{"a missing parameter", "-- src/pages/[id].svelte --\n<script module>\nexport const getStaticPaths = () => [{ params: {} }]\n</script>\n<p>x</p>", `returned params without "id"`},
		{"props of a route", "-- src/pages/[id].svelte --\n<script module>\nexport const getStaticPaths = () => [{ params: { id: \"a\" }, props: { title: \"A\" } }]\n</script>\n<p>x</p>", "gives its pages addresses alone: return { params: { … } }, and let every page ask for its data with query(), by page.params"},
		{"a list to paginate", "-- src/pages/[...page].svelte --\n<script module>\nimport { paginate } from \"shugo\"\nexport const getStaticPaths = () => paginate([1, 2, 3], { pageSize: 2 })\n</script>\n<p>x</p>", "paginate() in getStaticPaths takes how many items there are"},
		{"getStaticPaths that takes paginate", "-- src/pages/[...page].svelte --\n<script module>\nexport const getStaticPaths = ({ paginate }) => paginate(3, { pageSize: 2 })\n</script>\n<p>x</p>", `getStaticPaths gets no paginate: import { query, paginate } from "shugo"`},
		{"the global Shugo", "-- src/pages/index.svelte --\n<p>{Shugo.url.pathname}</p>", `There is no Shugo.url in Shugo: write page.url, with import { page } from "shugo"`},
		{"not a list", "-- src/pages/[id].svelte --\n<script module>\nexport const getStaticPaths = () => ({ id: 1 })\n</script>\n<p>x</p>", "must return a list of { params }"},
		{"a tag that is not a component", "-- src/pages/index.svelte --\n<script>\nconst Card = \"div\"\n</script>\n<Card />", "is not a function"},
		{"a component that shows itself forever", "-- src/pages/index.svelte --\n<script>\nimport Loop from \"../loop.svelte\"\n</script>\n<Loop />\n-- src/loop.svelte --\n<script>\nimport Loop from \"./loop.svelte\"\n</script>\n<Loop />", "shows itself again with the same parameters"},
		{"a script that waits", "-- src/pages/index.svelte --\n<script>\nconst items = await Promise.resolve([1])\n</script>\n<p>{items.length}</p>", "src/pages/index.svelte:2:15: a page is computed without waiting"},
		{"getStaticPaths that waits", "-- src/pages/[id].svelte --\n<script module>\nexport async function getStaticPaths() { return [{ params: { id: \"a\" } }] }\n</script>\n<p>x</p>", "getStaticPaths of /[id] waits"},
		{"Svelte 4 by its option", "-- src/pages/index.svelte --\n<svelte:options runes={false} />\n<script>\nexport let a = 1\n</script>\n<p>{a}</p>", "src/pages/index.svelte:1:17: runes={false} turns on the syntax of Svelte 4"},
		{"a page that imports its layout", "-- src/pages/+layout.svelte --\n<script>\nlet { children } = $props()\n</script>\n<main>{@render children()}</main>\n-- src/pages/index.svelte --\n<script>\nimport Layout from \"./+layout.svelte\"\n</script>\n<Layout><p>x</p></Layout>", "./+layout.svelte is a layout, which wraps the pages of its folder by itself, and no file imports it"},
		{"a component of a package", "-- src/pages/index.svelte --\n<script>\nimport Button from \"ui-kit/Button.svelte\"\n</script>\n<Button />", "ui-kit gives components of Svelte, and Shugo does not compile the components of packages yet"},
		{"a package of components", "-- node_modules/ui-kit/package.json --\n{\"name\": \"ui-kit\", \"svelte\": \"./index.js\"}\n-- src/pages/index.svelte --\n<script>\nimport { Button } from \"ui-kit\"\n</script>\n<Button />", "ui-kit gives components of Svelte"},
		{"a type imported as a value", "-- src/lib/shop.ts --\nexport interface Product { title: string }\nexport const price = 1\n-- src/pages/index.svelte --\n<script lang=\"ts\">\nimport { price, Product } from \"@/lib/shop\"\nconst item: Product = { title: \"Jug\" }\n</script>\n<p>{item.title} {price}</p>", "Product is a type, and a type is imported with type"},
		{"a component that shows itself without end", "-- src/components/Tree.svelte --\n<script>\nimport Tree from \"./Tree.svelte\"\nlet { depth } = $props()\n</script>\n<ul><Tree depth={{ next: depth }} /></ul>\n-- src/pages/index.svelte --\n<script>\nimport Tree from \"../components/Tree.svelte\"\n</script>\n<Tree depth={{}} />", "src/components/Tree.svelte shows itself without end: src/components/Tree.svelte → src/components/Tree.svelte"},
		{"markup inside braces", "-- src/pages/index.svelte --\n<script>\nconst cond = true\n</script>\n<p>{cond && <b>x</b>}</p>", "src/pages/index.svelte:4:13: markup cannot stand inside {…}"},
		{"a tag of the first syntax", "-- src/pages/index.svelte --\n<g-if cond={true}>x</g-if>", "there is no <g-if>: write {#if condition}"},
		{"a slot of Svelte 4", "-- src/pages/index.svelte --\n<script>\nimport C from \"../c.svelte\"\n</script>\n<C><b slot=\"x\">t</b></C>\n-- src/c.svelte --\n<p>c</p>", `slot="…" is a part of the slots of Svelte 4`},
		{"a template outside src", "-- src/pages/index.svelte --\n<script>\nimport X from \"../../components/X.svelte\"\n</script>\n<X />\n-- components/X.svelte --\n<p>x</p>", "components/X.svelte lies outside src/: the templates of a project are in src/"},
		{"<slot> of Svelte 4", "-- src/c.svelte --\n<div><slot /></div>\n-- src/pages/index.svelte --\n<p>x</p>", "there is no <slot>: write {@render children?.()}"},
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
// the .svelte file, not in the code generated from it.
func TestMistakesPointAtTheTemplate(t *testing.T) {
	cases := []struct{ name, page, want string }{
		{"a module that is not there", "<script>\nimport { helper } from \"../lib/missing\"\n</script>\n<h1>{helper()}</h1>", `src/pages/index.svelte:2:24: Could not resolve "../lib/missing"`},
		{"an expression cut short", "<p>ok</p>\n<h1 class={a +}>x</h1>", `src/pages/index.svelte:2:15: Unexpected token`},
		{"a name that is not a name", `{#each [1] as 1x}{x}{/each}`, "src/pages/index.svelte:1:15: Expected identifier or destructure pattern"},
		{"a mistake in the script", "<script>\nconst a = 1\nconst b = (a +\n</script>\n<p>{a}</p>", "src/pages/index.svelte:4:1: Unexpected token"},
		{"markup inside braces", "<ul>\n  {[1].map(n => <li>{n}</li>)}\n</ul>", "src/pages/index.svelte:2:17: markup cannot stand inside {…}"},
		{"a failure while computing", "<script>\nconst product = { title: \"x\" }\n</script>\n<p>\n  {product.missing.field}\n</p>", "at src/pages/index.svelte:5:"},
		{"a failure in a component", "<script>\nimport Card from \"../card.svelte\"\n</script>\n<Card />\n-- src/card.svelte --\n<script>let { item } = $props()</script>\n<b>{item.title}</b>", "at src/card.svelte:2:"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, "-- src/pages/index.svelte --\n"+c.page).tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("\n got: %v\nwant: %s", err, c.want)
			}
		})
	}
}

// What Svelte 5 warns about, Shugo warns about too, and builds the page.
func TestWarnings(t *testing.T) {
	s := newSite(t, `
-- src/components/Countdown.svelte --
<script>let { n } = $props()</script>
<b>{n}</b>{#if n > 0}<svelte:self n={n - 1} />{/if}
-- src/pages/index.svelte --
<script>
import Countdown from "../components/Countdown.svelte"
const Shown = Countdown
</script>
<svelte:component this={Shown} n={1} />
<p>{#if true}<div>block</div>{/if}</p>`)
	compiled, err := compile.Build(s.project())
	if err != nil {
		t.Fatal(err)
	}
	warnings := strings.Join(compiled.Warnings, "\n")
	for _, want := range []string{
		"src/components/Countdown.svelte:2:22: `<svelte:self>` is deprecated — use self-imports",
		"src/pages/index.svelte:5:1: `<svelte:component>` is deprecated in runes mode",
		"src/pages/index.svelte:6:14: `<div>` cannot be a child of `<p>`",
	} {
		if !strings.Contains(warnings, want) {
			t.Errorf("no warning %q in\n%s", want, warnings)
		}
	}
	s.build()
	expect(t, s.page("/"), "<b>1</b><b>0</b> <p><div>block</div></p>")
}

// Every page stands in the shell of src/site.html, copied as it is: the
// page goes to the place %shugo.body%, what its templates give the head to
// %shugo.head%. A project without the file gets the shell of the framework.
func TestShell(t *testing.T) {
	s := newSite(t, `
-- src/site.html --
<!doctype html>
<html lang="en">
<head><meta charset="utf-8">%shugo.head%</head>
<body class="site">%shugo.body%</body>
</html>
-- src/pages/index.svelte --
<svelte:head><title>One</title><meta name="a" content="b"><title>Two</title></svelte:head>
<p>x</p>`)
	s.build()
	expect(t, s.page("/"), "<!doctype html>\n<html lang=\"en\">\n<head><meta charset=\"utf-8\"><meta name=\"a\" content=\"b\"/><title>Two</title></head>\n<body class=\"site\"><p>x</p></body>\n</html>")

	// The shell is a source of the project: a change of it is a change of
	// the compiled project.
	s.write(`
-- src/site.html --
<html><head>%shugo.head%</head><body><div id="app">%shugo.body%</div></body></html>`)
	if changed, err := compile.Changed(s.project()); err != nil || !changed {
		t.Errorf("a new shell is not seen: %v, %v", changed, err)
	}
	s.build()
	expect(t, s.page("/"), `<html><head><meta name="a" content="b"/><title>Two</title></head><body><div id="app"><p>x</p></div></body></html>`)

	s.remove("src/site.html")
	s.build()
	expect(t, s.page("/"), strings.NewReplacer(skeleton.HeadPlace, `<meta name="a" content="b"/><title>Two</title>`, skeleton.BodyPlace, "<p>x</p>").Replace(skeleton.DefaultShell))
}

// The document around the pages is written once, in src/site.html: a
// template that writes it is told where it goes.
func TestShellMistakes(t *testing.T) {
	for _, c := range []struct{ name, files, want string }{
		{"a layout that writes the document", `
-- src/pages/+layout.svelte --
<script>
let { children } = $props()
</script>
<html><body>{@render children()}</body></html>
-- src/pages/index.svelte --
<p>x</p>`, "src/pages/+layout.svelte:4:1: <html> is written in src/site.html, the shell of every page"},
		{"a page that writes a head", `
-- src/pages/index.svelte --
<head><title>x</title></head><p>x</p>`, "<head> is written in src/site.html"},
		{"a doctype", `
-- src/pages/index.svelte --
<!doctype html><p>x</p>`, "<!doctype html> is written in src/site.html"},
		{"a shell without the place of the page", `
-- src/site.html --
<html><head>%shugo.head%</head><body></body></html>
-- src/pages/index.svelte --
<p>x</p>`, "src/site.html: %shugo.body% must stand once, where the page goes, and stands 0 times"},
		{"places the other way round", `
-- src/site.html --
%shugo.body%%shugo.head%
-- src/pages/index.svelte --
<p>x</p>`, "src/site.html: %shugo.head% must stand before %shugo.body%"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, c.files).tryBuild()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want %q", err, c.want)
			}
			if err != nil && strings.Contains(err.Error(), "\n") {
				t.Errorf("one mistake is told more than once:\n%v", err)
			}
		})
	}
}
