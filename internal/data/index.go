package data

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// index is an index of a table: the columns rows are found by through it,
// in their order.
type index struct {
	name    string
	columns []string
	ours    bool // Shugo made it
}

// Index makes sure that the rows of a table can be found by the given
// columns, in their order, without reading the whole table. It makes an
// index unless one that is there already starts with these columns: the key
// of the table, a unique column, an index made by hand, one of Shugo's own.
// It tells the name of the index it made, or nothing.
//
// The project's code asks for the indexes its questions need, so the author
// of a project does not have to think of them; what it names is checked.
//
// An index of Shugo's that the new one starts with is dropped: the longer
// one serves whoever the shorter one served.
func (d *DB) Index(table string, columns []string) (made string, err error) {
	t, found := d.tables[table]
	if !found {
		return "", fmt.Errorf("the project's code asked for an index of the table %s, which the data database does not have", table)
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("the project's code asked for an index of the table %s over no columns", table)
	}
	for i, column := range columns {
		if _, known := t.Column(column); !known || slices.Contains(columns[:i], column) {
			return "", fmt.Errorf("the project's code asked for an index of the table %s over %s, which cannot be", table, strings.Join(columns, ", "))
		}
	}
	if !t.listed {
		if t.indexes, err = d.indexes(t); err != nil {
			return "", err
		}
		t.listed = true
	}
	for _, one := range t.indexes {
		if len(one.columns) >= len(columns) && slices.Equal(one.columns[:len(columns)], columns) {
			return "", nil
		}
	}

	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// The name tells what the index is of. Two tables may come to the same
	// name; the later one gets a number.
	plain := reserved + table + "_" + strings.Join(columns, "_")
	name := plain
	for n := 2; ; n++ {
		var taken int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = ?", name).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			break
		}
		name = plain + "_" + strconv.Itoa(n)
	}
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quote(column)
	}
	if _, err := tx.Exec("CREATE INDEX " + quote(name) + " ON " + quote(t.Name) + " (" + strings.Join(quoted, ", ") + ")"); err != nil {
		return "", fmt.Errorf("cannot make an index of the table %s over %s: %w", table, strings.Join(columns, ", "), err)
	}
	kept := t.indexes[:0]
	for _, one := range t.indexes {
		if one.ours && len(one.columns) < len(columns) && slices.Equal(one.columns, columns[:len(one.columns)]) {
			if _, err := tx.Exec("DROP INDEX " + quote(one.name)); err != nil {
				return "", err
			}
			continue
		}
		kept = append(kept, one)
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	t.indexes = append(kept, index{name, columns, true})
	return name, nil
}

// indexes reads the indexes of a table that rows can be found by through
// their columns as they are: not one over a part of the rows, over an
// expression, or with another way of comparing text than the queries have.
func (d *DB) indexes(t *Table) ([]index, error) {
	rows, err := d.db.Query(`SELECT name FROM pragma_index_list(?) WHERE partial = 0 ORDER BY seq`, t.Name)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var out []index
	for _, name := range names {
		one := index{name: name, ours: strings.HasPrefix(name, reserved)}
		plain := true
		parts, err := d.db.Query(`SELECT coalesce(name, ''), coalesce(coll, '') FROM pragma_index_xinfo(?) WHERE key = 1 ORDER BY seqno`, name)
		if err != nil {
			return nil, err
		}
		for parts.Next() {
			var column, collation string
			if err := parts.Scan(&column, &collation); err != nil {
				parts.Close()
				return nil, err
			}
			plain = plain && column != "" && collation == "BINARY"
			one.columns = append(one.columns, column)
		}
		if err := parts.Close(); err != nil {
			return nil, err
		}
		if plain {
			out = append(out, one)
		}
	}
	return out, nil
}
