// Package syntax reads .shugo files. A file is written the way a component of
// Svelte 5 is: a <script> with the code that runs every time the component is
// shown, a <script module> with code that runs once, a template, and a
// <style>. The package knows the shape of all this — elements, attributes,
// {tags} and {#blocks} — and nothing about what it means.
package syntax

import (
	"fmt"
	"strings"
)

// File is one parsed .shugo file.
type File struct {
	Path   string // as it should appear in messages
	Source string
	Module *Script // <script module>: runs once, when the file is loaded
	Script *Script // <script>: runs every time the component is shown
	Style  *Script // <style>: the styles of the component
	Body   []Node  // the template
	// Warnings are what is written in a way Svelte allows but warns about.
	Warnings []*Error
}

// Script is a <script> or a <style> at the top level of a file.
type Script struct {
	Start int  // where "<" is
	Code  Span // what stands between the opening and the closing tag
	Attrs []Attr
}

// Span is Source[Start:End].
type Span struct{ Start, End int }

// Code is a piece of JavaScript and where it starts in the file.
type Code struct {
	Text  string
	Start int
}

// Node is a part of a template: *Text, *HTMLComment, *Doctype, *Tag, *IfBlock,
// *EachBlock, *AwaitBlock, *KeyBlock, *SnippetBlock or *Element.
type Node interface{ Offset() int }

// Text is markup text, as written: entities are not decoded.
type Text struct {
	Start int
	Raw   string
}

// HTMLComment is <!-- … -->.
type HTMLComment struct {
	Start int
	Data  string
}

// Doctype is <!doctype …>, kept as written.
type Doctype struct {
	Start int
	Raw   string
}

// TagKind tells what a {tag} is.
type TagKind int

const (
	Expression TagKind = iota // {value}
	HTMLTag                   // {@html value}
	ConstTag                  // {@const name = value}
	RenderTag                 // {@render snippet(…)}
	DebugTag                  // {@debug …}
)

// Tag is {…}: an expression, or one of {@html}, {@const}, {@render}, {@debug}.
type Tag struct {
	Start int // where "{" is
	Kind  TagKind
	Code  Code // the expression, or what follows the keyword
}

// IfBlock is {#if} with its {:else if} and {:else} branches.
type IfBlock struct {
	Start    int
	Branches []Branch
}

// Branch is one branch of an {#if}: its test, nil for {:else}, and its body.
type Branch struct {
	Start int
	Test  *Code
	Body  []Node
}

// EachBlock is {#each list as item, index (key)}…{:else}…{/each}.
type EachBlock struct {
	Start int
	List  Code
	Item  *Code // the name or the pattern of one item; nil when there is none
	Index string
	Key   *Code
	Body  []Node
	Else  []Node // the body of {:else}; nil when there is none
}

// AwaitBlock is {#await value}…{:then name}…{:catch name}…{/await}.
type AwaitBlock struct {
	Start     int
	Value     Code
	Pending   []Node
	Then      *Branch // nil when there is no {:then}; Test holds the name
	Catch     *Branch // nil when there is no {:catch}
	ThenFirst bool    // {#await value then name}: there is no pending body
}

// KeyBlock is {#key value}…{/key}.
type KeyBlock struct {
	Start int
	Value Code
	Body  []Node
}

// SnippetBlock is {#snippet name(parameters)}…{/snippet}.
type SnippetBlock struct {
	Start  int
	Name   string
	Params Code // what stands between the parentheses
	Body   []Node
}

// Element is a tag: an element of HTML, a component, or one of svelte:.
type Element struct {
	Start    int // where "<" is
	Name     string
	Attrs    []Attr
	Children []Node
	// SelfClosing tells that the element was written as <name … />.
	SelfClosing bool
}

func (n *Text) Offset() int         { return n.Start }
func (n *HTMLComment) Offset() int  { return n.Start }
func (n *Doctype) Offset() int      { return n.Start }
func (n *Tag) Offset() int          { return n.Start }
func (n *IfBlock) Offset() int      { return n.Start }
func (n *EachBlock) Offset() int    { return n.Start }
func (n *AwaitBlock) Offset() int   { return n.Start }
func (n *KeyBlock) Offset() int     { return n.Start }
func (n *SnippetBlock) Offset() int { return n.Start }
func (n *Element) Offset() int      { return n.Start }

// AttrKind tells what an attribute is.
type AttrKind int

const (
	Plain     AttrKind = iota // name, name="…", name={…}, name="a {b}"
	Shorthand                 // {name}
	Spread                    // {...value}
	Directive                 // class:name, style:name, bind:name, on:name, use:, transition:, in:, out:, animate:, let:
	Attach                    // {@attach …}
)

