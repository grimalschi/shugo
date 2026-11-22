// Package compile turns the sources of a project into what the next stages
// run: one JavaScript bundle that computes the values of every template,
// reading the data database of the project through Shugo.query, and the
// program the runtime puts those values into.
package compile

import (
	"strings"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/syntax"
)

type islandBinding struct {
	path    string
	request bool // at least one call binds props to request data
}

// unit is one compiled .shugo file.
type unit struct {
	file   *syntax.File
	role   role
	module *module
	// parts are the operations of the template and of every snippet in it.
	parts  [][]program.Op
	styles []string // the <style> of the file, scoped to its elements
	// outlets are where the file shows what pages give to layouts: "head"
	// for a layout with <head>; provided are what the file gives, with the
	// place of each in the file.
	outlets  []string
	provided map[string]int
	// paths tells that the file exports getStaticPaths.
	paths bool
	// warnings are what is written in a way Svelte allows but warns about.
	warnings []string
	islands  map[string]islandBinding
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
	var scripts []*script
	for _, at := range []*syntax.Script{f.Module, f.Script} {
		if at == nil || strings.TrimSpace(source[at.Code.Start:at.Code.End]) == "" {
			scripts = append(scripts, &script{smap: &sourcemap.Map{}, at: at})
			continue
		}
		s, err := compileScript(f, at, at == f.Module)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, s)
	}
	moduleScript, instance := scripts[0], scripts[1]

	var islands map[string]islandBinding
	if len(moduleScript.islands)+len(instance.islands) > 0 {
		islands = map[string]islandBinding{}
		for _, s := range scripts {
			for name, path := range s.islands {
				islands[name] = islandBinding{path: path}
			}
		}
	}
	l := &lowerer{file: f, role: r, islands: islands}
	var styles []string
	if f.Style != nil {
		css := source[f.Style.Code.Start:f.Style.Code.End]
		if strings.TrimSpace(css) != "" {
			l.hash = scopeClass(path)
			styles = append(styles, scopeCSS(css, l.hash))
		}
	}
	nodes := l.fragment(f.Body, context{top: true})
	if r == layout && !renders(nodes) {
		l.errorf(0, "a layout shows the page inside itself: put {@render children()} where the page goes, with let { children } = $props()")
	}
	if len(l.errs) > 0 {
		return nil, l.errs
	}

	// The module: what must stay at the top of it, then one function that
	// runs the rest of the script and computes the template.
	m := &module{file: f}
	for _, s := range scripts {
		for _, lines := range s.top {
			m.script(s, lines)
		}
	}
	m.add(`import { component as $$component, $state, $derived, $effect, $bindable, $inspect, $host } from "shugo:runtime";`)
	m.add("const $$self = $$component(%d, function ($$, Shugo) {", id)
	m.add("const $props = () => $$.props();")
	for _, lines := range instance.body() {
		m.script(instance, lines)
	}
	e := &emitter{m: m}
	e.part(nodes)
	m.add("});\nexport default $$self;")

	return &unit{
		file: f, role: r, module: m, parts: e.parts, styles: styles,
		islands: islands,
		outlets: l.outlets, provided: l.provided, paths: moduleScript.paths || instance.paths,
		warnings: append(warningsOf(f), l.warnings...),
		own:      m.page || e.provides || readsPage(strings.Join(instance.lines, "\n")) || readsPage(strings.Join(moduleScript.lines, "\n")),
	}, nil
}

// warningsOf returns the warnings the parser had about a file.
func warningsOf(f *syntax.File) []string {
	var out []string
	for _, w := range f.Warnings {
		out = append(out, w.Error())
	}
	return out
}

// renders tells whether a template shows a snippet anywhere: a layout must
// show the page.
func renders(nodes []node) bool {
	for _, n := range nodes {
		switch n := n.(type) {
		case render:
			return true
		case choice:
			for _, b := range n.branches {
				if renders(b.body) {
					return true
				}
			}
		case loop:
			if renders(n.body) || renders(n.empty) {
				return true
			}
		case dynamic:
			if renders(n.children) {
				return true
			}
		case call:
			for _, g := range n.given {
				if renders(g.body) {
					return true
				}
			}
		}
	}
	return false
}
