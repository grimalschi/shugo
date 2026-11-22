package search

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const declaration = `{
  "table":"inventory",
  "options":{
    "text":["heading","description"],
    "where":{"section":"one","published":true},
    "fields":["slug"],
    "filters":{
      "kind":"multi",
      "cost":"range",
      "in_stock":{"type":"boolean","column":"available","label":"In stock"}
    },
    "sorts":{
      "cheap":{"cost":"asc","rank":"desc"},
      "expensive":{"by":[{"column":"cost","direction":"desc"}],"label":"Expensive first"}
    },
    "defaultSort":"cheap",
    "pageSize":2
  }
}`

const fixture = `
CREATE TABLE inventory (
	sku TEXT PRIMARY KEY, slug TEXT, heading TEXT, description TEXT,
	kind TEXT, cost REAL, available INTEGER, rank INTEGER, section TEXT,
	published INTEGER, secret TEXT
);
INSERT INTO inventory VALUES
	('a','tea','Flax Cup','handmade','cups',15.5,1,1,'one',1,'private-a'),
	('b','coffee','North Cup','handmade','cups',30.5,0,1,'one',1,'private-b'),
	('c','tray','Flax Tray','glaze','serve',8.75,1,2,'one',1,'private-c'),
	('d','private','Salt Cup','glaze','other-scope',1,1,2,'two',1,'private-d'),
	('e','invisible','Quiet Cup','handmade','secret-facet',1,1,2,'one',0,'private-e'),
	('f','unlisted','Leaf Cup','glaze','invisible',1,1,2,'one',NULL,'private-f');
CREATE TABLE notes (id INTEGER PRIMARY KEY, text TEXT);
INSERT INTO notes VALUES (1, 'handmade'), (2, 'matte glaze');
CREATE TABLE events (id INTEGER PRIMARY KEY, happened_at DATETIME);
INSERT INTO events VALUES (1, '2026-01-01 03:20:00'), (2, NULL);
`

func definition(t *testing.T, text string) Definition {
	t.Helper()
	var declaration Declaration
	if err := json.Unmarshal([]byte(text), &declaration); err != nil {
		t.Fatal(err)
	}
	d, err := Normalize(declaration)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func builds(t *testing.T, d Definition) (string, string, *Index) {
	t.Helper()
	dir := t.TempDir()
	source, path := filepath.Join(dir, "source #?.db"), filepath.Join(dir, "search #?.sqlite")
	db, err := openFile(source, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fixture); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Build(t.Context(), source, path, d); err != nil {
		t.Fatal(err)
	}
	index, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { index.Close() })
	return source, path, index
}

