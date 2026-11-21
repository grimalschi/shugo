package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The templates of Shugo are written as components of Svelte 5 and give the
// HTML that the server renderer of Svelte gives for them. test/svelte holds
// the reference components of the 30 tasks of the template trial, with what
// Svelte 5.59.1 renders for every dataset of a task; SHUGO_SVELTE_TASK=03 runs the
// tasks whose name has 03.
//
// Two differences are taken out before pages are compared. The class that
// scopes the styles of a component is named after its file, which is not
// the file Svelte compiled, and Shugo puts it on every element of the
// component where Svelte puts it on those its selectors match. The marks of
// hydration Svelte writes, <!--…--> and the <!> of a <select>, are taken out
// of what it renders when the references are made, and here.
func TestSvelteTasks(t *testing.T) {
	tasks, err := filepath.Glob("svelte/*/task.json")
	if err != nil {
		t.Fatal(err)
	}
	only := os.Getenv("SHUGO_SVELTE_TASK")
	for _, path := range tasks {
		dir := filepath.Dir(path)
		if only != "" && !strings.Contains(dir, only) {
			continue
		}
		t.Run(filepath.Base(dir), func(t *testing.T) { svelteTask(t, dir) })
	}
}

func svelteTask(t *testing.T, dir string) {
	var task struct {
		Entry    string            `json:"entry"`
		Datasets []json.RawMessage `json:"datasets"`
	}
	var expected []struct {
		Body string `json:"body"`
		Head string `json:"head"`
	}
	read := func(name string, into any) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, into); err != nil {
			t.Fatal(err)
		}
	}
	read("task.json", &task)
	read("expected.json", &expected)

	var files strings.Builder
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	helpers := false
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".shugo") && name != "helpers.js" {
			continue
		}
		helpers = helpers || name == "helpers.js"
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&files, "-- components/%s --\n%s\n", name, data)
	}
	// A page for every dataset shows the component with it and nothing else.
	for i, data := range task.Datasets {
		fmt.Fprintf(&files, "-- pages/d%d.shugo --\n<script>\n  import Entry from \"../components/%s.shugo\";\n", i, task.Entry)
		spread := "{...props}"
		if helpers {
			files.WriteString("  import helpers from \"../components/helpers.js\";\n")
			spread += " {...helpers}"
		}
		// "</script>" inside the code would end the script, in Svelte too.
		fmt.Fprintf(&files, "  const props = %s;\n</script>\n<Entry %s />\n", strings.ReplaceAll(string(data), "</", "<\\/"), spread)
	}
	s := newSite(t, files.String())
	if _, err := s.tryBuild(); err != nil {
		t.Fatal(err)
	}
	for i := range task.Datasets {
		got := unscoped(s.page(fmt.Sprintf("/d%d", i)))
		if want := unscoped(expected[i].Body); got != want {
			t.Errorf("dataset %d:\n got %s\nwant %s", i, got, want)
		}
	}
}

var (
	scopeClassName = regexp.MustCompile(` ?\bsvelte-[a-z0-9]+\b`)
	emptyClass     = regexp.MustCompile(` class=""`)
)

// unscoped takes the classes that scope styles out of a page.
func unscoped(html string) string {
	html = strings.ReplaceAll(html, "<!>", "")
	html = scopeClassName.ReplaceAllString(html, "")
	html = strings.ReplaceAll(html, `class=" `, `class="`)
	return emptyClass.ReplaceAllString(html, "")
}
