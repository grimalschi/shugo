package compile

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/syntax"
)

// parseScript compiles code as the <script> of a page, or as its <script
// module>.
func parseScript(t *testing.T, code string, module bool) (*syntax.File, *script, error) {
	t.Helper()
	open := "<script>"
	if module {
		open = "<script module>"
	}
	f, err := syntax.Parse("src/pages/a.svelte", open+"\n"+code+"\n</script>\n<p/>")
	if err != nil {
		t.Fatal(err)
	}
	at := f.Script
	if module {
		at = f.Module
	}
	s, err := compileScript(f, at, module)
	return f, s, err
}

func lines(s *script, spans []span) string {
	var out []string
	for _, sp := range spans {
		out = append(out, s.lines[sp.from:sp.to]...)
	}
	return strings.Join(out, "\n")
}

// The imports of a <script> go to the top of the module; the rest runs every
// time the component is shown.
func TestScriptSplit(t *testing.T) {
	_, s, err := parseScript(t, `
import { price, type Product } from "../lib/shop"
import Card from "../components/Card.svelte"
import "../styles/main.css"
import type { Unused } from "./types"
interface Props { title: string }

const { title } = $props() as Props
const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
const text = "import x from 'y'"
if (!title) {
  throw new Error("no title")
}
function helper(a: number): string { return String(a) }
const tpl = `+"`export ${ {a: 1}.a }`"+`
`, false)
	if err != nil {
		t.Fatal(err)
	}
	wantTop := `import { price } from "../lib/shop";
import Card from "../components/Card.svelte";
import "../styles/main.css";`
	if got := lines(s, s.top); got != wantTop {
		t.Errorf("imports\n got:\n%s\nwant:\n%s", got, wantTop)
	}
	wantBody := `const { title } = $props();
const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first();
const text = "import x from 'y'";
if (!title) {
  throw new Error("no title");
}
function helper(a) {
  return String(a);
}
const tpl = ` + "`export ${{ a: 1 }.a}`" + `;`
	if got := lines(s, s.body()); got != wantBody {
		t.Errorf("body\n got:\n%s\nwant:\n%s", got, wantBody)
	}
	if s.paths {
		t.Error("getStaticPaths was found where there is none")
	}
}

// <script module> stays whole at the top of the module; getStaticPaths is
// found however it is exported.
func TestScriptForms(t *testing.T) {
	cases := []struct {
		name, code string
		paths      bool
	}{
		{"no script code", "", false},
		{"only types", "interface Props { a: string }", false},
		{"a function", "import { a } from \"./a\"\nexport function getStaticPaths({ query }): { params: { slug: string } }[] {\n  return query(\"products\").all()\n}", true},
		{"constant with a function", "export const getStaticPaths = () => { return [] }\nconst a = 1", true},
		// Waiting is refused later, with a hint; the split itself takes it.
		{"an async function", "export async function getStaticPaths() { return [] }\nconst a = 1", true},
		{"export list", "const a = 1\nasync function getStaticPaths() { return [] }\nexport { getStaticPaths }", true},
		{"class", "export class A { m() { return {} } }\nconst a = 1", false},
		{"import.meta is an expression", "const u = import.meta.url", false},
		{"a body that mentions getStaticPaths", "const x = { getStaticPaths: 1 }", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, s, err := parseScript(t, c.code, true)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := lines(s, s.top), strings.Join(s.lines, "\n"); got != want {
				t.Errorf("top\n got:\n%s\nwant:\n%s", got, want)
			}
			if s.paths != c.paths {
				t.Errorf("paths = %v", s.paths)
			}
		})
	}
}

func TestScriptErrors(t *testing.T) {
	cases := []struct{ name, code, want string }{
		{"syntax", "const a = 1\nconst b = (a +\nconst c = 2", `src/pages/a.svelte:4:1: Unexpected "const"`},
		{"return outside of a function", "const a = 1\nif (a) return", `src/pages/a.svelte:3:8: Top-level return cannot be used inside an ECMAScript module`},
		{"export default", "export default 1", `src/pages/a.svelte:2:1: the script cannot have "export default": the file itself is what it exports`},
		{"markup in the script", "const a = <b>x</b>", `src/pages/a.svelte:2:19: Unterminated regular expression`},
		{"a prop of Svelte 4", "export let title = \"x\"", `src/pages/a.svelte:2:8: "export let" declares a prop in Svelte 4: a component of Svelte 5 takes its props with let { name } = $props()`},
		{"an export outside <script module>", "const a = 1\nexport const getStaticPaths = () => []", `src/pages/a.svelte:3:8: an export goes into <script module>, which runs once; this <script> runs every time the component is shown: <script module> export function getStaticPaths() { … } </script>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := parseScript(t, c.code, false)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != c.want {
				t.Errorf("\n got %s\nwant %s", err, c.want)
			}
		})
	}
}
