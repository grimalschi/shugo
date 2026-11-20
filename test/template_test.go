package test

import (
	"strings"
	"testing"
)

// one builds a site of one page and returns its HTML.
func one(t *testing.T, page string, more ...string) string {
	t.Helper()
	s := newSite(t, "-- pages/index.shugo --\n"+page+"\n"+strings.Join(more, "\n"))
	s.build()
	return s.page("/")
}

func TestText(t *testing.T) {
	got := one(t, `---
const title = "Tom & <Jerry>"
const nothing = null
const list = ["a", 1, false, null, ["b", "c"]]
---
<h1>{title}</h1>
<p>{nothing}{undefined}{false}|{true}|{0}|{12.5}|{"quotes \"'"}|{list}|{`+"`${1 + 1}`"+`}</p>`)
	expect(t, got, `<h1>Tom &amp; &lt;Jerry&gt;</h1>
<p>|true|0|12.5|quotes &#34;&#39;|a1bc|2</p>`)
}

func TestAttributes(t *testing.T) {
	got := one(t, `---
const url = "/a?b=1&c=2"
const off = false
const rest = { id: "x", hidden: true, title: null, "data-n": 5 }
---
<a href={url} class={off && "active"} title={null} data-on={off} aria-hidden={off} hidden={!off} tabindex={0} lang="en" download>a</a>
<input disabled={off} checked value={"<1>"} {...rest} />
<img src="a.png" alt='it"s'>`)
	expect(t, got, `<a href="/a?b=1&amp;c=2" data-on="false" aria-hidden="false" hidden tabindex="0" lang="en" download>a</a>
<input checked value="&lt;1&gt;" id="x" hidden data-n="5">
<img src="a.png" alt='it"s'>`)
}

func TestAddresses(t *testing.T) {
	got := one(t, `---
const bad = "javascript:alert(1)"
const sneaky = " Java\nScript:alert(1)"
const picture = "data:image/png;base64,AAAA"
const attrs = { href: bad }
---
<a href={bad}>1</a><a href={sneaky}>2</a><a href="/ok">3</a><a href={"mailto:a@b.c"}>4</a><img src={picture}><a title={bad}>5</a><a {...attrs}>6</a>`)
	expect(t, got, `<a href="about:invalid#blocked">1</a><a href="about:invalid#blocked">2</a><a href="/ok">3</a><a href="mailto:a@b.c">4</a><img src="data:image/png;base64,AAAA"><a title="javascript:alert(1)">5</a><a href="about:invalid#blocked">6</a>`)
}

func TestConditions(t *testing.T) {
	got := one(t, `---
const n = 2
---
<g-if cond={n === 1}>one</g-if>
<g-else-if cond={n === 2}>two</g-else-if>
<g-else>many</g-else>
<g-if cond={n > 5}>big</g-if>
<g-if cond={n}><g-if cond={n > 1}>nested</g-if><g-else>no</g-else></g-if>`)
	expect(t, got, "two\n\nnested")
}

func TestLoops(t *testing.T) {
	got := one(t, `---
const items = [{ id: 1, tags: ["a", "b"] }, { id: 2, tags: [] }]
const missing = undefined
const set = new Set(["x", "y"])
const pairs = new Map([["k", "v"]])
---
<g-for each={items} as="item" index="i"><li>{i}:{item.id}<g-for each={item.tags} as="tag">[{tag}]</g-for></li></g-for>
<g-for each={missing} as="x">never</g-for>|<g-for each={set} as="x">{x}</g-for>|<g-for each={pairs} as="[key, value]">{key}={value}</g-for>|<g-for each={items} as="{ id }">{id}</g-for>`)
	expect(t, got, "<li>0:1[a][b]</li><li>1:2</li>\n|xy|k=v|12")
}

func TestRawHTMLAndRawElements(t *testing.T) {
	got := one(t, `---
const body = "<b>bold</b>"
---
<article><g-html value={body} /></article>
<script>if (a < b) { console.log("{not an expression}") }</script>
<style is:inline>a { color: red }</style>
<!-- a comment with {braces} -->
<br><hr /><div class="x" />`)
	expect(t, got, `<article><b>bold</b></article>
<script>if (a < b) { console.log("{not an expression}") }</script>
<style>a { color: red }</style>
<!-- a comment with {braces} -->
<br><hr><div class="x"></div>`)
}

const card = `-- components/Card.shugo --
---
const { title, big = false } = Shugo.props
---
<section class={big ? "card big" : "card"}>
<h2>{title}</h2>
<g-slot><p>empty</p></g-slot>
<g-if cond={Shugo.slots.has("footer")}><footer><g-slot name="footer" /></footer></g-if>
</section>`

func TestComponentsAndSlots(t *testing.T) {
	got := one(t, `---
import Card from "../components/Card.shugo"
const name = "world"
---
<Card title="A" />
<Card title={"B"} big>
  <p>Hello, {name}</p>
  <small slot="footer">footer {name}</small>
</Card>
<Card title="C"><g-fragment slot="footer">one</g-fragment><g-fragment slot="footer">two</g-fragment></Card>`, card)
	expect(t, got, `<section class="card">
<h2>A</h2>
<p>empty</p>

</section>
<section class="card big">
<h2>B</h2>

  <p>Hello, world</p>
  
<footer><small>footer world</small>
</footer>
</section>
<section class="card">
<h2>C</h2>
<p>empty</p>
<footer>onetwo</footer>
</section>`)
}

