// Package program is what the compiler makes of a project for the runtime:
// the markup of every template with the places where values go. It holds no
// expression and no code of the project, so the runtime can show any page
// without running anything the project's author wrote.
package program

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Version 5 adds named request props and function-driven catalog projection.
const Version = 5

const versionStaticSvelte = 2
const versionIslandWorker = 3

// Site is a whole compiled project.
type Site struct {
	Version int `json:"version"`
	// Files are the templates. A block names its template by position here.
	Files  []File  `json:"files"`
	Routes []Route `json:"routes"`
	// Mounts is markup the framework adds to pages by name, such as the
	// links to style sheets under "head".
	Mounts map[string]string `json:"mounts,omitempty"`
	// Origin is the address of the site: "https://example.com".
	Origin string `json:"origin,omitempty"`
	// Redirects sends one path to another.
	Redirects map[string]string `json:"redirects,omitempty"`
	// Assets lists this generation's public compiler outputs. A nil list
	// marks an older program, whose assets folder had only one generation.
	Assets []string `json:"assets"`
	// IslandPrograms holds the precompiled markup of request-bound islands
	// and the component views they call. Keys are source-relative paths.
	IslandPrograms map[string][]IslandOp `json:"islandPrograms,omitempty"`
	// CatalogSearch tells evaluation to prepare the catalog index for search().
	CatalogSearch bool `json:"catalogSearch,omitempty"`
}

// File is one compiled template.
type File struct {
	Path string `json:"path"`
	// Shared tells that the template shows the same on every page when it is
	// called with the same parameters: it reads nothing of the page it is on.
	// Such a block is computed and kept once for the whole site.
	Shared bool `json:"shared,omitempty"`
	// Outlets are the names of what pages give to layouts that the template
	// shows: "head" for a layout with a <head>, where <svelte:head> goes.
	Outlets []string `json:"outlets,omitempty"`
	// Parts are the template itself and then every snippet written in it,
	// and what stands in its <svelte:head>.
	Parts [][]Op `json:"parts"`
}

// Route is one page file and the addresses it gives.
type Route struct {
	File     int       `json:"file"`
	Pattern  string    `json:"pattern"` // "/products/[slug]"
	Segments []Segment `json:"segments"`
	// Layouts are the files wrapped around the page, the outermost first.
	Layouts []int `json:"layouts"`
}

// Segment is one part of an address between slashes: text, a parameter that
// stands for one part, or a parameter that stands for all the parts left.
type Segment struct {
	Text  string `json:"text,omitempty"`
	Param string `json:"param,omitempty"`
	Rest  string `json:"rest,omitempty"`
}

// Kind says what an operation does.
type Kind byte

const (
	Markup  Kind = 'r' // write markup as it is
	Text    Kind = 't' // write the next value as text
	HTML    Kind = 'h' // write the next value as markup
	Attr    Kind = 'a' // write an attribute that may be absent
	URL     Kind = 'u' // the same for an attribute that holds an address
	Spread  Kind = 's' // write attributes known only when the page is computed
	If      Kind = 'i' // go on with one of several branches, or with none
	For     Kind = 'f' // repeat the body for every item, or show the second body for none
	Call    Kind = 'c' // show the next component the block calls
	Render  Kind = 'n' // show a snippet: given by a caller, or a block of its own
	Element Kind = 'e' // write an element whose tag is a value: <svelte:element>
	Mount   Kind = 'm' // write markup the framework adds
	Island  Kind = 'd' // write an island's server markup and hydration payload
)

// Last is the text of a Render that shows the last block alone.
const Last = "last"

// Op is one step of showing a template. Every step except Markup, Call and
// Mount takes the next value the block has computed.
type Op struct {
	Kind Kind
	// Text is the markup of Markup and the name of Attr, URL and Mount.
	// A Render with the text Last shows only the last block of a snippet
	// the caller gave: the <title> of a page, which Svelte takes from the
	// last <title> in <svelte:head>.
	Text string
	// Bodies are the branches of If; the body of For and what it shows for
	// no items; the attributes and the content of Element.
	Bodies [][]Op
}

// On the wire an operation is a list: its kind, then its text if it has one,
// then its bodies: ["r","<p>"], ["t"], ["a","href"], ["i",[…],[…]],
// ["n","last"].
func (o Op) MarshalJSON() ([]byte, error) {
	parts := []any{string(rune(o.Kind))}
	if o.hasText() {
		parts = append(parts, o.Text)
	}
	for _, body := range o.Bodies {
		if body == nil {
			body = []Op{}
		}
		parts = append(parts, body)
	}
	return marshal(parts)
}

