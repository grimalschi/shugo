package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type RangeValue struct {
	From *float64 `json:"from"`
	To   *float64 `json:"to"`
}

type Query struct {
	Q       string         `json:"q"`
	Sort    string         `json:"sort"`
	Page    int            `json:"page"`
	Filters map[string]any `json:"filters"`
}

type FacetOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
	Selected bool   `json:"selected"`
}

type Facet struct {
	Type    string        `json:"type"`
	Label   string        `json:"label"`
	Options []FacetOption `json:"options"`
	Min     *float64      `json:"min"`
	Max     *float64      `json:"max"`
	Count   int           `json:"count"`
}

type SortChoice struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
}

type PageLink struct {
	Gap     bool   `json:"gap"`
	Current bool   `json:"current"`
	Label   string `json:"label"`
	Href    string `json:"href"`
}

type Result struct {
	Endpoint  string           `json:"endpoint"`
	Hits      []map[string]any `json:"hits"`
	Query     Query            `json:"query"`
	Facets    map[string]Facet `json:"facets"`
	Sorts     []SortChoice     `json:"sorts"`
	Total     int              `json:"total"`
	Page      int              `json:"page"`
	Pages     int              `json:"pages"`
	HasHits   bool             `json:"hasHits"`
	HasPages  bool             `json:"hasPages"`
	HasPrev   bool             `json:"hasPrev"`
	HasNext   bool             `json:"hasNext"`
	PrevHref  string           `json:"prevHref"`
	NextHref  string           `json:"nextHref"`
	PageLinks []PageLink       `json:"pageLinks"`
}

type predicate struct {
	sql  string
	args []any
}

type applied struct {
	name    string
	column  string
	filter  Filter
	matches []predicate
}

type queryPlan struct {
	source   string
	scope    []predicate
	hitScope []predicate
	filters  []applied
}

