package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/store"
)

const migrationCatalogData = `
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT, title TEXT, category TEXT, price INTEGER, discount INTEGER, available INTEGER, arrival TEXT, description TEXT, position INTEGER, note TEXT);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT, title TEXT, position INTEGER);
CREATE TABLE settings (id INTEGER PRIMARY KEY, show_catalog INTEGER);
INSERT INTO categories VALUES ('books', 'Books', 1);
INSERT INTO products VALUES
  ('p1', 'first', 'First', 'books', 100, NULL, 1, '2026-01-01', '', 1, 'original'),
  ('p2', 'second', 'Second', 'books', 200, NULL, 1, '2026-01-01', '', 2, '');
INSERT INTO settings VALUES (1, 1);
`

func TestSearchAndQueryReadersUpdatePrecisely(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/_components/Summary.svelte --
<script>
  import { search } from "shugo:catalog"
  const enabled = Shugo.query("settings").get(1).show_catalog
  const result = enabled ? search() : null
</script>
{#if result}<p>{result.hits[0].title}</p>{:else}<p>Paused</p>{/if}
-- src/pages/index.svelte --
<script>
  import Summary from "./_components/Summary.svelte"
</script>
<Summary />
-- src/pages/archive.svelte --
<script>
  import Summary from "./_components/Summary.svelte"
</script>
<Summary />
-- src/pages/notes.svelte --
<script>
  const product = Shugo.query("products").get("p1")
</script>
<p>{product.note}</p>
`)
	s.sql(migrationCatalogData)
	first := s.build()
	if first.Pages.Total != 3 || first.Shared.Total != 1 || s.has("/_components/Summary") {
		t.Fatalf("expected a single shared component with no route: %+v", first)
	}
	expect(t, s.page("/"), "<p>First</p>")
	expect(t, s.page("/archive"), "<p>First</p>")

	snapshot := func() os.FileInfo {
		t.Helper()
		var name string
		if err := s.store.View(func(v *store.View) error {
			var err error
			name, err = v.Catalog()
			return err
		}); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(s.project().Path(project.Output, name))
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	index := snapshot()
	s.sql(`UPDATE products SET note = 'edited' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 1 || report.Computed.Shared != 0 || report.CatalogRecomputed || !os.SameFile(index, snapshot()) {
		t.Fatalf("an unused catalog column only reaches its ordinary query reader: %+v", report)
	}
	expect(t, s.page("/notes"), "<p>edited</p>")

	s.sql(`UPDATE products SET title = 'Renamed' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 1 || !report.CatalogRecomputed || report.Render != 2 {
		t.Fatalf("a new search result reaches the one shared reader and renders its two pages: %+v", report)
	}
	expect(t, s.page("/"), "<p>Renamed</p>")
	expect(t, s.page("/archive"), "<p>Renamed</p>")

	s.sql(`UPDATE settings SET show_catalog = 0 WHERE id = 1`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 1 || report.CatalogRecomputed {
		t.Fatalf("only the shared block read the setting: %+v", report)
	}
	expect(t, s.page("/archive"), "<p>Paused</p>")

	s.sql(`UPDATE products SET title = 'Changed while hidden' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 0 || report.CatalogRecomputed {
		t.Fatalf("the old catalog reading should have been removed: %+v", report)
	}
}

func TestSearchCanRelistOnlyItsOwnRoute(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<h1>Home</h1>
-- src/pages/featured/[slug].svelte --
<script module>
  import { search } from "shugo:catalog"
  export function getStaticPaths() {
    return [{ params: { slug: search().hits[0].slug } }]
  }
</script>
<h1>{Shugo.params.slug}</h1>
`)
	s.sql(migrationCatalogData)
	s.build()
	expect(t, s.page("/featured/first"), "<h1>first</h1>")

	s.sql(`UPDATE products SET position = 3 WHERE id = 'p1'`)
	report := s.update()
	if report.Data.Routes != 1 || report.Computed.Pages != 1 || report.Pages.Added != 1 || report.Pages.Removed != 1 || !report.CatalogRecomputed {
		t.Fatalf("only the listing that used search should be reconsidered: %+v", report)
	}
	expect(t, s.page("/featured/second"), "<h1>second</h1>")
	if s.has("/featured/first") {
		t.Fatal("the previous address is still published")
	}
	expect(t, s.page("/"), "<h1>Home</h1>")
}

const searchTitlePage = `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { search } from "shugo:catalog"
  const result = search()
</script>
<h1>{result.hits[0].title}</h1>
`

func TestSearchTracksIndirectProjectionColumns(t *testing.T) {
	for _, test := range []struct {
		name, schema, change string
	}{
		{
			name: "view",
			schema: `ALTER TABLE products RENAME TO inventory;
CREATE VIEW products AS SELECT id, slug, note AS title, category, price, discount, available, arrival, description, position FROM inventory;`,
			change: "UPDATE inventory SET note = 'Updated' WHERE id = 'p1'",
		},
		{
			name: "generated column",
			schema: `ALTER TABLE products RENAME COLUMN title TO original_title;
ALTER TABLE products ADD COLUMN title TEXT GENERATED ALWAYS AS (note) VIRTUAL;`,
			change: "UPDATE products SET note = 'Updated' WHERE id = 'p1'",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := newSite(t, searchTitlePage)
			s.sql(migrationCatalogData + test.schema)
			s.build()
			expect(t, s.page("/"), "<h1>original</h1>")

			s.sql(test.change)
			if report := s.update(); report.Computed.Pages != 1 || !report.CatalogRecomputed {
				t.Fatalf("expected the catalog projection to follow the underlying change: %+v", report)
			}
			expect(t, s.page("/"), "<h1>Updated</h1>")
		})
	}
}

func TestSearchRefreshesWhenConcurrentWritesCancelOut(t *testing.T) {
	s := newSite(t, searchTitlePage)
	s.sql(migrationCatalogData)
	s.build()
	expect(t, s.page("/"), "<h1>First</h1>")

	s.sql("UPDATE products SET title = 'Final' WHERE id = 'p1'")
	source, err := data.Open(s.project().Data())
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	at, err := source.Position()
	if err != nil {
		t.Fatal(err)
	}
	s.sql("UPDATE products SET title = 'First' WHERE id = 'p1'; UPDATE products SET title = 'Final' WHERE id = 'p1'")

	// Model a publication whose catalog read the transient title after the
	// journal cursor was captured. The store still displays that title.
	update, err := s.store.Update()
	if err != nil {
		t.Fatal(err)
	}
	defer update.Rollback()
	if err := update.SetData(fmt.Sprintf("%s %d true", source.ID, at)); err != nil {
		t.Fatal(err)
	}
	if err := update.Commit(); err != nil {
		t.Fatal(err)
	}
	if report := s.update(); report.Data.Rows != 0 || !report.Data.Whole || report.Computed.Pages != 1 {
		t.Fatalf("an unsettled legacy publication must refresh after coalesced changes: %+v", report)
	}
	expect(t, s.page("/"), "<h1>Final</h1>")
	if report := s.update(); report.Computed.Pages != 0 || report.CatalogRecomputed {
		t.Fatalf("the pending refresh should be cleared: %+v", report)
	}
}

func TestModuleScopeSearchKeepsDependency(t *testing.T) {
	s := newSite(t, strings.Replace(searchTitlePage, "<script>", "<script module>", 1)+`
-- src/pages/other.svelte --
<p>Other page</p>
`)
	s.sql(migrationCatalogData)
	s.build()
	expect(t, s.page("/"), "<h1>First</h1>")

	s.sql("UPDATE products SET title = 'Updated' WHERE id = 'p1'")
	if report := s.update(); report.Computed.Pages != 2 || report.Pages.Changed != 1 || !report.CatalogRecomputed {
		t.Fatalf("the module initializer is tracked conservatively, with only changed output published: %+v", report)
	}
	expect(t, s.page("/"), "<h1>Updated</h1>")
}

func TestCatalogBuildInputExceedsEnvironmentValueLimit(t *testing.T) {
	s := newSite(t, strings.Replace(searchTitlePage, "<script>", "<script module>", 1))
	s.sql(migrationCatalogData + "UPDATE categories SET title = hex(zeroblob(100000));")
	s.build()
	expect(t, s.page("/"), "<h1>First</h1>")
	s.sql("UPDATE products SET title = 'Updated' WHERE id = 'p1'")
	s.update()
	expect(t, s.page("/"), "<h1>Updated</h1>")
}

func TestLegacySourceRequiresRenameWithoutDroppingThePublishedSite(t *testing.T) {
	s := newSite(t, `
-- src/pages/index.svelte --
<p>Published</p>
`)
	s.build()
	s.write(`
-- src/pages/old.shugo --
<p>Legacy source</p>
`)
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "old.svelte") || !strings.Contains(err.Error(), "old.shugo") {
		t.Fatalf("expected a migration diagnostic that names both paths, got %v", err)
	}
	expect(t, s.page("/"), "<p>Published</p>")
	if s.has("/old") {
		t.Fatal("a failed build published a new page")
	}
}
