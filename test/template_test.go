package test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// one builds a site of one page and returns its HTML.
func one(t *testing.T, page string, more ...string) string {
	t.Helper()
	s := newSite(t, "-- src/pages/index.svelte --\n"+page+"\n"+strings.Join(more, "\n"))
	s.build()
	return s.page("/")
}

// What a template gives is what Svelte 5.59 gives for it: every page these
// tests expect was rendered by Svelte from the same template.

func TestText(t *testing.T) {
	got := one(t, `<script>
const title = "Tom & <Jerry>"
const nothing = null
const list = ["a", 1, false, null, ["b", "c"]]
</script>
<h1>{title}</h1>
<p>{nothing}{undefined}{false}|{true}|{0}|{12.5}|{"quotes \"'"}|{list}|{`+"`${1 + 1}`"+`}</p>`)
	expect(t, got, `<h1>Tom &amp; &lt;Jerry></h1> <p>false|true|0|12.5|quotes "'|a,1,false,,b,c|2</p>`)
}

func TestAttributes(t *testing.T) {
	got := one(t, `<script>
const url = "/a?b=1&c=2"
const off = false
const rest = { id: "x", hidden: true, title: null, "data-n": 5 }
</script>
<a href={url} class={off && "active"} title={null} data-on={off} aria-hidden={off} hidden={!off} tabindex={0} lang="en" download>a</a>
<input disabled={off} checked value={"<1>"} {...rest} />
<img src="a.png" alt='it"s'>`)
	expect(t, got, `<a href="/a?b=1&amp;c=2" class="false" data-on="false" aria-hidden="false" hidden="" tabindex="0" lang="en" download="">a</a> <input checked="" value="&lt;1>" id="x" hidden="" data-n="5"/> <img src="a.png" alt="it&quot;s"/>`)
}

func TestClassAndStyle(t *testing.T) {
	got := one(t, `<script>
const on = true
const color = "red"
</script>
<p class={["a", false, { b: on, c: !on }]} class:d={on} style="margin: 0" style:color style:--size="2px" style:padding|important="1px">x</p>
<i class="  one   two " style="  top : 0 "></i>`)
	expect(t, got, `<p class="a b d" style="margin: 0; color: red; --size: 2px; padding: 1px !important;">x</p> <i class="one two" style="top : 0"></i>`)

	// A class or a style without a value, and empty ones, as Svelte writes them.
	got = one(t, `<script>
const on = true
const color = "red"
</script>
<p class>1</p><p class class:x={on}>2</p><p style>3</p><p style style:color>4</p><p class="">5</p><p style="">6</p>`)
	expect(t, got, `<p class="">1</p><p class="true x">2</p><p style="">3</p><p style="color: red;">4</p><p>5</p><p style="">6</p>`)
}

// Values reach the page as they are, with the characters that split the
// messages between the code of the project and Shugo among them.
func TestControlCharacters(t *testing.T) {
	got := one(t, `<script>
const odd = "a\u0000b\u0001c\u0002d\u0003e\u0004f\u0005g\u0004@"
</script>
<p title={odd}>{odd}</p>{@html odd}`)
	odd := "a\x00b\x01c\x02d\x03e\x04f\x05g\x04@"
	expect(t, got, `<p title="`+odd+`">`+odd+`</p>`+odd)

	// What may be the name of an attribute is decided by Svelte.
	expect(t, one(t, "<script>\nconst rest = { \"a\\u0001b\": 1, \"c d\": 2 }\n</script>\n<p {...rest}>x</p>"), "<p a\x01b=\"1\">x</p>")
}

