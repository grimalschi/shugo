package sourcemap

import (
	"reflect"
	"testing"
)

// The map esbuild writes for a three-line TypeScript file.
const sample = `{"version":3,"sources":["a.ts"],"mappings":"AACA,SAAS,mBAAiC;AAC1C,OAAO,UAAU;;AACjB,OAAO","names":[]}`

func TestParseAndFind(t *testing.T) {
	m, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Lines) != 4 || len(m.Lines[2]) != 0 {
		t.Fatalf("lines: %v", m.Lines)
	}
	cases := []struct {
		line, column             int
		sourceLine, sourceColumn int
		found                    bool
	}{
		{0, 0, 1, 0, true},
		{0, 8, 1, 0, true},   // inside the first segment
		{0, 9, 1, 9, true},   // the second segment starts here
		{0, 40, 1, 42, true}, // past the last segment of the line
		{1, 7, 2, 7, true},
		{2, 0, 0, 0, false}, // a line without segments
		{3, 0, 3, 0, true},
		{9, 0, 0, 0, false},
	}
	for _, c := range cases {
		source, line, column, found := m.Find(c.line, c.column)
		if found != c.found || (found && (source != "a.ts" || line != c.sourceLine || column != c.sourceColumn)) {
			t.Errorf("Find(%d, %d) = %q %d:%d %v, want %d:%d %v", c.line, c.column, source, line, column, found, c.sourceLine, c.sourceColumn, c.found)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	m := &Map{
		Sources: []string{"page.shugo", "lib.ts"},
		Lines: [][]Segment{
			{{0, 0, 10, 4}, {12, 0, 10, 16}, {30, -1, 0, 0}},
			nil,
			{{2, 1, 0, 0}, {5, 0, 3, 100000}},
			{{0, 0, 2, 7}},
		},
	}
	back, err := Parse(m.JSON())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Sources, m.Sources) {
		t.Errorf("sources: %v", back.Sources)
	}
	for i, line := range m.Lines {
		if len(line) == 0 && len(back.Lines[i]) == 0 {
			continue
		}
		if !reflect.DeepEqual(back.Lines[i], line) {
			t.Errorf("line %d: got %v, want %v", i, back.Lines[i], line)
		}
	}
}

func TestParseDamaged(t *testing.T) {
	for _, mappings := range []string{"A!AA", "AAA", "g"} {
		if _, err := Parse([]byte(`{"version":3,"sources":["a"],"mappings":"` + mappings + `"}`)); err == nil {
			t.Errorf("%q: no error", mappings)
		}
	}
}
