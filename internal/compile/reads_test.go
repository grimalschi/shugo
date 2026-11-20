package compile

import (
	"strings"
	"testing"
)

// treeOf returns what the template of a file reads, as "name=shape" for
// every variable in the order the template meets them.
func treeOf(t *testing.T, source string) string {
	t.Helper()
	u, err := readFile("components/x.shugo", source, 0, component)
	if err != nil {
		t.Fatalf("%v\n%s", err, source)
	}
	var out []string
	for i, name := range u.reads.names {
		out = append(out, name+"="+u.reads.shapes[i].String())
	}
	if u.reads.props != nil {
		out = append(out, "props="+u.reads.props.String())
	}
	return strings.Join(out, " ")
}

// The script that most cases share: a few variables, a function to call,
// and a component to show.
const scriptOfCases = `---
import Card from "./Card.shugo"
import { price, shop } from "../lib/shop"
import * as format from "../lib/format"
const item = load()
const items = loadAll()
const reviews = item.reviews.filter((r) => r.approved)
const attrs = { id: "x" }
---
`

// What a template reads of the variables of its script. A member is read
// alone; whatever is done to a value beyond reading its members reads it whole.
func TestWhatATemplateReads(t *testing.T) {
	cases := []struct{ name, template, want string }{
		{"a member", `{item.title}`, `item={"title":1}`},
		{"a member of a member", `<a href={item.links.self}>x</a>`, `item={"links":{"self":1}}`},
		{"an optional member", `{item?.discount?.percent}`, `item={"discount":{"percent":1}}`},
		{"the variable itself", `{item}`, `item=1`},
		{"a member read twice", `{item.title}{item.title}`, `item={"title":1}`},
		{"a member, then the value whole", `{item.title}{item}`, `item=1`},
		{"the value whole, then a member", `{item}{item.title}`, `item=1`},

		{"a condition reads whether the value is there", `<g-if cond={item.available}>yes</g-if>`, `item={"available":0}`},
		{"and what is read further", `<g-if cond={item.discount}>{item.discount.percent}</g-if>`, `item={"discount":{"percent":1}}`},
		{"every branch is read", `<g-if cond={item.a}>{item.b}</g-if><g-else-if cond={item.c}>{item.d}</g-else-if><g-else>{item.e}</g-else>`, `item={"a":0,"b":1,"c":0,"d":1,"e":1}`},
		{"a condition on the variable", `<g-if cond={item}>{item.title}</g-if>`, `item={"title":1}`},
		{"a condition that computes", `<g-if cond={item.stock > 0}>in stock</g-if>`, `item={"stock":1}`},
		{"not", `{!item.hidden}`, `item={"hidden":0}`},
		{"and: the left side matters only if there is nothing", `{item.flag && item.title}`, `item={"flag":0,"title":1}`},
		{"or: the left side is shown if it is there", `{item.nick || item.name}`, `item={"name":1,"nick":1}`},
		{"??", `{item.nick ?? "nobody"}`, `item={"nick":1}`},
		{"a choice", `{item.active ? item.title : "off"}`, `item={"active":0,"title":1}`},
		{"compared with nothing", `{item.discount == null ? "no" : item.price}`, `item={"discount":0,"price":1}`},
		{"compared with undefined", `{item.discount !== undefined}`, `item={"discount":0}`},
		{"compared with a value", `{item.kind === "cup"}`, `item={"kind":1}`},
		{"or in a condition", `<g-if cond={item.a || item.b}>x</g-if>`, `item={"a":0,"b":0}`},

		{"a loop reads the length and what the body reads", `<g-for each={item.variants} as="v">{v.title}</g-for>`, `item={"variants":[{"title":1}]}`},
		{"a loop over a variable", `<g-for each={reviews} as="r">{r.text} {r.author}</g-for>`, `reviews=[{"author":1,"text":1}]`},
		{"a loop that reads nothing of an item", `<g-for each={items} as="x">·</g-for>`, `items=[0]`},
		{"the item itself", `<g-for each={item.tags} as="tag">{tag}</g-for>`, `item={"tags":[1]}`},
		{"a loop inside a loop", `<g-for each={items} as="x"><g-for each={x.tags} as="tag">{tag}{x.id}</g-for></g-for>`, `items=[{"id":1,"tags":[1]}]`},
		{"the number of the item is not data", `<g-for each={items} as="x" index="i">{i}:{x.id}</g-for>`, `items=[{"id":1}]`},
		{"an item taken apart is read whole", `<g-for each={items} as="{ id, title }">{id}{title}</g-for>`, `items=[1]`},
		{"a loop over something computed", `<g-for each={items.filter((x) => x.ok)} as="x">{x.id}</g-for>`, `items=1`},
		{"the length alone", `{reviews.length}`, `reviews={"length":1}`},
		{"the length and the items", `{reviews.length}<g-for each={reviews} as="r">{r.text}</g-for>`, `reviews=[{"text":1}]`},
		{"a list asked for more than its items", `{items.total}<g-for each={items} as="x">{x.id}</g-for>`, `items=1`},
		{"a name of a loop hides a variable", `<g-for each={items} as="item">{item.id}</g-for>{item.title}`, `items=[{"id":1}] item={"title":1}`},

		{"a function the script imports is code", `{price(item.price)}`, `item={"price":1}`},
		{"a value given to a function is read whole", `{price(item)}`, `item=1`},
		{"a method reads what it is called on", `{item.tags.join(", ")}`, `item={"tags":1}`},
		{"a method of text", `{item.title.toUpperCase()}`, `item={"title":1}`},
		{"a function of an imported name", `{format.money(item.price)}`, `item={"price":1}`},
		{"an index", `{items[0].title}`, `items=1`},
		{"an index that is read", `{items[item.at]}`, `items=1 item={"at":1}`},
		{"spread attributes", `<p {...attrs}>x</p>`, `attrs=1`},
		{"arithmetic", `{item.price * 2 + item.tax}`, `item={"price":1,"tax":1}`},
		{"text with places", "{`/products/${item.slug}`}", `item={"slug":1}`},
		{"a function written in place", `{items.map((x) => x.id + item.suffix)}`, `items=1 item={"suffix":1}`},
		{"a name of a function written in place is its own", `{[1, 2].map((item) => item * 2)}`, ``},
		{"an object written in place", `{JSON.stringify({ a: item.a, b: [item.b], ...item.rest })}`, `item={"a":1,"b":1,"rest":1}`},
		{"typeof", `{typeof item.x}`, `item={"x":1}`},
		{"a global is not data", `{Math.max(1, 2)}{String(item.id)}`, `item={"id":1}`},

		{"an imported value is data", `{shop.name}`, `shop={"name":1}`},
		{"a component with a plain parameter", `<Card id={item.id} />`, `item={"id":1}`},
		{"a component given an object", `<Card product={item} />`, `item=1`},
		{"a component in a loop", `<g-for each={item.variants} as="v"><Card variant={v} /></g-for>`, `item={"variants":[1]}`},
		{"what is put inside a component", `<Card><b>{item.title}</b><i slot="foot">{item.price}</i></Card>`, `item={"price":1,"title":1}`},
		{"spread parameters", `<Card {...item.options} />`, `item={"options":1}`},
		{"what a slot shows when it is empty", `<g-slot>{item.title}</g-slot>`, `item={"title":1}`},
		{"markup", `<g-html value={item.html} />`, `item={"html":1}`},

		{"the parameters of the file", `{Shugo.props.title}`, `props={"title":1}`},
		{"the parameters whole", `{JSON.stringify(Shugo.props)}`, `props=1`},
		{"the page and the slots are not data of the script", `{Shugo.url.pathname}{Shugo.params.slug}<g-if cond={Shugo.slots.has("x")}>y</g-if>`, ``},
		{"types are taken out before reading", `{(item as any).title}{item.discount!.percent}`, `item={"discount":{"percent":1},"title":1}`},
	}
	for _, c := range cases {
		if got := treeOf(t, scriptOfCases+c.template); got != c.want {
			t.Errorf("%s: %s\n got: %s\nwant: %s", c.name, c.template, got, c.want)
		}
	}
}

