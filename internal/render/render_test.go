package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
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

// tape writes the values of a block.
func tape(values ...string) record.Tape {
	var t record.Tape
	for _, value := range values {
		t = t.Append([]byte(value))
	}
	return t
}

func op(kind program.Kind, text string, bodies ...[]program.Op) program.Op {
	return program.Op{Kind: kind, Text: text, Bodies: bodies}
}

func markup(text string) program.Op { return op(program.Markup, text) }

// A site of three templates:
//
//	0, the page:   <main> {text} <a href={…} {...attrs}> if/else-if, a loop, <Card>children</Card> </main>
//	1, the card:   <section> {@render children()} {@render footer?.()} </section>
//	2, the header: <header> {text} </header>
var site = &program.Site{Files: []program.File{
	{Path: "page.shugo", Parts: [][]program.Op{
		{
			markup("<main>"), op(program.Text, ""),
			markup("<a"), op(program.Attr, "href"), op(program.Attr, "class"), op(program.Spread, ""), markup(">"),
			op(program.If, "", []program.Op{markup("one")}, []program.Op{markup("two:"), op(program.Text, "")}),
			op(program.For, "", []program.Op{markup("<li>"), op(program.Text, ""), markup("</li>")}),
			op(program.Call, ""), op(program.Call, ""), op(program.HTML, ""), markup("</main>"),
		},
		{markup("<b>"), op(program.Text, ""), markup("</b>")}, // put into the card
	}},
	{Path: "card.shugo", Parts: [][]program.Op{{
		markup("<section>"), op(program.Render, ""), op(program.Render, ""), markup("</section>"),
	}}},
	{Path: "header.shugo", Parts: [][]program.Op{{markup("<header>"), op(program.Text, ""), markup("</header>")}}},
}}

// page builds a record of the page with its card and a shared header.
func page(m *memory, values ...string) *record.Record {
	m.shared = map[int64]*record.Record{7: {Blocks: []record.Block{{File: 2, Values: tape("Shop & Co")}}}}
	return &record.Record{Blocks: []record.Block{
		{File: 0, Values: tape(values...), Children: []int{2, 3}},
		{File: 0, Part: 1, Values: tape("fill")},
		{File: 1, Values: tape("f0:children", "-"), Snippets: []record.Snippet{{Name: "children", Blocks: []int{1}}}},
		{Use: true, Shared: 7},
	}}
}

var good = []string{"<Tom>", `=/a?b="c"`, "+", "2", "id", "=x", "hidden", "+", "2", "second", "2", "a", "b", "<i>raw</i>"}

func TestPage(t *testing.T) {
	m := &memory{}
	html, err := Page(nil, site, page(m, good...), m)
	if err != nil {
		t.Fatal(err)
	}
	want := `<main>&lt;Tom><a href="/a?b=&quot;c&quot;" class="" id="x" hidden="">two:second<li>a</li><li>b</li>` +
		`<section><b>fill</b></section><header>Shop &amp; Co</header><i>raw</i></main>`
	if string(html) != want {
		t.Errorf("\n got %s\nwant %s", html, want)
	}
}

func TestSnippetsNobodyGave(t *testing.T) {
	m := &memory{}
	rec := &record.Record{Blocks: []record.Block{
		// The card shows no children, and a footer that nobody gave: it
		// shows nothing there.
		{File: 1, Values: tape("-", "f0:footer")},
	}}
	html, err := Page(nil, site, rec, m)
	if err != nil || string(html) != "<section></section>" {
		t.Errorf("%s %v", html, err)
	}
}

// Whatever the values are, the page comes out whole or not at all.
func TestValuesThatDoNotFit(t *testing.T) {
	change := func(at int, value string) []string {
		out := append([]string(nil), good...)
		out[at] = value
		return out
	}
	cases := map[string][]string{
		"fewer values":             good[:len(good)-1],
		"more values":              append(append([]string(nil), good...), "extra"),
		"no values":                nil,
		"a branch that is not":     change(8, "3"),
		"a branch not a number":    change(8, "x"),
		"a negative count":         change(10, "-1"),
		"a count without items":    change(10, "200"),
		"an attribute sign":        change(1, "?x"),
		"an empty attribute":       change(2, ""),
		"a bad attribute name":     change(4, `on"x`),
		"a spread count too large": change(3, "999"),
	}
	for name, values := range cases {
		m := &memory{}
		html, err := Page([]byte("before"), site, page(m, values...), m)
		if err == nil {
			t.Errorf("%s: no error, got %s", name, html)
		}
		if string(html) != "before" {
			t.Errorf("%s: a part of the page was left: %s", name, html)
		}
	}
}

func TestRecordsThatDoNotFit(t *testing.T) {
	cases := map[string]func(m *memory, rec *record.Record){
		"fewer children":        func(_ *memory, rec *record.Record) { rec.Blocks[0].Children = []int{2} },
		"more children":         func(_ *memory, rec *record.Record) { rec.Blocks[0].Children = []int{2, 3, 1} },
		"a child out of range":  func(_ *memory, rec *record.Record) { rec.Blocks[0].Children = []int{2, 9} },
		"an unknown template":   func(_ *memory, rec *record.Record) { rec.Blocks[2].File = 9 },
		"an unknown part":       func(_ *memory, rec *record.Record) { rec.Blocks[1].Part = 5 },
		"a lost shared block":   func(m *memory, _ *record.Record) { delete(m.shared, 7) },
		"damaged values":        func(_ *memory, rec *record.Record) { rec.Blocks[1].Values = record.Tape{9, 'x'} },
		"a top out of range":    func(_ *memory, rec *record.Record) { rec.Top = 9 },
		"a block inside itself": func(_ *memory, rec *record.Record) { rec.Blocks[2].Snippets[0].Blocks = []int{2} },
		"a shared block inside itself": func(m *memory, _ *record.Record) {
			m.shared[7] = &record.Record{Blocks: []record.Block{{File: 1, Values: tape("-", "-")}, {Use: true, Shared: 7}}, Top: 1}
		},
		"a snippet sign that is not": func(_ *memory, rec *record.Record) { rec.Blocks[2].Values = tape("x0", "-") },
		"a snippet too far out":      func(_ *memory, rec *record.Record) { rec.Blocks[2].Values = tape("f9:children", "-") },
	}
	for name, damage := range cases {
		m := &memory{}
		rec := page(m, good...)
		damage(m, rec)
		if html, err := Page(nil, site, rec, m); err == nil {
			t.Errorf("%s: no error, got %s", name, html)
		}
	}
}

func TestCheck(t *testing.T) {
	m := &memory{}
	rec := page(m, good...)
	for i := range rec.Blocks {
		if b := &rec.Blocks[i]; !b.Use {
			if err := Check(site, b); err != nil {
				t.Errorf("block %d: %v", i, err)
			}
		}
	}
	whole := rec.Blocks[0].Values
	rec.Blocks[0].Values = whole[:len(whole)-12]
	if err := Check(site, &rec.Blocks[0]); err == nil {
		t.Error("values cut short pass the check")
	}
	rec.Blocks[0].Values, rec.Blocks[0].Children = whole, nil
	if err := Check(site, &rec.Blocks[0]); err == nil {
		t.Error("a block without its children passes the check")
	}
}

// Values are escaped as Svelte escapes them: "&" and "<" between tags, the
// double quote too inside an attribute.
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
