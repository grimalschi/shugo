package compile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/svelte"
)

// islandParts are the parts of the graphs of the islands that Svelte has
// compiled: a component or a module with runes, compiled once for the
// browser and for the server, both bundles take it from here.
type islandParts struct {
	mu    sync.Mutex
	parts map[string]islandPart // by the path from the root of the project
}

type islandPart struct {
	code *svelte.IslandCode
	err  error // the mistakes Svelte found in the file
}

type islandEntry struct {
	path     string // source-relative component path, also the island ID
	url, css string
}

type islandOutput struct {
	names    map[string]islandEntry
	eval     string
	programs map[string][]skeleton.IslandOp
	assets   []api.OutputFile
	files    []string
	warnings []string
}

// buildIslands registers the island entries that regular templates import.
func (b *builder) buildIslands() (*islandOutput, error) {
	roots := map[string]bool{}
	for _, u := range b.units {
		for _, island := range u.Facts.Islands {
			name := island.Source
			errorf := func(format string, arguments ...any) error {
				return Errors{{Path: u.path, Line: 1, Column: 1, Message: fmt.Sprintf(format, arguments...)}}
			}
			var path string
			switch {
			case strings.HasPrefix(name, "@/"):
				path = b.project.Path(project.Source, filepath.FromSlash(name[2:]))
			case strings.HasPrefix(name, "./"), strings.HasPrefix(name, "../"):
				path = b.project.Path(filepath.Dir(u.path), filepath.FromSlash(name))
			default:
				return nil, errorf("import the island %q using a relative path or @/ from src/", name)
			}
			rel := b.relative(path)
			if !strings.HasPrefix(rel, project.Source+"/") {
				return nil, errorf("the island %q must be inside src/", name)
			}
			if _, err := os.Stat(path); err != nil {
				return nil, errorf("the island %q could not be read: %v", name, err)
			}
			roots[rel] = roots[rel] || island.Request
		}
	}
	if len(roots) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(roots))
	for path := range roots {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	browserSources := map[string]string{}
	var browserEntries []api.EntryPoint
	for i, path := range paths {
		entry := "shugo-island-entry:" + path
		browserSources[entry] = browserIsland(b.project.Path(filepath.FromSlash(path)))
		browserEntries = append(browserEntries, api.EntryPoint{InputPath: entry, OutputPath: fmt.Sprintf("island-%d", i+1)})
	}
	options := b.islandOptions()
	options.EntryPointsAdvanced = browserEntries
	options.Platform = api.PlatformBrowser
	options.Outdir = b.project.Path(project.Output, Assets)
	options.EntryNames = "[name]-[hash]"
	options.ChunkNames = "island-shared-[hash]"
	options.AssetNames = "island-file-[name]-[hash]"
	options.Splitting = true
	options.Loader = assetLoaders
	options.Plugins = []api.Plugin{b.sveltePlugin(browserSources, true)}
	browser := api.Build(options)
	web, err := islandMetadata(browser)
	if err != nil {
		return nil, err
	}
	out := &islandOutput{names: map[string]islandEntry{}, assets: browser.OutputFiles, files: web.inputs(), warnings: islandWarnings(browser.Warnings)}
	for file, entry := range web.Outputs {
		const prefix = "shugo-island:shugo-island-entry:"
		if path, ok := strings.CutPrefix(entry.EntryPoint, prefix); ok {
			var css string
			if entry.CSSBundle != "" {
				css = AssetsURL + filepath.Base(entry.CSSBundle)
			}
			out.names[path] = islandEntry{path: path, url: AssetsURL + filepath.Base(file), css: css}
		}
	}
	if len(out.names) != len(paths) {
		return nil, fmt.Errorf("Svelte bundler: made %d browser entries for %d islands", len(out.names), len(paths))
	}

	const serverEntry = "shugo-island-entry:eval"
	serverSources := map[string]string{serverEntry: b.evalIslands(paths, out.names)}
	options = b.islandOptions()
	options.EntryPoints = []string{serverEntry}
	options.Platform = api.PlatformNode
	options.Outfile = b.project.Path(project.Output, "island-eval.mjs")
	options.Plugins = []api.Plugin{b.sveltePlugin(serverSources, false)}
	server := api.Build(options)
	ssr, err := islandMetadata(server)
	if err != nil {
		return nil, err
	}
	if len(server.OutputFiles) != 1 {
		return nil, fmt.Errorf("Svelte bundler: the evaluation code should be one file, got %d", len(server.OutputFiles))
	}
	out.eval = string(server.OutputFiles[0].Contents)
	out.programs, err = b.islandPrograms(paths, roots)
	if err != nil {
		return nil, err
	}
	out.files = append(out.files, ssr.inputs()...)
	slices.Sort(out.files)
	out.files = slices.Compact(out.files)
	return out, nil
}

