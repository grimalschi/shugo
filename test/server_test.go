package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
)

const small = `
-- shugo.json --
{"redirects": {"/old": "/about"}}
-- pages/+layout.shugo --
---
import "../styles/main.css"
---
<html><head><title>T</title></head><body><g-slot /></body></html>
-- pages/index.shugo --
<h1>home</h1>
-- pages/about.shugo --
<h1>about</h1>
-- pages/404.shugo --
<h1>no such page</h1>
-- pages/café au lait.shugo --
<h1>café</h1>
-- styles/main.css --
h1 { color: red }
-- public/robots.txt --
User-agent: *
-- public/img/dot.svg --
<svg xmlns="http://www.w3.org/2000/svg"/>
`

func serve(t *testing.T, s *site) *httptest.Server { return serveSite(t, s) }

func serveSite(t testing.TB, s *site) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(&server.Server{
		Pages:  s.pages,
		Public: filepath.Join(s.root, project.Public),
		Assets: filepath.Join(s.root, project.Output, compile.Assets),
		Failed: func(err error) { t.Error(err) },
	})
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, ts *httptest.Server, address string) (int, http.Header, string) {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(ts.URL + address)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header, string(body)
}

func TestServer(t *testing.T) {
	s := newSite(t, small)
	s.build()
	ts := serve(t, s)

	status, header, body := get(t, ts, "/")
	if status != 200 || header.Get("Content-Type") != "text/html; charset=utf-8" || !strings.Contains(body, "<h1>home</h1>") {
		t.Errorf("/: %d %v %s", status, header, body)
	}
	// The compiler links the style sheet of the site from the head.
	link := body[strings.Index(body, `href="`)+len(`href="`):]
	link = link[:strings.Index(link, `"`)]
	if !strings.HasPrefix(link, "/_assets/styles.") {
		t.Fatalf("the page does not link the style sheet: %s", body)
	}
	if status, header, body := get(t, ts, link); status != 200 || !strings.Contains(body, "color:red") || !strings.Contains(header.Get("Cache-Control"), "immutable") {
		t.Errorf("%s: %d %v %s", link, status, header, body)
	}

	cases := []struct {
		address  string
		status   int
		contains string
	}{
		{"/about", 200, "<h1>about</h1>"},
		{"/about/", 308, ""},
		{"/old", 301, ""},
		{"/robots.txt", 200, "User-agent"},
		{"/img/dot.svg", 200, "<svg"},
		{"/img", 404, "no such page"},
		{"/nothing/here", 404, "no such page"},
		{"/404", 404, "no such page"},
		{"/caf%C3%A9%20au%20lait", 200, "café"},
		{"/..%2f..%2fshugo.json", 404, "no such page"},
		{"/_assets/../../shugo.json", 404, ""},
		{"/_assets/missing.css", 404, ""},
	}
	for _, c := range cases {
		status, header, body := get(t, ts, c.address)
		if status != c.status || !strings.Contains(body, c.contains) {
			t.Errorf("%s: %d %q, want %d with %q", c.address, status, body, c.status, c.contains)
		}
		if c.address == "/about/" && header.Get("Location") != "/about" {
			t.Errorf("/about/ redirects to %q", header.Get("Location"))
		}
		if c.address == "/old" && header.Get("Location") != "/about" {
			t.Errorf("/old redirects to %q", header.Get("Location"))
		}
	}
	if response, err := http.Post(ts.URL+"/", "text/plain", nil); err != nil || response.StatusCode != 405 {
		t.Errorf("POST: %v %v", response, err)
	}
}

// A server that is running shows the result of an update at once.
func TestServerSeesUpdates(t *testing.T) {
	s := newShop(t)
	s.build()
	ts := serve(t, s)
	if _, _, body := get(t, ts, "/products/jug"); !strings.Contains(body, "Clay and wood · 3 products") {
		t.Fatalf("before: %s", body)
	}
	s.sql(`UPDATE settings SET motto = 'New'; DELETE FROM products WHERE slug = 'tray'; DELETE FROM related WHERE other = 'tray'`)
	s.update()
	if _, _, body := get(t, ts, "/products/jug"); !strings.Contains(body, "New · 2 products") {
		t.Errorf("after: %s", body)
	}
	if status, _, _ := get(t, ts, "/products/tray"); status != 404 {
		t.Errorf("the removed page answers %d", status)
	}
}

// export writes the site into the folder out.
func (s *site) export(out string) server.Exported {
	s.t.Helper()
	done, err := server.Export(s.pages, filepath.Join(s.root, project.Public), filepath.Join(s.root, project.Output, compile.Assets), out)
	if err != nil {
		s.t.Fatal(err)
	}
	return done
}

