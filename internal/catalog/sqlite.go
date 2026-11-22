package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const catalogApplicationID = 0x474f4354 // GOCT

const schema = `
CREATE TABLE category (
	id TEXT PRIMARY KEY NOT NULL,
	title TEXT NOT NULL,
	position INTEGER NOT NULL
);
CREATE TABLE size (
	id TEXT PRIMARY KEY NOT NULL,
	position INTEGER NOT NULL
);
CREATE TABLE product (
	rowid INTEGER PRIMARY KEY,
	id TEXT NOT NULL UNIQUE,
	slug TEXT NOT NULL,
	title TEXT NOT NULL,
	category TEXT NOT NULL REFERENCES category(id),
	price INTEGER NOT NULL,
	original_price INTEGER NOT NULL,
	available INTEGER NOT NULL CHECK(available IN (0, 1)),
	arrival TEXT NOT NULL,
	position INTEGER NOT NULL
);
CREATE TABLE product_size (
	product INTEGER NOT NULL REFERENCES product(rowid),
	size TEXT NOT NULL REFERENCES size(id),
	position INTEGER NOT NULL,
	PRIMARY KEY(product, size)
) WITHOUT ROWID;
CREATE VIRTUAL TABLE product_text USING fts5(
	terms, content='', contentless_delete=1, tokenize='unicode61 remove_diacritics 0'
);`

const indexes = `
CREATE INDEX product_size_by_size ON product_size(size, product);
CREATE INDEX product_by_position ON product(position, id);
CREATE INDEX product_by_price ON product(price, position, id);
CREATE INDEX product_by_price_desc ON product(price DESC, position, id);
CREATE INDEX product_by_arrival ON product(arrival DESC, position, id);
CREATE INDEX product_by_category ON product(category, available, price, position, id);
CREATE INDEX product_by_stock ON product(available, price, category);
INSERT INTO product_text(product_text) VALUES('optimize');
ANALYZE;`

func openFile(name string, settings url.Values) (*sql.DB, error) {
	path, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	address := url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: settings.Encode()}
	return sql.Open("sqlite", address.String())
}

func openIndex(ctx context.Context, name string, readers int) (*Index, error) {
	db, err := openFile(name, url.Values{"mode": {"ro"}, "immutable": {"1"}})
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(readers)
	db.SetMaxIdleConns(min(readers, 2))
	return loadIndex(ctx, db)
}

func loadIndex(ctx context.Context, db *sql.DB) (_ *Index, err error) {
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	var applicationID, version int
	if err := db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&applicationID); err != nil {
		return nil, err
	}
	if applicationID != catalogApplicationID {
		return nil, fmt.Errorf("not a Shugo catalog index (application ID %x)", applicationID)
	}
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if version != indexVersion {
		return nil, fmt.Errorf("unsupported catalog index version %d", version)
	}
	index := &Index{db: db}
	index.categories, err = readOptions(ctx, db, "SELECT id, title FROM category ORDER BY position, id")
	if err != nil {
		return nil, err
	}
	index.sizes, err = readOptions(ctx, db, "SELECT id, id FROM size ORDER BY position, id")
	if err != nil {
		return nil, err
	}
	// Validate the tables and columns used by searches without reading all
	// products at startup.
	rows, err := db.QueryContext(ctx, `
SELECT p.rowid, p.id, p.slug, p.title, p.category, p.price, p.original_price,
       p.available, p.arrival, p.position, ps.product, ps.size, ps.position, ft.rowid
FROM product AS p, product_size AS ps, product_text AS ft LIMIT 0`)
	if err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return index, nil
}

func readOptions(ctx context.Context, db *sql.DB, query string) ([]option, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var options []option
	for rows.Next() {
		var opt option
		if err := rows.Scan(&opt.Value, &opt.Label); err != nil {
			return nil, err
		}
		options = append(options, opt)
	}
	return options, rows.Err()
}

