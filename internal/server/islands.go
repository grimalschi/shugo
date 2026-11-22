package server

import (
	"context"
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
	dir     string
	mu      sync.Mutex
	current *catalogSnapshot
	closed  bool
}

type catalogSnapshot struct {
	name    string
	index   *catalog.Index
	readers int
}

func NewCatalog(dir string) *Catalog {
	return &Catalog{dir: dir}
}

func (i *Catalog) search(ctx context.Context, v *store.View, target *url.URL) (catalog.Result, error) {
	name, err := v.Catalog()
	if err != nil {
		return catalog.Result{}, err
	}
	if name == "" {
		return catalog.Result{}, errNoCatalog
	}
	if len(target.RawQuery) > maxSearchQuery {
		return catalog.Result{}, errSearchTooLong
	}
	snapshot, err := i.snapshot(name)
	if err != nil {
		return catalog.Result{}, err
	}
	defer i.release(snapshot)
	return snapshot.index.Search(ctx, target.Query())
}

var errNoCatalog = errors.New("the store has no catalog index")
var errSearchTooLong = errors.New("catalog query is too long")

func (i *Catalog) snapshot(name string) (*catalogSnapshot, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return nil, errors.New("catalog reader is closed")
	}
	if i.current == nil || i.current.name != name {
		if filepath.Base(name) != name || !filepath.IsLocal(name) {
			return nil, errors.New("invalid catalog index name")
		}
		index, err := catalog.Open(filepath.Join(i.dir, name))
		if err != nil {
			return nil, err
		}
		old := i.current
		i.current = &catalogSnapshot{name: name, index: index}
		if old != nil && old.readers == 0 {
			old.index.Close()
		}
	}
	i.current.readers++
	return i.current, nil
}

func (i *Catalog) release(snapshot *catalogSnapshot) {
	i.mu.Lock()
	defer i.mu.Unlock()
	snapshot.readers--
	if snapshot != i.current && snapshot.readers == 0 {
		snapshot.index.Close()
	}
}

// Close releases the current index. Active searches release their snapshots
// when they finish.
func (i *Catalog) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.closed = true
	old := i.current
	i.current = nil
	if old != nil && old.readers == 0 {
		return old.index.Close()
	}
	return nil
}

// Without a published index the path remains available to the site itself.
func (s *Server) searchCatalog(w http.ResponseWriter, r *http.Request) bool {
	var result catalog.Result
	err := s.Pages.store.View(func(v *store.View) error {
		var err error
		result, err = s.Catalog.search(r.Context(), v, r.URL)
		return err
	})
	if errors.Is(err, errNoCatalog) {
		return false
	}
	if errors.Is(err, errSearchTooLong) {
		http.Error(w, errSearchTooLong.Error(), http.StatusRequestURITooLong)
		return true
	}
	if err != nil {
		s.fail(w, err)
		return true
	}
	data, err := json.Marshal(result)
	if err != nil {
		s.fail(w, err)
		return true
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return true
	}
	w.Write(data)
	return true
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

func (i *Catalog) forRequest(ctx context.Context, target *url.URL, rendered *bool) replaceIsland {
	var checked bool
	var initial json.RawMessage
	return func(v *store.View, site *program.Site, island *render.Island) error {
		if !checked {
			checked = true
			if len(target.RawQuery) > maxSearchQuery {
				return errSearchTooLong
			}
			if !usesSearch(target.Query()) {
				return nil
			}
			result, err := i.search(ctx, v, target)
			if err != nil {
				return err
			}
			initial, err = json.Marshal(result)
			if err != nil {
				return err
			}
		}
		if initial == nil {
			return nil
		}
		var props map[string]json.RawMessage
		if err := json.Unmarshal(island.Props, &props); err != nil {
			return err
		}
		if island.Search && len(island.Request) == 0 {
			props["initial"] = initial
		}
		for name, source := range island.Request {
			if source != "catalog" {
				return fmt.Errorf("island %s: unknown request data source %q", island.ID, source)
			}
			props[name] = initial
		}
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
