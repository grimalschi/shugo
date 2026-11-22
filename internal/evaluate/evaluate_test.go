package evaluate

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/binary"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/store"
)

// A site of two templates: a page, and a leaf that the page shows.
var site = &skeleton.Site{
	Files: []skeleton.File{
		{Path: "page.svelte"},
		{Path: "leaf.svelte", Shared: true},
	},
	Routes: []skeleton.Route{
		{File: 0, Pattern: "/special", Segments: []skeleton.Segment{{Text: "special"}}},
		{File: 0, Pattern: "/[slug]", Segments: []skeleton.Segment{{Param: "slug"}}},
		{File: 0, Pattern: "/[...rest]", Segments: []skeleton.Segment{{Rest: "rest"}}},
	},
}

// frame writes one message the way the JavaScript side does.
func frame(kind byte, body string) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(body)+1))
	return append(append(out, kind), body...)
}

// What the JavaScript side sends for a page with a leaf inside it: one
// block of HTML, with no head and no title.
func pageRecord(path, text string) string {
	return "p\x02" + path + "\x021\x020\x02" +
		"\x01b\x02" + text + "<leaf>\x02\x02"
}

// The same page showing a shared block in place of its own leaf: a hole
// for block 1, which is the use of the shared block.
func pageWith(path, text, shared string) string {
	return "p\x02" + path + "\x021\x020\x02" +
		"\x01b\x02" + text + "\x00c1\x00\x02\x02" +
		"\x01u\x02" + shared + "\x02"
}

// A shared block: a leaf, or a block that shows another shared block.
func leafRecord(name, text string) string {
	return "s\x02" + name + "\x020\x020\x02\x01b\x02" + text + "\x02\x02"
}

func wrapRecord(name, text, shared string) string {
	return "s\x02" + name + "\x020\x020\x02\x01b\x02" + text + "\x00c1\x00\x02\x02\x01u\x02" + shared + "\x02"
}

// A lab is a store with updates run against it by hand.
type lab struct {
	t     *testing.T
	path  string
	store *store.Store
}

func newLab(t *testing.T) *lab {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return &lab{t, path, st}
}

// update runs one update: do feeds records to the keeper, and the update is
// kept unless do or the end of it fails.
func (l *lab) update(do func(k *keeper) error) (*Report, error) {
	l.t.Helper()
	u, err := l.store.Update()
	if err != nil {
		l.t.Fatal(err)
	}
	defer u.Rollback()
	report := &Report{}
	k, err := newKeeper(site, u, report, 10)
	if err != nil {
		l.t.Fatal(err)
	}
	if err := do(k); err != nil {
		return report, err
	}
	if err := k.finish(); err != nil {
		return report, err
	}
	return report, u.Commit()
}

