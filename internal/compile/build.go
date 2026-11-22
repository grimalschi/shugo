package compile

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/jsrt"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/route"
	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/sourcemap"
	"github.com/grimalschi/shugo/internal/svelte"
)

// The files a compiled project consists of, under the output folder.
const (
	Bundle  = "eval.mjs"     // the code of the project: renders the pages it is asked for
	Site    = "site.json"    // the skeleton of the site: templates and routes
	Assets  = "assets"       // style sheets, published under AssetsURL
	Sources = "sources.json" // what the project was compiled from
)

// AssetsURL is where the files of the Assets folder are published.
const AssetsURL = "/_assets/"

// searchModule is the framework's module of searches; a project takes
// search from the module shugo.
const searchModule = "shugo:search"

// Result is a compiled project.
type Result struct {
	Site *skeleton.Site
	// ID changes whenever the compiled project does.
	ID       string
	Warnings []string
}

// Build compiles the project and writes the result under its output folder.
func Build(p *project.Project) (*Result, error) {
	paths, err := p.Templates()
	if err != nil {
		return nil, err
	}
	modules, err := p.PageModules()
	if err != nil {
		return nil, err
	}
	b := &builder{project: p, paths: paths, modules: modules, ids: map[string]int{}}
	for id, path := range paths {
		b.ids[path] = id
	}
	if err := b.compile(); err != nil {
		return nil, err
	}
	routes, err := b.routes()
	if err != nil {
		return nil, err
	}
	islands, err := b.buildIslands()
	if err != nil {
		return nil, err
	}
	bundled, err := b.bundle(routes, islands)
	if err != nil {
		return nil, err
	}
	shell, err := b.readShell()
	if err != nil {
		return nil, err
	}
	site := &skeleton.Site{
		Origin: p.Config.Site, Redirects: p.Config.Redirects, Routes: routes, Shell: shell, Assets: []string{},
		Search: bundled.search,
	}
	if site.Origin == "" {
		site.Origin = "http://localhost"
	}
	var warnings []string
	for i, path := range paths {
		site.Files = append(site.Files, skeleton.File{Path: path, Shared: bundled.shared[path]})
		warnings = append(warnings, b.units[i].warnings()...)
	}
	for _, path := range b.modules {
		site.Files = append(site.Files, skeleton.File{Path: path})
	}

	output := p.Path(project.Output)
	assets := filepath.Join(output, Assets)
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return nil, err
	}
	stylesheet, err := b.styles(bundled.styles)
	if err != nil {
		return nil, err
	}
	site.Assets = append(site.Assets, stylesheet.assets...)
	bundled.files = append(bundled.files, stylesheet.inputs...)
	if islands != nil {
		for _, file := range islands.assets {
			if err := writeHashedFile(file.Path, file.Contents); err != nil {
				return nil, err
			}
			site.Assets = append(site.Assets, filepath.Base(file.Path))
		}
		site.IslandPrograms = islands.programs
		bundled.files = append(bundled.files, islands.files...)
		warnings = append(warnings, islands.warnings...)
	}
	if sheet := stylesheet.sheet; sheet != nil {
		name := "styles." + digest(sheet)[:10] + ".css"
		if err := writeHashedFile(filepath.Join(assets, name), sheet); err != nil {
			return nil, err
		}
		site.Assets = append(site.Assets, name)
		site.Mounts = map[string]string{"head": `<link rel="stylesheet" href="` + AssetsURL + name + `">`}
	}
	slices.Sort(site.Assets)

	data, err := site.Encode()
	if err != nil {
		return nil, err
	}
	sources, err := json.Marshal(b.sources(bundled.files))
	if err != nil {
		return nil, err
	}
	for name, content := range map[string][]byte{Bundle: bundled.code, Bundle + ".map": bundled.smap, Site: data, Sources: sources} {
		if err := os.WriteFile(filepath.Join(output, name), content, 0o644); err != nil {
			return nil, err
		}
	}
	return &Result{Site: site, ID: digest(data, bundled.code), Warnings: append(warnings, bundled.warnings...)}, nil
}

