package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/store"
)

// report is everything the page shows.
type report struct {
	Project  string
	Seed     string
	Files    []sourceFile
	Compiled *compiled
	Build    stage
	Story    pageStory
	Steps    []step
	Final    *look
	Serving  serving
}

type sourceFile struct {
	Path, Source, Module string
	Number               int // the number of the file in the skeleton; -1 for files that are not templates
	Shared               bool
}

// stage is the state after a run of shugo, with the conversation of the run.
type stage struct {
	Report            string
	Look              *look
	Requests, Answers []frame
	Rounds            []round
}

// step is one change of the data and the update after it.
type step struct {
	Number            int
	Plan              stepPlan
	Position          int64
	Journal           []journalLine
	Changes           []changeView
	Report            string
	Summary           map[string]any
	Diff              storeDiff
	HTML              []htmlChange
	Requests, Answers []frame
	Rounds            []round
	After             *look
}

type changeView struct {
	change
	Hits   []hit
	Misses []miss
}

// storeDiff is what an update changed in the store.
type storeDiff struct {
	PagesAdded, PagesChanged, PagesRemoved    []string
	SharedAdded, SharedChanged, SharedRemoved []string
	ReadsAdded, ReadsRemoved                  []readRow
	ShowsAdded, ShowsRemoved                  []showRow
	Version                                   [2]string
	// Build and Remove are what an export made before the update does after
	// it: it builds the pages whose HTML may have changed and removes the
	// files of the pages that went away, and touches no other page. The
	// store tells it: see exportAfter.
	Build, Remove []string
}

func diff(before, after storeLook) storeDiff {
	var d storeDiff
	pages := func(s storeLook) map[string]string {
		m := map[string]string{}
		for _, p := range s.Pages {
			m[p.Path] = p.Hash + string(p.RawReads)
		}
		return m
	}
	shared := func(s storeLook) map[string]string {
		m := map[string]string{}
		for _, b := range s.Shared {
			m[fmt.Sprintf("%d %s", b.ID, b.Name)] = b.Hash + string(b.RawReads)
		}
		return m
	}
	compare := func(was, is map[string]string) (added, changed, removed []string) {
		for k, v := range is {
			if old, ok := was[k]; !ok {
				added = append(added, k)
			} else if old != v {
				changed = append(changed, k)
			}
		}
		for k := range was {
			if _, ok := is[k]; !ok {
				removed = append(removed, k)
			}
		}
		sort.Strings(added)
		sort.Strings(changed)
		sort.Strings(removed)
		return
	}
	d.PagesAdded, d.PagesChanged, d.PagesRemoved = compare(pages(before), pages(after))
	d.SharedAdded, d.SharedChanged, d.SharedRemoved = compare(shared(before), shared(after))
	reads := func(s storeLook) map[string]readRow {
		m := map[string]readRow{}
		for _, r := range s.Reads {
			m[fmt.Sprintf("%d %x %s", r.Shape, r.Raw, r.Reader)] = r
		}
		return m
	}
	wasR, isR := reads(before), reads(after)
	for k, r := range isR {
		if _, ok := wasR[k]; !ok {
			d.ReadsAdded = append(d.ReadsAdded, r)
		}
	}
	for k, r := range wasR {
		if _, ok := isR[k]; !ok {
			d.ReadsRemoved = append(d.ReadsRemoved, r)
		}
	}
	shows := func(s storeLook) map[showRow]bool {
		m := map[showRow]bool{}
		for _, r := range s.Shows {
			m[r] = true
		}
		return m
	}
	wasS, isS := shows(before), shows(after)
	for r := range isS {
		if !wasS[r] {
			d.ShowsAdded = append(d.ShowsAdded, r)
		}
	}
	for r := range wasS {
		if !isS[r] {
			d.ShowsRemoved = append(d.ShowsRemoved, r)
		}
	}
	for _, list := range [][]readRow{d.ReadsAdded, d.ReadsRemoved} {
		sort.Slice(list, func(a, b int) bool { return list[a].Reader+string(list[a].Raw) < list[b].Reader+string(list[b].Raw) })
	}
	d.Version = [2]string{metaValue(before, "version"), metaValue(after, "version")}
	return d
}

