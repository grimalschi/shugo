// Package syntax reads .shugo files: a script between two "---" lines and a
// template after it. It knows the shape of the template — elements,
// attributes, {expressions} — and nothing about what the tags mean.
package syntax

import (
	"fmt"
	"regexp"
	"strings"
)

// File is one parsed .shugo file.
type File struct {
	Path   string // as it should appear in messages
	Source string
	Script Span   // the code between the "---" lines; empty when there is none
	Body   []Node // the template
}

// Span is Source[Start:End].
type Span struct{ Start, End int }

// Node is a part of a template: *Text, *Expr or *Element.
type Node interface{ Offset() int }

// Text is markup that goes to the page as written.
type Text struct {
	Start int
	Value string
}

// Expr is {code}.
type Expr struct {
	Start int // where Code starts in the source
	Code  string
}

// Element is a tag and what is inside it.
type Element struct {
	Start    int // where "<" is
	Name     string
	Attrs    []Attr
	Children []Node
	// Closed tells that the element was written without a closing tag: it is
	// a void element such as <br>, or it ends with "/>".
	Closed bool
}

// AttrKind tells how an attribute got its value.
type AttrKind int

const (
	Bare    AttrKind = iota // disabled
	Literal                 // class="card"
	Dynamic                 // class={name}, or {name} for name={name}
	Spread                  // {...attrs}
)

// Attr is one attribute of a tag.
type Attr struct {
	Start      int
	Name       string // empty for a spread
	Kind       AttrKind
	Value      string // a literal without its quotes, or the code
	ValueStart int
	Raw        string // the attribute as written
}

func (n *Text) Offset() int    { return n.Start }
func (n *Expr) Offset() int    { return n.Start }
func (n *Element) Offset() int { return n.Start }

