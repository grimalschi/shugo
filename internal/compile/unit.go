// Package compile turns the sources of a project into what the next stages
// run: one JavaScript bundle that loads the data and computes the values of
// every template, and the program the runtime puts those values into.
package compile

import (
	"slices"
	"strings"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/syntax"
)

// unit is one compiled .shugo file.
type unit struct {
	file   *syntax.File
	role   role
	script *script
	module *module
	// parts are the operations of the template and of every fill in it.
	parts  [][]program.Op
	styles []string
	// outlets are the slots the template shows; provided are the slots it
	// fills for the layouts around it, with the place of each in the file.
	outlets  []string
	provided map[string]int
	// own tells that the file itself cannot be computed once for all pages:
	// it reads the page it is shown on, or gives markup to the layouts of
	// that page.
	own bool
}

// compileFile compiles one file. id is the number its blocks carry.
func compileFile(path, source string, id int, r role) (*unit, error) {
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
	l := &lowerer{file: f, role: r}
	nodes := l.list(f.Body, true, false)
	if r == layout && !slices.Contains(l.outlets, defaultSlot) {
		l.errorf(0, "a layout shows the page inside itself: put <g-slot /> where the page goes")
	}
	if len(l.errs) > 0 {
		return nil, l.errs
	}

	// The module: what must stay at the top of it, then one function that
	// runs the rest of the script and computes the template.
	m := &module{file: f, last: f.Script.End}
	for _, lines := range s.top {
		m.script(s, lines)
	}
	m.add(`import { component as $$component } from "shugo:runtime";`)
	m.add("export default $$component(%d, async function ($$, Shugo) {", id)
	for _, lines := range s.body() {
		m.script(s, lines)
	}
	e := &emitter{m: m}
	e.part(nodes)
	m.add("});")

	return &unit{
		file: f, role: r, script: s, module: m, parts: e.parts, styles: l.styles,
		outlets: l.outlets, provided: l.provided,
		own: m.page || e.provides || readsPage(strings.Join(s.lines, "\n")),
	}, nil
}
