package catalog

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// The experiment has one catalog route. The results API returns pagination
// links for it, using the same order and escaping as URLSearchParams.
const catalogPath = "/catalog"

type SortOption struct {
	option
	Selected bool `json:"selected"`
}

var sortOptions = []option{
	{Value: "popular", Label: "Most popular"},
	{Value: "cheap", Label: "Lowest price first"},
	{Value: "expensive", Label: "Highest price first"},
	{Value: "new", Label: "Newest first"},
}

type PageLink struct {
	Gap     bool   `json:"gap"`
	Current bool   `json:"current"`
	Label   string `json:"label"`
	Href    string `json:"href"`
}

// prepareView computes the values shared by the browser and the HTML
// renderer. Both get the same labels, selected options and link targets.
func (r *Result) prepareView() {
	r.TotalLabel = "Products: " + strconv.Itoa(r.Total)
	r.HasHits = r.Total > 0
	r.HasPages = r.Pages > 1
	r.HasPrev = r.Page > 1
	r.HasNext = r.Page < r.Pages
	r.Facets.StockLabel = countLabel(r.Facets.Stock)
	selectedFacets(r.Facets.Category, r.Filters.Category)
	selectedFacets(r.Facets.Size, r.Filters.Size)
	for at := range r.Hits {
		item := &r.Hits[at]
		item.Discounted = item.Price != item.OriginalPrice
		item.PriceLabel = money(item.Price)
		item.OriginalPriceLabel = item.PriceLabel
		if item.Discounted {
			item.OriginalPriceLabel = money(item.OriginalPrice)
		}
		item.StockClass, item.StockLabel = "product-stock", "Out of stock"
		if item.Available {
			item.StockClass, item.StockLabel = "product-stock available", "In stock"
		}
		item.HasSizes = len(item.Sizes) > 0
		item.SizesLabel = strings.Join(item.Sizes, " · ")
	}
	r.Sorts = make([]SortOption, len(sortOptions))
	for at, option := range sortOptions {
		r.Sorts[at] = SortOption{option: option, Selected: r.Filters.Sort == option.Value}
	}
	r.PageLinks = []PageLink{}
	if r.HasPages {
		r.preparePageLinks()
	}
}

func countLabel(count int) string {
	if count == 1 {
		return "1 product"
	}
	return strconv.Itoa(count) + " products"
}

func selectedFacets(facets []Facet, selected []string) {
	for at := range facets {
		row := &facets[at]
		row.CountLabel = countLabel(row.Count)
		row.Selected = slices.Contains(selected, row.Value)
	}
}

func money(value int64) string {
	text := strconv.FormatInt(value, 10)
	var out strings.Builder
	out.Grow(len(text) + len(text)/3 + 1)
	digits := text
	if value < 0 {
		out.WriteByte('-')
		digits = digits[1:]
	}
	out.WriteByte('$')
	for at := range len(digits) {
		if at > 0 && (len(digits)-at)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteByte(digits[at])
	}
	return out.String()
}

func (r *Result) preparePageLinks() {
	base := href(r.Filters)
	if r.HasPrev {
		r.PrevHref = pageHref(base, r.Page-1)
	}
	if r.HasNext {
		r.NextHref = pageHref(base, r.Page+1)
	}
	candidates := []int{1, r.Page - 1, r.Page, r.Page + 1, r.Pages}
	slices.Sort(candidates)
	last := 0
	for _, page := range candidates {
		if page < 1 || page > r.Pages || page == last {
			continue
		}
		if last > 0 && page-last > 1 {
			r.PageLinks = append(r.PageLinks, PageLink{Gap: true, Label: "…"})
		}
		link := PageLink{Current: page == r.Page, Label: strconv.Itoa(page)}
		if !link.Current {
			link.Href = pageHref(base, page)
		}
		r.PageLinks = append(r.PageLinks, link)
		last = page
	}
}

func pageHref(base string, page int) string {
	if page == 1 {
		return base
	}
	delimiter := "&"
	if base == catalogPath {
		delimiter = "?"
	}
	return base + delimiter + "page=" + strconv.Itoa(page)
}

var formEscape = strings.NewReplacer("~", "%7E", "%2A", "*")

func href(filters Filters) string {
	var out strings.Builder
	out.WriteString(catalogPath)
	add := func(name, value string) {
		if out.Len() == len(catalogPath) {
			out.WriteByte('?')
		} else {
			out.WriteByte('&')
		}
		out.WriteString(name)
		out.WriteByte('=')
		out.WriteString(formEscape.Replace(url.QueryEscape(value)))
	}
	if filters.Q != "" {
		add("q", filters.Q)
	}
	for _, category := range filters.Category {
		add("category", category)
	}
	for _, size := range filters.Size {
		add("size", size)
	}
	if filters.Stock {
		add("stock", "1")
	}
	if filters.PriceFrom != nil {
		add("price_from", strconv.FormatInt(*filters.PriceFrom, 10))
	}
	if filters.PriceTo != nil {
		add("price_to", strconv.FormatInt(*filters.PriceTo, 10))
	}
	if filters.Sort != "popular" {
		add("sort", filters.Sort)
	}
	return out.String()
}
