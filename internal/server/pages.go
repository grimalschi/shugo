// Package server shows a site from its store: it answers requests with pages
// built on the fly, and it can write the pages out as files.
package server

import (
	"errors"
	"fmt"
	"mime"
	"sync"

	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/store"
)

// Pages builds the pages of a site from its store.
//
// Nothing of the site is kept in memory: every page is read from the store
// when it is asked for, its record and the records of the shared blocks it
// shows, all in one read of the store. So a page always comes whole from one
// state of the site, and an update is seen by the next page asked for, at
// no cost to the server. Only the skeleton of the site is kept, decoded once
// for every compiled project: it is code, not data, and does not change
// between builds.
type Pages struct {
	store *store.Store
	// Mounts is markup added to every page by the name of its place, on top
	// of what the compiled project has there.
	Mounts map[string]string

	mu    sync.Mutex
	build string
	site  *skeleton.Site
}

// NewPages returns the pages of the site kept in st.
func NewPages(st *store.Store) *Pages {
	return &Pages{store: st}
}

// errEmpty tells that the store holds no site.
var errEmpty = errors.New("the store holds no site yet: run `shugo build` first")

// Ready tells whether the store holds a site that can be shown.
func (p *Pages) Ready() error {
	return p.store.View(func(v *store.View) error {
		_, err := p.skeleton(v)
		return err
	})
}

// skeleton returns the skeleton of the compiled project the store holds.
func (p *Pages) skeleton(v *store.View) (*skeleton.Site, error) {
	build, err := v.BuildID()
	if err != nil {
		return nil, err
	}
	if build == "" {
		return nil, errEmpty
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.site != nil && p.build == build {
		return p.site, nil
	}
	_, data, err := v.Build()
	if err != nil {
		return nil, err
	}
	site, err := skeleton.Decode(data)
	if err != nil {
		return nil, err
	}
	for name, markup := range p.Mounts {
		if site.Mounts == nil {
			site.Mounts = map[string]string{}
		}
		site.Mounts[name] += markup
	}
	p.build, p.site = build, site
	return site, nil
}

// source gives the renderer the shared blocks a page shows, read from the
// same state of the store as the page. A block shown twice is read once.
type source struct {
	view   *store.View
	shared map[int64]*record.Record
}

func (s *source) Shared(id int64) (*record.Record, error) {
	if rec, read := s.shared[id]; read {
		return rec, nil
	}
	data, found, err := s.view.Shared(id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("there is no shared block %d", id)
	}
	rec, err := record.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("shared block %d: %w", id, err)
	}
	if s.shared == nil {
		s.shared = map[int64]*record.Record{}
	}
	s.shared[id] = rec
	return rec, nil
}

// Page is one page as the store has it at some moment.
type Page struct {
	// Body is the page: its HTML, or the file a module gives.
	Body []byte
	// Type is the media type of the body: "text/html; charset=utf-8" for
	// a page of templates.
	Type  string
	Found bool
	// Site is the skeleton the page was built by.
	Site *skeleton.Site
}

// Render appends the page at path to dst.
func (p *Pages) Render(dst []byte, path string) (Page, error) {
	return p.render(dst, path, nil)
}

// render is Render for a request that may give islands of the page other
// input: replace gives it to them.
func (p *Pages) render(dst []byte, path string, replace replaceIsland) (Page, error) {
	page := Page{Body: dst}
	err := p.store.View(func(v *store.View) error {
		site, err := p.skeleton(v)
		if err != nil {
			return err
		}
		page.Site = site
		data, route, found, err := v.Page(path)
		if err != nil || !found {
			return err
		}
		var update func(*render.Island) error
		if replace != nil {
			update = func(island *render.Island) error { return replace(v, site, island) }
		}
		body, err := buildWithIslands(dst, site, route, data, &source{view: v}, update)
		if err != nil {
			return fmt.Errorf("page %s: %w", path, err)
		}
		page.Body, page.Type, page.Found = body, mediaType(site, route), true
		return nil
	})
	return page, err
}

// build builds a page of a route from its record.
func build(dst []byte, site *skeleton.Site, route int, data []byte, src *source) ([]byte, error) {
	return buildWithIslands(dst, site, route, data, src, nil)
}

// buildWithIslands is build for a request that may give islands of the
// page other input.
func buildWithIslands(dst []byte, site *skeleton.Site, route int, data []byte, src *source, replace func(*render.Island) error) ([]byte, error) {
	rec, err := record.Decode(data)
	if err != nil {
		return dst, err
	}
	return render.PageWithIslands(dst, site, route, rec, src, replace)
}

// mediaType tells what a page of a route is: HTML, or for a module the
// type of the files it gives, as their extension names it for a file of
// the public folder.
func mediaType(site *skeleton.Site, route int) string {
	if !site.FileRoute(route) {
		return "text/html; charset=utf-8"
	}
	if t := mime.TypeByExtension(site.Routes[route].Extension); t != "" {
		return t
	}
	return "application/octet-stream"
}
