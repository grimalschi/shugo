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
			take(16, "  values: tape hash, 16 bytes")
			list("nested blocks")
		}
		slots := number("  filled slots: %d")
		for s := 0; s < slots; s++ {
			size := number("  slot name length: %d")
			take(size, "  slot name: "+string(data[at:min(at+size, len(data))]))
			list("blocks in the slot")
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
	Slot  int    // the place of the value in the tape, from 1; 0 for none
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
	value, slot, err := w.take()
	if err != nil {
		return 0, 0, err
	}
	var n int
	_, err = fmt.Sscan(value, &n)
	return n, slot, err
}

func (w *walker) add(depth int, op, what, value string, slot int) {
	w.out = append(w.out, walkStep{Depth: depth, Op: op, What: what, Value: value, Slot: slot})
}

func (w *walker) ops(ops []program.Op, depth int) error {
	for _, op := range ops {
		switch op.Kind {
		case program.Markup:
			w.add(depth, "r", "unchanged markup", op.Text, 0)
		case program.Text, program.HTML:
			value, slot, err := w.take()
			if err != nil {
				return err
			}
			what := "text: escaped"
			if op.Kind == program.HTML {
				what = "rendered HTML: inserted as-is"
			}
			w.add(depth, string(rune(op.Kind)), what, value, slot)
		case program.Attr, program.URL:
			value, slot, err := w.take()
			if err != nil {
				return err
			}
			sign := map[byte]string{'=': "attribute with a value", '-': "no attribute", '+': "attribute name only"}[firstByte(value)]
			what := fmt.Sprintf("attribute %s — %s", op.Text, sign)
			if op.Kind == program.URL {
				what += "; the URL is validated"
			}
			w.add(depth, string(rune(op.Kind)), what, value, slot)
		case program.Spread:
			count, slot, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "s", "spread attributes: number of name–value pairs", fmt.Sprint(count), slot)
			for range count {
				name, s1, err := w.take()
				if err != nil {
					return err
				}
				value, s2, err := w.take()
				if err != nil {
					return err
				}
				w.add(depth+1, "s", "name", name, s1)
				w.add(depth+1, "s", "value", value, s2)
			}
		case program.If:
			taken, slot, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "i", fmt.Sprintf("conditional with %d branches: selected branch number, or 0 if none", len(op.Bodies)), fmt.Sprint(taken), slot)
			if taken > 0 {
				if err := w.ops(op.Bodies[taken-1], depth+1); err != nil {
					return err
				}
			}
		case program.For:
			count, slot, err := w.number()
			if err != nil {
				return err
			}
			w.add(depth, "f", "loop: iteration count", fmt.Sprint(count), slot)
			for i := 0; i < count; i++ {
				w.add(depth+1, "", fmt.Sprintf("iteration %d", i+1), "", 0)
				if err := w.ops(op.Bodies[0], depth+2); err != nil {
					return err
				}
			}
		case program.Call:
			if w.child >= len(w.block.Children) {
				return fmt.Errorf("fewer children than calls")
			}
			child := w.block.Children[w.child]
			w.child++
			w.add(depth, "c", "component: next nested block", "", 0)
			last := &w.out[len(w.out)-1]
			if c := &w.rec.Blocks[child]; c.Use {
				last.What += fmt.Sprintf(" — block %d, a reference to shared block %d", child, c.Shared)
				last.Link, last.Ref = fmt.Sprintf("shared-%d", c.Shared), fmt.Sprintf("shared block %d", c.Shared)
			} else {
				last.What += fmt.Sprintf(" — block %d: %s", child, w.site.Files[c.File].Path)
			}
		case program.Slot:
			filled, slot, err := w.number()
			if err != nil {
				return err
			}
			if filled == 0 {
				w.add(depth, "o", fmt.Sprintf("slot «%s»: 1 means filled, 0 means the default content is shown", op.Text), "0", slot)
				if err := w.ops(op.Bodies[0], depth+1); err != nil {
					return err
				}
				continue
			}
			where := "provided by the caller"
			if fills := w.block.Slot(op.Text); len(fills) > 0 {
				where = fmt.Sprintf("the supplied blocks %v are in this record", fills)
			} else if w.shared {
				where = "provided by the page that shows this shared block; the blocks are in that page record"
			}
			w.add(depth, "o", fmt.Sprintf("slot «%s»: %s", op.Text, where), "1", slot)
		case program.Mount:
			w.add(depth, "m", fmt.Sprintf("framework insertion point «%s» on every page", op.Text), w.site.Mounts[op.Text], 0)
		}
	}
	return nil
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
	"s": "spread attributes", "i": "conditional", "f": "loop", "c": "component", "o": "slot", "m": "insertion point",
}

