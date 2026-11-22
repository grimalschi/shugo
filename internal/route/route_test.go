package route

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ file, pattern string }{
		{"index.svelte", "/"},
		{"about.svelte", "/about"},
		{"blog/index.svelte", "/blog"},
		{"blog/[slug].svelte", "/blog/[slug]"},
		{"docs/[...path].svelte", "/docs/[...path]"},
		{"(shop)/cart.svelte", "/cart"},
		{"(shop)/products/[id]/reviews.svelte", "/products/[id]/reviews"},
		{"404.svelte", "/404"},
		{"café/index.svelte", "/café"},
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
	for _, file := range []string{
		"_parts/card.svelte",
		"blog/_draft.svelte",
		"index.island.svelte",
		"(shop)/products/[id]/Cart.island.svelte",
		"blog/readme.md",
	} {
		if _, ok, err := Parse(file); ok || err != nil {
			t.Errorf("%s: %v %v", file, ok, err)
		}
	}
	errors := map[string]string{
		"blog/+page.svelte": "kept for the framework",
		"[...all]/x.svelte": "must be the last part",
		"post-[id].svelte":  "mixes text and a parameter",
		"[a]/[a].svelte":    "is there twice",
		"[my-id].svelte":    "is not a name of a parameter",
	}
	for file, want := range errors {
		if _, _, err := Parse(file); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", file, err, want)
		}
	}
}

func TestOrder(t *testing.T) {
	files := []string{"[...all].svelte", "products/[slug].svelte", "[section]/new.svelte", "products/new.svelte", "index.svelte", "products/index.svelte"}
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
