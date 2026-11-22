package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	*snapshots[*catalog.Index]
}

func NewCatalog(dir string) *Catalog {
	return &Catalog{newSnapshots(dir, 1, catalog.Open)}
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
	snapshot, err := i.take(name)
	if err != nil {
		return catalog.Result{}, err
	}
	defer i.release(snapshot)
	return snapshot.index.Search(ctx, target.Query())
}

var errNoCatalog = errors.New("the store has no catalog index")
var errSearchTooLong = errors.New("search query is too long")

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
	s.writeSearch(w, r, result, err)
	return true
}

func (s *Server) writeSearch(w http.ResponseWriter, r *http.Request, result any, err error) {
	if errors.Is(err, errSearchTooLong) {
		http.Error(w, errSearchTooLong.Error(), http.StatusRequestURITooLong)
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

func (s *Server) forRequest(ctx context.Context, target *url.URL, rendered *bool) replaceIsland {
	var query url.Values
	if len(target.RawQuery) <= maxSearchQuery {
		query = target.Query()
	}
	results := map[string]json.RawMessage{}
	return func(v *store.View, site *program.Site, island *render.Island) error {
		if len(target.RawQuery) > maxSearchQuery {
			return errSearchTooLong
		}
		bindings := island.Request
		if island.Search && len(bindings) == 0 {
			bindings = map[string]string{"initial": "catalog"}
		}
		var props map[string]json.RawMessage
		for name, source := range bindings {
			result, known := results[source]
			if !known {
				var err error
				result, err = s.requestSearch(ctx, v, target, query, source)
				if err != nil {
					return fmt.Errorf("island %s: %w", island.ID, err)
				}
				results[source] = result
			}
			if result == nil {
				continue
			}
			if props == nil {
				if err := json.Unmarshal(island.Props, &props); err != nil {
					return err
				}
			}
			props[name] = result
		}
		if props == nil {
			return nil
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

func (s *Server) requestSearch(ctx context.Context, v *store.View, target *url.URL, query url.Values, source string) (json.RawMessage, error) {
	switch {
	case source == "catalog" && s.Catalog != nil:
		if usesSearch(query) {
			result, err := s.Catalog.search(ctx, v, target)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		}
	case strings.HasPrefix(source, "search:") && s.Search != nil:
		id := strings.TrimPrefix(source, "search:")
		binding, found, err := v.Search(id)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("search %s has no published binding", id)
		}
		if binding.Definition.UsesQuery(query) {
			result, err := s.Search.query(ctx, binding, query)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		}
	default:
		return nil, fmt.Errorf("unknown request data source %q", source)
	}
	return nil, nil
}
