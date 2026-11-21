package test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/cli"
	"github.com/grimalschi/shugo/internal/evaluate"
)

// shugo runs the command as a user would and returns what it printed.
func shugo(t *testing.T, arguments ...string) (stdout, stderr string, status int) {
	t.Helper()
	var out, errs bytes.Buffer
	if runtime := os.Getenv("SHUGO_RUNTIME"); runtime != "" {
		arguments = append(arguments, "-runtime", runtime)
	}
	status = cli.Main(arguments, &out, &errs)
	return out.String(), errs.String(), status
}

func TestCommands(t *testing.T) {
	s := newShop(t)
	s.store.Close() // the commands open the store themselves

	if _, errs, status := shugo(t, "serve", s.root); status != 1 || !strings.Contains(errs, "run `shugo build` first") {
		t.Errorf("serve before a build: %d %s", status, errs)
	}
	out, errs, status := shugo(t, "build", s.root)
	if status != 0 || !strings.Contains(out, "built in") || !strings.Contains(out, "5 pages, 5 shared blocks") {
		t.Fatalf("build: %d\n%s\n%s", status, out, errs)
	}

	// The report as JSON, for programs.
	s.sql(`UPDATE settings SET motto = 'New'`)
	out, errs, status = shugo(t, "update", "-json", s.root)
	var report evaluate.Report
	if err := json.Unmarshal([]byte(out), &report); status != 0 || err != nil {
		t.Fatalf("update: %d %v\n%s\n%s", status, err, out, errs)
	}
	if report.Shared.Changed != 1 || report.Pages.Changed != 0 || report.Render != 5 || report.Time["total"] <= 0 {
		t.Errorf("report: %+v", report)
	}

	// In words, an update says that the header is on every page.
	s.sql(`UPDATE settings SET motto = 'Newer'`)
	if out, _, _ := shugo(t, "update", s.root); !strings.Contains(out, "the HTML differs on 5 pages") {
		t.Errorf("update in words: %s", out)
	}

	dist := filepath.Join(s.root, "site")
	out, errs, status = shugo(t, "export", "-out", dist, s.root)
	if status != 0 || !strings.Contains(out, "5 pages") {
		t.Fatalf("export: %d\n%s\n%s", status, out, errs)
	}
	if html := s.read("site/about/index.html"); !strings.Contains(html, "Newer · 3 products") {
		t.Errorf("the exported page: %s", html)
	}

	for _, wrong := range [][]string{{"nonsense"}, {"build", "a", "b"}, {"build", "-nonsense"}} {
		if _, _, status := shugo(t, wrong...); status != 2 {
			t.Errorf("shugo %v: status %d", wrong, status)
		}
	}
	if _, errs, status := shugo(t, "build", t.TempDir()); status != 1 || !strings.Contains(errs, "not a Shugo project") {
		t.Errorf("a folder without pages: %d %s", status, errs)
	}
}

// A mistake is printed with the lines of the file around it.
func TestCommandShowsTheMistake(t *testing.T) {
	s := newSite(t, "-- pages/index.shugo --\n<main>\n  <p>café!</div>\n</main>")
	s.store.Close()
	_, errs, status := shugo(t, "build", s.root)
	want := "error: pages/index.shugo:2:11: </div> does not close <p> opened at 2:3\n     1 | <main>\n     2 |   <p>café!</div>\n       |           ^\n"
	if status != 1 || errs != want {
		t.Errorf("status %d\n got: %q\nwant: %q", status, errs, want)
	}
}

// Data that is imported, not read, is a part of the compiled project. An
// update notices that the project has changed and compiles it again, so the
// site never shows data that is out of date.
func TestUpdateCompilesAChangedProject(t *testing.T) {
	s := newSite(t, `
-- data/items.json --
["a"]
-- lib/data.ts --
import items from "../data/items.json"
export const getItems = (): string[] => items
-- pages/index.shugo --
---
import { getItems } from "@/lib/data"
---
<p>{getItems()}</p>
`)
	s.store.Close()
	if out, errs, status := shugo(t, "build", s.root); status != 0 {
		t.Fatalf("build: %s %s", out, errs)
	}
	if out, errs, _ := shugo(t, "update", s.root); !strings.Contains(out, "nothing has changed") || strings.Contains(errs, "compiling") {
		t.Errorf("an update with nothing new: %s %s", out, errs)
	}
	s.write("-- data/items.json --\n[\"b\"]")
	out, errs, status := shugo(t, "update", s.root)
	if status != 0 || !strings.Contains(errs, "the project has changed since it was compiled") {
		t.Fatalf("update: %d\n%s\n%s", status, out, errs)
	}
	shugo(t, "export", s.root)
	if html := s.read("_build/dist/index.html"); html != "<p>b</p>" {
		t.Errorf("the page: %s", html)
	}
	// A new template is a change of the project too.
	s.write("-- pages/more.shugo --\n<p>more</p>")
	if _, errs, _ := shugo(t, "update", s.root); !strings.Contains(errs, "the project has changed") {
		t.Errorf("an update after a new page: %s", errs)
	}
}
