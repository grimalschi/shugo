package evaluate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
	if !refresh && len(old) > 0 {
		w.catalog = old
		return false, nil
	}

	temp, err := os.CreateTemp(w.project.Path(project.Output), "catalog-*.tmp")
	if err != nil {
		return false, err
	}
	path := temp.Name()
	temp.Close()
	defer os.Remove(path)
	index, err := catalog.Build(context.Background(), w.project.Data(), path)
	if err != nil {
		return false, fmt.Errorf("catalog projection: %w", err)
	}
	w.catalog, err = json.Marshal(index.Search(nil))
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
	name := fmt.Sprintf("catalog-%x.json", hash.Sum(nil)[:16])
	if err := os.Rename(path, w.project.Path(project.Output, name)); err != nil {
		return false, err
	}
	return !bytes.Equal(old, w.catalog), u.SetCatalog(name, w.catalog)
}
