package catalog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
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
	('v4', 'p3', 'small', 1), ('v5', 'p3', 'medium', 2), ('v6', 'p4', 'medium', 1);
`

func testShop(t *testing.T) (*Index, string, string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	source, path := filepath.Join(dir, "shop #?.sqlite"), filepath.Join(dir, "catalog.json")
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
	return i, source, path, db
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
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		query url.Values
		want  []string
	}{
		{"default", nil, []string{"p1", "p2", "p3", "p4"}},
		{"Multiple search terms", url.Values{"q": {"  FLAX, handmade  "}}, []string{"p1"}},
		{"stock and facets", url.Values{"category": {"c1"}, "size": {"small"}, "stock": {"1"}}, []string{"p1"}},
		{"repeatable filters", url.Values{"category": {"c1", "c2"}, "size": {"small", "medium"}}, []string{"p1", "p2", "p3", "p4"}},
		{"expensive", url.Values{"sort": {"expensive"}}, []string{"p3", "p4", "p2", "p1"}},
		{"new", url.Values{"sort": {"new"}}, []string{"p3", "p4", "p2", "p1"}},
		{"discount and reversed range", url.Values{"price_from": {"75"}, "price_to": {"61"}}, []string{"p1", "p2"}},
		{"bad values", url.Values{"category": {"unknown"}, "size": {"unknown"}, "sort": {"garbage"}, "stock": {"no"}, "price_to": {"-50"}, "page": {"oops"}}, []string{"p1", "p2", "p3", "p4"}},
		{"empty", url.Values{"category": {"c0"}}, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := i.Search(tc.query)
			if !slices.Equal(ids(got.Hits), tc.want) || got.Total != len(tc.want) {
				t.Fatalf("results %v with total %d; want %v", ids(got.Hits), got.Total, tc.want)
			}
			if !reflect.DeepEqual(got, reopened.Search(tc.query)) {
				t.Fatal("persisted snapshot disagrees with the snapshot that built it")
			}
		})
	}
	got := i.Search(url.Values{"category": {"unknown", "c1", "c1"}, "size": {"small"}, "stock": {"1"}})
	if !slices.Equal(got.Filters.Category, []string{"c1"}) || got.Hits[0].Price != 61 || got.Hits[0].OriginalPrice != 121 || got.Hits[0].CategoryTitle != "Teapots" ||
		!slices.Equal(got.Hits[0].Sizes, []string{"small"}) {
		t.Fatalf("filter normalization and card projection: %+v", got)
	}
	want := Facets{
		Category: []Facet{{option{"c1", "Teapots"}, 1}, {option{"c0", "Empty"}, 0}, {option{"c2", "Tableware"}, 1}},
		Size:     []Facet{{option{"small", "small"}, 1}, {option{"medium", "medium"}, 0}},
		Stock:    1,
	}
	if !reflect.DeepEqual(got.Facets, want) {
		t.Fatalf("facets %+v; want %+v", got.Facets, want)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, fragment := range []string{`"category":["c1"]`, `"priceFrom":null`, `"categoryTitle":"Teapots"`, `{"value":"c1","label":"Teapots","count":1}`} {
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
	if out := i.Search(nil).Hits[0].Sizes[0]; out != "small" {
		t.Fatalf("search result altered the index: %s", out)
	}
}

func TestPagination(t *testing.T) {
	i, err := prepare(benchmarkShop(55))
	if err != nil {
		t.Fatal(err)
	}
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
			got := i.Search(q)
			if got.Total != 55 || got.Pages != 3 || got.Page != tc.page || got.Filters.Page != got.Page ||
				len(got.Hits) != tc.count || got.Hits[0].ID != tc.first || got.Hits[tc.count-1].ID != tc.last {
				t.Fatalf("page %+v, cards %v", got.Filters, ids(got.Hits))
			}
		})
	}
	q := url.Values{"page": {"9000"}, "category": {"c1"}}
	got := i.Search(q)
	if !slices.Equal(ids(got.Hits), []string{"p1", "p51"}) || got.Total != 2 || got.Page != 1 || got.Pages != 1 {
		t.Fatalf("clamped page %+v, cards %v", got.Filters, ids(got.Hits))
	}
	got = i.Search(url.Values{"q": {"absent"}, "page": {"6"}})
	if got.Page != 1 || got.Pages != 1 || got.Total != 0 || got.Hits == nil || len(got.Hits) != 0 {
		t.Fatalf("empty page %+v", got)
	}
}

func TestBuildPublishesSnapshot(t *testing.T) {
	original, source, path, db := testShop(t)
	before := original.Search(nil)
	if _, err := db.Exec("UPDATE products SET price = 171, title = 'New teapot' WHERE id = 'p1'; UPDATE categories SET title = 'New teapots' WHERE id = 'c1'"); err != nil {
		t.Fatal(err)
	}
	unchanged, err := Open(path)
	if err != nil || !reflect.DeepEqual(unchanged.Search(nil), before) {
		t.Fatalf("source updates leaked out before publication: %+v, %v", unchanged, err)
	}
	published, err := Build(t.Context(), source, path)
	if err != nil {
		t.Fatal(err)
	}
	now, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := now.Search(nil)
	item := got.Hits[0]
	if item.Price != 86 || item.Title != "New teapot" || item.CategoryTitle != "New teapots" ||
		!reflect.DeepEqual(got, published.Search(nil)) || !reflect.DeepEqual(before, original.Search(nil)) {
		t.Fatalf("loading the new version: %+v", got)
	}
	if _, err := db.Exec("DROP TABLE variants"); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(t.Context(), source, path); err == nil {
		t.Fatal("build without variants table succeeded")
	}
	still, err := Open(path)
	if err != nil || !reflect.DeepEqual(got, still.Search(nil)) {
		t.Fatalf("failed build replaced a good snapshot: %+v, %v", still, err)
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
			b.Run("default", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					index.Search(nil)
				}
			})
			b.Run("filtered", func(b *testing.B) {
				v := url.Values{"q": {"North"}, "category": {"c2", "c4"}, "size": {"small"}, "stock": {"1"}, "price_to": {"3000"}, "sort": {"cheap"}}
				b.ReportAllocs()
				for b.Loop() {
					index.Search(v)
				}
			})
		})
	}
}
