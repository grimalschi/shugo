package compile

import (
	"github.com/tdewolff/parse/v2/js"

	"github.com/grimalschi/shugo/internal/syntax"
)

// reads is the dependency tree of a template: what it reads of the
// variables of the script, of the values the script imports, and of the
// parameters of the file. The template is given exactly this and nothing
// else, so it cannot depend on what is not here.
type reads struct {
	names  []string // variables and imported values, in the order the template meets them
	shapes []*shape // what it reads of each
	props  *shape   // what it reads of Shugo.props; nil if nothing
}

// reading is how an expression uses the value of one of its parts.
type reading int

const (
	asValue reading = iota // the value itself goes somewhere
	asTruth                // only whether there is a value matters
)

// analyzer reads the expressions of one template and notes what they read.
//
// The rules lean to the safe side. A chain of members, item.price, reads
// that member. A condition reads whether the value is there. A loop over
// such a chain reads what its body reads of every item. Whatever else is
// done with a value — it is passed to a function, a method of it is called,
// it is indexed, spread, compared — reads the value whole.
type analyzer struct {
	file   *syntax.File
	script *bindings
	// loops are the names that the <g-for> tags around the place being read
	// give to their items, the innermost last. A name stands for the items
	// of a list the template reads, or for nothing that is followed.
	loops []map[string]*shape

	order  []string
	shapes map[string]*shape
	props  *shape
	// called are the imports the template calls or shows as components:
	// they are code, which the template imports itself.
	called map[string]bool

	at     int // where the code being read starts
	blamed map[string]bool
	errs   Errors
}

// analyze finds what the template of a file reads.
func analyze(f *syntax.File, script *bindings, nodes []node) (*reads, Errors) {
	a := &analyzer{file: f, script: script, shapes: map[string]*shape{}, called: map[string]bool{}, blamed: map[string]bool{}}
	a.nodes(nodes)
	out := &reads{props: a.props}
	for _, name := range a.order {
		// An import that is called anywhere is code everywhere: the
		// template imports it, and whatever it reads of it is not data.
		if !a.called[name] {
			a.shapes[name].settle()
			out.names = append(out.names, name)
			out.shapes = append(out.shapes, a.shapes[name])
		}
	}
	if out.props != nil {
		out.props.settle()
	}
	return out, a.errs
}

func (a *analyzer) errorf(format string, arguments ...any) {
	a.errs = append(a.errs, a.file.Errorf(a.at, format, arguments...))
}

// nodes reads the expressions of a piece of a template.
func (a *analyzer) nodes(list []node) {
	for _, n := range list {
		switch n := n.(type) {
		case output:
			a.code(n.expr, asValue)
		case attribute:
			a.code(n.expr, asValue)
		case spread:
			a.code(n.expr, asValue)
		case choice:
			for _, b := range n.branches {
				if b.test != nil {
					a.code(*b.test, asTruth)
				}
				a.nodes(b.body)
			}
		case loop:
			a.loop(n)
		case call:
			a.tag(n.target)
			for _, p := range n.props {
				if p.expr != nil {
					a.code(*p.expr, asValue)
				}
			}
			for _, f := range n.fills {
				a.nodes(f.body)
			}
		case outlet:
			a.nodes(n.fallback)
		case provide:
			a.nodes(n.body)
		}
	}
}

// code reads one expression of the template.
func (a *analyzer) code(c code, how reading) {
	a.at = c.start
	e, err := parseCode(a.file, c, "", "")
	if err != nil {
		a.errs = append(a.errs, err)
		return
	}
	a.read(e, how)
}

// loop reads <g-for>. Over a chain of members it reads, of every item, what
// the body reads of the name given to the item.
func (a *analyzer) loop(n loop) {
	names := map[string]*shape{}
	a.at = n.each.start
	each, failed := parseCode(a.file, n.each, "", "")
	if failed != nil {
		a.errs = append(a.errs, failed)
	}
	var items *shape
	if each != nil {
		if list, followed := a.chain(each); !followed {
			a.read(each, asValue)
		} else if list != nil {
			items = list.item()
		}
	}

	a.at = n.item.start
	pattern, failed := parseCode(a.file, n.item, "(", ") => 0")
	if failed != nil {
		a.errs = append(a.errs, failed)
	} else if arrow, ok := unwrap(pattern).(*js.ArrowFunc); ok && len(arrow.Params.List) == 1 && arrow.Params.Rest == nil {
		if name, plain := arrow.Params.List[0].Binding.(*js.Var); plain && arrow.Params.List[0].Default == nil {
			names[string(name.Data)] = items
		} else {
			// The item is taken apart at once: what is taken of it is not
			// followed, so every item is read whole.
			if items != nil {
				items.all()
			}
			bound(arrow.Params.List[0].Binding, names)
		}
	}
	if n.index != nil {
		names[n.index.text] = nil
	}

	a.loops = append(a.loops, names)
	a.nodes(n.body)
	a.loops = a.loops[:len(a.loops)-1]
}

