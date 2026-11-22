package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
)

func uvarint(b []byte) (uint64, int) { return binary.Uvarint(b) }

func float64frombits(bits uint64) float64 { return math.Float64frombits(bits) }

// byteRow is a few bytes of an encoded record and what they mean.
type byteRow struct {
	Offset  int
	Hex     string
	Meaning string
}

// annotate goes through a record as record.Decode does and tells what
// every few bytes mean.
func annotate(data []byte, site *program.Site) []byteRow {
	var out []byteRow
	at := 0
	take := func(n int, meaning string) {
		end := min(at+n, len(data))
		out = append(out, byteRow{at, spaced(data[at:end]), meaning})
		at = end
	}
	number := func(meaning string) int {
		v, n := binary.Uvarint(data[at:])
		if n <= 0 {
			take(len(data)-at, "corrupt data")
			return 0
		}
		take(n, fmt.Sprintf(meaning, v))
		return int(v)
	}
	list := func(meaning string) {
		count := number("number of entries in «" + meaning + "»: %d")
		for i := 0; i < count; i++ {
			number("  block number: %d")
		}
	}
	number("record starts at block %d")
	count := number("blocks in the record: %d")
	for i := 0; i < count && at < len(data); i++ {
		file, n := binary.Uvarint(data[at:])
		if n <= 0 {
			break
		}
		if file == 0 {
			take(n, fmt.Sprintf("block %d: 0 is a reference to a shared block", i))
			number("  shared block number: %d")
		} else {
			path := ""
			if int(file)-1 < len(site.Files) {
				path = site.Files[file-1].Path
			}
			take(n, fmt.Sprintf("block %d: file %d + 1 is %s", i, file-1, path))
			number("  template part: %d")
			end := at + number("  value tape: %d bytes")
			for k := 1; at < min(end, len(data)); k++ {
				size := number(fmt.Sprintf("    value %d: %%d bytes", k))
				take(size, "    «"+visible(string(data[at:min(at+size, len(data))]))+"»")
			}
			list("nested blocks")
		}
		snippets := number("  parameterless snippets supplied by the caller: %d")
		for s := 0; s < snippets; s++ {
			size := number("  snippet name length: %d")
			take(size, "  snippet name: "+string(data[at:min(at+size, len(data))]))
			list("snippet blocks")
		}
	}
	return out
}

func spaced(b []byte) string {
	parts := make([]string, len(b))
	for i, c := range b {
		parts[i] = fmt.Sprintf("%02x", c)
	}
	return strings.Join(parts, " ")
}

// walkStep is one operation of a template, with the value it took from the tape.
type walkStep struct {
	Depth int
	Op    string
	What  string
	Value string
	Place int    // the place of the value in the tape, from 1; 0 for none
	Link  string // the block it leads to
	Ref   string
}

// walk goes through the template of a block as the renderer does and tells,
// for every operation, which value of the tape it takes.
func walk(site *program.Site, rec *record.Record, at int, values []string, sharedOwner bool) ([]walkStep, error) {
	b := &rec.Blocks[at]
	ops, ok := site.Part(b.File, b.Part)
	if !ok {
		return nil, fmt.Errorf("no template %d, part %d", b.File, b.Part)
	}
	w := &walker{site: site, rec: rec, block: b, values: values, shared: sharedOwner}
	err := w.ops(ops, 0)
	if err == nil && w.next < len(values) {
		err = fmt.Errorf("%d values are left over", len(values)-w.next)
	}
	return w.out, err
}

type walker struct {
	site   *program.Site
	rec    *record.Record
	block  *record.Block
	values []string
	next   int
	child  int
	shared bool
	out    []walkStep
}

func (w *walker) take() (string, int, error) {
	if w.next >= len(w.values) {
		return "", 0, fmt.Errorf("the tape ends too early")
	}
	w.next++
	return w.values[w.next-1], w.next, nil
}

func (w *walker) number() (int, int, error) {
	value, place, err := w.take()
	if err != nil {
		return 0, 0, err
	}
	var n int
	_, err = fmt.Sscan(value, &n)
	return n, place, err
}