// Attr is one attribute of a tag.
type Attr struct {
	Start int
	Kind  AttrKind
	// Name is the name of the attribute; for a directive, what follows the colon.
	Name string
	// Directive is the kind of a directive: "class", "style", "bind", "on", …
	Directive string
	// Modifiers follow "|": style:color|important, on:click|once.
	Modifiers []string
	// Value is nil for an attribute written without one, as disabled or
	// class:active. Otherwise it is made of parts: text and expressions.
	Value []Part
	// Quoted tells that the value was written in quotes.
	Quoted bool
	Raw    string // the attribute as written
}

// Part is a piece of the value of an attribute: text as written, or an expression.
type Part struct {
	Start int
	Text  string // raw: entities are not decoded
	Expr  *Code
}

// Expr returns the expression of an attribute whose value is one expression
// and nothing else: name={…}, {name}, {...value}, class:name={…}.
func (a Attr) Expr() (Code, bool) {
	if len(a.Value) == 1 && a.Value[0].Expr != nil {
		return *a.Value[0].Expr, true
	}
	return Code{}, false
}

// Attr returns the plain attribute with the given name.
func (e *Element) Attr(name string) (Attr, bool) {
	for _, a := range e.Attrs {
		if a.Kind == Plain && a.Name == name {
			return a, true
		}
	}
	return Attr{}, false
}

// Error is a mistake in a .shugo file, with the place it was found at.
type Error struct {
	Path    string
	Line    int // from 1
	Column  int // from 1, in characters
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Column, e.Message)
}

// Position turns an offset in the source into a line and a column, both from 1.
func (f *File) Position(offset int) (line, column int) {
	offset = min(max(offset, 0), len(f.Source))
	before := f.Source[:offset]
	start := strings.LastIndexByte(before, '\n') + 1
	return strings.Count(before, "\n") + 1, len([]rune(before[start:])) + 1
}

// Errorf makes an error that points at an offset in the file.
func (f *File) Errorf(offset int, format string, arguments ...any) *Error {
	line, column := f.Position(offset)
	return &Error{Path: f.Path, Line: line, Column: column, Message: fmt.Sprintf(format, arguments...)}
}

// Elements that never have content or a closing tag.
var void = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "command": true, "embed": true, "hr": true, "img": true,
	"input": true, "keygen": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

// Void tells whether an element of HTML never has content or a closing tag.
func Void(name string) bool { return void[strings.ToLower(name)] }

// Elements whose content is not a template: it is kept as written.
var raw = map[string]bool{"script": true, "style": true}

// What the closing tag of an element may be left out before, as in HTML:
// <li> ends where the next <li> starts. Svelte closes such an element there.
// direct elements close it only right inside it; the rest anywhere.
var autoclosing = map[string][]string{
	"li":       {"li"},
	"dt":       {"dt", "dd"},
	"dd":       {"dt", "dd"},
	"p":        {"address", "article", "aside", "blockquote", "div", "dl", "fieldset", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "hr", "main", "menu", "nav", "ol", "p", "pre", "section", "table", "ul"},
	"rt":       {"rt", "rp"},
	"rp":       {"rt", "rp"},
	"optgroup": {"optgroup"},
	"option":   {"option", "optgroup"},
	"thead":    {"tbody", "tfoot"},
	"tbody":    {"tbody", "tfoot"},
	"tfoot":    {"tbody"},
	"tr":       {"tr", "tbody"},
	"td":       {"td", "th", "tr"},
	"th":       {"td", "th", "tr"},
}

// closedBy tells whether an element ends where the given tag starts.
func closedBy(current, next string) bool {
	for _, name := range autoclosing[current] {
		if name == next {
			return true
		}
	}
	return false
}

// The directives an attribute name can start with.
var directives = map[string]bool{
	"class": true, "style": true, "bind": true, "on": true, "use": true,
	"transition": true, "in": true, "out": true, "animate": true, "let": true,
}

// Parse reads a .shugo file.
func Parse(path, source string) (*File, error) {
	f := &File{Path: path, Source: source}
	if strings.HasPrefix(strings.TrimLeft(source, " \t\r\n"), "---") {
		return nil, f.Errorf(strings.Index(source, "---"), `the code of a component goes into <script>…</script>, as in Svelte, not between "---" lines`)
	}
	p := &parser{file: f, src: source}
	body, why, err := p.nodes(nil, "")
	if err != nil {
		return nil, err
	}
	if why == endTag {
		if name := tagName(p.src[p.pos+2:]); Void(name) {
			return nil, p.errorf(p.pos, "</%s>: <%s> never has a closing tag", name, name)
		}
		return nil, p.errorf(p.pos, "<%s closes nothing", strings.SplitN(p.src[p.pos+1:], ">", 2)[0]+">")
	}
	f.Body = body
	return f, nil
}