func asks(t *testing.T, index *Index, def Definition, query url.Values) Result {
	t.Helper()
	result, err := index.Search(t.Context(), def, query)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hitIDs(result Result, key string) []any {
	keys := make([]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		keys = append(keys, hit[key])
	}
	return keys
}

func TestSearchBindingAndIndexIdentity(t *testing.T) {
	d := definition(t, declaration)
	uppercase := definition(t, strings.ReplaceAll(declaration, `"inventory"`, `"INVENTORY"`))
	if uppercase.ID != d.ID || uppercase.IndexID != d.IndexID || uppercase.Table != d.Table {
		t.Fatalf("SQLite table spelling should keep the same identity: %+v", uppercase)
	}
	other := definition(t, strings.ReplaceAll(declaration, `"one"`, `"two"`))
	if other.IndexID != d.IndexID || other.ID == d.ID || d.Endpoint() != EndpointPrefix+d.ID {
		t.Fatalf("unexpected identities: %+v versus %+v", d, other)
	}
	view := definition(t, strings.ReplaceAll(declaration, `"pageSize":2`, `"pageSize":3`))
	if view.IndexID != d.IndexID || view.ID == d.ID {
		t.Fatalf("different pagination should reuse its data index: %+v", view)
	}
	expanded := definition(t, strings.ReplaceAll(declaration, `"fields":["slug"]`, `"fields":["slug","secret"]`))
	if expanded.IndexID == d.IndexID || expanded.ID == d.ID {
		t.Fatalf("additional columns need a different index: %+v", expanded)
	}
	if expected := []string{"available", "cost", "description", "heading", "kind", "published", "rank", "section", "slug"}; !slices.Equal(expected, d.Columns()) {
		t.Fatalf("columns got %v, want %v", d.Columns(), expected)
	}
	if len(d.Options.Sorts["cheap"].By) != 2 || d.Options.Sorts["cheap"].By[0].Column != "cost" {
		t.Fatalf("shorthand sort order: %+v", d.Options.Sorts["cheap"])
	}
	if !d.UsesQuery(url.Values{"cost_to": {"10"}}) || !d.UsesQuery(url.Values{"q": {"text"}}) ||
		d.UsesQuery(url.Values{"ignored": {"5"}, "cost": {"5"}}) {
		t.Fatal("binding recognized the wrong URL parameters")
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var saved Definition
	if err := json.Unmarshal(encoded, &saved); err != nil {
		t.Fatal(err)
	}
	again, err := Normalize(Declaration{Table: saved.Table, Options: saved.Options})
	if err != nil || again.ID != d.ID || again.IndexID != d.IndexID {
		t.Fatalf("definition round-trip output %+v, error %v", again, err)
	}
}

func TestScopedSearch(t *testing.T) {
	d := definition(t, declaration)
	_, _, index := builds(t, d)
	got := asks(t, index, d, nil)
	if got.Total != 3 || got.Page != 1 || got.Pages != 2 || !got.HasHits || !got.HasNext ||
		got.NextHref != "?page=2" || got.HasPrev || got.Endpoint != d.Endpoint() ||
		!slices.Equal(hitIDs(got, "sku"), []any{"c", "a"}) {
		t.Fatalf("initial search: %+v", got)
	}
	if len(got.Facets["kind"].Options) != 2 ||
		got.Facets["kind"].Options[0] != (FacetOption{Value: "cups", Label: "cups", Count: 2}) ||
		got.Facets["kind"].Options[1] != (FacetOption{Value: "serve", Label: "serve", Count: 1}) ||
		got.Facets["in_stock"].Count != 2 {
		t.Fatalf("facets escaped the scope or have wrong counts: %+v", got.Facets)
	}
	if got.Query.Filters["in_stock"] != nil || got.Query.Filters["cost"] != (RangeValue{}) {
		t.Fatalf("unselected filter values: %+v", got.Query)
	}
	for _, hit := range got.Hits {
		for _, private := range []string{"published", "section", "secret"} {
			if _, leaked := hit[private]; leaked {
				t.Fatalf("hit exposes %q: %+v", private, hit)
			}
		}
	}
	page := asks(t, index, d, url.Values{"page": {"9999"}})
	if page.Page != 2 || page.Query.Page != 2 || page.PrevHref != "?" || page.HasNext ||
		!slices.Equal(hitIDs(page, "sku"), []any{"b"}) {
		t.Fatalf("last page: %+v", page)
	}

	filtered := asks(t, index, d, url.Values{
		"q": {"  CUPS "}, "kind": {"cups", "other-scope"}, "in_stock": {"1"}, "cost_from": {"10.25"},
	})
	if filtered.Query.Q != "CUPS" || filtered.Total != 1 || !slices.Equal(hitIDs(filtered, "sku"), []any{"a"}) ||
		!slices.Equal(filtered.Query.Filters["kind"].([]string), []string{"cups"}) ||
		filtered.Facets["kind"].Options[1].Count != 0 {
		t.Fatalf("filtered search: %+v", filtered)
	}
	bounds := filtered.Facets["cost"]
	if bounds.Min == nil || bounds.Max == nil || *bounds.Min != 15.5 || *bounds.Max != 15.5 {
		t.Fatalf("range bounds should respect the other filters: %+v", bounds)
	}
	reversed := asks(t, index, d, url.Values{"cost_from": {"35"}, "cost_to": {"25"}, "in_stock": {"false"}, "sort": {"expensive"}})
	if reversed.Total != 1 || !slices.Equal(hitIDs(reversed, "sku"), []any{"b"}) ||
		reversed.Query.Filters["in_stock"] != false || reversed.Query.Sort != "expensive" {
		t.Fatalf("reversed bounds: %+v", reversed)
	}
	empty := asks(t, index, d, url.Values{"q": {`"script literal from another scope"`}, "page": {"5"}})
	if empty.Total != 0 || empty.Hits == nil || empty.HasHits || empty.Page != 1 || empty.Pages != 1 {
		t.Fatalf("empty search: %+v", empty)
	}

	other := definition(t, strings.ReplaceAll(declaration, `"one"`, `"two"`))
	got = asks(t, index, other, url.Values{"q": {"cups"}})
	if got.Total != 1 || !slices.Equal(hitIDs(got, "sku"), []any{"d"}) ||
		len(got.Facets["kind"].Options) != 1 || got.Facets["kind"].Options[0].Value != "other-scope" ||
		got.Endpoint == d.Endpoint() {
		t.Fatalf("other scope using the same index: %+v", got)
	}
	// Reading another binding must not overwrite the first scope's facet cache.
	if got := asks(t, index, d, nil); got.Total != 3 || len(got.Facets["kind"].Options) != 2 {
		t.Fatalf("scope options changed after reading another binding: %+v", got)
	}
}

func TestSeparateSourceAndFailedPublication(t *testing.T) {
	d := definition(t, declaration)
	source, path, first := builds(t, d)
	second := definition(t, `{"table":"notes","options":{"text":["text"]}}`)
	if _, err := first.Search(t.Context(), second, nil); err == nil {
		t.Fatal("the wrong source definition opened the first index")
	}
	otherPath := filepath.Join(filepath.Dir(path), "notes.sqlite")
	if err := Build(t.Context(), source, otherPath, second); err != nil {
		t.Fatal(err)
	}
	notes, err := Open(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	defer notes.Close()
	got := asks(t, notes, second, url.Values{"q": {"glazes"}})
	if got.Total != 1 || !slices.Equal(hitIDs(got, "id"), []any{int64(2)}) {
		t.Fatalf("second source: %+v", got)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	missing := definition(t, `{"table":"inventory","options":{"fields":["missing"]}}`)
	if err := Build(t.Context(), source, path, missing); err == nil {
		t.Fatal("build with a missing column succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil || !slices.Equal(before, after) {
		t.Fatal("failed build replaced a published generation")
	}
	if err := Build(t.Context(), source, source, d); err == nil {
		t.Fatal("build tried to replace the data database")
	}
	if _, err := Open(source); err == nil {
		t.Fatal("opened a regular database as a search index")
	}
}

func TestScopePreservesSQLiteDateAndNullValues(t *testing.T) {
	d := definition(t, `{"table":"events","options":{"where":{"happened_at":"2026-01-01 03:20:00"},"fields":["happened_at"]}}`)
	_, _, index := builds(t, d)
	got := asks(t, index, d, nil)
	if got.Total != 1 || got.Hits[0]["happened_at"] != "2026-01-01 03:20:00" {
		t.Fatalf("fixed equality changed the design's stored date: %+v", got)
	}
	null := definition(t, `{"table":"events","options":{"where":{"happened_at":null},"fields":["happened_at"]}}`)
	got = asks(t, index, null, nil)
	if null.IndexID != d.IndexID || got.Total != 1 || got.Hits[0]["id"] != int64(2) || got.Hits[0]["happened_at"] != nil {
		t.Fatalf("fixed nullable scope: %+v", got)
	}
}

func TestSharedIndexAcrossConcurrentScopes(t *testing.T) {
	one := definition(t, declaration)
	two := definition(t, strings.ReplaceAll(declaration, `"one"`, `"two"`))
	_, _, index := builds(t, one)
	for _, tc := range []struct {
		name string
		d    Definition
		ids  []any
	}{
		{"one", one, []any{"a", "b"}},
		{"two", two, []any{"d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for range 10 {
				got := asks(t, index, tc.d, url.Values{"q": {"cups"}, "section": {"untrusted"}})
				if !slices.Equal(hitIDs(got, "sku"), tc.ids) {
					t.Fatalf("query escaped its fixed scope: %+v", got)
				}
			}
		})
	}
}

func TestInvalidDeclaration(t *testing.T) {
	cases := []string{
		`{"table":"sqlite_schema","options":{}}`,
		`{"table":"a; DROP TABLE products","options":{}}`,
		`{"table":"a","options":{"where":{"bad":[]}}}`,
		`{"table":"a","options":{"filters":{"q":"multi"}}}`,
		`{"table":"a","options":{"filters":{"price":"range","price_from":"boolean"}}}`,
		`{"table":"a","options":{"filters":{"price":"wrong"}}}`,
		`{"table":"a","options":{"sorts":{"new":{"date":"desc; DROP TABLE a"}}}}`,
		`{"table":"a","options":{"defaultSort":"absent"}}`,
		`{"table":"a","options":{"pageSize":-1}}`,
	}
	for _, input := range cases {
		var d Declaration
		if err := json.Unmarshal([]byte(input), &d); err != nil {
			continue
		}
		if _, err := Normalize(d); err == nil {
			t.Errorf("accepted invalid declaration %s", input)
		}
	}
}

func TestBuildCancellationLeavesGeneration(t *testing.T) {
	d := definition(t, declaration)
	source, path, _ := builds(t, d)
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Build(cancelled, source, path, d); err == nil {
		t.Fatal("cancelled catalog build succeeded")
	}
	db, err := openFile(source, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("UPDATE inventory SET cost=2 WHERE sku='a'"); err != nil {
		t.Fatal(err)
	}
	index, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	if got := asks(t, index, d, nil); !slices.Equal(hitIDs(got, "sku"), []any{"c", "a"}) {
		t.Fatalf("cancelled build changed the published generation: %+v", got)
	}
}