// A component that the script has chosen is a value the template is given.
func TestComponentChosenByTheScript(t *testing.T) {
	got := treeOf(t, "---\nimport A from \"./A.shugo\"\nimport B from \"./B.shugo\"\nconst Tag = Shugo.props.big ? A : B\n---\n<Tag />")
	if got != "Tag=1" {
		t.Errorf("got %s", got)
	}
}

// What a template may not do, each with what the message must say.
func TestWhatATemplateMayNotDo(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"call a function of the script", "---\nfunction label(x) { return x }\n---\n{label(1)}", "label, a function that the script declares"},
		{"call an arrow of the script", "---\nconst label = (x) => x\n---\n{label(1)}", "label, a function that the script declares"},
		{"pass a function of the script", "---\nconst label = (x) => x\n---\n{[1].map(label)}", "label, a function that the script declares"},
		{"use a class of the script", "---\nclass Money {}\n---\n{new Money()}", "Money, a function that the script declares"},
		{"wait", "---\nconst data = 1\n---\n{await data}", "a template cannot wait"},
		{"ask the time", "{new Date().getFullYear()}", "new Date() gives the time of the moment"},
		{"write what is not an expression", "{a +}", "Unexpected"},
	}
	for _, c := range cases {
		_, err := readFile("components/x.shugo", c.source, 0, component)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s:\n got: %v\nwant: %s", c.name, err, c.want)
		}
	}
	// A date made of something is not the time of the moment.
	if _, err := readFile("components/x.shugo", "{new Date(2026, 0, 1).getFullYear()}", 0, component); err != nil {
		t.Errorf("a date with arguments: %v", err)
	}
}

