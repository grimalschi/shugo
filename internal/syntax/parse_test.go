package syntax

import (
	"fmt"
	"strings"
	"testing"
)

// dump writes a tree in one line, so that a test reads like the template.
func dump(nodes []Node) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n := n.(type) {
		case *Text:
			fmt.Fprintf(&b, "%q", n.Value)
		case *Expr:
			fmt.Fprintf(&b, "{%s}", n.Code)
		case *Element:
			b.WriteString("<" + n.Name)
			for _, a := range n.Attrs {
				switch a.Kind {
				case Bare:
					b.WriteString(" " + a.Name)
				case Literal:
					fmt.Fprintf(&b, " %s=%q", a.Name, a.Value)
				case Dynamic:
					fmt.Fprintf(&b, " %s={%s}", a.Name, a.Value)
				case Spread:
					fmt.Fprintf(&b, " {...%s}", a.Value)
				}
			}
			if n.Closed {
				b.WriteString("/>")
			} else {
				b.WriteString(">" + dump(n.Children) + "</" + n.Name + ">")
			}
		}
	}
	return b.String()
}

func TestParse(t *testing.T) {
	cases := []struct{ name, source, script, want string }{
		{"text only", "hello", "", `"hello"`},
		{"script and template", "---\nconst a = 1\n---\n<p>{a}</p>\n", "const a = 1\n", `<p>{a}</p>`},
		{"empty script", "---\n---\n<p/>", "", `<p/>`},
		{"dashes later are text", "<p>a</p>\n---\n", "", `<p>"a"</p>"\n---"`},
		{"void and self-closing", `<br><img src="a.png"><div class="x" />`, "", `<br/><img src="a.png"/><div class="x"/>`},
		{"attributes", `<a href={url} {id} {...rest} hidden data-x='1' title=plain>t</a>`, "",
			`<a href={url} id={id} {...rest} hidden data-x="1" title="plain">"t"</a>`},
		{"nested braces and strings", `{fn({a: "}"}, '}', ` + "`${ {b: 1}.b }`" + `)}`, "", `{fn({a: "}"}, '}', ` + "`${ {b: 1}.b }`" + `)}`},
		{"regular expression with a brace", `{name.replace(/}/g, "")}`, "", `{name.replace(/}/g, "")}`},
		{"division is not a regular expression", `{a / b}{c / d}`, "", `{a / b}{c / d}`},
		{"comment-only expression gives nothing", `a{/* note */}b`, "", `"a""b"`},
		{"html comment and doctype are text", "<!doctype html>\n<!-- {not code} <b> -->", "", `"<!doctype html>\n<!-- {not code} <b> -->"`},
		{"script and style are raw", `<style>a{b:c}</style><script>if (a<b) {}</script>`, "", `<style>"a{b:c}"</style><script>"if (a<b) {}"</script>`},
		{"a lone < is text", `a < b`, "", `"a < b"`},
		{"components", `<Card title="x"><ui.Icon name={n} /></Card>`, "", `<Card title="x"><ui.Icon name={n}/></Card>`},
		{"a component named like a void element", `<Link href="/">home</Link>`, "", `<Link href="/">"home"</Link>`},
		{"template is trimmed", "\n\n  <p>a</p>  \n", "", `<p>"a"</p>`},
		{"typescript in expressions", `{(item as Item)!.price / 2}`, "", `{(item as Item)!.price / 2}`},
		{"comparisons and generics are not markup", `{a < b && c > d ? x : y}{list.map(<T,>(v: T) => v)}{i<n}`, "", `{a < b && c > d ? x : y}{list.map(<T,>(v: T) => v)}{i<n}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, err := Parse("test.shugo", c.source)
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(f.Body); got != c.want {
				t.Errorf("template\n got %s\nwant %s", got, c.want)
			}
			if got := f.Source[f.Script.Start:f.Script.End]; got != c.script {
				t.Errorf("script\n got %q\nwant %q", got, c.script)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"unclosed element", "<div>\n  <p>a</p>", `test.shugo:1:1: <div> is never closed`},
		{"wrong closing tag", "<div>\n  <p>a</span>\n</div>", `test.shugo:2:7: </span> does not close <p> opened at 2:3`},
		{"stray closing tag", "a</p>", `test.shugo:1:2: </p> closes nothing`},
		{"unclosed expression", "<p>{a + (b</p>", `test.shugo:1:4: "{" is never closed (unterminated regular expression at 1:12)`},
		{"unclosed expression at the end", "<p>{a + (b", `test.shugo:1:4: "{" is never closed`},
		{"unclosed string in expression", `<p>{"a}</p>`, `test.shugo:1:4: "{" is never closed (unterminated string at 1:5)`},
		{"unclosed script", "---\nconst a = 1\n<p/>", `test.shugo:1:1: the script that starts with "---" has no closing "---" line`},
		{"unclosed attribute", `<a href="x>t</a>`, `test.shugo:1:9: the value of attribute href is never closed with "`},
		{"unfinished tag", `<a href="x"`, `test.shugo:1:1: tag <a is not finished with >`},
		{"bad shorthand", `<a {a.b}>t</a>`, `test.shugo:1:4: expected {name} or {...attributes} among the attributes of <a>`},
		{"markup in an expression", "<ul>\n  {items.map(item => <li>{item}</li>)}\n</ul>", `test.shugo:2:3: ` + noMarkup},
		{"markup after a condition", `<p>{user && <b>{user.name}</b>}</p>`, `test.shugo:1:4: ` + noMarkup},
		{"position counts characters", "<p>hello</p>\n<b>café</i>", `test.shugo:2:8: </i> does not close <b> opened at 2:1`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("test.shugo", c.source)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != c.want {
				t.Errorf("\n got %s\nwant %s", err, c.want)
			}
		})
	}
}

func TestScannerTokens(t *testing.T) {
	kinds := map[TokenKind]string{Space: "_", Comment: "c", Word: "w", Number: "n", String: "s", Template: "t", Regexp: "r", Punct: "p"}
	cases := []struct{ source, want string }{
		{`a / b / c`, "w_p_w_p_w"},
		{`x = /a\/b[/]/g.test(y)`, "w_p_rpwpwp"},
		{`return /re/`, "w_r"},
		{`(a) / 2`, "pwp_p_n"},
		{`a! / b`, "wp_p_w"},
		{`a != /b/`, "w_pp_r"},
		{`i++ / 2`, "wp_p_n"},
		{"`a ${ `b ${c}` } d` + 1", "t_p_n"},
		{`'it\'s' // rest`, "s_c"},
		{`/* a */ 1.5e3`, "c_n"},
		{`x => /a/`, "w_pp_r"},
	}
	for _, c := range cases {
		s := NewScanner(c.source, 0)
		var got strings.Builder
		for {
			token, err := s.Next()
			if err != nil {
				t.Fatalf("%s: %v", c.source, err)
			}
			if token.Kind == EOF {
				break
			}
			got.WriteString(kinds[token.Kind])
		}
		if got.String() != c.want {
			t.Errorf("%s\n got %s\nwant %s", c.source, got.String(), c.want)
		}
	}
}