// Svelte writes comments for the browser to hydrate a page with; a static
// page has none of them. A comment the page writes itself, with {@html},
// stays.
func TestNoCommentsOfSvelte(t *testing.T) {
	got := one(t, `<script>
import Card from "../components/Card.svelte"
import Field from "../components/Field.svelte"
const items = [1, 2]
const none = []
const tag = "h2"
const promise = Promise.resolve(1)
let { children } = $props()
</script>
<svelte:head><title>T {items.length}</title><meta name="a" content="b"><Card n={9} /></svelte:head>
{#if items.length > 1}<p>many</p>{:else if items.length}<p>one</p>{:else}<p>none</p>{/if}
{#each items as item, i (item)}<i>{i}:{item}</i>{/each}
{#each none as item}<i>{item}</i>{:else}<i>empty</i>{/each}
{#key items}<b>key</b>{/key}
{#await promise}<i>waiting</i>{:then value}<i>{value}</i>{/await}
<svelte:element this={tag}>dynamic {tag}</svelte:element>
{@html "<!--more--><!----><b>raw</b><!--[-->"}
{#snippet line(x)}<u>{x}</u>{/snippet}
{@render line("a")}{@render children?.()}
text {items[0]} and {items[1]}
<Card n={1}><s>given</s></Card><Card n={2} />
<Field />
<select value="b"><option>a</option><option>b</option></select>
<svelte:boundary><b>bounded</b></svelte:boundary>`,
		`-- src/components/Card.svelte --
<script>let { n, children } = $props()</script>
<section>{#if n > 1}{n}{/if}{@render children?.()}</section>`,
		`-- src/components/Field.svelte --
<script>const id = $props.id()</script>
<label for={id}>x</label><input {id}>`)
	// The title goes after the rest of the head, as in Svelte.
	expect(t, got, `<meta name="a" content="b"/><section>9</section><title>T 2</title>`+
		`<p>many</p> <i>0:1</i><i>1:2</i> <i>empty</i> <b>key</b> <i>waiting</i> <h2>dynamic h2</h2> <!--more--><!----><b>raw</b><!--[--> `+
		`<u>a</u> text 1 and 2 <section><s>given</s></section><section>2</section> <label for="s1">x</label><input id="s1"/> `+
		`<select><option>a</option><option selected="">b</option></select> <b>bounded</b>`)
}

// A template takes the modules of Svelte that have a use on a server from
// the Svelte that Shugo ships; a project installs nothing for them.
func TestModulesOfSvelte(t *testing.T) {
	got := one(t, `<script>
import { writable, get } from "svelte/store"
import { SvelteMap } from "svelte/reactivity"
import { cubicOut } from "svelte/easing"
import { fade } from "svelte/transition"
const count = writable(2)
const names = new SvelteMap([["a", "A"]])
</script>
<p transition:fade>{get(count)} {names.get("a")} {cubicOut(1)}</p>`)
	expect(t, got, "<p>2 A 1</p>")

	s := newSite(t, `
-- src/pages/index.svelte --
<script>
import { run } from "svelte/legacy"
</script>
<p>x</p>`)
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "svelte/legacy is not a part of it") {
		t.Errorf("a module of Svelte that a server has no use for: %v", err)
	}
}

// An address is written as it is given, as Svelte writes it: what a link
// may lead to is the project's affair.
func TestAddresses(t *testing.T) {
	got := one(t, `<script>
const bad = "javascript:alert(1)"
const sneaky = " Java\nScript:alert(1)"
const picture = "data:image/png;base64,AAAA"
const attrs = { href: bad }
</script>
<a href={bad}>1</a><a href={sneaky}>2</a><a href="/ok">3</a><a href={"mailto:a@b.c"}>4</a><img src={picture}><a title={bad}>5</a><a {...attrs}>6</a>`)
	expect(t, got, `<a href="javascript:alert(1)">1</a><a href=" Java
Script:alert(1)">2</a><a href="/ok">3</a><a href="mailto:a@b.c">4</a><img src="data:image/png;base64,AAAA"/><a title="javascript:alert(1)">5</a><a href="javascript:alert(1)">6</a>`)
}

func TestConditions(t *testing.T) {
	got := one(t, `<script>
const n = 2
</script>
{#if n === 1}one{:else if n === 2}two{:else}many{/if}
{#if n > 5}big{/if}
{#if n}{#if n > 1}nested{:else}no{/if}{/if}`)
	expect(t, got, "two  nested")
}

func TestLoops(t *testing.T) {
	got := one(t, `<script>
const items = [{ id: 1, tags: ["a", "b"] }, { id: 2, tags: [] }]
const missing = undefined
const set = new Set(["x", "y"])
const pairs = new Map([["k", "v"]])
</script>
{#each items as item, i}<li>{i}:{item.id}{#each item.tags as tag}[{tag}]{/each}</li>{/each}
{#each missing as x}never{/each}|{#each set as x}{x}{/each}|{#each pairs as [key, value]}{key}={value}{/each}|{#each items as { id }}{id}{/each}`)
	expect(t, got, "<li>0:1[a][b]</li><li>1:2</li> |xy|k=v|12")
}

