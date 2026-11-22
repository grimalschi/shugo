package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// PageSize is the number of cards returned by one search.
const PageSize = 24

type Filters struct {
	Q         string   `json:"q"`
	Category  []string `json:"category"`
	Size      []string `json:"size"`
	Stock     bool     `json:"stock"`
	PriceFrom *int64   `json:"priceFrom"`
	PriceTo   *int64   `json:"priceTo"`
	Sort      string   `json:"sort"`
	Page      int      `json:"page"`
}

type Hit struct {
	Product
	CategoryTitle      string `json:"categoryTitle"`
	PriceLabel         string `json:"priceLabel"`
	OriginalPriceLabel string `json:"originalPriceLabel"`
	Discounted         bool   `json:"discounted"`
	StockClass         string `json:"stockClass"`
	StockLabel         string `json:"stockLabel"`
	HasSizes           bool   `json:"hasSizes"`
	SizesLabel         string `json:"sizesLabel"`
}

type Facet struct {
	option
	Count      int    `json:"count"`
	CountLabel string `json:"countLabel"`
	Selected   bool   `json:"selected"`
}

type Facets struct {
	Category   []Facet `json:"category"`
	Size       []Facet `json:"size"`
	Stock      int     `json:"stock"`
	StockLabel string  `json:"stockLabel"`
}

// Result is shared by the default render, request SSR and the JSON endpoint.
type Result struct {
	Filters    Filters      `json:"filters"`
	Hits       []Hit        `json:"hits"`
	Facets     Facets       `json:"facets"`
	Total      int          `json:"total"`
	TotalLabel string       `json:"totalLabel"`
	Page       int          `json:"page"`
	Pages      int          `json:"pages"`
	HasHits    bool         `json:"hasHits"`
	HasPages   bool         `json:"hasPages"`
	HasPrev    bool         `json:"hasPrev"`
	HasNext    bool         `json:"hasNext"`
	PrevHref   string       `json:"prevHref"`
	NextHref   string       `json:"nextHref"`
	PageLinks  []PageLink   `json:"pageLinks"`
	Sorts      []SortOption `json:"sorts"`
}

func normalize(values url.Values, categories, sizes []option) Filters {
	f := Filters{
		Q:        boundedText(values.Get("q")),
		Category: selected(values["category"], categories),
		Size:     selected(values["size"], sizes),
		Stock:    values.Get("stock") == "1",
		Sort:     values.Get("sort"),
	}
	f.PriceFrom = number(values.Get("price_from"))
	f.PriceTo = number(values.Get("price_to"))
	if f.PriceFrom != nil && f.PriceTo != nil && *f.PriceFrom > *f.PriceTo {
		f.PriceFrom, f.PriceTo = f.PriceTo, f.PriceFrom
	}
	if !slices.ContainsFunc(sortOptions, func(option option) bool { return option.Value == f.Sort }) {
		f.Sort = "popular"
	}
	page, _ := strconv.Atoi(values.Get("page"))
	f.Page = max(1, page)
	return f
}

func boundedText(value string) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	const maxRunes = 240
	runes := 0
	for at := range value {
		if runes == maxRunes {
			return strings.TrimSpace(value[:at])
		}
		runes++
	}
	return value
}

func number(value string) *int64 {
	n, err := strconv.ParseInt(value, 10, 64)
	// Prices and filters go to JavaScript as exact integers.
	if err != nil || n < 0 || n > 1<<53-1 {
		return nil
	}
	return &n
}

func selected(values []string, options []option) []string {
	result := []string{}
	for _, option := range options {
		if slices.Contains(values, option.Value) {
			result = append(result, option.Value)
		}
	}
	return result
}

func facets(options []option) []Facet {
	result := make([]Facet, len(options))
	for id, option := range options {
		result[id].option = option
	}
	return result
}

