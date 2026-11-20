package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/jsrt"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/route"
	"github.com/grimalschi/shugo/internal/syntax"
)

// The files a compiled project consists of, under the output folder.
const (
	Bundle  = "eval.mjs"     // loads the data and computes the values of all pages
	Site    = "site.json"    // the program of the site: templates and routes
	Assets  = "assets"       // style sheets, published under AssetsURL
	Sources = "sources.json" // what the project was compiled from
)

// AssetsURL is where the files of the Assets folder are published.
const AssetsURL = "/_assets/"

// Result is a compiled project.
type Result struct {
	Site *program.Site
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
	b := &builder{project: p, paths: paths, ids: map[string]int{}}
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
	bundled, err := b.bundle(routes)
	if err != nil {
		return nil, err
	}
	site := &program.Site{Origin: p.Config.Site, Redirects: p.Config.Redirects, Routes: routes}
	if site.Origin == "" {
		site.Origin = "http://localhost"
	}
	for i, path := range paths {
		site.Files = append(site.Files, program.File{Path: path, Parts: b.units[i].parts, Slots: b.units[i].outlets, Shared: bundled.shared[path]})
	}

	output := p.Path(project.Output)
	assets := filepath.Join(output, Assets)
	if err := os.RemoveAll(assets); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return nil, err
	}
	sheet, err := b.styles(bundled.styles)
	if err != nil {
		return nil, err
	}
	if sheet != nil {
		name := "styles." + digest(sheet)[:10] + ".css"
		if err := os.WriteFile(filepath.Join(assets, name), sheet, 0o644); err != nil {
			return nil, err
		}
		site.Mounts = map[string]string{"head": `<link rel="stylesheet" href="` + AssetsURL + name + `">`}
	}

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
	return &Result{Site: site, ID: digest(data, bundled.code), Warnings: bundled.warnings}, nil
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
	site, err := program.Decode(data)
	if err != nil {
		return nil, err
	}
	return &Result{Site: site, ID: digest(data, code)}, nil
}

// sources is the list of what a project was compiled from: its templates,
// and every file that went into the bundle with a hash of its content.
type sources struct {
	Templates []string          `json:"templates"`
	Files     map[string]string `json:"files"`
}

