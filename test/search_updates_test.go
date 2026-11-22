package test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grimalschi/shugo/internal/project"
)

// How an update follows the searches of a site: the first results of a
// search are read like rows, and a change reaches the readers of what it
// changed.

const searchShop = `
CREATE TABLE products (id TEXT PRIMARY KEY, slug TEXT, title TEXT, price INTEGER, position INTEGER, note TEXT);
CREATE TABLE settings (id INTEGER PRIMARY KEY, show_search INTEGER);
INSERT INTO products VALUES ('p1', 'first', 'First', 100, 1, 'original'), ('p2', 'second', 'Second', 200, 2, '');
INSERT INTO settings VALUES (1, 1);
`

// firstProduct is a search whose first result is the product of the
// smallest position.
const firstProduct = `search("products", { text: ["title"], fields: ["slug", "price"], sorts: { first: { position: "asc" } }, defaultSort: "first", pageSize: 1 })`

func TestSearchAndQueryReadersUpdatePrecisely(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/_components/Summary.svelte --
<script>
  import { query, search } from "shugo"
  const enabled = query("settings").get(1).show_search
  const result = enabled ? `+firstProduct+` : null
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
  import { query } from "shugo"
  const product = query("products").get("p1")
</script>
<p>{product.note}</p>
`)
	s.sql(searchShop)
	first := s.build()
	if first.Pages.Total != 3 || first.Shared.Total != 1 || s.has("/_components/Summary") {
		t.Fatalf("expected one shared component and no address for it: %+v", first)
	}
	expect(t, s.page("/"), "<p>First</p>")
	expect(t, s.page("/archive"), "<p>First</p>")

	// A column that no search keeps reaches its query alone.
	s.sql(`UPDATE products SET note = 'edited' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 1 || report.Computed.Shared != 0 || report.SearchIndexes != 0 {
		t.Fatalf("a column outside the search reaches its query alone: %+v", report)
	}
	expect(t, s.page("/notes"), "<p>edited</p>")

	// New first results reach the one shared reader, shown on two pages.
	s.sql(`UPDATE products SET title = 'Renamed' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 1 || report.SearchIndexes != 1 || report.Render != 2 {
		t.Fatalf("new first results reach the shared reader: %+v", report)
	}
	expect(t, s.page("/"), "<p>Renamed</p>")
	expect(t, s.page("/archive"), "<p>Renamed</p>")

	// The setting is read by the shared block alone; once it stops
	// searching, a change of the products reaches nobody.
	s.sql(`UPDATE settings SET show_search = 0 WHERE id = 1`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 1 {
		t.Fatalf("only the shared block read the setting: %+v", report)
	}
	expect(t, s.page("/archive"), "<p>Paused</p>")
	s.sql(`UPDATE products SET title = 'Changed while hidden' WHERE id = 'p1'`)
	if report := s.update(); report.Computed.Pages != 0 || report.Computed.Shared != 0 || report.SearchIndexes != 0 {
		t.Fatalf("a search nobody asks for is not kept: %+v", report)
	}
}

// The list of pages of a route may come from a search: new first results
// list that route again, and no other.
func TestSearchCanRelistOnlyItsOwnRoute(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<h1>Home</h1>
-- src/pages/featured/[slug].svelte --
<script module>
  import { page, search } from "shugo"
  export function getStaticPaths() {
    return [{ params: { slug: `+firstProduct+`.hits[0].slug } }]
  }
</script>
<h1>{page.params.slug}</h1>
`)
	s.sql(searchShop)
	s.build()
	expect(t, s.page("/featured/first"), "<h1>first</h1>")

	s.sql(`UPDATE products SET position = 3 WHERE id = 'p1'`)
	report := s.update()
	if report.Data.Routes != 1 || report.Computed.Pages != 1 || report.Pages.Added != 1 || report.Pages.Removed != 1 {
		t.Fatalf("only the listing that searched is listed again: %+v", report)
	}
	expect(t, s.page("/featured/second"), "<h1>second</h1>")
	if s.has("/featured/first") {
		t.Fatal("the previous address is still published")
	}
	expect(t, s.page("/"), "<h1>Home</h1>")
}

// A search asked at the top of a module belongs to every computation: its
// results may be kept in the module and shown anywhere. Only what comes out
// different is published.
func TestModuleScopeSearchKeepsDependency(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script module>
  import { search } from "shugo"
  const result = `+firstProduct+`
</script>
<h1>{result.hits[0].title}</h1>
-- src/pages/other.svelte --
<p>Other page</p>
`)
	s.sql(searchShop)
	s.build()
	expect(t, s.page("/"), "<h1>First</h1>")

	s.sql("UPDATE products SET title = 'Updated' WHERE id = 'p1'")
	if report := s.update(); report.Computed.Pages != 2 || report.Pages.Changed != 1 {
		t.Fatalf("a search at the top of a module reaches every page, and only the changed one is published: %+v", report)
	}
	expect(t, s.page("/"), "<h1>Updated</h1>")
}

// The index of a search and the pages read the data database on connections
// of their own. A write between those reads would leave them at two states
// of the data: the update starts over and publishes one state.
func TestSearchWithConcurrentWrite(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/index.svelte --
<script>
  import { existsSync, renameSync } from "node:fs"
  import { query, search } from "shugo"
  if (existsSync("search-race.arm")) {
    renameSync("search-race.arm", "search-race.ready")
    const deadline = Date.now() + 10000
    const barrier = new Int32Array(new SharedArrayBuffer(4))
    while (!existsSync("search-race.done")) {
      if (Date.now() >= deadline) throw new Error("the writer of the race timed out")
      Atomics.wait(barrier, 0, 0, 5)
    }
  }
  const product = query("products").get("p1")
  const result = `+firstProduct+`
</script>
<h1>source {product.price} index {result.hits[0].price}</h1>
`)
	s.sql(searchShop)
	s.build()
	expect(t, s.page("/"), "<h1>source 100 index 100</h1>")
	s.sql(`UPDATE products SET price = 150 WHERE id = 'p1'`)
	if err := os.WriteFile(filepath.Join(s.root, "search-race.arm"), nil, 0o600); err != nil {
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
				_, err := os.Stat(filepath.Join(s.root, "search-race.ready"))
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
			return os.WriteFile(filepath.Join(s.root, "search-race.done"), nil, 0o600)
		}()
	}()
	report, updateErr := s.tryUpdate()
	cancel()
	if err := <-written; err != nil {
		t.Fatalf("the writer of the race: %v", err)
	}
	if updateErr != nil {
		t.Fatalf("an update with a concurrent write: %v", updateErr)
	}
	if report.SearchRetries != 1 {
		t.Fatalf("expected one more attempt after the data changed, report %+v", report)
	}
	expect(t, s.page("/"), "<h1>source 170 index 170</h1>")
	indices, err := filepath.Glob(s.project().Path(project.Output, "search-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) == 0 || len(indices) > 2 {
		t.Fatalf("the first index and the published one, got %v", indices)
	}

	s.sql(`UPDATE products SET price = 150 WHERE id = 'p1'`)
	s.update()
	expect(t, s.page("/"), "<h1>source 150 index 150</h1>")
}