func TestNestedComponentsPassSlots(t *testing.T) {
	got := one(t, `---
import Outer from "../components/Outer.shugo"
---
<Outer><b>text</b></Outer>`, `-- components/Outer.shugo --
---
import Inner from "./Inner.shugo"
---
<div class="outer"><Inner><i>before</i><g-slot /></Inner></div>
-- components/Inner.shugo --
<div class="inner"><g-slot /></div>`)
	expect(t, got, `<div class="outer"><div class="inner"><i>before</i><b>text</b></div></div>`)
}

func TestRecursionAndDynamicComponents(t *testing.T) {
	got := one(t, `---
import Tree from "../components/Tree.shugo"
import A from "../components/A.shugo"
import B from "../components/B.shugo"
const tree = { name: "root", children: [{ name: "a", children: [{ name: "a1", children: [] }] }, { name: "b", children: [] }] }
const Tag = tree.children.length > 1 ? B : A
---
<Tree node={tree} /><Tag />`, `-- components/Tree.shugo --
---
const { node } = Shugo.props
---
<li>{node.name}<g-if cond={node.children.length}><ul><g-for each={node.children} as="child"><Shugo.self node={child} /></g-for></ul></g-if></li>
-- components/A.shugo --
<a>A</a>
-- components/B.shugo --
<b>B</b>`)
	expect(t, got, `<li>root<ul><li>a<ul><li>a1</li></ul></li><li>b</li></ul></li><b>B</b>`)
}

func TestTypeScript(t *testing.T) {
	got := one(t, `---
import { price, type Product } from "@/lib/shop"
interface Props { product?: Product }
const { product = { title: "Jug", cents: 1250 } as Product } = Shugo.props as Props
enum Size { Small = "S", Large = "L" }
const sizes: Size[] = [Size.Small, Size.Large]
---
<p>{(product as Product).title}: {price(product!.cents)} {sizes.map((size: Size): string => size.toLowerCase()).join("/")}</p>`, `-- lib/shop.ts --
export interface Product { title: string; cents: number }
export const price = (cents: number): string => (cents / 100).toFixed(2)`)
	expect(t, got, `<p>Jug: 12.50 s/l</p>`)
}

func TestNamespaceOfComponentsAndShorthand(t *testing.T) {
	got := one(t, `---
import * as ui from "@/components/ui"
const src = "/a.png"
const title = "T"
---
<ui.Button href="/x">go</ui.Button><img {src} alt=""><ui.Title {title} /><p title="{not code}">{"{"}{/* nothing */}</p>`, `-- components/ui.ts --
export { default as Button } from "./Button.shugo"
export { default as Title } from "./Title.shugo"
-- components/Button.shugo --
<a class="button" href={Shugo.props.href}><g-slot /></a>
-- components/Title.shugo --
<h1>{Shugo.props.title}</h1>`)
	expect(t, got, `<a class="button" href="/x">go</a><img src="/a.png" alt=""><h1>T</h1><p title="{not code}">{</p>`)
}

func TestStyles(t *testing.T) {
	s := newSite(t, `
-- pages/+layout.shugo --
---
import "../styles/main.css"
---
<html><head><title>T</title></head><body><g-slot /></body></html>
<style>
  body { margin: 0 }
</style>
-- pages/index.shugo --
---
import Card from "../components/Card.shugo"
---
<Card /><Card />
<style is:inline>h1 { color: blue }</style>
-- components/Card.shugo --
<b class="card">card</b>
<style>.card { color: green }</style>
-- styles/main.css --
@import "./base.css";
.logo { background: url("./logo.svg") }
-- styles/base.css --
html { font-family: sans-serif }
-- styles/logo.svg --
<svg xmlns="http://www.w3.org/2000/svg"/>
`)
	s.build()
	page := s.page("/")
	// A <style> of a template goes to the style sheet of the site, once,
	// however many times the template is shown; is:inline stays in the page.
	if strings.Contains(page, "color: green") || strings.Contains(page, "margin: 0") || !strings.Contains(page, "<style>h1 { color: blue }</style>") {
		t.Errorf("page: %s", page)
	}
	link := page[strings.Index(page, `href="/_assets/`)+len(`href="/_assets/`):]
	sheet := s.read("_build/assets/" + link[:strings.Index(link, `"`)])
	for _, want := range []string{"font-family:sans-serif", `.logo{background:url("/_assets/logo.`, ".card{color:green}", "body{margin:0}"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("the style sheet has no %q: %s", want, sheet)
		}
	}
	if strings.Count(sheet, ".card{") != 1 {
		t.Errorf("the styles of a component are there more than once: %s", sheet)
	}
	// What a style sheet points at is copied next to it.
	name := sheet[strings.Index(sheet, "/_assets/logo.")+len("/_assets/"):]
	if got := s.read("_build/assets/" + name[:strings.Index(name, `"`)]); !strings.Contains(got, "<svg") {
		t.Errorf("the picture of the style sheet: %q", got)
	}
}