// sources hashes the files of the project that the compiled result depends on.
func (b *builder) sources(bundled []string) sources {
	out := sources{Templates: b.paths, Files: map[string]string{}}
	for _, name := range append(bundled, project.Settings) {
		out.Files[name] = hashFile(b.project.Path(filepath.FromSlash(name)))
	}
	return out
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
// compiled from: a template was added or removed, or a file that went into
// the bundle has other content. Then the compiled project is out of date, and
// computing it again would show the old code.
func Changed(p *project.Project) (bool, error) {
	data, err := os.ReadFile(p.Path(project.Output, Sources))
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	var compiled sources
	if err := json.Unmarshal(data, &compiled); err != nil {
		return true, nil
	}
	templates, err := p.Templates()
	if err != nil {
		return false, err
	}
	if !slices.Equal(templates, compiled.Templates) {
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
	paths   []string       // the .shugo files, sorted
	ids     map[string]int // the number of every file: its place in paths
	units   []*unit
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

// compile compiles every file, several at a time.
func (b *builder) compile() error {
	b.units = make([]*unit, len(b.paths))
	failures := make([]error, len(b.paths))
	next := make(chan int)
	var group sync.WaitGroup
	for range runtime.NumCPU() {
		group.Go(func() {
			for id := range next {
				path := b.paths[id]
				source, err := os.ReadFile(b.project.Path(filepath.FromSlash(path)))
				if err == nil {
					b.units[id], err = compileFile(path, string(source), id, roleOf(path))
				}
				failures[id] = err
			}
		})
	}
	for id := range b.paths {
		next <- id
	}
	close(next)
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

// routes makes a route of every page and finds the layouts around it.
func (b *builder) routes() ([]program.Route, error) {
	var errs Errors
	var out []program.Route
	segments := map[int][]route.Segment{} // by file
	shapes := map[string]string{}
	for id, path := range b.paths {
		u := b.units[id]
		if u.role != page {
			continue
		}
		parsed, _, err := route.Parse(strings.TrimPrefix(path, project.Pages+"/"))
		if err != nil {
			errs = append(errs, u.file.Errorf(0, "%s", err))
			continue
		}
		if other, taken := shapes[route.Shape(parsed)]; taken {
			errs = append(errs, u.file.Errorf(0, "this page and %s give the same addresses", other))
			continue
		}
		shapes[route.Shape(parsed)] = path

		r := program.Route{File: id, Pattern: route.Pattern(parsed), Segments: []program.Segment{}, Layouts: []int{}}
		dynamic := false
		for _, s := range parsed {
			switch s.Kind {
			case route.Param:
				r.Segments = append(r.Segments, program.Segment{Param: s.Name})
				dynamic = true
			case route.Rest:
				r.Segments = append(r.Segments, program.Segment{Rest: s.Name})
				dynamic = true
			default:
				r.Segments = append(r.Segments, program.Segment{Text: s.Name})
			}
		}
		if dynamic && !u.script.paths {
			errs = append(errs, u.file.Errorf(0, "the name of this page has a parameter, so its script must say which pages there are: export async function getStaticPaths() { return [{ params: { … } }] }"))
		}
		// Every folder from pages/ down to the page may hold a layout.
		for dir := project.Pages; ; {
			if layout, found := b.ids[dir+"/"+route.Layout]; found {
				r.Layouts = append(r.Layouts, layout)
			}
			rest := strings.TrimPrefix(path, dir+"/")
			name, _, deeper := strings.Cut(rest, "/")
			if !deeper {
				break
			}
			dir += "/" + name
		}
		errs = append(errs, b.provided(u, r.Layouts)...)
		segments[id] = parsed
		out = append(out, r)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	route.Sort(out, func(r program.Route) []route.Segment { return segments[r.File] })
	return out, nil
}

// provided checks that every slot a page fills for its layouts is shown by
// one of them, and the same for every layout and the layouts around it. The
// nearest layout that has the slot shows it.
func (b *builder) provided(page *unit, layouts []int) Errors {
	var errs Errors
	check := func(u *unit, around []int) {
		for name, at := range u.provided {
			shown := false
			for _, layout := range around {
				shown = shown || slices.Contains(b.units[layout].outlets, name)
			}
			if !shown {
				errs = append(errs, u.file.Errorf(at, `slot=%q gives markup to a layout, and no layout around this file has <g-slot name=%q />`, name, name))
			}
		}
	}
	check(page, layouts)
	for i, layout := range layouts {
		check(b.units[layout], layouts[:i])
	}
	return errs
}

// bundled is what the bundler made of the modules of all files.
type bundled struct {
	code, smap []byte
	files      []string        // the files of the project that went into the bundle
	shared     map[string]bool // which files can be computed once for all pages
	styles     []style         // the style sheets in the order they apply
	warnings   []string
}

// bundle links the modules of all files, the code they import and the
// library of the framework into one file.
func (b *builder) bundle(routes []program.Route) (*bundled, error) {
	root := b.project.Root
	// The entry names the module of every file: a page is computed from its
	// route, and a shared block from its name alone, so any file may be what
	// a computation starts from. The rest of what the run needs to know is
	// in the program of the site.
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
	for _, first := range []bool{true, false} {
		for id, path := range b.paths {
			if routed[id] == first {
				fmt.Fprintf(&entry, "import * as $%d from %s;\n", id, quote("./"+path))
				table = append(table, fmt.Sprintf("[%d, $%d]", id, id))
			}
		}
	}
	fmt.Fprintf(&entry, "await run([%s]);\n", strings.Join(table, ", "))

	plugin := api.Plugin{Name: "shugo", Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo:`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{Path: args.Path, Namespace: "shugo"}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "shugo"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			contents := jsrt.Source
			if args.Path == "shugo:entry" {
				contents = entry.String()
			}
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: root}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `\.shugo$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			id, known := b.ids[b.relative(args.Path)]
			if !known {
				return api.OnLoadResult{}, fmt.Errorf("this file is not among the sources of the project")
			}
			u := b.units[id]
			contents := u.module.text(u.script)
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderTS, ResolveDir: filepath.Dir(args.Path)}, nil
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
		Alias:         map[string]string{"@": root},
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

var markupInCode = regexp.MustCompile(`<[A-Za-z/][^<>]*>`)

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
// error at that place. A place in a generated module becomes the place in
// the template that the code was written from.
func (b *builder) diagnose(message api.Message) *syntax.Error {
	at := message.Location
	id, known := b.ids[b.relative(at.File)]
	if !known {
		return &syntax.Error{Path: at.File, Line: at.Line, Column: at.Column + 1, Message: message.Text}
	}
	u := b.units[id]
	text := message.Text
	if line := at.Line - 1; line < len(u.module.from) && u.module.from[line].kind == 't' && markupInCode.MatchString(u.module.lines[line]) {
		text += "; markup cannot stand inside {…}: put <g-if> or <g-for> around it instead"
	}
	return u.file.Errorf(u.module.locate(u.script, at.Line-1, at.Column), "%s", text)
}

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
	}
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return err
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

	// A file that depends on the page passes that on to all that import it.
	own := map[string]bool{}
	for _, name := range names {
		if id, known := b.ids[name]; known {
			own[name] = b.units[id].own
		}
	}
	for changed := true; changed; {
		changed = false
		for _, name := range names {
			for _, imported := range meta.Inputs[name].Imports {
				if own[imported.Path] && !own[name] {
					own[name], changed = true, true
				}
			}
		}
	}
	out.shared = map[string]bool{}
	for name := range meta.Inputs {
		if id, known := b.ids[name]; known && b.units[id].role != page {
			out.shared[name] = !own[name]
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
		} else if id, known := b.ids[name]; known {
			for _, text := range b.units[id].styles {
				out.styles = append(out.styles, style{text: text, from: name})
			}
		}
	}
	for _, name := range names {
		if strings.HasSuffix(name, "shugo:entry") {
			visit(name)
		}
	}
	return nil
}