func TestExport(t *testing.T) {
	s := newSite(t, small)
	s.build()
	out := filepath.Join(s.root, "dist")
	export := func() server.Exported { return s.export(out) }
	// What was in the folder before the first export is not the export's.
	for name, text := range map[string]string{"old/index.html": "old", "notes.txt": "notes", ".git/x/index.html": "git"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(out, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Four pages, two public files and the style sheet.
	if done := export(); done.Pages != 4 || done.Files != 7 || done.Written != 7 || done.Removed != 0 {
		t.Errorf("first export: %+v", done)
	}
	for _, name := range []string{"index.html", "about/index.html", "404.html", "café au lait/index.html", "robots.txt", "img/dot.svg"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(name))); err != nil {
			t.Error(err)
		}
	}
	if done := export(); done.Written != 0 || done.Removed != 0 {
		t.Errorf("an export with nothing new: %+v", done)
	}
	// One page changes and one goes away: one file is written, one removed.
	// The project has changed, so every page is built again, and the
	// folder is cleared of pages the site does not have.
	s.write("-- pages/about.shugo --\n<h1>about us</h1>")
	s.remove("pages/café au lait.shugo")
	s.build()
	if done := export(); done.Pages != 3 || done.Built != 3 || done.Written != 1 || done.Removed != 2 {
		t.Errorf("after a change: %+v", done)
	}
	for name, there := range map[string]bool{"café au lait/index.html": false, "café au lait": false, "old/index.html": false, "notes.txt": true, ".git/x/index.html": true} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(name))); (err == nil) != there {
			t.Errorf("%s: there %v, want %v", name, err == nil, there)
		}
	}
}

// After a change of the data, an export builds the pages whose HTML may
// differ, and no others.
func TestExportAfterUpdate(t *testing.T) {
	s := newShop(t)
	s.build()
	out := filepath.Join(s.root, "dist")
	if done := s.export(out); done.Pages != 5 || done.Built != 5 || done.Written != 5 {
		t.Errorf("first export: %+v", done)
	}
	if done := s.export(out); done.Built != 0 || done.Written != 0 {
		t.Errorf("an export with nothing new: %+v", done)
	}

	// The cup has its own page, and its card is on the main page and on
	// the page of the jug.
	s.sql(`UPDATE products SET price = 6 WHERE slug = 'cup'`)
	s.update()
	if done := s.export(out); done.Pages != 5 || done.Built != 3 || done.Written != 3 || done.Removed != 0 {
		t.Errorf("after a new price: %+v", done)
	}
	if jug, err := os.ReadFile(filepath.Join(out, "products", "jug", "index.html")); err != nil || !strings.Contains(string(jug), "Cup — 6") {
		t.Errorf("the page of the jug: %s %v", jug, err)
	}

	// The header counts the products, so a product that goes changes every
	// page, and its own is removed.
	s.sql(`DELETE FROM related WHERE other = 'tray'`)
	s.sql(`DELETE FROM products WHERE slug = 'tray'`)
	s.update()
	if done := s.export(out); done.Pages != 4 || done.Built != 4 || done.Removed != 1 {
		t.Errorf("after a product is gone: %+v", done)
	}

	// Another store says nothing about what is written: every page is
	// built, and only what differs is written.
	other := newShop(t)
	other.build()
	if done := other.export(out); done.Built != 5 || done.Written != 5 || done.Removed != 0 {
		t.Errorf("from another store: %+v", done)
	}
	if done := s.export(out); done.Built != 4 || done.Written != 4 || done.Removed != 1 {
		t.Errorf("back from another store: %+v", done)
	}
}

// While updates go on, every page a server gives belongs to one state of
// the store: the header counts as many products as the page lists.
func TestPagesStayWholeDuringUpdates(t *testing.T) {
	s := newShop(t)
	s.build()
	stop := make(chan struct{})
	failures := make(chan string, 16)
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				page, err := s.pages.Render(nil, "/")
				if err != nil || !page.Found {
					failures <- fmt.Sprintf("the page was not built: %v", err)
					return
				}
				html := string(page.HTML)
				listed := strings.Count(html, "<li>")
				if !strings.Contains(html, fmt.Sprintf("· %d products</header>", listed)) {
					failures <- "the header and the list come from different states: " + html
					return
				}
			}
		})
	}
	s.sql(`DELETE FROM related WHERE other = 'tray'`)
	for round := range 12 {
		if round%2 == 0 {
			s.sql(`DELETE FROM products WHERE slug = 'tray'`)
		} else {
			s.sql(fmt.Sprintf(`INSERT INTO products VALUES ('tray', 'Tray', %d, '', 3)`, round))
		}
		s.update()
	}
	close(stop)
	readers.Wait()
	select {
	case failure := <-failures:
		t.Error(failure)
	default:
	}
}
