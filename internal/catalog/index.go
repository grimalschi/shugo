// Package catalog is the search backend for the experimental catalog island.
// It reads a projection of the example shop into an immutable snapshot.
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "modernc.org/sqlite"
)

const indexVersion = 1

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Product holds only the data needed to display a result card.
type Product struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	Price         int64    `json:"price"`
	OriginalPrice int64    `json:"originalPrice"`
	Available     bool     `json:"available"`
	Sizes         []string `json:"sizes"`
}

type entry struct {
	Product
	Position int64  `json:"position"`
	Arrival  string `json:"arrival"`
	Text     string `json:"text"`
}

type snapshot struct {
	Version    int      `json:"version"`
	Categories []option `json:"categories"`
	Products   []entry  `json:"products"`
}

// Index answers requests against a single published snapshot.
type Index struct {
	data       snapshot
	categories map[string]int
	sizes      []option
	sizeIDs    map[string]int
	orders     map[string][]int
}

// Build reads the shop's categories, products and variants in one read
// transaction and atomically replaces the search snapshot on success.
func Build(ctx context.Context, sourceDB, indexFile string) (*Index, error) {
	if sourceDB == "" {
		return nil, fmt.Errorf("the catalog island needs a data database")
	}
	data, err := readShop(ctx, sourceDB)
	if err != nil {
		return nil, fmt.Errorf("read catalog data: %w", err)
	}
	index, err := prepare(data)
	if err != nil {
		return nil, err
	}
	if err := publish(indexFile, data); err != nil {
		return nil, fmt.Errorf("write catalog index: %w", err)
	}
	return index, nil
}

func readShop(ctx context.Context, name string) (snapshot, error) {
	var data snapshot
	settings := url.Values{"mode": {"ro"}, "_pragma": {"busy_timeout(10000)"}}
	address := url.URL{Scheme: "file", Path: name, RawQuery: settings.Encode()}
	db, err := sql.Open("sqlite", address.String())
	if err != nil {
		return data, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return data, err
	}
	defer tx.Rollback()

	data.Version = indexVersion
	if data.Categories, err = readCategories(ctx, tx); err != nil {
		return data, err
	}
	if data.Products, err = readProducts(ctx, tx); err != nil {
		return data, err
	}
	if err := readSizes(ctx, tx, data.Products); err != nil {
		return data, err
	}
	return data, tx.Commit()
}

func readCategories(ctx context.Context, tx *sql.Tx) ([]option, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, title FROM categories ORDER BY position, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []option
	for rows.Next() {
		var category option
		if err := rows.Scan(&category.Value, &category.Label); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func readProducts(ctx context.Context, tx *sql.Tx) ([]entry, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, slug, title, category, price, coalesce(discount, 0), available, arrival, description, position
FROM products
ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []entry
	for rows.Next() {
		var p entry
		var discount int64
		var description string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Category, &p.OriginalPrice, &discount, &p.Available, &p.Arrival, &description, &p.Position); err != nil {
			return nil, err
		}
		p.Price = p.OriginalPrice
		if discount != 0 {
			p.Price = int64(math.Floor(float64(p.OriginalPrice)*(100-float64(discount))/100 + 0.5))
		}
		p.Text = fold(p.Title + " " + description)
		p.Sizes = []string{}
		products = append(products, p)
	}
	return products, rows.Err()
}

func readSizes(ctx context.Context, tx *sql.Tx, products []entry) error {
	byID := make(map[string]int, len(products))
	for id := range products {
		byID[products[id].ID] = id
	}
	rows, err := tx.QueryContext(ctx, "SELECT product, title FROM variants ORDER BY product, position, sku")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var product, title string
		if err := rows.Scan(&product, &title); err != nil {
			return err
		}
		if id, exists := byID[product]; exists && !slices.Contains(products[id].Sizes, title) {
			products[id].Sizes = append(products[id].Sizes, title)
		}
	}
	return rows.Err()
}

func publish(name string, data snapshot) error {
	f, err := os.CreateTemp(filepath.Dir(name), ".catalog-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), name)
}

// Open reads and prepares a build-produced catalog snapshot.
func Open(name string) (*Index, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return decode(file)
}

func decode(source io.Reader) (*Index, error) {
	var data snapshot
	decoder := json.NewDecoder(source)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("read catalog index: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after the catalog index")
	}
	return prepare(data)
}

func prepare(data snapshot) (*Index, error) {
	if data.Version != indexVersion {
		return nil, fmt.Errorf("unsupported catalog index version %d", data.Version)
	}
	i := &Index{data: data, categories: map[string]int{}, sizeIDs: map[string]int{}, orders: map[string][]int{}}
	for id, category := range data.Categories {
		i.categories[category.Value] = id
	}
	sizes := map[string]bool{}
	for _, p := range data.Products {
		if _, ok := i.categories[p.Category]; !ok {
			return nil, fmt.Errorf("catalog product %q has unknown category %q", p.ID, p.Category)
		}
		for _, size := range p.Sizes {
			sizes[size] = true
		}
	}
	var orderedSizes []string
	for _, size := range []string{"small", "medium", "large"} {
		if sizes[size] {
			orderedSizes = append(orderedSizes, size)
			delete(sizes, size)
		}
	}
	var extraSizes []string
	for size := range sizes {
		extraSizes = append(extraSizes, size)
	}
	slices.Sort(extraSizes)
	for id, size := range append(orderedSizes, extraSizes...) {
		i.sizes = append(i.sizes, option{Value: size, Label: size})
		i.sizeIDs[size] = id
	}
	i.orderProducts()
	return i, nil
}

func fold(value string) string { return strings.ToLower(value) }
