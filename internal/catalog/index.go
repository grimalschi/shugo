// Package catalog is the search backend for the experimental catalog island.
// It reads a projection of the example shop into an immutable SQLite snapshot.
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
	"unicode"

	"github.com/blevesearch/snowballstem"
	"github.com/blevesearch/snowballstem/english"
	_ "modernc.org/sqlite"

	"github.com/grimalschi/shugo/internal/data"
)

const indexVersion = 1

type PriceChange struct {
	ID                   string
	OriginalPrice, Price int64
}

// UpdatePlan tells whether a published projection can be updated using prices
// from the journal or needs a new projection of the source.
type UpdatePlan struct {
	Rebuild bool
	Prices  []PriceChange
}

func PlanUpdate(source *data.DB, changes []data.Change) UpdatePlan {
	if len(changes) == 0 {
		return UpdatePlan{}
	}
	full := UpdatePlan{Rebuild: true}
	columns := make(map[*data.Table][]int, len(projectionColumns))
	products := source.TableFold("products")
	var idColumn, priceColumn, discountColumn int
	for name, fields := range projectionColumns {
		table := source.TableFold(name)
		if table == nil {
			return full // a view may read other tables
		}
		for index, field := range table.Columns {
			lower := strings.ToLower(field)
			if len(lower) == len(field) && slices.Contains(fields, lower) {
				columns[table] = append(columns[table], index)
				if table == products {
					switch lower {
					case "id":
						idColumn = index
					case "price":
						priceColumn = index
					case "discount":
						discountColumn = index
					}
				}
			}
		}
		if len(columns[table]) != len(fields) {
			return full // a generated column may depend on other columns
		}
	}
	var plan UpdatePlan
	for _, change := range changes {
		changed := false
		for _, column := range columns[change.Table] {
			if !change.Differs(column) {
				continue
			}
			if change.Table != products || change.Old == nil || change.New == nil ||
				(column != priceColumn && column != discountColumn) {
				return full
			}
			changed = true
		}
		if !changed {
			continue
		}
		id, idOK := change.New[idColumn].(string)
		price, priceOK := change.New[priceColumn].(int64)
		discount, discountOK := change.New[discountColumn].(int64)
		if !idOK || !priceOK || (!discountOK && change.New[discountColumn] != nil) {
			return full
		}
		plan.Prices = append(plan.Prices, PriceChange{ID: id, OriginalPrice: price, Price: effectivePrice(price, discount)})
	}
	return plan
}

var projectionColumns = map[string][]string{
	"categories": {"id", "title", "position"},
	"products":   {"id", "slug", "title", "category", "price", "discount", "available", "arrival", "description", "position"},
	"variants":   {"product", "title", "position", "sku"},
}

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
	db         *sql.DB
	categories []option
	sizes      []option
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
	index, err := publish(ctx, indexFile, data)
	if err != nil {
		return nil, fmt.Errorf("write catalog index: %w", err)
	}
	return index, nil
}

func readShop(ctx context.Context, name string) (snapshot, error) {
	var data snapshot
	settings := url.Values{"mode": {"ro"}, "_pragma": {"busy_timeout(10000)"}}
	db, err := openFile(name, settings)
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
		p.Price = effectivePrice(p.OriginalPrice, discount)
		p.Text = p.Title + " " + description
		p.Sizes = []string{}
		products = append(products, p)
	}
	return products, rows.Err()
}

func effectivePrice(original, discount int64) int64 {
	if discount == 0 {
		return original
	}
	return int64(math.Floor(float64(original)*(100-float64(discount))/100 + 0.5))
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

// Open reads the metadata of a published SQLite snapshot. Its path must remain
// immutable for the lifetime of the index.
func Open(name string) (*Index, error) {
	ctx := context.Background()
	if filepath.Ext(name) == ".json" {
		file, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		// Earlier builds used .json. Support those generations and SQLite
		// files created by Build callers that retained the old extension.
		var header [16]byte
		if _, err := file.ReadAt(header[:], 0); err != nil && err != io.EOF {
			return nil, err
		}
		if string(header[:]) != "SQLite format 3\x00" {
			return decode(file)
		}
	}
	return openIndex(ctx, name, 4)
}

// Close releases the snapshot's SQLite connections.
func (i *Index) Close() error {
	return i.db.Close()
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
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := writeIndex(ctx, db, data); err != nil {
		db.Close()
		return nil, err
	}
	return loadIndex(ctx, db)
}

func sizeNames(products []entry) []string {
	sizes := map[string]struct{}{}
	for _, p := range products {
		for _, size := range p.Sizes {
			sizes[size] = struct{}{}
		}
	}
	var orderedSizes []string
	for _, size := range []string{"small", "medium", "large"} {
		if _, exists := sizes[size]; exists {
			orderedSizes = append(orderedSizes, size)
			delete(sizes, size)
		}
	}
	var extraSizes []string
	for size := range sizes {
		extraSizes = append(extraSizes, size)
	}
	slices.Sort(extraSizes)
	return append(orderedSizes, extraSizes...)
}

func fold(value string) string { return strings.ToLower(value) }

func searchWords(value string) []string {
	return stemWords(value, nil)
}

func stemWords(value string, cache map[string]string) []string {
	words := strings.FieldsFunc(fold(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for at, word := range words {
		stem, found := cache[word]
		if !found {
			env := snowballstem.NewEnv(word)
			english.Stem(env)
			stem = env.Current()
			// Repeated words benefit from a build-local cache. Bound the
			// memory used by product IDs and other one-off tokens.
			if cache != nil && len(cache) < 32768 {
				cache[word] = stem
			}
		}
		words[at] = stem
	}
	return words
}
