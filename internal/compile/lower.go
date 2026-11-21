package compile

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/syntax"
)

// The parser gives a tree of tags and blocks. Lowering decides what they
// mean and gives the template as the compiler sees it: markup that goes to
// the page as written, and the places where something is computed. The rules
// are those of the server renderer of Svelte 5, so that a template gives the
// same HTML here and there.

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
	// output writes the result of an expression: as text, or as markup for {@html}.
	output struct {
		kind program.Kind
		expr code
	}
	// constant is {@const name = value}: code that runs at its place and shows nothing.
	constant struct{ code code }
	// attribute is an attribute whose value is computed: written by the
	// rules of Svelte, and left out where they say so.
	attribute struct {
		kind    program.Kind // Attr, or URL for an address
		name    string
		value   value
		boolean bool
		// class and style, with the directives and the scoping class.
		class, style bool
		directives   []directive
		hash         string
	}
	// spread is every attribute of an element that has {...attributes}:
	// which of them are written is known only when the page is computed.
	spread struct {
		entries  []entry
		classes  []directive
		styles   []directive
		hash     string
		preserve bool // the names keep their case: the element is not HTML
		// chooses names the variable that gets the value of a <select>,
		// which is not written as an attribute.
		chooses string
	}
	// choice is {#if} with its branches, or {#await}.
	choice struct {
		branches []branch
		await    *code // the value of {#await}; nil for {#if}
	}
	branch struct {
		test *code // the condition; for {#await}, the name of the value; nil for {:else}
		body []node
	}
	// loop is {#each}.
	loop struct {
		list  code
		item  *code
		index string
		body  []node
		empty []node // {:else}; nil when there is none
	}
	// call shows a component.
	call struct {
		target   code // the tag: Card, ui.Button, or the expression of svelte:component
		self     bool // <svelte:self>
		optional bool // <svelte:component this={…}> shows nothing for a value that is not a component
		props    []prop
		// given are the snippets without parameters written inside the tag,
		// children among them: they are computed here, where they are
		// written, and the component shows them from the caller's record.
		given []piece
		// snippets are the snippets with parameters written inside the tag:
		// props of the component, computed where it renders them.
		snippets []snipp
	}
	// piece is named markup computed in the template that writes it: a
	// snippet without parameters given to a component, or <svelte:head>.
	piece struct {
		name string
		body []node
	}
	// snipp is {#snippet name(parameters)}: a piece of the template that is
	// shown by {@render}, maybe in another component.
	snipp struct {
		name   string
		params code
		body   []node
	}
	// render is {@render snippet(…)}.
	render struct {
		expr     code // the snippet
		args     code // what stands between the parentheses
		optional bool // snippet?.(): nothing for a missing snippet
	}
	// dynamic is <svelte:element this={tag}>.
	dynamic struct {
		tag      code
		attrs    []node
		children []node
	}
	// provide is markup a page or a layout gives to the <head> of the page:
	// what stands in <svelte:head> under "head", and its <title> under
	// "title".
	provide struct{ piece }
	// outlet is the place in the <head> of a layout or a page where what
	// was provided is shown.
	outlet struct{ name string }
	// mount is a place where the framework adds markup of its own.
	mount struct{ name string }
	// selected says whether an <option> is the chosen one of its <select>.
	selected struct {
		value  value
		within string // the name of the variable that holds the value of the select
	}
	// declare keeps a value in a variable: the value of a <select>.
	declare struct {
		name  string
		value value
	}
	// shown writes a value as text: the value of a <textarea>.
	shown struct{ value value }
)

// code is a piece of the project's JavaScript and where it starts in the file.
type code struct {
	text  string
	start int
}

// value is the value of an attribute: text and expressions in a row.
type value struct {
	parts []valuePart
	bare  bool // written without a value: true
}

type valuePart struct {
	text string // decoded
	expr *code
}

// static tells whether a value is text alone, and gives it.
func (v value) static() (string, bool) {
	if v.bare {
		return "", false
	}
	var out strings.Builder
	for _, p := range v.parts {
		if p.expr != nil {
			return "", false
		}
		out.WriteString(p.text)
	}
	return out.String(), true
}

