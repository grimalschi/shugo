package compile

import (
	"encoding/json"
	"html"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/syntax"
)

// The parser gives a tree of tags. Lowering decides what the tags mean and
// gives the template as the compiler sees it: markup that goes to the page
// as written, and the places where something is computed.

// role is what a file is used for; some tags make sense only in some roles.
type role int

const (
	component role = iota
	page
	layout
)

// node is one of the types below.
type node any

type (
	// markup goes to the page as written.
	markup struct{ text string }
	// output writes the result of an expression: as text, or as markup for <g-html>.
	output struct {
		kind program.Kind
		expr code
	}
	// attribute is name={expr}: it is left out when the result is null,
	// undefined or false.
	attribute struct {
		kind program.Kind // Attr, or URL for an address
		name string
		expr code
	}
	// spread is {...attributes}.
	spread struct{ expr code }
	// choice is <g-if> with the <g-else-if> and <g-else> after it.
	choice struct{ branches []branch }
	branch struct {
		test *code // nil for <g-else>
		body []node
	}
	// loop is <g-for>.
	loop struct {
		each  code
		item  code
		index *code
		body  []node
	}
	// call shows a component.
	call struct {
		target code // the tag: Card, ui.Button
		props  []prop
		fills  []fill
	}
	// fill is markup the caller puts into a slot of the component.
	fill struct {
		name string
		body []node
	}
	// outlet is <g-slot>: the place where a component shows a fill.
	outlet struct {
		name     string
		fallback []node
	}
	// provide is a fill that a page or a layout gives to the layouts around it.
	provide struct{ fill }
	// mount is a place where the framework adds markup of its own.
	mount struct{ name string }
)

// code is a piece of the project's JavaScript and where it starts in the file.
type code struct {
	text  string
	start int
}

// prop is one parameter given to a component.
type prop struct {
	name    string
	literal string // JavaScript for a value written in the template: "text", true
	expr    *code  // or the expression
	spread  bool   // {...props}: expr holds the object
}

const defaultSlot = "default"

// What Astro has and Shugo writes differently. The usual authors of
// templates know Astro well, so the message says what to write instead.
var notSupported = map[string]string{
	"set:html":    `use <g-html value={…} /> inside the element`,
	"set:text":    `write {…} inside the element`,
	"class:list":  `build the string in an expression: class={[…].filter(Boolean).join(" ")}`,
	"define:vars": `it is not supported: write the values into data- attributes or into a <script type="application/json">`,
	"is:raw":      `it is not supported`,
}

type lowerer struct {
	file   *syntax.File
	role   role
	errs   Errors
	styles []string
	// outlets are the names of the slots the template shows, and provided
	// the names of the slots it fills for the layouts around it, each with
	// the place it does so.
	outlets  []string
	provided map[string]int
}

func (l *lowerer) errorf(offset int, format string, arguments ...any) {
	l.errs = append(l.errs, l.file.Errorf(offset, format, arguments...))
}

// list lowers the nodes of one level. top tells that this is the top level of
// a page or a layout, where slot="name" gives markup to the layouts around.
// nested tells that the level belongs to a control tag standing right inside
// a component, where slot="name" would have no effect.
func (l *lowerer) list(nodes []syntax.Node, top, nested bool) []node {
	var out []node
	for i := 0; i < len(nodes); i++ {
		switch n := nodes[i].(type) {
		case *syntax.Text:
			out = append(out, markup{n.Value})
		case *syntax.Expr:
			out = append(out, output{program.Text, code{n.Code, n.Start}})
		case *syntax.Element:
			if name, ok := slotOf(n); ok && top && l.role != component {
				if _, seen := l.provided[name]; !seen {
					if l.provided == nil {
						l.provided = map[string]int{}
					}
					l.provided[name] = n.Start
				}
				out = append(out, provide{fill{name, l.fillBody(n)}})
				continue
			} else if ok && nested {
				l.errorf(n.Start, `slot="%s" works only on a tag that stands right inside the component, not inside <g-if> or <g-for>`, name)
			}
			if n.Name == "g-if" {
				var chain choice
				chain, i = l.choice(nodes, i, nested)
				out = append(out, chain)
				continue
			}
			out = append(out, l.element(n, nested)...)
		}
	}
	return out
}