func (w *walker) add(depth int, op, what, value string, place int) {
	w.out = append(w.out, walkStep{Depth: depth, Op: op, What: what, Value: value, Place: place})
}

func (w *walker) ops(ops []program.Op, depth int) error {
	for _, op := range ops {
		switch op.Kind {
		case program.Markup:
			w.add(depth, "r", "unchanged markup", op.Text, 0)
		case program.Text, program.HTML, program.Island:
			value, place, err := w.take()
			if err != nil {
				return err
			}
			what := "text: escaped"
			if op.Kind == program.HTML {
				what = "rendered HTML: inserted as-is"
			}
			if op.Kind == program.Island {
				what = "Svelte island: server-rendered markup and hydration data"
			}
			w.add(depth, string(rune(op.Kind)), what, value, place)
		case program.Attr, program.URL:
			value, place, err := w.take()
			if err != nil {
				return err
			}
			sign := map[byte]string{'=': "attribute with a value", '-': "no attribute", '+': "attribute name only"}[firstByte(value)]
			what := fmt.Sprintf("attribute %s — %s", op.Text, sign)
			if op.Kind == program.URL {
				what += "; the URL is validated"
			}
			w.add(depth, string(rune(op.Kind)), what, value, place)
		case program.Spread:
			count, place, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "s", "spread attributes: number of name–value pairs", fmt.Sprint(count), place)
			for range count {
				name, p1, err := w.take()
				if err != nil {
					return err
				}
				value, p2, err := w.take()
				if err != nil {
					return err
				}
				w.add(depth+1, "s", "name", name, p1)
				w.add(depth+1, "s", "value", value, p2)
			}
		case program.If:
			taken, place, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "i", fmt.Sprintf("conditional with %d branches: selected branch number, or 0 if none", len(op.Bodies)), fmt.Sprint(taken), place)
			if taken > 0 {
				if err := w.ops(op.Bodies[taken-1], depth+1); err != nil {
					return err
				}
			}
		case program.For:
			count, place, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "f", "loop: iteration count", fmt.Sprint(count), place)
			for i := 0; i < count; i++ {
				w.add(depth+1, "", fmt.Sprintf("iteration %d", i+1), "", 0)
				if err := w.ops(op.Bodies[0], depth+2); err != nil {
					return err
				}
			}
			if count == 0 && len(op.Bodies) == 2 {
				w.add(depth+1, "", "{:else}: the list is empty", "", 0)
				if err := w.ops(op.Bodies[1], depth+2); err != nil {
					return err
				}
			}
		case program.Call:
			child, err := w.nextChild()
			if err != nil {
				return err
			}
			w.add(depth, "c", "component: next nested block", "", 0)
			w.link(child)
		case program.Render:
			value, place, err := w.take()
			if err != nil {
				return err
			}
			switch firstByte(value) {
			case '-':
				w.add(depth, "n", "{@render}: no snippet, so nothing is rendered", value, place)
			case 'c':
				child, err := w.nextChild()
				if err != nil {
					return err
				}
				w.add(depth, "n", "{@render}: snippet with parameters, evaluated here in the next nested block", value, place)
				w.link(child)
			default:
				levels, name, _ := strings.Cut(value[1:], ":")
				where := fmt.Sprintf("its blocks are in the record of the component %s levels up", levels)
				if levels == "0" {
					switch blocks := w.block.Snippet(name); {
					case len(blocks) > 0:
						where = fmt.Sprintf("its blocks %v are in this record", blocks)
					case w.shared:
						where = "its blocks are in the record of the page that shows this shared block"
					}
				}
				what := fmt.Sprintf("{@render %s()}: snippet supplied by the caller; %s", name, where)
				if op.Text == program.Last {
					what += "; only the last block is shown because the page has one <title>"
				}
				w.add(depth, "n", what, value, place)
			}
		case program.Element:
			tag, place, err := w.take()
			if err != nil {
				return err
			}
			if tag == "" {
				w.add(depth, "e", "<svelte:element>: no tag, so nothing is rendered", tag, place)
				continue
			}
			w.add(depth, "e", "<svelte:element>: tag name", tag, place)
			if err := w.ops(op.Bodies[0], depth+1); err != nil {
				return err
			}
			if !program.IsVoid(tag) {
				if err := w.ops(op.Bodies[1], depth+1); err != nil {
					return err
				}
			}
		case program.Mount:
			w.add(depth, "m", fmt.Sprintf("framework insertion point «%s» on every page", op.Text), w.site.Mounts[op.Text], 0)
		}
	}
	return nil
}

