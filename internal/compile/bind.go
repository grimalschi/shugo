package compile

import (
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"

	"github.com/grimalschi/shugo/internal/syntax"
)

// bindings is what the script of a file declares, as far as its template is
// concerned, and what the script does that decides where it may run.
type bindings struct {
	imports map[string]bool // the names the script imports
	vars    map[string]bool // the variables the script declares, exported ones too
	funcs   map[string]bool // those of them declared as a function or a class
	// waits tells that the script waits for something: it has await outside
	// of the functions it declares. now tells that it asks what time it is.
	waits, now bool
}

// bind reads the script that esbuild has taken the types out of.
func bind(f *syntax.File, s *script) (*bindings, error) {
	b := &bindings{imports: map[string]bool{}, vars: map[string]bool{}, funcs: map[string]bool{}}
	if len(s.lines) == 0 {
		return b, nil
	}
	ast, err := js.Parse(parse.NewInputString(strings.Join(s.lines, "\n")), js.Options{})
	if err != nil {
		return nil, f.Errorf(f.Script.Start, "the script uses syntax that Shugo cannot read yet (declarations with `using` and decorators are among it): %s", firstLine(err.Error()))
	}
	for _, statement := range ast.List {
		switch n := statement.(type) {
		case *js.ImportStmt:
			if n.Default != nil {
				b.imports[string(n.Default)] = true
			}
			for _, alias := range n.List {
				switch {
				case alias.Binding != nil:
					b.imports[string(alias.Binding)] = true
				case alias.Name != nil:
					b.imports[string(alias.Name)] = true
				}
			}
		case *js.ExportStmt:
			// What an exported function does happens when the framework
			// calls it, not when the script runs.
			b.declared(n.Decl)
		default:
			b.declared(statement)
			js.Walk(&scriptWalk{b: b}, statement)
		}
	}
	// A var inside a block belongs to the whole script as well.
	for _, v := range ast.Scope.Declared {
		b.vars[string(v.Data)] = true
	}
	return b, nil
}

// declared notes a declaration that makes a function or a class.
func (b *bindings) declared(n js.INode) {
	switch n := n.(type) {
	case *js.FuncDecl:
		if n.Name != nil {
			b.funcs[string(n.Name.Data)] = true
		}
	case *js.ClassDecl:
		if n.Name != nil {
			b.funcs[string(n.Name.Data)] = true
		}
	case *js.VarDecl:
		for _, element := range n.List {
			name, plain := element.Binding.(*js.Var)
			if !plain {
				continue
			}
			switch element.Default.(type) {
			case *js.ArrowFunc, *js.FuncDecl, *js.ClassDecl:
				b.funcs[string(name.Data)] = true
			}
		}
	}
}

// scriptWalk looks through the statements of a script for what decides
// where it may run.
type scriptWalk struct {
	b      *bindings
	inside int // how deep in the functions the script declares
}

func (w *scriptWalk) Enter(n js.INode) js.IVisitor {
	switch n := n.(type) {
	case *js.FuncDecl, *js.ArrowFunc, *js.MethodDecl:
		w.inside++
	case *js.UnaryExpr:
		w.b.waits = w.b.waits || (n.Op == js.AwaitToken && w.inside == 0)
	case *js.ForOfStmt:
		w.b.waits = w.b.waits || (n.Await && w.inside == 0)
	case *js.NewExpr:
		w.b.now = w.b.now || asksTheTime(n)
	}
	return w
}

func (w *scriptWalk) Exit(n js.INode) {
	switch n.(type) {
	case *js.FuncDecl, *js.ArrowFunc, *js.MethodDecl:
		w.inside--
	}
}

// asksTheTime tells whether an expression is new Date() without arguments.
func asksTheTime(n *js.NewExpr) bool {
	name, isFree := freeName(n.X)
	return isFree && name == "Date" && (n.Args == nil || len(n.Args.List) == 0)
}

// freeName returns the name an expression is, if it is nothing but a name
// that the code around it does not declare.
func freeName(e js.IExpr) (string, bool) {
	v, isVar := unwrap(e).(*js.Var)
	if !isVar {
		return "", false
	}
	for v.Link != nil {
		v = v.Link
	}
	return string(v.Data), v.Decl == js.NoDecl
}

// unwrap takes the brackets off an expression.
func unwrap(e js.IExpr) js.IExpr {
	for {
		group, ok := e.(*js.GroupExpr)
		if !ok {
			return e
		}
		e = group.X
	}
}

// A piece of code from a template is wrapped into a call before it is read,
// so that it is read as an expression whatever it starts with. The code
// stands on lines of its own, which keeps the places esbuild reports simple.
const (
	wrapStart = "__e((\n"
	wrapEnd   = "\n))"
	wrapLines = 2 // lines before the code
)

// parseCode reads a piece of the project's JavaScript written in a
// template: esbuild takes the types out of it, then the tree is made. before
// and after are put around the code to make it an expression of the kind
// wanted; the tree is of everything inside the wrapping.
func parseCode(f *syntax.File, c code, before, after string) (js.IExpr, *syntax.Error) {
	result := api.Transform(wrapStart+before+"\n"+c.text+"\n"+after+wrapEnd, api.TransformOptions{
		Loader:   api.LoaderTS,
		Target:   api.ESNext,
		Charset:  api.CharsetUTF8,
		LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		message := result.Errors[0]
		at := c.start
		if where := message.Location; where != nil && where.Line > wrapLines {
			at += min(offsetOf(c.text, where.Line-wrapLines-1, where.Column), len(c.text))
		}
		return nil, f.Errorf(at, "%s", message.Text)
	}
	ast, err := js.Parse(parse.NewInputString(string(result.Code)), js.Options{})
	if err != nil {
		return nil, f.Errorf(c.start, "this expression uses syntax that Shugo cannot read yet: %s", firstLine(err.Error()))
	}
	if len(ast.List) == 1 {
		if statement, ok := ast.List[0].(*js.ExprStmt); ok {
			if call, ok := statement.Value.(*js.CallExpr); ok && len(call.Args.List) == 1 && !call.Args.List[0].Rest {
				return call.Args.List[0].Value, nil
			}
		}
	}
	return nil, f.Errorf(c.start, "this is not one expression")
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}