// bound collects the names that a pattern gives to the parts of a value.
func bound(pattern js.IBinding, names map[string]*shape) {
	switch p := pattern.(type) {
	case *js.Var:
		names[string(p.Data)] = nil
	case *js.BindingArray:
		for _, element := range p.List {
			bound(element.Binding, names)
		}
		bound(p.Rest, names)
	case *js.BindingObject:
		for _, item := range p.List {
			bound(item.Value.Binding, names)
		}
		if p.Rest != nil {
			names[string(p.Rest.Data)] = nil
		}
	}
}

// tag reads the name of a component: <Card>, <ui.Button>. A component the
// script imports is code. A tag may also name a variable that holds a
// component, chosen by the script: that one is read as a value.
func (a *analyzer) tag(target code) {
	a.at = target.start
	e, err := parseCode(a.file, target, "", "")
	if err != nil {
		a.errs = append(a.errs, err)
		return
	}
	root := unwrap(e)
	for {
		dot, more := root.(*js.DotExpr)
		if !more {
			break
		}
		root = unwrap(dot.X)
	}
	if name, isFree := freeName(root); isFree && a.script.imports[name] && !a.isBound(name) {
		a.called[name] = true
		return
	}
	a.read(e, asValue)
}

// isBound tells whether a <g-for> around gives the name to its items.
func (a *analyzer) isBound(name string) bool {
	for _, names := range a.loops {
		if _, bound := names[name]; bound {
			return true
		}
	}
	return false
}

// named returns what the template reads of a name: the shape of the items
// of a loop, of a variable of the script or of an imported value. It
// returns nil for what is not followed: an item taken apart, a number of a
// loop, a name the template does not get from the script.
func (a *analyzer) named(name string) *shape {
	for i := len(a.loops) - 1; i >= 0; i-- {
		if s, bound := a.loops[i][name]; bound {
			return s
		}
	}
	switch {
	case a.script.funcs[name]:
		if !a.blamed[name] {
			a.blamed[name] = true
			a.errorf("the template uses %s, a function that the script declares. A template sees the data of the script, not its code: move the function to a module and import it, or call it in the script and show the result", name)
		}
		return nil
	case !a.script.vars[name] && !a.script.imports[name]:
		return nil
	}
	if s, read := a.shapes[name]; read {
		return s
	}
	s := &shape{}
	a.shapes[name] = s
	a.order = append(a.order, name)
	return s
}

// parameters returns the shape of Shugo.props.
func (a *analyzer) parameters() *shape {
	if a.props == nil {
		a.props = &shape{}
	}
	return a.props
}

// chain follows a chain of members, item.discount.percent, from the name it
// starts at, and returns the shape of what it ends at. followed tells that
// the chain is accounted for; the shape is nil where it ends at something
// the template is not given by the script: a global, the page, a name of its
// own. A chain that is not followed starts at something computed, and the
// caller reads that.
func (a *analyzer) chain(e js.IExpr) (s *shape, followed bool) {
	switch n := unwrap(e).(type) {
	case *js.Var:
		name, isFree := freeName(n)
		switch {
		case !isFree:
			return nil, true
		case name == "Shugo" && !a.isBound(name):
			return a.parameters(), true
		}
		return a.named(name), true
	case *js.DotExpr:
		var member string
		switch y := n.Y.(type) {
		case js.LiteralExpr:
			member = string(y.Data)
		case *js.LiteralExpr:
			member = string(y.Data)
		default:
			return nil, false
		}
		if name, isFree := freeName(n.X); isFree && name == "Shugo" && !a.isBound(name) {
			if member == "props" {
				return a.parameters(), true
			}
			// The page, the slots and the component itself are not data
			// of the script: they come with the place the file is shown at.
			return nil, true
		}
		parent, followed := a.chain(n.X)
		if !followed || parent == nil {
			return nil, followed
		}
		return parent.field(member), true
	}
	return nil, false
}

