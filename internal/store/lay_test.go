package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// A store laid out in another way is emptied: its tables go, whatever they
// were, and the tables of this layout take their place.
func TestOtherLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE TABLE meta (name TEXT PRIMARY KEY, value BLOB NOT NULL)",
		"INSERT INTO meta VALUES ('layout', 'old')",
		`CREATE TABLE "an old ""table""" (x)`,
		"CREATE TABLE page (path TEXT PRIMARY KEY)",
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	err = st.View(func(v *View) error {
		var tables int
		if err := v.db.QueryRowContext(t.Context(), `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'an old "table"'`).Scan(&tables); err != nil {
			return err
		}
		if tables != 0 {
			t.Error("the table of the other layout is still there")
		}
		mark, err := v.Mark()
		if mark.Store == "" {
			t.Error("the store has no name")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
