package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/evaluate"
)

// expectComputed compares what an update ran and computed with what it
// should have: the scripts of everything, the templates of what has changed.
func expectComputed(t *testing.T, r *evaluate.Report, pages, shared int) {
	t.Helper()
	if r.Computed.Pages != pages || r.Computed.Shared != shared {
		t.Errorf("computed %d pages and %d shared blocks, want %d and %d", r.Computed.Pages, r.Computed.Shared, pages, shared)
	}
}

// An update runs the scripts of every page and shared block, and computes
// the templates of those that read something that has changed.
func TestOnlyWhatChangedIsComputed(t *testing.T) {
	s := newSite(t, shop)
	first := s.build()
	expectComputed(t, first, 5, 5)

	nothing := s.update()
	expectComputed(t, nothing, 0, 0)
	// Five pages; the header, the layout and three cards.
	if nothing.Loaded.Pages != 5 || nothing.Loaded.Shared != 5 {
		t.Errorf("the scripts of %d pages and %d shared blocks ran", nothing.Loaded.Pages, nothing.Loaded.Shared)
	}

	// No template reads the note.
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		products[0]["note"] = "changed"
		return products
	})
	expectComputed(t, s.update(), 0, 0)

	// The header alone reads the motto.
	s.change(func(data map[string]any, products []map[string]any) []map[string]any {
		data["motto"] = "Handmade"
		return products
	})
	expectComputed(t, s.update(), 0, 1)

	// The price of the cup is read by its page and by its card. The main
	// page and the page of the jug show the card and are not computed.
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		products[1]["price"] = 6
		return products
	})
	r := s.update()
	expectComputed(t, r, 1, 1)
	if r.Render != 3 || !strings.Contains(s.page("/products/jug"), "Cup — 6") {
		t.Errorf("pages with new HTML: %v; the page of the jug: %s", r.RenderNames, s.page("/products/jug"))
	}

	// The list of related products is read by the page of the jug only.
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		products[0]["related"] = []string{"tray", "cup"}
		return products
	})
	r = s.update()
	expectComputed(t, r, 1, 0)
	expectChanges(t, r, evaluate.Changes{Changed: 1, ChangedNames: []string{"/products/jug"}}, evaluate.Changes{})

	// A new product: its page and its card, the main page that lists it,
	// and the header that counts the products.
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		return append(products, map[string]any{"slug": "pot", "title": "Pot", "price": 12, "related": []string{}})
	})
	expectComputed(t, s.update(), 2, 2)

	// A check computes everything, and finds nothing that the fingerprints missed.
	checked, err := s.run(evaluate.Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	expectComputed(t, checked, 6, 6)
	if len(checked.Unseen) != 0 || checked.Pages.Changed+checked.Shared.Changed != 0 {
		t.Errorf("the check found %v; changed %+v %+v", checked.Unseen, checked.Pages, checked.Shared)
	}
}

// run computes the compiled project again with the given options.
func (s *site) run(options evaluate.Options) (*evaluate.Report, error) {
	s.t.Helper()
	p := s.project()
	compiled, err := compile.Load(p)
	if err != nil {
		s.t.Fatal(err)
	}
	options.Log = &s.printed
	defer s.pages.Refresh()
	return evaluate.Run(p, compiled, s.store, options)
}

// A template works on a copy of what it reads, made of data alone.
func TestWhatATemplateIsGiven(t *testing.T) {
	got := one(t, `---
import { year } from "../lib/dates"
const made = new Date(Date.UTC(2026, 10, 20))
const tags = new Set(["clay", "wood"])
const stock = new Map([["jug", 2], ["cup", 0]])
const item = { title: "Jug", price: { amount: 10, unit: "$" }, hidden: { secret: 1 }, big: 12345678901234567890n }
const nothing = null
---
<p>{year(made)} {made.getUTCMonth()} {tags.has("clay")} {tags.size} {stock.get("jug")}</p>
<p>{item.title} {item.price.amount}{item.price.unit} {item.big} {item.missing} {nothing?.title}</p>
<ul><g-for each={tags} as="tag"><li>{tag}</li></g-for></ul>
<ul><g-for each={stock} as="[name, count]"><li>{name}:{count}</li></g-for></ul>
<g-if cond={item.hidden}>hidden is there</g-if>`, `-- lib/dates.ts --
export const year = (date: Date) => date.getUTCFullYear()`)
	expect(t, got, `<p>2026 10 true 2 2</p>
<p>Jug 10$ 12345678901234567890  </p>
<ul><li>clay</li><li>wood</li></ul>
<ul><li>jug:2</li><li>cup:0</li></ul>
hidden is there`)
}

