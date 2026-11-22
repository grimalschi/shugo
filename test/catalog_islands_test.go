package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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

func TestCatalogPriceRefresh(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { search } from "shugo:catalog"
  if (Shugo.query("products").select("note").get("p1").note === "fail") {
    throw new Error("price refresh test guard")
  }
  const result = search()
</script>
<h1>{result.hits[0].title}: {result.hits[0].priceLabel}</h1>
`)
	s.sql(`
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT, title TEXT, category TEXT, price INTEGER, discount INTEGER, available INTEGER, arrival TEXT, description TEXT, position INTEGER, note TEXT);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT, title TEXT, position INTEGER);
INSERT INTO categories VALUES ('c1', 'Tableware', 1), ('c2', 'Decor', 2);
INSERT INTO products VALUES
  ('p1', 'wave', 'Wave Cup', 'c1', 120, NULL, 1, '2026-01-01', 'handmade', 1, ''),
  ('p2', 'flax', 'Flax Cup', 'c1', 180, NULL, 1, '2026-01-01', 'handmade', 2, '');
INSERT INTO variants VALUES ('v1', 'p1', 'small', 1);
`)
	s.build()

	output := s.project().Path(project.Output)
	extension := server.NewCatalog(output)
	t.Cleanup(func() { extension.Close() })
	ts := httptest.NewServer(&server.Server{Pages: s.pages, Catalog: extension})
	defer ts.Close()

	generation := func() string {
		t.Helper()
		var name string
		if err := s.store.View(func(v *store.View) error {
			var err error
			name, err = v.Catalog()
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return name
	}
	type pricedCard struct {
		id              string
		price, original int64
	}
	assertPrices := func(label string, result catalog.Result, want ...pricedCard) {
		t.Helper()
		got := make([]pricedCard, 0, len(result.Hits))
		for _, hit := range result.Hits {
			got = append(got, pricedCard{hit.ID, hit.Price, hit.OriginalPrice})
		}
		if result.Total != len(want) || !slices.Equal(got, want) {
			t.Fatalf("%s: cards %v (total %d), want %v", label, got, result.Total, want)
		}
	}
	query := func(values url.Values, want ...pricedCard) {
		t.Helper()
		status, _, body := get(t, ts, "/_islands/catalog?"+values.Encode())
		if status != http.StatusOK {
			t.Fatalf("catalog %v: status %d, response %s", values, status, body)
		}
		var result catalog.Result
		if err := json.Unmarshal([]byte(body), &result); err != nil {
			t.Fatalf("catalog %v: decode response: %v", values, err)
		}
		if !slices.Equal(result.Filters.Size, values["size"]) {
			t.Fatalf("catalog %v: normalized size filter %v", values, result.Filters.Size)
		}
		assertPrices(values.Encode(), result, want...)
	}

	firstName := generation()
	first, err := catalog.Open(filepath.Join(output, firstName))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	cheapCups := url.Values{"q": {"cups"}, "sort": {"cheap"}}
	limitedCups := url.Values{"q": {"cups"}, "sort": {"cheap"}, "price_to": {"120"}}
	query(cheapCups, pricedCard{"p1", 120, 120}, pricedCard{"p2", 180, 180})
	query(limitedCups, pricedCard{"p1", 120, 120})

	s.sql(`UPDATE products SET price = 150, note = 'memo' WHERE id = 'p1';
UPDATE products SET discount = 60 WHERE id = 'p2';`)
	query(limitedCups, pricedCard{"p1", 120, 120})
	if report := s.update(); report.CatalogPrices != 2 || !report.CatalogRecomputed {
		t.Fatalf("expected two price deltas and a refreshed catalog page, report %+v", report)
	}
	secondName := generation()
	if secondName == firstName {
		t.Fatal("price refresh did not publish a new catalog generation")
	}
	query(cheapCups, pricedCard{"p2", 72, 180}, pricedCard{"p1", 150, 150})
	query(limitedCups, pricedCard{"p2", 72, 180})
	expect(t, s.page("/"), "<h1>Wave Cup: $150</h1>")
	prior, err := first.Search(t.Context(), limitedCups)
	if err != nil {
		t.Fatal(err)
	}
	assertPrices("previous generation after publication", prior, pricedCard{"p1", 120, 120})

	s.sql(`UPDATE products SET note = 'memo updated' WHERE id = 'p1'`)
	if report := s.update(); report.CatalogPrices != 0 || report.CatalogRecomputed || generation() != secondName {
		t.Fatalf("note update changed the catalog generation, report %+v", report)
	}

	s.sql(`UPDATE products SET price = 80, note = 'fail' WHERE id = 'p1'`)
	if _, err := s.tryUpdate(); err == nil || !strings.Contains(err.Error(), "price refresh test guard") {
		t.Fatalf("expected guarded page evaluation to fail, got %v", err)
	}
	if now := generation(); now != secondName {
		t.Fatalf("failed update published %q, previously %q", now, secondName)
	}
	query(limitedCups, pricedCard{"p2", 72, 180})
	expect(t, s.page("/"), "<h1>Wave Cup: $150</h1>")

	s.sql(`UPDATE products SET note = 'memo updated' WHERE id = 'p1'`)
	if report := s.update(); report.CatalogPrices != 1 {
		t.Fatalf("retried update did not apply its price delta, report %+v", report)
	}
	query(limitedCups, pricedCard{"p2", 72, 180}, pricedCard{"p1", 80, 80})
	expect(t, s.page("/"), "<h1>Wave Cup: $80</h1>")

	for _, change := range []struct {
		name, sql string
		filter    url.Values
		want      []pricedCard
	}{
		{"title", `UPDATE products SET title = 'Wave Vase', price = 79 WHERE id = 'p1'`,
			url.Values{"q": {"vases"}}, []pricedCard{{"p1", 79, 79}}},
		{"variant", `UPDATE variants SET title = 'large' WHERE sku = 'v1';