// single returns the expression of a value that is one expression alone.
func (v value) single() (code, bool) {
	if len(v.parts) == 1 && v.parts[0].expr != nil {
		return *v.parts[0].expr, true
	}
	return code{}, false
}

// entry is one attribute among those of an element with a spread.
type entry struct {
	name   string
	value  value
	spread *code
}

// directive is class:name={…} or style:name={…}.
type directive struct {
	name      string
	value     value
	important bool
}

// prop is one parameter given to a component.
type prop struct {
	name   string
	value  value
	spread *code // {...props}
}

// The tags of Shugo's first syntax and of Astro, with what to write instead.
var oldTags = map[string]string{
	"g-if":       "{#if condition}…{:else if other}…{:else}…{/if}",
	"g-else-if":  "{:else if condition} inside {#if}",
	"g-else":     "{:else} inside {#if} or {#each}",
	"g-for":      "{#each items as item, i}…{/each}",
	"g-html":     "{@html value}",
	"g-slot":     "{@render children?.()}, with children taken from $props()",
	"g-fragment": "{#snippet name()}…{/snippet} inside the component's tag",
	"slot":       "{@render children?.()}, with children taken from $props(): Svelte 5 passes markup as snippets",
	"Fragment":   "{#snippet name()}…{/snippet} inside the component's tag",
}

type lowerer struct {
	file     *syntax.File
	role     role
	errs     Errors
	warnings []string
	// hash is the class that scopes the <style> of the component to its own
	// elements; empty when it has none.
	hash string
	// outlets are where what pages give to layouts is shown: "head".
	outlets []string
	// provided are what the template gives to the layouts around it, each
	// with the place it does so.
	provided map[string]int
	// selects counts the <select> elements met, to name their variables.
	selects int
}

func (l *lowerer) errorf(offset int, format string, arguments ...any) {
	l.errs = append(l.errs, l.file.Errorf(offset, format, arguments...))
}

func (l *lowerer) warnf(offset int, format string, arguments ...any) {
	l.warnings = append(l.warnings, l.file.Errorf(offset, format, arguments...).Error())
}

// context is what lowering needs to know about where a fragment stands.
type context struct {
	parent   string // the element the fragment stands in; "" for others
	top      bool   // the top level of the component
	preserve bool   // inside <pre> or <textarea>: white space stays as written
	svg      bool   // inside <svg>
	raw      bool   // inside <script> or <style>: text is written as it is
	within   string // the variable that holds the value of the <select> around, if any
	head     bool   // inside <svelte:head>
	// ancestors are the elements around, the parent first, as far as the
	// component or the snippet: what cannot stand inside them is checked.
	ancestors []ancestor
}

var (
	startSpaces = regexp.MustCompile(`^[ \t\r\n]+`)
	endSpaces   = regexp.MustCompile(`[ \t\r\n]+$`)
	notSpace    = regexp.MustCompile(`[^ \t\r\n]`)
)