type parser struct {
	file *File
	src  string
	pos  int
	// open are the elements and blocks the parser is inside of, the
	// outermost first.
	open []string
}

func (p *parser) errorf(offset int, format string, arguments ...any) error {
	return p.file.Errorf(offset, format, arguments...)
}

func (p *parser) warnf(offset int, format string, arguments ...any) {
	p.file.Warnings = append(p.file.Warnings, p.file.Errorf(offset, format, arguments...))
}

// end tells why a fragment stopped.
type end int

const (
	endOfFile   end = iota
	endTag          // a closing tag: the parser stands on "</"
	endBlock        // {:…} or {/…}: the parser stands on "{"
	endImplicit     // an element closed by the start of another tag
)

// nodes reads nodes up to the end of their parent: a closing tag, the next
// branch or the end of a block, or the end of the file.
func (p *parser) nodes(parent *Element, block string) ([]Node, end, error) {
	var out []Node
	text := p.pos
	flush := func() {
		if p.pos > text {
			out = append(out, &Text{Start: text, Raw: p.src[text:p.pos]})
		}
	}
	for p.pos < len(p.src) {
		rest := p.src[p.pos:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			flush()
			close := strings.Index(rest[4:], "-->")
			if close < 0 {
				return nil, 0, p.errorf(p.pos, "a comment is never closed with -->")
			}
			out = append(out, &HTMLComment{Start: p.pos, Data: rest[4 : 4+close]})
			p.pos += 4 + close + 3
			text = p.pos

		case strings.HasPrefix(rest, "<!") && len(rest) > 2 && isTagStart(rest[2]):
			flush()
			close := strings.IndexByte(rest, '>')
			if close < 0 {
				return nil, 0, p.errorf(p.pos, "<! is never closed with >")
			}
			out = append(out, &Doctype{Start: p.pos, Raw: rest[:close+1]})
			p.pos += close + 1
			text = p.pos

		case strings.HasPrefix(rest, "</"):
			flush()
			return out, endTag, nil

		case rest[0] == '<' && len(rest) > 1 && isTagStart(rest[1]):
			flush()
			name := tagName(rest[1:])
			if parent != nil && closedBy(parent.Name, name) {
				// <li>a<li>b: the first item ends where the second starts.
				return out, endImplicit, nil
			}
			node, err := p.element(parent == nil && block == "" && len(p.open) == 0)
			if err != nil {
				return nil, 0, err
			}
			if node != nil {
				out = append(out, node)
			}
			text = p.pos

		case rest[0] == '{':
			flush()
			if len(rest) > 1 && (rest[1] == ':' || rest[1] == '/') {
				if block == "" && parent == nil {
					return nil, 0, p.errorf(p.pos, "%s stands outside of any block", p.braced())
				}
				return out, endBlock, nil
			}
			node, err := p.tag()
			if err != nil {
				return nil, 0, err
			}
			if node != nil {
				out = append(out, node)
			}
			text = p.pos

		default:
			p.pos++
		}
	}
	flush()
	return out, endOfFile, nil
}

// braced returns the {…} the parser stands on, for a message.
func (p *parser) braced() string {
	end := strings.IndexByte(p.src[p.pos:], '}')
	if end < 0 || end > 40 {
		return "{" + strings.SplitN(p.src[p.pos+1:], "\n", 2)[0]
	}
	return p.src[p.pos : p.pos+end+1]
}

// tagName reads the name of a tag at the start of s.
func tagName(s string) string {
	end := 0
	for end < len(s) && !isSpace(s[end]) && !strings.ContainsRune("/>", rune(s[end])) {
		end++
	}
	return s[:end]
}