// marshal writes JSON without escaping "<", ">" and "&": markup is most of
// what a program holds.
func marshal(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(out.Bytes(), "\n"), nil
}

func (o *Op) UnmarshalJSON(data []byte) error {
	var parts []json.RawMessage
	var kind string
	if err := json.Unmarshal(data, &parts); err != nil || len(parts) == 0 || json.Unmarshal(parts[0], &kind) != nil || len(kind) != 1 {
		return fmt.Errorf("program: damaged operation %.40s", data)
	}
	*o = Op{Kind: Kind(kind[0])}
	parts = parts[1:]
	// The text of a Render is there or not; a body is never a string.
	optional := o.Kind == Render && len(parts) > 0 && len(parts[0]) > 0 && parts[0][0] == '"'
	if o.hasText() || optional {
		if len(parts) == 0 || json.Unmarshal(parts[0], &o.Text) != nil {
			return fmt.Errorf("program: operation %q has no text", kind)
		}
		parts = parts[1:]
	}
	for _, part := range parts {
		var body []Op
		if err := json.Unmarshal(part, &body); err != nil {
			return err
		}
		o.Bodies = append(o.Bodies, body)
	}
	return o.check()
}

// hasText tells whether the operation is written with its text: always for
// the kinds that need one, and for a Render that shows the last block.
func (o Op) hasText() bool {
	switch o.Kind {
	case Markup, Attr, URL, Mount:
		return true
	case Render:
		return o.Text != ""
	}
	return false
}

// check tells whether the operation has the bodies its kind needs.
func (o Op) check() error {
	want := 0
	switch o.Kind {
	case Markup, Text, HTML, Attr, URL, Spread, Call, Render, Mount, Island:
	case For:
		want = min(max(1, len(o.Bodies)), 2)
	case Element:
		want = 2
	case If:
		want = max(1, len(o.Bodies))
	default:
		return fmt.Errorf("program: unknown operation %q", rune(o.Kind))
	}
	if len(o.Bodies) != want {
		return fmt.Errorf("program: operation %q has %d bodies", rune(o.Kind), len(o.Bodies))
	}
	if o.Kind == Render && o.Text != "" && o.Text != Last {
		return fmt.Errorf("program: a render with the text %q", o.Text)
	}
	return nil
}

// Part returns the operations of a block of the given template.
func (s *Site) Part(file, part int) ([]Op, bool) {
	if file < 0 || file >= len(s.Files) || part < 0 || part >= len(s.Files[file].Parts) {
		return nil, false
	}
	return s.Files[file].Parts[part], true
}

// Decode reads a site and checks that this runtime can show it.
func Decode(data []byte) (*Site, error) {
	site := &Site{}
	if err := json.Unmarshal(data, site); err != nil {
		return nil, err
	}
	if site.Version == versionIslandWorker {
		var legacy struct {
			Worker string `json:"islandWorker"`
		}
		if err := json.Unmarshal(data, &legacy); err != nil {
			return nil, err
		}
		if legacy.Worker != "" {
			return nil, fmt.Errorf("program: this site used JavaScript for request-time island rendering; run `shugo build` to compile it for Go")
		}
	}
	if site.Version < versionStaticSvelte || site.Version > Version {
		return nil, fmt.Errorf("program: unsupported format %d, run `shugo build`", site.Version)
	}
	return site, nil
}

// Encode writes a site.
func (s *Site) Encode() ([]byte, error) {
	s.Version = Version
	return marshal(s)
}

// Attributes that hold an address. The runtime replaces an address that
// could run code, such as javascript:alert(1).
var addresses = map[string]bool{
	"href": true, "src": true, "action": true, "formaction": true, "poster": true,
	"cite": true, "background": true, "xlink:href": true,
}

// IsAddress tells whether an attribute holds an address.
func IsAddress(name string) bool { return addresses[name] }

// Elements that never have content or a closing tag. Svelte writes them as <br/>.
var void = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "command": true, "embed": true, "hr": true, "img": true,
	"input": true, "keygen": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

// IsVoid tells whether an element of HTML never has content or a closing tag.
func IsVoid(name string) bool { return void[strings.ToLower(name)] }
