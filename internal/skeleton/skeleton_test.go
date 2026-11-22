package skeleton

import (
	"reflect"
	"strings"
	"testing"
)

func TestSiteRoundTrip(t *testing.T) {
	site := &Site{
		Files: []File{{Path: "src/pages/index.svelte", Shared: true, Parts: [][]Op{
			{
				{Kind: Markup, Text: `<a class="x"`},
				{Kind: Attr, Text: "href"},
				{Kind: Spread},
				{Kind: Markup, Text: ">"},
				{Kind: Text},
				{Kind: If, Bodies: [][]Op{{{Kind: HTML}}, {}}},
				{Kind: For, Bodies: [][]Op{{{Kind: Call}, {Kind: Attr, Text: "id"}}}},
				{Kind: Render},
				{Kind: For, Bodies: [][]Op{{{Kind: Text}}, {{Kind: Markup, Text: "none"}}}},
				{Kind: Element, Bodies: [][]Op{{{Kind: Attr, Text: "id"}}, {{Kind: Text}}}},
				{Kind: Island},
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
		`{"version":4,"files":[]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["x"]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["u","href"]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["f"]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["r"]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["t",[]]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[[]]]}]}`,
		`{"version":7,"files":[{"path":"a","parts":[[["n","last"]]]}]}`,
	}
	for _, data := range cases {
		if _, err := Decode([]byte(data)); err == nil {
			t.Errorf("%s: no error", data)
		}
	}
}

func TestShell(t *testing.T) {
	shell, err := ParseShell("<html><head>" + HeadPlace + "</head><body>" + BodyPlace + "</body></html>")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Shell{"<html><head>", "</head><body>", "</body></html>"}); shell != want {
		t.Errorf("got %+v", shell)
	}
	if _, err := ParseShell(DefaultShell); err != nil {
		t.Errorf("the default shell: %v", err)
	}
	for _, text := range []string{"", HeadPlace, BodyPlace + HeadPlace, HeadPlace + BodyPlace + BodyPlace} {
		if _, err := ParseShell(text); err == nil {
			t.Errorf("%q: no error", text)
		}
	}
}
