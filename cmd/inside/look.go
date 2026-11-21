package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/server"
	"github.com/grimalschi/shugo/internal/store"
)

// A look is everything that lies between the stages at one moment: the data
// database with its journal, the store, and the HTML of every page.
type look struct {
	Data  dataLook
	Store storeLook
	HTML  map[string]string // by address
}

// dataLook is the data database of the project.
type dataLook struct {
	Tables   []table
	Triggers []named
	Indexes  []named
	Journal  []journalLine
	Settings []pair
}

type table struct {
	Name    string
	Columns []string
	Key     string
	Rows    []row
}

type row struct {
	Values []any
	Key    any
}

type named struct{ Name, SQL string }

type pair struct{ Name, Value string }

// journalLine is one line of the journal: a row of a table that was
// touched, and what it was before, as a JSON list; empty if it was not there.
type journalLine struct {
	Seq   int64
	Table string
	Key   any
	Old   string
}

// storeLook is the store: what the runtime shows pages from, and what the
// next update finds its way by.
type storeLook struct {
	Meta   []pair
	Pages  []pageRow
	Shared []sharedRow
	Shows  []showRow
	Routes []routeRow
	Shapes []shapeRow
	Reads  []readRow
	// Gone is every page that went away, with the version it went in: an
	// export removes its file.
	Gone []goneRow
}

type goneRow struct {
	Path    string
	Version int64
}

type pageRow struct {
	Path     string
	Hash     string
	Version  int64 // in which the record last changed
	Route    int
	Raw      []byte
	Record   *record.Record
	Readings []reading
	RawReads []byte
}

type sharedRow struct {
	ID       int64
	Name     string
	Hash     string
	Version  int64 // in which the record last changed
	Raw      []byte
	Record   *record.Record
	Readings []reading
	RawReads []byte
}

type showRow struct {
	Shared int64
	User   string
}

type routeRow struct {
	ID       int
	Props    bool
	Readings []reading
}

type shapeRow struct {
	ID      int64
	Spec    string
	Table   string
	By      []string
	Columns []string
}

// reading is one thing a reader read: rows of a table found by values.
type reading struct {
	Shape int64
	Key   []any
	Raw   []byte
}

type readRow struct {
	Shape  int64
	Key    []any
	Raw    []byte
	Reader string
}

// lookAt reads everything there is to read about a project now.
func lookAt(project string) (*look, error) {
	l := &look{HTML: map[string]string{}}
	var err error
	if l.Data, err = lookAtData(filepath.Join(project, "data", "site.db")); err != nil {
		return nil, fmt.Errorf("the data database: %w", err)
	}
	path := filepath.Join(project, "_build", "store.db")
	if l.Store, err = lookAtStore(path); err != nil {
		return nil, fmt.Errorf("the store: %w", err)
	}
	st, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	pages := server.NewPages(st)
	for _, p := range l.Store.Pages {
		page, err := pages.Render(nil, p.Path)
		if err != nil {
			return nil, err
		}
		l.HTML[p.Path] = string(page.HTML)
	}
	return l, nil
}