UPDATE products SET price = 82 WHERE id = 'p1';`,
			url.Values{"q": {"vases"}, "size": {"large"}}, []pricedCard{{"p1", 82, 82}}},
		{"category", `UPDATE products SET category = 'c2', price = 84 WHERE id = 'p1'`,
			url.Values{"q": {"vases"}, "category": {"c2"}}, []pricedCard{{"p1", 84, 84}}},
		{"insert", `INSERT INTO products VALUES ('p3', 'north', 'North Vase', 'c2', 50, NULL, 1, '2026-01-01', '', 3, '');
UPDATE products SET price = 86 WHERE id = 'p1';`,
			url.Values{"q": {"vases"}, "sort": {"cheap"}}, []pricedCard{{"p3", 50, 50}, {"p1", 86, 86}}},
		{"delete", `DELETE FROM products WHERE id = 'p3'; UPDATE products SET price = 88 WHERE id = 'p1'`,
			url.Values{"q": {"vases"}}, []pricedCard{{"p1", 88, 88}}},
	} {
		s.sql(change.sql)
		if report := s.update(); report.CatalogPrices != 0 {
			t.Fatalf("%s change used price-only refresh, report %+v", change.name, report)
		}
		query(change.filter, change.want...)
	}
	query(cheapCups, pricedCard{"p2", 72, 180})

	if err := os.Remove(filepath.Join(output, generation())); err != nil {
		t.Fatal(err)
	}
	if report := s.update(); report.CatalogPrices != 0 || report.CatalogRecomputed || report.Computed.Pages != 0 {
		t.Fatalf("repairing an unchanged catalog should keep its default result, report %+v", report)
	}
	restored, err := catalog.Open(filepath.Join(output, generation()))
	if err != nil {
		t.Fatalf("no-op update did not restore the missing catalog: %v", err)
	}
	restoredResult, err := restored.Search(t.Context(), cheapCups)
	restored.Close()
	if err != nil {
		t.Fatal(err)
	}
	assertPrices("restored generation", restoredResult, pricedCard{"p2", 72, 180})

	if err := os.Remove(filepath.Join(output, generation())); err != nil {
		t.Fatal(err)
	}
	s.sql(`UPDATE products SET price = 89 WHERE id = 'p1'`)
	if report := s.update(); report.CatalogPrices != 0 {
		t.Fatalf("missing previous generation used price-only refresh, report %+v", report)
	}
	query(url.Values{"q": {"vases"}}, pricedCard{"p1", 89, 89})
}

func TestCatalogConcurrentWrite(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { existsSync, renameSync } from "node:fs"
  import { search } from "shugo:catalog"
  if (existsSync("catalog-race.arm")) {
    renameSync("catalog-race.arm", "catalog-race.ready")
    const deadline = Date.now() + 10000
    const barrier = new Int32Array(new SharedArrayBuffer(4))
    while (!existsSync("catalog-race.done")) {
      if (Date.now() >= deadline) throw new Error("catalog race writer timed out")
      Atomics.wait(barrier, 0, 0, 5)
    }
  }
  const product = Shugo.query("products").get("p1")
  const result = search()
</script>
<h1>source {product.price} index {result.hits[0].price}</h1>
`)
	s.sql(`
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT, title TEXT, category TEXT, price INTEGER, discount INTEGER, available INTEGER, arrival TEXT, description TEXT, position INTEGER);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT, title TEXT, position INTEGER);
INSERT INTO categories VALUES ('c1', 'Tableware', 1);
INSERT INTO products VALUES ('p1', 'wave', 'Wave Cup', 'c1', 100, NULL, 1, '2026-01-01', 'handmade', 1);
`)
	s.build()
	expect(t, s.page("/"), "<h1>source 100 index 100</h1>")
	s.sql(`UPDATE products SET price = 150 WHERE id = 'p1'`)
	if err := os.WriteFile(filepath.Join(s.root, "catalog-race.arm"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", "file:"+s.project().Data()+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	written := make(chan error, 1)
	go func() {
		written <- func() error {
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()
			for {
				_, err := os.Stat(filepath.Join(s.root, "catalog-race.ready"))
				if err == nil {
					break
				}
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
			if _, err := db.ExecContext(ctx, "UPDATE products SET price = 170 WHERE id = 'p1'"); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(s.root, "catalog-race.done"), nil, 0o600)
		}()
	}()
	report, updateErr := s.tryUpdate()
	cancel()
	if err := <-written; err != nil {
		t.Fatalf("catalog race writer: %v", err)
	}
	if updateErr != nil {
		t.Fatalf("update with a concurrent write: %v", updateErr)
	}
	if report.CatalogRetries != 1 {
		t.Fatalf("expected one retry after the catalog source moved, report %+v", report)
	}
	expect(t, s.page("/"), "<h1>source 170 index 170</h1>")
	generations, err := filepath.Glob(s.project().Path(project.Output, "catalog-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(generations) != 2 {
		t.Fatalf("initial and committed generations expected, got %v", generations)
	}

	s.sql(`UPDATE products SET price = 150 WHERE id = 'p1'`)
	s.update()
	expect(t, s.page("/"), "<h1>source 150 index 150</h1>")
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
