package test

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/evaluate"
	"github.com/grimalschi/shugo/internal/store"
)

func TestSearchBindingsShareIndexAndTrackTheirReaders(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/cups.svelte --
<script>
  import { search } from "shugo:search"
  const cups = search("products", { text: ["title"], where: { category: "cups" }, pageSize: 1 })
</script>
<a href={cups.endpoint}>{cups.hits[0].title}</a>
-- src/pages/plates.svelte --
<script>
  import { search } from "shugo:search"
  const plates = search("products", { text: ["title"], where: { category: "plates" }, pageSize: 2 })
</script>
<a href={plates.endpoint}>{plates.hits[0].title}</a>
`)
	s.sql(`
CREATE TABLE products (
  id INTEGER PRIMARY KEY, name TEXT, category TEXT, internal_note TEXT,
  title TEXT GENERATED ALWAYS AS (name || ' in stock')
);
INSERT INTO products (name, category) VALUES ('Cup', 'cups'), ('Plate', 'plates');
`)
	if report := s.build(); report.SearchIndexes != 1 {
		t.Fatalf("two bindings should build one index, got %+v", report)
	}
	checkPage := func(page, title string) string {
		t.Helper()
		html := s.page(page)
		_, href, ok := strings.Cut(html, `href="/_islands/search/`)
		id, text, _ := strings.Cut(href, `">`)
		if !ok || !strings.HasPrefix(text, title+"</a>") {
			t.Fatalf("%s rendered %q", page, html)
		}
		return id
	}
	cupID := checkPage("/cups", "Cup in stock")
	plateID := checkPage("/plates", "Plate in stock")
	if cupID == plateID {
		t.Fatal("separate scopes share the same binding")
	}
	binding := func(id string) (store.SearchBinding, bool) {
		t.Helper()
		var out store.SearchBinding
		var found bool
		if err := s.store.View(func(v *store.View) (err error) {
			out, found, err = v.Search(id)
			return
		}); err != nil {
			t.Fatal(err)
		}
		return out, found
	}
	cups, cupFound := binding(cupID)
	plates, plateFound := binding(plateID)
	if !cupFound || !plateFound || cups.Snapshot != plates.Snapshot {
		t.Fatalf("scoped bindings should share a snapshot, got %q and %q", cups.Snapshot, plates.Snapshot)
	}

	s.sql(`UPDATE products SET name = 'New cup' WHERE category = 'cups'`)
	if report := s.update(); report.SearchIndexes != 1 || report.Computed.Pages != 1 || report.Pages.Changed != 1 {
		t.Fatalf("generated field update should rebuild once and recompute its reader, got %+v", report)
	}
	checkPage("/cups", "New cup in stock")
	checkPage("/plates", "Plate in stock")

	s.remove("src/pages/cups.svelte")
	s.build()
	if _, found := binding(cupID); found {
		t.Fatal("removed search binding is still published")
	}
	if _, found := binding(plateID); !found {
		t.Fatal("remaining search binding was removed")
	}
}

func TestSearchDuringModuleInitialization(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/lib/articles.js --
import { search } from "shugo:search";
export const articles = search("articles", { fields: ["title"] });
-- src/pages/index.svelte --
<script>
  import { articles } from "../lib/articles.js"
</script>
<p>{articles.hits[0].title}</p>
`)
	s.sql(`CREATE TABLE articles(id INTEGER PRIMARY KEY, title TEXT, internal_note TEXT);
INSERT INTO articles(title) VALUES ('An article');`)
	s.build()
	expect(t, s.page("/"), "<p>An article</p>")
	s.sql(`UPDATE articles SET internal_note = 'draft notes' WHERE id = 1`)
	if report := s.update(); report.SearchIndexes != 0 || report.Computed.Pages != 0 {
		t.Fatalf("unprojected column change reevaluated the search: %+v", report)
	}
	s.sql(`UPDATE articles SET title = 'New title' WHERE id = 1`)
	s.update()
	expect(t, s.page("/"), "<p>New title</p>")
}

// A search is published while a page asks for it: a page that stops
// asking takes its binding away, and one that asks again brings it back.
func TestSearchBindingRemovedWhenThePageStopsSearching(t *testing.T) {
	s := newSite(t, `
-- shugo.json --
{"data":"data/shop.db"}
-- src/pages/[slug].svelte --
<script module>
  import { query } from "shugo"
  import { search } from "shugo:search"
  export const getStaticPaths = () => [{ params: { slug: "articles" } }]
</script>
<script>
  const initial = query("settings").get(1).enabled ? search("articles", { fields: ["title"] }) : null
</script>
{#if initial}<a href={initial.endpoint}>{initial.hits[0].title}</a>{:else}<p>Search disabled</p>{/if}
`)
	s.sql(`CREATE TABLE articles(id INTEGER PRIMARY KEY, title TEXT);
INSERT INTO articles(title) VALUES ('An article');
CREATE TABLE settings(id INTEGER PRIMARY KEY, enabled INTEGER);
INSERT INTO settings VALUES (1, 1);`)
	s.build()
	html := s.page("/articles")
	_, href, ok := strings.Cut(html, `href="/_islands/search/`)
	id, _, _ := strings.Cut(href, `">`)
	if !ok || id == "" {
		t.Fatalf("expected a search binding in %q", html)
	}
	s.sql(`UPDATE settings SET enabled = 0 WHERE id = 1`)
	if report := s.update(); report.Computed.Pages != 1 || report.SearchIndexes != 0 {
		t.Fatalf("expected the existing page to drop its search, got %+v", report)
	}
	expect(t, s.page("/articles"), "<p>Search disabled</p>")
	if err := s.store.View(func(v *store.View) error {
		_, found, err := v.Search(id)
		if found {
			t.Error("unused binding is still published")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	s.sql(`UPDATE settings SET enabled = 1 WHERE id = 1`)
	report, err := s.run(evaluate.Options{Check: true})
	if err != nil || len(report.Unseen) != 0 || report.SearchIndexes != 1 {
		t.Fatalf("check should foresee the page restoring its search, got %+v, %v", report, err)
	}
	expect(t, s.page("/articles"), html)
}
