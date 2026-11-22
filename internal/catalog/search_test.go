package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const shop = `
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT, position INTEGER);
CREATE TABLE products (
	id TEXT PRIMARY KEY, slug TEXT, title TEXT, category TEXT, price INTEGER, discount INTEGER,
	available INTEGER, arrival TEXT, description TEXT, note TEXT, position INTEGER
);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT, title TEXT, position INTEGER);
INSERT INTO categories VALUES ('c2', 'Tableware', 3), ('c1', 'Teapots', 1), ('c0', 'Empty', 2);
INSERT INTO products VALUES
	('p1', 'tea-flax', 'Flax Teapot', 'c1', 121, 50, 1, '2027-01-10', 'handmade', 'private', 1),
	('p2', 'tea-north', 'North Teapot', 'c1', 75, NULL, 0, '2027-02-10', 'handmade', 'private', 2),
	('p3', 'vase-flax', 'Flax Vase', 'c2', 125, NULL, 1, '2027-03-10', 'matte glaze', 'private', 3),
	('p4', 'cup-flax', 'Flax Cup', 'c2', 125, NULL, 1, '2027-03-10', 'matte glaze', 'private', 3);
INSERT INTO variants VALUES
	('v1', 'p1', 'small', 1), ('v2', 'p1', 'small', 2), ('v3', 'p2', 'small', 1),
	('v4', 'p3', 'small', 1), ('v5', 'p3', 'medium', 2), ('v6', 'p4', 'medium', 1),
	('v7', 'p3', 'banquet', 3), ('v8', 'p4', 'dinner', 0), ('v9', 'p4', 'XL', 2);
`

func testShop(t *testing.T) (*Index, string, string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	source, path := filepath.Join(dir, "shop #?.sqlite"), filepath.Join(dir, "catalog #?.db")
	address := url.URL{Scheme: "file", Path: source}
	db, err := sql.Open("sqlite", address.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(shop); err != nil {
		t.Fatal(err)
	}
	i, err := Build(t.Context(), source, path)
	if err != nil {
		t.Fatal(err)
	}
	closeIndexAfterTest(t, i)
	return i, source, path, db
}

func closeIndexAfterTest(t testing.TB, index *Index) {
	t.Helper()
	t.Cleanup(func() {
		if err := index.Close(); err != nil {
			t.Error(err)
		}
	})
}

func openTestIndex(t *testing.T, path string) *Index {
	t.Helper()
	index, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	closeIndexAfterTest(t, index)
	return index
}

func searchTestIndex(t testing.TB, index *Index, values url.Values) Result {
	t.Helper()
	result, err := index.Search(t.Context(), values)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func ids(hits []Hit) []string {
	result := make([]string, 0, len(hits))
	for _, item := range hits {
		result = append(result, item.ID)
	}
	return result
}

func TestSearch(t *testing.T) {
	i, _, path, _ := testShop(t)
	reopened := openTestIndex(t, path)
	for _, tc := range []struct {
		name  string
		query url.Values
		want  []string
	}{
		{"default", nil, []string{"p1", "p2", "p3", "p4"}},
		{"Multiple search terms", url.Values{"q": {"  FLAX, handmade  "}}, []string{"p1"}},
		{"English singular", url.Values{"q": {"cup"}}, []string{"p4"}},
		{"English plural", url.Values{"q": {"cups"}}, []string{"p4"}},
		{"Plural search with case folding", url.Values{"q": {"CUPS"}}, []string{"p4"}},
		{"FTS operators are plain text", url.Values{"q": {"cup OR teapot"}}, []string{}},
		{"punctuation without terms", url.Values{"q": {"!?()"}}, []string{"p1", "p2", "p3", "p4"}},
		{"stock and facets", url.Values{"category": {"c1"}, "size": {"small"}, "stock": {"1"}}, []string{"p1"}},
		{"repeatable filters", url.Values{"category": {"c1", "c2"}, "size": {"small", "medium"}}, []string{"p1", "p2", "p3", "p4"}},
		{"repeatable arbitrary sizes", url.Values{"category": {"c2"}, "size": {"XL", "banquet"}}, []string{"p3", "p4"}},
		{"expensive", url.Values{"sort": {"expensive"}}, []string{"p3", "p4", "p2", "p1"}},
		{"new", url.Values{"sort": {"new"}}, []string{"p3", "p4", "p2", "p1"}},
		{"discount and reversed range", url.Values{"price_from": {"75"}, "price_to": {"61"}}, []string{"p1", "p2"}},
		{"bad values", url.Values{"category": {"unknown"}, "size": {"unknown"}, "sort": {"garbage"}, "stock": {"no"}, "price_to": {"-50"}, "page": {"oops"}}, []string{"p1", "p2", "p3", "p4"}},
		{"empty", url.Values{"category": {"c0"}}, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := searchTestIndex(t, reopened, tc.query)
			if !slices.Equal(ids(got.Hits), tc.want) || got.Total != len(tc.want) {
				t.Fatalf("results %v with total %d; want %v", ids(got.Hits), got.Total, tc.want)
			}
			if !reflect.DeepEqual(got, searchTestIndex(t, i, tc.query)) {
				t.Fatal("persisted snapshot disagrees with the snapshot that built it")
			}
		})
	}
	got := searchTestIndex(t, reopened, url.Values{"category": {"unknown", "c1", "c1"}, "size": {"small"}, "stock": {"1"}})
	if !slices.Equal(got.Filters.Category, []string{"c1"}) || got.Hits[0].Price != 61 || got.Hits[0].OriginalPrice != 121 || got.Hits[0].CategoryTitle != "Teapots" ||
		!slices.Equal(got.Hits[0].Sizes, []string{"small"}) {
		t.Fatalf("filter normalization and card projection: %+v", got)
	}
	want := Facets{
		Category: []Facet{
			{option: option{"c1", "Teapots"}, Count: 1, CountLabel: "1 product", Selected: true},
			{option: option{"c0", "Empty"}, CountLabel: "0 products"},
			{option: option{"c2", "Tableware"}, Count: 1, CountLabel: "1 product"},
		},
		Size: []Facet{
			{option: option{"small", "small"}, Count: 1, CountLabel: "1 product", Selected: true},
			{option: option{"medium", "medium"}, CountLabel: "0 products"},
			{option: option{"XL", "XL"}, CountLabel: "0 products"},
			{option: option{"banquet", "banquet"}, CountLabel: "0 products"},
			{option: option{"dinner", "dinner"}, CountLabel: "0 products"},
		},
		Stock: 1, StockLabel: "1 product",
	}
	if !reflect.DeepEqual(got.Facets, want) {
		t.Fatalf("facets %+v; want %+v", got.Facets, want)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, fragment := range []string{`"category":["c1"]`, `"priceFrom":null`, `"categoryTitle":"Teapots"`, `{"value":"c1","label":"Teapots","count":1,"countLabel":"1 product","selected":true}`} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("response %s is missing %s", body, fragment)
		}
	}
	for _, fragment := range []string{"private", "description", "handmade"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("response %s leaked %s", body, fragment)
		}
	}
	got.Hits[0].Sizes[0] = "changed"
	if out := searchTestIndex(t, reopened, nil).Hits[0].Sizes[0]; out != "small" {
		t.Fatalf("search result altered the index: %s", out)
	}
}

