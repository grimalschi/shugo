package catalog

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
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
	CategoryTitle string `json:"categoryTitle"`
}

type Facet struct {
	option
	Count int `json:"count"`
}

type Facets struct {
	Category []Facet `json:"category"`
	Size     []Facet `json:"size"`
	Stock    int     `json:"stock"`
}

// Result is shared by the default render, request SSR and the JSON endpoint.
type Result struct {
	Filters Filters `json:"filters"`
	Hits    []Hit   `json:"hits"`
	Facets  Facets  `json:"facets"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	Pages   int     `json:"pages"`
}

var sortNames = []string{"popular", "cheap", "expensive", "new"}

func (i *Index) orderProducts() {
	for _, sort := range sortNames {
		order := make([]int, len(i.data.Products))
		for id := range order {
			order[id] = id
		}
		slices.SortFunc(order, func(a, b int) int {
			first, second := i.data.Products[a], i.data.Products[b]
			var result int
			switch sort {
			case "cheap":
				result = cmp.Compare(first.Price, second.Price)
			case "expensive":
				result = cmp.Compare(second.Price, first.Price)
			case "new":
				result = cmp.Compare(second.Arrival, first.Arrival)
			}
			if result == 0 {
				result = cmp.Compare(first.Position, second.Position)
			}
			if result == 0 {
				result = cmp.Compare(first.ID, second.ID)
			}
			return result
		})
		i.orders[sort] = order
	}
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
	if !slices.Contains(sortNames, f.Sort) {
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

// Search filters the prepared order and counts each facet under all the
// other filters. Repeatable filter values are ORed within one facet.
func (i *Index) Search(values url.Values) Result {
	f := normalize(values, i.data.Categories, i.sizes)
	words := strings.FieldsFunc(fold(f.Q), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	result := Result{Filters: f, Facets: Facets{Category: facets(i.data.Categories), Size: facets(i.sizes)}}
	page := min(f.Page, pageCount(len(i.data.Products)))
	first, end := (page-1)*PageSize, page*PageSize
	var hits, last [PageSize]int
	for _, id := range i.orders[f.Sort] {
		p := &i.data.Products[id]
		if f.PriceFrom != nil && p.Price < *f.PriceFrom || f.PriceTo != nil && p.Price > *f.PriceTo || !matchesText(p.Text, words) {
			continue
		}
		category := len(f.Category) == 0 || slices.Contains(f.Category, p.Category)
		size := matchesSize(p.Sizes, f.Size)
		stock := !f.Stock || p.Available
		if size && stock {
			result.Facets.Category[i.categories[p.Category]].Count++
		}
		if category && stock {
			for _, value := range p.Sizes {
				result.Facets.Size[i.sizeIDs[value]].Count++
			}
		}
		if category && size && p.Available {
			result.Facets.Stock++
		}
		if category && size && stock {
			if result.Total >= first && result.Total < end {
				hits[result.Total-first] = id
			}
			if page > 1 {
				last[result.Total%PageSize] = id
			}
			result.Total++
		}
	}

	result.Pages = pageCount(result.Total)
	result.Page = min(page, result.Pages)
	result.Filters.Page = result.Page
	if page > result.Pages {
		first = (result.Page - 1) * PageSize
		hits = last
	}
	count := min(PageSize, result.Total-first)
	result.Hits = make([]Hit, 0, count)
	for _, id := range hits[:count] {
		p := i.data.Products[id].Product
		p.Sizes = append([]string{}, p.Sizes...)
		result.Hits = append(result.Hits, Hit{Product: p, CategoryTitle: i.data.Categories[i.categories[p.Category]].Label})
	}
	return result
}

func pageCount(total int) int { return max(1, (total+PageSize-1)/PageSize) }

func matchesText(text string, words []string) bool {
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}

func matchesSize(sizes, selected []string) bool {
	if len(selected) == 0 {
		return true
	}
	for _, size := range sizes {
		if slices.Contains(selected, size) {
			return true
		}
	}
	return false
}