// nextChild takes the next block the block shows.
func (w *walker) nextChild() (int, error) {
	if w.child >= len(w.block.Children) {
		return 0, fmt.Errorf("fewer children than calls")
	}
	w.child++
	return w.block.Children[w.child-1], nil
}

// link tells, at the last step, which block it leads to.
func (w *walker) link(child int) {
	last := &w.out[len(w.out)-1]
	if c := &w.rec.Blocks[child]; c.Use {
		last.What += fmt.Sprintf(" — block %d, a reference to shared block %d", child, c.Shared)
		last.Link, last.Ref = fmt.Sprintf("shared-%d", c.Shared), fmt.Sprintf("shared block %d", c.Shared)
	} else {
		last.What += fmt.Sprintf(" — block %d: %s", child, w.site.Files[c.File].Path)
	}
}

func firstByte(s string) byte {
	if s == "" {
		return 0
	}
	return s[0]
}

// opNames explains the operations of a program.
var opNames = map[string]string{
	"r": "markup", "t": "text", "h": "rendered HTML", "a": "attribute", "u": "URL",
	"s": "spread attributes", "i": "conditional", "f": "loop", "c": "component", "n": "{@render}",
	"e": "<svelte:element>", "m": "insertion point", "d": "Svelte island",
}

// opLine is one operation of a program as the report lists it.
type opLine struct {
	Depth int
	Code  string
	Name  string
	Text  string
	Takes int // how many values it takes from the tape, when that is known
}

// listOps writes out a part of a program.
func listOps(ops []program.Op, depth int) []opLine {
	var out []opLine
	for _, op := range ops {
		code := string(rune(op.Kind))
		line := opLine{Depth: depth, Code: code, Name: opNames[code], Text: op.Text}
		switch op.Kind {
		case program.Text, program.HTML, program.Island, program.Attr, program.URL, program.If, program.For, program.Render, program.Element:
			line.Takes = 1
		}
		out = append(out, line)
		for i, body := range op.Bodies {
			label := ""
			switch op.Kind {
			case program.If:
				label = fmt.Sprintf("branch %d", i+1)
			case program.For:
				label = map[int]string{0: "loop body", 1: "{:else}: empty list fallback"}[i]
			case program.Element:
				label = map[int]string{0: "attributes", 1: "content"}[i]
			}
			out = append(out, opLine{Depth: depth + 1, Name: label})
			out = append(out, listOps(body, depth+2)...)
		}
	}
	return out
}

// frame is one message of the conversation between shugo and the
// JavaScript of the project.
type frame struct {
	Kind    string
	Size    int
	Body    string
	Meaning string
	Records []wireRecord
}

// wireRecord is a record as the JavaScript side sends it.
type wireRecord struct {
	Fields []string
	Blocks [][]string
	Raw    string
}

// What the messages mean, by who sends them.
var (
	fromGo = map[byte]string{
		'L': "list pages by route",
		'C': "compute these pages and shared blocks: p or s \x02 URL or name \x02 route",
		'Q': "no more work; exit",
		'K': "from earlier runs: indexed queries and read shapes with their IDs",
		'I': "index response: 1 means built, 0 means a suitable index already existed",
	}
	fromJS = map[byte]string{
		'P': "pages by route: route \x02 URL",
		'G': "reads by the route's getStaticPaths: route \x02 supplies props \x02 reads",
		'S': "new read shape: ID in this exchange \x02 table, lookup columns, relevant columns",
		'I': "build an index: table \x02 for rows (1) or counts (0) \x02 number of lookup columns \x02 columns",
		'R': "records and reads for pages and shared blocks",
		'D': "request completed",
		'E': "project code failed",
	}
)

