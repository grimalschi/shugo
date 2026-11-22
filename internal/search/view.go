package search

import (
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

func (r *Result) navigation(d Definition) {
	r.HasHits = r.Total > 0
	r.HasPages = r.Pages > 1
	r.HasPrev = r.Page > 1
	r.HasNext = r.Page < r.Pages
	for _, name := range slices.Sorted(maps.Keys(d.Options.Sorts)) {
		sort := d.Options.Sorts[name]
		r.Sorts = append(r.Sorts, SortChoice{Value: name, Label: sort.Label, Selected: name == r.Query.Sort})
	}
	if !r.HasPages {
		return
	}

	params := r.Query.values(d.Options.DefaultSort)
	href := func(page int) string {
		if page > 1 {
			params.Set("page", strconv.Itoa(page))
		} else {
			params.Del("page")
		}
		return "?" + formEncoding.Replace(params.Encode())
	}
	if r.HasPrev {
		r.PrevHref = href(r.Page - 1)
	}
	if r.HasNext {
		r.NextHref = href(r.Page + 1)
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
		link := PageLink{Label: strconv.Itoa(page), Current: page == r.Page}
		if !link.Current {
			link.Href = href(page)
		}
		r.PageLinks = append(r.PageLinks, link)
		last = page
	}
}

var formEncoding = strings.NewReplacer("~", "%7E", "%2A", "*")

func (q Query) values(defaultSort string) url.Values {
	values := url.Values{}
	if q.Q != "" {
		values.Set("q", q.Q)
	}
	if q.Sort != defaultSort {
		values.Set("sort", q.Sort)
	}
	for name, value := range q.Filters {
		switch v := value.(type) {
		case []string:
			if len(v) > 0 {
				values[name] = v
			}
		case RangeValue:
			if v.From != nil {
				values.Set(name+"_from", strconv.FormatFloat(*v.From, 'g', -1, 64))
			}
			if v.To != nil {
				values.Set(name+"_to", strconv.FormatFloat(*v.To, 'g', -1, 64))
			}
		case bool:
			if v {
				values.Set(name, "1")
			} else {
				values.Set(name, "0")
			}
		}
	}
	return values
}
