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
	unique  bool // no two rows hold the same values in its columns
	ours    bool // Shugo made it
}

// serves tells whether the rows that hold given values in the columns of
// equal are found through the index, in the order of the columns of order:
// the index starts with the columns of equal, whichever way round they
// stand, and goes on with those of order. An index that lets no two rows
// hold the same values serves as soon as all of its columns are given: it
// finds one row, and one row has no order.
func (i index) serves(equal, order []string) bool {
	within := func(columns, of []string) bool {
		for _, column := range columns {
			if !slices.Contains(of, column) {
				return false
			}
		}
		return true
	}
	if i.unique && within(i.columns, equal) {
		return true
	}
	if len(i.columns) < len(equal)+len(order) {
		return false
	}
	return within(equal, i.columns[:len(equal)]) && slices.Equal(i.columns[len(equal):len(equal)+len(order)], order)
}

// Index makes sure that the rows of a table that hold given values in the
// columns of equal can be found, in the order of the columns of order,
// without reading the whole table. It makes an index unless one that is
// there already serves: the key of the table, a unique column, an index
// made by hand, one of Shugo's own. It tells the name of the index it made,
// or nothing.
//
// The project's code asks for the indexes its questions need, so the author
// of a project does not have to think of them; what it names is checked.
//
// An index of Shugo's that the new one starts with is dropped, and named in
// dropped: the longer one serves whoever the shorter one served.
func (d *DB) Index(table string, equal, order []string) (made string, dropped []string, err error) {
	t, found := d.tables[table]
	if !found {
		return "", nil, fmt.Errorf("the project's code asked for an index of the table %s, which the data database does not have", table)
	}
	columns := slices.Concat(equal, order)
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("the project's code asked for an index of the table %s over no columns", table)
	}
	for i, column := range columns {
		if _, known := t.Column(column); !known || slices.Contains(columns[:i], column) {
			return "", nil, fmt.Errorf("the project's code asked for an index of the table %s over %s, which cannot be", table, strings.Join(columns, ", "))
		}
	}
	if !t.listed {
		if t.indexes, err = d.indexes(t); err != nil {
			return "", nil, err
		}
		t.listed = true
	}
	for _, one := range t.indexes {
		if one.serves(equal, order) {
			return "", nil, nil
		}
	}
	made, dropped, err = d.make(t, columns)
	return made, dropped, err
}

// make makes an index of a table over columns that have none.
func (d *DB) make(t *Table, columns []string) (name string, dropped []string, err error) {
	table := t.Name
	tx, err := d.db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()
	// The name tells what the index is of. Two tables may come to the same
	// name; the later one gets a number.
	plain := reserved + table + "_" + strings.Join(columns, "_")
	name = plain
	for n := 2; ; n++ {
		var taken int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = ?", name).Scan(&taken); err != nil {
			return "", nil, err
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
		return "", nil, fmt.Errorf("cannot make an index of the table %s over %s: %w", table, strings.Join(columns, ", "), err)
	}
	var kept []index
	for _, one := range t.indexes {
		if one.ours && len(one.columns) < len(columns) && slices.Equal(one.columns, columns[:len(one.columns)]) {
			if _, err := tx.Exec("DROP INDEX " + quote(one.name)); err != nil {
				return "", nil, err
			}
			dropped = append(dropped, one.name)
			continue
		}
		kept = append(kept, one)
	}
	if err := tx.Commit(); err != nil {
		return "", nil, err
	}
	t.indexes = append(kept, index{name: name, columns: columns, ours: true})
	return name, dropped, nil
}

// indexes reads the indexes of a table that rows can be found by through
// their columns as they are: not one over a part of the rows, over an
// expression, or with another way of comparing text than the queries have.
func (d *DB) indexes(t *Table) ([]index, error) {
	rows, err := d.db.Query(`SELECT name, "unique", origin = 'pk' FROM pragma_index_list(?) WHERE partial = 0 ORDER BY seq`, t.Name)
	if err != nil {
		return nil, err
	}
	var listed []index
	keyed := false
	for rows.Next() {
		var one index
		var key bool
		if err := rows.Scan(&one.name, &one.unique, &key); err != nil {
			rows.Close()
			return nil, err
		}
		one.ours = strings.HasPrefix(one.name, reserved)
		listed = append(listed, one)
		keyed = keyed || key
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var out []index
	if !keyed {
		// A key of whole numbers has no index: the rows of the table
		// themselves lie in its order.
		out = append(out, index{columns: []string{t.Columns[t.Key]}, unique: true})
	}
	for _, one := range listed {
		plain := true
		parts, err := d.db.Query(`SELECT coalesce(name, ''), coalesce(coll, '') FROM pragma_index_xinfo(?) WHERE key = 1 ORDER BY seqno`, one.name)
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
