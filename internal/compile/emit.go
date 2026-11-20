package compile

import (
	"encoding/json"

	"github.com/grimalschi/shugo/internal/program"
)

// emitter writes a lowered template twice at once: as JavaScript that
// computes its values, and as the operations that put those values into the
// markup. The two must agree on the order of values, which is why one walk
// writes both.
//
// In the JavaScript "$$" is the block being computed. A piece of markup put
// into a component is a function that takes its own "$$", so the same code
// writes to the right block wherever it stands. Nothing here waits: a
// template works on data that is already there.
type emitter struct {
	m *module
	// parts are the operations of the template and of every fill in it.
	parts [][]program.Op
	// provides tells that the template gives markup to the layouts around it.
	provides bool
}

func (e *emitter) nodes(list []node) []program.Op {
	var ops []program.Op
	for _, n := range list {
		switch n := n.(type) {
		case markup:
			// Markup written in a row is one operation.
			if last := len(ops) - 1; last >= 0 && ops[last].Kind == program.Markup {
				ops[last].Text += n.text
			} else if n.text != "" {
				ops = append(ops, program.Op{Kind: program.Markup, Text: n.text})
			}
		case output:
			method := "text"
			if n.kind == program.HTML {
				method = "html"
			}
			e.call("$$."+method+"((", n.expr, "));")
			ops = append(ops, program.Op{Kind: n.kind})
		case attribute:
			e.call("$$.attr("+quote(n.name)+", (", n.expr, "));")
			ops = append(ops, program.Op{Kind: n.kind, Text: n.name})
		case spread:
			e.call("$$.spread((", n.expr, "));")
			ops = append(ops, program.Op{Kind: program.Spread})
		case choice:
			ops = append(ops, e.choice(n))
		case loop:
			ops = append(ops, e.loop(n))
		case call:
			e.component(n)
			ops = append(ops, program.Op{Kind: program.Call})
		case outlet:
			ops = append(ops, e.outlet(n))
		case provide:
			e.provides = true
			e.m.add("$$.provide(%s, %d, ($$) => {", quote(n.name), len(e.parts))
			e.part(n.body)
			e.m.add("});")
		case mount:
			ops = append(ops, program.Op{Kind: program.Mount, Text: n.name})
		}
	}
	return ops
}

// call writes project code between two pieces of the compiler's own.
func (e *emitter) call(before string, c code, after string) {
	e.m.add("%s", before)
	e.m.code(c)
	e.m.add("%s", after)
}

// part writes a body as the next part: the template itself, then every fill
// in the order they are met. The function the body is computed in is already
// open, and its header names the part by the number it gets here.
func (e *emitter) part(body []node) {
	at := len(e.parts)
	e.parts = append(e.parts, nil)
	e.parts[at] = e.nodes(body)
}

// choice writes the chain of branches. The value it leaves says which branch
// was taken, counting from 1, or 0 when none was.
func (e *emitter) choice(n choice) program.Op {
	op := program.Op{Kind: program.If}
	closed := false
	for i, b := range n.branches {
		switch {
		case b.test == nil:
			e.m.add("} else {")
			closed = true
		case i == 0:
			e.call("if ((", *b.test, ")) {")
		default:
			e.call("} else if ((", *b.test, ")) {")
		}
		e.m.add("$$.branch(%d);", i+1)
		op.Bodies = append(op.Bodies, e.nodes(b.body))
	}
	if !closed {
		e.m.add("} else {\n$$.branch(0);")
	}
	e.m.add("}")
	return op
}

// loop writes the loop. The value it leaves is the number of items, known
// only after the last one.
func (e *emitter) loop(n loop) program.Op {
	e.m.add("{\nconst $$count = $$.loop();\nlet $$index = 0;\nfor (const")
	e.m.code(n.item)
	e.call("of $$.items((", n.each, "))) {")
	if n.index != nil {
		e.call("const", *n.index, "= $$index;")
	}
	e.m.add("$$index++;")
	body := e.nodes(n.body)
	e.m.add("}\n$$.count($$count, $$index);\n}")
	return program.Op{Kind: program.For, Bodies: [][]program.Op{body}}
}

// component writes the call of a component with its parameters and with the
// markup put inside it.
func (e *emitter) component(n call) {
	e.call("$$.call((", n.target, "), {")
	for _, p := range n.props {
		switch {
		case p.spread:
			e.call("...(", *p.expr, "),")
		case p.expr != nil:
			e.call(quote(p.name)+": (", *p.expr, "),")
		default:
			e.m.add("%s: %s,", quote(p.name), p.literal)
		}
	}
	if len(n.fills) == 0 {
		e.m.add("});")
		return
	}
	e.m.add("}, [")
	for _, f := range n.fills {
		// The number of the part is known before its body is written.
		e.m.add("[%s, %d, ($$) => {", quote(f.name), len(e.parts))
		e.part(f.body)
		e.m.add("}],")
	}
	e.m.add("]);")
}

// outlet writes the place of a slot. The value it leaves says whether the
// caller put something there; if not, the markup inside <g-slot> is shown.
func (e *emitter) outlet(n outlet) program.Op {
	e.m.add("if (!$$.slot(%s)) {", quote(n.name))
	fallback := e.nodes(n.fallback)
	e.m.add("}")
	return program.Op{Kind: program.Slot, Text: n.name, Bodies: [][]program.Op{fallback}}
}

func quote(text string) string {
	out, _ := json.Marshal(text)
	return string(out)
}
