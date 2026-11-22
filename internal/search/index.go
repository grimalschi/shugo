package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/blevesearch/snowballstem"
	"github.com/blevesearch/snowballstem/english"
	_ "modernc.org/sqlite"
)

const (
	indexVersion  = 1
	applicationID = 0x474f5352 // GOSR
)

type indexMetadata struct {
	ID      string   `json:"id"`
	Columns []string `json:"columns"`
	Primary string   `json:"primary"`
}

// Index is a reader of a published snapshot. Its file name must remain
// immutable while readers can open new connections.
type Index struct {
	db        *sql.DB
	meta      indexMetadata
	columns   map[string]string
	domainsMu sync.RWMutex
	domains   map[string]map[string][]string
}

func (i *Index) Close() error { return i.db.Close() }

func openFile(path string, params url.Values) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute), RawQuery: params.Encode()}
	return sql.Open("sqlite", name.String())
}

func columnName(at int) string { return "c" + strconv.Itoa(at) }

// Build projects a real data table read in one SQLite transaction. It
// replaces targetPath atomically after the private snapshot is complete.
func Build(ctx context.Context, sourcePath, targetPath string, definition Definition) error {
	if sourcePath == "" || targetPath == "" {
		return fmt.Errorf("a search index needs a source database and a destination")
	}
	normalized, err := Normalize(Declaration{Table: definition.Table, Options: definition.Options})
	if err != nil {
		return err
	}
	if normalized.ID != definition.ID || normalized.IndexID != definition.IndexID {
		return fmt.Errorf("search index build requires its normalized definition")
	}
	if err := distinctFiles(sourcePath, targetPath); err != nil {
		return err
	}
	dir := filepath.Dir(targetPath)
	stage, err := os.CreateTemp(dir, ".search-*.sqlite")
	if err != nil {
		return err
	}
	name := stage.Name()
	defer os.Remove(name)
	if err := stage.Close(); err != nil {
		return err
	}
	if err := writeIndex(ctx, sourcePath, name, normalized); err != nil {
		return fmt.Errorf("build search for %q: %w", normalized.Table, err)
	}
	file, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	folder, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer folder.Close()
	if err := os.Rename(name, targetPath); err != nil {
		return err
	}
	return folder.Sync()
}

func distinctFiles(source, target string) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	targetInfo, err := os.Stat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if os.SameFile(sourceInfo, targetInfo) {
		return fmt.Errorf("search index cannot replace its source database")
	}
	return nil
}

func writeIndex(ctx context.Context, sourcePath, destination string, d Definition) error {
	source, err := openFile(sourcePath, url.Values{"mode": {"ro"}, "_pragma": {"busy_timeout(10000)"}})
	if err != nil {
		return err
	}
	defer source.Close()
	input, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer input.Rollback()
	cols, primary, err := sourceColumns(ctx, input, d.Table)
	if err != nil {
		return err
	}
	for _, column := range d.Columns() {
		if _, exists := cols[foldIdentifier(column)]; !exists {
			return fmt.Errorf("the source table has no column %q", column)
		}
	}
	fields := uniqueSorted(append(d.Columns(), primary))
	meta := indexMetadata{ID: d.IndexID, Columns: fields, Primary: primary}

	index, err := openFile(destination, url.Values{
		"mode": {"rw"}, "_pragma": {"journal_mode(OFF)", "synchronous(OFF)"},
	})
	if err != nil {
		return err
	}
	defer index.Close()
	index.SetMaxOpenConns(1)
	output, err := index.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer output.Rollback()
	if err := createIndexTables(ctx, output, meta, len(d.Options.Text) != 0); err != nil {
		return err
	}
	if err := projectRows(ctx, input, output, d, cols, fields, primary); err != nil {
		return err
	}
	if err := input.Commit(); err != nil {
		return err
	}
	for _, column := range d.indexedColumns() {
		physical := columnName(slices.Index(fields, column))
		ddl := "CREATE INDEX records_" + physical + " ON records(" + physical + ")"
		if _, err := output.ExecContext(ctx, ddl); err != nil {
			return err
		}
	}
	if len(d.Options.Text) != 0 {
		if _, err := output.ExecContext(ctx, "INSERT INTO search_terms(search_terms) VALUES('optimize')"); err != nil {
			return err
		}
	}
	if _, err := output.ExecContext(ctx, "ANALYZE"); err != nil {
		return err
	}
	if err := output.Commit(); err != nil {
		return err
	}
	return index.Close()
}

func sourceColumns(ctx context.Context, tx *sql.Tx, table string) (map[string]string, string, error) {
	var actual string
	if err := tx.QueryRowContext(ctx,
		"SELECT name FROM sqlite_schema WHERE type='table' AND name = ? COLLATE NOCASE", table).Scan(&actual); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("no source table %q", table)
		}
		return nil, "", err
	}
	rows, err := tx.QueryContext(ctx, "SELECT name, pk FROM pragma_table_xinfo(?) WHERE hidden != 1 ORDER BY cid", actual)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	columns := map[string]string{}
	var key string
	for rows.Next() {
		var name string
		var order int
		if err := rows.Scan(&name, &order); err != nil {
			return nil, "", err
		}
		if order != 0 {
			if key != "" {
				return nil, "", fmt.Errorf("search needs a primary key of one column")
			}
			key = name
		}
		columns[foldIdentifier(name)] = name
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if key == "" {
		return nil, "", fmt.Errorf("search needs a primary key of one column")
	}
	return columns, key, nil
}

