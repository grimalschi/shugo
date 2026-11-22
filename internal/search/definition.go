// Package search builds and queries immutable SQLite projections of site data.
package search

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const EndpointPrefix = "/_islands/search/"

type Declaration struct {
	Table   string  `json:"table"`
	Options Options `json:"options"`
}

type Options struct {
	Text        []string          `json:"text,omitempty"`
	Where       map[string]any    `json:"where,omitempty"`
	Fields      []string          `json:"fields,omitempty"`
	Filters     map[string]Filter `json:"filters,omitempty"`
	Sorts       map[string]Sort   `json:"sorts,omitempty"`
	DefaultSort string            `json:"defaultSort,omitempty"`
	PageSize    int               `json:"pageSize,omitempty"`
}

func (o *Options) UnmarshalJSON(data []byte) error {
	type options Options
	return decodeStrict(data, (*options)(o))
}

const (
	Multi   = "multi"
	Range   = "range"
	Boolean = "boolean"
)

type Filter struct {
	Column string `json:"column"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
}

func (f *Filter) UnmarshalJSON(data []byte) error {
	type filter Filter
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var kind string
		if err := json.Unmarshal(data, &kind); err != nil {
			return err
		}
		*f = Filter{Type: kind}
		return nil
	}
	return decodeStrict(data, (*filter)(f))
}

type Order struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type Sort struct {
	By    []Order `json:"by"`
	Label string  `json:"label,omitempty"`
}

func (s *Sort) UnmarshalJSON(data []byte) error {
	type sort Sort
	decoder := json.NewDecoder(bytes.NewReader(data))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return fmt.Errorf("sort must be an object")
	}
	var shorthand []Order
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		column := token.(string)
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		if column == "by" && len(value) > 0 && value[0] == '[' {
			return decodeStrict(data, (*sort)(s))
		}
		var direction string
		if err := json.Unmarshal(value, &direction); err != nil {
			return fmt.Errorf("sort column %q direction must be a string", column)
		}
		shorthand = append(shorthand, Order{Column: column, Direction: direction})
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	*s = Sort{By: shorthand}
	return nil
}

func decodeStrict(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return fmt.Errorf("unexpected data after search options")
	}
	return nil
}

// Definition names one search binding and the projection it reads. Bindings
// with the same physical specification can share an index.
type Definition struct {
	ID      string  `json:"id"`
	IndexID string  `json:"indexId"`
	Table   string  `json:"table"`
	Options Options `json:"options"`
}

func (d Definition) Endpoint() string { return EndpointPrefix + d.ID }

// Columns is the set of data columns that can change this projection, in
// addition to the source primary key.
func (d Definition) Columns() []string {
	names := append(d.projection(), slices.Collect(maps.Keys(d.Options.Where))...)
	return uniqueSorted(names)
}

func (d Definition) projection() []string {
	names := slices.Concat(d.Options.Fields, d.Options.Text, d.indexColumns())
	return uniqueSorted(names)
}

func (d Definition) indexColumns() []string {
	var names []string
	for _, f := range d.Options.Filters {
		names = append(names, f.Column)
	}
	for _, sort := range d.Options.Sorts {
		for _, order := range sort.By {
			names = append(names, order.Column)
		}
	}
	return uniqueSorted(names)
}

// UsesQuery tells whether a URL has a parameter owned by this binding.
func (d Definition) UsesQuery(values url.Values) bool {
	if values.Has("page") || len(d.Options.Text) > 0 && values.Has("q") ||
		len(d.Options.Sorts) > 0 && values.Has("sort") {
		return true
	}
	for name, filter := range d.Options.Filters {
		if filter.Type == Range {
			if values.Has(name+"_from") || values.Has(name+"_to") {
				return true
			}
		} else if values.Has(name) {
			return true
		}
	}
	return false
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z_0-9]*$`)

func validIdentifier(name string) bool { return len(name) <= 128 && identifier.MatchString(name) }

// Normalize validates a declaration and derives its reproducible identities.
func Normalize(decl Declaration) (Definition, error) {
	table := strings.ToLower(decl.Table)
	if !validIdentifier(decl.Table) || strings.HasPrefix(table, "sqlite_") || strings.HasPrefix(table, "_shugo_") {
		return Definition{}, fmt.Errorf("invalid search table %q", decl.Table)
	}
	d := Definition{Table: table}
	o := decl.Options
	d.Options.Text = uniqueSorted(slices.Clone(o.Text))
	d.Options.Fields = uniqueSorted(slices.Clone(o.Fields))
	for _, name := range slices.Concat(d.Options.Fields, d.Options.Text) {
		if !validIdentifier(name) {
			return Definition{}, fmt.Errorf("invalid search column %q", name)
		}
	}
	d.Options.PageSize = o.PageSize
	if o.PageSize == 0 {
		d.Options.PageSize = 24
	} else if o.PageSize < 1 || o.PageSize > 250 {
		return Definition{}, fmt.Errorf("search pageSize must be between 1 and 250")
	}
	var err error
	if d.Options.Where, err = normalizeWhere(o.Where); err != nil {
		return Definition{}, err
	}
	if d.Options.Filters, err = normalizeFilters(o.Filters); err != nil {
		return Definition{}, err
	}
	if d.Options.Sorts, d.Options.DefaultSort, err = normalizeSorts(o.Sorts, o.DefaultSort); err != nil {
		return Definition{}, err
	}
	d.ID = digest(Declaration{Table: d.Table, Options: d.Options})
	d.IndexID = digest(struct {
		Table                string
		Text, Columns, KeyBy []string
	}{d.Table, d.Options.Text, d.Columns(), d.indexedColumns()})
	return d, nil
}