// element reads an element from its "<". top tells that it stands at the top
// level of the file, where <script> and <style> belong to the component.
func (p *parser) element(top bool) (Node, error) {
	e := &Element{Start: p.pos}
	p.pos++
	e.Name = tagName(p.src[p.pos:])
	if !validTagName(e.Name) {
		return nil, p.errorf(e.Start, "<%s is not a valid name of a tag", e.Name)
	}
	p.pos += len(e.Name)
	for {
		p.space()
		if p.pos >= len(p.src) {
			return nil, p.errorf(e.Start, "the tag <%s is not finished with >", e.Name)
		}
		if strings.HasPrefix(p.src[p.pos:], "/>") {
			p.pos += 2
			e.SelfClosing = true
			break
		}
		if p.src[p.pos] == '>' {
			p.pos++
			break
		}
		attr, err := p.attribute(e)
		if err != nil {
			return nil, err
		}
		for _, other := range e.Attrs {
			if attr.Kind == Plain && other.Kind == Plain && other.Name == attr.Name {
				return nil, p.errorf(attr.Start, "the attribute %s is written twice", attr.Name)
			}
		}
		e.Attrs = append(e.Attrs, attr)
	}

	if top && (e.Name == "script" || e.Name == "style") {
		return p.topLevel(e)
	}
	switch {
	case e.SelfClosing:
		return e, nil
	case Void(e.Name):
		return e, nil
	case raw[e.Name]:
		close := indexFold(p.src[p.pos:], "</"+e.Name)
		if close < 0 {
			return nil, p.errorf(e.Start, "<%s> is never closed", e.Name)
		}
		if close > 0 {
			e.Children = []Node{&Text{Start: p.pos, Raw: p.src[p.pos : p.pos+close]}}
		}
		p.pos += close
		return e, p.closing(e)
	}

	p.open = append(p.open, e.Name)
	defer func() { p.open = p.open[:len(p.open)-1] }()
	children, why, err := p.nodes(e, "")
	if err != nil {
		return nil, err
	}
	e.Children = children
	switch why {
	case endImplicit:
		return e, nil
	case endOfFile:
		return nil, p.errorf(e.Start, "<%s> is never closed", e.Name)
	case endBlock:
		return nil, p.errorf(p.pos, "%s stands inside <%s>, which was opened inside the block and is not closed", p.braced(), e.Name)
	}
	return e, p.closing(e)
}

// closing reads the closing tag of e, or leaves a closing tag of an element
// around it in place: <div><p>text</div> closes the <p> there, as Svelte does.
func (p *parser) closing(e *Element) error {
	start := p.pos
	name := tagName(p.src[p.pos+2:])
	if name == e.Name {
		p.pos += 2 + len(name)
		p.space()
		if p.pos >= len(p.src) || p.src[p.pos] != '>' {
			return p.errorf(start, "the closing tag </%s is not finished with >", name)
		}
		p.pos++
		return nil
	}
	if Void(name) {
		return p.errorf(start, "</%s>: <%s> never has a closing tag", name, name)
	}
	// An element further out has this name: this one ends here.
	for i := len(p.open) - 2; i >= 0; i-- {
		if p.open[i] == name {
			p.warnf(e.Start, "<%s> is closed by </%s>: write </%s> before it", e.Name, name, e.Name)
			return nil
		}
		if strings.HasPrefix(p.open[i], "{") {
			break
		}
	}
	line, column := p.file.Position(e.Start)
	return p.errorf(start, "</%s> does not close <%s> opened at %d:%d", name, e.Name, line, column)
}

// topLevel reads a <script> or a <style> of the component.
func (p *parser) topLevel(e *Element) (Node, error) {
	close := indexFold(p.src[p.pos:], "</"+e.Name)
	if close < 0 {
		return nil, p.errorf(e.Start, "<%s> is never closed", e.Name)
	}
	s := &Script{Start: e.Start, Code: Span{p.pos, p.pos + close}, Attrs: e.Attrs}
	p.pos += close
	if err := p.closing(e); err != nil {
		return nil, err
	}
	switch {
	case e.Name == "style":
		if p.file.Style != nil {
			return nil, p.errorf(e.Start, "a component has one <style>")
		}
		p.file.Style = s
	case isModule(e):
		if p.file.Module != nil {
			return nil, p.errorf(e.Start, "a component has one <script module>")
		}
		p.file.Module = s
	default:
		if p.file.Script != nil {
			return nil, p.errorf(e.Start, "a component has one <script>; markup goes after it, and a script for the browser goes inside an element or <svelte:head>")
		}
		p.file.Script = s
	}
	return nil, nil
}

// isModule tells a <script module> from the script of a component.
func isModule(e *Element) bool {
	for _, a := range e.Attrs {
		if a.Kind != Plain {
			continue
		}
		if a.Name == "module" || (a.Name == "context" && len(a.Value) == 1 && a.Value[0].Text == "module") {
			return true
		}
	}
	return false
}

