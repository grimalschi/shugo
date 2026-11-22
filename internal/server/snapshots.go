package server

import (
	linkedlist "container/list"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
)

// snapshots keeps immutable search files open. An evicted file is closed
// when the last request using it releases it.
type snapshots[T io.Closer] struct {
	mu     sync.Mutex
	dir    string
	limit  int
	open   func(string) (T, error)
	loaded map[string]*snapshot[T]
	recent linkedlist.List
	closed bool
}

type snapshot[T io.Closer] struct {
	name    string
	index   T
	entry   *linkedlist.Element
	readers int
}

func newSnapshots[T io.Closer](dir string, limit int, open func(string) (T, error)) *snapshots[T] {
	return &snapshots[T]{dir: dir, limit: limit, open: open, loaded: map[string]*snapshot[T]{}}
}

func (p *snapshots[T]) take(name string) (*snapshot[T], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("search index reader is closed")
	}
	s := p.loaded[name]
	if s == nil {
		if !filepath.IsLocal(name) || filepath.Base(name) != name {
			return nil, fmt.Errorf("invalid search index filename %q", name)
		}
		index, err := p.open(filepath.Join(p.dir, name))
		if err != nil {
			return nil, err
		}
		s = &snapshot[T]{name: name, index: index}
		p.loaded[name] = s
		s.entry = p.recent.PushFront(s)
		if p.recent.Len() > p.limit {
			p.remove(p.recent.Back().Value.(*snapshot[T]))
		}
	} else {
		p.recent.MoveToFront(s.entry)
	}
	s.readers++
	return s, nil
}

func (p *snapshots[T]) release(s *snapshot[T]) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s.readers--
	if s.entry == nil && s.readers == 0 {
		s.index.Close()
	}
}

func (p *snapshots[T]) remove(s *snapshot[T]) error {
	delete(p.loaded, s.name)
	p.recent.Remove(s.entry)
	s.entry = nil
	if s.readers == 0 {
		return s.index.Close()
	}
	return nil
}

func (p *snapshots[T]) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	var result error
	for p.recent.Len() > 0 {
		result = errors.Join(result, p.remove(p.recent.Back().Value.(*snapshot[T])))
	}
	return result
}
