package compile

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/syntax"
)

func parseScript(t *testing.T, code string) (*syntax.File, *script, error) {
	t.Helper()
	f, err := syntax.Parse("pages/a.shugo", "---\n"+code+"\n---\n<p/>")
	if err != nil {
		t.Fatal(err)
	}
	s, err := compileScript(f)
	return f, s, err
}

func lines(s *script, spans []span) string {
	var out []string
	for _, sp := range spans {
		out = append(out, s.lines[sp.from:sp.to]...)
	}
	return strings.Join(out, "\n")
}

func TestScriptSplit(t *testing.T) {
	_, s, err := parseScript(t, `
import { price, type Product } from "../lib/shop"
import Card from "../components/Card.shugo"
import "../styles/main.css"
import type { Unused } from "./types"
interface Props { title: string }

export function getStaticPaths({ query }): { params: { slug: string } }[] {
  return query("products").select("slug").all().map((p: Product) => ({ params: { slug: p.slug } }))
}
export const prerender = { a: 1 }

const { title } = Shugo.props as Props
const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
const text = "import x from 'y'"
if (!title) {
  throw new Error("no title")
}
function helper(a: number): string { return String(a) }
const tpl = `+"`export ${ {a: 1}.a }`"+`
`)
	if err != nil {
		t.Fatal(err)
	}
	wantTop := `import { price } from "../lib/shop";
import Card from "../components/Card.shugo";
import "../styles/main.css";
export function getStaticPaths({ query }) {
  return query("products").select("slug").all().map((p) => ({ params: { slug: p.slug } }));
}
export const prerender = { a: 1 };`
	if got := lines(s, s.top); got != wantTop {
		t.Errorf("imports and exports\n got:\n%s\nwant:\n%s", got, wantTop)
	}
	wantBody := `const { title } = Shugo.props;
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
	if !s.paths {
		t.Error("getStaticPaths was not noticed")
	}
}

func TestScriptForms(t *testing.T) {
	cases := []struct {
		name, code, top string
		paths           bool
	}{
		{"no script code", "", "", false},
		{"only types", "interface Props { a: string }", "", false},
		{"constant with a function", "export const getStaticPaths = () => { return [] }\nconst a = 1", "export const getStaticPaths = () => {\n  return [];\n};", true},
		// Waiting is refused later, with a hint; the split itself takes it.
		{"an async function", "export async function getStaticPaths() { return [] }\nconst a = 1", "export async function getStaticPaths() {\n  return [];\n}", true},
		{"export list", "const a = 1\nasync function getStaticPaths() { return [] }\nexport { getStaticPaths }", "export { getStaticPaths };", true},
		{"class", "export class A { m() { return {} } }\nconst a = 1", "export class A {\n  m() {\n    return {};\n  }\n}", false},
		{"import.meta is an expression", "const u = import.meta.url", "", false},
		{"a body that mentions getStaticPaths", "const x = { getStaticPaths: 1 }", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, s, err := parseScript(t, c.code)
			if err != nil {
				t.Fatal(err)
			}
			if got := lines(s, s.top); got != c.top {
				t.Errorf("top\n got:\n%s\nwant:\n%s", got, c.top)
			}
			if s.paths != c.paths {
				t.Errorf("paths = %v", s.paths)
			}
		})
	}
}

func TestScriptErrors(t *testing.T) {
	cases := []struct{ name, code, want string }{
		{"syntax", "const a = 1\nconst b = (a +\nconst c = 2", `pages/a.shugo:4:1: Unexpected "const"`},
		{"return outside of a function", "const a = 1\nif (a) return", `pages/a.shugo:3:8: Top-level return cannot be used inside an ECMAScript module`},
		{"export default", "export default 1", `pages/a.shugo:2:1: the script cannot have "export default": the file itself is what it exports`},
		{"markup in the script", "const a = <b>x</b>", `pages/a.shugo:2:19: Unterminated regular expression`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := parseScript(t, c.code)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != c.want {
				t.Errorf("\n got %s\nwant %s", err, c.want)
			}
		})
	}
}
