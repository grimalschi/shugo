package syntax

import (
	"fmt"
	"strings"
	"testing"
)

// show writes a parsed template compactly, to compare with what a test expects.
func show(nodes []Node) string {
	var out strings.Builder
	for _, n := range nodes {
		switch n := n.(type) {
		case *Text:
			out.WriteString(n.Raw)
		case *HTMLComment:
			fmt.Fprintf(&out, "<!--%s-->", n.Data)
		case *Doctype:
			out.WriteString(n.Raw)
		case *Tag:
			kinds := map[TagKind]string{Expression: "", HTMLTag: "@html ", ConstTag: "@const ", RenderTag: "@render ", DebugTag: "@debug "}
			fmt.Fprintf(&out, "{%s%s}", kinds[n.Kind], n.Code.Text)
		case *IfBlock:
			for i, b := range n.Branches {
				switch {
				case i == 0:
					fmt.Fprintf(&out, "{#if %s}", b.Test.Text)
				case b.Test != nil:
					fmt.Fprintf(&out, "{:else if %s}", b.Test.Text)
				default:
					out.WriteString("{:else}")
				}
				out.WriteString(show(b.Body))
			}
			out.WriteString("{/if}")
		case *EachBlock:
			fmt.Fprintf(&out, "{#each %s", n.List.Text)
			if n.Item != nil {
				fmt.Fprintf(&out, " as %s", n.Item.Text)
			}
			if n.Index != "" {
				fmt.Fprintf(&out, ", %s", n.Index)
			}
			if n.Key != nil {
				fmt.Fprintf(&out, " (%s)", n.Key.Text)
			}
			out.WriteString("}" + show(n.Body))
			if n.Else != nil {
				out.WriteString("{:else}" + show(n.Else))
			}
			out.WriteString("{/each}")
		case *AwaitBlock:
			fmt.Fprintf(&out, "{#await %s}%s", n.Value.Text, show(n.Pending))
			if n.Then != nil {
				fmt.Fprintf(&out, "{:then %s}%s", n.Then.Test.Text, show(n.Then.Body))
			}
			out.WriteString("{/await}")
		case *KeyBlock:
			fmt.Fprintf(&out, "{#key %s}%s{/key}", n.Value.Text, show(n.Body))
		case *SnippetBlock:
			fmt.Fprintf(&out, "{#snippet %s(%s)}%s{/snippet}", n.Name, n.Params.Text, show(n.Body))
		case *Element:
			out.WriteString("<" + n.Name)
			for _, a := range n.Attrs {
				out.WriteString(" " + showAttr(a))
			}
			if n.SelfClosing {
				out.WriteString("/>")
				continue
			}
			out.WriteString(">" + show(n.Children) + "</" + n.Name + ">")
		}
	}
	return out.String()
}

func showAttr(a Attr) string {
	name := a.Name
	if a.Kind == Directive {
		name = a.Directive + ":" + a.Name
		for _, m := range a.Modifiers {
			name += "|" + m
		}
	}
	switch a.Kind {
	case Spread:
		return "{..." + a.Value[0].Expr.Text + "}"
	case Shorthand:
		return "{" + a.Name + "}"
	case Attach:
		return "{@attach}"
	}
	if a.Value == nil {
		return name
	}
	var value strings.Builder
	for _, part := range a.Value {
		if part.Expr != nil {
			value.WriteString("{" + part.Expr.Text + "}")
		} else {
			value.WriteString(part.Text)
		}
	}
	if a.Quoted {
		return name + `="` + value.String() + `"`
	}
	return name + "=" + value.String()
}

