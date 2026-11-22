package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/store"
)

// maxSearchQuery is the longest query of a search a server reads.
const maxSearchQuery = 8192

var errSearchTooLong = errors.New("search query is too long")

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

type replaceIsland func(*store.View, *skeleton.Site, *render.Island) error

func (s *Server) forRequest(ctx context.Context, target *url.URL, rendered *bool) replaceIsland {
	var query url.Values
	if len(target.RawQuery) <= maxSearchQuery {
		query = target.Query()
	}
	results := map[string]json.RawMessage{}
	return func(v *store.View, site *skeleton.Site, island *render.Island) error {
		if len(target.RawQuery) > maxSearchQuery {
			return errSearchTooLong
		}
		var props map[string]json.RawMessage
		for name, source := range island.Request {
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
