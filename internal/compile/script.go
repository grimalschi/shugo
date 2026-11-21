package compile

import (
	"strconv"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/syntax"
)

// script is the code of a <script> after esbuild has taken the types out of
// it. The template is written inside a function that runs once for every
// place the file is shown at, and the script goes into that function with
// it. Imports and exports cannot be inside a function, so they are told apart
// from the rest and stay at the top of the module. All of <script module>
// stays there.
type script struct {
	lines []string       // the JavaScript, by line
	smap  *sourcemap.Map // where every line came from in the .shugo file
	top   []span         // imports and exports, as ranges of lines
	// paths tells that the script exports getStaticPaths.
	paths   bool
	islands map[string]string // explicit default imports of .svelte files
	// at is the <script> in the file, and module tells that it is
	// <script module>.
	at     *syntax.Script
	module bool
}

// span is lines[from:to].
type span struct{ from, to int }

// What makes esbuild keep every import that is not marked as a type: the
// template may use a name the script itself never mentions.
const keepImports = `{"compilerOptions":{"verbatimModuleSyntax":true}}`

// An empty export makes the script a module for esbuild, where a "return"
// outside of a function is an error. Without it the return would end the
// function the script is put into.
const moduleMark = "export {};"

func compileScript(f *syntax.File, at *syntax.Script, module bool) (*script, error) {
	source := f.Source[at.Code.Start:at.Code.End]
	// The script keeps its place in the file, so that esbuild reports lines
	// and columns of the .shugo file.
	padding := strings.Repeat("\n", strings.Count(f.Source[:at.Code.Start], "\n"))
	result := api.Transform(padding+source+"\n"+moduleMark, api.TransformOptions{
		Loader:      api.LoaderTS,
		Target:      api.ESNext,
		Sourcefile:  f.Path,
		Sourcemap:   api.SourceMapExternal,
		TsconfigRaw: keepImports,
		LogLevel:    api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		var errs Errors
		for _, message := range result.Errors {
			errs = append(errs, scriptError(f, at, message))
		}
		return nil, errs
	}
	smap, err := sourcemap.Parse(result.Map)
	if err != nil {
		return nil, err
	}
	s := &script{smap: smap, at: at, module: module}
	code := strings.TrimSuffix(strings.TrimSpace(string(result.Code)), moduleMark)
	if err := s.split(f, strings.TrimSpace(code)); err != nil {
		return nil, err
	}
	if module && len(s.lines) > 0 {
		s.top = []span{{0, len(s.lines)}}
	}
	return s, nil
}

// scriptError turns a message of esbuild about the script into an error that
// points into the .shugo file.
func scriptError(f *syntax.File, at *syntax.Script, message api.Message) *syntax.Error {
	if message.Location == nil {
		return f.Errorf(at.Code.Start, "%s", message.Text)
	}
	offset := offsetOf(f.Source, message.Location.Line-1, message.Location.Column)
	if offset >= at.Code.End {
		// esbuild ran into the mark added after the script: the last
		// statement of the script is not finished.
		end := at.Code.Start + len(strings.TrimRight(f.Source[at.Code.Start:at.Code.End], " \t\r\n"))
		return f.Errorf(end, "the script ends in the middle of a statement: a bracket or a quote is not closed")
	}
	return f.Errorf(offset, "%s", message.Text)
}

// split finds the imports and exports among the statements of code. esbuild
// writes every statement from the start of a line and ends it with ";" or
// with the brace of its body, which is what makes the search this simple.
func (s *script) split(f *syntax.File, code string) error {
	if code == "" {
		return nil
	}
	s.lines = strings.Split(code, "\n")
	lineAt := func(offset int) int { return strings.Count(code[:offset], "\n") }

	scan := syntax.NewScanner(code, 0)
	depth := 0
	after := "" // the token before, to tell `a.import` from an import
	for {
		token, err := next(scan)
		if err != nil {
			return internalError(f, "cannot read the transformed script: %v", err)
		}
		if token.Kind == syntax.EOF {
			return nil
		}
		text := scan.Text(token)
		depth += nesting(token, text)
		statement := depth == 0 && token.Kind == syntax.Word && after != "." && (text == "import" || text == "export")
		after = text
		if !statement {
			continue
		}
		end, err := s.statement(scan, code, text == "export")
		if err != nil {
			return internalError(f, "cannot find the end of an %s in the transformed script: %v", text, err)
		}
		if end < 0 {
			continue // import("module") and import.meta are expressions
		}
		from, to := lineAt(token.Start), lineAt(end-1)+1
		if (token.Start > 0 && code[token.Start-1] != '\n') || (end < len(code) && code[end] != '\n') {
			return internalError(f, "an %s shares a line with other code in the transformed script", text)
		}
		if text == "export" {
			if err := s.exported(f, code[token.End:], from); err != nil {
				return err
			}
		} else if err := s.islandImport(f, code[token.End:end], from); err != nil {
			return err
		}
		s.top = append(s.top, span{from, to})
		*scan = *syntax.NewScanner(code, end)
		after = ""
	}
}

// islandImport finds the local binding of an explicit Svelte island import.
// esbuild has normalized the quotation and removed type-only imports.
func (s *script) islandImport(f *syntax.File, clause string, line int) error {
	if !strings.Contains(clause, ".svelte") {
		return nil
	}
	scan := syntax.NewScanner(clause, 0)
	var tokens []syntax.Token
	for {
		token, err := next(scan)
		if err != nil {
			return internalError(f, "cannot read an island import: %v", err)
		}
		if token.Kind == syntax.EOF || scan.Text(token) == ";" {
			break
		}
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 {
		return nil
	}
	path, _ := strconv.Unquote(scan.Text(tokens[len(tokens)-1]))
	if !strings.HasSuffix(path, ".svelte") {
		return nil
	}
	at := s.at.Start
	if _, l, column, found := s.smap.Find(line, 0); found {
		at = offsetAt(f.Source, l, column)
	}
	if len(tokens) != 3 || tokens[0].Kind != syntax.Word || scan.Text(tokens[1]) != "from" {
		return f.Errorf(at, `import an island as a default component, for example import Gallery from "./Gallery.svelte"`)
	}
	if s.islands == nil {
		s.islands = map[string]string{}
	}
	s.islands[scan.Text(tokens[0])] = path
	return nil
}

// exported checks an export that stands on the given line. The file itself
// is what a .shugo module exports, and only <script module> runs once for
// the module: the other <script> runs every time the component is shown,
// and what it declares belongs to that showing.
func (s *script) exported(f *syntax.File, rest string, line int) error {
	at := s.at.Start
	if _, l, column, found := s.smap.Find(line, 0); found {
		at = offsetAt(f.Source, l, column)
	}
	switch {
	case strings.HasPrefix(rest, " default"):
		return f.Errorf(at, `the script cannot have "export default": the file itself is what it exports`)
	case s.module:
		return nil
	case strings.HasPrefix(rest, " let ") || strings.HasPrefix(rest, " var "):
		return f.Errorf(at, `"export let" declares a prop in Svelte 4: a component of Svelte 5 takes its props with let { name } = $props()`)
	}
	return f.Errorf(at, `an export goes into <script module>, which runs once; this <script> runs every time the component is shown: <script module> export function getStaticPaths() { … } </script>`)
}

// statement reads on from an "import" or "export" and returns the offset
// after the end of the statement, or -1 if the keyword starts an expression
// such as import("module").
func (s *script) statement(scan *syntax.Scanner, code string, export bool) (int, error) {
	if !export {
		peek := *scan
		if token, err := next(&peek); err == nil && (peek.Text(token) == "(" || peek.Text(token) == ".") {
			return -1, nil
		}
	}
	depth := 0
	declares := "" // the keyword of what is exported: "function", "const", …
	inBody := false
	for {
		token, err := next(scan)
		if err != nil {
			return 0, err
		}
		if token.Kind == syntax.EOF {
			return len(code), nil
		}
		text := scan.Text(token)
		switch {
		case token.Kind != syntax.Word || inBody:
		case text == "getStaticPaths":
			s.paths = s.paths || export
		case depth == 0 && declares == "":
			switch text {
			case "function", "class", "const", "let", "var":
				declares = text
			}
		}
		// A function or a class ends with the brace of its body. Everything
		// else, a constant that holds a function too, ends with ";".
		block := declares == "function" || declares == "class"
		if block && depth == 0 && text == "{" {
			inBody = true
		}
		if depth += nesting(token, text); depth != 0 {
			continue
		}
		if (block && inBody && text == "}") || (!block && text == ";") {
			return token.End, nil
		}
	}
}

// next returns the next token that is code.
func next(scan *syntax.Scanner) (syntax.Token, error) {
	for {
		token, err := scan.Next()
		if err != nil || (token.Kind != syntax.Space && token.Kind != syntax.Comment) {
			return token, err
		}
	}
}

// nesting tells by how much a token changes the depth of brackets.
func nesting(token syntax.Token, text string) int {
	if token.Kind != syntax.Punct {
		return 0
	}
	switch text {
	case "(", "[", "{":
		return 1
	case ")", "]", "}":
		return -1
	}
	return 0
}

// body returns the ranges of lines that are not imports or exports.
func (s *script) body() []span {
	var out []span
	at := 0
	for _, top := range s.top {
		if top.from > at {
			out = append(out, span{at, top.from})
		}
		at = top.to
	}
	if at < len(s.lines) {
		out = append(out, span{at, len(s.lines)})
	}
	return out
}

// offsetOf returns the offset of a line (from 0) and a column in bytes.
func offsetOf(source string, line, column int) int {
	offset := 0
	for ; line > 0; line-- {
		end := strings.IndexByte(source[offset:], '\n')
		if end < 0 {
			return len(source)
		}
		offset += end + 1
	}
	return min(offset+column, len(source))
}