// clean applies the rules of white space of Svelte to the nodes of one
// fragment: comments go, {@const} and snippets are taken out to come first,
// white space at the edges goes and white space between nodes shrinks to one
// space, unless the fragment is inside <pre> or <textarea>.
func (l *lowerer) clean(nodes []syntax.Node, ctx context) (hoisted, regular []syntax.Node) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *syntax.HTMLComment:
			continue
		case *syntax.Tag:
			if n.Kind == syntax.ConstTag || n.Kind == syntax.DebugTag {
				hoisted = append(hoisted, n)
				continue
			}
		case *syntax.SnippetBlock:
			hoisted = append(hoisted, n)
			continue
		case *syntax.Element:
			switch n.Name {
			case "svelte:head", "svelte:window", "svelte:document", "svelte:body", "svelte:options":
				hoisted = append(hoisted, n)
				continue
			}
		case *syntax.Text:
			// Decoded once here: the rules work on the text a reader sees.
			data := n.Raw
			if !ctx.raw {
				data = html.UnescapeString(n.Raw)
			}
			regular = append(regular, &syntax.Text{Start: n.Start, Raw: data})
			continue
		}
		regular = append(regular, n)
	}
	if ctx.preserve || ctx.raw {
		if ctx.parent == "pre" && len(regular) > 0 {
			if first, ok := regular[0].(*syntax.Text); ok && (first.Raw == "\n" || first.Raw == "\r\n") {
				regular = regular[1:]
			}
		}
		return hoisted, regular
	}

	text := func(n syntax.Node) (*syntax.Text, bool) {
		t, ok := n.(*syntax.Text)
		return t, ok
	}
	for len(regular) > 0 {
		if t, ok := text(regular[0]); ok && !notSpace.MatchString(t.Raw) {
			regular = regular[1:]
			continue
		}
		break
	}
	if len(regular) > 0 {
		if t, ok := text(regular[0]); ok {
			t.Raw = startSpaces.ReplaceAllString(t.Raw, "")
		}
	}
	for len(regular) > 0 {
		if t, ok := text(regular[len(regular)-1]); ok && !notSpace.MatchString(t.Raw) {
			regular = regular[:len(regular)-1]
			continue
		}
		break
	}
	if len(regular) > 0 {
		if t, ok := text(regular[len(regular)-1]); ok {
			t.Raw = endSpaces.ReplaceAllString(t.Raw, "")
		}
	}
	removable := ctx.svg && ctx.parent != "text"
	switch ctx.parent {
	case "select", "tr", "table", "tbody", "thead", "tfoot", "colgroup", "datalist":
		removable = true
	}
	isExpression := func(n syntax.Node) bool {
		t, ok := n.(*syntax.Tag)
		return ok && t.Kind == syntax.Expression
	}
	var out []syntax.Node
	for i, n := range regular {
		t, ok := text(n)
		if !ok {
			out = append(out, n)
			continue
		}
		var prev, next syntax.Node
		if i > 0 {
			prev = regular[i-1]
		}
		if i+1 < len(regular) {
			next = regular[i+1]
		}
		if prev == nil || !isExpression(prev) {
			replace := " "
			if pt, ok := text(prev); ok && endSpaces.MatchString(pt.Raw) {
				replace = ""
			}
			t.Raw = startSpaces.ReplaceAllString(t.Raw, replace)
		}
		if next == nil || !isExpression(next) {
			t.Raw = endSpaces.ReplaceAllString(t.Raw, " ")
		}
		if t.Raw != "" && (t.Raw != " " || !removable) {
			out = append(out, t)
		}
	}
	return hoisted, out
}

// fragment lowers the nodes of one level.
func (l *lowerer) fragment(nodes []syntax.Node, ctx context) []node {
	hoisted, regular := l.clean(nodes, ctx)
	var out []node
	for _, n := range hoisted {
		out = append(out, l.hoisted(n, ctx)...)
	}
	for _, n := range regular {
		out = append(out, l.node(n, ctx)...)
	}
	return out
}

func (l *lowerer) hoisted(n syntax.Node, ctx context) []node {
	switch n := n.(type) {
	case *syntax.Tag:
		if n.Kind == syntax.ConstTag {
			return []node{constant{code{n.Code.Text, n.Code.Start}}}
		}
	case *syntax.SnippetBlock:
		return []node{snipp{n.Name, code{n.Params.Text, n.Params.Start}, l.fragment(n.Body, context{svg: ctx.svg})}}
	case *syntax.Element:
		if n.Name != "svelte:head" {
			return nil // window, document, body and options do nothing on a server
		}
		if !ctx.top {
			l.errorf(n.Start, "<svelte:head> stands at the top level of a component, not inside elements or blocks")
			return nil
		}
		if l.role == component {
			l.errorf(n.Start, "<svelte:head> works in pages and in +layout files for now")
			return nil
		}
		l.provides("head", n.Start)
		return []node{provide{piece{"head", l.fragment(n.Children, context{head: true})}}}
	}
	return nil
}

