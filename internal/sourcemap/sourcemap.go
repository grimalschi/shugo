// Package sourcemap reads and writes source maps (version 3): the tables that
// say which place of a source file a place of generated code came from. The
// compiler writes one for every module it generates, and an error at run time
// is traced back to the template through the map of the whole bundle.
package sourcemap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Segment ties a column of a generated line to a place in a source file.
type Segment struct {
	Column       int // in the generated line, from 0
	Source       int // which of Map.Sources; -1 when the column comes from no source
	SourceLine   int // from 0
	SourceColumn int // from 0
}

// Map is a source map with its segments decoded, by generated line.
type Map struct {
	Sources []string
	Lines   [][]Segment
}

type file struct {
	Version  int      `json:"version"`
	Sources  []string `json:"sources"`
	Mappings string   `json:"mappings"`
	Names    []string `json:"names"`
}

// Parse reads a source map.
func Parse(data []byte) (*Map, error) {
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("sourcemap: %w", err)
	}
	m := &Map{Sources: f.Sources}
	// Every field except the generated column goes on from its last value.
	source, sourceLine, sourceColumn := 0, 0, 0
	for _, line := range strings.Split(f.Mappings, ";") {
		var segments []Segment
		column := 0
		for line != "" {
			var text string
			text, line, _ = strings.Cut(line, ",")
			fields, err := decode(text)
			if err != nil {
				return nil, err
			}
			switch len(fields) {
			case 1:
				column += fields[0]
				segments = append(segments, Segment{Column: column, Source: -1})
			case 4, 5:
				column += fields[0]
				source += fields[1]
				sourceLine += fields[2]
				sourceColumn += fields[3]
				segments = append(segments, Segment{column, source, sourceLine, sourceColumn})
			default:
				return nil, fmt.Errorf("sourcemap: a segment of %d fields", len(fields))
			}
		}
		m.Lines = append(m.Lines, segments)
	}
	return m, nil
}

// Find returns the place a generated place came from: the segment at or
// before the column.
func (m *Map) Find(line, column int) (source string, sourceLine, sourceColumn int, found bool) {
	if line < 0 || line >= len(m.Lines) {
		return "", 0, 0, false
	}
	segments := m.Lines[line]
	i := sort.Search(len(segments), func(i int) bool { return segments[i].Column > column }) - 1
	if i < 0 || segments[i].Source < 0 || segments[i].Source >= len(m.Sources) {
		return "", 0, 0, false
	}
	return m.Sources[segments[i].Source], segments[i].SourceLine, segments[i].SourceColumn, true
}

// JSON writes the map.
func (m *Map) JSON() []byte {
	var mappings strings.Builder
	source, sourceLine, sourceColumn := 0, 0, 0
	for i, segments := range m.Lines {
		if i > 0 {
			mappings.WriteByte(';')
		}
		column := 0
		for j, s := range segments {
			if j > 0 {
				mappings.WriteByte(',')
			}
			encode(&mappings, s.Column-column)
			column = s.Column
			if s.Source < 0 {
				continue
			}
			encode(&mappings, s.Source-source)
			encode(&mappings, s.SourceLine-sourceLine)
			encode(&mappings, s.SourceColumn-sourceColumn)
			source, sourceLine, sourceColumn = s.Source, s.SourceLine, s.SourceColumn
		}
	}
	data, _ := json.Marshal(file{Version: 3, Sources: m.Sources, Mappings: mappings.String(), Names: []string{}})
	return data
}

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// A number is written five bits at a time, the lowest first; the sixth bit of
// a digit says that more follow, and the lowest bit of the number is its sign.
func encode(out *strings.Builder, n int) {
	v := n << 1
	if n < 0 {
		v = -n<<1 | 1
	}
	for {
		digit := v & 31
		if v >>= 5; v > 0 {
			digit |= 32
		}
		out.WriteByte(alphabet[digit])
		if v == 0 {
			return
		}
	}
}

func decode(text string) ([]int, error) {
	var fields []int
	value, shift := 0, 0
	for i := 0; i < len(text); i++ {
		digit := strings.IndexByte(alphabet, text[i])
		if digit < 0 || shift > 60 {
			return nil, fmt.Errorf("sourcemap: damaged segment %q", text)
		}
		value |= (digit & 31) << shift
		if digit&32 != 0 {
			shift += 5
			continue
		}
		if value&1 != 0 {
			value = -(value >> 1)
		} else {
			value >>= 1
		}
		fields = append(fields, value)
		value, shift = 0, 0
	}
	if shift != 0 {
		return nil, fmt.Errorf("sourcemap: damaged segment %q", text)
	}
	return fields, nil
}
