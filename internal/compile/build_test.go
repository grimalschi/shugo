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
	for path, want := range map[string]bool{"src/components/Header.svelte": true, "src/components/Card.svelte": true, "src/pages/+layout.svelte": true, "src/pages/about.svelte": false} {
		if shared[path] != want {
			t.Errorf("%s: shared = %v, want %v", path, shared[path], want)
		}
	}
	if os.Getenv("SHUGO_SHOW") != "" {
		code, _ := os.ReadFile(filepath.Join(root, project.Output, Bundle))
		t.Logf("\n%s", code)
		site, _ := os.ReadFile(filepath.Join(root, project.Output, Site))
		t.Logf("\n%s", site)
	}
	if result.Site.Mounts["head"] == "" {
		t.Error("the style sheet is not linked")
	}
}

func TestRegularSvelteDependencies(t *testing.T) {
	p := templateProject(t, map[string]string{
		"src/pages/index.svelte": `<script>import Frame from '../components/Frame.svelte'</script><Frame />`,
		"src/components/Frame.svelte": `
<script>import * as ui from '../lib/ui.ts'</script>
<main><ui.Label /></main>`,
		"src/lib/ui.ts": `export { default as Label } from '../components/Label.svelte'`,
		"src/components/Label.svelte": `
<script>
import './label.css'
const path = Shugo.url.pathname
</script>
<p class="label">{path}</p>
<style>p { color: red; }</style>`,
		"src/components/label.css":             `.label { font-style: italic; }`,
		"src/pages/Unreferenced.island.svelte": `<script>INVALID JS(`,
	})
	// No Svelte compiler process is needed for a project with ordinary
	// templates, even if it keeps an unused island file beside its pages.
	p.Config.Runtime = p.Path("missing-runtime")
	built, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Site.Routes) != 1 || built.Site.Routes[0].Pattern != "/" || len(built.Site.IslandPrograms) != 0 {
		t.Fatalf("unexpected route or island entry: %+v", built.Site)
	}
	for _, file := range built.Site.Files {
		if project.IsIsland(file.Path) || file.Shared {
			t.Errorf("islands are absent and page-dependent components are not shared: %+v", file)
		}
	}
	if len(built.Site.Assets) != 1 || !strings.HasSuffix(built.Site.Assets[0], ".css") {
		t.Fatalf("expected only the ordinary style asset: %v", built.Site.Assets)
	}
	css, err := os.ReadFile(p.Path(project.Output, Assets, built.Site.Assets[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), "color:red") || !strings.Contains(string(css), "font-style:italic") {
		t.Fatalf("expected styles from the component and its CSS import: %s", css)
	}
	if changed, err := Changed(p); err != nil || changed {
		t.Fatalf("fresh build: changed = %v, error = %v", changed, err)
	}
	if err := os.WriteFile(p.Path("src/lib/ui.ts"), []byte("// a forwarder modification"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, err := Changed(p); err != nil || !changed {
		t.Fatalf("changed import: changed = %v, error = %v", changed, err)
	}
}

func TestStaleTemplateImport(t *testing.T) {
	p := templateProject(t, map[string]string{
		"src/pages/index.svelte":     `<script>import Card from '../components/Card.shugo'</script><Card />`,
		"src/components/Card.svelte": `<p>new</p>`,
	})
	if _, err := Build(p); err == nil || !strings.Contains(err.Error(), `update the import "../components/Card.shugo" to "../components/Card.svelte"`) {
		t.Fatalf("expected a migration diagnostic, got %v", err)
	}
}

func templateProject(t *testing.T, files map[string]string) *project.Project {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := project.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