// frames reads a recorded side of the conversation: a length in four bytes,
// a letter, the message.
func frames(path string, names map[byte]string) ([]frame, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var out []frame
	for len(data) >= 4 {
		size := int(binary.LittleEndian.Uint32(data))
		if size < 1 || 4+size > len(data) {
			break
		}
		body := data[5 : 4+size]
		f := frame{Kind: string(data[4]), Size: size, Body: string(body), Meaning: names[data[4]]}
		if data[4] == 'R' {
			for _, one := range strings.Split(string(body), "\x03") {
				parts := strings.Split(one, "\x01")
				r := wireRecord{Fields: strings.Split(parts[0], "\x02"), Raw: one}
				for _, block := range parts[1:] {
					r.Blocks = append(r.Blocks, strings.Split(block, "\x02"))
				}
				f.Records = append(f.Records, r)
			}
		}
		out = append(out, f)
		data = data[4+size:]
	}
	return out, nil
}

// visible shows the separators of the conversation, and the mark of an
// escaped character, as letters.
func visible(s string) string {
	return strings.NewReplacer("\x00", "␀", "\x01", "␁", "\x02", "␂", "\x03", "␃", "\x04", "␄").Replace(s)
}

func pretty(raw string) string {
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return raw
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	return string(out)
}

func dataKey(key []any) []byte { return data.Key(key) }

// round is one request of Go and what the JavaScript side did for it, told
// by its outcome rather than message by message.
type round struct {
	Ask     string
	Units   []string // the pages and shared blocks Go named
	Pages   []string // for a listing: the pages the routes gave
	Routes  []routeNote
	Records []recordNote
	Named   []string // the shared blocks the records show, each once
	Indexes []indexNote
	Shapes  []string // kinds of readings met for the first time in this run
	Failed  string
	Told    bool // Go only told something: no answer is waited for
	Number  int  // of the requests that are answered, from 1
}

type routeNote struct {
	Route    string
	Props    bool
	Readings []string
}

type recordNote struct {
	Kind, Key string
	Blocks    int
	Shows     []string
	Readings  []string
}

type indexNote struct {
	Table        string
	Equal, Order []string
	Rows, Made   bool
}

// spec is a kind of readings as the JavaScript side tells it.
type spec struct {
	Table   string   `json:"table"`
	By      []string `json:"by"`
	Columns []string `json:"columns"`
}

func (s spec) String() string {
	out := s.Table
	if len(s.By) > 0 {
		out += " by " + strings.Join(s.By, ", ")
	} else {
		out += ", all rows"
	}
	if len(s.Columns) > 0 {
		out += "; relevant columns: " + strings.Join(s.Columns, ", ")
	}
	return out
}

