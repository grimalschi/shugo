package compile

import (
	"strings"
	"testing"
)

// The styles of a component apply to it alone, the way Svelte writes them.
// What Svelte gives was taken from Svelte 5.59.1 for a component named
// X.svelte; it also comments out selectors that match nothing, which
// Shugo does not.
func TestScopeCSS(t *testing.T) {
	hash := scopeClass("X.svelte")
	if hash != "svelte-1lj1c1s" {
		t.Errorf("the class of X.svelte is %s, Svelte makes svelte-1lj1c1s", hash)
	}
	cases := map[string]string{
		"main p { color: red }":                             "main.svelte-1lj1c1s p:where(.svelte-1lj1c1s) { color: red }",
		`main > p.x:hover a::before { content: "x" }`:       `main.svelte-1lj1c1s > p.x:where(.svelte-1lj1c1s):hover a:where(.svelte-1lj1c1s)::before { content: "x" }`,
		":global(body) main { margin: 0 }":                  "body main.svelte-1lj1c1s { margin: 0 }",
		":global(.dark) p { color: blue }":                  ".dark p.svelte-1lj1c1s { color: blue }",
		"* { box-sizing: border-box }":                      ".svelte-1lj1c1s { box-sizing: border-box }",
		":hover { color: green }":                           ".svelte-1lj1c1s:hover { color: green }",
		`a[href^="#"] { color: pink }`:                      `a[href^="#"].svelte-1lj1c1s { color: pink }`,
		"span:not(.y) { font: inherit }":                    "span.svelte-1lj1c1s:not(.y) { font: inherit }",
		"@media (min-width: 10px) { ul li { padding: 0 } }": "@media (min-width: 10px) { ul.svelte-1lj1c1s li:where(.svelte-1lj1c1s) { padding: 0 } }",
		"p :global(strong) { font-weight: 700 }":            "p.svelte-1lj1c1s strong { font-weight: 700 }",
		"@keyframes spin { from { opacity: 0 } to { opacity: 1 } }\nspan { animation: spin 1s; }": "@keyframes svelte-1lj1c1s-spin { from { opacity: 0 } to { opacity: 1 } }\nspan.svelte-1lj1c1s { animation: svelte-1lj1c1s-spin 1s; }",
		":global { .free { color: black } }":                                                      " .free { color: black } ",
	}
	for css, want := range cases {
		if got := scopeCSS(css, hash); strings.TrimSpace(got) != strings.TrimSpace(want) {
			t.Errorf("%s\n got %s\nwant %s", css, got, want)
		}
	}
}
