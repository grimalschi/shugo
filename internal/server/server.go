package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grimalschi/shugo/internal/compile"
)

// Server answers requests for the pages and the files of one site. A page
// is built anew for every request from what the store has at that moment.
type Server struct {
	Pages *Pages
	// Public is the folder of files that are published as they are, and
	// Assets the folder of files the compiler made, published under
	// compile.AssetsURL.
	Public, Assets string
	// Live, when set, tells open pages to reload after every update.
	Live *Live
	// Failed is called with every error that a visitor sees as a status 500.
	Failed func(error)
}

// The page shown for an address that has no page.
const notFound = "/404"

var buffers = sync.Pool{New: func() any { return new([]byte) }}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "this site only answers GET and HEAD", http.StatusMethodNotAllowed)
		return
	}
	address := r.URL.Path
	switch {
	case strings.HasPrefix(address, compile.AssetsURL):
		// The name of such a file changes with its content.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		s.file(w, r, s.Assets, strings.TrimPrefix(address, compile.AssetsURL))
		return
	case s.Live != nil && address == livePath:
		s.Live.serve(w, r)
		return
	case len(address) > 1 && strings.HasSuffix(address, "/"):
		// A page has one address, the one without a slash at the end.
		target := strings.TrimRight(address, "/")
		if target == "" {
			target = "/"
		}
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
		return
	}

	buffer := buffers.Get().(*[]byte)
	defer buffers.Put(buffer)
	page, err := s.Pages.Render((*buffer)[:0], address)
	if err != nil {
		s.fail(w, err)
		return
	}
	if page.Found && address != notFound {
		*buffer = page.Body
		s.answer(w, r, http.StatusOK, page)
		return
	}
	if page.Site != nil {
		if target, ok := page.Site.Redirects[address]; ok {
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
	}
	if s.file(w, r, s.Public, address) {
		return
	}
	// No page and no file: the page for that, or a plain answer.
	page, err = s.Pages.Render((*buffer)[:0], notFound)
	if err != nil {
		s.fail(w, err)
		return
	}
	if !page.Found {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	*buffer = page.Body
	s.answer(w, r, http.StatusNotFound, page)
}

// answer answers with a page. The page may differ at the next request, so
// the browser asks every time, with the tag of the page it has: the same
// page is answered with 304 and no body.
func (s *Server) answer(w http.ResponseWriter, r *http.Request, status int, page Page) {
	header := w.Header()
	header.Set("Content-Type", page.Type)
	header.Set("Cache-Control", "no-cache")
	if status == http.StatusOK {
		tag := pageTag(page.Body)
		header.Set("ETag", tag)
		if hasTag(r.Header.Get("If-None-Match"), tag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	write(w, r, status, page.Body)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	if s.Failed != nil {
		s.Failed(err)
	}
	http.Error(w, "500 the page could not be built", http.StatusInternalServerError)
}

// file answers with a file of a folder if there is one at the address.
func (s *Server) file(w http.ResponseWriter, r *http.Request, folder, address string) bool {
	if folder == "" {
		return false
	}
	// Clean keeps the path inside the folder whatever the address holds.
	name := filepath.Join(folder, filepath.FromSlash(path.Clean("/"+address)))
	info, err := os.Stat(name)
	if err != nil || info.IsDir() {
		if folder == s.Assets {
			http.NotFound(w, r)
		}
		return false
	}
	serveFile(w, r, name, info)
	return true
}