const noShugo = "__shugo_island_runtime_unavailable_3ef1b7c4__"

func (b *builder) islandOptions() api.BuildOptions {
	return api.BuildOptions{
		Bundle: true, Format: api.FormatESModule, Target: api.ESNext, Charset: api.CharsetUTF8,
		Conditions: []string{"production"}, // Svelte and packages without what only runs in development
		Alias:      map[string]string{"@": b.project.Path(project.Source)}, Define: map[string]string{"Shugo": noShugo},
		AbsWorkingDir: b.project.Root, LegalComments: api.LegalCommentsNone, Metafile: true, LogLevel: api.LogLevelSilent,
		MinifyIdentifiers: true, MinifySyntax: true, MinifyWhitespace: true,
	}
}

// sveltePlugin has Svelte compile the graph of an island, for the browser
// or for the server, ordinary components it shares with pages included; the
// modules of Svelte come from the package svelte the build runs.
func (b *builder) sveltePlugin(entries map[string]string, client bool) api.Plugin {
	return api.Plugin{Name: "shugo-svelte", Setup: func(build api.PluginBuild) {
		rejectOldTemplates(build)
		b.sveltePackage.Runtime(build, client)
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo-island-`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{Path: args.Path, Namespace: "shugo-island"}, nil
		})
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo(:|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{}, fmt.Errorf("Svelte islands cannot import %s: pass data through JSON props or an endpoint", args.Path)
		})
		b.checkPackages(build, false)
		build.OnLoad(api.OnLoadOptions{Filter: `^shugo-island-entry:`, Namespace: "shugo-island"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			contents := entries[args.Path]
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: b.project.Root}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `^shugo-island-css:`, Namespace: "shugo-island"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			// esbuild loads this module after its component.
			path := strings.TrimPrefix(args.Path, "shugo-island-css:")
			part, err := b.islandPart(path)
			if err != nil {
				return api.OnLoadResult{}, err
			}
			return api.OnLoadResult{Contents: &part.CSS, Loader: api.LoaderCSS, ResolveDir: filepath.Dir(path)}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `\.svelte(\.js|\.ts)?$`, Namespace: "file"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			return b.svelteLoad(args.Path, client)
		})
		if !client {
			build.OnLoad(api.OnLoadOptions{Filter: `\.css$`, Namespace: "file"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				empty := ""
				return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
			})
		}
	}}
}

// islandPart compiles a component or a module with runes of the graph of an
// island, once for both bundles. Svelte reads JavaScript alone, so a
// .svelte.ts file loses its types first.
func (b *builder) islandPart(path string) (*svelte.IslandCode, error) {
	rel := b.relative(path)
	b.islands.mu.Lock()
	defer b.islands.mu.Unlock()
	if part, done := b.islands.parts[rel]; done {
		return part.code, part.err
	}
	code, err := b.compileIslandPart(path, rel)
	if b.islands.parts == nil {
		b.islands.parts = map[string]islandPart{}
	}
	b.islands.parts[rel] = islandPart{code, err}
	return code, err
}

func (b *builder) compileIslandPart(path, rel string) (*svelte.IslandCode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(data)
	if strings.HasSuffix(path, ".svelte.ts") {
		stripped := api.Transform(source, api.TransformOptions{Loader: api.LoaderTS, Target: api.ESNext, Sourcefile: rel, LogLevel: api.LogLevelSilent})
		if len(stripped.Errors) > 0 {
			return nil, fmt.Errorf("%s: %s", rel, stripped.Errors[0].Text)
		}
		source = string(stripped.Code)
	}
	compilers, err := b.compilers()
	if err != nil {
		return nil, err
	}
	return compilers.CompileIsland(rel, source, !strings.HasSuffix(path, ".svelte"))
}

func (b *builder) svelteLoad(path string, client bool) (api.OnLoadResult, error) {
	rel := b.relative(path)
	if id, known := b.ids[rel]; known && b.units[id].role != component {
		return api.OnLoadResult{}, fmt.Errorf("an island cannot import the page or layout %s; extract a reusable Svelte component", rel)
	}
	part, err := b.islandPart(path)
	var found svelte.Mistakes
	if errors.As(err, &found) {
		return api.OnLoadResult{Errors: messagesOf(rel, found, nil)}, nil
	}
	if err != nil {
		return api.OnLoadResult{}, err
	}
	code := part.Server
	var warnings []api.Message
	if client {
		code = part.Client
		if part.CSS != "" {
			code += "\nimport " + quote("shugo-island-css:"+path) + ";\n"
		}
		warnings = messagesOf(rel, part.Warnings, nil)
	}
	if strings.Contains(code, "Shugo") && readsShugo(code) {
		return api.OnLoadResult{Errors: []api.Message{{Text: "a component inside a Svelte island cannot read Shugo; pass data as a prop",
			Location: &api.Location{File: rel, Line: 1}}}}, nil
	}
	return api.OnLoadResult{Contents: &code, Loader: api.LoaderJS, ResolveDir: filepath.Dir(path), Warnings: warnings}, nil
}

// Check a compiled component's free JavaScript references. A prop or local
// named Shugo, a property on another object, and strings are ordinary values.
func readsShugo(code string) bool {
	parsed := api.Transform(code, api.TransformOptions{Loader: api.LoaderJS, Target: api.ESNext,
		Define: map[string]string{"Shugo": noShugo}, LogLevel: api.LogLevelSilent})
	return bytes.Contains(parsed.Code, []byte(noShugo))
}

type islandMeta struct {
	Inputs  map[string]json.RawMessage
	Outputs map[string]struct {
		EntryPoint string `json:"entryPoint"`
		CSSBundle  string `json:"cssBundle"`
	}
}

func (m *islandMeta) inputs() []string {
	var files []string
	for file := range m.Inputs {
		if !strings.Contains(file, ":") {
			files = append(files, file)
		}
	}
	return files
}

func islandMetadata(result api.BuildResult) (*islandMeta, error) {
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("Svelte islands: %s", strings.Join(islandWarnings(result.Errors), "\n"))
	}
	for _, file := range result.OutputFiles {
		if (strings.HasSuffix(file.Path, ".js") || strings.HasSuffix(file.Path, ".mjs")) && bytes.Contains(file.Contents, []byte(noShugo)) {
			return nil, fmt.Errorf("a module imported inside a Svelte island cannot read Shugo; pass data as a prop")
		}
	}
	var meta islandMeta
	if err := json.Unmarshal([]byte(result.Metafile), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func islandWarnings(messages []api.Message) []string {
	var warnings []string
	for _, m := range messages {
		if m.Location == nil {
			warnings = append(warnings, m.Text)
		} else {
			at := m.Location
			warnings = append(warnings, (&Error{Path: at.File, Line: at.Line, Column: at.Column + 1, Message: m.Text}).Error())
		}
	}
	return warnings
}

func browserIsland(component string) string {
	return `import { hydrate } from 'svelte';
import Component from ` + quote(component) + `;

const entry = new URL(import.meta.url).pathname;
for (const island of document.querySelectorAll('[data-shugo-island]')) {
  if (island.dataset.shugoIsland !== entry) continue;
  const target = island.querySelector(':scope > [data-shugo-target]');
  const props = island.querySelector(':scope > script[data-shugo-props]');
  hydrate(Component, { target, props: JSON.parse(props.textContent), recover: false });
  island.dataset.shugoHydrated = '';
}
`
}

func (b *builder) evalIslands(paths []string, islands map[string]islandEntry) string {
	var entry strings.Builder
	entry.WriteString("import { render } from 'svelte/server';\n")
	for i, path := range paths {
		fmt.Fprintf(&entry, "import Component%d from %s;\n", i, quote(b.project.Path(filepath.FromSlash(path))))
	}
	entry.WriteString("export const islands = new Map([\n")
	for i, path := range paths {
		island := islands[path]
		fmt.Fprintf(&entry, "[%s, { id: %s, url: %s, css: %s, render: props => render(Component%d, { props }) }],\n",
			quote(island.path), quote(island.path), quote(island.url), quote(island.css), i)
	}
	entry.WriteString("]);\n")
	return entry.String()
}

// islandPrograms makes the programs of the islands bound to a request, and
// of the components they show: Go builds their HTML by them on request.
func (b *builder) islandPrograms(paths []string, request map[string]bool) (map[string][]skeleton.IslandOp, error) {
	compilers, err := b.compilers()
	if err != nil {
		return nil, err
	}
	programs := make(map[string][]skeleton.IslandOp)
	visiting := make(map[string]bool)
	var add func(string) error
	add = func(path string) error {
		if visiting[path] {
			return fmt.Errorf("%s: request-time island components cannot call themselves recursively", path)
		}
		if _, done := programs[path]; done {
			return nil
		}
		part, err := b.islandPart(b.project.Path(filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		lowered, err := compilers.LowerIsland(part.Server)
		var found svelte.Mistakes
		if errors.As(err, &found) {
			// The place of the mistake in the code of Svelte is a place in
			// the file the code was compiled from.
			m := found[0]
			if origin, err := sourcemap.Parse([]byte(part.Map)); err == nil {
				if _, line, column, ok := origin.Find(m.Line-1, m.Column-1); ok {
					return Errors{{Path: path, Line: line + 1, Column: column + 1, Message: "request-time island: " + m.Text}}
				}
			}
			return fmt.Errorf("%s: request-time island: %s (generated SSR %d:%d)", path, m.Text, m.Line, m.Column)
		}
		if err != nil {
			return err
		}
		var program []skeleton.IslandOp
		if err := json.Unmarshal(lowered, &program); err != nil {
			return err
		}
		visiting[path] = true
		defer delete(visiting, path)
		if err := b.resolveIslandCalls(path, program, add); err != nil {
			return err
		}
		programs[path] = program
		return nil
	}
	for _, path := range paths {
		if request[path] {
			if err := add(path); err != nil {
				return nil, err
			}
		}
	}
	return programs, nil
}

func (b *builder) resolveIslandCalls(parent string, ops []skeleton.IslandOp, add func(string) error) error {
	for i := range ops {
		op := &ops[i]
		if op.Kind == "component" {
			var path string
			switch {
			case strings.HasPrefix(op.Text, "@/"):
				path = b.project.Path(project.Source, filepath.FromSlash(op.Text[2:]))
			case strings.HasPrefix(op.Text, "./"), strings.HasPrefix(op.Text, "../"):
				path = b.project.Path(filepath.Dir(parent), filepath.FromSlash(op.Text))
			default:
				return fmt.Errorf("%s: request-time child %q needs a relative path or @/", parent, op.Text)
			}
			op.Text = b.relative(path)
			if !strings.HasPrefix(op.Text, project.Source+"/") {
				return fmt.Errorf("%s: request-time child %q must be inside src/", parent, op.Text)
			}
			if err := add(op.Text); err != nil {
				return err
			}
		}
		if err := b.resolveIslandCalls(parent, op.Body, add); err != nil {
			return err
		}
		if err := b.resolveIslandCalls(parent, op.Alt, add); err != nil {
			return err
		}
	}
	return nil
}
