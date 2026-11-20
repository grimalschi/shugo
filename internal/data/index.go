package data

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// index is an index of a table: the columns rows are found by through it,
// in their order. Rows that hold the same values in all of them follow each
// other as they lie in the table.
type index struct {
	name    string
	columns []string
	unique  bool // no two rows hold the same values in its columns
}

// serves tells whether the rows that hold given values in the columns of
// equal are found through the index: it starts with these columns,
// whichever way round they stand.
//
// Rows that are taken, not counted, come in the order of the columns of
// order and then as they lie in the table. For that the index has to go on
// with the columns of order and end there: one that goes on would give the
// rows in the order of its further columns, and SQLite would have to sort
// them, reading every row it finds — also the ones a limit leaves out.
//
// An index that lets no two rows hold the same values serves as soon as all
// of its columns are given: it finds one row, and one row has no order.
func (i index) serves(equal, order []string, rows bool) bool {
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
	wanted := len(equal) + len(order)
	if len(i.columns) < wanted || (rows && len(i.columns) > wanted) {
		return false
	}
	return within(equal, i.columns[:len(equal)]) && slices.Equal(i.columns[len(equal):wanted], order)
}

// Index makes sure that the rows of a table that hold given values in the
// columns of equal can be found without reading the whole table: to be
// counted, or, if rows is set, to be taken in the order of the columns of
// order. It makes an index unless one that is there already serves: the key
// of the table, a unique column, an index made by hand, one of Shugo's own.
// It tells the name of the index it made, or nothing.
//
// The project's code asks for the indexes its questions need, so the author
// of a project does not have to think of them; what it names is checked.
func (d *DB) Index(table string, equal, order []string, rows bool) (made string, err error) {
	t, found := d.tables[table]
	if !found {
		return "", fmt.Errorf("the project's code asked for an index of the table %s, which the data database does not have", table)
	}
	columns := slices.Concat(equal, order)
	if len(columns) == 0 {
		return "", fmt.Errorf("the project's code asked for an index of the table %s over no columns", table)
	}
	for i, column := range columns {
		if _, known := t.Column(column); !known || slices.Contains(columns[:i], column) {
			return "", fmt.Errorf("the project's code asked for an index of the table %s over %s, which cannot be", table, strings.Join(columns, ", "))
		}
	}
	if !rows && len(order) > 0 {
		return "", fmt.Errorf("the project's code asked for an index of the table %s to count rows in an order, which cannot be", table)
	}
	if !t.listed {
		if t.indexes, err = d.indexes(t); err != nil {
			return "", err
		}
		t.listed = true
	}
	for _, one := range t.indexes {
		if one.serves(equal, order, rows) {
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
	made = plain
	for n := 2; ; n++ {
		var taken int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = ?", made).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			break
		}
		made = plain + "_" + strconv.Itoa(n)
	}
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quote(column)
	}
	if _, err := tx.Exec("CREATE INDEX " + quote(made) + " ON " + quote(t.Name) + " (" + strings.Join(quoted, ", ") + ")"); err != nil {
		return "", fmt.Errorf("cannot make an index of the table %s over %s: %w", table, strings.Join(columns, ", "), err)
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	t.indexes = append(t.indexes, index{name: made, columns: columns})
	return made, nil
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
