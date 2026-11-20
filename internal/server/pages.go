// Package server shows a site from its store: it answers requests with pages
// built on the fly, and it can write all pages out as files.
package server

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/store"
)

// Pages builds the pages of a site from its store.
//
// It keeps the structure of the whole site in memory as of one update: the
// record of every page and of every shared block. A page is built from that
// alone, so all of it belongs to one state of the store, and no request
// waits for the store. Values are read when first needed and kept: under
// their name they never change.
type Pages struct {
	store *store.Store
	// Mounts is markup added to every page by the name of its place, on top
	// of what the compiled project has there.
	Mounts map[string]string

	current atomic.Pointer[state]
	checked atomic.Int64 // when the store was last asked for its version
	loading sync.Mutex

	values sync.Map // record.Hash → []byte
	held   atomic.Int64
}

const (
	// How long a server may go on showing the site as it was before an
	// update made by another process.
	staleFor = 50 * time.Millisecond
	// Most bytes of values kept in memory.
	maxHeld = 256 << 20
)

// state is the site as of one update of the store.
type state struct {
	version int64
	build   string
	site    *program.Site
	pages   map[string][]byte // records by address
	shared  map[int64]*record.Record
}

// NewPages returns the pages of the site kept in st.
func NewPages(st *store.Store) *Pages {
	return &Pages{store: st}
}

// Refresh makes the next page come from the store as it is now. A server
// finds an update of another process by itself, within staleFor.
func (p *Pages) Refresh() {
	p.checked.Store(0)
}

// state returns the site as of the latest update it knows of.
func (p *Pages) state() (*state, error) {
	now := time.Now().UnixNano()
	if st := p.current.Load(); st != nil && now-p.checked.Load() < int64(staleFor) {
		return st, nil
	}
	p.loading.Lock()
	defer p.loading.Unlock()
	st := p.current.Load()
	version, err := p.store.Version()
	if err != nil {
		return nil, err
	}
	if st == nil || st.version != version {
		if st, err = p.load(st); err != nil {
			return nil, err
		}
		p.current.Store(st)
	}
	p.checked.Store(now)
	return st, nil
}

// load reads the structure of the site from one state of the store.
func (p *Pages) load(last *state) (*state, error) {
	next := &state{pages: map[string][]byte{}, shared: map[int64]*record.Record{}}
	err := p.store.View(func(view *store.View) error {
		var err error
		if next.version, err = view.Version(); err != nil {
			return err
		}
		var data []byte
		if next.build, data, err = view.Build(); err != nil {
			return err
		}
		switch {
		case next.build == "":
			return errors.New("the store holds no site yet: run `shugo build` first")
		case last != nil && last.build == next.build:
			next.site = last.site
		default:
			if next.site, err = program.Decode(data); err != nil {
				return err
			}
			for name, markup := range p.Mounts {
				if next.site.Mounts == nil {
					next.site.Mounts = map[string]string{}
				}
				next.site.Mounts[name] += markup
			}
		}
		if err := view.Pages(func(path string, data []byte) { next.pages[path] = data }); err != nil {
			return err
		}
		var damaged error
		err = view.SharedBlocks(func(id int64, data []byte) {
			rec, err := record.Decode(data)
			if err != nil {
				damaged = fmt.Errorf("shared block %d: %w", id, err)
			}
			next.shared[id] = rec
		})
		return errors.Join(err, damaged)
	})
	return next, err
}

// errGone tells that the store no longer has the values of a block: an
// update has dropped them since the state was read.
var errGone = errors.New("the store has dropped the values of a block")

// source gives the renderer what the records of one state name.
type source struct {
	pages *Pages
	state *state
}

func (s source) Values(hash record.Hash) ([]byte, error) {
	if kept, ok := s.pages.values.Load(hash); ok {
		return kept.([]byte), nil
	}
	data, found, err := s.pages.store.Values(hash)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errGone
	}
	if s.pages.held.Add(int64(len(data)+len(hash))) > maxHeld {
		s.pages.values.Clear()
		s.pages.held.Store(0)
	}
	s.pages.values.Store(hash, data)
	return data, nil
}

func (s source) Shared(id int64) (*record.Record, error) {
	rec, ok := s.state.shared[id]
	if !ok {
		return nil, fmt.Errorf("there is no shared block %d", id)
	}
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
	for attempt := 0; ; attempt++ {
		st, err := p.state()
		if err != nil {
			return Page{HTML: dst}, err
		}
		data, found := st.pages[path]
		if !found {
			return Page{HTML: dst, Site: st.site}, nil
		}
		html, err := p.render(dst, st, data)
		if errors.Is(err, errGone) && attempt == 0 {
			// The page has changed since the state was read: read it again.
			p.Refresh()
			continue
		}
		if err != nil {
			return Page{HTML: dst, Site: st.site}, fmt.Errorf("page %s: %w", path, err)
		}
		return Page{HTML: html, Found: true, Site: st.site}, nil
	}
}

func (p *Pages) render(dst []byte, st *state, data []byte) ([]byte, error) {
	rec, err := record.Decode(data)
	if err != nil {
		return dst, err
	}
	return render.Page(dst, st.site, rec, source{p, st})
}

// Each builds every page of the site, all from one state of the store, in
// the order of addresses, and calls visit with each. The HTML is valid only
// until visit returns.
func (p *Pages) Each(visit func(path string, html []byte) error) error {
	p.Refresh()
	st, err := p.state()
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(st.pages))
	for path := range st.pages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var html []byte
	for _, path := range paths {
		if html, err = p.render(html[:0], st, st.pages[path]); err != nil {
			return fmt.Errorf("page %s: %w", path, err)
		}
		if err := visit(path, html); err != nil {
			return err
		}
	}
	return nil
}
