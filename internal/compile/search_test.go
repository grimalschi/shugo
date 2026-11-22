package compile

import "testing"

func TestSearchModuleFlags(t *testing.T) {
	for _, test := range []struct {
		name, source    string
		catalog, search bool
	}{
		{"unused import", `<script>import { search } from "shugo:search"</script><p>Page</p>`, false, false},
		{"legacy", `<script>import { search } from "shugo:catalog"</script><p>{search().total}</p>`, true, false},
		{"generic", `<script>import { search } from "shugo:search"</script><p>{search("items", {}).total}</p>`, false, true},
		{"both", `<script>
import { search as catalog } from "shugo:catalog";
import { search } from "shugo:search";
</script><p>{catalog().total + search("items").total}</p>`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := templateProject(t, map[string]string{"src/pages/index.svelte": test.source})
			compiled, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			if compiled.Site.CatalogSearch != test.catalog || compiled.Site.Search != test.search {
				t.Fatalf("legacy and generic search flags = %v, %v; want %v, %v", compiled.Site.CatalogSearch,
					compiled.Site.Search, test.catalog, test.search)
			}
		})
	}
}
