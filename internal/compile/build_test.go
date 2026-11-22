package compile

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestBuildParserUsesProjectRuntimeDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runtime fixture uses a POSIX shell")
	}
	p := templateProject(t, map[string]string{"src/pages/index.svelte": "<p>hello</p>"})
	bin, err := exec.LookPath(p.Runtime())
	if err != nil {
		t.Fatal(err)
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHUGO_TEST_PARSER_RUNTIME", bin)
	if err := os.WriteFile(p.Path("runtime"), []byte("#!/bin/sh\nexec \"$SHUGO_TEST_PARSER_RUNTIME\" \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p.Config.Runtime = "./runtime"
	t.Chdir(t.TempDir())
	if err := os.Mkdir("relative-temp", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", "relative-temp")
	if _, err := Build(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Path("runtime"), []byte("#!/bin/sh\nprintf '%s\\n' '{\"ready\":false}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(p); err == nil || !strings.Contains(err.Error(), "missing startup acknowledgement") {
		t.Fatalf("expected a parser startup error, got %v", err)
	}
	files, err := os.ReadDir("relative-temp")
	if err != nil || len(files) != 0 {
		t.Fatalf("parser leaked its extracted compiler: %v, %v", files, err)
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
	// The embedded parser works without npm packages. An unused island
	// beside a page does not enter either compilation graph.
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

func TestStylesTrackTransitiveInputs(t *testing.T) {
	for _, dep := range []struct{ name, replacement string }{
		{"src/styles/nested.css", `.mark { color: blue; background: url("./mark.svg"); }`},
		{"src/styles/mark.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="8"/>`},
	} {
		t.Run(filepath.Base(dep.name), func(t *testing.T) {
			p := templateProject(t, map[string]string{
				"src/pages/index.svelte": `<script>import '../styles/main.css'</script><p class="mark">hello</p>`,
				"src/styles/main.css":    `@import "./nested.css";`,
				"src/styles/nested.css":  `.mark { color: red; background: url("./mark.svg"); }`,
				"src/styles/mark.svg":    `<svg xmlns="http://www.w3.org/2000/svg"/>`,
			})
			original, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p.Path(filepath.FromSlash(dep.name)), []byte(dep.replacement), 0o644); err != nil {
				t.Fatal(err)
			}
			if changed, err := Changed(p); err != nil || !changed {
				t.Fatalf("%s changed: changed = %v, error = %v", dep.name, changed, err)
			}
			next, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			if original.Site.Mounts["head"] == next.Site.Mounts["head"] {
				t.Fatal("published stylesheet needs a new content hash")
			}
		})
	}
}

func TestManifestHashesParsedTemplate(t *testing.T) {
	const name = "src/pages/index.svelte"
	const parsed = "<p>before</p>"
	p := templateProject(t, map[string]string{name: parsed})
	b := &builder{project: p, paths: []string{name}, ids: map[string]int{name: 0}}
	if err := b.compile(); err != nil {
		t.Fatal(err)
	}
	// The source changed after parsing and before the manifest was made.
	if err := os.WriteFile(p.Path(name), []byte("<p>after</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := b.sources([]string{name})
	if got := manifest.Files[name]; got != digest([]byte(parsed)) {
		t.Fatalf("manifest hashed a template different from the compiled template: %q", got)
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

func TestManifestDetectsCompilerChanges(t *testing.T) {
	p := templateProject(t, map[string]string{"src/pages/index.svelte": "<p>hello</p>"})
	if _, err := Build(p); err != nil {
		t.Fatal(err)
	}
	manifest := p.Path(project.Output, Sources)
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var previous sources
	if err := json.Unmarshal(data, &previous); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*sources)
	}{
		{"no parser stamp", func(s *sources) { s.Parser = "" }},
		{"old input schema", func(s *sources) { s.Revision = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			old := previous
			test.edit(&old)
			data, err := json.Marshal(old)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifest, data, 0o644); err != nil {
				t.Fatal(err)
			}
			if changed, err := Changed(p); err != nil || !changed {
				t.Fatalf("previous compiler manifest: changed = %v, error = %v", changed, err)
			}
		})
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