func (d Definition) indexedColumns() []string {
	return uniqueSorted(append(d.indexColumns(), slices.Collect(maps.Keys(d.Options.Where))...))
}

func normalizeWhere(input map[string]any) (map[string]any, error) {
	if len(input) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(input))
	for _, column := range slices.Sorted(maps.Keys(input)) {
		if !validIdentifier(column) {
			return nil, fmt.Errorf("invalid where column %q", column)
		}
		value, err := normalizeScalar(input[column])
		if err != nil {
			return nil, fmt.Errorf("where.%s: %w", column, err)
		}
		out[column] = value
	}
	return out, nil
}

func normalizeScalar(input any) (any, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("expected a JSON scalar: %w", err)
	}
	var decoded any
	if err := decodeStrict(data, &decoded); err != nil {
		return nil, err
	}
	switch v := decoded.(type) {
	case nil, bool, string:
		return v, nil
	case json.Number:
		if whole, err := v.Int64(); err == nil {
			return whole, nil
		}
		if real, err := v.Float64(); err == nil && !math.IsInf(real, 0) && !math.IsNaN(real) {
			if real == math.Trunc(real) && real >= -1<<53 && real <= 1<<53 {
				return int64(real), nil
			}
			return real, nil
		}
	}
	return nil, fmt.Errorf("expected a string, number, boolean or null")
}

func normalizeFilters(input map[string]Filter) (map[string]Filter, error) {
	if len(input) == 0 {
		return nil, nil
	}
	out := make(map[string]Filter, len(input))
	params := map[string]string{"page": "page", "q": "q", "sort": "sort"}
	for _, name := range slices.Sorted(maps.Keys(input)) {
		f := input[name]
		if f.Column == "" {
			f.Column = name
		}
		if !validIdentifier(name) || !validIdentifier(f.Column) {
			return nil, fmt.Errorf("invalid filter %q column %q", name, f.Column)
		}
		if f.Label == "" {
			f.Label = name
		}
		keys := []string{name}
		switch f.Type {
		case Multi, Boolean:
		case Range:
			keys = []string{name + "_from", name + "_to"}
		default:
			return nil, fmt.Errorf("filter %q requires type multi, range or boolean", name)
		}
		for _, key := range keys {
			if existing, used := params[key]; used {
				return nil, fmt.Errorf("filter %q parameter %q conflicts with %q", name, key, existing)
			}
			params[key] = name
		}
		out[name] = f
	}
	return out, nil
}

func normalizeSorts(input map[string]Sort, fallback string) (map[string]Sort, string, error) {
	if len(input) == 0 && fallback == "" {
		return nil, "", nil
	}
	out := make(map[string]Sort, len(input))
	names := slices.Sorted(maps.Keys(input))
	for _, name := range names {
		sort := input[name]
		if !validIdentifier(name) || len(sort.By) == 0 {
			return nil, "", fmt.Errorf("invalid sort %q: give an order by at least one column", name)
		}
		sort.By = slices.Clone(sort.By)
		if sort.Label == "" {
			sort.Label = name
		}
		seen := map[string]bool{}
		for at, order := range sort.By {
			column := strings.ToLower(order.Column)
			if !validIdentifier(order.Column) || seen[column] {
				return nil, "", fmt.Errorf("sort %q has an invalid or repeated column %q", name, order.Column)
			}
			seen[column] = true
			if order.Direction == "" {
				order.Direction = "asc"
			}
			order.Direction = strings.ToLower(order.Direction)
			if order.Direction != "asc" && order.Direction != "desc" {
				return nil, "", fmt.Errorf("sort %q has an invalid direction for %q", name, order.Column)
			}
			sort.By[at] = order
		}
		out[name] = sort
	}
	if fallback == "" {
		fallback = names[0]
	}
	if _, exists := out[fallback]; !exists {
		return nil, "", fmt.Errorf("unknown default sort %q", fallback)
	}
	return out, fallback, nil
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func digest(value any) string {
	data, _ := json.Marshal(value) // only normalized options and their scalars
	return fmt.Sprintf("%x", sha256.Sum256(data))[:32]
}
