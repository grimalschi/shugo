package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/grimalschi/shugo/internal/data"
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
func annotate(data []byte) []byteRow {
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
	text := func(meaning string) {
		size := number(meaning + ": %d bytes")
		if size > 0 {
			take(size, "  «"+visible(string(data[at:min(at+size, len(data))]))+"»")
		}
	}
	fragment := func(what string) {
		holes := number(what + ": holes in the HTML: %d")
		for h := 0; h < holes && at < len(data); h++ {
			text(fmt.Sprintf("  text %d", h+1))
			switch kind := data[at]; kind {
			case 'c':
				take(1, fmt.Sprintf("  hole %d: c is a block in this record", h+1))
				number("    block number: %d")
			case 'g':
				take(1, fmt.Sprintf("  hole %d: g is a snippet supplied at the call site", h+1))
				text("    snippet name")
			case 'i':
				take(1, fmt.Sprintf("  hole %d: i is an island", h+1))
				text("    island JSON")
			default:
				take(len(data)-at, "corrupt data")
			}
		}
		text(fmt.Sprintf("  final text %d", holes+1))
	}
	number("record starts at block %d")
	count := number("blocks in the record: %d")
	for i := 0; i < count && at < len(data); i++ {
		switch data[at] {
		case 'u':
			take(1, fmt.Sprintf("block %d: u is a reference to a shared block", i))
			number("  shared block number: %d")
			snippets := number("  parameterless snippets supplied at the call site: %d")
			for s := 0; s < snippets; s++ {
				text("  snippet name")
				blocks := number("  block count: %d")
				for b := 0; b < blocks; b++ {
					number("    block number: %d")
				}
			}
		case 'b':
			take(1, fmt.Sprintf("block %d: b is HTML", i))
			fragment("  body")
			fragment("  in <head>")
			text("  <title>")
		default:
			take(len(data)-at, "corrupt data")
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

// piece is a text of HTML or a hole in it, as the stand shows it.
type piece struct {
	Hole bool
	Text string // the HTML, or what fills the hole
	Link string // where what fills the hole is told about
}

// pieces tells a fragment piece by piece. rec is the record it is in: a hole
// for a block of it names the shared block that block stands for.
func pieces(f record.Fragment, rec *record.Record) []piece {
	var out []piece
	for i, text := range f.Texts {
		if len(text) > 0 {
			out = append(out, piece{Text: string(text)})
		}
		if i == len(f.Holes) {
			break
		}
		switch hole := f.Holes[i]; hole.Kind {
		case record.Child:
			p := piece{Hole: true, Text: fmt.Sprintf("block %d", hole.Block)}
			if hole.Block < len(rec.Blocks) && rec.Blocks[hole.Block].Use {
				shared := rec.Blocks[hole.Block].Shared
				p.Text += fmt.Sprintf(": shared block %d", shared)
				p.Link = fmt.Sprintf("shared-%d", shared)
			}
			out = append(out, p)
		case record.Given:
			out = append(out, piece{Hole: true, Text: "snippet " + string(hole.Text) + " from the call site"})
		case record.Island:
			out = append(out, piece{Hole: true, Text: "island " + string(hole.Text)})
		}
	}
	return out
}

// fragmentText writes a fragment in one line, with its holes in brackets.
func fragmentText(f record.Fragment) string {
	var out strings.Builder
	for i, text := range f.Texts {
		out.Write(text)
		if i == len(f.Holes) {
			break
		}
		switch hole := f.Holes[i]; hole.Kind {
		case record.Child:
			fmt.Fprintf(&out, "⟨block %d⟩", hole.Block)
		case record.Given:
			fmt.Fprintf(&out, "⟨snippet %s⟩", hole.Text)
		case record.Island:
			out.WriteString("⟨island⟩")
		}
	}
	return out.String()
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
		'G': "reads by the route's getStaticPaths: route \x02 reads",
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
				if route, reads, found := strings.Cut(m.Body, "\x02"); found {
					r.Routes = append(r.Routes, routeNote{Route: route, Readings: readingsIn(reads, shapes)})
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