// exportAfter asks the store what an export made at version since would
// do now: which pages it would build again, and whose files it would remove.
func exportAfter(path string, since int64) (build, remove []string, err error) {
	st, err := store.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer st.Close()
	err = st.View(func(v *store.View) error {
		return v.Changes(since, func(path string, _ int, record []byte) error {
			if record == nil {
				remove = append(remove, path)
			} else {
				build = append(build, path)
			}
			return nil
		})
	})
	sort.Strings(build)
	sort.Strings(remove)
	return build, remove, err
}

func metaValue(s storeLook, name string) string {
	for _, p := range s.Meta {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

// htmlChange is a page whose HTML differs, with the lines that do.
type htmlChange struct {
	Path  string
	Lines []diffLine
}

type diffLine struct {
	Kind string // "=", "-" or "+"
	Text string
}

func pagesThatDiffer(before, after map[string]string) []htmlChange {
	var paths []string
	for path, html := range after {
		if before[path] != html {
			paths = append(paths, path)
		}
	}
	for path := range before {
		if _, still := after[path]; !still {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	var out []htmlChange
	for _, path := range paths {
		out = append(out, htmlChange{path, lineDiff(before[path], after[path])})
	}
	return out
}

// lineDiff compares two short texts line by line and keeps the lines that
// differ with one line around them.
func lineDiff(a, b string) []diffLine {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	if a == "" {
		x = nil
	}
	if b == "" {
		y = nil
	}
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var all []diffLine
	i, j := 0, 0
	for i < len(x) || j < len(y) {
		switch {
		case i < len(x) && j < len(y) && x[i] == y[j]:
			all = append(all, diffLine{"=", x[i]})
			i, j = i+1, j+1
		case j < len(y) && (i == len(x) || lcs[i][j+1] >= lcs[i+1][j]):
			all = append(all, diffLine{"+", y[j]})
			j++
		default:
			all = append(all, diffLine{"-", x[i]})
			i++
		}
	}
	keep := make([]bool, len(all))
	for k, line := range all {
		if line.Kind != "=" {
			for n := max(0, k-1); n <= min(len(all)-1, k+1); n++ {
				keep[n] = true
			}
		}
	}
	var out []diffLine
	for k, line := range all {
		if keep[k] {
			out = append(out, line)
		} else if len(out) > 0 && out[len(out)-1].Kind != "…" {
			out = append(out, diffLine{"…", ""})
		}
	}
	return out
}

// pageStory follows one page from its record to its HTML.
type pageStory struct {
	Path   string
	Page   *pageRow
	Blocks []blockView
	Shared []sharedView
	HTML   string
	Bytes  []byteRow
}

type blockView struct {
	Index      int
	Use        bool
	Shared     int64
	SharedName string
	Snippets   []string
	Body, Head []piece
	Title      string
}

type sharedView struct {
	ID     int64
	Name   string
	Blocks []blockView
}

func blocks(s storeLook, rec *record.Record) []blockView {
	var out []blockView
	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		v := blockView{Index: i, Use: b.Use, Shared: b.Shared, Title: string(b.Title)}
		for _, snippet := range b.Snippets {
			v.Snippets = append(v.Snippets, fmt.Sprintf("%s = blocks %v", snippet.Name, snippet.Blocks))
		}
		if b.Use {
			v.SharedName = s.SharedName(b.Shared)
		} else {
			v.Body, v.Head = pieces(b.Body, rec), pieces(b.Head, rec)
		}
		out = append(out, v)
	}
	return out
}

func story(c *compiled, l *look, path string) pageStory {
	st := pageStory{Path: path, HTML: l.HTML[path]}
	for i := range l.Store.Pages {
		if l.Store.Pages[i].Path == path {
			st.Page = &l.Store.Pages[i]
		}
	}
	if st.Page == nil {
		return st
	}
	st.Blocks = blocks(l.Store, st.Page.Record)
	st.Bytes = annotate(st.Page.Raw)
	for _, id := range sharedOf(l.Store, st.Page.Record) {
		b := l.Store.SharedRow(id)
		st.Shared = append(st.Shared, sharedView{ID: id, Name: b.Name, Blocks: blocks(l.Store, b.Record)})
	}
	return st
}

// sharedOf lists the shared blocks a record shows, and the ones they show.
func sharedOf(s storeLook, rec *record.Record) []int64 {
	var out []int64
	seen := map[int64]bool{}
	var visit func(r *record.Record)
	visit = func(r *record.Record) {
		for i := range r.Blocks {
			if b := &r.Blocks[i]; b.Use && !seen[b.Shared] {
				seen[b.Shared] = true
				out = append(out, b.Shared)
				if row := s.SharedRow(b.Shared); row.Record != nil {
					visit(row.Record)
				}
			}
		}
	}
	visit(rec)
	return out
}

// serving tells what the server reads to answer for a page: the record of
// the page and the record of every shared block it shows, each with one
// lookup by key, all in one read transaction. Nothing else of the site is
// in its memory.
type serving struct {
	Path    string
	Lookups []lookup
	Bytes   int
}

type lookup struct {
	What  string
	Bytes int
}

func serve(l *look, path string) serving {
	sv := serving{Path: path}
	add := func(what string, raw []byte) {
		sv.Lookups = append(sv.Lookups, lookup{what, len(raw)})
		sv.Bytes += len(raw)
	}
	for i := range l.Store.Pages {
		if p := &l.Store.Pages[i]; p.Path == path {
			add("page record "+path, p.Raw)
			for _, id := range sharedOf(l.Store, p.Record) {
				b := l.Store.SharedRow(id)
				add(fmt.Sprintf("shared block %d record: %s", id, b.Name), b.Raw)
			}
		}
	}
	return sv
}

// What the template asks of the looks at the store.

func (s storeLook) SharedRow(id int64) sharedRow {
	for _, b := range s.Shared {
		if b.ID == id {
			return b
		}
	}
	return sharedRow{}
}

func (s storeLook) SharedName(id int64) string { return s.SharedRow(id).Name }

func (s storeLook) Shape(id int64) shapeRow {
	if id == store.SearchResultShape {
		return shapeRow{ID: id, Table: "first search results"}
	}
	for _, sh := range s.Shapes {
		if sh.ID == id {
			return sh
		}
	}
	return shapeRow{}
}

// Users are the pages and shared blocks that show a shared block.
func (s storeLook) Users(id int64) []string {
	var out []string
	for _, r := range s.Shows {
		if r.Shared == id {
			out = append(out, r.User)
		}
	}
	return out
}

// rowReader is a kind of reading that finds a row of the data, with the
// readers that made it.
type rowReader struct {
	Shape   shapeRow
	Key     []any
	Readers []string
}

// ReadersOf finds who read a row of the data: the readings whose values of
// search the row holds.
func (s storeLook) ReadersOf(t table, r row) []rowReader {
	var out []rowReader
	for _, sh := range s.Shapes {
		if sh.Table != t.Name {
			continue
		}
		key := keyFrom(r.Values, t.Columns, sh.By)
		raw := string(dataKey(key))
		var readers []string
		for _, rr := range s.Reads {
			if rr.Shape == sh.ID && string(rr.Raw) == raw {
				readers = append(readers, rr.Reader)
			}
		}
		if len(readers) > 0 {
			out = append(out, rowReader{sh, keyOrEmpty(key), readers})
		}
	}
	return out
}

// RowsOf lists the keys of the rows of the data that a reading finds now.
func (l *look) RowsOf(sh shapeRow, key []any) []string {
	want := string(dataKey(key))
	var out []string
	for _, t := range l.Data.Tables {
		if t.Name != sh.Table {
			continue
		}
		for _, r := range t.Rows {
			if string(dataKey(keyFrom(r.Values, t.Columns, sh.By))) == want {
				out = append(out, fmt.Sprint(r.Key))
			}
		}
	}
	return out
}

//go:embed report.html
var page string

var anchorUnsafe = regexp.MustCompile(`[^A-Za-z0-9]+`)

func (r *report) render() ([]byte, error) {
	funcs := template.FuncMap{
		"anchor": func(prefix, s string) string {
			return prefix + "-" + strings.Trim(anchorUnsafe.ReplaceAllString(s, "-"), "-")
		},
		"reader": func(reader string) template.HTML {
			label, target := reader, ""
			switch {
			case strings.HasPrefix(reader, "/"):
				target = "page-" + strings.Trim(anchorUnsafe.ReplaceAllString(reader, "-"), "-")
				label = "page " + reader
			case strings.HasPrefix(reader, "@"):
				label = "page list for route " + reader[1:]
				target = "route-" + reader[1:]
			default:
				if id, err := strconv.ParseInt(reader, 10, 64); err == nil {
					label = fmt.Sprintf("shared block %d: %s", id, r.Final.Store.SharedName(id))
					if name := r.Build.Look.Store.SharedName(id); name != "" {
						label = fmt.Sprintf("shared block %d: %s", id, name)
					}
					target = fmt.Sprintf("shared-%d", id)
				}
			}
			return template.HTML(fmt.Sprintf(`<a href="#%s">%s</a>`, target, template.HTMLEscapeString(label)))
		},
		"short": func(s string) string {
			if len(s) > 8 {
				return s[:8]
			}
			return s
		},
		"visible": visible,
		"sep":     separators,
		"counted": counted,
		"key": func(key []any) string {
			parts := make([]string, len(key))
			for i, v := range key {
				parts[i] = shown(v)
			}
			if len(parts) == 0 {
				return "— (no conditions)"
			}
			return strings.Join(parts, ", ")
		},
		"shown": shown,
		"rowOf": func(values []any, columns []string) string {
			if values == nil {
				return "— (no row)"
			}
			parts := make([]string, len(values))
			for i, v := range values {
				parts[i] = columns[i] + "=" + shown(v)
			}
			return strings.Join(parts, ", ")
		},
		"join":     strings.Join,
		"fragment": fragmentText,
		"pad": func(depth int) template.CSS {
			return template.CSS(fmt.Sprintf("padding-left: %.1fem", float64(depth)*1.4+0.3))
		},
		"add":    func(a, b int) int { return a + b },
		"number": func(v any) string { return fmt.Sprint(v) },
		"get": func(m map[string]any, path ...string) any {
			var v any = m
			for _, p := range path {
				mm, ok := v.(map[string]any)
				if !ok {
					return nil
				}
				v = mm[p]
			}
			return v
		},
		"names": func(v any) string {
			list, _ := v.([]any)
			parts := make([]string, len(list))
			for i, x := range list {
				parts[i] = fmt.Sprint(x)
			}
			return strings.Join(parts, ", ")
		},
	}
	t, err := template.New("page").Funcs(funcs).Parse(page)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = t.Execute(&out, r)
	return out.Bytes(), err
}

// separators shows the separators of the conversation as small marks with
// their codes, which every font can draw.
func separators(s string) template.HTML {
	var out strings.Builder
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] <= 4 {
			out.WriteString(template.HTMLEscapeString(s[start:i]))
			fmt.Fprintf(&out, `<span class="sep">%02x</span>`, s[i])
			start = i + 1
		}
	}
	out.WriteString(template.HTMLEscapeString(s[start:]))
	return template.HTML(out.String())
}

// shown writes a value of the data as the report shows it.
func shown(v any) string {
	switch v := v.(type) {
	case nil:
		return "NULL"
	case string:
		return strconv.Quote(v)
	case []byte:
		return strconv.Quote(string(v))
	}
	return fmt.Sprint(v)
}
