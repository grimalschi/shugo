// Package compile turns the sources of a project into what the next stages
// run, and into the program the runtime puts values into.
//
// The JavaScript of a project is compiled into two bundles that share no
// module. The load side holds the scripts: they run with the data of the
// project and give their variables. The render side holds the templates:
// they are given a copy of exactly what they read of those variables, and
// nothing else of the project's data exists there. Between the two stands
// the dependency tree of every template — what it reads — which the
// compiler derives from its expressions.
package compile

import (
	"slices"
	"strings"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/syntax"
)

// unit is one .shugo file on its way through the compiler.
type unit struct {
	id     int
	file   *syntax.File
	role   role
	script *script
	names  *bindings // what the script declares
	nodes  []node    // the template
	reads  *reads    // what the template reads of the script

	styles []string
	// outlets are the slots the template shows; provided are the slots it
	// fills for the layouts around it, with the place of each in the file.
	outlets  []string
	provided map[string]int

	// loads tells that the file is computed in two steps: its script runs
	// on the load side and gives its variables, and its template is given
	// a copy of what it reads of them. A page and a layout are always
	// computed so. A component is when its script loads data, and why says
	// what makes it so; otherwise its script runs with its template, on
	// what the component is given.
	loads bool
	why   string

	// load is the module of the script for the load side, render the module
	// for the render side: the template, with the script of a file that
	// does not load.
	load, render *module
	// parts are the operations of the template and of every fill in it.
	parts [][]program.Op
	// own tells that the file itself cannot be computed once for all pages:
	// it reads the page it is shown on, or gives markup to the layouts of
	// that page.
	own bool
}

// readFile reads one file as far as it can be understood alone: its script,
// its template and what the template reads. id is the number its blocks carry.
func readFile(path, source string, id int, r role) (*unit, error) {
	f, err := syntax.Parse(path, source)
	if err != nil {
		return nil, Errors{err.(*syntax.Error)}
	}
	s := &script{smap: &sourcemap.Map{}}
	if f.Script.End > f.Script.Start {
		if s, err = compileScript(f); err != nil {
			return nil, err
		}
	}
	names, err := bind(f, s)
	if err != nil {
		return nil, Errors{err.(*syntax.Error)}
	}
	l := &lowerer{file: f, role: r}
	nodes := l.list(f.Body, true, false)
	if r == layout && !slices.Contains(l.outlets, defaultSlot) {
		l.errorf(0, "a layout shows the page inside itself: put <g-slot /> where the page goes")
	}
	if len(l.errs) > 0 {
		return nil, l.errs
	}
	read, errs := analyze(f, names, nodes)
	if len(errs) > 0 {
		return nil, errs
	}
	u := &unit{
		id: id, file: f, role: r, script: s, names: names, nodes: nodes, reads: read,
		styles: l.styles, outlets: l.outlets, provided: l.provided,
	}
	u.load = u.loadModule()
	return u, nil
}

// propsName is how the parameters of a file are written among the
// variables its template reads.
const propsName = "Shugo.props"

// loadModule writes the module of the load side: the imports and exports of
// the script, then a function that runs the rest of the script and returns
// the variables the template reads, then what the template reads of each.
func (u *unit) loadModule() *module {
	m := &module{file: u.file, last: u.file.Script.End}
	for _, top := range u.script.top {
		m.script(u.script, top.span)
	}
	m.add(`import { loader as $$loader } from "shugo:load";`)
	m.add("export default $$loader(%d, async function (Shugo) {", u.id)
	for _, lines := range u.script.body() {
		m.script(u.script, lines)
	}
	names := slices.Clone(u.reads.names)
	shapes := make([]string, len(u.reads.shapes))
	for i, s := range u.reads.shapes {
		shapes[i] = s.String()
	}
	if u.reads.props != nil {
		names = append(names, propsName)
		shapes = append(shapes, u.reads.props.String())
	}
	m.add("return [%s];", strings.Join(names, ", "))
	m.add("}, %s, [%s]);", quoteAll(names), strings.Join(shapes, ", "))
	return m
}

// renderModule writes the module of the render side and the operations of
// the template. The template of a file that loads is a function of the copy
// of what it reads; for any other file the script runs first, in the same
// function. Either way nothing waits there.
func (u *unit) renderModule() {
	m := &module{file: u.file, last: u.file.Script.End}
	for _, top := range u.script.top {
		// What a file that loads exports is for the load side: the list of
		// its pages, made from the data.
		if !u.loads || !top.export {
			m.script(u.script, top.span)
		}
	}
	if u.loads {
		m.add(`import { template as $$template } from "shugo:render";`)
		m.add("export default $$template(%d, function ($$, Shugo, $$part) {", u.id)
		if len(u.reads.names) > 0 {
			m.add("const [%s] = $$part;", strings.Join(u.reads.names, ", "))
		}
		if u.reads.props != nil {
			m.add("Shugo.props = $$part[%d];", len(u.reads.names))
		}
	} else {
		m.add(`import { component as $$component } from "shugo:render";`)
		m.add("export default $$component(%d, function ($$, Shugo) {", u.id)
		for _, lines := range u.script.body() {
			m.script(u.script, lines)
		}
	}
	e := &emitter{m: m}
	e.part(u.nodes)
	m.add("});")
	u.render, u.parts = m, e.parts
	u.own = m.page || e.provides || readsPage(strings.Join(u.script.lines, "\n"))
}

func quoteAll(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = quote(name)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