// What a project may not do, and what it is told. Every one of these would
// let a page show something else without Shugo knowing that it has to be
// computed again.
func TestRulesOfLoading(t *testing.T) {
	const data = `-- data.json --
{"title": "Shop"}
-- lib/data.ts --
import { readFileSync } from "node:fs"
export const data = JSON.parse(readFileSync("data.json", "utf8"))
export const shout = (text: string) => text.toUpperCase()
`
	cases := []struct{ name, files, want string }{
		{"a template calls a function of a module that reads a file", data + `-- pages/index.shugo --
---
import { data, shout } from "../lib/data"
---
<p>{shout(data.title)}</p>`, "pages/index.shugo → lib/data.ts → node:fs: the template calls shout of a module that reads files"},

		{"a template calls a function of a module that waits when it is imported", `-- lib/remote.ts --
export const posts = await Promise.resolve([{ title: "a" }])
export const shout = (text: string) => text.toUpperCase()
-- pages/index.shugo --
---
import { posts, shout } from "../lib/remote"
---
<p>{shout(posts[0].title)}</p>`, "this module waits for data when it is imported"},

		{"a template calls a function that asks the network", `-- lib/net.ts --
export const get = (address: string) => String(fetch(address))
-- pages/index.shugo --
---
import { get } from "../lib/net"
---
<p>{get("https://example.com")}</p>`, "fetch is called in code that a template calls"},

		{"a template calls a function that asks the clock", `-- lib/clock.ts --
export const stamp = () => String(Date.now())
-- pages/index.shugo --
---
import { stamp } from "../lib/clock"
---
<p>{stamp()}</p>`, "the time of the moment is asked for with Date.now()"},

		{"a component that waits is given an object", `-- components/More.shugo --
---
const { post } = Shugo.props
const more = await Promise.resolve([post.title])
---
<p>{more.join()}</p>
-- pages/index.shugo --
---
import More from "../components/More.shugo"
const post = { title: "a" }
---
<More post={post} />`, "components/More.shugo loads data: its script waits for data (await). So it is computed once for the whole site and takes only plain values: text, numbers, true, false, null. Here post is an object"},

		{"a component that reads a file is given an object", data + `-- components/Title.shugo --
---
import { data } from "../lib/data"
const { style } = Shugo.props
---
<h1 class={style.name}>{data.title}</h1>
-- pages/index.shugo --
---
import Title from "../components/Title.shugo"
---
<Title style={{ name: "big" }} />`, "components/Title.shugo loads data: its script imports lib/data.ts, which reads files (node:fs)"},

		{"a component that loads data reads the page", `-- components/Crumbs.shugo --
---
const home = await Promise.resolve("Home")
---
<p>{home} › {Shugo.url.pathname}</p>
-- pages/index.shugo --
---
import Crumbs from "../components/Crumbs.shugo"
---
<Crumbs />`, "this component loads data — its script waits for data (await) — so it is computed once for the whole site; and it reads the page it is shown on"},

		{"a variable that the template reads holds a function", `-- pages/index.shugo --
---
const tools = { shout: (text: string) => text.toUpperCase(), name: "tools" }
---
<p>{tools.shout("a")}</p>`, "pages/index.shugo: the template reads tools.shout, which is a function"},

		{"a variable that the template reads holds an instance of a class", `-- lib/money.ts --
export class Money { constructor(public amount: number) {} }
-- pages/index.shugo --
---
import { Money } from "../lib/money"
const price = new Money(5)
---
<p>{JSON.stringify(price)}</p>`, "the template reads price, which is an instance of Money"},

		{"data that contains itself", `-- pages/index.shugo --
---
const node: any = { name: "a" }
node.self = node
---
<p>{JSON.stringify(node)}</p>`, "the template reads node"},
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

// What the rules leave open.
func TestWhatLoadingAllows(t *testing.T) {
	// The same component that waits, called with plain values: a shared
	// block, computed once.
	got := one(t, `---
import More from "../components/More.shugo"
---
<More title="a" /><More title="a" /><More title="b" />`, `-- components/More.shugo --
---
const { title } = Shugo.props
const more = await Promise.resolve([title, title])
---
<p>{more.join("+")}</p>`)
	expect(t, got, `<p>a+a</p><p>a+a</p><p>b+b</p>`)

	// A member of an instance of a class is read as any member is.
	got = one(t, `---
import { Money } from "../lib/money"
const price = new Money(5)
---
<p>{price.amount}</p>`, `-- lib/money.ts --
export class Money { constructor(public amount: number) {} }`)
	expect(t, got, `<p>5</p>`)

	// A module may read a file in a function that no template calls: what a
	// template calls of the same module is still code that loads nothing.
	got = one(t, `---
import { load, shout } from "../lib/both"
const title = load()
---
<p>{shout(title)}</p>`, `-- lib/both.ts --
import { existsSync } from "node:fs"
export const load = () => (existsSync("shugo.json") ? "with settings" : "bare")
export const shout = (text: string) => text.toUpperCase()`)
	expect(t, got, `<p>BARE</p>`)

	// The time of the moment is data like any other: the script takes it.
	got = one(t, `---
const year = new Date().getUTCFullYear() > 2000
---
<p>{year}</p>`)
	expect(t, got, `<p>true</p>`)
}

// A layout that reads the page is computed with every page it stands
// around, script and all; its data is a part of what the page is computed from.
func TestLayoutThatReadsThePage(t *testing.T) {
	s := newSite(t, `
-- data.json --
{"title": "Shop", "note": "x"}
-- lib/data.ts --
import { readFileSync } from "node:fs"
export const data = () => JSON.parse(readFileSync("data.json", "utf8"))
-- pages/+layout.shugo --
---
import { data } from "../lib/data"
const { title } = data()
const here = Shugo.url.pathname
const wide = Shugo.slots.has("side")
---
<body class={wide ? "wide" : "narrow"}><h1>{title}</h1><nav>{here}</nav><g-slot name="side" /><main><g-slot /></main></body>
-- pages/index.shugo --
<p>home</p>
-- pages/about.shugo --
<aside slot="side">side</aside><p>about</p>`)
	first := s.build()
	if first.Shared.Total != 0 {
		t.Errorf("shared blocks: %+v", first.Shared)
	}
	expect(t, s.page("/"), `<body class="narrow"><h1>Shop</h1><nav>/</nav><main><p>home</p></main></body>`)
	expect(t, s.page("/about"), `<body class="wide"><h1>Shop</h1><nav>/about</nav><aside>side</aside><main><p>about</p></main></body>`)

	expectComputed(t, s.update(), 0, 0)
	if err := os.WriteFile(filepath.Join(s.root, "data.json"), []byte(`{"title": "Shop", "note": "y"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	expectComputed(t, s.update(), 0, 0)
	if err := os.WriteFile(filepath.Join(s.root, "data.json"), []byte(`{"title": "Market", "note": "y"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	expectComputed(t, s.update(), 2, 0)
	expect(t, s.page("/about"), `<body class="wide"><h1>Market</h1><nav>/about</nav><aside>side</aside><main><p>about</p></main></body>`)
}

// The compiler does not look into packages. A package that reads a file by
// itself, called by a component that only shows what it is given, makes a
// page that changes without Shugo knowing. A check finds it.
func TestCheckFindsWhatIsReadPastShugo(t *testing.T) {
	s := newSite(t, `
-- node_modules/sneaky/package.json --
{"name": "sneaky", "version": "1.0.0", "main": "index.js"}
-- node_modules/sneaky/index.js --
exports.motto = () => require("fs").readFileSync("motto.txt", "utf8").trim()
-- motto.txt --
old
-- components/Motto.shugo --
---
import { motto } from "sneaky"
const { style } = Shugo.props
---
<b class={style.name}>{motto()}</b>
-- pages/index.shugo --
---
import Motto from "../components/Motto.shugo"
---
<Motto style={{ name: "big" }} />`)
	s.build()
	expect(t, s.page("/"), `<b class="big">old</b>`)

	if err := os.WriteFile(filepath.Join(s.root, "motto.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Nothing that the page is seen to read has changed.
	expectComputed(t, s.update(), 0, 0)
	expect(t, s.page("/"), `<b class="big">old</b>`)

	checked, err := s.run(evaluate.Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(checked.Unseen) != 1 || checked.Unseen[0] != "the page /" {
		t.Errorf("the check found %v", checked.Unseen)
	}
	expect(t, s.page("/"), `<b class="big">new</b>`)
}

// The store may know a shared block whose data is gone. If nothing shows
// the block any more, it goes too; if something does, the update fails and
// says whose script it was.
func TestSharedBlockWhoseDataIsGone(t *testing.T) {
	s := newSite(t, shop)
	s.build()
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		products[0]["related"] = []string{"cup"} // nothing shows the tray any more
		return products[:2]
	})
	r := s.update()
	if r.Shared.Removed != 1 || r.Shared.RemovedNames[0] != cardTray {
		t.Errorf("shared blocks: %+v", r.Shared)
	}

	// The cup is gone, and the page of the jug still shows its card.
	s.change(func(_ map[string]any, products []map[string]any) []map[string]any {
		return products[:1]
	})
	_, err := s.run(evaluate.Options{})
	if err == nil || !strings.Contains(err.Error(), "while computing the shared block "+cardCup) {
		t.Errorf("%v", err)
	}
}

// A script that changes data it shares with other scripts makes what a page
// shows depend on which script ran first. It is found when a page is
// computed from other data than its fingerprint was made of.
func TestScriptThatChangesSharedData(t *testing.T) {
	_, err := newSite(t, `
-- lib/data.ts --
export const list = [3, 1, 2]
export const load = () => (Date.now() > 0 ? list : [])
-- pages/first.shugo --
---
import { load } from "../lib/data"
const numbers = load()
---
<p>{numbers.join()}</p>
-- pages/second.shugo --
---
import { load } from "../lib/data"
const sorted = load().sort()
---
<p>{sorted.join()}</p>`).tryBuild()
	if err == nil || !strings.Contains(err.Error(), "another script changes data that this one has read") {
		t.Errorf("%v", err)
	}
}
