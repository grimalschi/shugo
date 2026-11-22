package test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The random test writes pages at random, builds them, and compares every
// page with HTML that this file works out by itself, straight from the tree
// of the page and the data, without JavaScript, values or skeletons. Then it
// changes the data at random several times: after every update the pages
// must again be what the data says.
//
// The pages use three components written by hand below, with children,
// snippets, shared blocks and a component inside a component; what is
// random is everything around them.

const randomComponents = `
-- src/lib/data.ts --
import { readFileSync } from "node:fs"
export const data = JSON.parse(readFileSync("data.json", "utf8"))
-- src/components/Leaf.shugo --
<script>
let { text } = $props()
</script>
<i>{text}</i>
-- src/components/Box.shugo --
<script>
let { item, big = false, children, foot } = $props()
</script>
<div class={big ? "box big" : "box"}>{item.name}{#if children}{@render children()}{:else}none:{item.id}{/if}{#if foot}<footer>{@render foot()}</footer>{/if}</div>
-- src/components/Wrap.shugo --
<script>
import Leaf from "./Leaf.shugo"
let { children } = $props()
</script>
<section><Leaf text="w" />{@render children?.()}</section>
`

// tree is a piece of a page: it can write itself as a template and as HTML.
type tree interface {
	source(out *strings.Builder)
	html(out *strings.Builder, s scope)
}

// scope is the variables a piece of a page sees.
type scope map[string]any

// expr is an expression: its JavaScript and what it gives.
type expr struct {
	js   string
	eval func(scope) any
}

type (
	text  string
	value struct{ e expr }
	raw   struct{ e expr }
	elem  struct {
		tag   string
		attrs []attr
		kids  []tree
	}
	attr struct {
		name    string
		literal string // for name="literal"; otherwise e is used
		bare    bool
		e       expr
	}
	choose struct {
		tests []expr // one more branch than tests means the last is {:else}
		then  [][]tree
	}
	each struct {
		list  expr
		item  string
		index string
		kids  []tree
	}
	leaf struct{ e expr }
	box  struct {
		item       expr
		big        bool
		body, foot []tree
	}
	wrap struct{ kids []tree }
)

func (t text) source(out *strings.Builder)        { out.WriteString(string(t)) }
func (t text) html(out *strings.Builder, _ scope) { out.WriteString(string(t)) }

func (v value) source(out *strings.Builder) { out.WriteString("{" + v.e.js + "}") }
func (v value) html(out *strings.Builder, s scope) {
	out.WriteString(escapeText(stringify(v.e.eval(s))))
}

func (r raw) source(out *strings.Builder) { out.WriteString("{@html " + r.e.js + "}") }
func (r raw) html(out *strings.Builder, s scope) {
	out.WriteString(stringify(r.e.eval(s)))
}

func (e elem) source(out *strings.Builder) {
	out.WriteString("<" + e.tag)
	for _, a := range e.attrs {
		switch {
		case a.bare:
			out.WriteString(" " + a.name)
		case a.e.js == "":
			out.WriteString(" " + a.name + `="` + a.literal + `"`)
		default:
			out.WriteString(" " + a.name + "={" + a.e.js + "}")
		}
	}
	out.WriteString(">")
	sources(out, e.kids)
	out.WriteString("</" + e.tag + ">")
}

func (e elem) html(out *strings.Builder, s scope) {
	out.WriteString("<" + e.tag)
	for _, a := range e.attrs {
		switch {
		case a.bare:
			out.WriteString(" " + a.name + `=""`)
		case a.e.js == "":
			out.WriteString(" " + a.name + `="` + a.literal + `"`)
		default:
			out.WriteString(attributeOf(a.name, a.e.eval(s)))
		}
	}
	out.WriteString(">")
	htmls(out, e.kids, s)
	out.WriteString("</" + e.tag + ">")
}

func (c choose) source(out *strings.Builder) {
	for i, body := range c.then {
		switch {
		case i == 0:
			out.WriteString("{#if " + c.tests[i].js + "}")
		case i < len(c.tests):
			out.WriteString("{:else if " + c.tests[i].js + "}")
		default:
			out.WriteString("{:else}")
		}
		sources(out, body)
	}
	out.WriteString("{/if}")
}

func (c choose) html(out *strings.Builder, s scope) {
	for i, body := range c.then {
		if i >= len(c.tests) || truthy(c.tests[i].eval(s)) {
			htmls(out, body, s)
			return
		}
	}
}