// read notes what an expression reads.
func (a *analyzer) read(e js.IExpr, how reading) {
	switch n := e.(type) {
	case nil, *js.LiteralExpr, *js.NewTargetExpr, *js.ImportMetaExpr:
	case *js.GroupExpr:
		a.read(n.X, how)
	case *js.Var, *js.DotExpr:
		if s, followed := a.chain(n); !followed {
			a.read(n.(*js.DotExpr).X, asValue)
		} else if s != nil && how == asValue {
			s.all()
		}
	case *js.IndexExpr:
		a.read(n.X, asValue)
		a.read(n.Y, asValue)
	case *js.CallExpr:
		a.call(n.X)
		for _, argument := range n.Args.List {
			a.read(argument.Value, asValue)
		}
	case *js.NewExpr:
		if asksTheTime(n) {
			a.errorf("new Date() gives the time of the moment, and a template must give the same for the same data. Take the time in the script of a page, a layout or a shared block and show the variable")
		}
		a.call(n.X)
		if n.Args != nil {
			for _, argument := range n.Args.List {
				a.read(argument.Value, asValue)
			}
		}
	case *js.TemplateExpr:
		if n.Tag != nil {
			a.call(n.Tag)
		}
		for _, part := range n.List {
			a.read(part.Expr, asValue)
		}
	case *js.UnaryExpr:
		switch n.Op {
		case js.NotToken:
			a.read(n.X, asTruth)
		case js.AwaitToken:
			a.errorf("a template cannot wait: put the await into the script of a page, a layout or a shared block and show the variable")
			a.read(n.X, asValue)
		default:
			a.read(n.X, asValue)
		}
	case *js.BinaryExpr:
		a.binary(n, how)
	case *js.CondExpr:
		a.read(n.Cond, asTruth)
		a.read(n.X, how)
		a.read(n.Y, how)
	case *js.CommaExpr:
		for i, part := range n.List {
			if i == len(n.List)-1 {
				a.read(part, how)
			} else {
				a.read(part, asValue)
			}
		}
	case *js.ArrayExpr:
		for _, element := range n.List {
			a.read(element.Value, asValue)
		}
	case *js.ArrowFunc:
		for _, parameter := range n.Params.List {
			a.read(parameter.Default, asValue)
		}
		js.Walk(&inside{a}, &n.Body)
	default:
		// An object, a function, a class, or something not foreseen: all
		// that is read inside is read as a value.
		js.Walk(&inside{a}, e)
	}
}

// binary notes what the two sides of an operator read.
func (a *analyzer) binary(n *js.BinaryExpr, how reading) {
	switch n.Op {
	case js.AndToken:
		// The left side goes on only when there is nothing: and nothing
		// is the same however it is read.
		a.read(n.X, asTruth)
		a.read(n.Y, how)
	case js.OrToken, js.NullishToken:
		a.read(n.X, how)
		a.read(n.Y, how)
	case js.EqEqToken, js.NotEqToken, js.EqEqEqToken, js.NotEqEqToken:
		switch {
		case isNothing(n.Y):
			a.read(n.X, asTruth)
		case isNothing(n.X):
			a.read(n.Y, asTruth)
		default:
			a.read(n.X, asValue)
			a.read(n.Y, asValue)
		}
	default:
		a.read(n.X, asValue)
		a.read(n.Y, asValue)
	}
}

// isNothing tells whether an expression is null or undefined as written.
func isNothing(e js.IExpr) bool {
	switch n := unwrap(e).(type) {
	case *js.LiteralExpr:
		return n.TokenType == js.NullToken
	case *js.Var:
		name, isFree := freeName(n)
		return isFree && name == "undefined"
	case *js.UnaryExpr:
		return n.Op == js.VoidToken // esbuild writes undefined as void 0
	}
	return false
}

// call notes what calling an expression reads. A name the script imports is
// code. A method reads what it is called on whole — unless that is an
// imported name, as in format.price(x): then the import is code.
func (a *analyzer) call(callee js.IExpr) {
	switch n := unwrap(callee).(type) {
	case *js.Var:
		name, isFree := freeName(n)
		switch {
		case !isFree:
		case a.script.imports[name] && !a.isBound(name):
			a.called[name] = true
		default:
			// A function kept in data is not data: reading it whole makes
			// the JavaScript side say so when it meets one.
			a.read(n, asValue)
		}
	case *js.DotExpr:
		if name, isFree := freeName(n.X); isFree && a.script.imports[name] && !a.isBound(name) {
			a.called[name] = true
			return
		}
		a.read(n.X, asValue)
	default:
		a.read(callee, asValue)
	}
}

// inside reads everything within a function or an object as values. It
// stops at every expression and hands it to the analyzer, which knows the
// names that are followed.
type inside struct{ a *analyzer }

func (w *inside) Enter(n js.INode) js.IVisitor {
	switch n := n.(type) {
	case *js.Var, *js.DotExpr, *js.IndexExpr, *js.CallExpr, *js.NewExpr, *js.TemplateExpr, *js.UnaryExpr,
		*js.BinaryExpr, *js.CondExpr, *js.CommaExpr, *js.ArrayExpr, *js.ArrowFunc, *js.GroupExpr:
		w.a.read(n.(js.IExpr), asValue)
		return nil
	}
	return w
}

func (w *inside) Exit(js.INode) {}
