package project

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTemplates(t *testing.T) {
	p := templateProject(t,
		"src/pages/index.svelte",
		"src/pages/+layout.svelte",
		"src/pages/(shop)/_parts/Card.svelte",
		"src/pages/(shop)/cart.island.svelte",
		"src/components/Card.svelte",
		"src/components/Cart.island.svelte",
		"src/styles/main.css",
		"src/node_modules/pkg/Page.shugo",
		"src/.archive/Page.shugo",
		"node_modules/pkg/Page.svelte",
	)
	files, err := p.Templates()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"src/components/Card.svelte",
		"src/pages/(shop)/_parts/Card.svelte",
		"src/pages/+layout.svelte",
		"src/pages/index.svelte",
	}
	if !slices.Equal(files, want) {
		t.Fatalf("got %q, want %q", files, want)
	}
}

func TestLegacyTemplatesRequireRename(t *testing.T) {
	// Opening the project still works. Serving a compiled site does not
	// enumerate its source templates.
	p := templateProject(t, "src/pages/index.shugo")
	_, err := p.Templates()
	diagnostic := "src/pages/index.shugo: rename this template to src/pages/index.svelte and update its imports"
	if err == nil || !strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("got %v, want diagnostic %q", err, diagnostic)
	}
}

func templateProject(t *testing.T, names ...string) *Project {
	t.Helper()
	root := t.TempDir()
	for _, name := range names {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