func (l *lowerer) node(n syntax.Node, ctx context) []node {
	switch n := n.(type) {
	case *syntax.Text:
		if ctx.raw {
			return []node{markup{n.Raw}}
		}
		return []node{markup{escapeText(n.Raw)}}
	case *syntax.Doctype:
		return []node{markup{n.Raw}}
	case *syntax.Tag:
		switch n.Kind {
		case syntax.Expression:
			if len(ctx.ancestors) > 0 && !ctx.ancestors[0].blocked {
				if rule, ok := disallowedChildren[ctx.ancestors[0].name]; ok && rule.only != nil {
					l.errorf(n.Start, "text cannot stand right inside <%s>: <%s> only allows <%s>", ctx.ancestors[0].name, ctx.ancestors[0].name, strings.Join(rule.only, ">, <"))
				}
			}
			return []node{output{program.Text, code{n.Code.Text, n.Code.Start}}}
		case syntax.HTMLTag:
			return []node{output{program.HTML, code{n.Code.Text, n.Code.Start}}}
		case syntax.RenderTag:
			return l.render(n)
		}
	case *syntax.IfBlock:
		var out choice
		for _, b := range n.Branches {
			br := branch{body: l.fragment(b.Body, ctx.inner())}
			if b.Test != nil {
				br.test = &code{b.Test.Text, b.Test.Start}
			}
			out.branches = append(out.branches, br)
		}
		return []node{out}
	case *syntax.EachBlock:
		out := loop{list: code{n.List.Text, n.List.Start}, index: n.Index, body: l.fragment(n.Body, ctx.inner())}
		if n.Item != nil {
			out.item = &code{n.Item.Text, n.Item.Start}
		}
		if n.Else != nil {
			out.empty = l.fragment(n.Else, ctx.inner())
			if out.empty == nil {
				out.empty = []node{}
			}
		}
		return []node{out}
	case *syntax.AwaitBlock:
		// On a server the value is not waited for: a promise shows the
		// pending markup, anything else is the value itself.
		out := choice{await: &code{n.Value.Text, n.Value.Start}}
		out.branches = append(out.branches, branch{body: l.fragment(n.Pending, ctx.inner())})
		then := branch{body: []node{}}
		if n.Then != nil {
			then.body = l.fragment(n.Then.Body, ctx.inner())
			if n.Then.Test != nil {
				then.test = &code{n.Then.Test.Text, n.Then.Test.Start}
			}
		}
		out.branches = append(out.branches, then)
		return []node{out}
	case *syntax.KeyBlock:
		return l.fragment(n.Body, ctx.inner())
	case *syntax.Element:
		return l.element(n, ctx)
	}
	return nil
}

// inner is the context of a fragment inside a block: the same element and
// white space, no longer the top of the component, and a block between it
// and the elements around.
func (c context) inner() context {
	c.top = false
	blocked := make([]ancestor, len(c.ancestors))
	for i, a := range c.ancestors {
		blocked[i] = ancestor{a.name, true}
	}
	c.ancestors = blocked
	return c
}

func (l *lowerer) render(n *syntax.Tag) []node {
	expr := strings.TrimSpace(n.Code.Text)
	open := strings.IndexByte(expr, '(')
	if open < 0 || !strings.HasSuffix(expr, ")") {
		l.errorf(n.Start, "{@render} calls a snippet: {@render children()}, {@render row(item)}")
		return nil
	}
	callee := strings.TrimSpace(expr[:open])
	optional := strings.HasSuffix(callee, "?.")
	callee = strings.TrimSuffix(callee, "?.")
	start := n.Code.Start + strings.Index(n.Code.Text, expr)
	return []node{render{expr: code{callee, start}, args: code{expr[open+1 : len(expr)-1], start + open + 1}, optional: optional}}
}

