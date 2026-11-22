// Package skeleton is what the compiler makes of a project for the runtime:
// its files, its routes and the shell the pages stand in. It holds no code of
// the project, so the runtime can show any page without running anything the
// project's author wrote: the HTML of the pages is kept when they are
// computed.
package skeleton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Version changes when an older runtime could not read the format any more,
// or this one an older format: 2 is the skeleton of Svelte templates, with
// snippets and <svelte:element>; 3 writes addresses as they are, as Svelte
// does, and has no operation of its own for them; 4 puts the head of a page
// together where the <head> ends, and layouts show no head of their own; 5
// puts every page into the shell of src/site.html; 7 writes Svelte islands
// and keeps the programs of those shown on request, with the searches they
// are bound to; 8 has no templates: Svelte renders the pages, and they are
// kept as HTML. 6 was a format of the experiment with islands.
const Version = 8

// Site is a whole compiled project.
type Site struct {
	Version int `json:"version"`
	// Files are the templates and the modules among the pages, by number.
	Files  []File  `json:"files"`
	Routes []Route `json:"routes"`
	// Shell is the document every page stands in.
	Shell Shell `json:"shell"`
	// Mounts is markup the framework adds to pages by name: under "head" at
	// the end of the head, as the links to style sheets, and under "body" at
	// the end of the body.
	Mounts map[string]string `json:"mounts,omitempty"`
	// Origin is the address of the site: "https://example.com".
	Origin string `json:"origin,omitempty"`
	// Redirects sends one path to another.
	Redirects map[string]string `json:"redirects,omitempty"`
	// Assets lists the public files the compiler made for this build. The
	// assets folder keeps those of earlier builds too, for a server that
	// still shows them; an export copies the files of this list alone.
	Assets []string `json:"assets"`
	// IslandPrograms holds the precompiled markup of request-bound islands
	// and the component views they call. Keys are source-relative paths.
	IslandPrograms map[string][]IslandOp `json:"islandPrograms,omitempty"`
	// Search tells that the project searches tables: the indices of its
	// searches are kept up to date while it is computed.
	Search bool `json:"search,omitempty"`
}

// File is one compiled template, or a module among the pages.
type File struct {
	Path string `json:"path"`
	// Shared tells that the template shows the same on every page when it is
	// given the same props: it reads nothing of the page it is on, and
	// nothing of the place it stands at. Such a block is computed and kept
	// once for the whole site.
	Shared bool `json:"shared,omitempty"`
}

// Route is one file among the pages and the addresses it gives: a page of
// templates, or a module that gives files of another type.
type Route struct {
	File     int       `json:"file"`
	Pattern  string    `json:"pattern"` // "/products/[slug]", "/sitemap.xml"
	Segments []Segment `json:"segments"`
	// Extension ends the addresses of a route whose file is a module: it
	// gives files of that type, ".xml", and not pages of HTML.
	Extension string `json:"extension,omitempty"`
	// Layouts are the files wrapped around the page, the outermost first.
	Layouts []int `json:"layouts"`
}

// FileRoute tells whether a route of the site is that of a module among the
// pages, which gives files that are not HTML.
func (s *Site) FileRoute(route int) bool {
	return route >= 0 && route < len(s.Routes) && s.Routes[route].Extension != ""
}

// Segment is one part of an address between slashes: text, a parameter that
// stands for one part, or a parameter that stands for all the parts left.
type Segment struct {
	Text  string `json:"text,omitempty"`
	Param string `json:"param,omitempty"`
	Rest  string `json:"rest,omitempty"`
}

// Shell is the document every page stands in, src/site.html: its markup
// before the place of the head, %shugo.head%, between it and the place of
// the body, %shugo.body%, and after that.
type Shell struct {
	Before string `json:"before"`
	Middle string `json:"middle"`
	After  string `json:"after"`
}

// The places of a shell.
const (
	HeadPlace = "%shugo.head%"
	BodyPlace = "%shugo.body%"
)

// DefaultShell is the shell of a project without src/site.html.
const DefaultShell = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    %shugo.head%
  </head>
  <body>
    %shugo.body%
  </body>
</html>
`

// ParseShell cuts the text of a shell at its two places.
func ParseShell(text string) (Shell, error) {
	for _, p := range []struct{ place, what string }{{HeadPlace, "what the templates give the <head>"}, {BodyPlace, "the page"}} {
		if n := strings.Count(text, p.place); n != 1 {
			return Shell{}, fmt.Errorf("%s must stand once, where %s goes, and stands %d times", p.place, p.what, n)
		}
	}
	before, rest, _ := strings.Cut(text, HeadPlace)
	middle, after, found := strings.Cut(rest, BodyPlace)
	if !found {
		return Shell{}, fmt.Errorf("%s must stand before %s", HeadPlace, BodyPlace)
	}
	return Shell{Before: before, Middle: middle, After: after}, nil
}

// marshal writes JSON without escaping "<", ">" and "&": a shell is markup.
func marshal(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(out.Bytes(), "\n"), nil
}

// Decode reads a site and checks that this runtime can show it.
func Decode(data []byte) (*Site, error) {
	site := &Site{}
	if err := json.Unmarshal(data, site); err != nil {
		return nil, err
	}
	if site.Version != Version {
		return nil, fmt.Errorf("skeleton: the project was compiled for format %d, this runtime reads format %d: run `shugo build`", site.Version, Version)
	}
	return site, nil
}

// Encode writes a site.
func (s *Site) Encode() ([]byte, error) {
	s.Version = Version
	return marshal(s)
}
