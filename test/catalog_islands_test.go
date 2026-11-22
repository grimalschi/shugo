package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/catalog"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
	"github.com/grimalschi/shugo/internal/store"
)

const catalogAddressPage = `
-- src/pages/[section]/catalog.svelte --
<script module>
  export const getStaticPaths = () => [{ params: { section: "_islands" } }]
</script>
<p>Site catalog page</p>
`

func TestCatalogSnapshot(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { search } from "shugo:catalog"
  const result = search()
</script>
<h1>{result.hits[0].title}</h1>
`)
	s.sql(`
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE Products (id TEXT PRIMARY KEY, slug TEXT, Title TEXT, category TEXT, price INTEGER, discount INTEGER, available INTEGER, arrival TEXT, description TEXT, position INTEGER);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT, title TEXT, stock INTEGER, position INTEGER);
INSERT INTO categories VALUES ('c', 'Category', 1);
WITH RECURSIVE ids(id) AS (VALUES (1) UNION ALL SELECT id + 1 FROM ids WHERE id < 25)
INSERT INTO products SELECT 'p' || id, 'item-' || id, 'Item ' || id, 'c', 100, NULL, 1, '2026-01-01', '', id FROM ids;
INSERT INTO variants VALUES ('sku', 'p1', 'large', 1, 1);
`)
	s.build()
	extension := server.NewCatalog(s.project().Path(project.Output))
	t.Cleanup(func() { extension.Close() })
	ts := httptest.NewServer(&server.Server{Pages: s.pages, Catalog: extension})
	defer ts.Close()

	query := func(search, want string) {
		t.Helper()
		status, header, body := get(t, ts, "/_islands/catalog"+search)
		if status != 200 || header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("catalog %s: %d %s", search, status, body)
		}
		var result catalog.Result
		if err := json.Unmarshal([]byte(body), &result); err != nil || len(result.Hits) == 0 || result.Hits[0].Title != want {
			t.Fatalf("catalog %s: %v, want first hit %s, response %s", search, err, want, body)
		}
	}
	query("", "Item 1")
	query("?page=2", "Item 25")
	expect(t, s.page("/"), "<h1>Item 1</h1>")
	output := s.project().Path(project.Output)
	var firstSnapshot string
	if err := s.store.View(func(v *store.View) error {
		name, err := v.Catalog()
		firstSnapshot = name
		return err
	}); err != nil {
		t.Fatal(err)
	}
	dbFile, err := os.Open(filepath.Join(output, firstSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	var header [16]byte
	_, err = io.ReadFull(dbFile, header[:])
	dbFile.Close()
	if err != nil || filepath.Ext(firstSnapshot) != ".sqlite" || string(header[:]) != "SQLite format 3\x00" {
		t.Fatalf("unexpected published catalog %q: header %q, error %v", firstSnapshot, header, err)
	}
	response, err := http.Head(ts.URL + "/_islands/catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if body, err := io.ReadAll(response.Body); err != nil || response.StatusCode != 200 || len(body) != 0 || response.ContentLength <= 0 {
		t.Fatalf("HEAD: %d, length %d, body %q, error %v", response.StatusCode, response.ContentLength, body, err)
	}

	s.sql(`UPDATE products SET title = 'Flax Cup' WHERE id = 'p25'`)
	query("?page=2", "Item 25") // the advertised snapshot doesn't read the source directly
	if report := s.update(); report.Computed.Pages != 0 || report.CatalogRecomputed {
		t.Fatalf("change outside the initial result: %+v", report)
	}
	query("?page=2", "Flax Cup")
	query("?q="+url.QueryEscape("cups"), "Flax Cup")
	expect(t, s.page("/"), "<h1>Item 1</h1>")

	s.sql(`UPDATE products SET title = 'Changed first' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 1 || !report.CatalogRecomputed {
		t.Fatalf("change in the initial result: %+v", report)
	}
	query("", "Changed first")
	expect(t, s.page("/"), "<h1>Changed first</h1>")

	s.sql(`DROP TABLE variants`)
	if _, err := s.tryUpdate(); err == nil || !strings.Contains(err.Error(), "catalog projection") {
		t.Fatalf("expected catalog projection error, got %v", err)
	}
	query("", "Changed first")
	expect(t, s.page("/"), "<h1>Changed first</h1>")

	s.write(`
-- src/pages/index.svelte --
<h1>Static shop</h1>
` + catalogAddressPage)
	s.build()
	expect(t, s.page("/"), "<h1>Static shop</h1>")
	expectNoCatalogIndex(t, s, ts)
}

func TestCatalogUnusedSearch(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { search } from "shugo:catalog"
</script>
<p>Regular page</p>
`+catalogAddressPage)
	s.sql(`CREATE TABLE notes (id TEXT PRIMARY KEY, body TEXT)`)
	s.build()
	expect(t, s.page("/"), "<p>Regular page</p>")
	ts := httptest.NewServer(&server.Server{
		Pages: s.pages, Catalog: server.NewCatalog(s.project().Path(project.Output)),
	})
	defer ts.Close()
	expectNoCatalogIndex(t, s, ts)
}

func expectNoCatalogIndex(t *testing.T, s *site, ts *httptest.Server) {
	t.Helper()
	if err := s.store.View(func(v *store.View) error {
		name, err := v.Catalog()
		if name != "" {
			t.Errorf("unexpected published catalog index %q", name)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	status, header, body := get(t, ts, "/_islands/catalog")
	if status != http.StatusOK || !strings.HasPrefix(header.Get("Content-Type"), "text/html") {
		t.Fatalf("site catalog page: %d, %s, %s", status, header.Get("Content-Type"), body)
	}
	expect(t, body, "<p>Site catalog page</p>")
}