// element lowers a tag: an element of HTML, a component or one of svelte:.
func (l *lowerer) element(e *syntax.Element, ctx context) []node {
	if hint, old := oldTags[e.Name]; old {
		l.errorf(e.Start, "there is no <%s>: write %s", e.Name, hint)
		return nil
	}
	switch e.Name {
	case "svelte:element":
		return l.dynamicElement(e, ctx)
	case "svelte:self":
		// Svelte 5 still shows them, and warns; so does Shugo.
		name := strings.TrimSuffix(l.file.Path[strings.LastIndex(l.file.Path, "/")+1:], ".shugo")
		l.warnf(e.Start, "<svelte:self> is deprecated in Svelte 5: a component imports itself, import %s from \"./%s.shugo\"", name, name)
		return []node{l.call(e, ctx, code{"$$self", e.Start}, true, false)}
	case "svelte:component":
		l.warnf(e.Start, "<svelte:component> is deprecated in Svelte 5: a component in a variable is shown by its tag, {@const Shown = cond ? A : B}<Shown />")
		this, ok := l.this(e)
		if !ok {
			return nil
		}
		return []node{l.call(e, ctx, this, false, true)}
	case "svelte:boundary":
		return l.fragment(e.Children, ctx.inner())
	case "svelte:fragment":
		l.errorf(e.Start, "<svelte:fragment> is a part of the slots of Svelte 4: write {#snippet name()}…{/snippet} inside the component's tag")
		return nil
	}
	if strings.HasPrefix(e.Name, "svelte:") {
		l.errorf(e.Start, "there is no <%s>", e.Name)
		return nil
	}
	if isComponent(e.Name) {
		return []node{l.call(e, ctx, code{e.Name, e.Start + 1}, false, false)}
	}
	return l.html5(e, ctx)
}

// this returns the expression of this={…}.
func (l *lowerer) this(e *syntax.Element) (code, bool) {
	for _, a := range e.Attrs {
		if a.Kind == syntax.Plain && a.Name == "this" {
			if c, ok := a.Expr(); ok {
				return code{c.Text, c.Start}, true
			}
			if text, ok := l.value(a).static(); ok {
				return code{quote(text), a.Start}, true
			}
		}
	}
	l.errorf(e.Start, "<%s> needs this={…}", e.Name)
	return code{}, false
}

// isComponent tells a component from an element of HTML by its name:
// <Card> and <ui.Button> are components.
func isComponent(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first) || strings.Contains(name, ".")
}

// value turns the parts of an attribute into a value, with the text decoded.
func (l *lowerer) value(a syntax.Attr) value {
	if a.Value == nil {
		return value{bare: true}
	}
	var out value
	for _, p := range a.Value {
		if p.Expr != nil {
			out.parts = append(out.parts, valuePart{expr: &code{p.Expr.Text, p.Expr.Start}})
		} else {
			out.parts = append(out.parts, valuePart{text: html.UnescapeString(p.Text)})
		}
	}
	return out
}

// provides notes that the template gives markup to the <head> of the page.
func (l *lowerer) provides(name string, at int) {
	if l.provided == nil {
		l.provided = map[string]int{}
	}
	if _, seen := l.provided[name]; !seen {
		l.provided[name] = at
	}
}

// html5 lowers an element of HTML.
func (l *lowerer) html5(e *syntax.Element, ctx context) []node {
	name := e.Name
	svg := ctx.svg || name == "svg"
	if !svg {
		name = strings.ToLower(name)
	}
	if name == "title" && ctx.head && !svg {
		// A page has one title, as in Svelte: of the titles in <svelte:head>
		// the last in the order of the document, the page's before its
		// layouts', and it stands after the rest of the head.
		l.provides("title", e.Start)
		ctx.head = false
		return []node{provide{piece{"title", l.html5(e, ctx)}}}
	}
	if name == "head" && l.role != component && !slices.Contains(l.outlets, "head") {
		l.outlets = append(l.outlets, "head", "title")
	}
	if !svg {
		l.placed(name, e.Start, ctx.ancestors)
	}
	hash := ""
	if l.hash != "" {
		hash = l.hash
	}
	attrs, content, selectValue, err := l.attributes(e, name, hash, svg, ctx)
	if err {
		return nil
	}
	out := []node{markup{"<" + name}}
	out = append(out, attrs...)
	inner := context{parent: name, preserve: ctx.preserve || name == "pre" || name == "textarea", svg: svg && name != "foreignObject", within: ctx.within,
		ancestors: append([]ancestor{{name: name}}, ctx.ancestors...)}
	if len(attrs) == 1 {
		if sp, ok := attrs[0].(spread); ok && sp.chooses != "" {
			inner.within = sp.chooses
		}
	}
	var prelude []node
	if selectValue != nil {
		l.selects++
		inner.within = "$$select" + strconv.Itoa(l.selects)
		prelude = append(prelude, declare{inner.within, *selectValue})
	}
	if name == "option" && ctx.within != "" {
		if v, ok := optionValue(e, l); ok {
			out = append(out, selected{v, ctx.within})
		}
	}
	if syntax.Void(name) {
		return append(append(prelude, out...), markup{"/>"})
	}
	out = append(out, markup{">"})
	switch {
	case content != nil:
		out = append(out, *content)
	case raw(name):
		inner.raw = true
		out = append(out, l.fragment(e.Children, inner)...)
	default:
		out = append(out, l.fragment(e.Children, inner)...)
	}
	// What pages give the head, then the framework's own markup, end the
	// head; the framework's own markup ends the body.
	if name == "head" && l.role != component {
		out = append(out, outlet{"head"}, outlet{"title"})
	}
	if name == "head" || name == "body" {
		out = append(out, mount{name})
	}
	out = append(out, markup{"</" + name + ">"})
	return append(prelude, out...)
}

