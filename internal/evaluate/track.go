package evaluate

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/store"
)

// The project's code reads its data through queries, and every query that
// runs leaves readings with the page, the shared block or the list of pages
// it ran for: see the JavaScript side. Here the readings are kept, and a row
// of the data that has changed is led back to whoever read it.

// reading is one thing a reader took from the data: the rows of a table
// that hold the values of its key in the columns its shape names.
type reading struct {
	shape int64
	key   string
}

// shape is what the readings of one kind are of.
type shape struct {
	id    int64
	table *data.Table
	// by are the columns the rows were found by: a reading holds a value for
	// each. matters are the columns a change of which matters to the reader,
	// those of by among them; of a table of many columns the last ones share
	// a bit.
	by      []int
	matters uint64
}

func bit(column int) uint64 { return 1 << min(column, 63) }

// shapes are the shapes the store knows and the ones the project's code has
// told in this run.
type shapes struct {
	update *store.Update
	source *data.DB
	known  map[string]*shape // by what the project's code calls them
	told   map[int]*shape    // by the number the project's code gave them in this run
	of     map[*data.Table][]*shape
}

func newShapes(u *store.Update, source *data.DB) (*shapes, error) {
	s := &shapes{update: u, source: source, known: map[string]*shape{}, told: map[int]*shape{}, of: map[*data.Table][]*shape{}}
	var failure error
	err := u.Shapes(func(id int64, spec string) {
		// A shape of a table or a column that is gone has no rows to find.
		if one, err := s.read(spec); err == nil {
			one.id = id
			s.add(spec, one)
		} else if !errors.As(err, new(*unknown)) {
			failure = err
		}
	})
	return s, errors.Join(err, failure)
}

func (s *shapes) add(spec string, one *shape) {
	s.known[spec] = one
	s.of[one.table] = append(s.of[one.table], one)
}

// unknown is a shape of something the data database does not have.
type unknown struct{ what string }

func (u *unknown) Error() string {
	return "the project's code read " + u.what + ", which the data database does not have"
}

// read reads a shape as the project's code writes it.
func (s *shapes) read(spec string) (*shape, error) {
	var written struct {
		Table   string   `json:"table"`
		By      []string `json:"by"`
		Columns []string `json:"columns"`
	}
	if err := json.Unmarshal([]byte(spec), &written); err != nil {
		return nil, errorf("a damaged shape of readings: %w", err)
	}
	table, found := s.source.Table(written.Table)
	if !found {
		return nil, &unknown{"the table " + written.Table}
	}
	one := &shape{table: table}
	for _, name := range written.By {
		at, found := table.Column(name)
		if !found {
			return nil, &unknown{"the column " + written.Table + "." + name}
		}
		one.by = append(one.by, at)
		one.matters |= bit(at)
	}
	for _, name := range written.Columns {
		at, found := table.Column(name)
		if !found {
			return nil, &unknown{"the column " + written.Table + "." + name}
		}
		one.matters |= bit(at)
	}
	return one, nil
}

// tell takes a shape from the project's code with the number it will name
// the shape by.
func (s *shapes) tell(number int, spec string) error {
	one, known := s.known[spec]
	if !known {
		var err error
		if one, err = s.read(spec); err != nil {
			return err
		}
		if one.id, err = s.update.NewShape(spec); err != nil {
			return err
		}
		s.add(spec, one)
	}
	s.told[number] = one
	return nil
}

// index makes sure that the rows a question of the project's code looks for
// can be found without reading their whole table, and tells whether an
// index had to be made for it.
func (k *keeper) index(table string, columns []string) (bool, error) {
	started := time.Now()
	name, err := k.shapes.source.Index(table, columns)
	if err != nil || name == "" {
		return false, err
	}
	k.report.Data.Indexes = append(k.report.Data.Indexes, name)
	k.report.Time["indexes"] += float64(time.Since(started).Microseconds()) / 1000
	return true, nil
}

// reached finds the readers of rows that have changed: the pages, the
// shared blocks and the lists of pages to compute again.
//
// A row reaches a reading if it holds, or held, the values of the reading
// in the columns the reading found rows by, and if it was added, removed, or
// changed in a column that matters to the reading.
func (s *shapes) reached(changes []data.Change) (map[string]struct{}, error) {
	readers := map[string]struct{}{}
	for i := range changes {
		c := &changes[i]
		var changed uint64
		for at := range c.Table.Columns {
			if c.Differs(at) {
				changed |= bit(at)
			}
		}
		for _, one := range s.of[c.Table] {
			if c.Old != nil && c.New != nil && changed&one.matters == 0 {
				continue
			}
			before, after := one.key(c.Old), one.key(c.New)
			for at, key := range [][]byte{before, after} {
				if key == nil || (at == 1 && bytes.Equal(before, after)) {
					continue
				}
				if err := s.update.Readers(one.id, key, func(reader string) { readers[reader] = struct{}{} }); err != nil {
					return nil, err
				}
			}
		}
	}
	return readers, nil
}

// key writes the values a row holds in the columns of the shape. A row that
// is not there has none, nor has one with nothing in one of the columns: a
// query never finds a row by a value it does not hold.
func (s *shape) key(row []any) []byte {
	if row == nil {
		return nil
	}
	values := make([]any, len(s.by))
	for i, at := range s.by {
		if values[i] = row[at]; values[i] == nil {
			return nil
		}
	}
	return data.Key(values)
}

