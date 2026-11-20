package data

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Change is one row that is not what it was when the journal was read last:
// added (Old is nil), removed (New is nil), or holding other values. The
// values stand in the order of the columns of the table; each is nil, an
// int64, a float64 or a string.
type Change struct {
	Table    *Table
	Old, New []any
}

// Differs tells whether the value of a column is not what it was. In a row
// that was added or removed every value is.
func (c *Change) Differs(column int) bool {
	return c.Old == nil || c.New == nil || c.Old[column] != c.New[column]
}

// Position returns where the journal stands: the number of its latest line.
// What is read from the database after this moment holds every change up to
// that line at the least.
func (d *DB) Position() (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	return position(tx)
}

func position(tx *sql.Tx) (at int64, err error) {
	// The journal may be empty: its lines are dropped once they are read.
	err = tx.QueryRow("SELECT seq FROM sqlite_sequence WHERE name = ?", journal).Scan(&at)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return at, err
}

// Changes returns the rows that are not what they were when the journal
// stood at after, and where it stands now. A row that was touched and holds
// what it held is not among them.
//
// lost tells that the journal cannot say what has changed since then: lines
// after that position were dropped on behalf of someone else, or the
// position is not of this journal. Then everything may have changed.
//
// The lines up to after are dropped: whoever asks has dealt with them.
func (d *DB) Changes(after int64) (changes []Change, now int64, lost bool, err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, 0, false, err
	}
	defer tx.Rollback()
	if now, err = position(tx); err != nil {
		return nil, 0, false, err
	}
	var dropped int64
	if err := tx.QueryRow("SELECT value FROM " + settings + " WHERE name = 'dropped'").Scan(&dropped); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, false, err
	}
	if after < dropped || after > now {
		return nil, now, true, nil
	}

	// Of every row only its first line counts: it tells what the row was
	// before all that happened to it since.
	type touched struct {
		table *Table
		id    any
		old   sql.NullString
	}
	var rows []touched
	seen := map[string]struct{}{}
	lines, err := tx.Query("SELECT tbl, id, old FROM "+journal+" WHERE seq > ? ORDER BY seq", after)
	if err != nil {
		return nil, 0, false, err
	}
	for lines.Next() {
		var name string
		var one touched
		if err := lines.Scan(&name, &one.id, &one.old); err != nil {
			lines.Close()
			return nil, 0, false, err
		}
		if one.table = d.tables[name]; one.table == nil {
			continue // of a table that is gone: its journal would have another name
		}
		row := name + "\x00" + string(Key([]any{plain(one.id)}))
		if _, already := seen[row]; !already {
			seen[row] = struct{}{}
			rows = append(rows, one)
		}
	}
	if err := lines.Close(); err != nil {
		return nil, 0, false, err
	}

	current := map[*Table]*sql.Stmt{}
	for _, one := range rows {
		read := current[one.table]
		if read == nil {
			t := one.table
			if read, err = tx.Prepare("SELECT " + t.row("") + " FROM " + quote(t.Name) + " WHERE " + quote(t.Columns[t.Key]) + " = ?"); err != nil {
				return nil, 0, false, err
			}
			current[one.table] = read
		}
		change := Change{Table: one.table}
		if one.old.Valid {
			if change.Old, err = values(one.old.String, len(one.table.Columns)); err != nil {
				return nil, 0, false, fmt.Errorf("the journal of the table %s: %w", one.table.Name, err)
			}
		}
		var kept sql.NullString
		if err := read.QueryRow(one.id).Scan(&kept); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, false, err
		}
		if kept.Valid {
			if change.New, err = values(kept.String, len(one.table.Columns)); err != nil {
				return nil, 0, false, err
			}
		}
		if change.Old == nil && change.New == nil {
			continue // added and removed since
		}
		same := change.Old != nil && change.New != nil
		for at := 0; same && at < len(change.Old); at++ {
			same = change.Old[at] == change.New[at]
		}
		if !same {
			changes = append(changes, change)
		}
	}

	if after > dropped {
		if _, err := tx.Exec("DELETE FROM "+journal+" WHERE seq <= ?", after); err != nil {
			return nil, 0, false, err
		}
		if _, err := tx.Exec("INSERT INTO "+settings+" (name, value) VALUES ('dropped', ?) ON CONFLICT (name) DO UPDATE SET value = excluded.value", after); err != nil {
			return nil, 0, false, err
		}
	}
	return changes, now, false, tx.Commit()
}

// values reads a row that the database wrote as a JSON list.
func values(text string, columns int) ([]any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var row []any
	if err := decoder.Decode(&row); err != nil || len(row) != columns {
		return nil, fmt.Errorf("a row of %d values is written as %.80s", columns, text)
	}
	for i, value := range row {
		row[i] = plain(value)
	}
	return row, nil
}

// plain gives a value one of the forms that values are compared in: nil,
// int64, float64 or string. SQLite finds 5 and 5.0 equal, so a whole
// number is an int64 however it was written.
func plain(value any) any {
	switch v := value.(type) {
	case nil, int64, string:
		return v
	case json.Number:
		if whole, err := v.Int64(); err == nil {
			return whole
		}
		real, _ := v.Float64()
		return plain(real)
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < 1<<62 {
			return int64(v)
		}
		return v
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	case []byte:
		return "\x01" + strings.ToUpper(fmt.Sprintf("%x", v))
	default:
		return fmt.Sprint(v)
	}
}

// Key writes a list of values as bytes, so that two lists are written alike
// exactly when SQLite finds each value of one equal to the value of the
// other at its place. Values come as plain gives them, or as JSON does. A
// list of no values is a byte of its own: no bytes at all would be taken
// for no key.
func Key(list []any) []byte {
	var out bytes.Buffer
	if len(list) == 0 {
		out.WriteByte(4)
	}
	for _, value := range list {
		switch v := plain(value).(type) {
		case nil:
			out.WriteByte(0)
		case int64:
			out.WriteByte(1)
			out.Write(binary.BigEndian.AppendUint64(nil, uint64(v)))
		case float64:
			out.WriteByte(2)
			out.Write(binary.BigEndian.AppendUint64(nil, math.Float64bits(v)))
		case string:
			out.WriteByte(3)
			out.Write(binary.AppendUvarint(nil, uint64(len(v))))
			out.WriteString(v)
		}
	}
	return out.Bytes()
}
