// Package test runs whole sites through every stage: it compiles a project,
// computes it on a real JavaScript runtime, and looks at the pages and at
// what an update reports.
package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/evaluate"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
	"github.com/grimalschi/shugo/internal/store"
)

// site is a project in a temporary folder with its store.
type site struct {
	t       *testing.T
	root    string
	store   *store.Store
	pages   *server.Pages
	runtime string
	// printed is what the project's code has printed.
	printed strings.Builder
}

// newSite makes a project of files written one after another, each under a
// line "-- path --".
func newSite(t *testing.T, files string) *site {
	t.Helper()
	s := &site{t: t, root: t.TempDir(), runtime: os.Getenv("SHUGO_RUNTIME")}
	s.write(files)
	if err := os.MkdirAll(filepath.Join(s.root, project.Output), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(s.root, project.Output, "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	s.store, s.pages = st, server.NewPages(st)
	return s
}

// write writes files given as newSite takes them.
func (s *site) write(files string) {
	s.t.Helper()
	var name string
	var body strings.Builder
	flush := func() {
		if name == "" {
			return
		}
		path := filepath.Join(s.root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			s.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.TrimSuffix(body.String(), "\n")), 0o644); err != nil {
			s.t.Fatal(err)
		}
		body.Reset()
	}
	for _, line := range strings.SplitAfter(strings.TrimPrefix(files, "\n"), "\n") {
		if rest, ok := strings.CutPrefix(line, "-- "); ok && strings.HasSuffix(strings.TrimSpace(rest), " --") {
			flush()
			name = strings.TrimSuffix(strings.TrimSpace(rest), " --")
			continue
		}
		body.WriteString(line)
	}
	flush()
}

func (s *site) remove(name string) {
	s.t.Helper()
	if err := os.Remove(filepath.Join(s.root, filepath.FromSlash(name))); err != nil {
		s.t.Fatal(err)
	}
}

func (s *site) project() *project.Project {
	s.t.Helper()
	p, err := project.Open(s.root)
	if err != nil {
		s.t.Fatal(err)
	}
	if s.runtime != "" {
		p.Config.Runtime = s.runtime
	}
	return p
}

// tryBuild compiles the project and computes it.
func (s *site) tryBuild() (*evaluate.Report, error) {
	p := s.project()
	compiled, err := compile.Build(p)
	if err != nil {
		return nil, err
	}
	return evaluate.Run(p, compiled, s.store, evaluate.Options{Log: &s.printed})
}

func (s *site) build() *evaluate.Report {
	s.t.Helper()
	report, err := s.tryBuild()
	if err != nil {
		s.t.Fatalf("build: %v", err)
	}
	return report
}

// update computes the compiled project again, as after a change of data.
func (s *site) update() *evaluate.Report {
	s.t.Helper()
	p := s.project()
	compiled, err := compile.Load(p)
	if err != nil {
		s.t.Fatal(err)
	}
	report, err := evaluate.Run(p, compiled, s.store, evaluate.Options{Log: &s.printed})
	if err != nil {
		s.t.Fatalf("update: %v", err)
	}
	return report
}

// page returns the HTML of a page.
func (s *site) page(address string) string {
	s.t.Helper()
	page, err := s.pages.Render(nil, address)
	if err != nil {
		s.t.Fatalf("page %s: %v", address, err)
	}
	if !page.Found {
		s.t.Fatalf("there is no page %s", address)
	}
	return string(page.HTML)
}

func (s *site) has(address string) bool {
	s.t.Helper()
	page, err := s.pages.Render(nil, address)
	if err != nil {
		s.t.Fatalf("page %s: %v", address, err)
	}
	return page.Found
}

// expect compares the HTML of a page with what it should be.
func expect(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("\n got: %s\nwant: %s", got, want)
	}
}
