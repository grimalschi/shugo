package compile

import (
	"strings"
	"testing"
)

// A project searches when its code calls search; importing it is not enough.
func TestSearchModuleFlags(t *testing.T) {
	for _, test := range []struct {
		name, source string
		search       bool
	}{
		{"no search", `<script>import { query } from "shugo"</script><p>Page</p>`, false},
		{"unused import", `<script>import { search } from "shugo"</script><p>Page</p>`, false},
		{"a search", `<script>import { search } from "shugo"</script><p>{search("items", {}).total}</p>`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := templateProject(t, map[string]string{"src/pages/index.svelte": test.source})
			compiled, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			if compiled.Site.Search != test.search {
				t.Fatalf("searches = %v, want %v", compiled.Site.Search, test.search)
			}
		})
	}
}

// search came from shugo:search, and in the experiment with islands from
// shugo:catalog: both are told to take it from shugo.
func TestSearchFromOldModules(t *testing.T) {
	for _, module := range []string{"shugo:search", "shugo:catalog"} {
		p := templateProject(t, map[string]string{"src/pages/index.svelte": `<script>import { search } from "` + module + `"</script><p>{search("items").total}</p>`})
		if _, err := Build(p); err == nil || !strings.Contains(err.Error(), `import { search } from "shugo"`) {
			t.Errorf("%s: %v", module, err)
		}
	}
}
