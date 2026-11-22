package skeleton

import (
	"reflect"
	"strings"
	"testing"
)

func TestSiteRoundTrip(t *testing.T) {
	site := &Site{
		Files:  []File{{Path: "src/pages/index.svelte", Shared: true}},
		Shell:  Shell{Before: `<html><head><meta charset="utf-8">`, Middle: "</head><body>", After: "</body></html>"},
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
	if !reflect.DeepEqual(back.Shell, site.Shell) || !reflect.DeepEqual(back.Files, site.Files) {
		t.Errorf("not the same after a round trip: %+v", back)
	}
	if want := `"before":"<html><head><meta charset=\"utf-8\">"`; !strings.Contains(string(data), want) {
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
