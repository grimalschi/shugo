package evaluate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/grimalschi/shugo/internal/catalog"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/store"
)

// catalogPreparation owns a staged snapshot until the source reads have been
// validated. Its default result is also the value used for page evaluation.
type catalogPreparation struct {
	changed bool
	prices  int
	stage   string
	name    string
	initial []byte
}

func (p *catalogPreparation) discard() {
	if p.stage != "" {
		os.Remove(p.stage)
	}
}

func (p *catalogPreparation) publish(u *store.Update) error {
	if p.stage == "" {
		return nil
	}
	dir := filepath.Dir(p.stage)
	output, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer output.Close()
	if err := os.Rename(p.stage, filepath.Join(dir, p.name)); err != nil {
		return err
	}
	p.stage = ""
	if err := output.Sync(); err != nil {
		return err
	}
	return u.SetCatalog(p.name, p.initial)
}

// prepareCatalog computes an immutable projection and gives its default
// result to the evaluator. The caller publishes it with the evaluated pages.
func (w *worker) prepareCatalog(u *store.Update, plan catalog.UpdatePlan) (catalogPreparation, error) {
	var prepared catalogPreparation
	old, err := u.CatalogDefault()
	if err != nil {
		return prepared, err
	}
	published, err := u.Catalog()
	if err != nil {
		return prepared, err
	}
	dir := w.project.Path(project.Output)
	current := filepath.Ext(published) == ".sqlite" && filepath.IsLocal(published) && filepath.Base(published) == published
	if current {
		info, err := os.Lstat(filepath.Join(dir, published))
		current = err == nil && info.Mode().IsRegular()
	}
	if !plan.Rebuild && len(plan.Prices) == 0 && len(old) > 0 && current {
		w.catalog = old
		return prepared, nil
	}

	temp, err := os.CreateTemp(dir, "catalog-*.tmp")
	if err != nil {
		return prepared, err
	}
	path := temp.Name()
	defer func() {
		if prepared.stage == "" {
			os.Remove(path)
		}
	}()
	if err := temp.Close(); err != nil {
		return prepared, err
	}
	ctx := context.Background()
	var index *catalog.Index
	if !plan.Rebuild && current && len(plan.Prices) > 0 {
		// An artifact that cannot be updated is restored from the source.
		index, err = catalog.RefreshPrices(ctx, filepath.Join(dir, published), path, plan.Prices)
		if err == nil {
			prepared.prices = len(plan.Prices)
		}
	}
	if index == nil {
		index, err = catalog.Build(ctx, w.project.Data(), path)
		if err != nil {
			return prepared, fmt.Errorf("catalog projection: %w", err)
		}
	}
	result, err := index.Search(ctx, nil)
	closeErr := index.Close()
	if err != nil {
		return prepared, fmt.Errorf("read new catalog projection: %w", err)
	}
	if closeErr != nil {
		return prepared, closeErr
	}
	w.catalog, err = json.Marshal(result)
	if err != nil {
		return prepared, err
	}
	file, err := os.Open(path)
	if err != nil {
		return prepared, err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	closeErr = file.Close()
	if err != nil {
		return prepared, err
	}
	if closeErr != nil {
		return prepared, closeErr
	}
	prepared.stage = path
	prepared.name = fmt.Sprintf("catalog-%x.sqlite", hash.Sum(nil)[:16])
	prepared.initial = w.catalog
	prepared.changed = !bytes.Equal(old, w.catalog)
	return prepared, nil
}