// Search reads hits and disjunctive facets in the same generation. Fixed
// equality predicates apply to all results, including each facet's options.
func (i *Index) Search(ctx context.Context, definition Definition, params url.Values) (Result, error) {
	d, err := Normalize(Declaration{Table: definition.Table, Options: definition.Options})
	if err != nil {
		return Result{}, err
	}
	if d.ID != definition.ID || d.IndexID != definition.IndexID || d.IndexID != i.meta.ID {
		return Result{}, fmt.Errorf("definition %q does not match this search index", definition.ID)
	}
	tx, err := i.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()

	query := parseQuery(d, params)
	scope := i.fixed(d)
	domains, err := i.multiDomains(ctx, tx, d, scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Endpoint: d.Endpoint(), Hits: []map[string]any{}, Query: query,
		Facets: make(map[string]Facet, len(d.Options.Filters)), Sorts: []SortChoice{}, PageLinks: []PageLink{},
	}
	plan := i.plan(d, params, scope, domains, &result)
	if err := plan.facets(ctx, tx, &result); err != nil {
		return Result{}, fmt.Errorf("count search facets: %w", err)
	}
	result.Pages = max(1, (result.Total-1)/d.Options.PageSize+1)
	result.Page = min(query.Page, result.Pages)
	result.Query.Page = result.Page
	if result.Total > 0 {
		if err := i.hits(ctx, tx, plan, d, &result); err != nil {
			return Result{}, fmt.Errorf("read search hits: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Result{}, err
	}
	result.navigation(d)
	return result, nil
}

func parseQuery(d Definition, params url.Values) Query {
	q := Query{Sort: d.Options.DefaultSort, Page: 1, Filters: map[string]any{}}
	if len(d.Options.Text) > 0 {
		q.Q = limitQuery(params.Get("q"))
	}
	if sort := params.Get("sort"); d.Options.Sorts[sort].By != nil {
		q.Sort = sort
	}
	if page, err := strconv.Atoi(params.Get("page")); err == nil && page >= 1 {
		q.Page = page
	}
	return q
}

func limitQuery(q string) string {
	q = strings.TrimSpace(strings.ToValidUTF8(q, ""))
	count := 0
	for at := range q {
		if count == 240 {
			return strings.TrimSpace(q[:at])
		}
		count++
	}
	return q
}

func number(text string) *float64 {
	n, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return nil
	}
	return &n
}

func boolean(text string) *bool {
	switch text {
	case "1", "true":
		v := true
		return &v
	case "0", "false":
		v := false
		return &v
	default:
		return nil
	}
}

func (i *Index) fixed(d Definition) []predicate {
	scope := make([]predicate, 0, len(d.Options.Where))
	for _, name := range slices.Sorted(maps.Keys(d.Options.Where)) {
		scope = append(scope, predicate{
			sql: "r." + i.columns[name] + " IS ?", args: []any{d.Options.Where[name]},
		})
	}
	return scope
}

func (i *Index) multiDomains(ctx context.Context, tx *sql.Tx, d Definition, scope []predicate) (map[string][]string, error) {
	i.domainsMu.RLock()
	existing, known := i.domains[d.ID]
	i.domainsMu.RUnlock()
	if known {
		return existing, nil
	}
	domains := map[string][]string{}
	for name, f := range d.Options.Filters {
		if f.Type != Multi {
			continue
		}
		col := "r." + i.columns[f.Column]
		domain, err := multiDomain(ctx, tx, col, scope)
		if err != nil {
			return nil, fmt.Errorf("read search filter %q options: %w", name, err)
		}
		domains[name] = domain
	}
	i.domainsMu.Lock()
	i.domains[d.ID] = domains
	i.domainsMu.Unlock()
	return domains, nil
}

func multiDomain(ctx context.Context, tx *sql.Tx, column string, scope []predicate) ([]string, error) {
	condition := predicate{sql: column + " IS NOT NULL"}
	from, args := sqlFrom(" FROM records AS r", scope, []predicate{condition})
	rows, err := tx.QueryContext(ctx, "SELECT DISTINCT CAST("+column+" AS TEXT) AS value"+from+" ORDER BY value", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var domain []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		domain = append(domain, value)
	}
	return domain, rows.Err()
}

func textMatch(q string) string {
	words := stemWords(q, nil)
	for at, word := range words {
		words[at] = `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
	}
	return strings.Join(words, " AND ")
}

func (i *Index) plan(d Definition, params url.Values, scope []predicate, domains map[string][]string, result *Result) queryPlan {
	plan := queryPlan{source: " FROM records AS r", scope: scope, hitScope: slices.Clone(scope)}
	if match := textMatch(result.Query.Q); match != "" {
		plan.source = " FROM search_terms CROSS JOIN records AS r ON r.rowid = search_terms.rowid"
		plan.hitScope = append(plan.hitScope, predicate{sql: "search_terms MATCH ?", args: []any{match}})
	}
	for _, name := range slices.Sorted(maps.Keys(d.Options.Filters)) {
		f := d.Options.Filters[name]
		a := applied{name: name, column: "r." + i.columns[f.Column], filter: f}
		facet := Facet{Type: f.Type, Label: f.Label, Options: []FacetOption{}}
		switch f.Type {
		case Multi:
			selected := []string{}
			for _, value := range domains[name] {
				checked := slices.Contains(params[name], value)
				if checked {
					selected = append(selected, value)
				}
				facet.Options = append(facet.Options, FacetOption{Value: value, Label: value, Selected: checked})
			}
			result.Query.Filters[name] = selected
			if len(selected) > 0 {
				data, _ := json.Marshal(selected)
				a.matches = append(a.matches, predicate{
					sql: "CAST(" + a.column + " AS TEXT) IN (SELECT value FROM json_each(?))", args: []any{string(data)},
				})
			}
		case Range:
			value := RangeValue{From: number(params.Get(name + "_from")), To: number(params.Get(name + "_to"))}
			if value.From != nil && value.To != nil && *value.From > *value.To {
				value.From, value.To = value.To, value.From
			}
			result.Query.Filters[name] = value
			if value.From != nil || value.To != nil {
				a.matches = append(a.matches, predicate{sql: numeric(a.column)})
			}
			if value.From != nil {
				a.matches = append(a.matches, predicate{sql: a.column + " >= ?", args: []any{*value.From}})
			}
			if value.To != nil {
				a.matches = append(a.matches, predicate{sql: a.column + " <= ?", args: []any{*value.To}})
			}
		case Boolean:
			value := boolean(params.Get(name))
			result.Query.Filters[name] = nil
			if value != nil {
				result.Query.Filters[name] = *value
				a.matches = append(a.matches, predicate{sql: a.column + " = ?", args: []any{*value}})
			}
		}
		result.Facets[name] = facet
		plan.filters = append(plan.filters, a)
	}
	return plan
}

func numeric(column string) string { return "typeof(" + column + ") IN ('integer','real')" }

func sqlFrom(source string, scope, selection []predicate) (string, []any) {
	var where []string
	var args []any
	for _, list := range [][]predicate{scope, selection} {
		for _, one := range list {
			where = append(where, one.sql)
			args = append(args, one.args...)
		}
	}
	if len(where) > 0 {
		source += " WHERE " + strings.Join(where, " AND ")
	}
	return source, args
}

func (q queryPlan) selection(omit string) []predicate {
	var matches []predicate
	for _, a := range q.filters {
		if a.name != omit {
			matches = append(matches, a.matches...)
		}
	}
	return matches
}

func (q queryPlan) facets(ctx context.Context, tx *sql.Tx, result *Result) error {
	var prefix string
	var args []any
	source, scope := q.source, q.hitScope
	if len(q.hitScope) > len(q.scope) && len(q.filters) > 0 {
		// Enumerate FTS matches once. The CTE only carries facet fields.
		var fields []string
		for _, a := range q.filters {
			fields = append(fields, a.column)
		}
		base, baseArgs := sqlFrom(q.source, q.hitScope, nil)
		prefix = "WITH matching AS MATERIALIZED (SELECT " + strings.Join(uniqueSorted(fields), ",") + base + ") "
		args = append(args, baseArgs...)
		source, scope = " FROM matching AS r", nil
	}
	from, totalArgs := sqlFrom(source, scope, q.selection(""))
	parts := []string{"SELECT -1, NULL, count(*), NULL, NULL" + from}
	args = append(args, totalArgs...)
	for at, a := range q.filters {
		selection := q.selection(a.name)
		var selectSQL, group string
		switch a.filter.Type {
		case Multi:
			selectSQL = "CAST(" + a.column + " AS TEXT), count(*), NULL, NULL"
			selection = append(selection, predicate{sql: a.column + " IS NOT NULL"})
			group = " GROUP BY 2"
		case Range:
			selectSQL = "NULL, 0, min(" + a.column + "), max(" + a.column + ")"
			selection = append(selection, predicate{sql: numeric(a.column)})
		case Boolean:
			selectSQL = "NULL, count(*), NULL, NULL"
			selection = append(selection, predicate{sql: a.column + " = 1"})
		}
		from, filterArgs := sqlFrom(source, scope, selection)
		parts = append(parts, "SELECT "+strconv.Itoa(at)+", "+selectSQL+from+group)
		args = append(args, filterArgs...)
	}
	rows, err := tx.QueryContext(ctx, prefix+strings.Join(parts, " UNION ALL "), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	counts := make([]map[string]int, len(q.filters))
	for rows.Next() {
		var at, count int
		var value sql.NullString
		var lo, hi *float64
		if err := rows.Scan(&at, &value, &count, &lo, &hi); err != nil {
			return err
		}
		if at < 0 {
			result.Total = count
			continue
		}
		a := q.filters[at]
		facet := result.Facets[a.name]
		switch a.filter.Type {
		case Multi:
			if counts[at] == nil {
				counts[at] = map[string]int{}
			}
			counts[at][value.String] = count
		case Range:
			facet.Min, facet.Max = lo, hi
		case Boolean:
			facet.Count = count
		}
		result.Facets[a.name] = facet
	}
	for at, a := range q.filters {
		for pos, choice := range result.Facets[a.name].Options {
			result.Facets[a.name].Options[pos].Count = counts[at][choice.Value]
		}
	}
	return rows.Err()
}

func (i *Index) hits(ctx context.Context, tx *sql.Tx, q queryPlan, d Definition, result *Result) error {
	fields := uniqueSorted(append(d.projection(), i.meta.Primary))
	columns := make([]string, len(fields))
	for at, field := range fields {
		columns[at] = "r." + i.columns[field]
	}
	var ordering []string
	for _, order := range d.Options.Sorts[result.Query.Sort].By {
		term := "r." + i.columns[order.Column]
		if order.Direction == "desc" {
			term += " DESC"
		}
		ordering = append(ordering, term)
	}
	ordering = append(ordering, "r.rowid")
	from, args := sqlFrom(q.source, q.hitScope, q.selection(""))
	args = append(args, d.Options.PageSize, (result.Page-1)*d.Options.PageSize)
	rows, err := tx.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+from+
		" ORDER BY "+strings.Join(ordering, ",")+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	values, pointers := rowSlots(len(fields))
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		hit := make(map[string]any, len(fields))
		for at, field := range fields {
			hit[field] = values[at]
		}
		result.Hits = append(result.Hits, hit)
	}
	return rows.Err()
}
