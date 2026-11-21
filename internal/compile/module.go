package compile

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/syntax"
)

// module is the JavaScript module written for one .shugo file, line by line,
// with the place in the file every line came from. The places serve twice:
// they turn an error of the bundler into an error in the template, and they
// become the source map that does the same for an error at run time.
type module struct {
	file  *syntax.File
	lines []string
	from  []origin
	// page tells that some code reads what differs from page to page:
	// Shugo.url, Shugo.params. Such a file cannot be computed once for all pages.
	page bool
	// last is where the latest code of the project ended. A mistake there
	// usually shows up in the compiler's own line right after it.
	last int
}

// origin is where a line of the module came from.
type origin struct {
	kind   byte // 0 for a line of the compiler's own, 't' for the template, 's' for the script
	offset int  // 't': where the line starts in the file; 0: the place to blame
	line   int  // 's': the line of the transformed script
}

// add writes lines of the compiler's own.
func (m *module) add(format string, arguments ...any) {
	for _, line := range strings.Split(fmt.Sprintf(format, arguments...), "\n") {
		m.lines = append(m.lines, line)
		m.from = append(m.from, origin{offset: m.last})
	}
}

// code writes code of the template on lines of its own, exactly as written.
func (m *module) code(c code) {
	m.page = m.page || readsPage(c.text)
	offset := c.start
	for _, line := range strings.SplitAfter(c.text, "\n") {
		m.lines = append(m.lines, strings.TrimSuffix(line, "\n"))
		m.from = append(m.from, origin{kind: 't', offset: offset})
		offset += len(line)
	}
	m.last = offset
}

// script writes lines of the transformed script.
func (m *module) script(s *script, lines span) {
	for i := lines.from; i < lines.to; i++ {
		m.lines = append(m.lines, s.lines[i])
		m.from = append(m.from, origin{kind: 's', line: i})
	}
}

// locate returns the offset in the file that a place in the module came from.
func (m *module) locate(s *script, line, column int) int {
	if line < 0 || line >= len(m.from) {
		return 0
	}
	switch from := m.from[line]; from.kind {
	case 't':
		return from.offset + min(column, len(m.lines[line]))
	case 's':
		text := m.lines[line]
		_, sourceLine, sourceColumn, found := s.smap.Find(from.line, utf16Length(text[:min(column, len(text))]))
		if !found {
			return m.file.Script.Start
		}
		return offsetAt(m.file.Source, sourceLine, sourceColumn)
	default:
		return from.offset
	}
}

// text returns the module with its source map attached.
func (m *module) text(s *script) string {
	smap := &sourcemap.Map{Sources: []string{pathBase(m.file.Path)}, Lines: make([][]sourcemap.Segment, len(m.lines))}
	for i, from := range m.from {
		switch from.kind {
		case 't':
			smap.Lines[i] = m.segments(m.lines[i], from.offset)
		case 's':
			if from.line < len(s.smap.Lines) {
				smap.Lines[i] = s.smap.Lines[from.line]
			}
		}
	}
	return strings.Join(m.lines, "\n") + "\n//# sourceMappingURL=data:application/json;base64," +
		base64.StdEncoding.EncodeToString(smap.JSON()) + "\n"
}

// segments ties every token of a line of template code to its place in the file.
func (m *module) segments(line string, offset int) []sourcemap.Segment {
	sourceLine := strings.Count(m.file.Source[:offset], "\n")
	lineStart := strings.LastIndexByte(m.file.Source[:offset], '\n') + 1
	before := utf16Length(m.file.Source[lineStart:offset])
	var out []sourcemap.Segment
	scan := syntax.NewScanner(line, 0)
	for {
		token, err := scan.Next()
		if err != nil || token.Kind == syntax.EOF {
			return out
		}
		if token.Kind == syntax.Space || token.Kind == syntax.Comment {
			continue
		}
		column := utf16Length(line[:token.Start])
		out = append(out, sourcemap.Segment{Column: column, SourceLine: sourceLine, SourceColumn: before + column})
	}
}

// readsPage tells whether code uses Shugo for more than the parameters, the
// slots, the component itself and the data database: Shugo.url, Shugo.params,
// Shugo.paginate, or Shugo as a value.
func readsPage(text string) bool {
	if !strings.Contains(text, "Shugo") {
		return false
	}
	scan := syntax.NewScanner(text, 0)
	after := ""
	for {
		token, err := next(scan)
		if err != nil {
			return true
		}
		if token.Kind == syntax.EOF {
			return false
		}
		word := scan.Text(token)
		if word == "Shugo" && after != "." {
			peek := *scan
			dot, _ := next(&peek)
			member, _ := next(&peek)
			switch name := peek.Text(member); {
			case peek.Text(dot) != ".":
				return true
			case name != "props" && name != "slots" && name != "self" && name != "query":
				return true
			}
		}
		after = word
	}
}

// Columns of a source map count UTF-16 units, as JavaScript does.
func utf16Length(text string) int {
	n := 0
	for _, r := range text {
		if n++; r >= 0x10000 {
			n++
		}
	}
	return n
}

// offsetAt returns the offset of a line (from 0) and a column in UTF-16 units.
func offsetAt(source string, line, column int) int {
	offset := offsetOf(source, line, 0)
	for column > 0 && offset < len(source) && source[offset] != '\n' {
		r, size := utf8.DecodeRuneInString(source[offset:])
		offset += size
		if column--; r >= 0x10000 {
			column--
		}
	}
	return offset
}

func pathBase(path string) string {
	return path[strings.LastIndexByte(path, '/')+1:]
}
