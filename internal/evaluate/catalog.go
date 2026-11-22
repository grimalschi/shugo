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

// prepareCatalog writes a new immutable projection before publishing its name
// in the same store transaction as the evaluated pages.
func (w *worker) prepareCatalog(u *store.Update, refresh bool) (bool, error) {
	old, err := u.CatalogDefault()
	if err != nil {
		return false, err
	}
	published, err := u.Catalog()
	if err != nil {
		return false, err
	}
	if !refresh && len(old) > 0 && filepath.Ext(published) == ".sqlite" {
		w.catalog = old
		return false, nil
	}

	dir := w.project.Path(project.Output)
	output, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer output.Close()
	temp, err := os.CreateTemp(dir, "catalog-*.tmp")
	if err != nil {
		return false, err
	}
	path := temp.Name()
	defer os.Remove(path)
	if err := temp.Close(); err != nil {
		return false, err
	}
	ctx := context.Background()
	index, err := catalog.Build(ctx, w.project.Data(), path)
	if err != nil {
		return false, fmt.Errorf("catalog projection: %w", err)
	}
	result, err := index.Search(ctx, nil)
	closeErr := index.Close()
	if err != nil {
		return false, fmt.Errorf("read new catalog projection: %w", err)
	}
	if closeErr != nil {
		return false, closeErr
	}
	w.catalog, err = json.Marshal(result)
	if err != nil {
		return false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	file.Close()
	if err != nil {
		return false, err
	}
	name := fmt.Sprintf("catalog-%x.sqlite", hash.Sum(nil)[:16])
	if err := os.Rename(path, filepath.Join(dir, name)); err != nil {
		return false, err
	}
	if err := output.Sync(); err != nil {
		return false, err
	}
	return !bytes.Equal(old, w.catalog), u.SetCatalog(name, w.catalog)
}