// Search returns one page and counts each facet under all the other filters.
// Repeatable values are ORed within a facet. Text terms are ANDed.
func (i *Index) Search(ctx context.Context, values url.Values) (Result, error) {
	f := normalize(values, i.categories, i.sizes)
	q := searchQuery{
		filters:    f,
		match:      matchTerms(f.Q),
		categories: filterList(f.Category),
		sizes:      filterList(f.Size),
	}
	result := Result{
		Filters: f,
		Hits:    make([]Hit, 0, PageSize),
		Facets:  Facets{Category: facets(i.categories), Size: facets(i.sizes)},
	}
	tx, err := i.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Result{}, fmt.Errorf("open catalog search transaction: %w", err)
	}
	defer tx.Rollback()

	if err := q.countFacets(ctx, tx, &result); err != nil {
		return Result{}, fmt.Errorf("count catalog facets: %w", err)
	}
	result.Pages = pageCount(result.Total)
	result.Page = min(f.Page, result.Pages)
	result.Filters.Page = result.Page
	if result.Total > 0 {
		if err := q.loadHits(ctx, tx, &result); err != nil {
			return Result{}, fmt.Errorf("read catalog cards: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("finish catalog search transaction: %w", err)
	}
	result.prepareView()
	return result, nil
}

func pageCount(total int) int { return max(1, (total-1)/PageSize+1) }

type searchQuery struct {
	filters    Filters
	match      string
	categories string
	sizes      string
}

func matchTerms(text string) string {
	words := searchWords(text)
	for i, word := range words {
		words[i] = `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
	}
	return strings.Join(words, " AND ")
}

func filterList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(values) // A slice of strings is always encodable.
	return string(encoded)
}

type disjunctiveFacet int

const (
	categoryFacet disjunctiveFacet = 1 << iota
	sizeFacet
)

func (q searchQuery) fromWhere(join string, omit disjunctiveFacet, matched bool) (string, []any) {
	from := " FROM product AS p"
	var conditions []string
	var args []any
	if matched {
		from = " FROM matching AS p"
	} else {
		if q.match != "" {
			from = " FROM product_text CROSS JOIN product AS p ON p.rowid = product_text.rowid"
			conditions = append(conditions, "product_text MATCH ?")
			args = append(args, q.match)
		}
		if q.filters.PriceFrom != nil {
			conditions = append(conditions, "p.price >= ?")
			args = append(args, *q.filters.PriceFrom)
		}
		if q.filters.PriceTo != nil {
			conditions = append(conditions, "p.price <= ?")
			args = append(args, *q.filters.PriceTo)
		}
		if q.filters.Stock {
			conditions = append(conditions, "p.available = 1")
		}
	}
	if omit&categoryFacet == 0 && q.categories != "" {
		conditions = append(conditions, "p.category IN (SELECT value FROM json_each(?))")
		args = append(args, q.categories)
	}
	if omit&sizeFacet == 0 && q.sizes != "" {
		conditions = append(conditions, `EXISTS (
	SELECT 1 FROM product_size AS chosen WHERE chosen.product = p.rowid
	AND chosen.size IN (SELECT value FROM json_each(?))
)`)
		args = append(args, q.sizes)
	}
	from += join
	if len(conditions) > 0 {
		from += " WHERE " + strings.Join(conditions, " AND ")
	}
	return from, args
}

func (q searchQuery) countFacets(ctx context.Context, tx *sql.Tx, result *Result) error {
	var prefix string
	var args []any
	matched := q.match != ""
	if matched {
		// Share text candidates across the disjunctive facets. CROSS JOIN in
		// the source makes SQLite enumerate FTS matches before rowid lookups.
		from, baseArgs := q.fromWhere("", categoryFacet|sizeFacet, false)
		prefix = "WITH matching AS MATERIALIZED (SELECT p.rowid, p.category, p.available" + from + ") "
		args = append(args, baseArgs...)
	}
	categories, categoryArgs := q.fromWhere("", categoryFacet, matched)
	sizeSource, sizeArgs := q.fromWhere(" CROSS JOIN product_size AS bucket ON bucket.product = p.rowid", sizeFacet, matched)
	sizeCounts := "SELECT 1, bucket.size, count(*), 0" + sizeSource + " GROUP BY bucket.size"
	if !matched && len(sizeArgs) == 0 && !q.filters.Stock {
		sizeCounts = "SELECT 1, size, count(*), 0 FROM product_size GROUP BY size"
	}
	args = append(args, categoryArgs...)
	args = append(args, sizeArgs...)
	rows, err := tx.QueryContext(ctx, prefix+
		"SELECT 0, p.category, count(*), count(*) FILTER (WHERE p.available = 1)"+categories+" GROUP BY p.category UNION ALL "+sizeCounts, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	categoryCounts := make(map[string]int, len(result.Facets.Category))
	sizeCountsByName := make(map[string]int, len(result.Facets.Size))
	for rows.Next() {
		var name string
		var kind, total, available int
		if err := rows.Scan(&kind, &name, &total, &available); err != nil {
			return err
		}
		if kind == 1 {
			sizeCountsByName[name] = total
			continue
		}
		categoryCounts[name] = total
		// The category groups ignore category selections. Summing the selected
		// groups recovers the total and the available-only count.
		if len(q.filters.Category) == 0 || slices.Contains(q.filters.Category, name) {
			result.Total += total
			result.Facets.Stock += available
		}
	}
	for id, option := range result.Facets.Category {
		result.Facets.Category[id].Count = categoryCounts[option.Value]
	}
	for id, option := range result.Facets.Size {
		result.Facets.Size[id].Count = sizeCountsByName[option.Value]
	}
	return rows.Err()
}

func orderBy(sort string) string {
	const tieBreak = "p.position, p.id"
	switch sort {
	case "cheap":
		return "p.price, " + tieBreak
	case "expensive":
		return "p.price DESC, " + tieBreak
	case "new":
		return "p.arrival DESC, " + tieBreak
	default:
		return tieBreak
	}
}

func (q searchQuery) loadHits(ctx context.Context, tx *sql.Tx, result *Result) error {
	from, args := q.fromWhere("", 0, false)
	args = append(args, PageSize, (result.Page-1)*PageSize)
	order := orderBy(q.filters.Sort)
	rows, err := tx.QueryContext(ctx, `
WITH page AS MATERIALIZED (
	SELECT p.rowid`+from+` ORDER BY `+order+` LIMIT ? OFFSET ?
)
SELECT p.id, p.slug, p.title, p.category, c.title, p.price, p.original_price, p.available,
	(SELECT json_group_array(ps.size ORDER BY ps.position)
	 FROM product_size AS ps WHERE ps.product = p.rowid)
FROM page
JOIN product AS p ON p.rowid = page.rowid
JOIN category AS c ON c.id = p.category
ORDER BY `+order, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var hit Hit
		var sizes string
		if err := rows.Scan(&hit.ID, &hit.Slug, &hit.Title, &hit.Category, &hit.CategoryTitle, &hit.Price, &hit.OriginalPrice, &hit.Available, &sizes); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(sizes), &hit.Sizes); err != nil {
			return fmt.Errorf("decode sizes for catalog product %q: %w", hit.ID, err)
		}
		if hit.Sizes == nil {
			hit.Sizes = []string{}
		}
		result.Hits = append(result.Hits, hit)
	}
	return rows.Err()
}
