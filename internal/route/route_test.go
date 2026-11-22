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
	for _, file := range []string{"_parts/card.shugo", "blog/_draft.shugo", "blog/_posts.ts", "_lib/feed.ts"} {
		if _, ok, err := Parse(file); ok || err != nil {
			t.Errorf("%s: %v %v", file, ok, err)
		}
	}
	errors := map[string]string{
		"blog/+page.shugo":  "kept for the framework",
		"[...all]/x.shugo":  "must be the last part",
		"post-[id].shugo":   "mixes text and a parameter",
		"[a]/[a].shugo":     "is there twice",
		"[my-id].shugo":     "is not a name of a parameter",
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

func TestModule(t *testing.T) {
	for file, want := range map[string]bool{"sitemap.xml.ts": true, "feed.js": true, "app.d.ts": false, "page.shugo": false, "style.css": false} {
		if Module(file) != want {
			t.Errorf("%s: %v", file, !want)
		}
	}
}

func TestOrder(t *testing.T) {
	files := []string{"[...all].shugo", "products/[slug].shugo", "[section]/new.shugo", "products/new.shugo", "index.shugo", "products/index.shugo", "products/[id].json.ts", "sitemap.xml.ts"}
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
