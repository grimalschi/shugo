// Package route turns the files under src/pages/ into the addresses of the site.
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

// What stands in Shugo for the route files of SvelteKit, which an agent
// used to SvelteKit writes first.
var sveltekit = map[string]string{
	"+page":   "in Shugo a page is a file named after its address, src/pages/about.shugo for /about and src/pages/blog/[slug].shugo for /blog/…",
	"+error":  "the page for addresses that have no page is src/pages/404.shugo",
	"+server": "Shugo has no endpoints; a page gives HTML",
}

// Parse reads the path of a page file under src/pages/, such as
// "products/[slug].shugo". It returns no segments and false for a file that
// gives no address: one whose name or folder starts with "_".
func Parse(file string) ([]Segment, bool, error) {
	var out []Segment
	parts := strings.Split(strings.TrimSuffix(file, Extension), "/")
	for i, part := range parts {
		last := i == len(parts)-1
		switch {
		case strings.HasPrefix(part, "_"):
			return nil, false, nil
		case strings.HasPrefix(part, "+"):
			if hint, known := sveltekit[part]; known {
				return nil, false, fmt.Errorf("%s is a file of SvelteKit: %s", part, hint)
			}
			return nil, false, fmt.Errorf("names that start with + are kept for the framework, and %s is not one of them (there is %s)", part, Layout)
		case strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") && !last:
			// A folder in parentheses groups pages without changing their addresses.
		case last && part == "index":
		case strings.HasPrefix(part, "[...") && strings.HasSuffix(part, "]"):
			if !last {
				return nil, false, fmt.Errorf("%s takes the rest of the address, so it must be the last part of the path", part)
			}
			out = append(out, Segment{Rest, part[len("[...") : len(part)-1]})
		case strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]"):
			out = append(out, Segment{Param, part[1 : len(part)-1]})
		case strings.ContainsAny(part, "[]"):
			return nil, false, fmt.Errorf("%s mixes text and a parameter; a part of the path is either text or one [parameter]", part)
		default:
			out = append(out, Segment{Static, part})
		}
	}
	seen := map[string]bool{}
	for _, s := range out {
		if s.Kind == Static {
			continue
		}
		if !identifier(s.Name) {
			return nil, false, fmt.Errorf("[%s] is not a name of a parameter: use letters, digits and _", s.Name)
		}
		if seen[s.Name] {
			return nil, false, fmt.Errorf("the parameter [%s] is there twice", s.Name)
		}
		seen[s.Name] = true
	}
	return out, true, nil
}

// Pattern writes segments the way they stand in file names: /products/[slug].
func Pattern(segments []Segment) string {
	if len(segments) == 0 {
		return "/"
	}
	var out strings.Builder
	for _, s := range segments {
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
	return out.String()
}

// Shape is the pattern without the names of parameters. Two files of one
// shape give the same addresses.
func Shape(segments []Segment) string {
	shape := make([]Segment, len(segments))
	for i, s := range segments {
		if shape[i] = s; s.Kind != Static {
			shape[i].Name = ""
		}
	}
	return Pattern(shape)
}

// Less tells which of two routes wins when both give one address: the one
// that names more of it. At the first place where they differ the end of the
// address beats text, text beats a parameter, and a parameter beats the rest.
func Less(a, b []Segment) bool {
	kind := func(segments []Segment, i int) Kind {
		if i >= len(segments) {
			return Static - 1
		}
		return segments[i].Kind
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		if kind(a, i) != kind(b, i) {
			return kind(a, i) < kind(b, i)
		}
	}
	return Pattern(a) < Pattern(b)
}

// Sort puts routes in the order of Less.
func Sort[T any](routes []T, segments func(T) []Segment) {
	sort.SliceStable(routes, func(i, j int) bool { return Less(segments(routes[i]), segments(routes[j])) })
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