func (e each) source(out *strings.Builder) {
	out.WriteString("{#each " + e.list.js + " as " + e.item)
	if e.index != "" {
		out.WriteString(", " + e.index)
	}
	out.WriteString("}")
	sources(out, e.kids)
	out.WriteString("{/each}")
}

func (e each) html(out *strings.Builder, s scope) {
	list, _ := e.list.eval(s).([]any)
	for i, item := range list {
		inner := scope{}
		for name, v := range s {
			inner[name] = v
		}
		inner[e.item] = item
		if e.index != "" {
			inner[e.index] = float64(i)
		}
		htmls(out, e.kids, inner)
	}
}

func (l leaf) source(out *strings.Builder) { out.WriteString("<Leaf text={" + l.e.js + "} />") }
func (l leaf) html(out *strings.Builder, s scope) {
	out.WriteString("<i>" + escapeText(stringify(l.e.eval(s))) + "</i>")
}

func (b box) source(out *strings.Builder) {
	out.WriteString("<Box item={" + b.item.js + "}")
	if b.big {
		out.WriteString(" big")
	}
	if b.body == nil && b.foot == nil {
		out.WriteString(" />")
		return
	}
	out.WriteString(">")
	sources(out, b.body)
	if b.foot != nil {
		out.WriteString("{#snippet foot()}")
		sources(out, b.foot)
		out.WriteString("{/snippet}")
	}
	out.WriteString("</Box>")
}

func (b box) html(out *strings.Builder, s scope) {
	item := b.item.eval(s).(map[string]any)
	class := "box"
	if b.big {
		class = "box big"
	}
	out.WriteString(`<div class="` + class + `">` + escapeText(stringify(item["name"])))
	if b.body != nil {
		htmls(out, b.body, s)
	} else {
		out.WriteString("none:" + escapeText(stringify(item["id"])))
	}
	if b.foot != nil {
		out.WriteString("<footer>")
		htmls(out, b.foot, s)
		out.WriteString("</footer>")
	}
	out.WriteString("</div>")
}

func (w wrap) source(out *strings.Builder) {
	out.WriteString("<Wrap>")
	sources(out, w.kids)
	out.WriteString("</Wrap>")
}

func (w wrap) html(out *strings.Builder, s scope) {
	out.WriteString("<section><i>w</i>")
	htmls(out, w.kids, s)
	out.WriteString("</section>")
}

func sources(out *strings.Builder, kids []tree) {
	for _, kid := range kids {
		kid.source(out)
	}
}

func htmls(out *strings.Builder, kids []tree, s scope) {
	for _, kid := range kids {
		kid.html(out, s)
	}
}

// What follows says once more, in Go, how Svelte writes values; it is
// written from what Svelte does, not from the code of Shugo. Svelte 5.59.1
// rendered 400 pages of this generator exactly as this model does.

// stringify is how a value is written into a page: a string as it is, null
// and undefined as nothing, anything else as String() of JavaScript makes it.
func stringify(v any) string {
	if v == nil {
		return ""
	}
	return jsString(v)
}

// jsString is String() of JavaScript: an array is its items joined by
// commas, with null and undefined as empty items.
func jsString(v any) string {
	switch v := v.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = stringify(item)
		}
		return strings.Join(parts, ",")
	}
	return "[object Object]"
}

func truthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v != ""
	}
	return true
}

var (
	textEscaper      = strings.NewReplacer("&", "&amp;", "<", "&lt;")
	attributeEscaper = strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;")
)

func escapeText(s string) string { return textEscaper.Replace(s) }

// attributeOf writes an attribute whose value is an expression. null and
// undefined leave it out; hidden is there or not, as the value is true or
// false, unless the value is "until-found", and "" is there, as in markup;
// class takes the truthy items of an array, joined by spaces, and is left
// out when empty.
func attributeOf(name string, v any) string {
	switch {
	case name == "hidden" && v != "until-found":
		if v != "" && !truthy(v) {
			return ""
		}
		return ` hidden=""`
	case name == "class":
		class := classOf(v)
		if class == "" {
			return ""
		}
		return ` class="` + attributeEscaper.Replace(class) + `"`
	case v == nil:
		return ""
	}
	return " " + name + `="` + attributeEscaper.Replace(jsString(v)) + `"`
}

// classOf is the class an expression gives: Svelte passes arrays through
// clsx.
func classOf(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case []any:
		var parts []string
		for _, item := range v {
			if truthy(item) {
				parts = append(parts, classOf(item))
			}
		}
		return strings.Join(parts, " ")
	}
	return jsString(v)
}

// maker writes random pages.
type maker struct{ r *rand.Rand }