// readings are the changes of who read what that a run has found and not
// told the store yet.
type readings struct {
	update *store.Update
	list   []read
}

// read says that a reader made a reading, or no longer makes it.
type read struct {
	reading
	reader string
	made   bool
}

// settle tells the store, in the order of the readings: that is the order
// the store keeps them in. A page may be dropped and come back within one
// run, so of what was said about one reader and one reading the last word
// counts.
func (r *readings) settle() error {
	sort.SliceStable(r.list, func(a, b int) bool {
		if r.list[a].reading != r.list[b].reading {
			return r.list[a].reading.before(r.list[b].reading)
		}
		return r.list[a].reader < r.list[b].reader
	})
	for _, one := range r.list {
		tell := r.update.Unread
		if one.made {
			tell = r.update.Read
		}
		if err := tell(one.shape, []byte(one.key), one.reader); err != nil {
			return err
		}
	}
	r.list = r.list[:0]
	return nil
}

func (r reading) before(other reading) bool {
	if r.shape != other.shape {
		return r.shape < other.shape
	}
	return r.key < other.key
}

// read takes what a reader read while it was computed, as the project's
// code sent it: a JSON list of readings, each the number of its shape
// followed by the values of its key. It returns the readings as the store keeps them
// with the reader, and whether they differ from the ones it has, kept. The
// difference is noted for the store. No text means that the reader read
// nothing, or is gone.
func (k *keeper) read(reader string, kept []byte, text []byte) (encoded []byte, changed bool, err error) {
	var now []reading
	if len(text) > 0 {
		if k.shapes == nil {
			return nil, false, errorf("readings of a data database, and the project names none")
		}
		decoder := json.NewDecoder(bytes.NewReader(text))
		decoder.UseNumber()
		var list []any
		if err := decoder.Decode(&list); err != nil {
			return nil, false, errorf("damaged readings: %w", err)
		}
		// One reading after another: the number of its shape, then a value
		// for every column the shape finds rows by.
		for len(list) > 0 {
			number, _ := list[0].(json.Number)
			at, err := number.Int64()
			shape := k.shapes.told[int(at)]
			if err != nil || shape == nil || len(list)-1 < len(shape.by) {
				return nil, false, errorf("a reading of a shape it has not told")
			}
			now = append(now, reading{shape.id, string(data.Key(list[1 : 1+len(shape.by)]))})
			list = list[1+len(shape.by):]
		}
		sort.Slice(now, func(a, b int) bool { return now[a].before(now[b]) })
		now = uniq(now)
	}
	for _, one := range now {
		encoded = binary.AppendUvarint(encoded, uint64(one.shape))
		encoded = binary.AppendUvarint(encoded, uint64(len(one.key)))
		encoded = append(encoded, one.key...)
	}
	if bytes.Equal(encoded, kept) {
		return kept, false, nil
	}
	before, err := readingsOf(kept)
	if err != nil {
		return nil, false, err
	}
	// Both lists are sorted: one pass finds what came and what went.
	for a, b := 0, 0; a < len(before) || b < len(now); {
		switch {
		case b == len(now) || (a < len(before) && before[a].before(now[b])):
			k.reads.list = append(k.reads.list, read{before[a], reader, false})
			a++
		case a == len(before) || now[b].before(before[a]):
			k.reads.list = append(k.reads.list, read{now[b], reader, true})
			b++
		default:
			a, b = a+1, b+1
		}
	}
	return encoded, true, nil
}

func uniq(sorted []reading) []reading {
	out := sorted[:0]
	for i, one := range sorted {
		if i == 0 || one != sorted[i-1] {
			out = append(out, one)
		}
	}
	return out
}

// readingsOf reads the readings of a reader as the store keeps them.
func readingsOf(kept []byte) ([]reading, error) {
	var out []reading
	for len(kept) > 0 {
		shape, a := binary.Uvarint(kept)
		if a <= 0 {
			return nil, errors.New("store: damaged readings")
		}
		size, b := binary.Uvarint(kept[a:])
		if b <= 0 || uint64(len(kept)-a-b) < size {
			return nil, errors.New("store: damaged readings")
		}
		out = append(out, reading{int64(shape), string(kept[a+b : a+b+int(size)])})
		kept = kept[a+b+int(size):]
	}
	return out, nil
}

// route takes what the project's code said of the list of pages of a route:
// whether the pages get more than their addresses from it, and what the
// listing read from the data.
func (k *keeper) route(index int, props bool, text []byte) error {
	if index < 0 || index >= len(k.site.Routes) {
		return errorf("the pages of route %d, which the site does not have", index)
	}
	was, kept, found, err := k.update.Route(index)
	if err != nil {
		return err
	}
	reads, changed, err := k.read(store.RouteReader(index), kept, text)
	if err != nil || (found && !changed && was == props) {
		return err
	}
	return k.update.PutRoute(index, props, reads)
}

// fits tells whether an address can be of a page of the route.
func fits(route *program.Route, path string) bool {
	var parts []string
	if path != "/" {
		parts = strings.Split(path[1:], "/")
	}
	for at, segment := range route.Segments {
		switch {
		case segment.Rest != "":
			// The rest takes what the segments after it leave, be it nothing.
			return len(parts) >= len(route.Segments)-at-1
		case len(parts) == 0 || (segment.Text != "" && segment.Text != parts[0]):
			return false
		}
		parts = parts[1:]
	}
	return len(parts) == 0
}
