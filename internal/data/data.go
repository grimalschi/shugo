// Package data is the database a project keeps its data in: one SQLite file
// that pages read through Shugo.query.
//
// The database keeps a journal of itself. Triggers that this package puts on
// every table note each row that is added, changed or removed, with what the
// row was before. An update reads from the journal which rows are not what
// they were at the previous update, so it never has to load all the data and
// compare it. Whoever writes to the database, with whatever tool, cannot
// forget to say what they changed.
package data

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

// What the package keeps in the database beside the tables of the project.
// A name that starts like these is not a table of the project.
const (
	reserved = "_shugo_"
	journal  = reserved + "journal"
	settings = reserved + "meta"
)

// The journal holds one line for every row touched: where the row is and
// what it was, as a JSON list of its values, or nothing if it was not there.
// The numbers of the lines only grow, also after old lines are dropped.
const layout = `
CREATE TABLE IF NOT EXISTS ` + journal + ` (seq INTEGER PRIMARY KEY AUTOINCREMENT, tbl TEXT NOT NULL, id, old TEXT);
CREATE TABLE IF NOT EXISTS ` + settings + ` (name TEXT PRIMARY KEY, value) WITHOUT ROWID;
`

// DB is the data database of a project.
type DB struct {
	db *sql.DB
	// ID names the journal. Another database has another name, and so has
	// this one after its tables were changed: what its journal said before
	// no longer tells everything that differs.
	ID     string
	tables map[string]*Table
}

// Table is one table of the database.
type Table struct {
	Name    string
	Columns []string
	Key     int // which column is the primary key
	// unique are the columns of every unique index beside the key.
	unique [][]int
	// indexes are the indexes rows of the table can be found by, once they
	// were asked about: see Index.
	indexes []index
	listed  bool
}

// Column returns the place of a column among the columns of the table.
func (t *Table) Column(name string) (int, bool) {
	for i, column := range t.Columns {
		if column == name {
			return i, true
		}
	}
	return 0, false
}

// Open opens the data database at path and makes sure that it keeps its
// journal. It writes to the file: the journal and its triggers live there.
func Open(path string) (*DB, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("the data database %s is not there: shugo.json names it, and something has to fill it before the site is built", path)
	}
	// Readers of a database in WAL mode do not hold up whoever writes to it.
	options := url.Values{"mode": {"rw"}, "_txlock": {"immediate"}, "_pragma": {"busy_timeout(10000)", "journal_mode(WAL)", "synchronous(NORMAL)"}}
	db, err := sql.Open("sqlite", "file:"+path+"?"+options.Encode())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	d := &DB{db: db, tables: map[string]*Table{}}
	if err := d.prepare(); err != nil {
		db.Close()
		return nil, fmt.Errorf("the data database %s: %w", path, err)
	}
	return d, nil
}

// Close closes the database.
func (d *DB) Close() error { return d.db.Close() }

// Table returns a table of the project by its name.
func (d *DB) Table(name string) (*Table, bool) {
	t, found := d.tables[name]
	return t, found
}

// journaled tells what the journal has to be for the tables as they are
// now: the triggers it needs and a name for the set of them. inPlace tells
// that it is so already: it was made for these very tables, and all of its
// triggers are still there as they were made.
func journaled(tx *sql.Tx) (tables []*Table, wanted []string, signature string, inPlace bool, err error) {
	if tables, err = readTables(tx); err != nil {
		return nil, nil, "", false, err
	}
	for _, t := range tables {
		wanted = append(wanted, t.triggers()...)
	}
	sum := sha256.Sum256([]byte(strings.Join(wanted, "\n")))
	signature = hex.EncodeToString(sum[:16])
	kept, err := readTriggers(tx)
	if err != nil {
		return nil, nil, "", false, err
	}
	inPlace = len(kept) == len(wanted)
	for _, statement := range wanted {
		inPlace = inPlace && kept[statement]
	}
	var made string
	if err := tx.QueryRow("SELECT value FROM " + settings + " WHERE name = 'tables'").Scan(&made); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, "", false, err
	}
	return tables, wanted, signature, inPlace && made == signature, nil
}