// must runs an update that puts the given records, and what they show next.
func (l *lab) must(records ...string) *Report {
	l.t.Helper()
	report, err := l.update(func(k *keeper) error {
		for _, one := range records {
			if err := k.put([]byte(one)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return report
}

// held tells which pages the store holds, and checks its own count of them.
func (l *lab) held() (pages []string) {
	l.t.Helper()
	err := l.store.View(func(v *store.View) error {
		totals, err := v.Totals()
		if err != nil {
			return err
		}
		err = v.Pages(func(path string, _ int, _ []byte) error {
			pages = append(pages, path)
			return nil
		})
		if err != nil {
			return err
		}
		// The store has no reading of all shared blocks: the file is asked.
		db, err := sql.Open("sqlite", l.path)
		if err != nil {
			return err
		}
		defer db.Close()
		var count int
		if err := db.QueryRow("SELECT count(*) FROM shared").Scan(&count); err != nil {
			return err
		}
		if totals.Pages != len(pages) || totals.Shared != count {
			l.t.Errorf("the store counts %+v, and holds %d pages and %d shared blocks", totals, len(pages), count)
		}
		return nil
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return pages
}

// mark returns which state of the store it is.
func (l *lab) mark() (mark store.Mark) {
	l.t.Helper()
	err := l.store.View(func(v *store.View) (err error) {
		mark, err = v.Mark()
		return err
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return mark
}

// changes returns what an export made at version since would do: the pages
// it builds again, whose HTML has changed after it, and the pages whose
// files it removes.
func (l *lab) changes(since int64) (built, gone []string) {
	l.t.Helper()
	err := l.store.View(func(v *store.View) error {
		return v.Changes(since, func(path string, _ int, record []byte) error {
			if record != nil {
				built = append(built, path)
			} else {
				gone = append(gone, path)
			}
			return nil
		})
	})
	if err != nil {
		l.t.Fatal(err)
	}
	sort.Strings(built)
	sort.Strings(gone)
	return built, gone
}

func talk(stream []byte) *wire {
	return &wire{asks: io.Discard, answers: bufio.NewReader(bytes.NewReader(stream))}
}

func TestList(t *testing.T) {
	stream := append(frame('P', "0\x02/special\x031\x02/about"), frame('G', "1\x02")...)
	stream = append(stream, frame('D', `{"shadowed":2}`)...)
	var pages []listed
	report, err := newLab(t).update(func(k *keeper) (err error) {
		if pages, err = talk(stream).list(nil, k); err != nil {
			return err
		}
		if _, found, err := k.update.Route(1); err != nil || !found {
			t.Errorf("what was said of the route is not kept: %v %v", found, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []listed{{0, "/special"}, {1, "/about"}}; !reflect.DeepEqual(pages, want) || report.Shadowed != 2 {
		t.Errorf("pages %v, shadowed %d", pages, report.Shadowed)
	}
}

func TestCompute(t *testing.T) {
	l := newLab(t)
	stream := append(frame('R', pageRecord("/", "home")+"\x03"+pageRecord("/about", "about")), frame('D', "")...)
	report, err := l.update(func(k *keeper) error {
		return talk(stream).compute([]unit{{name: "/"}, {name: "/about"}}, k)
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Computed.Pages != 2 || report.Pages.Added != 2 || report.Pages.Total != 2 || report.Blocks != 2 || report.Render != 2 {
		t.Errorf("report: %+v", report)
	}
}

// The code of a project runs outside of our control. Whatever it sends, the
// result is an error and nothing else.
func TestComputeRejects(t *testing.T) {
	good := pageRecord("/", "home")
	withShared := pageWith("/", "home", "leaf{}")
	done := string(frame('D', ""))
	cases := map[string][]byte{
		"a cut header":                     frame('R', good)[:3],
		"a cut message":                    frame('R', good)[:20],
		"an empty message":                 {0, 0, 0, 0},
		"an unknown kind":                  frame('X', good),
		"an unknown record":                frame('R', strings.Replace(good, "p\x02", "q\x02", 1)),
		"a header without fields":          frame('R', "p\x02/"),
		"a top that is not there":          frame('R', strings.Replace(good, "\x021\x020\x02\x01", "\x021\x029\x02\x01", 1)),
		"a route that is not there":        frame('R', strings.Replace(good, "\x021\x020\x02\x01", "\x027\x020\x02\x01", 1)),
		"readings without a data database": frame('R', strings.Replace(good, "\x021\x020\x02\x01", "\x021\x020\x02[[0]]\x01", 1)),
		"an address without /":             frame('R', pageRecord("about", "x")),
		"an address with ..":               frame('R', pageRecord("/a/../b", "x")),
		"a page twice":                     frame('R', good+"\x03"+good),
		"a child that is not":              frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00c4\x00", 1)),
		"a child twice":                    frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00c1\x00and\x00c1\x00", 1)),
		"a negative child":                 frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00c-1\x00", 1)),
		"a block shown nowhere":            frame('R', good+"\x01b\x02orphan\x02\x02"),
		"a hole of no kind":                frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00x1\x00", 1)),
		"an empty hole":                    frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00\x00", 1)),
		"a hole without its text after":    frame('R', strings.Replace(withShared, "\x00c1\x00", "\x00c1", 1)),
		"a block without its title":        frame('R', strings.Replace(good, "<leaf>\x02\x02", "<leaf>\x02", 1)),
		"a damaged escape":                 frame('R', strings.Replace(good, "home", "ho\x04Ze", 1)),
		"an escape cut short":              frame('R', strings.Replace(good, "home", "home\x04", 1)),
		"damaged snippets":                 frame('R', withShared+"="),
		"a snippet of no block":            frame('R', withShared+"children=9"),
		"a shared block of a use":          frame('R', "s\x02name\x020\x020\x02\x01u\x02other\x02"),
		"a damaged error":                  frame('E', "{"),
		"a message of a gigabyte+":         {0xff, 0xff, 0xff, 0x7f, 'R'},
		"another page than asked":          append(frame('R', pageRecord("/other", "x")), done...),
		"nothing of what was asked":        []byte(done),
		"more than was asked":              append(frame('R', good+"\x03"+pageRecord("/more", "x")), done...),
		"an answer that stops":             frame('R', good),
	}
	for name, stream := range cases {
		l := newLab(t)
		_, err := l.update(func(k *keeper) error { return talk(stream).compute([]unit{{name: "/"}}, k) })
		if err == nil {
			t.Errorf("%s: no error", name)
		}
		if pages := l.held(); len(pages) != 0 {
			t.Errorf("%s: the store kept %v", name, pages)
		}
	}
}

func TestListRejects(t *testing.T) {
	for name, stream := range map[string][]byte{
		"records":              append(frame('R', pageRecord("/", "x")), frame('D', "{}")...),
		"a page without route": append(frame('P', "/"), frame('D', "{}")...),
		"a bad address":        append(frame('P', "0\x02about"), frame('D', "{}")...),
		"a damaged summary":    append(frame('P', "0\x02/"), frame('D', "{")...),
		"an unknown route":     append(frame('P', "9\x02/"), frame('D', "{}")...),
		"a note of no route":   append(frame('G', "9\x02"), frame('D', "{}")...),
		"a damaged note":       append(frame('G', "1"), frame('D', "{}")...),
	} {
		_, err := newLab(t).update(func(k *keeper) error {
			_, err := talk(stream).list(nil, k)
			return err
		})
		if err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestScriptErrorIsReturned(t *testing.T) {
	_, err := newLab(t).update(func(k *keeper) error {
		_, err := talk(frame('E', `{"message":"boom","stack":"Error: boom\n at x","place":"the page /"}`)).list(nil, k)
		return err
	})
	script, ok := err.(*ScriptError)
	if !ok || script.Message != "boom" || !strings.Contains(script.Error(), "while computing the page /") {
		t.Errorf("%v", err)
	}
}

// A page names the shared blocks it shows, and the keeper tells which of
// them the run still has to compute.
func TestSharedBlocksAreAskedFor(t *testing.T) {
	l := newLab(t)
	_, err := l.update(func(k *keeper) error {
		if err := k.put([]byte(pageWith("/", "home", "card{}"))); err != nil {
			return err
		}
		if names := k.take(); !reflect.DeepEqual(names, []string{"card{}"}) {
			t.Errorf("asked for %v", names)
		}
		// The same block again, and one more.
		if err := k.put([]byte(pageWith("/a", "a", "card{}"))); err != nil {
			return err
		}
		if err := k.put([]byte(pageWith("/b", "b", "note{}"))); err != nil {
			return err
		}
		if names := k.take(); !reflect.DeepEqual(names, []string{"note{}"}) {
			t.Errorf("asked for %v", names)
		}
		if err := k.put([]byte(leafRecord("card{}", "card"))); err != nil {
			return err
		}
		return k.put([]byte(leafRecord("note{}", "note")))
	})
	if err != nil {
		t.Fatal(err)
	}

	// A later update that computes one page does not ask for a shared block
	// the store has, unless it computes the whole site.
	_, err = l.update(func(k *keeper) error {
		if err := k.put([]byte(pageWith("/", "home again", "card{}"))); err != nil {
			return err
		}
		if names := k.take(); len(names) != 0 {
			t.Errorf("asked for %v", names)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.update(func(k *keeper) error {
		if err := k.computeAll(); err != nil {
			return err
		}
		if err := k.put([]byte(pageWith("/", "home once more", "card{}"))); err != nil {
			return err
		}
		if names := k.take(); !reflect.DeepEqual(names, []string{"card{}"}) {
			t.Errorf("asked for %v", names)
		}
		return k.put([]byte(leafRecord("card{}", "card")))
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSharedBlockWithoutRecord(t *testing.T) {
	l := newLab(t)
	_, err := l.update(func(k *keeper) error { return k.put([]byte(pageWith("/", "home", "card{}"))) })
	if err == nil || !strings.Contains(err.Error(), "card{}") {
		t.Errorf("%v", err)
	}
}

// An update of a part of the site leaves the rest as it is, and tells on
// which pages the HTML differs.
func TestPartOfTheSite(t *testing.T) {
	l := newLab(t)
	first := l.must(
		pageWith("/", "home", "card{}"), pageWith("/a", "a", "card{}"), pageRecord("/b", "b"),
		leafRecord("card{}", "card"),
	)
	if first.Pages.Total != 3 || first.Shared.Total != 1 || first.Render != 3 {
		t.Errorf("first: %+v", first)
	}

	// Nothing is computed: nothing changes.
	none := l.must()
	if none.Pages.Total != 3 || none.Shared.Total != 1 || none.Render != 0 {
		t.Errorf("nothing computed: %+v", none)
	}

	// The shared block alone: two pages show it, and an export writes them
	// and no others.
	before := l.mark().Version
	card := l.must(leafRecord("card{}", "new card"))
	if card.Computed.Shared != 1 || card.Shared.Changed != 1 || card.Pages.Changed != 0 {
		t.Errorf("the card: %+v", card)
	}
	if want := []string{"/", "/a"}; !reflect.DeepEqual(card.RenderNames, want) {
		t.Errorf("pages with new HTML: %v, want %v", card.RenderNames, want)
	}
	if built, gone := l.changes(before); !reflect.DeepEqual(built, []string{"/", "/a"}) || gone != nil {
		t.Errorf("an export builds %v and removes %v", built, gone)
	}
	if built, _ := l.changes(0); len(built) != 3 {
		t.Errorf("an export from scratch builds %v", built)
	}

	// The same record again is no change.
	same := l.must(leafRecord("card{}", "new card"), pageRecord("/b", "b"))
	if same.Computed.Pages != 1 || same.Computed.Shared != 1 || same.Pages.Changed+same.Shared.Changed != 0 || same.Render != 0 {
		t.Errorf("the same again: %+v", same)
	}

	// One page stops showing the block; the other still does.
	one := l.must(pageRecord("/a", "a"))
	if one.Pages.Changed != 1 || one.Shared.Removed != 0 || one.Shared.Total != 1 {
		t.Errorf("one page lets go: %+v", one)
	}
	// Another page takes it up.
	other := l.must(pageWith("/b", "b", "card{}"))
	if other.Pages.Changed != 1 || other.Shared.Total != 1 {
		t.Errorf("another page shows it: %+v", other)
	}

	// The last pages let go: the block goes.
	last := l.must(pageRecord("/", "home"), pageRecord("/b", "b"))
	if last.Shared.Removed != 1 || last.Shared.Total != 0 || !reflect.DeepEqual(last.Shared.RemovedNames, []string{"card{}"}) {
		t.Errorf("the last pages let go: %+v", last)
	}
}

func TestRemovedPage(t *testing.T) {
	l := newLab(t)
	l.must(pageWith("/", "home", "card{}"), pageRecord("/a", "a"), leafRecord("card{}", "card"))
	before := l.mark().Version
	report, err := l.update(func(k *keeper) error { return k.remove("/") })
	if err != nil {
		t.Fatal(err)
	}
	if report.Pages.Removed != 1 || report.Pages.Total != 1 || report.Shared.Removed != 1 || report.Render != 0 {
		t.Errorf("%+v", report)
	}
	if pages := l.held(); !reflect.DeepEqual(pages, []string{"/a"}) {
		t.Errorf("pages: %v", pages)
	}
	// An export made before removes the file of the page.
	if built, gone := l.changes(before); built != nil || !reflect.DeepEqual(gone, []string{"/"}) {
		t.Errorf("an export builds %v and removes %v", built, gone)
	}
}

// A shared block that changes is seen on the pages that show it through
// another shared block too.
func TestChangeThroughSharedBlocks(t *testing.T) {
	l := newLab(t)
	l.must(
		pageWith("/", "home", "wrap{}"), pageWith("/a", "a", "card{}"), pageRecord("/b", "b"),
		wrapRecord("wrap{}", "wrap", "card{}"), leafRecord("card{}", "card"),
	)
	report := l.must(leafRecord("card{}", "new card"))
	if want := []string{"/", "/a"}; !reflect.DeepEqual(report.RenderNames, want) {
		t.Errorf("pages with new HTML: %v, want %v", report.RenderNames, want)
	}
}

func TestSharedBlockThatShowsItself(t *testing.T) {
	for name, records := range map[string][]string{
		"itself":  {pageWith("/", "home", "wrap{}"), wrapRecord("wrap{}", "wrap", "wrap{}")},
		"through": {pageWith("/", "home", "wrap{}"), wrapRecord("wrap{}", "wrap", "more{}"), wrapRecord("more{}", "more", "wrap{}")},
	} {
		l := newLab(t)
		_, err := l.update(func(k *keeper) error {
			for _, one := range records {
				if err := k.put([]byte(one)); err != nil {
					return err
				}
			}
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "shows itself again") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Records of another compiled project are dropped, and with them what the
// store knew of the pages before: an export made before writes everything.
func TestRecordsOfAnotherProject(t *testing.T) {
	l := newLab(t)
	l.must(pageRecord("/", "home"), pageRecord("/gone", "gone"))
	report, err := l.update(func(k *keeper) error {
		if err := k.clear(); err != nil {
			return err
		}
		return k.put([]byte(pageRecord("/", "home")))
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pages.Added != 1 || report.Pages.Total != 1 {
		t.Errorf("%+v", report)
	}
	if mark := l.mark(); mark.Origin != mark.Version {
		t.Errorf("the store knows changes from version %d on, and is at %d", mark.Origin, mark.Version)
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

func TestFits(t *testing.T) {
	special, slug, rest := &site.Routes[0], &site.Routes[1], &site.Routes[2]
	nested := &skeleton.Route{Segments: []skeleton.Segment{{Text: "docs"}, {Rest: "path"}, {Text: "edit"}}}
	sitemap := &skeleton.Route{Pattern: "/sitemap.xml", Segments: []skeleton.Segment{{Text: "sitemap"}}, Extension: ".xml"}
	files := &skeleton.Route{Pattern: "/docs/[...path].md", Segments: []skeleton.Segment{{Text: "docs"}, {Rest: "path"}}, Extension: ".md"}
	for _, c := range []struct {
		route *skeleton.Route
		path  string
		fits  bool
	}{
		{special, "/special", true}, {special, "/other", false}, {special, "/special/more", false}, {special, "/", false},
		{slug, "/special", true}, {slug, "/a/b", false}, {slug, "/", false},
		{rest, "/", true}, {rest, "/a", true}, {rest, "/a/b/c", true},
		{nested, "/docs/edit", true}, {nested, "/docs/a/b/edit", true}, {nested, "/docs", false}, {nested, "/blog/a/edit", false},
		{sitemap, "/sitemap.xml", true}, {sitemap, "/sitemap", false}, {sitemap, "/sitemap.xml/a", false}, {sitemap, "/a.xml", false},
		{files, "/docs.md", true}, {files, "/docs/a/b.md", true}, {files, "/docs/.md", false}, {files, "/docs/a", false},
	} {
		if got := fits(c.route, c.path); got != c.fits {
			t.Errorf("%s fits %s: %v", c.path, c.route.Pattern, got)
		}
	}
}

// What a reader read is kept with it, and only the difference from what it
// read before is told to the store.
func TestReadings(t *testing.T) {
	kept, err := readingsOf(nil)
	if err != nil || len(kept) != 0 {
		t.Errorf("no readings: %v %v", kept, err)
	}
	var encoded []byte
	for _, one := range []reading{{1, "a"}, {1, "b"}, {300, ""}, {300, "a longer key"}} {
		encoded = binary.AppendUvarint(encoded, uint64(one.shape))
		encoded = binary.AppendUvarint(encoded, uint64(len(one.key)))
		encoded = append(encoded, one.key...)
	}
	if got, err := readingsOf(encoded); err != nil || len(got) != 4 || got[3] != (reading{300, "a longer key"}) || got[2].key != "" {
		t.Errorf("%v %v", got, err)
	}
	for cut := 1; cut < len(encoded); cut++ {
		if got, err := readingsOf(encoded[:cut]); err == nil && len(got) == 4 {
			t.Errorf("readings cut at %d are read whole", cut)
		}
	}
}