func (l *lowerer) element(e *syntax.Element, nested bool) []node {
	switch {
	case e.Name == "g-else-if" || e.Name == "g-else":
		l.errorf(e.Start, "<%s> must come right after a <g-if> or a <g-else-if>", e.Name)
	case e.Name == "g-for":
		return l.loop(e, nested)
	case e.Name == "g-html":
		return l.html(e)
	case e.Name == "g-slot":
		return l.outlet(e)
	case e.Name == "g-fragment":
		l.only(e)
		return l.list(e.Children, false, nested)
	case strings.HasPrefix(e.Name, "g-"):
		l.errorf(e.Start, "there is no <%s>: the tags of Shugo are g-if, g-else-if, g-else, g-for, g-html, g-slot and g-fragment", e.Name)
	case e.Name == "slot":
		l.errorf(e.Start, "use <g-slot> for the place where a component shows what was put inside it")
	case e.Name == "Fragment":
		l.errorf(e.Start, "use <g-fragment> to group markup without a tag of its own")
	case isComponent(e.Name):
		return []node{l.call(e)}
	default:
		return l.html5(e)
	}
	return nil
}

// isComponent tells a component from an element of HTML by its name:
// <Card> and <ui.Button> are components.
func isComponent(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first) || strings.Contains(name, ".")
}

// slotOf returns the name in slot="name".
func slotOf(e *syntax.Element) (string, bool) {
	a, ok := e.Attr("slot")
	if !ok || a.Kind != syntax.Literal {
		return "", false
	}
	return a.Value, true
}

// fillBody lowers a tag that fills a slot: the tag without its slot="name",
// or just what is inside it for <g-fragment>.
func (l *lowerer) fillBody(e *syntax.Element) []node {
	if name, _ := slotOf(e); !validSlot(name) {
		l.errorf(e.Start, "slot=%q: the name of a slot is made of letters, digits, - and _", name)
	}
	if e.Name == "g-fragment" {
		l.only(e, "slot")
		return l.list(e.Children, false, false)
	}
	bare := *e
	bare.Attrs = nil
	for _, a := range e.Attrs {
		if a.Name != "slot" {
			bare.Attrs = append(bare.Attrs, a)
		}
	}
	return l.list([]syntax.Node{&bare}, false, false)
}