func (m *maker) pick(options ...string) string { return options[m.r.IntN(len(options))] }

func field(path ...string) func(scope) any {
	return func(s scope) any {
		var at any = map[string]any(s)
		for _, name := range path {
			object, _ := at.(map[string]any)
			at = object[name]
		}
		return at
	}
}

// exprs returns the expressions that can stand where the variables of vars
// are in scope: "item" for an item of the list, "tag" for one of its tags,
// "i" for a number.
func (m *maker) exprs(vars map[string]bool) []expr {
	out := []expr{
		{"data.title", field("data", "title")},
		{"data.count", field("data", "count")},
		{"data.flag", field("data", "flag")},
		{"data.none", field("data", "none")},
		{"data.missing", field("data", "missing")},
		{"data.user.name", field("data", "user", "name")},
		{"data.count + 1", func(s scope) any { return field("data", "count")(s).(float64) + 1 }},
		{"data.items.length", func(s scope) any { return float64(len(field("data", "items")(s).([]any))) }},
		{"data.title.toUpperCase()", func(s scope) any { return strings.ToUpper(field("data", "title")(s).(string)) }},
		{"`[${data.user.name}]`", func(s scope) any { return "[" + field("data", "user", "name")(s).(string) + "]" }},
		{"data.flag && data.title", func(s scope) any {
			if field("data", "flag")(s) == true {
				return field("data", "title")(s)
			}
			return false
		}},
	}
	if vars["item"] {
		out = append(out,
			expr{"item.name", field("item", "name")},
			expr{"item.price", field("item", "price")},
			expr{"item.active", field("item", "active")},
			expr{"item.tags", field("item", "tags")},
			expr{"item.price * 2", func(s scope) any { return field("item", "price")(s).(float64) * 2 }},
			expr{`item.tags.join("; ")`, func(s scope) any {
				tags := field("item", "tags")(s).([]any)
				parts := make([]string, len(tags))
				for i, tag := range tags {
					parts[i] = tag.(string)
				}
				return strings.Join(parts, "; ")
			}},
			expr{`item.active ? "on" : null`, func(s scope) any {
				if field("item", "active")(s) == true {
					return "on"
				}
				return nil
			}},
		)
	}
	if vars["tag"] {
		out = append(out, expr{"tag", field("tag")})
	}
	if vars["i"] {
		out = append(out, expr{"i", field("i")}, expr{"i % 2 === 0", func(s scope) any { return int(field("i")(s).(float64))%2 == 0 }})
	}
	return out
}

func (m *maker) expr(vars map[string]bool) expr {
	all := m.exprs(vars)
	return all[m.r.IntN(len(all))]
}

// place is where a piece of a page is written: the variables in scope, and
// whether an element that holds only phrasing content is around, p, span or
// b, where HTML allows no div and no section.
type place struct {
	vars     map[string]bool
	phrasing bool
}

func (p place) with(names ...string) place {
	vars := map[string]bool{}
	for name := range p.vars {
		vars[name] = true
	}
	for _, name := range names {
		vars[name] = true
	}
	return place{vars, p.phrasing}
}

func (m *maker) nodes(p place, depth int) []tree {
	out := make([]tree, 1+m.r.IntN(3))
	for i := range out {
		out[i] = m.node(p, depth)
	}
	return out
}

