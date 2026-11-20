package route

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ file, pattern string }{
		{"index.shugo", "/"},
		{"about.shugo", "/about"},
		{"blog/index.shugo", "/blog"},
		{"blog/[slug].shugo", "/blog/[slug]"},
		{"docs/[...path].shugo", "/docs/[...path]"},
		{"(shop)/cart.shugo", "/cart"},
		{"(shop)/products/[id]/reviews.shugo", "/products/[id]/reviews"},
		{"404.shugo", "/404"},
		{"café/index.shugo", "/café"},
	}
	for _, c := range cases {
		segments, ok, err := Parse(c.file)
		if err != nil || !ok {
			t.Errorf("%s: %v %v", c.file, ok, err)
			continue
		}
		if got := Pattern(segments); got != c.pattern {
			t.Errorf("%s: got %s, want %s", c.file, got, c.pattern)
		}
	}
}

func TestParseSkipsAndErrors(t *testing.T) {
	for _, file := range []string{"_parts/card.shugo", "blog/_draft.shugo"} {
		if _, ok, err := Parse(file); ok || err != nil {
			t.Errorf("%s: %v %v", file, ok, err)
		}
	}
	errors := map[string]string{
		"blog/+page.shugo": "kept for the framework",
		"[...all]/x.shugo": "must be the last part",
		"post-[id].shugo":  "mixes text and a parameter",
		"[a]/[a].shugo":    "is there twice",
		"[my-id].shugo":    "is not a name of a parameter",
	}
	for file, want := range errors {
		if _, _, err := Parse(file); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", file, err, want)
		}
	}
}

func TestOrder(t *testing.T) {
	files := []string{"[...all].shugo", "products/[slug].shugo", "[section]/new.shugo", "products/new.shugo", "index.shugo", "products/index.shugo"}
	var routes [][]Segment
	for _, file := range files {
		segments, _, _ := Parse(file)
		routes = append(routes, segments)
	}
	Sort(routes, func(s []Segment) []Segment { return s })
	var got []string
	for _, r := range routes {
		got = append(got, Pattern(r))
	}
	want := "/ /products /products/new /products/[slug] /[section]/new /[...all]"
	if strings.Join(got, " ") != want {
		t.Errorf("\n got %s\nwant %s", strings.Join(got, " "), want)
	}
}
