// Package route turns the files under src/pages/ into the addresses of the
// site: a template gives pages of HTML, a module files of another type.
package route

import (
	"fmt"
	"sort"
	"strings"
)

// Kind tells what a segment of an address matches.
type Kind int

const (
	Static Kind = iota // the segment as written: products
	Param              // any one segment: [slug]
	Rest               // all the segments that are left: [...path]
)

// Segment is one part of an address between slashes.
type Segment struct {
	Kind Kind
	Name string // the text of a static segment or the name of a parameter
}

// Extension is what the files of templates end with.
const Extension = ".shugo"

// Layout is the name of the file that wraps the pages of its folder.
const Layout = "+layout" + Extension

// Module tells whether a file is a module of code: among the pages, a
// module gives a file of the site that is not HTML. Declarations of types
// are no modules.
func Module(file string) bool {
	return (strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".js")) && !strings.HasSuffix(file, ".d.ts")
}

// Route is what the path of a file under src/pages/ says of the addresses
// it gives.
type Route struct {
	Segments []Segment
	// Extension ends the addresses of a module, which gives files of the
	// type it names: ".xml" for sitemap.xml.ts. A page of templates gives
	// HTML and has none.
	Extension string
}

// Parse reads the path of a file under src/pages/: a page of templates, such
// as "products/[slug].shugo", or a module, such as "sitemap.xml.ts". It
// returns false for a file that gives no address: one whose name or folder
// starts with "_".
func Parse(file string) (Route, bool, error) {
	var r Route
	module := Module(file)
	name := strings.TrimSuffix(file, Extension)
	if module {
		name = file[:len(file)-len(".ts")]
	}
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "_") {
			return Route{}, false, nil
		}
	}
	if module {
		last := parts[len(parts)-1]
		dot := strings.LastIndexByte(last, '.')
		if dot <= 0 || !word(last[dot+1:]) {
			return Route{}, false, fmt.Errorf("a module among the pages gives a file of the site, and its name tells the type of that file before %s: sitemap.xml%s gives /sitemap.xml; a module of helpers goes to src/lib/, or its name starts with _", file[len(name):], file[len(name):])
		}
		parts[len(parts)-1], r.Extension = last[:dot], last[dot:]
	}
	for i, part := range parts {
		last := i == len(parts)-1
		switch {
		case strings.HasPrefix(part, "+"):
			return Route{}, false, fmt.Errorf("names that start with + are kept for the framework, and %s is not one of them (there is %s)", part, Layout)
		case strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") && !last:
			// A folder in parentheses groups pages without changing their addresses.
		case last && part == "index" && !module:
		case strings.HasPrefix(part, "[...") && strings.HasSuffix(part, "]"):
			if !last {
				return Route{}, false, fmt.Errorf("%s takes the rest of the address, so it must be the last part of the path", part)
			}
			r.Segments = append(r.Segments, Segment{Rest, part[len("[...") : len(part)-1]})
		case strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]"):
			r.Segments = append(r.Segments, Segment{Param, part[1 : len(part)-1]})
		case strings.ContainsAny(part, "[]"):
			return Route{}, false, fmt.Errorf("%s mixes text and a parameter; a part of the path is either text or one [parameter]", part)
		default:
			r.Segments = append(r.Segments, Segment{Static, part})
		}
	}
	seen := map[string]bool{}
	for _, s := range r.Segments {
		if s.Kind == Static {
			continue
		}
		if !identifier(s.Name) {
			return Route{}, false, fmt.Errorf("[%s] is not a name of a parameter: use letters, digits and _", s.Name)
		}
		if seen[s.Name] {
			return Route{}, false, fmt.Errorf("the parameter [%s] is there twice", s.Name)
		}
		seen[s.Name] = true
	}
	return r, true, nil
}

// Pattern writes the route the way it stands in file names:
// /products/[slug], /sitemap.xml.
func (r Route) Pattern() string {
	if len(r.Segments) == 0 {
		return "/" + r.Extension
	}
	var out strings.Builder
	for _, s := range r.Segments {
		out.WriteByte('/')
		switch s.Kind {
		case Param:
			out.WriteString("[" + s.Name + "]")
		case Rest:
			out.WriteString("[..." + s.Name + "]")
		default:
			out.WriteString(s.Name)
		}
	}
	out.WriteString(r.Extension)
	return out.String()
}

// Shape is the pattern without the names of parameters. Two files of one
// shape give the same addresses.
func (r Route) Shape() string {
	shape := Route{Segments: make([]Segment, len(r.Segments)), Extension: r.Extension}
	for i, s := range r.Segments {
		if shape.Segments[i] = s; s.Kind != Static {
			shape.Segments[i].Name = ""
		}
	}
	return shape.Pattern()
}

// Less tells which of two routes wins when both give one address: the one
// that names more of it. At the first place where they differ the end of the
// address beats text, text beats a parameter, and a parameter beats the rest;
// where they do not, the type of a file at the end names more.
func Less(a, b Route) bool {
	kind := func(segments []Segment, i int) Kind {
		if i >= len(segments) {
			return Static - 1
		}
		return segments[i].Kind
	}
	for i := 0; i < len(a.Segments) || i < len(b.Segments); i++ {
		if kind(a.Segments, i) != kind(b.Segments, i) {
			return kind(a.Segments, i) < kind(b.Segments, i)
		}
	}
	if (a.Extension == "") != (b.Extension == "") {
		return a.Extension != ""
	}
	return a.Pattern() < b.Pattern()
}

// Sort puts routes in the order of Less.
func Sort[T any](routes []T, route func(T) Route) {
	sort.SliceStable(routes, func(i, j int) bool { return Less(route(routes[i]), route(routes[j])) })
}

// word tells whether a name is made of letters and digits alone.
func word(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range []byte(name) {
		if !(c >= '0' && c <= '9' || c|0x20 >= 'a' && c|0x20 <= 'z') {
			return false
		}
	}
	return true
}

func identifier(name string) bool {
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		return false
	}
	for _, c := range []byte(name) {
		if !(c == '_' || c >= '0' && c <= '9' || c|0x20 >= 'a' && c|0x20 <= 'z') {
			return false
		}
	}
	return true
}
