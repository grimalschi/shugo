package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/skeleton"
)

// memory is a source kept in a test.
type memory struct {
	shared map[int64]*record.Record
}

func (m *memory) Shared(id int64) (*record.Record, error) {
	rec, ok := m.shared[id]
	if !ok {
		return nil, fmt.Errorf("no shared block %d", id)
	}
	return rec, nil
}

// html makes a fragment: texts, and between them holes written as c3 (a
// block of the record), gname (a snippet given) or i{…} (an island).
func html(parts ...string) record.Fragment {
	var f record.Fragment
	for i, part := range parts {
		if i%2 == 0 {
			f.Texts = append(f.Texts, []byte(part))
			continue
		}
		hole := record.Hole{Kind: record.HoleKind(part[0])}
		if hole.Kind == record.Child {
			fmt.Sscan(part[1:], &hole.Block)
		} else {
			hole.Text = []byte(part[1:])
		}
		f.Holes = append(f.Holes, hole)
	}
	return f
}

// A site of a page in a layout, and a module that gives the file /feed.xml.
var site = &skeleton.Site{
	Files: []skeleton.File{{Path: "src/pages/index.svelte"}, {Path: "src/pages/+layout.svelte", Shared: true}, {Path: "src/pages/feed.xml.ts"}},
	Routes: []skeleton.Route{
		{File: 0, Pattern: "/", Layouts: []int{1}},
		{File: 2, Pattern: "/feed.xml", Extension: ".xml"},
	},
	Shell:  skeleton.Shell{Before: "<html><head>", Middle: "</head><body>", After: "</body></html>"},
	Mounts: map[string]string{"head": "<link>", "body": "<script></script>"},
}

// page builds the record of the page and the shared blocks it shows:
//
//	7, the layout: <body> with a title, a nav and the page as its children
//	8, a card: <section> with the snippet the page gave it, and a price
//	9, the price, shared by the card and the page
func page(m *memory) *record.Record {
	m.shared = map[int64]*record.Record{
		7: {Blocks: []record.Block{{
			Body:  html("<nav>Shop</nav><main>", "gchildren", "</main>"),
			Head:  html(`<meta name="layout">`),
			Title: []byte("<title>Shop</title>"),
		}}},
		8: {Top: 1, Blocks: []record.Block{
			{Use: true, Shared: 9},
			{Body: html("<section>", "gchildren", "", "c0", "</section>")},
		}},
		9: {Blocks: []record.Block{{Body: html("<b>9 &amp; $1</b>"), Head: html(`<meta name="price">`)}}},
	}
	return &record.Record{Top: 4, Blocks: []record.Block{
		{Use: true, Shared: 9},
		{Body: html("<p>a jug</p>", "c0", "")},
		{Use: true, Shared: 8, Snippets: []record.Snippet{{Name: "children", Blocks: []int{1}}}},
		{
			Body:  html("<h1>Jug</h1>", "c2", ""),
			Head:  html(`<meta name="page">`),
			Title: []byte("<title>Jug</title>"),
		},
		{Use: true, Shared: 7, Snippets: []record.Snippet{{Name: "children", Blocks: []int{3}}}},
	}}
}

func TestPage(t *testing.T) {
	m := &memory{}
	got, err := Page(nil, site, 0, page(m), m)
	if err != nil {
		t.Fatal(err)
	}
	// The head is put together in the order the blocks are shown, the last
	// title after it: the page's, shown inside the layout.
	want := `<html><head><meta name="layout"><meta name="page"><meta name="price"><meta name="price"><title>Jug</title><link></head><body>` +
		`<nav>Shop</nav><main><h1>Jug</h1><section><p>a jug</p><b>9 &amp; $1</b><b>9 &amp; $1</b></section></main><script></script></body></html>`
	if string(got) != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}