func raw(name string) bool { return name == "script" || name == "style" }

// optionValue returns what an <option> is compared with: its value
// attribute, or else its content when that is text and expressions alone.
func optionValue(e *syntax.Element, l *lowerer) (value, bool) {
	for _, a := range e.Attrs {
		if a.Kind == syntax.Plain && a.Name == "value" {
			return l.value(a), true
		}
	}
	var out value
	for _, child := range e.Children {
		switch child := child.(type) {
		case *syntax.Text:
			out.parts = append(out.parts, valuePart{text: html.UnescapeString(child.Raw)})
		case *syntax.Tag:
			if child.Kind != syntax.Expression {
				return value{}, false
			}
			out.parts = append(out.parts, valuePart{expr: &code{child.Code.Text, child.Code.Start}})
		case *syntax.HTMLComment:
		default:
			return value{}, false
		}
	}
	// Svelte compares the HTML of the content, white space cleaned up.
	if len(out.parts) > 0 && out.parts[0].expr == nil {
		out.parts[0].text = strings.TrimLeft(out.parts[0].text, " \t\r\n")
	}
	if last := len(out.parts) - 1; last >= 0 && out.parts[last].expr == nil {
		out.parts[last].text = strings.TrimRight(out.parts[last].text, " \t\r\n")
	}
	return out, true
}