// attribute reads one attribute of the tag e.
func (p *parser) attribute(e *Element) (Attr, error) {
	a := Attr{Start: p.pos}
	finish := func() (Attr, error) {
		a.Raw = p.src[a.Start:p.pos]
		return a, nil
	}
	if p.src[p.pos] == '{' {
		end, err := p.balanced(p.pos)
		if err != nil {
			return a, err
		}
		inner := p.src[p.pos+1 : end]
		trimmed := strings.TrimSpace(inner)
		start := p.pos + 1 + strings.Index(inner, trimmed)
		p.pos = end + 1
		switch {
		case strings.HasPrefix(trimmed, "..."):
			a.Kind = Spread
			a.Value = []Part{{Start: start, Expr: &Code{trimmed[3:], start + 3}}}
		case strings.HasPrefix(trimmed, "@attach"):
			a.Kind = Attach
		case isName(trimmed):
			a.Kind, a.Name = Shorthand, trimmed
			a.Value = []Part{{Start: start, Expr: &Code{trimmed, start}}}
		default:
			return a, p.errorf(a.Start, "expected {name} or {...attributes} among the attributes of <%s>", e.Name)
		}
		return finish()
	}

	start := p.pos
	for p.pos < len(p.src) && !isSpace(p.src[p.pos]) && !strings.ContainsRune(`=>"'{`, rune(p.src[p.pos])) {
		if p.src[p.pos] == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '>' {
			break
		}
		p.pos++
	}
	a.Name = p.src[start:p.pos]
	if a.Name == "" {
		return a, p.errorf(p.pos, "unexpected %q among the attributes of <%s>", p.src[p.pos], e.Name)
	}
	if kind, rest, found := strings.Cut(a.Name, ":"); found && directives[kind] {
		a.Kind, a.Directive = Directive, kind
		name, modifiers, _ := strings.Cut(rest, "|")
		a.Name = name
		if modifiers != "" {
			a.Modifiers = strings.Split(modifiers, "|")
		}
		if a.Name == "" {
			return a, p.errorf(a.Start, "%s: needs a name after the colon", kind)
		}
	}
	p.space()
	if p.pos >= len(p.src) || p.src[p.pos] != '=' {
		return finish()
	}
	p.pos++
	p.space()
	if p.pos >= len(p.src) {
		return a, p.errorf(a.Start, "the attribute %s has no value", a.Name)
	}
	value, quoted, err := p.value(a.Name)
	if err != nil {
		return a, err
	}
	a.Value, a.Quoted = value, quoted
	return finish()
}

// value reads the value of an attribute: "text {expression} text",
// 'the same', {expression}, or a word without quotes.
func (p *parser) value(name string) ([]Part, bool, error) {
	quote := p.src[p.pos]
	quoted := quote == '"' || quote == '\''
	if quoted {
		p.pos++
	}
	var parts []Part
	text := p.pos
	flush := func() {
		if p.pos > text {
			parts = append(parts, Part{Start: text, Text: p.src[text:p.pos]})
		}
	}
	for {
		if p.pos >= len(p.src) {
			return nil, false, p.errorf(text, "the value of the attribute %s is never closed", name)
		}
		c := p.src[p.pos]
		switch {
		case quoted && c == quote:
			flush()
			p.pos++
			if parts == nil {
				parts = []Part{{Start: p.pos - 1, Text: ""}}
			}
			return parts, true, nil
		case !quoted && (isSpace(c) || c == '>' || (c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '>')):
			flush()
			return parts, false, nil
		case c == '{':
			flush()
			end, err := p.balanced(p.pos)
			if err != nil {
				return nil, false, err
			}
			code := p.src[p.pos+1 : end]
			if blank(code) {
				return nil, false, p.errorf(p.pos, "the attribute %s has an empty {}", name)
			}
			parts = append(parts, Part{Start: p.pos, Expr: &Code{code, p.pos + 1}})
			p.pos = end + 1
			text = p.pos
			if !quoted && len(parts) == 1 {
				return parts, false, nil
			}
		default:
			p.pos++
		}
	}
}

// tag reads {…}: an expression, a {@tag} or the start of a {#block}.
func (p *parser) tag() (Node, error) {
	open := p.pos
	if strings.HasPrefix(p.src[p.pos:], "{#") {
		return p.block()
	}
	from := open + 1
	if strings.HasPrefix(p.src[open:], "{@") {
		from++
	}
	end, err := p.balancedFrom(open, from)
	if err != nil {
		return nil, err
	}
	p.pos = end + 1
	inner := p.src[open+1 : end]
	if strings.HasPrefix(inner, "@") {
		keyword := wordAt(inner[1:])
		code := Code{strings.TrimSpace(inner[1+len(keyword):]), open + 2 + len(keyword)}
		code.Start += strings.Index(inner[1+len(keyword):], code.Text)
		kinds := map[string]TagKind{"html": HTMLTag, "const": ConstTag, "render": RenderTag, "debug": DebugTag}
		kind, ok := kinds[keyword]
		if !ok {
			return nil, p.errorf(open, "there is no {@%s}: the tags are {@html …}, {@const …}, {@render …} and {@debug …}", keyword)
		}
		if code.Text == "" && kind != DebugTag {
			return nil, p.errorf(open, "{@%s} needs something after it", keyword)
		}
		return &Tag{Start: open, Kind: kind, Code: code}, nil
	}
	if blank(inner) {
		return nil, nil
	}
	return &Tag{Start: open, Kind: Expression, Code: Code{inner, open + 1}}, nil
}