func createIndexTables(ctx context.Context, tx *sql.Tx, m indexMetadata, text bool) error {
	columns := make([]string, len(m.Columns))
	for at := range columns {
		columns[at] = columnName(at)
	}
	schema := fmt.Sprintf(`PRAGMA application_id=%d;
PRAGMA user_version=%d;
CREATE TABLE meta(json TEXT NOT NULL);
CREATE TABLE records(%s);`, applicationID, indexVersion, strings.Join(columns, ","))
	if text {
		schema += `CREATE VIRTUAL TABLE search_terms USING fts5(
	terms, content='', tokenize='unicode61 remove_diacritics 0'
);`
	}
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return err
	}
	info, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO meta VALUES(?)", string(info))
	return err
}

func projectRows(ctx context.Context, input, output *sql.Tx, d Definition, cols map[string]string, fields []string, primary string) error {
	sources := make([]string, 0, len(fields))
	atPrimary := slices.Index(fields, primary)
	for _, name := range fields {
		// Unary + preserves SQLite's native storage class and removes the
		// driver-side DATE/DATETIME conversion of a named column.
		sources = append(sources, "+"+quote(cols[foldIdentifier(name)]))
	}
	from := "SELECT " + strings.Join(sources, ",") + " FROM " + quote(d.Table)
	rows, err := input.QueryContext(ctx, from+" ORDER BY "+quote(primary)+" COLLATE BINARY")
	if err != nil {
		return err
	}
	defer rows.Close()
	insert, err := output.PrepareContext(ctx, "INSERT INTO records VALUES(?"+strings.Repeat(",?", len(fields)-1)+")")
	if err != nil {
		return err
	}
	defer insert.Close()

	positions := make([]int, len(d.Options.Text))
	for at, name := range d.Options.Text {
		positions[at] = slices.Index(fields, name)
	}
	var fts *sql.Stmt
	if len(positions) > 0 {
		fts, err = output.PrepareContext(ctx, "INSERT INTO search_terms(rowid, terms) VALUES(?,?)")
		if err != nil {
			return err
		}
		defer fts.Close()
	}
	stems := make(map[string]string)
	values, pointers := rowSlots(len(sources))
	var rowid int64
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		if values[atPrimary] == nil {
			return fmt.Errorf("search source primary key %q cannot be null", primary)
		}
		for at, value := range values {
			if _, blob := value.([]byte); blob {
				return fmt.Errorf("search column %q contains a blob at row %v", fields[at], values[atPrimary])
			}
		}
		if _, err := insert.ExecContext(ctx, values...); err != nil {
			return err
		}
		rowid++
		if fts != nil {
			var text []string
			for _, at := range positions {
				if values[at] != nil {
					text = append(text, fmt.Sprint(values[at]))
				}
			}
			words := stemWords(strings.Join(text, " "), stems)
			if _, err := fts.ExecContext(ctx, rowid, strings.Join(words, " ")); err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func foldIdentifier(name string) string {
	return strings.Map(func(letter rune) rune {
		if letter >= 'A' && letter <= 'Z' {
			return letter + ('a' - 'A')
		}
		return letter
	}, name)
}

func rowSlots(count int) (values, pointers []any) {
	values, pointers = make([]any, count), make([]any, count)
	for at := range pointers {
		pointers[at] = &values[at]
	}
	return values, pointers
}

func stemWords(text string, cache map[string]string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for at, word := range words {
		if stem, exists := cache[word]; exists {
			words[at] = stem
			continue
		}
		env := snowballstem.NewEnv(word)
		english.Stem(env)
		words[at] = env.Current()
		if cache != nil && len(cache) < 32768 {
			cache[word] = words[at]
		}
	}
	return words
}

// Open validates the index header and reads the column names of a generation.
func Open(path string) (_ *Index, err error) {
	db, err := openFile(path, url.Values{"mode": {"ro"}, "immutable": {"1"}})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	var id, version int
	if err := db.QueryRow("PRAGMA application_id").Scan(&id); err != nil {
		return nil, err
	}
	if id != applicationID {
		return nil, fmt.Errorf("not a Shugo search index")
	}
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if version != indexVersion {
		return nil, fmt.Errorf("unsupported search index version %d", version)
	}
	var raw string
	if err := db.QueryRow("SELECT json FROM meta").Scan(&raw); err != nil {
		return nil, err
	}
	var info indexMetadata
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return nil, err
	}
	index := &Index{db: db, meta: info, columns: map[string]string{}, domains: map[string]map[string][]string{}}
	for at, column := range info.Columns {
		index.columns[column] = columnName(at)
	}
	if info.ID == "" || index.columns[info.Primary] == "" || len(index.columns) != len(info.Columns) {
		return nil, fmt.Errorf("invalid search index metadata")
	}
	return index, nil
}
