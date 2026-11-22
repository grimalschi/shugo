package catalog

import (
	"net/url"
	"reflect"
	"testing"
)

func TestSearchView(t *testing.T) {
	index, _, _, _ := testShop(t)
	result := searchTestIndex(t, index, url.Values{"sort": {"cheap"}})
	if !result.HasHits || result.HasPages || result.HasPrev || result.HasNext ||
		result.TotalLabel != "Products: 4" || result.PageLinks == nil || len(result.PageLinks) != 0 {
		t.Fatalf("initial view: %+v", result)
	}
	one, two := result.Hits[0], result.Hits[1]
	if one.PriceLabel != "$61" || one.OriginalPriceLabel != "$121" || !one.Discounted ||
		one.StockClass != "product-stock available" || one.StockLabel != "In stock" ||
		!one.HasSizes || one.SizesLabel != "small" || two.Discounted || two.StockClass != "product-stock" ||
		two.StockLabel != "Out of stock" {
		t.Fatalf("card labels: %+v; %+v", one, two)
	}
	for _, sort := range result.Sorts {
		if sort.Selected != (sort.Value == "cheap") {
			t.Fatalf("unexpected sort selection: %+v", result.Sorts)
		}
	}
	if got := money(-1234567); got != "-$1,234,567" {
		t.Fatalf("price label %q", got)
	}
	empty := searchTestIndex(t, index, url.Values{"q": {"nonexistent product"}})
	if empty.HasHits || empty.HasPages || empty.TotalLabel != "Products: 0" || empty.PrevHref != "" || empty.NextHref != "" {
		t.Fatalf("empty view: %+v", empty)
	}
}

func TestPageLinks(t *testing.T) {
	low, high := int64(192), int64(9000)
	filter := Filters{Q: "tea & ~*' <x>", Category: []string{"c2", "c4"}, Size: []string{"small"},
		Stock: true, PriceFrom: &low, PriceTo: &high, Sort: "cheap", Page: 5}
	result := Result{Filters: filter, Page: 5, Pages: 10}
	result.prepareView()
	const base = "/catalog?q=tea+%26+%7E*%27+%3Cx%3E&category=c2&category=c4&size=small&stock=1&price_from=192&price_to=9000&sort=cheap"
	if !result.HasPrev || !result.HasNext || result.PrevHref != base+"&page=4" || result.NextHref != base+"&page=6" {
		t.Fatalf("previous and next links: %q, %q", result.PrevHref, result.NextHref)
	}
	want := []PageLink{
		{Label: "1", Href: base},
		{Gap: true, Label: "…"},
		{Label: "4", Href: base + "&page=4"},
		{Current: true, Label: "5"},
		{Label: "6", Href: base + "&page=6"},
		{Gap: true, Label: "…"},
		{Label: "10", Href: base + "&page=10"},
	}
	if !reflect.DeepEqual(result.PageLinks, want) {
		t.Fatalf("page navigation: %+v", result.PageLinks)
	}

	first := Result{Filters: Filters{Sort: "popular", Page: 1}, Page: 1, Pages: 2}
	first.prepareView()
	if first.HasPrev || first.PrevHref != "" || !first.HasNext || first.NextHref != "/catalog?page=2" ||
		!reflect.DeepEqual(first.PageLinks, []PageLink{{Current: true, Label: "1"}, {Label: "2", Href: "/catalog?page=2"}}) {
		t.Fatalf("first page links: %+v", first)
	}
	last := Result{Filters: Filters{Sort: "popular", Page: 2}, Page: 2, Pages: 2}
	last.prepareView()
	if !last.HasPrev || last.PrevHref != "/catalog" || last.HasNext || last.NextHref != "" ||
		!reflect.DeepEqual(last.PageLinks, []PageLink{{Label: "1", Href: "/catalog"}, {Current: true, Label: "2"}}) {
		t.Fatalf("last page links: %+v", last)
	}
}