func TestDisjunctiveSizeFacets(t *testing.T) {
	index, _, _, _ := testShop(t)
	got := searchTestIndex(t, index, url.Values{"category": {"c2"}, "size": {"XL"}, "stock": {"1"}})
	if got.Total != 1 || len(got.Hits) != 1 || got.Hits[0].ID != "p4" {
		t.Fatalf("XL filter returned %v with total %d", ids(got.Hits), got.Total)
	}
	if want := []string{"dinner", "medium", "XL"}; !slices.Equal(got.Hits[0].Sizes, want) {
		t.Fatalf("card sizes %v; want %v in variant order", got.Hits[0].Sizes, want)
	}
	want := []Facet{
		{option: option{"small", "small"}, Count: 1, CountLabel: "1 product"},
		{option: option{"medium", "medium"}, Count: 2, CountLabel: "2 products"},
		{option: option{"XL", "XL"}, Count: 1, CountLabel: "1 product", Selected: true},
		{option: option{"banquet", "banquet"}, Count: 1, CountLabel: "1 product"},
		{option: option{"dinner", "dinner"}, Count: 1, CountLabel: "1 product"},
	}
	if !reflect.DeepEqual(got.Facets.Size, want) {
		t.Fatalf("size facets %+v; want %+v", got.Facets.Size, want)
	}
}

func TestPagination(t *testing.T) {
	i, err := prepare(benchmarkShop(55))
	if err != nil {
		t.Fatal(err)
	}
	closeIndexAfterTest(t, i)
	for _, tc := range []struct {
		path, first, last string
		count, page       int
	}{
		{"", "p1", "p24", 24, 1},
		{"page=2", "p25", "p48", 24, 2},
		{"page=3", "p49", "p55", 7, 3},
		{"page=9000", "p49", "p55", 7, 3},
	} {
		t.Run(tc.path, func(t *testing.T) {
			q, _ := url.ParseQuery(tc.path)
			got := searchTestIndex(t, i, q)
			if got.Total != 55 || got.Pages != 3 || got.Page != tc.page || got.Filters.Page != got.Page ||
				len(got.Hits) != tc.count || got.Hits[0].ID != tc.first || got.Hits[tc.count-1].ID != tc.last {
				t.Fatalf("page %+v, cards %v", got.Filters, ids(got.Hits))
			}
		})
	}
	q := url.Values{"page": {"9000"}, "category": {"c1"}}
	got := searchTestIndex(t, i, q)
	if !slices.Equal(ids(got.Hits), []string{"p1", "p51"}) || got.Total != 2 || got.Page != 1 || got.Pages != 1 {
		t.Fatalf("clamped page %+v, cards %v", got.Filters, ids(got.Hits))
	}
	got = searchTestIndex(t, i, url.Values{"q": {"absent"}, "page": {"6"}})
	if got.Page != 1 || got.Pages != 1 || got.Total != 0 || got.Hits == nil || len(got.Hits) != 0 {
		t.Fatalf("empty page %+v", got)
	}
}