// wordAt returns the word at the start of s.
func wordAt(s string) string {
	end := 0
	for end < len(s) && isWordByte(s[end]) {
		end++
	}
	return s[:end]
}

// header reads {#name …}, {:name …} or {/name} and returns what follows the
// name, with its place.
func (p *parser) header() (string, Code, error) {
	open := p.pos
	end, err := p.balancedFrom(open, open+2)
	if err != nil {
		return "", Code{}, err
	}
	p.pos = end + 1
	inner := p.src[open+2 : end]
	keyword := wordAt(inner)
	rest := inner[len(keyword):]
	trimmed := strings.TrimSpace(rest)
	start := open + 2 + len(keyword) + strings.Index(rest, trimmed)
	return keyword, Code{trimmed, start}, nil
}

// block reads a {#block} up to its {/end}.
func (p *parser) block() (Node, error) {
	open := p.pos
	keyword, head, err := p.header()
	if err != nil {
		return nil, err
	}
	p.open = append(p.open, "{#"+keyword)
	defer func() { p.open = p.open[:len(p.open)-1] }()
	switch keyword {
	case "if":
		return p.ifBlock(open, head)
	case "each":
		return p.eachBlock(open, head)
	case "await":
		return p.awaitBlock(open, head)
	case "key":
		if head.Text == "" {
			return nil, p.errorf(open, "{#key} needs a value: {#key value}")
		}
		body, err := p.branchBody("key")
		if err != nil {
			return nil, err
		}
		return &KeyBlock{Start: open, Value: head, Body: body}, p.closeBlock(open, "key")
	case "snippet":
		return p.snippetBlock(open, head)
	}
	return nil, p.errorf(open, "there is no {#%s}: the blocks are {#if}, {#each}, {#await}, {#key} and {#snippet}", keyword)
}

// branchBody reads the nodes of one branch of a block, up to the next {:…}
// or {/…}.
func (p *parser) branchBody(keyword string) ([]Node, error) {
	nodes, why, err := p.nodes(nil, keyword)
	if err != nil {
		return nil, err
	}
	switch why {
	case endOfFile:
		return nil, p.errorf(p.pos, "{#%s} is never closed with {/%s}", keyword, keyword)
	case endTag:
		name := tagName(p.src[p.pos+2:])
		return nil, p.errorf(p.pos, "</%s> closes an element opened outside of {#%s}: close the block first", name, keyword)
	}
	return nodes, nil
}

// next reads the {:…} or {/…} the parser stands on and returns its keyword
// ("else", "then", "catch" or "/if" …) and what follows it.
func (p *parser) next() (string, Code, error) {
	closing := p.src[p.pos+1] == '/'
	keyword, rest, err := p.header()
	if closing {
		keyword = "/" + keyword
	}
	return keyword, rest, err
}

func (p *parser) closeBlock(open int, keyword string) error {
	at := p.pos
	got, rest, err := p.next()
	if err != nil {
		return err
	}
	if got != "/"+keyword || rest.Text != "" {
		return p.errorf(at, "expected {/%s} to close the {#%s} at %s, got %s", keyword, keyword, p.where(open), p.src[at:p.pos])
	}
	return nil
}

func (p *parser) where(offset int) string {
	line, column := p.file.Position(offset)
	return fmt.Sprintf("%d:%d", line, column)
}

func (p *parser) ifBlock(open int, test Code) (Node, error) {
	if test.Text == "" {
		return nil, p.errorf(open, "{#if} needs a condition: {#if condition}")
	}
	b := &IfBlock{Start: open}
	branch := Branch{Start: open, Test: &test}
	for {
		body, err := p.branchBody("if")
		if err != nil {
			return nil, err
		}
		branch.Body = body
		b.Branches = append(b.Branches, branch)
		at := p.pos
		keyword, rest, err := p.next()
		if err != nil {
			return nil, err
		}
		switch {
		case keyword == "/if":
			return b, nil
		case keyword == "else" && branch.Test == nil:
			return nil, p.errorf(at, "{:else} comes after the last branch of {#if}: only {/if} may follow it")
		case keyword == "else" && rest.Text == "":
			branch = Branch{Start: at}
		case keyword == "else" && wordAt(rest.Text) == "if":
			test := Code{strings.TrimSpace(rest.Text[2:]), rest.Start + 2}
			test.Start += strings.Index(rest.Text[2:], test.Text)
			if test.Text == "" {
				return nil, p.errorf(at, "{:else if} needs a condition")
			}
			branch = Branch{Start: at, Test: &test}
		default:
			return nil, p.errorf(at, "%s does not belong in {#if}: it has {:else if …}, {:else} and {/if}", p.src[at:p.pos])
		}
	}
}

