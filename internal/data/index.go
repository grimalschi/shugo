package data

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// index is an index of a table.
type index struct {
	name string
	// columns are the columns rows are found by through it, in their order.
	columns []string
	// order is what rows come in from it: its columns, and then the key
	// where the table keeps its rows by the key.
	order  []string
	unique bool // no two rows hold the same values in its columns
}

// serves tells whether the rows that hold given values in the columns of
// equal are found through the index: it starts with these columns,
// whichever way round they stand.
//
// Rows that are taken, not counted, come in the order of the columns of
// order. For that the index has to go on with these columns and end there:
// one that goes on would give the rows in the order of its further columns,
// and SQLite would have to sort them, reading every row it finds — also the
// ones a limit leaves out.
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
	if len(i.order) < wanted || (rows && len(i.order) > wanted) {
		return false
	}
	return within(equal, i.order[:len(equal)]) && slices.Equal(i.order[len(equal):wanted], order)
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
		if t.indexes, t.byKey, err = d.indexes(t); err != nil {
			return "", err
		}
		t.listed = true
	}
	for _, one := range t.indexes {
		if one.serves(equal, order, rows) {
			return "", nil
		}
	}

	// Where the table keeps its rows by the key, an index goes on with the
	// key by itself: it is not named among its columns.
	one := index{columns: columns, order: columns}
	if key := t.Columns[t.Key]; t.byKey {
		if columns[len(columns)-1] == key {
			one.columns = columns[:len(columns)-1]
		} else if !slices.Contains(columns, key) {
			one.order = append(slices.Clone(columns), key)
		}
	}
	if len(one.columns) == 0 {
		// The order of the key alone is the table itself.
		return "", nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// The name tells what the index is of. Two tables may come to the same
	// name; the later one gets a number.
	plain := reserved + table + "_" + strings.Join(one.columns, "_")
	one.name = plain
	for n := 2; ; n++ {
		var taken int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = ?", one.name).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			break
		}
		one.name = plain + "_" + strconv.Itoa(n)
	}
	// The queries compare values exactly, whatever way of comparing text
	// the column was declared with; so does the index.
	quoted := make([]string, len(one.columns))
	for i, column := range one.columns {
		quoted[i] = quote(column) + " COLLATE BINARY"
	}
	if _, err := tx.Exec("CREATE INDEX " + quote(one.name) + " ON " + quote(t.Name) + " (" + strings.Join(quoted, ", ") + ")"); err != nil {
		return "", fmt.Errorf("cannot make an index of the table %s over %s: %w", table, strings.Join(one.columns, ", "), err)
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	t.indexes = append(t.indexes, one)
	return one.name, nil
}

// indexes reads the indexes of a table that rows can be found by through
// their columns as they are: not one over a part of the rows, over an
// expression, or with another way of comparing text than the queries have.
//
// byKey tells that the table keeps its rows by its key: the key is a whole
// number, or the table was made WITHOUT ROWID. Rows that hold the same
// values in the columns of an index then follow each other in the order of
// their keys. Otherwise they follow each other as they happen to lie in
// the table, which tells nothing: a row that is removed and put back as it
// was lies elsewhere.
func (d *DB) indexes(t *Table) (out []index, byKey bool, err error) {
	if err := d.db.QueryRow(`SELECT wr FROM pragma_table_list WHERE schema = 'main' AND name = ?`, t.Name).Scan(&byKey); err != nil {
		return nil, false, err
	}
	rows, err := d.db.Query(`SELECT name, "unique", origin = 'pk' FROM pragma_index_list(?) WHERE partial = 0 ORDER BY seq`, t.Name)
	if err != nil {
		return nil, false, err
	}
	var listed []index
	keyed := false
	for rows.Next() {
		var one index
		var key bool
		if err := rows.Scan(&one.name, &one.unique, &key); err != nil {
			rows.Close()
			return nil, false, err
		}
		listed = append(listed, one)
		keyed = keyed || key
	}
	if err := rows.Close(); err != nil {
		return nil, false, err
	}
	key := t.Columns[t.Key]
	if !keyed {
		// A key of whole numbers has no index: the rows of the table
		// themselves lie in its order.
		byKey = true
		out = append(out, index{columns: []string{key}, order: []string{key}, unique: true})
	}
	for _, one := range listed {
		plain := true
		parts, err := d.db.Query(`SELECT coalesce(name, ''), coalesce(coll, '') FROM pragma_index_xinfo(?) WHERE key = 1 ORDER BY seqno`, one.name)
		if err != nil {
			return nil, false, err
		}
		for parts.Next() {
			var column, collation string
			if err := parts.Scan(&column, &collation); err != nil {
				parts.Close()
				return nil, false, err
			}
			plain = plain && column != "" && collation == "BINARY"
			one.columns = append(one.columns, column)
		}
		if err := parts.Close(); err != nil {
			return nil, false, err
		}
		if !plain {
			continue
		}
		one.order = one.columns
		if byKey && !slices.Contains(one.columns, key) {
			one.order = append(slices.Clone(one.columns), key)
		}
		out = append(out, one)
	}
	return out, byKey, nil
}