func writeIndex(ctx context.Context, db *sql.DB, data snapshot) error {
	if data.Version != indexVersion {
		return fmt.Errorf("unsupported catalog index version %d", data.Version)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	header := fmt.Sprintf("PRAGMA application_id=%d; PRAGMA user_version=%d;", catalogApplicationID, indexVersion)
	if _, err := tx.ExecContext(ctx, header+schema); err != nil {
		return err
	}
	for position, item := range data.Categories {
		if _, err := tx.ExecContext(ctx, "INSERT INTO category VALUES (?, ?, ?)", item.Value, item.Label, position); err != nil {
			return err
		}
	}
	for position, size := range sizeNames(data.Products) {
		if _, err := tx.ExecContext(ctx, "INSERT INTO size VALUES (?, ?)", size, position); err != nil {
			return err
		}
	}
	if err := writeProducts(ctx, tx, data.Products); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, indexes); err != nil {
		return err
	}
	return tx.Commit()
}

func writeProducts(ctx context.Context, tx *sql.Tx, products []entry) error {
	cards, err := tx.PrepareContext(ctx, "INSERT INTO product VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer cards.Close()
	sizes, err := tx.PrepareContext(ctx, "INSERT INTO product_size VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer sizes.Close()
	terms, err := tx.PrepareContext(ctx, "INSERT INTO product_text(rowid, terms) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer terms.Close()

	stems := make(map[string]string)
	for at, p := range products {
		row := at + 1
		if _, err := cards.ExecContext(ctx, row, p.ID, p.Slug, p.Title, p.Category,
			p.Price, p.OriginalPrice, p.Available, p.Arrival, p.Position); err != nil {
			return fmt.Errorf("insert catalog product %q: %w", p.ID, err)
		}
		for position, size := range p.Sizes {
			if _, err := sizes.ExecContext(ctx, row, size, position); err != nil {
				return fmt.Errorf("insert variant size for catalog product %q: %w", p.ID, err)
			}
		}
		if _, err := terms.ExecContext(ctx, row, strings.Join(stemWords(p.Text, stems), " ")); err != nil {
			return fmt.Errorf("index catalog product %q: %w", p.ID, err)
		}
	}
	return nil
}

func publish(ctx context.Context, name string, data snapshot) (*Index, error) {
	return publishSQLite(ctx, name, func(path string) error {
		return writeFile(ctx, path, data)
	})
}

func publishSQLite(ctx context.Context, name string, write func(string) error) (*Index, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := os.Open(filepath.Dir(name))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	stage, err := os.CreateTemp(filepath.Dir(name), ".catalog-*.sqlite")
	if err != nil {
		return nil, err
	}
	path := stage.Name()
	defer os.Remove(path)
	if err := stage.Close(); err != nil {
		return nil, err
	}
	if err := write(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if syncErr != nil {
		return nil, syncErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	// Validate and keep this physical connection so the returned index
	// continues to see the same file after its name changes.
	index, err := openIndex(ctx, path, 1)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		index.Close()
		return nil, err
	}
	if err := os.Rename(path, name); err != nil {
		index.Close()
		return nil, err
	}
	if err := dir.Sync(); err != nil {
		index.Close()
		return nil, err
	}
	return index, nil
}

func openStage(ctx context.Context, path string) (*sql.DB, error) {
	db, err := openFile(path, nil)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	// The staging file is private and discarded if the build fails. Its
	// completed state is fsynced before publication.
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=OFF; PRAGMA synchronous=OFF;"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func writeFile(ctx context.Context, path string, data snapshot) error {
	db, err := openStage(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := writeIndex(ctx, db, data); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "VACUUM; INSERT INTO product_text(product_text) VALUES('integrity-check');"); err != nil {
		return err
	}
	var status string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&status); err != nil {
		return err
	}
	if status != "ok" {
		return fmt.Errorf("invalid catalog index: %s", status)
	}
	return db.Close()
}