// What the script declares, and what it does that decides where it runs.
func TestBindings(t *testing.T) {
	u, err := readFile("pages/x.shugo", `---
import Card, { a as b, type T } from "./x"
import * as ns from "./y"
export function getStaticPaths() { return [] }
export const title = "x"
const { item, big = false, ...rest } = Shugo.props
if (big) { var hoisted = 1 }
function helper() {}
const arrow = async () => { await 1 }
---
<p>x</p>`, 0, page)
	if err != nil {
		t.Fatal(err)
	}
	for name, from := range map[string]string{"Card": "./x", "b": "./x", "ns": "./y"} {
		if u.names.imports[name] != from {
			t.Errorf("%s is imported from %q, want %q", name, u.names.imports[name], from)
		}
	}
	for _, name := range []string{"getStaticPaths", "title", "item", "big", "rest", "hoisted", "helper", "arrow"} {
		if !u.names.vars[name] {
			t.Errorf("%s is not among the variables", name)
		}
	}
	for _, name := range []string{"getStaticPaths", "helper", "arrow"} {
		if !u.names.funcs[name] {
			t.Errorf("%s is not known to be a function", name)
		}
	}
	if u.names.waits || u.names.now {
		t.Errorf("waits=%v now=%v for a script that only declares a function that waits", u.names.waits, u.names.now)
	}

	for source, want := range map[string][2]bool{
		"const data = await load()":               {true, false},
		"for await (const x of stream) {}":        {true, false},
		"const year = new Date().getFullYear()":   {false, true},
		"const day = new Date(2026, 0, 1)":        {false, false},
		"const later = () => new Date()":          {false, true},
		"const load = async () => await fetch(u)": {false, false},
	} {
		u, err := readFile("components/x.shugo", "---\n"+source+"\n---\n<p>x</p>", 0, component)
		if err != nil {
			t.Fatalf("%s: %v", source, err)
		}
		if got := [2]bool{u.names.waits, u.names.now}; got != want {
			t.Errorf("%s: waits and now are %v, want %v", source, got, want)
		}
	}
}

// How code is found to reach outside of the program.
func TestReachesOut(t *testing.T) {
	for source, want := range map[string]string{
		`const x = await load()`:                                                  "waits for data (await)",
		`export const get = () => fetch("https://x")`:                             "calls fetch",
		`const key = process.env.KEY`:                                             "reads the environment of the process",
		`export const double = (x: number) => x * 2`:                              "",
		`const text = "we fetch nothing and await no one"`:                        "",
		`const o = { fetch: 1 }; export const f = o.fetch`:                        "calls fetch",
		`export const f = (o: { fetch: number }) => o.fetch + (o as any).process`: "calls fetch",
	} {
		if got := reachesOut(source, true); got != want {
			t.Errorf("%s: %q, want %q", source, got, want)
		}
	}
	if got := reachesOut("const x = await load()", false); got != "" {
		t.Errorf("await in a script, which is asked about apart: %q", got)
	}
}