// attributes lowers the attributes of an element of HTML. It returns what
// goes into the opening tag, the content that an attribute gives (the value
// of a <textarea>), and the value of a <select>.
func (l *lowerer) attributes(e *syntax.Element, name, hash string, svg bool, ctx context) ([]node, *node, *value, bool) {
	var (
		plain       []entry
		classes     []directive
		styles      []directive
		hasSpread   bool
		content     *node
		selectValue *value
	)
	for _, a := range e.Attrs {
		switch a.Kind {
		case syntax.Attach:
		case syntax.Spread:
			hasSpread = true
			c, _ := a.Expr()
			plain = append(plain, entry{spread: &code{c.Text, c.Start}})
		case syntax.Directive:
			switch a.Directive {
			case "class":
				v := l.value(a)
				if v.bare {
					v = value{parts: []valuePart{{expr: &code{a.Name, a.Start + len("class:")}}}}
				}
				classes = append(classes, directive{name: a.Name, value: v})
			case "style":
				v := l.value(a)
				if v.bare {
					v = value{parts: []valuePart{{expr: &code{a.Name, a.Start + len("style:")}}}}
				}
				styles = append(styles, directive{name: a.Name, value: v, important: slices.Contains(a.Modifiers, "important")})
			case "bind":
				c, ok := a.Expr()
				if !ok {
					c = syntax.Code{Text: a.Name, Start: a.Start + len("bind:")}
				}
				v := value{parts: []valuePart{{expr: &code{c.Text, c.Start}}}}
				switch {
				case a.Name == "value" && name == "textarea":
					var n node = output{program.Text, code{c.Text, c.Start}}
					content = &n
				case a.Name == "value" && name == "select":
					selectValue = &v
				case a.Name == "this" || a.Name == "group":
				case a.Name == "innerHTML":
					var n node = output{program.HTML, code{c.Text, c.Start}}
					content = &n
				case a.Name == "textContent" || a.Name == "innerText":
					var n node = output{program.Text, code{c.Text, c.Start}}
					content = &n
				default:
					plain = append(plain, entry{name: a.Name, value: v})
				}
			}
			// on:, use:, transition:, in:, out:, animate: and let: do nothing on a server.
		default: // a plain attribute, or {name}
			if isEvent(a.Name) {
				continue // handlers run in the browser
			}
			v := l.value(a)
			switch {
			case a.Name == "value" && name == "textarea":
				var n node = shown{v}
				if text, ok := v.static(); ok {
					n = markup{escapeText(text)}
				}
				content = &n
				continue
			case a.Name == "value" && name == "select":
				selectValue = &v
				continue
			}
			plain = append(plain, entry{name: a.Name, value: v})
		}
	}
	if hasSpread || (name == "input" && hasDefault(plain)) {
		sp := spread{entries: plain, classes: classes, styles: styles, hash: hash, preserve: svg}
		if name == "select" {
			// The value of a <select> may come with the spread: it chooses an
			// option and is not written.
			if selectValue != nil {
				sp.entries = append(sp.entries, entry{name: "value", value: *selectValue})
				selectValue = nil
			}
			l.selects++
			sp.chooses = "$$select" + strconv.Itoa(l.selects)
		}
		return []node{sp}, content, selectValue, false
	}
	var out []node
	classDone, styleDone := false, false
	for _, en := range plain {
		attrName := en.name
		if !svg {
			attrName = strings.ToLower(attrName)
		}
		switch {
		case attrName == "class":
			classDone = true
			out = append(out, l.classAttr(en.value, classes, hash))
			continue
		case attrName == "style":
			styleDone = true
			out = append(out, l.styleAttr(en.value, styles))
			continue
		}
		if en.value.bare {
			out = append(out, markup{" " + attrName + `=""`})
			continue
		}
		if text, ok := en.value.static(); ok {
			out = append(out, markup{" " + attrName + `="` + escapeAttr(text) + `"`})
			continue
		}
		kind := program.Attr
		if program.IsAddress(attrName) {
			kind = program.URL
		}
		out = append(out, attribute{kind: kind, name: attrName, value: en.value, boolean: !svg && booleanAttribute(attrName)})
	}
	if !classDone && (len(classes) > 0 || hash != "") {
		out = append(out, l.classAttr(value{parts: nil}, classes, hash))
	}
	if !styleDone && len(styles) > 0 {
		out = append(out, l.styleAttr(value{parts: nil}, styles))
	}
	return out, content, selectValue, false
}

func hasDefault(entries []entry) bool {
	for _, e := range entries {
		if e.name == "defaultValue" || e.name == "defaultChecked" {
			return true
		}
	}
	return false
}

// classAttr lowers the class of an element with its directives and the
// scoping class of the component.
//
// A class written without a value is empty, as in Svelte; next to
// class:name or the class of the styles it is true, and Svelte writes it as
// "true".
func (l *lowerer) classAttr(v value, classes []directive, hash string) node {
	if v.bare && len(classes) == 0 && hash == "" {
		return markup{` class=""`}
	}
	if text, ok := v.static(); ok && len(classes) == 0 {
		text = strings.TrimSpace(strings.Join(strings.Fields(text), " ") + " " + hash)
		if text == "" {
			return markup{""}
		}
		return markup{` class="` + escapeAttr(text) + `"`}
	}
	return attribute{kind: program.Attr, name: "class", value: v, class: true, directives: classes, hash: hash}
}

// styleAttr lowers the style of an element with its directives.
//
// A style written without a value is empty; next to style:name Svelte
// leaves it out.
func (l *lowerer) styleAttr(v value, styles []directive) node {
	if v.bare {
		if len(styles) == 0 {
			return markup{` style=""`}
		}
		v = value{}
	}
	if text, ok := v.static(); ok && len(styles) == 0 {
		text = strings.Join(strings.Fields(text), " ")
		return markup{` style="` + escapeAttr(text) + `"`}
	}
	return attribute{kind: program.Attr, name: "style", value: v, style: true, directives: styles}
}