// The file a module gives is its text, as it is.
func TestFiles(t *testing.T) {
	m := &memory{}
	file, err := Page(nil, site, 1, &record.Record{Blocks: []record.Block{{Body: record.Text([]byte("<rss>A & B</rss>"))}}}, m)
	if err != nil || string(file) != "<rss>A & B</rss>" {
		t.Errorf("a file: %s %v", file, err)
	}
	if _, err := Page(nil, site, 2, &record.Record{Blocks: []record.Block{{Body: record.Text([]byte("x"))}}}, m); err == nil {
		t.Error("a page of a route the site does not have")
	}
}

// A shared block shows what the place gave it, and nothing for what it was
// not given.
func TestSnippetsNobodyGave(t *testing.T) {
	m := &memory{}
	page(m)
	rec := &record.Record{Top: 1, Blocks: []record.Block{{Use: true, Shared: 8}, {Body: html("", "c0", "")}}}
	got, err := Page(nil, site, 0, rec, m)
	if err != nil || !strings.Contains(string(got), "<body><section><b>9 &amp; $1</b></section>") {
		t.Errorf("%s %v", got, err)
	}
}

// Whatever the record is, the page comes out whole or not at all.
func TestRecordsThatDoNotFit(t *testing.T) {
	cases := map[string]func(m *memory, rec *record.Record){
		"a child out of range":  func(_ *memory, rec *record.Record) { rec.Blocks[3].Body.Holes[0].Block = 9 },
		"a hole of no kind":     func(_ *memory, rec *record.Record) { rec.Blocks[3].Body.Holes[0].Kind = 'x' },
		"a lost shared block":   func(m *memory, _ *record.Record) { delete(m.shared, 9) },
		"a top out of range":    func(_ *memory, rec *record.Record) { rec.Top = 9 },
		"a block inside itself": func(_ *memory, rec *record.Record) { rec.Blocks[2].Snippets[0].Blocks = []int{3} },
		"a snippet in the page": func(_ *memory, rec *record.Record) { rec.Blocks[3].Body = html("", "gchildren", "") },
		"a shared block of uses": func(m *memory, _ *record.Record) {
			m.shared[9] = &record.Record{Blocks: []record.Block{{Use: true, Shared: 8}}}
		},
		"a shared block inside itself": func(m *memory, _ *record.Record) {
			m.shared[9] = &record.Record{Top: 1, Blocks: []record.Block{{Use: true, Shared: 9}, {Body: html("", "c0", "")}}}
		},
		"an island that is not": func(_ *memory, rec *record.Record) { rec.Blocks[1].Body = html("", `i{"id":""}`, "") },
	}
	for name, damage := range cases {
		m := &memory{}
		rec := page(m)
		damage(m, rec)
		if got, err := Page([]byte("before"), site, 0, rec, m); err == nil || string(got) != "before" {
			t.Errorf("%s: %v, got %s", name, err, got)
		}
	}
}

func TestCheck(t *testing.T) {
	good := &record.Block{Body: html("<p>", "c0", "</p>")}
	if err := Check(site, good); err != nil {
		t.Errorf("a block without islands: %v", err)
	}
	bad := &record.Block{Body: html("<p>", `i{"id":"x","url":"/elsewhere.js","props":{}}`, "</p>")}
	if err := Check(site, bad); err == nil {
		t.Error("an island from outside the assets passes the check")
	}
}

// Text is escaped as Svelte escapes it: "&" and "<" between tags, the double
// quote too inside an attribute.
func TestEscaping(t *testing.T) {
	if got, want := string(appendEscaped(nil, []byte(`a<b>&"c"'d' — é`), false)), `a&lt;b>&amp;"c"'d' — é`; got != want {
		t.Errorf("text:\n got %s\nwant %s", got, want)
	}
	if got, want := string(appendEscaped(nil, []byte(`a<b>&"c"'d' — é`), true)), `a&lt;b>&amp;&quot;c&quot;'d' — é`; got != want {
		t.Errorf("attribute:\n got %s\nwant %s", got, want)
	}
	if strings.ContainsAny(string(appendEscaped(nil, []byte(`"><script>`), true)), `"<`) {
		t.Error("a value can leave its attribute")
	}
}
