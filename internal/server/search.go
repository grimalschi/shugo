package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/grimalschi/shugo/internal/search"
	"github.com/grimalschi/shugo/internal/store"
)

// Search reads the generic search indices named by the store.
type Search struct {
	*snapshots[*search.Index]
}

func NewSearch(dir string) *Search {
	return &Search{newSnapshots(dir, 16, search.Open)}
}

func (s *Search) query(ctx context.Context, binding store.SearchBinding, params url.Values) (search.Result, error) {
	snapshot, err := s.take(binding.Snapshot)
	if err != nil {
		return search.Result{}, err
	}
	defer s.release(snapshot)
	return snapshot.index.Search(ctx, binding.Definition, params)
}

func (s *Server) searchIndex(w http.ResponseWriter, r *http.Request) bool {
	id, ok := strings.CutPrefix(r.URL.Path, search.EndpointPrefix)
	if !ok || id == "" || strings.Contains(id, "/") {
		return false
	}
	var found bool
	var result search.Result
	err := s.Pages.store.View(func(v *store.View) error {
		binding, exists, err := v.Search(id)
		found = exists
		if err != nil || !found {
			return err
		}
		if len(r.URL.RawQuery) > maxSearchQuery {
			return errSearchTooLong
		}
		result, err = s.Search.query(r.Context(), binding, r.URL.Query())
		return err
	})
	if err == nil && !found {
		return false
	}
	s.writeSearch(w, r, result, err)
	return true
}