func testParser(t *testing.T) *Parser {
	t.Helper()
	parser, err := Start("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(parser.Close)
	return parser
}

func TestParse(t *testing.T) {
	parser := testParser(t)
	cases := []struct{ source, want string }{
		{"<p>{a}</p>", "<p>{a}</p>"},
		{"{#if a}x{:else if b}y{:else}z{/if}", "{#if a}x{:else if b}y{:else}z{/if}"},
		{"{#each items as item, i (item.id)}<li>{item}</li>{:else}none{/each}", "{#each items as item, i (item.id)}<li>{item}</li>{:else}none{/each}"},
		{"{#each list as { a, b }, index}{a}{/each}", "{#each list as { a, b }, index}{a}{/each}"},
		{`<script lang="ts"></script>{#each (items as Item[]) as item}{item}{/each}`, "{#each items as Item[] as item}{item}{/each}"},
		{"{#each { length: 3 }}x{/each}", "{#each { length: 3 }}x{/each}"},
		{"{#await p}wait{:then v}{v}{/await}", "{#await p}wait{:then v}{v}{/await}"},
		{"{#key k}<i>{k}</i>{/key}", "{#key k}<i>{k}</i>{/key}"},
		{"{#snippet row(item, i)}<td>{item}</td>{/snippet}{@render row(x, 1)}", "{#snippet row(item, i)}<td>{item}</td>{/snippet}{@render row(x, 1)}"},
		{"{@html html}{@const n = a + 1}{@debug}", "{@html html}{@const n = a + 1}{@debug }"},
		{`<a href={url} title="Hi {name}!" {id} {...rest} class:active={on} style:color|important="red" onclick={go} disabled>x</a>`,
			`<a href={url} title="Hi {name}!" {id} {...rest} class:active={on} style:color|important="red" onclick={go} disabled>x</a>`},
		{"<input value={v}><br><img src=a.png />", "<input value={v}></input><br></br><img src=a.png/>"},
		{"<ul><li>a<li>b</ul>", "<ul><li>a</li><li>b</li></ul>"},
		{"<p>a<div>b</div>", "<p>a</p><div>b</div>"},
		{"<div><script>if (a < b) {}</script></div>", "<div><script>if (a < b) {}</script></div>"},
		{"<!doctype html><html><!-- c --></html>", "<!doctype html><html><!-- c --></html>"},
		{"<svelte:element this={tag}>x</svelte:element><Card.Body />", "<svelte:element this={tag}>x</svelte:element><Card.Body/>"},
	}
	for _, c := range cases {
		f, err := parser.Parse("a.svelte", c.source)
		if err != nil {
			t.Errorf("%s: %v", c.source, err)
			continue
		}
		if got := show(f.Body); got != c.want {
			t.Errorf("%s\n got %s\nwant %s", c.source, got, c.want)
		}
	}
}

func TestScriptsAndStyle(t *testing.T) {
	parser := testParser(t)
	source := `<script module>
  export function getStaticPaths() { return [] }
</script>
<script lang="ts">
  let { title } = $props();
</script>

<h1>{title}</h1>

<style>
  h1 { color: red; }
</style>`
	f, err := parser.Parse("a.svelte", source)
	if err != nil {
		t.Fatal(err)
	}
	code := func(s *Script) string { return strings.TrimSpace(source[s.Code.Start:s.Code.End]) }
	if f.Module == nil || code(f.Module) != "export function getStaticPaths() { return [] }" {
		t.Errorf("module: %+v", f.Module)
	}
	if f.Script == nil || code(f.Script) != "let { title } = $props();" {
		t.Errorf("script: %+v", f.Script)
	}
	if f.Style == nil || code(f.Style) != "h1 { color: red; }" {
		t.Errorf("style: %+v", f.Style)
	}
	if got := strings.TrimSpace(show(f.Body)); got != "<h1>{title}</h1>" {
		t.Errorf("body: %q", got)
	}
}

func TestParseErrors(t *testing.T) {
	parser := testParser(t)
	cases := []struct{ source, want string }{
		{"---\nconst a = 1\n---\n<p/>", `a.svelte:1:1: the code of a component goes into <script>`},
		{"<div>", "a.svelte:1:1: `<div>` was left open"},
		{"<div></span>", "a.svelte:1:6: `</span>` attempted to close an element that was not open"},
		{"</p>", "a.svelte:1:1: `</p>` attempted to close an element that was not open"},
		{"{#if a}x", "Block was left open"},
		{"{#if a}<p>x{/if}", "Unexpected block closing tag"},
		{"{#each a as}x{/each}", "Expected whitespace"},
		{"{:else}", "{:...} block is invalid at this position"},
		{"{#for a}x{/for}", "Expected 'if', 'each', 'await', 'key' or 'snippet'"},
		{"{@foo a}", "Expected 'html', 'render', 'attach', 'const', or 'debug'"},
		{"<script>a</script><script>b</script>", "A component can have a single top-level `<script>` element"},
		{`<p a="1" a="2"></p>`, "Attributes need to be unique"},
		{"<br></br>", "Void elements cannot have children or closing tags"},
		{"<script>const item: number = 42</script>", "a.svelte:1:19: Unexpected token"},
	}
	for _, c := range cases {
		_, err := parser.Parse("a.svelte", c.source)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s:\n got %v\nwant …%s…", c.source, err, c.want)
		}
	}
	if _, err := parser.Parse("valid.svelte", "<p>valid</p>"); err != nil {
		t.Fatalf("parse error stopped the compiler: %v", err)
	}
}

func TestImplicitClose(t *testing.T) {
	f, err := testParser(t).Parse("a.svelte", "<div><p>text</div>")
	if err != nil {
		t.Fatal(err)
	}
	if got := show(f.Body); got != "<div><p>text</p></div>" {
		t.Errorf("got %s", got)
	}
}

func TestSourceOffsets(t *testing.T) {
	parser := testParser(t)
	const source = "\ufeff<script lang=\"ts\">let café: string = \"yes\"</script>\n<p>🍋 {café}</p>"
	f, err := parser.Parse("café.svelte", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Source[f.Script.Code.Start:f.Script.Code.End]; got != "let café: string = \"yes\"" {
		t.Errorf("script range: %q", got)
	}
	element := f.Body[len(f.Body)-1].(*Element)
	tag := element.Children[1].(*Tag)
	if tag.Code.Start != strings.Index(source, "{café}")+1 || tag.Code.Text != "café" {
		t.Errorf("expression: %+v", tag.Code)
	}
	_, err = parser.Parse("café.svelte", "\ufeff<main>\n  🍋 café</div>")
	if err == nil || !strings.Contains(err.Error(), "café.svelte:2:9: `</div>`") {
		t.Fatalf("diagnostic at wrong offset: %v", err)
	}
}

func TestConcurrentParse(t *testing.T) {
	parser := testParser(t)
	for i := range 8 {
		name := fmt.Sprintf("template-%d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := "<p>" + name + "</p>"
			f, err := parser.Parse(name+".svelte", source)
			if err != nil {
				t.Fatal(err)
			}
			if parsed := show(f.Body); parsed != source {
				t.Errorf("parsed %q, want %q", parsed, source)
			}
		})
	}
}