func TestEmptyListsKeysAwaitAndConstants(t *testing.T) {
	got := one(t, `<script>
const none = []
const items = [{ id: 7, n: 2 }]
</script>
{#each none as x}<b>{x}</b>{:else}<i>empty</i>{/each}
{#each items as item (item.id)}{@const double = item.n * 2}<u>{double}</u>{/each}
{#await 5 then value}<q>{value}</q>{/await}{#await Promise.resolve(1)}<s>wait</s>{:then v}{v}{/await}
{#key items}<em>k</em>{/key}
{#each { length: 2 }, i}{i}{/each}`)
	expect(t, got, `<i>empty</i> <u>4</u> <q>5</q><s>wait</s> <em>k</em> 01`)
}

func TestRawHTMLAndRawElements(t *testing.T) {
	got := one(t, `<script>
const body = "<b>bold</b>"
</script>
<article>{@html body}</article>
<div><script>if (a < b) { console.log("{not an expression}") }</script></div>
<div><style>a { color: red }</style></div>
<!-- a comment with {braces} -->
<br><hr /><div class="x" />`)
	expect(t, got, `<article><b>bold</b></article> <div><script>if (a < b) { console.log("{not an expression}") }</script></div> <div><style>a { color: red }</style></div> <br/><hr/><div class="x"></div>`)
}

func TestDynamicElementsSelectAndTextarea(t *testing.T) {
	got := one(t, `<script>
const level = 3
const none = null
const size = "m"
const note = "a < b"
</script>
<svelte:element this={"h" + level} class="t">title</svelte:element><svelte:element this={none}>never</svelte:element><svelte:element this="br" />
<select value={size}><option value="s">S</option><option>m</option></select>
<textarea value={note}></textarea>`)
	expect(t, got, `<h3 class="t">title</h3><br> <select><option value="s">S</option><option selected="">m</option></select> <textarea>a &lt; b</textarea>`)
}

const card = `-- src/components/Card.svelte --
<script>
let { title, big = false, children, footer } = $props()
</script>
<section class={big ? "card big" : "card"}>
<h2>{title}</h2>
{#if children}{@render children()}{:else}<p>empty</p>{/if}
{#if footer}<footer>{@render footer()}</footer>{/if}
</section>`

// Markup given to a component is a snippet: its children, or a snippet
// written inside its tag.
func TestComponentsAndSnippets(t *testing.T) {
	got := one(t, `<script>
import Card from "../components/Card.svelte"
const name = "world"
</script>
<Card title="A" />
<Card title={"B"} big>
  <p>Hello, {name}</p>
  {#snippet footer()}<small>footer {name}</small>{/snippet}
</Card>
<Card title="C">{#snippet footer()}one two{/snippet}</Card>`, card)
	expect(t, got, `<section class="card"><h2>A</h2> <p>empty</p> </section> <section class="card big"><h2>B</h2> <p>Hello, world</p> <footer><small>footer world</small></footer></section> <section class="card"><h2>C</h2> <p>empty</p> <footer>one two</footer></section>`)
}

// A component may show the children it was given inside another component,
// or hand them to one as a parameter.
func TestNestedComponentsPassSnippets(t *testing.T) {
	got := one(t, `<script>
import Outer from "../components/Outer.svelte"
</script>
<Outer><b>text</b></Outer>`, `-- src/components/Outer.svelte --
<script>
import Inner from "./Inner.svelte"
import Shown from "./Shown.svelte"
let { children } = $props()
</script>
<div class="outer"><Inner><i>before</i>{@render children()}</Inner></div><Shown content={children} />
-- src/components/Inner.svelte --
<script>let { children } = $props()</script>
<div class="inner">{@render children()}</div>
-- src/components/Shown.svelte --
<script>let { content } = $props()</script>
<p>{@render content()}</p>`)
	expect(t, got, `<div class="outer"><div class="inner"><i>before</i><b>text</b></div></div><p><b>text</b></p>`)
}

func TestSnippetsWithParameters(t *testing.T) {
	got := one(t, `<script>
import List from "../components/List.svelte"
const label = "№"
</script>
{#snippet cell(x)}<td>{label}{x}</td>{/snippet}
<table><tbody><tr>{@render cell(1)}{@render cell(2)}</tr></tbody></table>
<List items={["a", "b"]}>{#snippet row(item, i)}<li>{i}: {item} {label}</li>{/snippet}</List>`, `-- src/components/List.svelte --
<script>let { items, row } = $props()</script>
<ul>{#each items as item, i}{@render row(item, i)}{/each}</ul>`)
	expect(t, got, `<table><tbody><tr><td>№1</td><td>№2</td></tr></tbody></table> <ul><li>0: a №</li><li>1: b №</li></ul>`)
}

