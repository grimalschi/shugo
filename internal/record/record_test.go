package record

import (
	"reflect"
	"testing"
)

func html(texts ...string) [][]byte {
	out := make([][]byte, len(texts))
	for i, text := range texts {
		out[i] = []byte(text)
	}
	return out
}

// sample is a page in a layout: the page shows a shared card, which it gives
// a snippet with another shared block in it, and an island.
func sample() *Record {
	return &Record{Top: 3, Blocks: []Block{
		{Use: true, Shared: 9},
		{Body: Fragment{Texts: html("<li>", "</li>"), Holes: []Hole{{Kind: Child, Block: 0}}}},
		{Use: true, Shared: 8, Snippets: []Snippet{{"children", []int{1}}}},
		{Use: true, Shared: 7, Snippets: []Snippet{{"children", []int{4}}}},
		{
			Body: Fragment{
				Texts: html("<h1>Jug</h1>", "", "x\x00y"),
				Holes: []Hole{{Kind: Child, Block: 2}, {Kind: Island, Text: []byte(`{"id":"src/c.island.svelte"}`)}},
			},
			Head:  Fragment{Texts: html(`<meta name="description" content="d">`)},
			Title: []byte("<title>Jug</title>"),
		},
	}}
}

func TestRoundTrip(t *testing.T) {
	r := sample()
	data := r.Encode()
	back, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	// What has no texts comes back with one empty text: compare what is
	// written.
	if again := back.Encode(); string(again) != string(data) {
		t.Errorf("not the same after a round trip:\n got %+v\nwant %+v", back, r)
	}
	if err := back.Check(); err != nil {
		t.Error(err)
	}
	page := back.Blocks[4]
	if string(page.Title) != "<title>Jug</title>" || string(page.Body.Texts[2]) != "x\x00y" || page.Body.Holes[1].Kind != Island || !back.Blocks[1].Head.Empty() {
		t.Errorf("the page: %+v", page)
	}
	if got := back.Blocks[2].Snippet([]byte("children")); !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("snippet children: %v", got)
	}
}

// Whatever the bytes are, decoding them gives a record or an error.
func TestDecodeDamaged(t *testing.T) {
	data := sample().Encode()
	for cut := 0; cut < len(data); cut++ {
		if _, err := Decode(data[:cut]); err == nil {
			t.Errorf("cut at %d: no error", cut)
		}
	}
	for i := range data {
		broken := append([]byte(nil), data...)
		broken[i] ^= 0xff
		if r, err := Decode(broken); err == nil {
			r.Check() // must not panic
		}
	}
	if _, err := Decode(append(data, 0)); err == nil {
		t.Error("bytes after the end: no error")
	}
}

func TestCheck(t *testing.T) {
	cases := map[string]func(r *Record){
		"shown at two places": func(r *Record) { r.Blocks[1].Body.Holes[0].Block = 2 },
		"shown nowhere":       func(r *Record) { r.Blocks[3].Snippets = nil },
		"a circle":            func(r *Record) { r.Blocks[1].Body.Holes[0].Block = 4 },
		"no such block":       func(r *Record) { r.Blocks[4].Body.Holes[0].Block = 9 },
		"no top":              func(r *Record) { r.Top = 9 },
		"texts and holes":     func(r *Record) { r.Blocks[4].Body.Texts = r.Blocks[4].Body.Texts[:1] },
	}
	for name, damage := range cases {
		r := sample()
		damage(r)
		if err := r.Check(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
