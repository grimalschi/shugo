package server

import (
	"net/http"
	"sync"
)

// Live tells the pages open in browsers to reload. It is for development:
// the pages get a short script that waits for the word.
type Live struct {
	mu       sync.Mutex
	watchers map[chan struct{}]struct{}
}

const livePath = "/_shugo/live"

// LiveScript is the markup that makes a page listen for reloads.
const LiveScript = `<script>new EventSource("` + livePath + `").addEventListener("reload", () => location.reload())</script>`

// Reload tells every open page to load itself again.
func (l *Live) Reload() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for watcher := range l.watchers {
		select {
		case watcher <- struct{}{}:
		default: // it has a reload waiting already
		}
	}
}

// serve keeps the connection of one page open and writes an event for
// every reload.
func (l *Live) serve(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "this connection cannot stay open", http.StatusInternalServerError)
		return
	}
	watcher := make(chan struct{}, 1)
	l.mu.Lock()
	if l.watchers == nil {
		l.watchers = map[chan struct{}]struct{}{}
	}
	l.watchers[watcher] = struct{}{}
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		delete(l.watchers, watcher)
		l.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(": connected\n\n"))
	flusher.Flush()
	for {
		select {
		case <-watcher:
			w.Write([]byte("event: reload\ndata: 1\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
