package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/project"
)

// copyTree copies a project into a temporary folder, so that a test never
// writes into the sources.
func copyTree(t *testing.T, from string) string {
	t.Helper()
	to := t.TempDir()
	if err := os.CopyFS(to, os.DirFS(from)); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(to, project.Output))
	return to
}

func TestBuildExample(t *testing.T) {
	root := copyTree(t, "../../examples/basic")
	p, err := project.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	var patterns []string
	for _, r := range result.Site.Routes {
		patterns = append(patterns, r.Pattern)
	}
	if got, want := strings.Join(patterns, " "), "/ /404 /about /products/[slug]"; got != want {
		t.Errorf("routes: %s, want %s", got, want)
	}
	shared := map[string]bool{}
	for _, f := range result.Site.Files {
		shared[f.Path] = f.Shared
	}
	// The header reads nothing of the page; the card too; the layout shows
	// both and has no script that reads the page.
	for path, want := range map[string]bool{"components/Header.shugo": true, "components/Card.shugo": true, "pages/+layout.shugo": true, "pages/about.shugo": false} {
		if shared[path] != want {
			t.Errorf("%s: shared = %v, want %v", path, shared[path], want)
		}
	}
	if os.Getenv("SHUGO_SHOW") != "" {
		for _, bundle := range []string{LoadBundle, RenderBundle} {
			code, _ := os.ReadFile(filepath.Join(root, project.Output, bundle))
			t.Logf("\n%s", code)
		}
		site, _ := os.ReadFile(filepath.Join(root, project.Output, Site))
		t.Logf("\n%s", site)
	}
	if result.Site.Mounts["head"] == "" {
		t.Error("the style sheet is not linked")
	}
}
