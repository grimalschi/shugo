package evaluate

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/store"
)

// A site of one template: text, then a component.
var site = &program.Site{Files: []program.File{
	{Path: "page.shugo", Parts: [][]program.Op{{{Kind: program.Text}, {Kind: program.Call}}}},
	{Path: "leaf.shugo", Parts: [][]program.Op{{{Kind: program.Text}}}},
}}

// frame writes one message the way the JavaScript side does.
func frame(kind byte, body string) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(body)+1))
	return append(append(out, kind), body...)
}

// What the JavaScript side sends for a page with one leaf inside it.
func pageRecord(path, text string) string {
	return "p\x02" + path + "\x020\x020" +
		"\x01b\x020\x020\x021\x02\x02\x00" + text +
		"\x01b\x021\x020\x02\x02\x02\x00leaf"
}

func newKeeper(t *testing.T) (*keeper, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	u, err := st.Update()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(u.Rollback)
	return &keeper{
		site: site, update: u, report: &Report{}, names: 10,
		pages: map[string]record.Hash{}, shared: map[string]store.Kept{}, values: map[record.Hash]struct{}{},
		seenPages: map[string]struct{}{}, seenShared: map[string]struct{}{}, numbers: map[int64]int64{}, live: map[record.Hash]struct{}{},
	}, st
}

func TestReceive(t *testing.T) {
	k, _ := newKeeper(t)
	stream := append(frame('R', pageRecord("/", "home")+"\x03"+pageRecord("/about", "about")), frame('D', `{"shadowed":2}`)...)
	if err := receive(bufio.NewReader(bytes.NewReader(stream)), k); err != nil {
		t.Fatal(err)
	}
	if !k.done || k.report.Shadowed != 2 || k.report.Pages.Added != 2 || k.report.Blocks != 4 || k.report.Values.Added != 3 {
		t.Errorf("done=%v report=%+v", k.done, k.report)
	}
}

// The code of a project runs outside of our control. Whatever it sends, the
// result is an error and nothing else.
func TestReceiveRejects(t *testing.T) {
	good := pageRecord("/", "home")
	cases := map[string][]byte{
		"a cut header":             frame('R', good)[:3],
		"a cut message":            frame('R', good)[:20],
		"an empty message":         {0, 0, 0, 0},
		"an unknown kind":          frame('X', good),
		"an unknown record":        frame('R', strings.Replace(good, "p\x02", "q\x02", 1)),
		"a header without fields":  frame('R', "p\x02/"),
		"a top that is not there":  frame('R', strings.Replace(good, "\x020\x020\x01", "\x020\x029\x01", 1)),
		"an address without /":     frame('R', pageRecord("about", "x")),
		"an address with ..":       frame('R', pageRecord("/a/../b", "x")),
		"a page twice":             frame('R', good+"\x03"+good),
		"an unknown template":      frame('R', strings.Replace(good, "b\x021\x020", "b\x027\x020", 1)),
		"an unknown part":          frame('R', strings.Replace(good, "b\x021\x020", "b\x021\x025", 1)),
		"a child that is not":      frame('R', strings.Replace(good, "\x021\x02\x02\x00home", "\x024\x02\x02\x00home", 1)),
		"a child twice":            frame('R', strings.Replace(good, "\x021\x02\x02\x00home", "\x021,1\x02\x02\x00home", 1)),
		"a block shown nowhere":    frame('R', good+"\x01b\x021\x020\x02\x02\x02\x00orphan"),
		"values that do not fit":   frame('R', strings.Replace(good, "\x00home", "\x00home\x00more", 1)),
		"no values":                frame('R', strings.Replace(good, "\x00home", "", 1)),
		"a negative child":         frame('R', strings.Replace(good, "\x021\x02\x02\x00home", "\x02-1\x02\x02\x00home", 1)),
		"damaged slots":            frame('R', strings.Replace(good, "\x021\x02\x02\x00home", "\x021\x02=\x02\x00home", 1)),
		"a shared block not sent":  frame('R', "p\x02/\x020\x020\x01u\x025\x02"),
		"a shared block of a use":  frame('R', "s\x02name\x020\x020\x01u\x020\x02"),
		"a damaged error":          frame('E', "{"),
		"a damaged summary":        frame('D', "{"),
		"a message of a gigabyte+": {0xff, 0xff, 0xff, 0x7f, 'R'},
	}
	for name, stream := range cases {
		k, _ := newKeeper(t)
		if err := receive(bufio.NewReader(bytes.NewReader(stream)), k); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestScriptErrorIsReturned(t *testing.T) {
	k, _ := newKeeper(t)
	err := receive(bufio.NewReader(bytes.NewReader(frame('E', `{"message":"boom","stack":"Error: boom\n at x","place":"the page /"}`))), k)
	script, ok := err.(*ScriptError)
	if !ok || script.Message != "boom" || !strings.Contains(script.Error(), "while computing the page /") {
		t.Errorf("%v", err)
	}
}

func TestValidPath(t *testing.T) {
	for _, path := range []string{"/", "/a", "/a/b c", "/café", "/a.b/c-d_e"} {
		if !validPath(path) {
			t.Errorf("%q is rejected", path)
		}
	}
	for _, path := range []string{"", "a", "/a/", "//", "/a//b", "/.", "/a/..", "/a?b", "/a#b", "/a\\b", "/a\x00b"} {
		if validPath(path) {
			t.Errorf("%q is accepted", path)
		}
	}
}