// A running server can still render a previously built store. Publish new
// hashed files without truncating the assets it refers to.
func writeHashedFile(path string, contents []byte) error {
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, contents) {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".shugo-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0o644); err != nil {
		return err
	}
	if _, err := f.Write(contents); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// Load reads a project that Build has compiled before.
func Load(p *project.Project) (*Result, error) {
	data, err := os.ReadFile(p.Path(project.Output, Site))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("the project is not compiled yet: run `shugo build` first")
	} else if err != nil {
		return nil, err
	}
	code, err := os.ReadFile(p.Path(project.Output, Bundle))
	if err != nil {
		return nil, err
	}
	site, err := skeleton.Decode(data)
	if err != nil {
		return nil, err
	}
	return &Result{Site: site, ID: digest(data, code)}, nil
}

// sources is the list of what a project was compiled from: its templates,
// the modules among its pages, and every file that went into the bundle
// with a hash of its content; and the format of the skeleton, the version of
// Svelte and the revision of the compiler that made it.
type sources struct {
	Templates []string          `json:"templates"`
	Modules   []string          `json:"modules"`
	Files     map[string]string `json:"files"`
	Format    int               `json:"format"`
	Svelte    string            `json:"svelte"`
	Revision  int               `json:"revision"`
}

// The compiler can need a fresh build even when the format of the skeleton
// and the version of Svelte stay the same. Revision 2 adds the search build
// protocol to the generated bundle, along with the current compiler input
// tracking; 3 takes search from the module shugo and drops shugo:catalog; 4
// has Svelte compile the templates.
const sourcesRevision = 4

// sources hashes the files of the project that the compiled result depends on.
func (b *builder) sources(bundled []string) sources {
	out := sources{Templates: b.paths, Modules: b.modules, Files: map[string]string{}, Format: skeleton.Version, Svelte: svelte.Version, Revision: sourcesRevision}
	// A template and the shell count as they were read: they may have
	// changed since.
	for _, unit := range b.units {
		out.Files[unit.path] = digest([]byte(unit.source))
	}
	out.Files[project.Shell] = b.shell
	for _, name := range append(bundled, project.Settings) {
		if _, read := out.Files[name]; !read {
			out.Files[name] = hashFile(b.project.Path(filepath.FromSlash(name)))
		}
	}
	return out
}

// readShell reads the shell every page stands in: src/site.html, or the
// one of the framework. It keeps the hash of what it read, as hashFile
// writes it.
func (b *builder) readShell() (skeleton.Shell, error) {
	text := skeleton.DefaultShell
	data, err := os.ReadFile(b.project.Path(filepath.FromSlash(project.Shell)))
	switch {
	case err == nil:
		text, b.shell = string(data), digest(data)
	case !errors.Is(err, fs.ErrNotExist):
		return skeleton.Shell{}, err
	}
	shell, err := skeleton.ParseShell(text)
	if err != nil {
		return skeleton.Shell{}, fmt.Errorf("%s: %w", project.Shell, err)
	}
	return shell, nil
}

// hashFile returns a hash of the content of a file, or nothing if it cannot
// be read: a file that is not there is a state of its own.
func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return digest(data)
}

// Changed tells whether the sources of the project are not what it was
// compiled from: a template or a module among the pages was added or
// removed, or a file that went into the bundle has other content. Then the
// compiled project is out of date, and computing it again would show the
// old code.
func Changed(p *project.Project) (bool, error) {
	data, err := os.ReadFile(p.Path(project.Output, Sources))
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	var compiled sources
	if err := json.Unmarshal(data, &compiled); err != nil || compiled.Format != skeleton.Version ||
		compiled.Svelte != svelte.Version || compiled.Revision != sourcesRevision {
		return true, nil
	}
	templates, err := p.Templates()
	if err != nil {
		return false, err
	}
	modules, err := p.PageModules()
	if err != nil {
		return false, err
	}
	if !slices.Equal(templates, compiled.Templates) || !slices.Equal(modules, compiled.Modules) {
		return true, nil
	}
	for name, hash := range compiled.Files {
		if hashFile(p.Path(filepath.FromSlash(name))) != hash {
			return true, nil
		}
	}
	return false, nil
}