// opLine is one operation of a program as the report lists it.
type opLine struct {
	Depth int
	Code  string
	Name  string
	Text  string
	Slots int // how many values it takes from the tape, when that is known
}

// listOps writes out a part of a program.
func listOps(ops []program.Op, depth int) []opLine {
	var out []opLine
	for _, op := range ops {
		code := string(rune(op.Kind))
		line := opLine{Depth: depth, Code: code, Name: opNames[code], Text: op.Text}
		switch op.Kind {
		case program.Text, program.HTML, program.Attr, program.URL, program.If, program.For, program.Slot:
			line.Slots = 1
		}
		out = append(out, line)
		for i, body := range op.Bodies {
			label := ""
			switch op.Kind {
			case program.If:
				label = fmt.Sprintf("branch %d", i+1)
			case program.For:
				label = "loop body"
			case program.Slot:
				label = "default content"
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
		'C': "compute these pages and shared blocks: p or s ␂ URL or name ␂ route",
		'Q': "no more work; exit",
		'I': "index response: 1 means built, 0 means a suitable index already existed",
	}
	fromJS = map[byte]string{
		'P': "pages by route: route ␂ URL",
		'G': "reads by the route's getStaticPaths: route ␂ supplies props ␂ reads",
		'S': "new read shape: ID in this exchange ␂ table, lookup columns, relevant columns",
		'I': "build an index: table ␂ for rows (1) or counts (0) ␂ number of lookup columns ␂ columns",
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
		f := frame{Kind: string(data[4]), Size: size, Body: visible(string(body)), Meaning: names[data[4]]}
		if data[4] == 'R' {
			for _, one := range strings.Split(string(body), "\x03") {
				parts := strings.Split(one, "\x01")
				r := wireRecord{Fields: strings.Split(parts[0], "\x02"), Raw: visible(one)}
				for _, block := range parts[1:] {
					fields := strings.Split(block, "\x02")
					for i := range fields {
						fields[i] = visible(fields[i])
					}
					r.Blocks = append(r.Blocks, fields)
				}
				f.Records = append(f.Records, r)
			}
		}
		out = append(out, f)
		data = data[4+size:]
	}
	return out, nil
}

// visible shows the separators of the conversation as letters.
func visible(s string) string {
	return strings.NewReplacer("\x00", "␀", "\x01", "␁", "\x02", "␂", "\x03", "␃").Replace(s)
}

// bundled is a record with its values inside, as proposed: the record, then
// every distinct values of its blocks with their name.
func bundled(raw []byte, rec *record.Record, values map[record.Hash][]byte) ([]byte, []byteRow) {
	var out []byte
	var rows []byteRow
	add := func(b []byte, meaning string) {
		rows = append(rows, byteRow{len(out), spaced(b), meaning})
		out = append(out, b...)
	}
	add(binary.AppendUvarint(nil, uint64(len(raw))), fmt.Sprintf("record length: %d bytes", len(raw)))
	add(raw, "current record: blocks and references to values and shared blocks")
	var hashes []record.Hash
	seen := map[record.Hash]bool{}
	for i := range rec.Blocks {
		if b := &rec.Blocks[i]; !b.Use && !seen[b.Values] {
			seen[b.Values] = true
			hashes = append(hashes, b.Values)
		}
	}
	add(binary.AppendUvarint(nil, uint64(len(hashes))), fmt.Sprintf("value tapes included: %d", len(hashes)))
	for _, h := range hashes {
		data := values[h]
		add(h[:], "value tape hash, which the block uses to locate its tape")
		add(binary.AppendUvarint(nil, uint64(len(data))), fmt.Sprintf("value tape length: %d bytes", len(data)))
		add(data, "value tape: "+visible(string(data)))
	}
	return out, rows
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