func open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func lookAtData(path string) (dataLook, error) {
	var d dataLook
	db, err := open(path)
	if err != nil {
		return d, err
	}
	defer db.Close()
	names, err := strings1(db, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' AND name NOT LIKE '\\_shugo\\_%' ESCAPE '\\' ORDER BY rowid")
	if err != nil {
		return d, err
	}
	for _, name := range names {
		t := table{Name: name}
		rows, err := db.Query("SELECT name, pk FROM pragma_table_info(?) ORDER BY cid", name)
		if err != nil {
			return d, err
		}
		for rows.Next() {
			var column string
			var pk int
			if err := rows.Scan(&column, &pk); err != nil {
				return d, err
			}
			t.Columns = append(t.Columns, column)
			if pk > 0 {
				t.Key = column
			}
		}
		rows.Close()
		values, err := all(db, `SELECT * FROM "`+name+`" ORDER BY "`+t.Key+`"`)
		if err != nil {
			return d, err
		}
		keyAt := indexOf(t.Columns, t.Key)
		for _, v := range values {
			t.Rows = append(t.Rows, row{Values: v, Key: v[keyAt]})
		}
		d.Tables = append(d.Tables, t)
	}
	for _, kind := range []struct {
		into  *[]named
		query string
	}{
		{&d.Triggers, "SELECT name, sql FROM sqlite_master WHERE type = 'trigger' ORDER BY tbl_name, name"},
		{&d.Indexes, "SELECT name, coalesce(sql, '') FROM sqlite_master WHERE type = 'index' AND name LIKE '\\_shugo\\_%' ESCAPE '\\' ORDER BY name"},
	} {
		rows, err := all(db, kind.query)
		if err != nil {
			return d, err
		}
		for _, r := range rows {
			*kind.into = append(*kind.into, named{text(r[0]), text(r[1])})
		}
	}
	lines, err := all(db, "SELECT seq, tbl, id, coalesce(old, '') FROM _shugo_journal ORDER BY seq")
	if err != nil {
		return d, err
	}
	for _, r := range lines {
		d.Journal = append(d.Journal, journalLine{Seq: r[0].(int64), Table: text(r[1]), Key: r[2], Old: text(r[3])})
	}
	settings, err := all(db, "SELECT name, value FROM _shugo_meta ORDER BY name")
	if err != nil {
		return d, err
	}
	for _, r := range settings {
		d.Settings = append(d.Settings, pair{text(r[0]), text(r[1])})
	}
	return d, nil
}

func lookAtStore(path string) (storeLook, error) {
	var s storeLook
	db, err := open(path)
	if err != nil {
		return s, err
	}
	defer db.Close()

	meta, err := all(db, "SELECT name, value FROM meta ORDER BY name")
	if err != nil {
		return s, err
	}
	for _, r := range meta {
		value := text(r[1])
		if text(r[0]) == "site" {
			value = fmt.Sprintf("project program, %d JSON bytes — see Compilation", len(value))
		}
		s.Meta = append(s.Meta, pair{text(r[0]), value})
	}

	pages, err := all(db, "SELECT path, hash, record, route, reads, version FROM page ORDER BY path")
	if err != nil {
		return s, err
	}
	for _, r := range pages {
		p := pageRow{Path: text(r[0]), Hash: hexOf(r[1]), Raw: r[2].([]byte), Route: int(r[3].(int64)), RawReads: bytesOf(r[4]), Version: r[5].(int64)}
		if p.Record, err = record.Decode(p.Raw); err != nil {
			return s, err
		}
		if p.Readings, err = readings(p.RawReads); err != nil {
			return s, err
		}
		s.Pages = append(s.Pages, p)
	}
	shared, err := all(db, "SELECT id, name, hash, record, reads, version FROM shared ORDER BY id")
	if err != nil {
		return s, err
	}
	for _, r := range shared {
		b := sharedRow{ID: r[0].(int64), Name: text(r[1]), Hash: hexOf(r[2]), Raw: r[3].([]byte), RawReads: bytesOf(r[4]), Version: r[5].(int64)}
		if b.Record, err = record.Decode(b.Raw); err != nil {
			return s, err
		}
		if b.Readings, err = readings(b.RawReads); err != nil {
			return s, err
		}
		s.Shared = append(s.Shared, b)
	}
	gone, err := all(db, "SELECT path, version FROM gone ORDER BY path")
	if err != nil {
		return s, err
	}
	for _, r := range gone {
		s.Gone = append(s.Gone, goneRow{text(r[0]), r[1].(int64)})
	}
	shows, err := all(db, "SELECT shared, user FROM shows ORDER BY shared, user")
	if err != nil {
		return s, err
	}
	for _, r := range shows {
		s.Shows = append(s.Shows, showRow{r[0].(int64), text(r[1])})
	}
	routes, err := all(db, "SELECT id, props, reads FROM route ORDER BY id")
	if err != nil {
		return s, err
	}
	for _, r := range routes {
		rr := routeRow{ID: int(r[0].(int64)), Props: r[1].(int64) != 0}
		if rr.Readings, err = readings(bytesOf(r[2])); err != nil {
			return s, err
		}
		s.Routes = append(s.Routes, rr)
	}
	shapes, err := all(db, "SELECT id, spec FROM shape ORDER BY id")
	if err != nil {
		return s, err
	}
	for _, r := range shapes {
		sh := shapeRow{ID: r[0].(int64), Spec: text(r[1])}
		var spec struct {
			Table   string   `json:"table"`
			By      []string `json:"by"`
			Columns []string `json:"columns"`
		}
		if err := json.Unmarshal([]byte(sh.Spec), &spec); err != nil {
			return s, err
		}
		sh.Table, sh.By, sh.Columns = spec.Table, spec.By, spec.Columns
		s.Shapes = append(s.Shapes, sh)
	}
	reads, err := all(db, "SELECT shape, key, reader FROM reads ORDER BY shape, key, reader")
	if err != nil {
		return s, err
	}
	for _, r := range reads {
		raw := bytesOf(r[1])
		key, err := keyOf(raw)
		if err != nil {
			return s, err
		}
		s.Reads = append(s.Reads, readRow{Shape: r[0].(int64), Key: key, Raw: raw, Reader: text(r[2])})
	}
	return s, nil
}

// readings reads the readings of a reader as the store keeps them: the
// number of a shape and the length of a key, as varints, then the key.
func readings(data []byte) ([]reading, error) {
	var out []reading
	for len(data) > 0 {
		shape, a := uvarint(data)
		size, b := uvarint(data[a:])
		if a <= 0 || b <= 0 || len(data) < a+b+int(size) {
			return nil, fmt.Errorf("damaged readings")
		}
		raw := data[a+b : a+b+int(size)]
		key, err := keyOf(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, reading{Shape: int64(shape), Key: key, Raw: raw})
		data = data[a+b+int(size):]
	}
	return out, nil
}

// keyOf reads what data.Key wrote: a byte that tells the kind of every
// value, then the value.
func keyOf(raw []byte) ([]any, error) {
	var out []any
	for at := 0; at < len(raw); {
		switch raw[at] {
		case 0:
			out = append(out, nil)
			at++
		case 1, 2:
			if at+9 > len(raw) {
				return nil, fmt.Errorf("damaged key")
			}
			bits := uint64(0)
			for _, b := range raw[at+1 : at+9] {
				bits = bits<<8 | uint64(b)
			}
			if raw[at] == 1 {
				out = append(out, int64(bits))
			} else {
				out = append(out, float64frombits(bits))
			}
			at += 9
		case 3:
			size, n := uvarint(raw[at+1:])
			if n <= 0 || at+1+n+int(size) > len(raw) {
				return nil, fmt.Errorf("damaged key")
			}
			out = append(out, string(raw[at+1+n:at+1+n+int(size)]))
			at += 1 + n + int(size)
		case 4:
			at++ // a key of no values
		default:
			return nil, fmt.Errorf("damaged key")
		}
	}
	return out, nil
}

// change is a row of the data that is not what it was.
type change struct {
	Table   string
	Key     any
	Old     []any // nil: the row was not there
	New     []any // nil: the row is not there any more
	Columns []string
	Changed []string
	Lines   []int64 // the lines of the journal it came from
}

// changes finds, as the update does, which rows are other than they were:
// the first line of a row in the journal tells what it was, the table what
// it is now.
func changes(d dataLook, after int64, project string) ([]change, error) {
	db, err := open(filepath.Join(project, "data", "site.db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	type id struct {
		table string
		key   string
	}
	first := map[id]*change{}
	var order []id
	for _, line := range d.Journal {
		if line.Seq <= after {
			continue
		}
		k := id{line.Table, fmt.Sprint(line.Key)}
		if c, seen := first[k]; seen {
			c.Lines = append(c.Lines, line.Seq)
			continue
		}
		c := &change{Table: line.Table, Key: line.Key, Lines: []int64{line.Seq}}
		if line.Old != "" {
			decoder := json.NewDecoder(strings.NewReader(line.Old))
			decoder.UseNumber()
			if err := decoder.Decode(&c.Old); err != nil {
				return nil, err
			}
		}
		first[k] = c
		order = append(order, k)
	}
	var out []change
	for _, k := range order {
		c := first[k]
		t := tableNamed(d, c.Table)
		c.Columns = t.Columns
		rows, err := all(db, `SELECT * FROM "`+t.Name+`" WHERE "`+t.Key+`" = ?`, c.Key)
		if err != nil {
			return nil, err
		}
		if len(rows) == 1 {
			c.New = rows[0]
		}
		for i, column := range t.Columns {
			var was, is any
			if c.Old != nil {
				was = c.Old[i]
			}
			if c.New != nil {
				is = c.New[i]
			}
			if string(data.Key([]any{was})) != string(data.Key([]any{is})) || (c.Old == nil) != (c.New == nil) {
				c.Changed = append(c.Changed, column)
			}
		}
		if c.Old != nil && c.New != nil && len(c.Changed) == 0 {
			continue // written over with what it held: no change
		}
		out = append(out, *c)
	}
	return out, nil
}

func tableNamed(d dataLook, name string) table {
	for _, t := range d.Tables {
		if t.Name == name {
			return t
		}
	}
	return table{Name: name}
}

// hit is one reading that a changed row reaches, with whoever made it.
type hit struct {
	Shape   shapeRow
	Key     []any
	Readers []string
	Why     string
}

// miss is a shape of the table of a changed row that the change does not reach.
type miss struct {
	Shape shapeRow
	Why   string
}

// reach finds, as the update does, the readings a change reaches.
func reach(s storeLook, c change) (hits []hit, misses []miss) {
	changed := map[string]bool{}
	for _, column := range c.Changed {
		changed[column] = true
	}
	for _, sh := range s.Shapes {
		if sh.Table != c.Table {
			continue
		}
		matters := append(append([]string{}, sh.By...), sh.Columns...)
		if c.Old != nil && c.New != nil {
			touched := false
			for _, column := range matters {
				touched = touched || changed[column]
			}
			if !touched {
				why := fmt.Sprintf("%s changed, but only %s matter to this read shape", strings.Join(c.Changed, ", "), strings.Join(matters, ", "))
				if len(matters) == 0 {
					why = fmt.Sprintf("%s changed, but for this read shape only insertion and deletion matter", strings.Join(c.Changed, ", "))
				}
				misses = append(misses, miss{sh, why})
				continue
			}
		}
		found := false
		for side, values := range [][]any{c.Old, c.New} {
			if values == nil {
				continue
			}
			key := make([]any, len(sh.By))
			missing := false
			for i, column := range sh.By {
				key[i] = values[indexOf(c.Columns, column)]
				missing = missing || key[i] == nil
			}
			if missing {
				continue
			}
			raw := data.Key(key)
			if side == 1 && c.Old != nil && string(raw) == string(data.Key(keyFrom(c.Old, c.Columns, sh.By))) {
				continue
			}
			var readers []string
			for _, r := range s.Reads {
				if r.Shape == sh.ID && string(r.Raw) == string(raw) {
					readers = append(readers, r.Reader)
				}
			}
			why := ""
			switch {
			case len(sh.By) == 0 && c.Old == nil:
				why = "the read selected all rows, and a row was added"
			case len(sh.By) == 0 && c.New == nil:
				why = "the read selected all rows, and a row was removed"
			case len(sh.By) == 0:
				why = "the read selected all rows, and a relevant column changed"
			case c.Old == nil:
				why = "a row with these lookup values was added"
			case c.New == nil:
				why = "a row with these lookup values was removed"
			case string(data.Key(keyFrom(c.Old, c.Columns, sh.By))) == string(data.Key(keyFrom(c.New, c.Columns, sh.By))):
				why = "the row has these lookup values and a relevant column changed"
			case side == 0:
				why = "the row previously had these lookup values"
			default:
				why = "the row now has these lookup values"
			}
			if len(readers) > 0 {
				found = true
				hits = append(hits, hit{Shape: sh, Key: keyOrEmpty(key), Readers: readers, Why: why})
			}
		}
		if !found {
			misses = append(misses, miss{sh, "no reader used these lookup values"})
		}
	}
	return hits, misses
}

func keyFrom(values []any, columns, by []string) []any {
	key := make([]any, len(by))
	for i, column := range by {
		key[i] = values[indexOf(columns, column)]
	}
	return key
}

func keyOrEmpty(key []any) []any {
	if key == nil {
		return []any{}
	}
	return key
}

// compiled is what the compiler wrote.
type compiled struct {
	Site    *program.Site
	Raw     string // site.json
	Modules map[string]string
	Bundle  int
}

// lookAtCompiled reads the program of the site and the module that the
// bundle holds for every file of the project.
func lookAtCompiled(project string) (*compiled, error) {
	out := &compiled{Modules: map[string]string{}}
	raw, err := os.ReadFile(filepath.Join(project, "_build", "site.json"))
	if err != nil {
		return nil, err
	}
	out.Raw = string(raw)
	if out.Site, err = program.Decode(raw); err != nil {
		return nil, err
	}
	bundle, err := os.ReadFile(filepath.Join(project, "_build", "eval.mjs"))
	if err != nil {
		return nil, err
	}
	out.Bundle = len(bundle)
	// The bundler marks the code of every file with a comment of its path.
	// A file's own code is its last piece: the one that calls component().
	var path string
	var lines []string
	flush := func() {
		code := strings.TrimSpace(strings.Join(lines, "\n"))
		if path != "" && strings.Contains(code, "component(") {
			out.Modules[path] = code
		} else if path != "" && !strings.HasSuffix(path, ".shugo") && !strings.Contains(path, ":") && code != "" {
			if _, kept := out.Modules[path]; !kept {
				out.Modules[path] = code
			}
		}
		lines = nil
	}
	for _, line := range strings.Split(string(bundle), "\n") {
		if strings.HasPrefix(line, "// ") && !strings.HasPrefix(line, "//#") && !strings.Contains(line[3:], " ") {
			flush()
			path = line[3:]
			continue
		}
		lines = append(lines, line)
	}
	flush()
	return out, nil
}

func all(db *sql.DB, query string, arguments ...any) ([][]any, error) {
	rows, err := db.Query(query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		out = append(out, values)
	}
	return out, rows.Err()
}

func strings1(db *sql.DB, query string) ([]string, error) {
	rows, err := all(db, query)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = text(r[0])
	}
	return out, nil
}

func text(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(v)
	case string:
		return v
	}
	return fmt.Sprint(v)
}

func bytesOf(v any) []byte {
	switch v := v.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	}
	return nil
}

func hexOf(v any) string { return fmt.Sprintf("%x", bytesOf(v)) }

func indexOf(list []string, item string) int {
	for i, one := range list {
		if one == item {
			return i
		}
	}
	return -1
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
