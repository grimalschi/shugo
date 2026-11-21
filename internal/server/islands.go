package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/grimalschi/shugo/internal/catalog"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/store"
)

const (
	catalogPath    = "/_islands/catalog"
	maxSearchQuery = 8192
)

// Islands reads the catalog snapshot named by the page store. The JS worker
// starts lazily: opening a default page or calling the JSON API uses only Go.
type Islands struct {
	dir       string
	mu        sync.Mutex
	indexName string
	index     *catalog.Index
	worker    islandWorker
}

func NewIslands(dir, runtime string, log io.Writer) *Islands {
	return &Islands{dir: dir, worker: islandWorker{runtime: runtime, dir: dir, log: log, gate: make(chan struct{}, 1)}}
}

func (i *Islands) Close() {
	i.worker.gate <- struct{}{}
	defer func() { <-i.worker.gate }()
	i.worker.stop()
}

func (i *Islands) search(v *store.View, query url.Values) (catalog.Result, error) {
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

func (i *Islands) snapshot(name string) (*catalog.Index, error) {
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
		result, err = s.Islands.search(v, r.URL.Query())
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

func (i *Islands) forRequest(ctx context.Context, target *url.URL, rendered *bool) replaceIsland {
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
		body, err := i.worker.render(ctx, site.IslandWorker, island.ID, encoded)
		if err != nil {
			return err
		}
		island.Props, island.Body = encoded, body
		*rendered = true
		return nil
	}
}

// A worker belongs to one compiled project. One request at a time uses its
// pipes. A broken or timed-out worker is discarded and starts fresh next time.
type islandWorker struct {
	dir, runtime string
	log          io.Writer
	gate         chan struct{}
	bundle       string
	process      *exec.Cmd
	asks         io.WriteCloser
	replies      *os.File
	lines        *bufio.Scanner
}

func (w *islandWorker) start(bundle string) error {
	if bundle == "" || filepath.Base(bundle) != bundle {
		return errors.New("invalid island SSR bundle name")
	}
	read, write, err := os.Pipe()
	if err != nil {
		return err
	}
	defer write.Close()
	command := exec.Command(w.runtime, filepath.Join(w.dir, bundle))
	command.Dir = filepath.Dir(w.dir)
	command.Stdout, command.Stderr = w.log, w.log
	command.ExtraFiles = []*os.File{write}
	input, err := command.StdinPipe()
	if err != nil {
		read.Close()
		return err
	}
	if err := command.Start(); err != nil {
		read.Close()
		input.Close()
		return fmt.Errorf("start island renderer: %w", err)
	}
	w.bundle, w.process, w.asks, w.replies = bundle, command, input, read
	w.lines = bufio.NewScanner(read)
	w.lines.Buffer(make([]byte, 32<<10), 8<<20)
	return nil
}

func (w *islandWorker) stop() {
	if w.process == nil {
		return
	}
	w.process.Process.Kill()
	w.asks.Close()
	w.replies.Close()
	w.process.Wait()
	w.bundle, w.process = "", nil
}

func (w *islandWorker) render(ctx context.Context, bundle, id string, props json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	select {
	case w.gate <- struct{}{}:
		defer func() { <-w.gate }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if w.bundle != bundle || w.process == nil {
		w.stop()
		if err := w.start(bundle); err != nil {
			return "", err
		}
	}
	type reply struct {
		Body, Head, Error string
		err               error
	}
	line, err := json.Marshal(struct {
		ID    string          `json:"id"`
		Props json.RawMessage `json:"props"`
	}{id, props})
	if err != nil {
		return "", err
	}
	done := make(chan reply, 1)
	go func() {
		answer := reply{}
		if _, err := w.asks.Write(append(line, '\n')); err != nil {
			answer.err = err
		} else if w.lines.Scan() {
			answer.err = json.Unmarshal(w.lines.Bytes(), &answer)
		} else if answer.err = w.lines.Err(); answer.err == nil {
			answer.err = io.ErrUnexpectedEOF
		}
		done <- answer
	}()
	select {
	case answer := <-done:
		if answer.err != nil {
			w.stop()
			return "", fmt.Errorf("island SSR connection: %w", answer.err)
		}
		if answer.Error != "" {
			return "", fmt.Errorf("island %s: %s", id, answer.Error)
		}
		if answer.Head != "" {
			return "", errors.New("put the island’s head elements in its Shugo page")
		}
		if answer.Body == "" {
			return "", errors.New("island renderer returned an empty body")
		}
		return answer.Body, nil
	case <-ctx.Done():
		w.stop()
		<-done
		return "", ctx.Err()
	}
}
