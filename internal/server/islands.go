package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/grimalschi/shugo/internal/catalog"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/store"
)

const (
	catalogPath    = "/_islands/catalog"
	maxSearchQuery = 8192
)

// Catalog reads the catalog snapshot named by the page store.
type Catalog struct {
	dir       string
	mu        sync.Mutex
	indexName string
	index     *catalog.Index
}

func NewCatalog(dir string) *Catalog {
	return &Catalog{dir: dir}
}

func (i *Catalog) search(v *store.View, query url.Values) (catalog.Result, error) {
	name, err := v.Catalog()
	if err != nil {
		return catalog.Result{}, err
	}
	if name == "" {
		return catalog.Result{}, errNoCatalog
	}
	index, err := i.snapshot(name)
	if err != nil {
		return catalog.Result{}, err
	}
	return index.Search(query), nil
}

var errNoCatalog = errors.New("the store has no catalog index")
var errSearchTooLong = errors.New("catalog query is too long")

func (i *Catalog) snapshot(name string) (*catalog.Index, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.indexName != name {
		if filepath.Base(name) != name {
			return nil, errors.New("invalid catalog index name")
		}
		index, err := catalog.Open(filepath.Join(i.dir, name))
		if err != nil {
			return nil, err
		}
		i.index, i.indexName = index, name
	}
	return i.index, nil
}

func (s *Server) searchCatalog(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.RawQuery) > maxSearchQuery {
		http.Error(w, errSearchTooLong.Error(), http.StatusRequestURITooLong)
		return
	}
	var result catalog.Result
	err := s.Pages.store.View(func(v *store.View) error {
		var err error
		result, err = s.Catalog.search(v, r.URL.Query())
		return err
	})
	if errors.Is(err, errNoCatalog) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	data, err := json.Marshal(result)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Write(data)
}

func usesSearch(query url.Values) bool {
	for _, name := range []string{"q", "category", "size", "stock", "price_from", "price_to", "sort", "page"} {
		if query.Has(name) {
			return true
		}
	}
	return false
}

type replaceIsland func(*store.View, *program.Site, *render.Island) error

func (i *Catalog) forRequest(target *url.URL, rendered *bool) replaceIsland {
	if len(target.RawQuery) > maxSearchQuery {
		return func(*store.View, *program.Site, *render.Island) error { return errSearchTooLong }
	}
	query := target.Query()
	if !usesSearch(query) {
		return nil
	}
	var initial json.RawMessage
	return func(v *store.View, site *program.Site, island *render.Island) error {
		if initial == nil {
			result, err := i.search(v, query)
			if err != nil {
				return err
			}
			initial, err = json.Marshal(result)
			if err != nil {
				return err
			}
		}
		var props map[string]json.RawMessage
		if err := json.Unmarshal(island.Props, &props); err != nil {
			return err
		}
		props["initial"] = initial
		encoded, err := json.Marshal(props)
		if err != nil {
			return err
		}
		body, err := render.IslandBody(site, island.ID, encoded)
		if err != nil {
			return err
		}
		island.Props, island.Body = encoded, body
		*rendered = true
		return nil
	}
}