func TestBuildPublishesSnapshot(t *testing.T) {
	original, source, path, db := testShop(t)
	before := searchTestIndex(t, original, nil)
	if _, err := db.Exec("UPDATE products SET price = 171, title = 'New teapot' WHERE id = 'p1'; UPDATE categories SET title = 'New teapots' WHERE id = 'c1'"); err != nil {
		t.Fatal(err)
	}
	unchanged := openTestIndex(t, path)
	if got := searchTestIndex(t, unchanged, nil); !reflect.DeepEqual(got, before) {
		t.Fatalf("source updates leaked out before publication: %+v", got)
	}
	published, err := Build(t.Context(), source, path)
	if err != nil {
		t.Fatal(err)
	}
	closeIndexAfterTest(t, published)
	now := openTestIndex(t, path)
	got := searchTestIndex(t, now, nil)
	item := got.Hits[0]
	if item.Price != 86 || item.Title != "New teapot" || item.CategoryTitle != "New teapots" ||
		!reflect.DeepEqual(got, searchTestIndex(t, published, nil)) {
		t.Fatalf("loading the new version: %+v", got)
	}
	for _, old := range []*Index{original, unchanged} {
		if got := searchTestIndex(t, old, nil); !reflect.DeepEqual(got, before) {
			t.Fatalf("open reader changed after publication: %+v", got)
		}
	}
	if _, err := db.Exec("DROP TABLE variants"); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(t.Context(), source, path); err == nil {
		t.Fatal("build without variants table succeeded")
	}
	if afterFailure := searchTestIndex(t, openTestIndex(t, path), nil); !reflect.DeepEqual(got, afterFailure) {
		t.Fatalf("failed build replaced a good snapshot: %+v", afterFailure)
	}
}

func TestOpenLegacySnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	legacy := benchmarkShop(2)
	legacy.Version = 1
	body, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	index := openTestIndex(t, path)
	got := searchTestIndex(t, index, url.Values{"q": {"North"}, "category": {"c2"}})
	if got.Total != 1 || !slices.Equal(ids(got.Hits), []string{"p2"}) {
		t.Fatalf("legacy projection returned %v with total %d", ids(got.Hits), got.Total)
	}
}

func TestOpenRejectsInvalidProjection(t *testing.T) {
	t.Run("corrupt file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.db")
		if err := os.WriteFile(path, []byte("invalid catalog index"), 0600); err != nil {
			t.Fatal(err)
		}
		if index, err := Open(path); err == nil {
			index.Close()
			t.Fatal("opened a corrupt index")
		}
	})
	t.Run("unrelated SQLite database", func(t *testing.T) {
		_, source, _, _ := testShop(t)
		if index, err := Open(source); err == nil {
			index.Close()
			t.Fatal("opened the source database as a search projection")
		}
	})
}

func TestSearchCanceled(t *testing.T) {
	index, _, _, _ := testShop(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := index.Search(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled search returned %v", err)
	}
}

func benchmarkShop(count int) snapshot {
	data := snapshot{Version: indexVersion}
	for n := range 50 {
		id := fmt.Sprintf("c%d", n+1)
		data.Categories = append(data.Categories, option{Value: id, Label: "Category " + id})
	}
	for n := range count {
		id := fmt.Sprintf("p%d", n+1)
		p := Product{
			ID: id, Slug: id, Title: "North Teapot " + id, Category: data.Categories[n%50].Value,
			Price: int64(700 + n%6000), OriginalPrice: int64(700 + n%6000), Available: n%7 != 0,
			Sizes: []string{"small", "medium"},
		}
		data.Products = append(data.Products, entry{Product: p, Position: int64(n), Arrival: "2027-01-10",
			Text: "north teapot " + id + " handmade, durable matte glaze"})
	}
	return data
}

func BenchmarkSearch(b *testing.B) {
	for _, count := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			index, err := prepare(benchmarkShop(count))
			if err != nil {
				b.Fatal(err)
			}
			closeIndexAfterTest(b, index)
			for _, tc := range []struct {
				name  string
				query url.Values
			}{
				{"default", nil},
				{"filtered", url.Values{"q": {"North"}, "category": {"c2", "c4"}, "size": {"small"}, "stock": {"1"}, "price_to": {"3000"}, "sort": {"cheap"}}},
			} {
				b.Run(tc.name, func(b *testing.B) {
					ctx := b.Context()
					b.ReportAllocs()
					for b.Loop() {
						if _, err := index.Search(ctx, tc.query); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
