// Package syntax represents Svelte templates as Shugo compiler input.
package syntax

import (
	"fmt"
	"strings"
)

// File is one parsed .svelte file.
type File struct {
	Path   string // as it should appear in messages
	Source string
	Module *Script  // <script module>: runs once, when the file is loaded
	Script *Script  // <script>: runs every time the component is shown
	Style  *Script  // <style>: the styles of the component
	Body   Fragment // the template
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
	Call  *RenderCall
}

// RenderCall is the invocation in a {@render} tag.
type RenderCall struct {
	Callee    Code
	Arguments []Code
	Optional  bool
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
	Body  Fragment
}

// EachBlock is {#each list as item, index (key)}…{:else}…{/each}.
type EachBlock struct {
	Start int
	List  Code
	Item  *Code // the name or the pattern of one item; nil when there is none
	Index string
	Key   *Code
	Body  Fragment
	Else  Fragment // the body of {:else}; nil when there is none
}

// AwaitBlock is {#await value}…{:then name}…{:catch name}…{/await}.
type AwaitBlock struct {
	Start     int
	Value     Code
	Pending   Fragment
	Then      *Branch // nil when there is no {:then}; Test holds the name
	Catch     *Branch // nil when there is no {:catch}
	ThenFirst bool    // {#await value then name}: there is no pending body
}

// KeyBlock is {#key value}…{/key}.
type KeyBlock struct {
	Start int
	Value Code
	Body  Fragment
}

// SnippetBlock is {#snippet name(parameters)}…{/snippet}.
type SnippetBlock struct {
	Start  int
	Name   string
	Params Code // what stands between the parentheses
	Body   Fragment
}

// Element is a tag: an element of HTML, a component, or one of svelte:.
type Element struct {
	Start    int // where "<" is
	Name     string
	Attrs    []Attr
	Children Fragment
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

// Error is a mistake in a .svelte file, with the place it was found at.
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