func (p *parser) eachBlock(open int, head Code) (Node, error) {
	b := &EachBlock{Start: open}
	if err := p.eachHead(open, head, b); err != nil {
		return nil, err
	}
	body, err := p.branchBody("each")
	if err != nil {
		return nil, err
	}
	b.Body = body
	at := p.pos
	keyword, rest, err := p.next()
	if err != nil {
		return nil, err
	}
	if keyword == "else" && rest.Text == "" {
		b.Else, err = p.branchBody("each")
		if err != nil {
			return nil, err
		}
		if b.Else == nil {
			b.Else = []Node{}
		}
		at = p.pos
		if keyword, rest, err = p.next(); err != nil {
			return nil, err
		}
	}
	if keyword != "/each" || rest.Text != "" {
		return nil, p.errorf(at, "expected {/each} to close the {#each} at %s, got %s", p.where(open), p.src[at:p.pos])
	}
	return b, nil
}

// eachHead reads "list as item, index (key)". The words are found among the
// tokens of the code, so that "as" inside a string or a bracket is left alone.
func (p *parser) eachHead(open int, head Code, b *EachBlock) error {
	if head.Text == "" {
		return p.errorf(open, "{#each} needs a list: {#each items as item}")
	}
	as := -1
	scan := NewScanner(head.Text, 0)
	depth := 0
	for {
		t, err := scan.Next()
		if err != nil || t.Kind == EOF {
			break
		}
		text := scan.Text(t)
		depth += nestingOf(t, text)
		if depth == 0 && t.Kind == Word && text == "as" {
			as = t.Start
			break
		}
	}
	if as < 0 {
		b.List = head
		return nil
	}
	b.List = Code{strings.TrimRight(head.Text[:as], " \t\r\n"), head.Start}
	rest := head.Text[as+2:]
	restStart := head.Start + as + 2
	// The key is the last thing in parentheses.
	trimmed := strings.TrimRight(rest, " \t\r\n")
	if strings.HasSuffix(trimmed, ")") {
		if openAt := matchingOpen(trimmed); openAt > 0 {
			before := strings.TrimRight(trimmed[:openAt], " \t\r\n")
			if before != "" && !strings.HasSuffix(before, ",") {
				key := Code{trimmed[openAt+1 : len(trimmed)-1], restStart + openAt + 1}
				b.Key = &key
				trimmed = before
			}
		}
	}
	// The index follows the last comma that stands outside of brackets.
	comma := -1
	scan = NewScanner(trimmed, 0)
	depth = 0
	for {
		t, err := scan.Next()
		if err != nil || t.Kind == EOF {
			break
		}
		text := scan.Text(t)
		depth += nestingOf(t, text)
		if depth == 0 && text == "," {
			comma = t.Start
		}
	}
	pattern := trimmed
	if comma >= 0 {
		b.Index = strings.TrimSpace(trimmed[comma+1:])
		pattern = trimmed[:comma]
		if !isName(b.Index) {
			return p.errorf(open, "the index of {#each} is a name: {#each items as item, i}")
		}
	}
	item := strings.TrimSpace(pattern)
	if item == "" {
		return p.errorf(open, "{#each} needs a name for one item after \"as\": {#each items as item}")
	}
	b.Item = &Code{item, restStart + strings.Index(rest, item)}
	return nil
}