// Attr returns the attribute with the given name.
func (e *Element) Attr(name string) (Attr, bool) {
	for _, a := range e.Attrs {
		if a.Name == name {
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
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
	"input": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

// Elements whose content is not markup: it is copied as written.
var raw = map[string]bool{"script": true, "style": true}

const fence = "---"

// Parse reads a .shugo file.
func Parse(path, source string) (*File, error) {
	f := &File{Path: path, Source: source}
	p := &parser{file: f, src: source}
	if err := p.script(); err != nil {
		return nil, err
	}
	body, err := p.nodes(nil)
	if err != nil {
		return nil, err
	}
	f.Body = trim(body)
	return f, nil
}

type parser struct {
	file *File
	src  string
	pos  int
}

func (p *parser) errorf(offset int, format string, arguments ...any) error {
	return p.file.Errorf(offset, format, arguments...)
}

// script finds the code between the "---" lines at the top of the file.
func (p *parser) script() error {
	first, _, _ := strings.Cut(p.src, "\n")
	if strings.TrimRight(first, " \t\r") != fence {
		return nil
	}
	start := len(first) + 1
	for at := start; at <= len(p.src); {
		line, _, found := strings.Cut(p.src[at:], "\n")
		if strings.TrimRight(line, " \t\r") == fence {
			p.file.Script = Span{start, at}
			p.pos = min(at+len(line)+1, len(p.src))
			return nil
		}
		if !found {
			break
		}
		at += len(line) + 1
	}
	return p.errorf(0, `the script that starts with "---" has no closing "---" line`)
}

// nodes reads markup up to the closing tag of parent, or to the end of the
// file when parent is nil.
func (p *parser) nodes(parent *Element) ([]Node, error) {
	var out []Node
	text := p.pos
	flush := func() {
		if p.pos > text {
			out = append(out, &Text{Start: text, Value: p.src[text:p.pos]})
		}
	}
	for p.pos < len(p.src) {
		rest := p.src[p.pos:]
		switch {
		case rest[0] == '{':
			flush()
			expr, err := p.expression()
			if err != nil {
				return nil, err
			}
			if expr != nil {
				out = append(out, expr)
			}
			text = p.pos

		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest, "-->")
			if end < 0 {
				return nil, p.errorf(p.pos, "comment is never closed with -->")
			}
			p.pos += end + len("-->")

		case strings.HasPrefix(rest, "</"):
			flush()
			start := p.pos
			p.pos += 2
			name := p.name()
			p.space()
			if p.pos >= len(p.src) || p.src[p.pos] != '>' {
				return nil, p.errorf(start, "closing tag </%s is not finished with >", name)
			}
			p.pos++
			if parent == nil {
				return nil, p.errorf(start, "</%s> closes nothing", name)
			}
			if name != parent.Name {
				line, column := p.file.Position(parent.Start)
				return nil, p.errorf(start, "</%s> does not close <%s> opened at %d:%d", name, parent.Name, line, column)
			}
			return out, nil

		case rest[0] == '<' && len(rest) > 1 && isTagStart(rest[1]):
			flush()
			element, err := p.element()
			if err != nil {
				return nil, err
			}
			out = append(out, element)
			text = p.pos

		default:
			// Text, a doctype, or a "<" that starts nothing.
			p.pos++
		}
	}
	flush()
	if parent != nil {
		return nil, p.errorf(parent.Start, "<%s> is never closed", parent.Name)
	}
	return out, nil
}

// expression reads {code}. A {} that holds only comments gives nothing.
func (p *parser) expression() (*Expr, error) {
	open := p.pos
	end, err := p.balanced(open)
	if err != nil {
		return nil, err
	}
	p.pos = end + 1
	code := p.src[open+1 : end]
	if blank(code) {
		return nil, nil
	}
	return &Expr{Start: open + 1, Code: code}, nil
}

// balanced returns the offset of the brace that closes the one at open. When
// there is none the scanner has run into the markup after the expression, so
// the error points at the brace and only mentions where the scanner gave up.
func (p *parser) balanced(open int) (int, error) {
	end, err := Balanced(p.src, open+1)
	if scan, ok := err.(*ScanError); ok {
		line, column := p.file.Position(scan.Offset)
		rest, _, _ := strings.Cut(p.src[open:], "\n")
		switch {
		case tagAfterOperator.MatchString(rest):
			return 0, p.errorf(open, "%s", noMarkup)
		case scan.Offset != open:
			return 0, p.errorf(open, `"{" is never closed (%s at %d:%d)`, scan.Message, line, column)
		}
		return 0, p.errorf(open, "%s", scan.Message)
	}
	if code := p.src[open+1 : end]; err == nil && tagAfterOperator.MatchString(code) && tagEnd.MatchString(code) {
		return 0, p.errorf(open, "%s", noMarkup)
	}
	return end, err
}

// Authors who know JSX write markup inside expressions: {items.map(item =>
// <li>{item}</li>)}. Here an expression is plain JavaScript, and the message
// says what to write instead. A tag is told from a comparison by what stands
// before it, and in code that the scanner could read, by the end of a tag.
var (
	tagAfterOperator = regexp.MustCompile(`(=>|&&|\|\||[?:(,]|\breturn)\s*<[A-Za-z]`)
	tagEnd           = regexp.MustCompile(`</[A-Za-z]|/>`)
)

const noMarkup = `markup cannot stand inside {…}: an expression is plain JavaScript. ` +
	`Write a list as <g-for each={items} as="item">…</g-for> and a condition as <g-if cond={…}>…</g-if>`

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

func (p *parser) element() (*Element, error) {
	e := &Element{Start: p.pos}
	p.pos++
	e.Name = p.name()
	for {
		p.space()
		if p.pos >= len(p.src) {
			return nil, p.errorf(e.Start, "tag <%s is not finished with >", e.Name)
		}
		if strings.HasPrefix(p.src[p.pos:], "/>") {
			p.pos += 2
			e.Closed = true
			return e, nil
		}
		if p.src[p.pos] == '>' {
			p.pos++
			break
		}
		attr, err := p.attribute(e)
		if err != nil {
			return nil, err
		}
		e.Attrs = append(e.Attrs, attr)
	}
	// Only the names of HTML are special: <Link> and <Script> are components.
	switch {
	case void[e.Name]:
		e.Closed = true
	case raw[e.Name]:
		end := indexFold(p.src[p.pos:], "</"+e.Name)
		if end < 0 {
			return nil, p.errorf(e.Start, "<%s> is never closed", e.Name)
		}
		if end > 0 {
			e.Children = []Node{&Text{Start: p.pos, Value: p.src[p.pos : p.pos+end]}}
		}
		p.pos += end + len("</") + len(e.Name)
		p.space()
		if p.pos >= len(p.src) || p.src[p.pos] != '>' {
			return nil, p.errorf(p.pos, "closing tag </%s is not finished with >", e.Name)
		}
		p.pos++
	default:
		children, err := p.nodes(e)
		if err != nil {
			return nil, err
		}
		e.Children = children
	}
	return e, nil
}

func (p *parser) attribute(e *Element) (Attr, error) {
	a := Attr{Start: p.pos}
	finish := func() (Attr, error) {
		a.Raw = p.src[a.Start:p.pos]
		return a, nil
	}
	if p.src[p.pos] == '{' {
		// {...attrs} or {name}
		end, err := p.balanced(p.pos)
		if err != nil {
			return a, err
		}
		code := p.src[p.pos+1 : end]
		a.ValueStart = p.pos + 1 + len(code) - len(strings.TrimLeft(code, " \t\r\n"))
		a.Value = strings.TrimSpace(code)
		p.pos = end + 1
		if rest, spread := strings.CutPrefix(a.Value, "..."); spread {
			a.Kind = Spread
			a.ValueStart += len("...")
			a.Value = rest
			return finish()
		}
		if !isName(a.Value) {
			return a, p.errorf(a.Start, "expected {name} or {...attributes} among the attributes of <%s>", e.Name)
		}
		a.Kind, a.Name = Dynamic, a.Value
		return finish()
	}

	start := p.pos
	for p.pos < len(p.src) && !isSpace(p.src[p.pos]) && !strings.ContainsRune(`=>/"'{<`, rune(p.src[p.pos])) {
		p.pos++
	}
	a.Name = p.src[start:p.pos]
	if a.Name == "" {
		return a, p.errorf(p.pos, "unexpected %q among the attributes of <%s>", p.src[p.pos], e.Name)
	}
	if p.pos >= len(p.src) || p.src[p.pos] != '=' {
		return finish()
	}
	p.pos++
	if p.pos >= len(p.src) {
		return a, p.errorf(a.Start, "attribute %s has no value", a.Name)
	}
	a.ValueStart = p.pos
	switch c := p.src[p.pos]; {
	case c == '"' || c == '\'':
		end := strings.IndexByte(p.src[p.pos+1:], c)
		if end < 0 {
			return a, p.errorf(p.pos, "the value of attribute %s is never closed with %c", a.Name, c)
		}
		a.Kind, a.ValueStart = Literal, p.pos+1
		a.Value = p.src[p.pos+1 : p.pos+1+end]
		p.pos += end + 2
	case c == '{':
		end, err := p.balanced(p.pos)
		if err != nil {
			return a, err
		}
		a.Kind, a.ValueStart = Dynamic, p.pos+1
		a.Value = p.src[p.pos+1 : end]
		p.pos = end + 1
		if blank(a.Value) {
			return a, p.errorf(a.Start, "attribute %s has an empty {}", a.Name)
		}
	default:
		for p.pos < len(p.src) && !isSpace(p.src[p.pos]) && p.src[p.pos] != '>' {
			p.pos++
		}
		a.Kind = Literal
		a.Value = p.src[a.ValueStart:p.pos]
		if a.Value == "" {
			return a, p.errorf(a.Start, "attribute %s has no value", a.Name)
		}
	}
	return finish()
}

func (p *parser) space() {
	for p.pos < len(p.src) && isSpace(p.src[p.pos]) {
		p.pos++
	}
}

// name reads the name of a tag: letters, digits and "-", "_", ".", ":".
func (p *parser) name() string {
	start := p.pos
	for p.pos < len(p.src) && (isWordByte(p.src[p.pos]) || strings.IndexByte("-.:", p.src[p.pos]) >= 0) {
		p.pos++
	}
	return p.src[start:p.pos]
}

func isTagStart(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }

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

func indexFold(s, substr string) int {
	return strings.Index(strings.ToLower(s), substr)
}

// trim drops the white space around the whole template: a component gives
// its markup and no stray line breaks around it.
func trim(nodes []Node) []Node {
	if len(nodes) > 0 {
		if first, ok := nodes[0].(*Text); ok {
			cut := strings.TrimLeft(first.Value, " \t\r\n")
			first.Start += len(first.Value) - len(cut)
			first.Value = cut
			if cut == "" {
				nodes = nodes[1:]
			}
		}
	}
	if len(nodes) > 0 {
		if last, ok := nodes[len(nodes)-1].(*Text); ok {
			last.Value = strings.TrimRight(last.Value, " \t\r\n")
			if last.Value == "" {
				nodes = nodes[:len(nodes)-1]
			}
		}
	}
	return nodes
}

// Void tells whether an element of HTML never has content or a closing tag.
func Void(name string) bool { return void[name] }