func validSlot(name string) bool {
	return name != "" && strings.IndexFunc(name, func(r rune) bool {
		return !(r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0
}

// html5 lowers an element of HTML: its markup, with the attributes and the
// content that are computed.
func (l *lowerer) html5(e *syntax.Element) []node {
	inline := false
	out := []node{markup{"<" + e.Name}}
	for _, a := range e.Attrs {
		switch {
		case a.Name == "is:inline":
			inline = true
		case a.Name == "is:global":
			// Styles are not scoped to their component yet: all of them are global.
		case notSupported[a.Name] != "":
			l.errorf(a.Start, "%s is not a part of Shugo: %s", a.Name, notSupported[a.Name])
		case strings.HasPrefix(a.Name, "client:") || strings.HasPrefix(a.Name, "transition:"):
			l.errorf(a.Start, "%s is not a part of Shugo: pages are static, write a <script> for what happens in the browser", a.Name)
		case a.Kind == syntax.Spread:
			out = append(out, spread{code{a.Value, a.ValueStart}})
		case a.Kind == syntax.Dynamic:
			kind := program.Attr
			if program.IsAddress(a.Name) {
				kind = program.URL
			}
			out = append(out, attribute{kind, a.Name, code{a.Value, a.ValueStart}})
		default:
			out = append(out, markup{" " + a.Raw})
		}
	}
	if e.Name == "style" && !inline {
		// The style sheets of all templates go into one file of the site.
		if len(out) > 1 {
			l.errorf(e.Start, "a <style> with attributes stays where it is only with is:inline")
		}
		for _, child := range e.Children {
			l.styles = append(l.styles, child.(*syntax.Text).Value)
		}
		return nil
	}
	out = append(out, markup{">"})
	if e.Closed && syntax.Void(e.Name) {
		return out
	}
	out = append(out, l.list(e.Children, false, false)...)
	// The framework adds its own markup at the end of the head and of the body.
	if e.Name == "head" || e.Name == "body" {
		out = append(out, mount{e.Name})
	}
	return append(out, markup{"</" + e.Name + ">"})
}

// choice lowers the <g-if> at nodes[at] and the branches after it. It returns
// the position of the last node it took.
func (l *lowerer) choice(nodes []syntax.Node, at int, nested bool) (choice, int) {
	var out choice
	add := func(e *syntax.Element) {
		b := branch{body: l.list(e.Children, false, nested)}
		if e.Name != "g-else" {
			if test, ok := l.expr(e, "cond"); ok {
				b.test = &test
			}
			l.only(e, "cond")
		} else {
			l.only(e)
		}
		out.branches = append(out.branches, b)
	}
	add(nodes[at].(*syntax.Element))
	for last := at; ; {
		// White space between the branches belongs to the way they are written.
		next := last + 1
		if next < len(nodes) {
			if text, ok := nodes[next].(*syntax.Text); ok && strings.TrimSpace(text.Value) == "" {
				next++
			}
		}
		if next >= len(nodes) {
			return out, last
		}
		e, ok := nodes[next].(*syntax.Element)
		if !ok || (e.Name != "g-else-if" && e.Name != "g-else") {
			return out, last
		}
		add(e)
		if last = next; e.Name == "g-else" {
			return out, last
		}
	}
}

func (l *lowerer) loop(e *syntax.Element, nested bool) []node {
	out := loop{body: l.list(e.Children, false, nested)}
	each, ok := l.expr(e, "each")
	out.each = each
	item, found := e.Attr("as")
	switch {
	case !found:
		l.errorf(e.Start, `<g-for> needs as="name": the name of one item, as in <g-for each={items} as="item">`)
		ok = false
	case item.Kind != syntax.Literal || strings.TrimSpace(item.Value) == "":
		l.errorf(item.Start, `as must be a name in quotes: as="item"`)
		ok = false
	}
	out.item = code{item.Value, item.ValueStart}
	if index, found := e.Attr("index"); found {
		if index.Kind != syntax.Literal || !isIdentifier(index.Value) {
			l.errorf(index.Start, `index must be a name in quotes: index="i"`)
			ok = false
		}
		out.index = &code{index.Value, index.ValueStart}
	}
	l.only(e, "each", "as", "index")
	if !ok {
		return nil
	}
	return []node{out}
}

func (l *lowerer) html(e *syntax.Element) []node {
	value, ok := l.expr(e, "value")
	l.only(e, "value")
	if strings.TrimSpace(textOf(e.Children)) != "" || len(e.Children) > 1 {
		l.errorf(e.Start, "<g-html> has nothing inside: write <g-html value={…} />")
	}
	if !ok {
		return nil
	}
	return []node{output{program.HTML, value}}
}

func (l *lowerer) outlet(e *syntax.Element) []node {
	name := defaultSlot
	if a, ok := e.Attr("name"); ok {
		if a.Kind != syntax.Literal || !validSlot(a.Value) {
			l.errorf(a.Start, `the name of a slot is written in quotes and made of letters, digits, - and _: name="aside"`)
		} else {
			name = a.Value
		}
	}
	l.only(e, "name")
	if l.role == page {
		l.errorf(e.Start, "a page is not called by anything, so it has nothing to show in <g-slot>; it works in components and in +layout files")
	}
	if !slices.Contains(l.outlets, name) {
		l.outlets = append(l.outlets, name)
	}
	return []node{outlet{name, l.list(e.Children, false, false)}}
}

func (l *lowerer) call(e *syntax.Element) node {
	out := call{target: code{e.Name, e.Start + 1}}
	for _, a := range e.Attrs {
		switch {
		case strings.HasPrefix(a.Name, "client:"):
			l.errorf(a.Start, "%s is not a part of Shugo: components are shown when the site is built, write a <script> for what happens in the browser", a.Name)
		case notSupported[a.Name] != "":
			l.errorf(a.Start, "%s is not a part of Shugo: %s", a.Name, notSupported[a.Name])
		case a.Kind == syntax.Spread:
			out.props = append(out.props, prop{spread: true, expr: &code{a.Value, a.ValueStart}})
		case a.Kind == syntax.Dynamic:
			out.props = append(out.props, prop{name: a.Name, expr: &code{a.Value, a.ValueStart}})
		case a.Kind == syntax.Literal:
			text, _ := json.Marshal(html.UnescapeString(a.Value))
			out.props = append(out.props, prop{name: a.Name, literal: string(text)})
		default:
			out.props = append(out.props, prop{name: a.Name, literal: "true"})
		}
	}

	// What is inside the tag goes into the slots of the component: a tag with
	// slot="name" into that slot, everything else into the default one.
	var rest []syntax.Node
	named := map[string]int{}
	for _, child := range e.Children {
		element, _ := child.(*syntax.Element)
		name, ok := "", false
		if element != nil {
			name, ok = slotOf(element)
		}
		if !ok {
			rest = append(rest, child)
			continue
		}
		body := l.fillBody(element)
		if at, seen := named[name]; seen {
			out.fills[at].body = append(out.fills[at].body, body...)
			continue
		}
		named[name] = len(out.fills)
		out.fills = append(out.fills, fill{name, body})
	}
	if !blankNodes(rest) {
		body := l.list(rest, false, true)
		if at, seen := named[defaultSlot]; seen {
			out.fills[at].body = append(body, out.fills[at].body...)
		} else {
			out.fills = append([]fill{{defaultSlot, body}}, out.fills...)
		}
	}
	return out
}

// expr returns the expression of an attribute that must be written as name={…}.
func (l *lowerer) expr(e *syntax.Element, name string) (code, bool) {
	a, ok := e.Attr(name)
	switch {
	case !ok:
		l.errorf(e.Start, "<%s> needs %s={…}", e.Name, name)
	case a.Kind != syntax.Dynamic:
		l.errorf(a.Start, "%s must be an expression in braces: %s={…}", name, name)
	default:
		return code{a.Value, a.ValueStart}, true
	}
	return code{}, false
}

// only reports the attributes of a tag of Shugo that it does not have.
func (l *lowerer) only(e *syntax.Element, allowed ...string) {
next:
	for _, a := range e.Attrs {
		for _, name := range allowed {
			if a.Name == name {
				continue next
			}
		}
		if len(allowed) == 0 {
			l.errorf(a.Start, "<%s> has no attributes", e.Name)
		} else {
			l.errorf(a.Start, "<%s> has no attribute %q; it has %s", e.Name, a.Name, strings.Join(allowed, ", "))
		}
	}
}

// blankNodes tells whether nodes hold nothing but white space.
func blankNodes(nodes []syntax.Node) bool {
	for _, n := range nodes {
		if text, ok := n.(*syntax.Text); !ok || strings.TrimSpace(text.Value) != "" {
			return false
		}
	}
	return true
}

func textOf(nodes []syntax.Node) string {
	var out strings.Builder
	for _, n := range nodes {
		if text, ok := n.(*syntax.Text); ok {
			out.WriteString(text.Value)
		}
	}
	return out.String()
}

func isIdentifier(text string) bool {
	if text == "" || (text[0] >= '0' && text[0] <= '9') {
		return false
	}
	return strings.IndexFunc(text, func(r rune) bool {
		return !(r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r))
	}) < 0
}
