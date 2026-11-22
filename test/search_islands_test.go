package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
)

var searchRows = regexp.MustCompile(`data-hit-id="([^"]+)"`)

type searchResponse struct {
	Endpoint string `json:"endpoint"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	Pages    int    `json:"pages"`
	Hits     []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"hits"`
	Query struct {
		Q       string `json:"q"`
		Sort    string `json:"sort"`
		Filters struct {
			Category  []string `json:"category"`
			Available *bool    `json:"available"`
			Price     struct {
				From *int `json:"from"`
				To   *int `json:"to"`
			} `json:"price"`
		} `json:"filters"`
	} `json:"query"`
}

func (r searchResponse) ids() []string {
	var ids []string
	for _, row := range r.Hits {
		ids = append(ids, row.ID)
	}
	return ids
}

// TestSearchIslands exercises two source tables, the same Svelte island and
// its browser API from the live Go server, then writes to the source journal.
func TestSearchIslands(t *testing.T) {
	demo := filepath.Join("..", "examples", "search")
	s := openSite(t, t.TempDir())
	if err := os.CopyFS(filepath.Join(s.root, "src"), os.DirFS(filepath.Join(demo, "src"))); err != nil {
		t.Fatal(err)
	}
	settings, err := os.ReadFile(filepath.Join(demo, "shugo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.root, "shugo.json"), settings, 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := os.ReadFile(filepath.Join(demo, "scripts", "fixture.sql"))
	if err != nil {
		t.Fatal(err)
	}
	s.sql(string(db))
	if report := s.build(); report.SearchIndexes != 2 {
		t.Fatalf("expected one index per source binding, report %+v", report)
	}

	extension := server.NewSearch(s.project().Path(project.Output))
	t.Cleanup(func() { extension.Close() })
	ts := httptest.NewServer(&server.Server{
		Pages: s.pages, Search: extension,
		Assets: s.project().Path(project.Output, compile.Assets),
		Failed: func(err error) { t.Error(err) },
	})
	t.Cleanup(ts.Close)

	api := func(endpoint string, query url.Values, want ...string) (searchResponse, string) {
		t.Helper()
		address := endpoint + "?" + query.Encode()
		status, headers, body := get(t, ts, address)
		if status != http.StatusOK || headers.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("%s: %d, headers %v, body %s", address, status, headers, body)
		}
		var result searchResponse
		if err := json.Unmarshal([]byte(body), &result); err != nil {
			t.Fatalf("decode %s: %v", address, err)
		}
		if result.Endpoint != endpoint || !slices.Equal(result.ids(), want) {
			t.Fatalf("%s: endpoint %q, rows %v, want %v", address, result.Endpoint, result.ids(), want)
		}
		if strings.Contains(body, "private-") || strings.Contains(body, "internal_code") || strings.Contains(body, "editor_note") {
			t.Fatalf("%s exposed a source-only field: %s", address, body)
		}
		return result, body
	}

	page := func(path string, query url.Values, ids ...string) (searchResponse, string) {
		t.Helper()
		address := path
		if len(query) != 0 {
			address += "?" + query.Encode()
		}
		status, _, html := get(t, ts, address)
		if status != http.StatusOK || !strings.Contains(html, `data-shugo-target><!--[-->`) {
			t.Fatalf("%s: expected initial Svelte HTML, status %d, body %s", address, status, html)
		}
		var rendered []string
		for _, row := range searchRows.FindAllStringSubmatch(html, -1) {
			rendered = append(rendered, row[1])
		}
		if !slices.Equal(rendered, ids) {
			t.Fatalf("%s: rendered rows %v, want %v", address, rendered, ids)
		}
		_, payload, ok := strings.Cut(html, `<script type="application/json" data-shugo-props>`)
		if !ok {
			t.Fatalf("%s: hydration payload is missing", address)
		}
		payload, _, ok = strings.Cut(payload, "</script>")
		if !ok {
			t.Fatalf("%s: hydration payload has no end", address)
		}
		var props struct{ Initial json.RawMessage }
		var result searchResponse
		if err := json.Unmarshal([]byte(payload), &props); err != nil {
			t.Fatalf("%s: invalid island props: %v", address, err)
		}
		if err := json.Unmarshal(props.Initial, &result); err != nil {
			t.Fatalf("%s: invalid initial search result: %v", address, err)
		}
		if !strings.HasPrefix(result.Endpoint, "/_islands/search/") {
			t.Fatalf("%s: invalid binding endpoint %q", address, result.Endpoint)
		}
		_, body := api(result.Endpoint, query, ids...)
		var initial, fetched any
		if e1, e2 := json.Unmarshal(props.Initial, &initial), json.Unmarshal([]byte(body), &fetched); e1 != nil || e2 != nil || !reflect.DeepEqual(initial, fetched) {
			t.Fatalf("%s: hydration and API results differ: %s versus %s", address, props.Initial, body)
		}
		return result, html
	}

	products, _ := page("/products", nil, "p4", "p1")
	articles, firstArticleHTML := page("/articles", nil, "a2", "a3")
	if products.Endpoint == articles.Endpoint || products.Total != 5 || products.Pages != 3 ||
		products.Query.Sort != "cheap" || articles.Total != 4 || articles.Query.Sort != "short" {
		t.Fatalf("unexpected default bindings: products %+v, articles %+v", products, articles)
	}

	onlyCup := url.Values{"q": {"cups"}, "category": {"ceramics"}, "price_to": {"200"}, "available": {"1"}}
	filtered, _ := page("/products", onlyCup, "p1")
	f := filtered.Query.Filters
	if filtered.Query.Q != "cups" || !slices.Equal(f.Category, []string{"ceramics"}) ||
		f.Price.To == nil || *f.Price.To != 200 || f.Available == nil || !*f.Available {
		t.Fatalf("unexpected normalized filters: %+v", filtered.Query)
	}
	onlyGuide := url.Values{"q": {"cup"}, "category": {"guides"}, "price_from": {"5"}, "available": {"1"}}
	guide, _ := page("/articles", onlyGuide, "a1")
	if guide.Query.Filters.Price.From == nil || *guide.Query.Filters.Price.From != 5 {
		t.Fatalf("article duration filter: %+v", guide.Query)
	}
	unavailable, _ := api(products.Endpoint, url.Values{"category": {"wood", "ceramics"}, "available": {"0"}}, "p2")
	if unavailable.Query.Filters.Available == nil || *unavailable.Query.Filters.Available {
		t.Fatalf("explicit false filter: %+v", unavailable.Query)
	}
	descending, _ := api(products.Endpoint, url.Values{"sort": {"expensive"}, "page": {"2"}}, "p5", "p1")
	if descending.Page != 2 || descending.Query.Sort != "expensive" {
		t.Fatalf("sort and page: %+v", descending)
	}
	api(articles.Endpoint, url.Values{"q": {"flax"}})
	api(products.Endpoint, url.Values{"q": {"clay"}})
	if status, _, _ := get(t, ts, "/_islands/search/no-such-binding"); status != http.StatusNotFound {
		t.Fatalf("unregistered binding status: %d", status)
	}

	s.sql(`UPDATE products SET title = 'Salt Mug', price = 110 WHERE id = 'p1'`)
	api(products.Endpoint, onlyCup, "p1")
	if report := s.update(); report.Computed.Pages != 1 || report.Pages.Changed != 1 || report.SearchIndexes != 1 {
		t.Fatalf("a product update should only refresh its binding and reader, report %+v", report)
	}
	updated, html := page("/products", nil, "p4", "p1")
	if updated.Endpoint != products.Endpoint || !strings.Contains(html, ">Salt Mug</li>") {
		t.Fatalf("updated result missing from the existing binding %q", updated.Endpoint)
	}
	page("/products", onlyCup)
	_, after := page("/articles", nil, "a2", "a3")
	expect(t, after, firstArticleHTML)
}
