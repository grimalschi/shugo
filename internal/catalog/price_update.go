package catalog

import (
	"context"
	"fmt"
	"io"
	"os"
)

// RefreshPrices copies a previously published standalone SQLite index and
// publishes a new generation with the given prices at a separate path.
func RefreshPrices(ctx context.Context, previous, dest string, changes []PriceChange) (*Index, error) {
	index, err := publishSQLite(ctx, dest, func(path string) error {
		if err := copyIndex(previous, path); err != nil {
			return fmt.Errorf("copy previous catalog index: %w", err)
		}
		return updatePrices(ctx, path, changes)
	})
	if err != nil {
		return nil, fmt.Errorf("refresh catalog prices: %w", err)
	}
	return index, nil
}

func copyIndex(previous, stage string) error {
	source, err := os.Open(previous)
	if err != nil {
		return err
	}
	defer source.Close()
	dest, err := os.OpenFile(stage, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dest, source)
	closeErr := dest.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func updatePrices(ctx context.Context, path string, changes []PriceChange) error {
	db, err := openStage(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statement, err := tx.PrepareContext(ctx, "UPDATE product SET original_price = ?, price = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, change := range changes {
		result, err := statement.ExecContext(ctx, change.OriginalPrice, change.Price, change.ID)
		if err != nil {
			return fmt.Errorf("update price for product %q: %w", change.ID, err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("catalog product %q not found", change.ID)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return db.Close()
}
