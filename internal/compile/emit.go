package compile

import (
	"encoding/json"
	"strings"

	"github.com/grimalschi/shugo/internal/program"
)

// emitter writes a lowered template twice at once: as JavaScript that
// computes its values, and as the operations that put those values into the
// markup. The two must agree on the order of values, which is why one walk
// writes both.
//
// In the JavaScript "$$" is the block being computed. A snippet is a
// function that takes its own "$$", so the same code writes to the right
// block wherever it is shown.
type emitter struct {
	m *module
	// parts are the operations of the template and of every snippet in it.
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
		case shown:
			e.m.add("$$.text(")
			e.value(n.value)
			e.m.add(");")
			ops = append(ops, program.Op{Kind: program.Text})
		case constant:
			e.call("const", n.code, ";")
		case declare:
			e.m.add("const %s = (", n.name)
			e.value(n.value)
			e.m.add(");")
		case attribute:
			e.attribute(n)
			ops = append(ops, program.Op{Kind: n.kind, Text: n.name})
		case selected:
			e.m.add("$$.attr(\"selected\", $$.selected(")
			e.value(n.value)
			e.m.add(", %s), true);", n.within)
			ops = append(ops, program.Op{Kind: program.Attr, Text: "selected"})
		case spread:
			e.spread(n)
			ops = append(ops, program.Op{Kind: program.Spread})
		case choice:
			ops = append(ops, e.choice(n))
		case loop:
			ops = append(ops, e.loop(n))
		case call:
			ops = append(ops, e.component(n)...)
		case island:
			e.call("$$.island((", n.target, "), {")
			e.props(n.props)
			e.m.add("});")
			ops = append(ops, program.Op{Kind: program.Island})
		case snipp:
			e.snippet(n)
		case render:
			e.m.add("$$.render(")
			e.m.code(n.expr)
			e.m.add(", [")
			if strings.TrimSpace(n.args.text) != "" {
				e.m.code(n.args)
			}
			e.m.add("], %t);", n.optional)
			ops = append(ops, program.Op{Kind: program.Render})
		case dynamic:
			ops = append(ops, e.dynamic(n))
		case provide:
			e.provides = true
			e.m.add("$$.provide(%s, %d, ($$) => {", quote(n.name), len(e.parts))
			e.part(n.body)
			e.m.add("});")
		case outlet:
			e.m.add("$$.outlet(%s);", quote(n.name))
			op := program.Op{Kind: program.Render}
			if n.name == "title" {
				op.Text = program.Last
			}
			ops = append(ops, op)
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

// value writes a value as one JavaScript expression: the text in quotes, the
// expression, or both joined by the rules of Svelte.
func (e *emitter) value(v value) {
	switch {
	case v.bare:
		e.m.add("true")
		return
	case len(v.parts) == 0:
		e.m.add(`""`)
		return
	}
	if c, ok := v.single(); ok {
		e.call("(", c, ")")
		return
	}
	e.m.add("(\"\"")
	for _, p := range v.parts {
		if p.expr == nil {
			e.m.add("+ %s", quote(p.text))
			continue
		}
		e.call("+ $$.str((", *p.expr, "))")
	}
	e.m.add(")")
}

// attribute writes the value of a computed attribute.
func (e *emitter) attribute(n attribute) {
	switch {
	case n.class:
		e.m.add("$$.attr(\"class\", $$.cls(")
		e.value(n.value)
		e.m.add(", %s, {", quote(n.hash))
		e.directives(n.directives)
		e.m.add("}));")
	case n.style:
		e.m.add("$$.attr(\"style\", $$.sty(")
		e.value(n.value)
		e.m.add(", [{")
		e.directives(only(n.directives, false))
		e.m.add("}, {")
		e.directives(only(n.directives, true))
		e.m.add("}]));")
	default:
		e.m.add("$$.attr(%s, ", quote(n.name))
		e.value(n.value)
		e.m.add(", %t);", n.boolean)
	}
}

// only returns the style directives with or without !important.
func only(directives []directive, important bool) []directive {
	var out []directive
	for _, d := range directives {
		if d.important == important {
			out = append(out, d)
		}
	}
	return out
}

// directives writes the entries of an object of directives: "name": value.
func (e *emitter) directives(list []directive) {
	for _, d := range list {
		e.m.add("%s: ", quote(d.name))
		e.value(d.value)
		e.m.add(",")
	}
}

// spread writes every attribute of an element that has {...attributes}, in
// their order: one written later wins.
func (e *emitter) spread(n spread) {
	if n.chooses != "" {
		e.m.add("const %s = $$.spread({", n.chooses)
	} else {
		e.m.add("$$.spread({")
	}
	for _, en := range n.entries {
		if en.spread != nil {
			e.call("...(", *en.spread, "),")
			continue
		}
		e.m.add("%s: ", quote(en.name))
		e.value(en.value)
		e.m.add(",")
	}
	e.m.add("}, %s, {", quote(n.hash))
	e.directives(n.classes)
	e.m.add("}, [{")
	e.directives(only(n.styles, false))
	e.m.add("}, {")
	e.directives(only(n.styles, true))
	e.m.add("}], %t, %t);", n.preserve, n.chooses != "")
}

// part writes a body as the next part: the template itself, then every
// snippet in the order they are met. The function the body is computed in is
// already open, and its header names the part by the number it gets here.
func (e *emitter) part(body []node) {
	at := len(e.parts)
	e.parts = append(e.parts, nil)
	e.parts[at] = e.nodes(body)
}

// choice writes the chain of branches. The value it leaves says which branch
// was taken, counting from 1, or 0 when none was.
func (e *emitter) choice(n choice) program.Op {
	op := program.Op{Kind: program.If}
	if n.await != nil {
		// {#await}: a promise shows what stands before {:then}; anything
		// else is the value, and shows {:then}.
		e.call("{\nconst $$awaited = (", *n.await, ");")
		e.m.add("if ($$.pending($$awaited)) {\n$$.branch(1);")
		op.Bodies = append(op.Bodies, e.nodes(n.branches[0].body))
		e.m.add("} else {\n$$.branch(2);")
		if then := n.branches[1]; then.test != nil {
			e.call("const", *then.test, "= $$awaited;")
		}
		op.Bodies = append(op.Bodies, e.nodes(n.branches[1].body))
		e.m.add("}\n}")
		return op
	}
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

// loop writes {#each}. The value it leaves is the number of items, known
// only after the last one; {:else} follows when there are none.
func (e *emitter) loop(n loop) program.Op {
	e.call("{\nconst $$list = $$.items((", n.list, "));")
	e.m.add("const $$count = $$.loop();\nfor (let $$i = 0; $$i < $$list.length; $$i++) {")
	if n.item != nil {
		e.call("const", *n.item, "= $$list[$$i];")
	}
	if n.index != "" {
		e.m.add("const %s = $$i;", n.index)
	}
	body := e.nodes(n.body)
	e.m.add("}\n$$.count($$count, $$list.length);")
	op := program.Op{Kind: program.For, Bodies: [][]program.Op{body}}
	if n.empty != nil {
		e.m.add("if ($$list.length === 0) {")
		op.Bodies = append(op.Bodies, e.nodes(n.empty))
		e.m.add("}")
	}
	e.m.add("}")
	return op
}

// snippet writes {#snippet}: a function that writes a block of its own part.
func (e *emitter) snippet(n snipp) {
	e.m.add("const %s = $$.snippet(%d, ($$, ", n.name, len(e.parts))
	if strings.TrimSpace(n.params.text) != "" {
		e.m.code(n.params)
	}
	e.m.add(") => {")
	e.part(n.body)
	e.m.add("});")
}

// component writes the call of a component with its parameters and its
// snippets. <svelte:component> with a value that is not a component shows
// nothing, so its call is a branch.
func (e *emitter) component(n call) []program.Op {
	if n.optional {
		e.call("if ((", n.target, ")) {\n$$.branch(1);")
	}
	if n.self {
		e.m.add("$$.call($$self, {")
	} else {
		e.call("$$.call((", n.target, "), {")
	}
	e.props(n.props)
	for _, s := range n.snippets {
		e.m.add("%s: $$.snippet(%d, ($$, ", quote(s.name), len(e.parts))
		e.m.code(s.params)
		e.m.add(") => {")
		e.part(s.body)
		e.m.add("}),")
	}
	e.m.add("}, [")
	for _, g := range n.given {
		// The number of the part is known before its body is written.
		e.m.add("[%s, %d, ($$) => {", quote(g.name), len(e.parts))
		e.part(g.body)
		e.m.add("}],")
	}
	e.m.add("]);")
	call := program.Op{Kind: program.Call}
	if !n.optional {
		return []program.Op{call}
	}
	e.m.add("} else {\n$$.branch(0);\n}")
	return []program.Op{{Kind: program.If, Bodies: [][]program.Op{{call}}}}
}

func (e *emitter) props(props []prop) {
	for _, p := range props {
		if p.spread != nil {
			e.call("...(", *p.spread, "),")
			continue
		}
		e.m.add("%s: ", quote(p.name))
		e.value(p.value)
		e.m.add(",")
	}
}

// dynamic writes <svelte:element>: the tag is a value, and so is whether it
// is shown at all.
func (e *emitter) dynamic(n dynamic) program.Op {
	e.call("{\nconst $$tag = $$.tag((", n.tag, "));\nif ($$tag !== \"\") {")
	attrs := e.nodes(n.attrs)
	e.m.add("if (!$$.void($$tag)) {")
	children := e.nodes(n.children)
	e.m.add("}\n}\n}")
	return program.Op{Kind: program.Element, Bodies: [][]program.Op{attrs, children}}
}

func quote(text string) string {
	out, _ := json.Marshal(text)
	return string(out)
}