func TestRenderCallWithParenthesizedExpressions(t *testing.T) {
	got := one(t, `<script>
const first = true
const missing = null
function fail() { throw new Error("an optional call must skip its arguments") }
</script>
{#snippet card(value, label)}<p>{value} {label}</p>{/snippet}
{#snippet other(value, label)}<b>{label} {value}</b>{/snippet}
{@render (first ? card : other)((1, 2), /[(]/.source)}{@render missing?.(fail())}`)
	expect(t, got, "<p>2 [(]</p>")
}

func TestRecursionAndDynamicComponents(t *testing.T) {
	got := one(t, `<script>
import Tree from "../components/Tree.svelte"
import Self from "../components/Self.svelte"
import A from "../components/A.svelte"
import B from "../components/B.svelte"
const tree = { name: "root", children: [{ name: "a", children: [{ name: "a1", children: [] }] }, { name: "b", children: [] }] }
const Tag = tree.children.length > 1 ? B : A
</script>
<Tree node={tree} /><Self node={tree} /><Tag /><svelte:component this={A} /><svelte:component this={null} />`, `-- src/components/Tree.svelte --
<script>
let { node } = $props()
</script>
<li>{node.name}{#if node.children.length}<ul>{#each node.children as child}<svelte:self node={child} />{/each}</ul>{/if}</li>
-- src/components/Self.svelte --
<script>
import Self from "./Self.svelte"
let { node } = $props()
</script>
<li>{node.name}{#if node.children.length}<ul>{#each node.children as child}<Self node={child} />{/each}</ul>{/if}</li>
-- src/components/A.svelte --
<a>A</a>
-- src/components/B.svelte --
<b>B</b>`)
	tree := `<li>root<ul><li>a<ul><li>a1</li></ul></li><li>b</li></ul></li>`
	expect(t, got, tree+tree+`<b>B</b><a>A</a>`)
}

// A template takes TypeScript as Svelte does: the types are taken away, and
// what TypeScript would turn into code of its own, as an enum, is a mistake.
func TestTypeScript(t *testing.T) {
	got := one(t, `<script lang="ts">
import { price, type Product } from "@/lib/shop"
interface Props { product?: Product }
const { product = { title: "Jug", cents: 1250 } as Product } = $props() as Props
const Size = { Small: "S", Large: "L" } as const
type Size = (typeof Size)[keyof typeof Size]
const sizes: Size[] = [Size.Small, Size.Large]
</script>
<p>{(product as Product).title}: {price(product!.cents)} {sizes.map((size: Size): string => size.toLowerCase()).join("/")}</p>`, `-- src/lib/shop.ts --
export interface Product { title: string; cents: number }
export const price = (cents: number): string => (cents / 100).toFixed(2)`)
	expect(t, got, `<p>Jug: 12.50 s/l</p>`)

	s := newSite(t, "-- src/pages/index.svelte --\n<script lang=\"ts\">\nenum Size { Small }\n</script>\n<p>{Size.Small}</p>")
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "src/pages/index.svelte:2:1: TypeScript language features like enums are not natively supported") {
		t.Errorf("an enum: %v", err)
	}
}

func TestNamespaceOfComponentsAndShorthand(t *testing.T) {
	got := one(t, `<script>
import * as ui from "@/components/ui"
const src = "/a.png"
const title = "T"
</script>
<ui.Button href="/x">go</ui.Button><img {src} alt=""><ui.Title {title} /><p title="a {'x'} b">{"{"}</p>`, `-- src/components/ui.ts --
export { default as Button } from "./Button.svelte"
export { default as Title } from "./Title.svelte"
-- src/components/Button.svelte --
<script>let { href, children } = $props()</script>
<a class="button" {href}>{@render children?.()}</a>
-- src/components/Title.svelte --
<script>let { title } = $props()</script>
<h1>{title}</h1>`)
	expect(t, got, `<a class="button" href="/x">go</a><img src="/a.png" alt=""/><h1>T</h1><p title="a x b">{</p>`)
}

