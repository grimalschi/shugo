// Package server shows a site from its store: it answers requests with pages
// built on the fly, and it can write all pages out as files.
package server

import (
	"fmt"
	"sync"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/store"
)

// Pages builds the pages of a site from its store.
type Pages struct {
	store *store.Store
	// Mounts is markup added to every page by the name of its place, on top
	// of what the compiled project has there.
	Mounts map[string]string

	mu      sync.Mutex
	current *state

	// Values never change under their name, so they are kept in memory
	// across updates, up to a limit.
	values sync.Map // record.Hash → []byte
	held   int      // bytes, guarded by mu
}

// Most bytes of values kept in memory.
const maxHeld = 256 << 20

// state is what stays the same between two updates of the store.
type state struct {
	version int64
	build   string
	site    *program.Site
	shared  sync.Map // the number of a shared block → *record.Record
}

// NewPages returns the pages of the site kept in st.
func NewPages(st *store.Store) *Pages {
	return &Pages{store: st}
}

// state returns what is known about the store as view sees it.
func (p *Pages) state(view *store.View) (*state, error) {
	version, err := view.Version()
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current != nil && p.current.version == version {
		return p.current, nil
	}
	build, data, err := view.Build()
	if err != nil {
		return nil, err
	}
	if build == "" {
		return nil, fmt.Errorf("the store holds no site yet: run `shugo build` first")
	}
	next := &state{version: version, build: build}
	if p.current != nil && p.current.build == build {
		next.site = p.current.site
	} else {
		if next.site, err = program.Decode(data); err != nil {
			return nil, err
		}
		for name, markup := range p.Mounts {
			if next.site.Mounts == nil {
				next.site.Mounts = map[string]string{}
			}
			next.site.Mounts[name] += markup
		}
	}
	// A view older than the state already known is rare: an update ended
	// between two requests. It gets a state of its own.
	if p.current == nil || version > p.current.version {
		p.current = next
	}
	return next, nil
}

// source gives the renderer what a record names, from one view of the store.
type source struct {
	pages *Pages
	view  *store.View
	state *state
}

func (s source) Values(hash record.Hash) ([]byte, error) {
	if kept, ok := s.pages.values.Load(hash); ok {
		return kept.([]byte), nil
	}
	data, found, err := s.view.Values(hash)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("the store has lost the values of a block")
	}
	s.pages.hold(hash, data)
	return data, nil
}

func (p *Pages) hold(hash record.Hash, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.held += len(data) + len(hash); p.held > maxHeld {
		p.values.Clear()
		p.held = len(data) + len(hash)
	}
	p.values.Store(hash, data)
}

func (s source) Shared(id int64) (*record.Record, error) {
	if kept, ok := s.state.shared.Load(id); ok {
		return kept.(*record.Record), nil
	}
	data, found, err := s.view.Shared(id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("the store has lost a shared block")
	}
	rec, err := record.Decode(data)
	if err != nil {
		return nil, err
	}
	s.state.shared.Store(id, rec)
	return rec, nil
}

// Page is one page as the store has it at some moment.
type Page struct {
	HTML  []byte
	Found bool
	// Site is the program the page was built by.
	Site *program.Site
}

// Render appends the HTML of the page at path to dst. The whole page comes
// from one state of the store.
func (p *Pages) Render(dst []byte, path string) (Page, error) {
	var page Page
	err := p.store.View(func(view *store.View) error {
		st, err := p.state(view)
		if err != nil {
			return err
		}
		page.Site = st.site
		data, found, err := view.Page(path)
		if err != nil || !found {
			return err
		}
		page.HTML, err = p.render(dst, source{p, view, st}, data)
		page.Found = err == nil
		return err
	})
	if err != nil {
		return Page{HTML: dst, Site: page.Site}, fmt.Errorf("page %s: %w", path, err)
	}
	return page, nil
}

func (p *Pages) render(dst []byte, src source, data []byte) ([]byte, error) {
	rec, err := record.Decode(data)
	if err != nil {
		return dst, err
	}
	return render.Page(dst, src.state.site, rec, src)
}

// Each builds every page of the site, all from one state of the store, and
// calls visit with each. The HTML is valid only until visit returns.
func (p *Pages) Each(visit func(path string, html []byte) error) error {
	return p.store.View(func(view *store.View) error {
		st, err := p.state(view)
		if err != nil {
			return err
		}
		src := source{p, view, st}
		var html []byte
		return view.Pages(func(path string, data []byte) error {
			if html, err = p.render(html[:0], src, data); err != nil {
				return fmt.Errorf("page %s: %w", path, err)
			}
			return visit(path, html)
		})
	})
}
