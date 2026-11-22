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
		// A module gives files of the type its name tells.
		{"sitemap.xml.ts", "/sitemap.xml"},
		{"robots.txt.js", "/robots.txt"},
		{"blog/index.xml.ts", "/blog/index.xml"},
		{"api/[id].json.ts", "/api/[id].json"},
		{"(feeds)/rss.xml.ts", "/rss.xml"},
	}
	for _, c := range cases {
		r, ok, err := Parse(c.file)
		if err != nil || !ok {
			t.Errorf("%s: %v %v", c.file, ok, err)
			continue
		}
		if got := r.Pattern(); got != c.pattern {
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
		"blog/_posts.ts",
		"_lib/feed.ts",
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
		"api/data.ts":       "its name tells the type of that file before .ts: sitemap.xml.ts gives /sitemap.xml",
		"feed.x-y.ts":       "its name tells the type",
		"post-[id].json.ts": "mixes text and a parameter",
	}
	for file, want := range errors {
		if _, _, err := Parse(file); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", file, err, want)
		}
	}
}

func TestOrder(t *testing.T) {
	files := []string{"[...all].svelte", "products/[slug].svelte", "[section]/new.svelte", "products/new.svelte", "index.svelte", "products/index.svelte", "products/[id].json.ts", "sitemap.xml.ts"}
	var routes []Route
	for _, file := range files {
		r, _, _ := Parse(file)
		routes = append(routes, r)
	}
	Sort(routes, func(r Route) Route { return r })
	var got []string
	for _, r := range routes {
		got = append(got, r.Pattern())
	}
	want := "/ /sitemap.xml /products /products/new /products/[id].json /products/[slug] /[section]/new /[...all]"
	if strings.Join(got, " ") != want {
		t.Errorf("\n got %s\nwant %s", strings.Join(got, " "), want)
	}
}