// matchingOpen returns the place of the "(" that the ")" at the end of s closes.
func matchingOpen(s string) int {
	depth := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

func nestingOf(t Token, text string) int {
	if t.Kind != Punct {
		return 0
	}
	switch text {
	case "(", "[", "{":
		return 1
	case ")", "]", "}":
		return -1
	}
	return 0
}

func (p *parser) awaitBlock(open int, head Code) (Node, error) {
	b := &AwaitBlock{Start: open}
	// {#await value then name} and {#await value catch name} skip the pending body.
	value, word, name := splitWord(head, "then", "catch")
	b.Value = value
	if b.Value.Text == "" {
		return nil, p.errorf(open, "{#await} needs a value: {#await promise}")
	}
	current := &b.Pending
	switch word {
	case "then":
		b.ThenFirst = true
		b.Then = &Branch{Start: open, Test: name}
		current = &b.Then.Body
	case "catch":
		b.ThenFirst = true
		b.Catch = &Branch{Start: open, Test: name}
		current = &b.Catch.Body
	}
	for {
		body, err := p.branchBody("await")
		if err != nil {
			return nil, err
		}
		*current = body
		at := p.pos
		keyword, rest, err := p.next()
		if err != nil {
			return nil, err
		}
		var named *Code
		if rest.Text != "" {
			named = &rest
		}
		switch {
		case keyword == "/await":
			return b, nil
		case keyword == "then" && b.Then == nil && b.Catch == nil:
			b.Then = &Branch{Start: at, Test: named}
			current = &b.Then.Body
		case keyword == "catch" && b.Catch == nil:
			b.Catch = &Branch{Start: at, Test: named}
			current = &b.Catch.Body
		default:
			return nil, p.errorf(at, "%s does not belong here: {#await} has {:then …}, {:catch …} and {/await}", p.src[at:p.pos])
		}
	}
}

// splitWord splits code at the first of the words that stands outside of
// brackets: "promise then value" gives "promise", "then" and "value".
func splitWord(code Code, words ...string) (Code, string, *Code) {
	scan := NewScanner(code.Text, 0)
	depth := 0
	for {
		t, err := scan.Next()
		if err != nil || t.Kind == EOF {
			return code, "", nil
		}
		text := scan.Text(t)
		depth += nestingOf(t, text)
		if depth != 0 || t.Kind != Word {
			continue
		}
		for _, word := range words {
			if text != word {
				continue
			}
			before := Code{strings.TrimRight(code.Text[:t.Start], " \t\r\n"), code.Start}
			after := strings.TrimSpace(code.Text[t.End:])
			if after == "" {
				return before, word, nil
			}
			return before, word, &Code{after, code.Start + t.End + strings.Index(code.Text[t.End:], after)}
		}
	}
}

func (p *parser) snippetBlock(open int, head Code) (Node, error) {
	name := wordAt(head.Text)
	rest := strings.TrimSpace(head.Text[len(name):])
	if name == "" || !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
		return nil, p.errorf(open, "a snippet is written {#snippet name(parameters)}")
	}
	params := rest[1 : len(rest)-1]
	b := &SnippetBlock{Start: open, Name: name, Params: Code{params, head.Start + strings.Index(head.Text, rest) + 1}}
	body, err := p.branchBody("snippet")
	if err != nil {
		return nil, err
	}
	b.Body = body
	return b, p.closeBlock(open, "snippet")
}

// balanced returns the offset of the brace that closes the one at open.
func (p *parser) balanced(open int) (int, error) { return p.balancedFrom(open, open+1) }

// balancedFrom does the same when the code starts at from: after the sign of
// {#block}, {:branch}, {/block} or {@tag}, which is not JavaScript.
func (p *parser) balancedFrom(open, from int) (int, error) {
	end, err := Balanced(p.src, from)
	if scan, ok := err.(*ScanError); ok {
		line, column := p.file.Position(scan.Offset)
		if scan.Offset != open {
			return 0, p.errorf(open, `"{" is never closed (%s at %d:%d)`, scan.Message, line, column)
		}
		return 0, p.errorf(open, "%s", scan.Message)
	}
	return end, err
}

// blank tells whether code holds nothing but spaces and comments.
func blank(code string) bool {
	s := NewScanner(code, 0)
	for {
		t, err := s.Next()
		if err != nil || (t.Kind != Space && t.Kind != Comment && t.Kind != EOF) {
			return false
		}
		if t.Kind == EOF {
			return true
		}
	}
}

func (p *parser) space() {
	for p.pos < len(p.src) && isSpace(p.src[p.pos]) {
		p.pos++
	}
}

func isTagStart(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }

// validTagName tells whether a name can name a tag: an element, a component
// (Card, ui.Button) or svelte:something.
func validTagName(name string) bool {
	if name == "" || !isTagStart(name[0]) {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !isWordByte(c) && c != '-' && c != '.' && c != ':' {
			return false
		}
	}
	return true
}

// isName tells whether text is one JavaScript identifier.
func isName(text string) bool {
	if text == "" || isDigit(text[0]) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if !isWordByte(text[i]) {
			return false
		}
	}
	return true
}

// IsName tells whether text is one JavaScript identifier.
func IsName(text string) bool { return isName(text) }

func indexFold(s, substr string) int {
	return strings.Index(strings.ToLower(s), substr)
}