// isEvent tells an attribute that handles an event in the browser: onclick.
func isEvent(name string) bool {
	return len(name) > 2 && strings.HasPrefix(strings.ToLower(name), "on")
}

// booleanAttribute tells the attributes of HTML that are there or not: the
// list of Svelte.
func booleanAttribute(name string) bool {
	switch strings.ToLower(name) {
	case "allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "disabled",
		"formnovalidate", "indeterminate", "inert", "ismap", "loop", "multiple", "muted", "nomodule",
		"novalidate", "open", "playsinline", "readonly", "required", "reversed", "seamless", "selected",
		"webkitdirectory", "defer", "disablepictureinpicture", "disableremoteplayback":
		return true
	}
	return false
}

func (l *lowerer) dynamicElement(e *syntax.Element, ctx context) []node {
	this, ok := l.this(e)
	if !ok {
		return nil
	}
	rest := *e
	rest.Attrs = nil
	for _, a := range e.Attrs {
		if !(a.Kind == syntax.Plain && a.Name == "this") {
			rest.Attrs = append(rest.Attrs, a)
		}
	}
	attrs, _, _, failed := l.attributes(&rest, "svelte:element", l.hash, ctx.svg, ctx)
	if failed {
		return nil
	}
	return []node{dynamic{tag: this, attrs: attrs, children: l.fragment(e.Children, context{svg: ctx.svg})}}
}

// call lowers the tag of a component: its parameters, and what is written
// inside it, which becomes its snippets.
func (l *lowerer) call(e *syntax.Element, ctx context, target code, self, optional bool) node {
	out := call{target: target, self: self, optional: optional}
	for _, a := range e.Attrs {
		switch a.Kind {
		case syntax.Spread:
			c, _ := a.Expr()
			out.props = append(out.props, prop{spread: &code{c.Text, c.Start}})
		case syntax.Shorthand, syntax.Plain:
			if (e.Name == "svelte:component" || e.Name == "svelte:element") && a.Name == "this" {
				continue
			}
			if a.Name == "slot" {
				l.errorf(a.Start, `slot="…" is a part of the slots of Svelte 4: write {#snippet %s()}…{/snippet} inside the component's tag`, "name")
				continue
			}
			out.props = append(out.props, prop{name: a.Name, value: l.value(a)})
		case syntax.Directive:
			if a.Directive == "bind" {
				c, ok := a.Expr()
				if !ok {
					c = syntax.Code{Text: a.Name, Start: a.Start + len("bind:")}
				}
				out.props = append(out.props, prop{name: a.Name, value: value{parts: []valuePart{{expr: &code{c.Text, c.Start}}}}})
			}
			// on: and let: do nothing on a server.
		}
	}

	// What is inside the tag: snippets declared there are props of the
	// component, everything else is its children.
	var rest []syntax.Node
	for _, child := range e.Children {
		if el, ok := child.(*syntax.Element); ok {
			if a, found := el.Attr("slot"); found {
				l.errorf(a.Start, `slot="…" is a part of the slots of Svelte 4: write {#snippet name()}…{/snippet} inside the component's tag and {@render name()} in the component`)
			}
		}
		if s, ok := child.(*syntax.SnippetBlock); ok {
			body := l.fragment(s.Body, context{svg: ctx.svg})
			if strings.TrimSpace(s.Params.Text) == "" {
				out.given = append(out.given, piece{s.Name, body})
			} else {
				out.snippets = append(out.snippets, snipp{s.Name, code{s.Params.Text, s.Params.Start}, body})
			}
			continue
		}
		rest = append(rest, child)
	}
	if hoisted, regular := l.clean(rest, context{svg: ctx.svg}); len(regular) > 0 || len(hoisted) > 0 {
		body := l.fragment(rest, context{svg: ctx.svg})
		if len(regular) > 0 {
			out.given = append([]piece{{"children", body}}, out.given...)
		}
	}
	return out
}

// escapeText escapes text between tags the way Svelte does: "&" and "<".
func escapeText(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;").Replace(text)
}

// escapeAttr escapes the value of an attribute, which is always written in
// double quotes: "&", the quote and "<".
func escapeAttr(text string) string {
	return strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;").Replace(text)
}