// rounds pairs every request of Go with the messages the JavaScript side
// sent for it, up to the one that says the request is done. The answers of
// Go to requests for indexes come in the order the requests were made.
func rounds(requests, answers []frame) []round {
	var replies []frame
	for _, f := range requests {
		if f.Kind == "I" {
			replies = append(replies, f)
		}
	}
	shapes := map[int]spec{}
	var out []round
	next, asked := 0, 0
	for _, f := range requests {
		var r round
		switch f.Kind {
		case "K":
			var told struct {
				Plans  []string `json:"plans"`
				Shapes [][2]any `json:"shapes"`
			}
			if json.Unmarshal([]byte(f.Body), &told) == nil {
				r.Ask = fmt.Sprintf("from earlier runs: %s already indexed, and %s", counted(len(told.Plans), "query", "queries"), counted(len(told.Shapes), "read shape", "read shapes"))
				r.Told = true
				for _, number := range told.Shapes {
					var s spec
					if text, ok := number[1].(string); ok && json.Unmarshal([]byte(text), &s) == nil {
						n, _ := number[0].(float64)
						shapes[int(n)] = s
					}
				}
			}
			out = append(out, r)
			continue
		case "L":
			r.Ask = "list pages for all routes"
			if f.Body != "" {
				r.Ask = "list pages again for routes " + f.Body
			}
		case "C":
			pages, shared := 0, 0
			for _, unit := range strings.Split(f.Body, "\x03") {
				fields := strings.Split(unit, "\x02")
				if fields[0] == "p" {
					pages++
					r.Units = append(r.Units, "page "+fields[1])
				} else if len(fields) > 1 {
					shared++
					r.Units = append(r.Units, "shared block "+fields[1])
				}
			}
			var what []string
			if pages > 0 {
				what = append(what, counted(pages, "page", "pages"))
			}
			if shared > 0 {
				what = append(what, counted(shared, "shared block", "shared blocks"))
			}
			r.Ask = "compute " + strings.Join(what, " and ")
		default:
			continue
		}
		asked++
		r.Number = asked
		named := map[string]bool{}
		for done := false; !done && next < len(answers); next++ {
			m := answers[next]
			switch m.Kind {
			case "D":
				done = true
			case "E":
				r.Failed, done = m.Body, true
			case "S":
				number, text, _ := strings.Cut(m.Body, "\x02")
				var s spec
				if json.Unmarshal([]byte(text), &s) == nil {
					n := 0
					fmt.Sscan(number, &n)
					shapes[n] = s
					r.Shapes = append(r.Shapes, s.String())
				}
			case "I":
				fields := strings.Split(m.Body, "\x02")
				if len(fields) >= 3 {
					equal := 0
					fmt.Sscan(fields[2], &equal)
					columns := fields[3:]
					equal = min(equal, len(columns))
					note := indexNote{Table: fields[0], Rows: fields[1] == "1", Equal: columns[:equal], Order: columns[equal:]}
					if len(replies) > 0 {
						note.Made, replies = replies[0].Body == "1", replies[1:]
					}
					r.Indexes = append(r.Indexes, note)
				}
			case "P":
				for _, one := range strings.Split(m.Body, "\x03") {
					route, path, _ := strings.Cut(one, "\x02")
					r.Pages = append(r.Pages, path+" (route "+route+")")
				}
			case "G":
				fields := strings.SplitN(m.Body, "\x02", 3)
				if len(fields) == 3 {
					r.Routes = append(r.Routes, routeNote{Route: fields[0], Props: fields[1] == "1", Readings: readingsIn(fields[2], shapes)})
				}
			case "R":
				for _, rec := range m.Records {
					note := recordNote{Kind: "page", Key: rec.Fields[1], Blocks: len(rec.Blocks)}
					if rec.Fields[0] == "s" {
						note.Kind = "shared block"
					}
					if len(rec.Fields) > 4 {
						note.Readings = readingsIn(rec.Fields[4], shapes)
					}
					for _, b := range rec.Blocks {
						if len(b) > 1 && b[0] == "u" {
							note.Shows = append(note.Shows, b[1])
							if !named[b[1]] {
								named[b[1]] = true
								r.Named = append(r.Named, b[1])
							}
						}
					}
					r.Records = append(r.Records, note)
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// counted formats a count with its singular or plural label.
func counted(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// readingsIn writes readings as the JavaScript side sends them: a JSON list
// of the number of a kind and then the values the rows were found by.
func readingsIn(text string, shapes map[int]spec) []string {
	if text == "" {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var list []any
	if decoder.Decode(&list) != nil {
		return []string{text}
	}
	var out []string
	for len(list) > 0 {
		number, _ := list[0].(json.Number)
		n, err := number.Int64()
		if err == nil && n == -1 {
			out = append(out, "first catalog results")
			list = list[1:]
			continue
		}
		s, known := shapes[int(n)]
		if !known || len(list)-1 < len(s.By) {
			return append(out, "?")
		}
		values := make([]string, len(s.By))
		for i := range s.By {
			values[i] = shown(list[1+i])
		}
		one := s.Table + ", all rows"
		if len(s.By) > 0 {
			one = s.Table + " where " + strings.Join(s.By, ", ") + " = " + strings.Join(values, ", ")
		}
		if len(s.Columns) > 0 {
			one += "; relevant columns: " + strings.Join(s.Columns, ", ")
		}
		out = append(out, one)
		list = list[1+len(s.By):]
	}
	return out
}