func (m *maker) node(p place, depth int) tree {
	kind := m.r.IntN(12)
	if depth <= 0 {
		kind = m.r.IntN(4)
	}
	switch kind {
	case 0:
		return text(m.pick("a", "bc", "x y", "&amp;", "a b", "1 &lt; 2"))
	case 1, 2:
		return value{m.expr(p.vars)}
	case 3:
		return leaf{m.expr(p.vars)}
	case 4, 5:
		tag := m.pick("p", "div", "span", "b")
		if p.phrasing {
			tag = m.pick("span", "b")
		}
		inner := p
		inner.phrasing = tag != "div"
		e := elem{tag: tag, kids: m.nodes(inner, depth-1)}
		names := []string{"class", "title", "hidden", "data-x", "aria-label", "id"}
		m.r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
		for _, name := range names[:m.r.IntN(3)] {
			switch m.r.IntN(4) {
			case 0:
				e.attrs = append(e.attrs, attr{name: name, literal: m.pick("a", "b c", "x&amp;y")})
			case 1:
				e.attrs = append(e.attrs, attr{name: name, bare: true})
			default:
				e.attrs = append(e.attrs, attr{name: name, e: m.expr(p.vars)})
			}
		}
		return e
	case 6, 7:
		c := choose{}
		for range 1 + m.r.IntN(3) {
			c.tests = append(c.tests, m.expr(p.vars))
			c.then = append(c.then, m.nodes(p, depth-1))
		}
		if m.r.IntN(2) == 0 {
			c.then = append(c.then, m.nodes(p, depth-1))
		}
		return c
	case 8:
		if p.vars["item"] && !p.vars["tag"] {
			return each{list: expr{"item.tags", field("item", "tags")}, item: "tag", kids: m.nodes(p.with("tag"), depth-1)}
		}
		if p.vars["item"] {
			return value{m.expr(p.vars)}
		}
		e := each{list: expr{"data.items", field("data", "items")}, item: "item"}
		inner := p.with("item")
		if m.r.IntN(2) == 0 {
			e.index, inner = "i", inner.with("i")
		}
		e.kids = m.nodes(inner, depth-1)
		return e
	case 9, 10:
		// Box writes a div and Wrap a section: not where only phrasing
		// content may stand.
		switch {
		case p.phrasing:
			return leaf{m.expr(p.vars)}
		case kind == 10 || !p.vars["item"]:
			return wrap{m.nodes(p, depth-1)}
		}
		b := box{item: expr{"item", field("item")}, big: m.r.IntN(2) == 0}
		if m.r.IntN(3) > 0 {
			b.body = m.nodes(p, depth-1)
		}
		if m.r.IntN(2) == 0 {
			b.foot = m.nodes(p, depth-1)
		}
		return b
	default:
		return raw{expr{"data.html", field("data", "html")}}
	}
}

// data makes the data of the pages.
func (m *maker) data() map[string]any {
	word := func() string {
		return m.pick("Jug", "Tom & Jerry", `"quoted"`, "<b>bold</b>", "pitcher", "it's", "", "a  b")
	}
	items := make([]any, m.r.IntN(5))
	for i := range items {
		tags := make([]any, m.r.IntN(4))
		for j := range tags {
			tags[j] = m.pick("new", "sale", "a&b", "<eco>")
		}
		items[i] = map[string]any{
			"id": float64(m.r.IntN(1000)), "name": word(), "price": float64(m.r.IntN(50)) / 2,
			"active": m.r.IntN(2) == 0, "tags": tags,
		}
	}
	return map[string]any{
		"title": word(), "count": float64(m.r.IntN(4)), "flag": m.r.IntN(2) == 0, "none": nil,
		"html": m.pick("<em>raw</em>", "", "a &amp; b"), "user": map[string]any{"name": word()}, "items": items,
	}
}

func TestRandomPages(t *testing.T) {
	seed := uint64(20261120)
	if text := os.Getenv("SHUGO_SEED"); text != "" {
		seed, _ = strconv.ParseUint(text, 10, 64)
	}
	pages, rounds := 150, 6
	if testing.Short() {
		pages, rounds = 30, 2
	}
	m := &maker{r: rand.New(rand.NewPCG(seed, 1))}

	var files strings.Builder
	files.WriteString(randomComponents)
	trees := make([][]tree, pages)
	for i := range trees {
		trees[i] = m.nodes(place{}, 4)
		// The edges of a template are trimmed; a tag at each end keeps the
		// text inside as written.
		fmt.Fprintf(&files, "-- src/pages/p%d.shugo --\n<script>\nimport Leaf from \"../components/Leaf.shugo\"\nimport Box from \"../components/Box.shugo\"\nimport Wrap from \"../components/Wrap.shugo\"\nimport { data } from \"../lib/data\"\n</script>\n<main>", i)
		sources(&files, trees[i])
		files.WriteString("</main>\n")
	}
	s := newSite(t, files.String())

	for round := range rounds {
		data := m.data()
		encoded, _ := json.Marshal(data)
		if err := os.WriteFile(filepath.Join(s.root, "data.json"), encoded, 0o644); err != nil {
			t.Fatal(err)
		}
		// The data of these pages lies in a file, past the data database:
		// every round is built anew.
		s.build()
		failed := 0
		for i, nodes := range trees {
			var want strings.Builder
			want.WriteString("<main>")
			htmls(&want, nodes, scope{"data": data})
			want.WriteString("</main>")
			if got := s.page(fmt.Sprintf("/p%d", i)); got != want.String() {
				var source strings.Builder
				sources(&source, nodes)
				t.Errorf("seed %d, round %d, page %d\ntemplate: %s\n    data: %s\n     got: %s\n    want: %s", seed, round, i, source.String(), encoded, got, want.String())
				if failed++; failed == 3 {
					t.FailNow()
				}
			}
		}
	}
}