func digest(parts ...[]byte) string {
	sum := sha256.New()
	for _, part := range parts {
		sum.Write(part)
	}
	return hex.EncodeToString(sum.Sum(nil))[:32]
}

type builder struct {
	project *project.Project
	paths   []string       // the regular .svelte files, sorted
	ids     map[string]int // the number of every template: its place in paths
	units   []*unit
	// modules are the modules among the pages that give files, sorted.
	// Their numbers follow those of the templates.
	modules []string
	// shell is the hash of src/site.html as it was read; empty without it.
	shell string
}

// roleOf tells what a file is by where it lies.
func roleOf(path string) role {
	rel, inPages := strings.CutPrefix(path, project.Pages+"/")
	switch {
	case !inPages:
		return component
	case pathBase(rel) == route.Layout:
		return layout
	}
	if _, routed, err := route.Parse(rel); err == nil && !routed {
		return component
	}
	return page
}

// compile has Svelte compile every template. A compiler compiles one file
// at a time, in a millisecond or two, and takes longer to start: a large
// project has a few work at once.
func (b *builder) compile() error {
	b.units = make([]*unit, len(b.paths))
	if len(b.paths) == 0 {
		return nil
	}
	compilers := make(chan *svelte.Compiler, min(runtime.NumCPU(), 1+len(b.paths)/32, 4))
	for range cap(compilers) {
		c, err := svelte.Start(b.project.Runtime(), b.project.Root)
		if err != nil {
			close(compilers)
			for c := range compilers {
				c.Close()
			}
			return err
		}
		compilers <- c
	}
	defer func() {
		close(compilers)
		for c := range compilers {
			c.Close()
		}
	}()

	failures := make([]error, len(b.paths))
	var group sync.WaitGroup
	for id, path := range b.paths {
		group.Go(func() {
			c := <-compilers
			defer func() { compilers <- c }()
			b.units[id], failures[id] = b.compileTemplate(c, path)
		})
	}
	group.Wait()

	var errs Errors
	for _, err := range failures {
		switch err := err.(type) {
		case nil:
		case Errors:
			errs = append(errs, err...)
		default:
			return err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (b *builder) compileTemplate(c *svelte.Compiler, path string) (*unit, error) {
	source, err := os.ReadFile(b.project.Path(filepath.FromSlash(path)))
	if err != nil {
		return nil, err
	}
	compiled, err := c.Compile(path, string(source))
	var found svelte.Mistakes
	if errors.As(err, &found) {
		return nil, mistakes(path, found)
	}
	if err != nil {
		return nil, err
	}
	u := &unit{path: path, source: string(source), role: roleOf(path), Template: compiled}
	if u.role == layout && !u.Facts.Renders {
		return nil, Errors{{Path: path, Line: 1, Column: 1, Message: "a layout shows the page inside itself: put {@render children()} where the page goes, with let { children } = $props()"}}
	}
	return u, nil
}

// routes makes a route of every page and of every module among the pages,
// and finds the layouts around the pages.
func (b *builder) routes() ([]skeleton.Route, error) {
	// A file among the pages: a page of templates, or a module, which has
	// no unit and whose number follows those of the templates.
	type source struct {
		id   int
		path string
		unit *unit
	}
	var sources []source
	for id, path := range b.paths {
		if b.units[id].role == page {
			sources = append(sources, source{id, path, b.units[id]})
		}
	}
	for i, path := range b.modules {
		sources = append(sources, source{len(b.paths) + i, path, nil})
	}

	var errs Errors
	var out []skeleton.Route
	parsed := map[int]route.Route{} // by file
	shapes := map[string]string{}
	for _, f := range sources {
		errorf := func(format string, arguments ...any) {
			errs = append(errs, &Error{Path: f.path, Line: 1, Column: 1, Message: fmt.Sprintf(format, arguments...)})
		}
		p, _, err := route.Parse(strings.TrimPrefix(f.path, project.Pages+"/"))
		if err != nil {
			errorf("%s", err)
			continue
		}
		if other, taken := shapes[p.Shape()]; taken {
			errorf("this file and %s give the same addresses", other)
			continue
		}
		shapes[p.Shape()] = f.path

		r := skeleton.Route{File: f.id, Pattern: p.Pattern(), Segments: []skeleton.Segment{}, Extension: p.Extension, Layouts: []int{}}
		dynamic := false
		for _, s := range p.Segments {
			switch s.Kind {
			case route.Param:
				r.Segments = append(r.Segments, skeleton.Segment{Param: s.Name})
				dynamic = true
			case route.Rest:
				r.Segments = append(r.Segments, skeleton.Segment{Rest: s.Name})
				dynamic = true
			default:
				r.Segments = append(r.Segments, skeleton.Segment{Text: s.Name})
			}
		}
		// What a module exports is known when it runs.
		if f.unit != nil && dynamic && !f.unit.Facts.Paths {
			errorf(`the name of this page has a parameter, so its <script module> must say which pages there are: import { query } from "shugo"; export function getStaticPaths() { return query("products").select("slug").all().map((row) => ({ params: { slug: row.slug } })) }; without a data database, return the list itself: [{ params: { … } }]`)
		}
		// Every folder from src/pages/ down to a page may hold a layout. A
		// module gives files that are not HTML, and has none.
		for dir := project.Pages; f.unit != nil; {
			if layout, found := b.ids[dir+"/"+route.Layout]; found {
				r.Layouts = append(r.Layouts, layout)
			}
			rest := strings.TrimPrefix(f.path, dir+"/")
			name, _, deeper := strings.Cut(rest, "/")
			if !deeper {
				break
			}
			dir += "/" + name
		}
		parsed[f.id] = p
		out = append(out, r)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	route.Sort(out, func(r skeleton.Route) route.Route { return parsed[r.File] })
	return out, nil
}

// bundled is what the bundler made of the modules of all files.
type bundled struct {
	code, smap []byte
	files      []string        // the files of the project that went into the bundle
	shared     map[string]bool // which files can be computed once for all pages
	styles     []style         // the style sheets in the order they apply
	warnings   []string
	search     bool
}

// bundle links the modules of all files, the code they import and the
// library of the framework into one file.
func (b *builder) bundle(routes []skeleton.Route, islands *islandOutput) (*bundled, error) {
	root := b.project.Root
	// The entry names the module of every file, the templates and then the
	// modules among the pages: a page is computed from its route, and a
	// shared block from its name alone, so any file may be what a
	// computation starts from. The rest of what the run needs to know is in
	// the skeleton of the site.
	//
	// Pages and layouts come first, in their order: style sheets apply in
	// the order of imports, and that order starts from the pages.
	routed := map[int]bool{}
	for _, r := range routes {
		routed[r.File] = true
		for _, layout := range r.Layouts {
			routed[layout] = true
		}
	}
	var entry strings.Builder
	entry.WriteString("import { run } from \"shugo:runtime\";\n")
	var table []string
	files := append(slices.Clip(b.paths), b.modules...)
	for _, first := range []bool{true, false} {
		for id, path := range files {
			if routed[id] == first {
				fmt.Fprintf(&entry, "import * as $%d from %s;\n", id, quote("./"+path))
				table = append(table, fmt.Sprintf("[%d, $%d]", id, id))
			}
		}
	}
	fmt.Fprintf(&entry, "await run([%s]);\n", strings.Join(table, ", "))

	plugin := api.Plugin{Name: "shugo", Setup: func(build api.PluginBuild) {
		rejectOldTemplates(build)
		// The code Svelte compiled takes svelte/internal/server through
		// the framework, which gives it its own {@html}; the rest of Svelte
		// is the server runtime Shugo ships.
		build.OnResolve(api.OnResolveOptions{Filter: `^svelte/internal/server$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			if args.Namespace == "shugo" {
				return api.OnResolveResult{}, nil
			}
			return api.OnResolveResult{Path: svelteServer, Namespace: "shugo"}, nil
		})
		svelte.Runtime(build)
		// shugo is the module the code of a project imports from; shugo:…
		// are the framework's own, and a template is shown through
		// shugo:runtime. A search came from shugo:search, and from
		// shugo:catalog in the experiment with islands, before 22 November
		// 2026: such an import is told where search is now.
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo(:|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			path := args.Path
			switch {
			case path == "shugo":
				path = shugoModule
			case args.Namespace == "file" && (path == searchModule || path == "shugo:catalog"):
				return api.OnResolveResult{}, fmt.Errorf(`search comes from the module shugo: import { search } from "shugo", and names its table: search("products", { … })`)
			}
			return api.OnResolveResult{Path: path, Namespace: "shugo"}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "shugo"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			var contents string
			switch args.Path {
			case "shugo:runtime":
				contents = jsrt.Source
			case "shugo:entry":
				contents = entry.String()
			case "shugo:query":
				contents = jsrt.Query
			case shugoModule:
				contents = "export { query, paginate, page } from \"shugo:runtime\";\nexport { search } from \"shugo:search\";"
			case "shugo:wire":
				contents = jsrt.Wire
			case searchModule:
				contents = jsrt.Search
			case svelteServer:
				contents = "export * from \"svelte/internal/server\";\nexport { html } from \"shugo:runtime\";"
			case "shugo:island-eval":
				if islands == nil {
					return api.OnLoadResult{}, fmt.Errorf("the Svelte island evaluation module needs a %s import", project.IslandSuffix)
				}
				contents = islands.eval
			default:
				return api.OnLoadResult{}, fmt.Errorf(`there is no module %s: a project imports from "shugo"`, args.Path)
			}
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: root}, nil
		})
		// A template is two modules: the component Svelte compiled, under
		// the suffix ?svelte, and the module the others import, which shows
		// that component through the framework.
		build.OnLoad(api.OnLoadOptions{Filter: `\.svelte$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			rel := b.relative(args.Path)
			if project.IsIsland(rel) {
				if islands == nil || islands.names[rel].path == "" {
					return api.OnLoadResult{}, fmt.Errorf("import the island %s as a default component directly in a regular %s template", rel, project.TemplateExtension)
				}
				contents := `import { island } from "shugo:runtime"; import { islands } from "shugo:island-eval"; export default island(islands.get(` + quote(rel) + `));`
				return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: filepath.Dir(args.Path)}, nil
			}
			id, known := b.ids[rel]
			switch {
			case known:
			case !strings.HasPrefix(rel, project.Source+"/"):
				return api.OnLoadResult{}, fmt.Errorf("%s lies outside %s/: the templates of a project are in %s/, components in %s/components/", rel, project.Source, project.Source, project.Source)
			default:
				return api.OnLoadResult{}, fmt.Errorf("this file is not among the sources of the project")
			}
			var contents string
			if args.Suffix == compiledSuffix {
				u := b.units[id]
				contents = u.Code + "\n//# sourceMappingURL=data:application/json;base64," + base64.StdEncoding.EncodeToString([]byte(u.Map)) + "\n"
			} else {
				compiled := quote("./" + pathBase(rel) + compiledSuffix)
				contents = fmt.Sprintf("import { component } from \"shugo:runtime\";\nimport compiled from %s;\nexport * from %s;\nexport default component(%d, compiled);\n", compiled, compiled, id)
			}
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: filepath.Dir(args.Path)}, nil
		})
		// Style sheets do not go into JavaScript: they are collected and
		// built into a file of their own.
		build.OnLoad(api.OnLoadOptions{Filter: `\.css$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			empty := ""
			return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
		})
	}}
	result := api.Build(api.BuildOptions{
		EntryPoints:   []string{"shugo:entry"},
		Bundle:        true,
		Format:        api.FormatESModule,
		Platform:      api.PlatformNode,
		Target:        api.ESNext,
		Charset:       api.CharsetUTF8,
		Packages:      api.PackagesExternal,
		External:      []string{"bun:sqlite"}, // built into Bun; Node has node:sqlite
		Alias:         map[string]string{"@": filepath.Join(root, project.Source)},
		Plugins:       []api.Plugin{plugin},
		AbsWorkingDir: root,
		Outfile:       filepath.Join(root, project.Output, Bundle),
		Sourcemap:     api.SourceMapLinked,
		LegalComments: api.LegalCommentsNone,
		Metafile:      true,
		Write:         false,
		LogLevel:      api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		return nil, b.diagnoseAll(result.Errors)
	}
	out := &bundled{}
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, ".map") {
			out.smap = file.Contents
		} else {
			out.code = file.Contents
		}
	}
	for _, message := range result.Warnings {
		// A page without getStaticPaths is what most pages are.
		if message.ID != "import-is-undefined" {
			out.warnings = append(out.warnings, b.diagnoseAll([]api.Message{message}).Error())
		}
	}
	if err := b.graph(result.Metafile, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Report stale imports even when the old file has already been renamed.
func rejectOldTemplates(build api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: `\.shugo$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if !filepath.IsAbs(args.Path) && !strings.HasPrefix(args.Path, "./") &&
			!strings.HasPrefix(args.Path, "../") && !strings.HasPrefix(args.Path, "@/") {
			return api.OnResolveResult{}, nil // an npm package may have a dot in its name
		}
		return api.OnResolveResult{}, fmt.Errorf("update the import %q to %q; Shugo templates now use %s", args.Path,
			strings.TrimSuffix(args.Path, ".shugo")+project.TemplateExtension, project.TemplateExtension)
	})
}

// unitOf returns the template a module of the bundle is made of: the
// component Svelte compiled from it, or the module that shows it.
func (b *builder) unitOf(name string) *unit {
	if id, known := b.ids[strings.TrimSuffix(name, compiledSuffix)]; known {
		return b.units[id]
	}
	return nil
}

// relative returns the path of a file of the project as the units know it.
func (b *builder) relative(path string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(b.project.Root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// diagnoseAll turns the messages of the bundler into errors in the files
// of the project.
func (b *builder) diagnoseAll(messages []api.Message) error {
	var errs Errors
	for _, message := range messages {
		if message.Location == nil {
			return errors.New(message.Text)
		}
		errs = append(errs, b.diagnose(message))
	}
	return errs
}

// diagnose turns a message of the bundler about a place in a file into an
// error at that place. A place in the code Svelte compiled from a template
// becomes the place in the template that the code was compiled from.
func (b *builder) diagnose(message api.Message) *Error {
	at := message.Location
	path := strings.TrimSuffix(b.relative(at.File), compiledSuffix)
	id, known := b.ids[path]
	if !known {
		return &Error{Path: at.File, Line: at.Line, Column: at.Column + 1, Message: message.Text}
	}
	u := b.units[id]
	line, column := 1, 1
	if smap, err := sourcemap.Parse([]byte(u.Map)); err == nil {
		if _, sourceLine, sourceColumn, found := smap.Find(at.Line-1, at.Column); found {
			line, column = sourceLine+1, sourceColumn+1
		}
	}
	return &Error{Path: path, Line: line, Column: column, Message: message.Text}
}

// shugoModule is the module shugo as the bundler knows it.
const shugoModule = "shugo:module"

// svelteServer is svelte/internal/server as the code Svelte compiled takes it.
const svelteServer = "shugo:svelte-server"

// compiledSuffix names the component Svelte compiled from a template.
const compiledSuffix = "?svelte"

// graph reads from the bundler which module imports which. It gives the
// order of style sheets and tells which files can be computed once for all
// pages: those that do not depend on the page themselves and import nothing
// that does.
func (b *builder) graph(metafile string, out *bundled) error {
	var meta struct {
		Inputs map[string]struct {
			Imports []struct {
				Path string `json:"path"`
			} `json:"imports"`
		} `json:"inputs"`
		Outputs map[string]struct {
			Inputs map[string]struct {
				Bytes int `json:"bytesInOutput"`
			} `json:"inputs"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return err
	}
	// Esbuild's output graph excludes unused imports and tree-shaken code.
	for _, file := range meta.Outputs {
		out.search = out.search || file.Inputs["shugo:"+searchModule].Bytes > 0
	}
	names := make([]string, 0, len(meta.Inputs))
	for name := range meta.Inputs {
		names = append(names, name)
		// The library of the framework and the entry are not files.
		if !strings.Contains(name, ":") {
			out.files = append(out.files, name)
		}
	}
	sort.Strings(names)

	// A file that depends on the page passes that on to all that import it,
	// and so does a file that depends on the place a component stands at.
	// A template knows it of itself; another module of the project depends
	// on the page when it takes page or paginate from the module shugo, and
	// on the place when it takes the context of Svelte.
	own, placed := map[string]bool{}, map[string]bool{}
	for _, name := range names {
		if u := b.unitOf(name); u != nil {
			own[name], placed[name] = u.own(), u.placed()
			continue
		}
		if strings.Contains(name, ":") {
			continue // the framework and Svelte are not files of the project
		}
		for _, imported := range meta.Inputs[name].Imports {
			if strings.HasSuffix(imported.Path, shugoModule) || strings.HasPrefix(imported.Path, "svelte:") {
				source, err := os.ReadFile(b.project.Path(filepath.FromSlash(name)))
				if err != nil {
					return err
				}
				own[name], placed[name] = importsPage(string(source)), importsContext(string(source))
				break
			}
		}
	}
	for _, flags := range []map[string]bool{own, placed} {
		for changed := true; changed; {
			changed = false
			for _, name := range names {
				for _, imported := range meta.Inputs[name].Imports {
					if flags[imported.Path] && !flags[name] {
						flags[name], changed = true, true
					}
				}
			}
		}
	}
	// A layout is kept apart from the pages in any case: once for the site
	// when it reads nothing of the page.
	out.shared = map[string]bool{}
	for name := range meta.Inputs {
		if id, known := b.ids[name]; known {
			switch b.units[id].role {
			case layout:
				out.shared[name] = !own[name]
			case component:
				out.shared[name] = !own[name] && !placed[name]
			}
		}
	}

	// Style sheets apply in the order of imports, a module's own after
	// those of the modules it imports.
	seen := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, imported := range meta.Inputs[name].Imports {
			visit(imported.Path)
		}
		if strings.HasSuffix(name, ".css") {
			out.styles = append(out.styles, style{path: b.project.Path(filepath.FromSlash(name))})
		} else if u := b.unitOf(name); u != nil && u.CSS != "" && !strings.HasSuffix(name, compiledSuffix) {
			out.styles = append(out.styles, style{text: u.CSS, from: u.path})
		}
	}
	for _, name := range names {
		if strings.HasSuffix(name, "shugo:entry") {
			visit(name)
		}
	}
	return nil
}