// prepare reads the tables and puts the journal in place where it is not
// there yet or was made for other tables.
func (d *DB) prepare() error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(layout); err != nil {
		return err
	}
	tables, wanted, signature, inPlace, err := journaled(tx)
	if err != nil {
		return err
	}
	for _, t := range tables {
		d.tables[t.Name] = t
	}
	if inPlace {
		if err := tx.QueryRow("SELECT value FROM " + settings + " WHERE name = 'id'").Scan(&d.ID); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Rows may have changed while a table had no triggers, so the journal
	// starts under a new name: nobody can rely on what it said before.
	rows, err := tx.Query("SELECT name FROM sqlite_master WHERE type = 'trigger' AND name LIKE ? ESCAPE '\\'", like(reserved))
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		stale = append(stale, name)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, name := range stale {
		if _, err := tx.Exec("DROP TRIGGER " + quote(name)); err != nil {
			return err
		}
	}
	for _, statement := range wanted {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("cannot make the journal keep track of a table: %w", err)
		}
	}
	name := make([]byte, 8)
	if _, err := rand.Read(name); err != nil {
		return err
	}
	d.ID = hex.EncodeToString(name)
	const put = "INSERT INTO " + settings + " (name, value) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"
	if _, err := tx.Exec(put, "tables", signature); err != nil {
		return err
	}
	if _, err := tx.Exec(put, "id", d.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// like makes a pattern that matches names starting with a prefix.
func like(prefix string) string {
	return strings.NewReplacer(`\`, `\\`, `_`, `\_`, `%`, `\%`).Replace(prefix) + "%"
}

// readTables finds the tables of the project and what the journal needs to
// know of each.
func readTables(tx *sql.Tx) ([]*Table, error) {
	names, err := list(tx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' AND name NOT LIKE ? ESCAPE '\\' ORDER BY name", like(reserved))
	if err != nil {
		return nil, err
	}
	var tables []*Table
	for _, name := range names {
		t := &Table{Name: name, Key: -1}
		rows, err := tx.Query("SELECT name, pk FROM pragma_table_info(?) ORDER BY cid", name)
		if err != nil {
			return nil, err
		}
		keys := 0
		for rows.Next() {
			var column string
			var key int
			if err := rows.Scan(&column, &key); err != nil {
				return nil, err
			}
			if key > 0 {
				t.Key = len(t.Columns)
				keys++
			}
			t.Columns = append(t.Columns, column)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if keys != 1 {
			return nil, fmt.Errorf("the table %s needs a primary key of one column: by it Shugo tells the rows of a table apart", name)
		}
		indexes, err := list(tx, `SELECT name FROM pragma_index_list(?) WHERE "unique" = 1 AND origin != 'pk' ORDER BY name`, name)
		if err != nil {
			return nil, err
		}
		for _, index := range indexes {
			columns, err := list(tx, "SELECT coalesce(name, '') FROM pragma_index_info(?) ORDER BY seqno", index)
			if err != nil {
				return nil, err
			}
			var places []int
			for _, column := range columns {
				// An index over an expression has no column to compare.
				if at, found := t.Column(column); found {
					places = append(places, at)
				}
			}
			if len(places) == len(columns) && len(places) > 0 {
				t.unique = append(t.unique, places)
			}
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func list(tx *sql.Tx, query string, arguments ...any) ([]string, error) {
	rows, err := tx.Query(query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

// readTriggers returns the statements that made the triggers of the journal.
func readTriggers(tx *sql.Tx) (map[string]bool, error) {
	statements, err := list(tx, "SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name LIKE ? ESCAPE '\\'", like(reserved))
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, statement := range statements {
		out[statement] = true
	}
	return out, nil
}

// quote writes a name the way SQL takes any name.
func quote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// row writes the expression that gives a row as a JSON list of its values.
// JSON has no place for a blob, so a blob is written as text that no other
// value is likely to be: a control character and its bytes in hexadecimal.
func (t *Table) row(from string) string {
	values := make([]string, len(t.Columns))
	for i, column := range t.Columns {
		value := from + quote(column)
		values[i] = "CASE WHEN typeof(" + value + ") = 'blob' THEN char(1) || hex(" + value + ") ELSE " + value + " END"
	}
	return "json_array(" + strings.Join(values, ", ") + ")"
}

// triggers writes the statements that make the journal keep track of the
// table. Each row touched leaves a line with what it was before:
//
//   - a row that is added leaves a line that says it was not there;
//   - a row that is changed or removed leaves what it was;
//   - a row that another one is about to replace (INSERT OR REPLACE and its
//     kin remove such rows without a word) leaves what it was, too.
func (t *Table) triggers() []string {
	table, key := quote(t.Name), quote(t.Columns[t.Key])
	name := func(kind string) string { return quote(reserved + t.Name + "_" + kind) }
	note := func(id, old string) string {
		return "INSERT INTO " + journal + " (tbl, id, old) VALUES (" + text(t.Name) + ", " + id + ", " + old + ");"
	}

	// The rows that a new or changed row would push out: the one with its
	// key, and every one that holds its values of a unique index.
	rivals := []string{key + " IS NEW." + key}
	for _, columns := range t.unique {
		var same []string
		for _, at := range columns {
			same = append(same, quote(t.Columns[at])+" IS NEW."+quote(t.Columns[at]))
		}
		rivals = append(rivals, strings.Join(same, " AND "))
	}
	pushed := "INSERT INTO " + journal + " (tbl, id, old) SELECT " + text(t.Name) + ", " + key + ", " + t.row("") +
		" FROM " + table + " WHERE ((" + strings.Join(rivals, ") OR (") + "))"

	var differs []string
	for _, column := range t.Columns {
		differs = append(differs, "OLD."+quote(column)+" IS NOT NEW."+quote(column))
	}
	return []string{
		"CREATE TRIGGER " + name("before") + " BEFORE INSERT ON " + table + " BEGIN " + pushed + "; END",
		"CREATE TRIGGER " + name("replace") + " BEFORE UPDATE ON " + table + " BEGIN " + pushed + " AND " + key + " IS NOT OLD." + key + "; END",
		"CREATE TRIGGER " + name("insert") + " AFTER INSERT ON " + table + " BEGIN " + note("NEW."+key, "NULL") + " END",
		"CREATE TRIGGER " + name("update") + " AFTER UPDATE ON " + table + " WHEN " + strings.Join(differs, " OR ") + " BEGIN " +
			note("OLD."+key, t.row("OLD.")) +
			// A row that got another key is a row removed and a row added.
			" INSERT INTO " + journal + " (tbl, id, old) SELECT " + text(t.Name) + ", NEW." + key + ", NULL WHERE NEW." + key + " IS NOT OLD." + key + "; END",
		"CREATE TRIGGER " + name("delete") + " AFTER DELETE ON " + table + " BEGIN " + note("OLD."+key, t.row("OLD.")) + " END",
	}
}

// text writes a text as SQL takes it.
func text(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