func TestStyles(t *testing.T) {
	s := newSite(t, `
-- src/site.html --
<html><head><title>T</title>%shugo.head%</head><body>%shugo.body%</body></html>
-- src/pages/+layout.svelte --
<script>
import "../styles/main.css"
let { children } = $props()
</script>
<main>{@render children()}</main>
<style>
  main { margin: 0 }
  :global(body) { padding: 0 }
</style>
-- src/pages/index.svelte --
<script>
import Card from "../components/Card.svelte"
</script>
<Card /><Card />
<div><style>h1 { color: blue }</style></div>
-- src/components/Card.svelte --
<b class="card">card</b><i class></i><i class=""></i>
<style>.card { color: green }</style>
-- src/styles/main.css --
@import "./base.css";
.logo { background: url("./logo.svg") }
-- src/styles/base.css --
html { font-family: sans-serif }
-- src/styles/logo.svg --
<svg xmlns="http://www.w3.org/2000/svg"/>
`)
	s.build()
	page := s.page("/")
	// The <style> of a component goes to the style sheet of the site, once,
	// however many times the component is shown, and applies to its own
	// elements; a <style> inside an element stays in the page as it is.
	if strings.Contains(page, "color: green") || strings.Contains(page, "margin: 0") || !strings.Contains(page, "<div><style>h1 { color: blue }</style></div>") {
		t.Errorf("page: %s", page)
	}
	card := scopeClass(t, page, "card")
	// The scoping class goes to the elements the styles of the component
	// select, as in Svelte.
	cardHTML := `<b class="card ` + card + `">card</b><i class=""></i><i></i>`
	if !strings.Contains(page, cardHTML+cardHTML) {
		t.Errorf("the cards are not scoped: %s", page)
	}
	link := page[strings.Index(page, `href="/_assets/`)+len(`href="/_assets/`):]
	sheetFile := link[:strings.Index(link, `"`)]
	sheet := s.read("_build/assets/" + sheetFile)
	for _, want := range []string{"font-family:sans-serif", `.logo{background:url("/_assets/logo.`, ".card." + card + "{color:green}", "main.svelte-", "body{padding:0}"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("the style sheet has no %q: %s", want, sheet)
		}
	}
	if strings.Count(sheet, ".card.") != 1 {
		t.Errorf("the styles of a component are there more than once: %s", sheet)
	}
	// What a style sheet points at is copied next to it.
	name := sheet[strings.Index(sheet, "/_assets/logo.")+len("/_assets/"):]
	logoFile := name[:strings.Index(name, `"`)]
	if got := s.read("_build/assets/" + logoFile); !strings.Contains(got, "<svg") {
		t.Errorf("the picture of the style sheet: %q", got)
	}

	out := filepath.Join(s.root, "dist")
	if done := s.export(out); done.Files != 3 {
		t.Fatalf("expected page, stylesheet and SVG in export: %+v", done)
	}
	s.write(`-- src/styles/logo.svg --
<svg xmlns="http://www.w3.org/2000/svg" width="20"/>`)
	s.build()
	if done := s.export(out); done.Files != 3 || done.Written != 3 || done.Removed != 2 {
		t.Fatalf("expected a new stylesheet and SVG and page: %+v", done)
	}
	for _, name := range []string{sheetFile, logoFile} {
		if _, err := os.Stat(filepath.Join(s.root, "_build", "assets", name)); err != nil {
			t.Errorf("previous asset unavailable to the server: %v", err)
		}
		if _, err := os.Stat(filepath.Join(out, "_assets", name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("previous asset retained by export: %s, %v", name, err)
		}
	}
}

// scopeClass returns the class that scopes the styles of a component, as
// it stands after another class of an element on the page.
func scopeClass(t *testing.T, page, after string) string {
	t.Helper()
	at := strings.Index(page, `class="`+after+` svelte-`)
	if at < 0 {
		t.Fatalf("no element of class %s is scoped: %s", after, page)
	}
	rest := page[at+len(`class="`+after+` `):]
	return rest[:strings.IndexByte(rest, '"')]
}

// A template of the old extension .shugo stops the build with a name for
// it, and the published site stays as it was.
func TestLegacySourceRequiresRenameWithoutDroppingThePublishedSite(t *testing.T) {
	s := newSite(t, `
-- src/pages/index.svelte --
<p>Published</p>
`)
	s.build()
	s.write(`
-- src/pages/old.shugo --
<p>Legacy source</p>
`)
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "old.svelte") || !strings.Contains(err.Error(), "old.shugo") {
		t.Fatalf("expected a migration diagnostic that names both paths, got %v", err)
	}
	expect(t, s.page("/"), "<p>Published</p>")
	if s.has("/old") {
		t.Fatal("a failed build published a new page")
	}
}
