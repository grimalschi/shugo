package program

import (
	"reflect"
	"strings"
	"testing"
)

func TestSiteRoundTrip(t *testing.T) {
	site := &Site{
		Files: []File{{Path: "pages/index.shugo", Shared: true, Parts: [][]Op{
			{
				{Kind: Markup, Text: `<a class="x"`},
				{Kind: URL, Text: "href"},
				{Kind: Spread},
				{Kind: Markup, Text: ">"},
				{Kind: Text},
				{Kind: If, Bodies: [][]Op{{{Kind: HTML}}, {}}},
				{Kind: For, Bodies: [][]Op{{{Kind: Call}, {Kind: Attr, Text: "id"}}}},
				{Kind: Render},
				{Kind: For, Bodies: [][]Op{{{Kind: Text}}, {{Kind: Markup, Text: "none"}}}},
				{Kind: Element, Bodies: [][]Op{{{Kind: Attr, Text: "id"}}, {{Kind: Text}}}},
				{Kind: Mount, Text: "head"},
			},
			{},
		}}},
		Routes: []Route{{File: 0, Pattern: "/[slug]", Segments: []Segment{{Param: "slug"}}, Layouts: []int{0}}},
		Mounts: map[string]string{"head": "<link>"},
	}
	data, err := site.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	// Empty lists come back as nil; compare what matters.
	again, _ := back.Encode()
	if string(again) != string(data) {
		t.Errorf("not the same after a round trip:\n%s\n%s", data, again)
	}
	if !reflect.DeepEqual(back.Files[0].Parts[0][5].Bodies[0], site.Files[0].Parts[0][5].Bodies[0]) {
		t.Errorf("branches differ: %v", back.Files[0].Parts[0][5])
	}
	if want := `["r","<a class=\"x\""]`; !strings.Contains(string(data), want) {
		t.Errorf("markup is escaped: %s", data)
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := []string{
		`{"version":2,"files":[]}`,
		`{"version":1,"files":[{"path":"a","parts":[[["x"]]]}]}`,
		`{"version":1,"files":[{"path":"a","parts":[[["f"]]]}]}`,
		`{"version":1,"files":[{"path":"a","parts":[[["r"]]]}]}`,
		`{"version":1,"files":[{"path":"a","parts":[[["t",[]]]]}]}`,
		`{"version":1,"files":[{"path":"a","parts":[[[]]]}]}`,
	}
	for _, data := range cases {
		if _, err := Decode([]byte(data)); err == nil {
			t.Errorf("%s: no error", data)
		}
	}
}
