package compile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/jsrt"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/syntax"
)

// side is one of the two bundles the JavaScript of a project is built into.
type side int

const (
	loadSide   side = iota // the scripts
	renderSide             // the templates
)

// bundled is what the bundler made of one side.
type bundled struct {
	code, smap []byte
	meta       *metafile
	warnings   []string
}

// metafile is what the bundler tells about a bundle: which module imports
// which, and what ended up in the output.
type metafile struct {
	Inputs map[string]struct {
		Imports []imported `json:"imports"`
	} `json:"inputs"`
	Outputs map[string]struct {
		Imports []imported `json:"imports"`
		Inputs  map[string]struct {
			Bytes int `json:"bytesInOutput"`
		} `json:"inputs"`
	} `json:"outputs"`
}

type imported struct {
	Path     string `json:"path"`
	External bool   `json:"external"`
}

// files returns the files of the project that went into the bundle.
func (m *metafile) files() []string {
	var out []string
	for name := range m.Inputs {
		// The library of the framework and the entry are not files.
		if !strings.Contains(name, ":") {
			out = append(out, name)
		}
	}
	return out
}

// modules returns the names of all modules, sorted.
func (m *metafile) modules() []string {
	names := make([]string, 0, len(m.Inputs))
	for name := range m.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// What the code of the render side finds in place of the ways out of the
// program. The library of that side defines them: each fails and says why.
var sealedNames = map[string]string{
	"fetch":       "__shugoNoNetwork",
	"Date.now":    "__shugoNoClock",
	"Math.random": "__shugoNoChance",
	"process":     "__shugoNoProcess",
}

// bundle links the modules of one side of all files, the code they import
// and the library of the framework into one file.
func (b *builder) bundle(s side, routes []program.Route) (*bundled, error) {
	root := b.project.Root
	entry := b.entry(s, routes)
	plugin := api.Plugin{Name: "shugo", Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo:`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{Path: args.Path, Namespace: "shugo"}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "shugo"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			contents, known := map[string]string{"shugo:entry": entry, "shugo:load": jsrt.Load, "shugo:render": jsrt.Render}[args.Path]
			if !known {
				return api.OnLoadResult{}, fmt.Errorf("there is no %s", args.Path)
			}
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS, ResolveDir: root}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `\.shugo$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			id, known := b.ids[b.relative(args.Path)]
			if !known {
				return api.OnLoadResult{}, fmt.Errorf("this file is not among the sources of the project")
			}
			u := b.units[id]
			contents := u.module(s).text(u.script)
			return api.OnLoadResult{Contents: &contents, Loader: api.LoaderTS, ResolveDir: filepath.Dir(args.Path)}, nil
		})
		// Style sheets do not go into JavaScript: they are collected and
		// built into a file of their own.
		build.OnLoad(api.OnLoadOptions{Filter: `\.css$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			empty := ""
			return api.OnLoadResult{Contents: &empty, Loader: api.LoaderJS}, nil
		})
	}}
	options := api.BuildOptions{
		EntryPoints:   []string{"shugo:entry"},
		Bundle:        true,
		Platform:      api.PlatformNode,
		Target:        api.ESNext,
		Charset:       api.CharsetUTF8,
		Packages:      api.PackagesExternal,
		Alias:         map[string]string{"@": root},
		Plugins:       []api.Plugin{plugin},
		AbsWorkingDir: root,
		Sourcemap:     api.SourceMapLinked,
		LegalComments: api.LegalCommentsNone,
		Metafile:      true,
		Write:         false,
		LogLevel:      api.LogLevelSilent,
	}
	if s == loadSide {
		options.Format = api.FormatESModule
		options.Outfile = filepath.Join(root, project.Output, LoadBundle)
	} else {
		// A format that cannot wait at the top of a module: the bundler
		// itself refuses a module of the render side that waits for data.
		options.Format = api.FormatCommonJS
		options.Outfile = filepath.Join(root, project.Output, RenderBundle)
		options.Define = sealedNames
	}
	result := api.Build(options)
	if len(result.Errors) > 0 {
		return nil, b.diagnoseAll(result.Errors, s)
	}
	out := &bundled{meta: &metafile{}}
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
			out.warnings = append(out.warnings, b.diagnoseAll([]api.Message{message}, s).Error())
		}
	}
	if err := json.Unmarshal([]byte(result.Metafile), out.meta); err != nil {
		return nil, err
	}
	return out, nil
}

// entry writes the module a side starts from. It names the module of every
// file: a page is computed from its route and a shared block from its name
// alone, so any file may be what a computation starts from. The rest of
// what the run needs to know is in the program of the site.
//
// Pages and layouts come first, in their order: style sheets apply in the
// order of imports, and that order starts from the pages.
func (b *builder) entry(s side, routes []program.Route) string {
	routed := map[int]bool{}
	for _, r := range routes {
		routed[r.File] = true
		for _, layout := range r.Layouts {
			routed[layout] = true
		}
	}
	var imports strings.Builder
	var table []string
	for _, first := range []bool{true, false} {
		for id, path := range b.paths {
			if routed[id] == first {
				fmt.Fprintf(&imports, "import * as $%d from %s;\n", id, quote("./"+path))
				table = append(table, fmt.Sprintf("[%d, $%d]", id, id))
			}
		}
	}
	if s == loadSide {
		return "import { run } from \"shugo:load\";\n" + imports.String() + fmt.Sprintf("await run([%s]);\n", strings.Join(table, ", "))
	}
	return "import { renderer } from \"shugo:render\";\n" + imports.String() + fmt.Sprintf("export default (site) => renderer(site, [%s]);\n", strings.Join(table, ", "))
}

// module returns the module of a file for one side.
func (u *unit) module(s side) *module {
	if s == loadSide {
		return u.load
	}
	return u.render
}

// outside names the modules of Node and Bun through which a program
// reaches what is outside of it, and what each of them reaches.
var outside = map[string]string{
	"fs": "files", "fs/promises": "files",
	"http": "the network", "https": "the network", "http2": "the network", "net": "the network", "tls": "the network",
	"dgram": "the network", "dns": "the network", "dns/promises": "the network",
	"child_process": "other programs", "worker_threads": "other programs", "cluster": "other programs",
	"sqlite": "a database", "bun:sqlite": "a database",
	"readline": "the terminal", "readline/promises": "the terminal",
	"bun": "files and the network",
}

// reaches tells what a module of the runtime reaches outside of the program.
func reaches(path string) (what string, ok bool) {
	what, ok = outside[strings.TrimPrefix(path, "node:")]
	return what, ok
}

// reaching are the names through which code reaches outside of the program
// without importing anything.
var reaching = map[string]string{
	"fetch":          "calls fetch",
	"process":        "reads the environment of the process",
	"Bun":            "uses Bun",
	"Deno":           "uses Deno",
	"XMLHttpRequest": "uses XMLHttpRequest",
	"WebSocket":      "uses WebSocket",
	"await":          "waits for data (await)",
}

var reachingWord = regexp.MustCompile(`\b(fetch|process|Bun|Deno|XMLHttpRequest|WebSocket|await)\b`)

// reachesOut tells how a piece of code reaches outside of the program, if
// it does. awaits tells whether waiting counts: in a module it does, and a
// script is asked about that separately.
func reachesOut(source string, awaits bool) string {
	found := func(word string) string {
		if word == "await" && !awaits {
			return ""
		}
		return reaching[word]
	}
	scan := syntax.NewScanner(source, 0)
	after := ""
	for {
		token, err := next(scan)
		if err != nil {
			// What cannot be read as JavaScript — types can do that — is
			// searched as text. That may find a word in a comment, which
			// errs to the safe side.
			for _, word := range reachingWord.FindAllString(source, -1) {
				if how := found(word); how != "" {
					return how
				}
			}
			return ""
		}
		if token.Kind == syntax.EOF {
			return ""
		}
		text := scan.Text(token)
		if token.Kind == syntax.Word && after != "." {
			if how := found(text); how != "" {
				return how
			}
		}
		after = text
	}
}

var scriptFile = regexp.MustCompile(`\.(ts|tsx|mts|cts|js|jsx|mjs|cjs)$`)

// sort tells the files that load data from those that only show what they
// are given, by what their scripts do and import. A module loads data if it
// reaches outside of the program, and whoever imports such a module loads
// data too. Every page and layout is computed as a file that loads.
func (b *builder) sort(meta *metafile) {
	cause := map[string]string{}
	names := meta.modules()
	for _, name := range names {
		// The library of the framework reads files to do its work; that is
		// not the project loading data.
		if strings.Contains(name, ":") {
			continue
		}
		if id, known := b.ids[name]; known {
			u := b.units[id]
			switch {
			case u.names.waits:
				cause[name] = "waits for data (await)"
			case u.names.now:
				cause[name] = "takes the time of the moment (new Date())"
			default:
				cause[name] = reachesOut(strings.Join(u.script.lines, "\n"), false)
			}
		} else if scriptFile.MatchString(name) {
			if source, err := os.ReadFile(b.project.Path(filepath.FromSlash(name))); err == nil {
				cause[name] = reachesOut(string(source), true)
			}
		}
		for _, imp := range meta.Inputs[name].Imports {
			if what, ok := reaches(imp.Path); ok && imp.External && cause[name] == "" {
				cause[name] = "reads " + what + " (" + imp.Path + ")"
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, name := range names {
			if cause[name] != "" {
				continue
			}
			for _, imp := range meta.Inputs[name].Imports {
				if !imp.External && cause[imp.Path] != "" {
					cause[name], changed = "imports "+imp.Path+", which "+cause[imp.Path], true
					break
				}
			}
		}
	}
	for _, u := range b.units {
		u.why = cause[u.file.Path]
		u.loads = u.role != component || u.why != ""
	}
}

// sealed makes sure that the code of the render side reaches nothing
// outside of the program. What a template shows must depend only on what
// it is given: then a page needs computing again only when that changes.
func (b *builder) sealed(meta *metafile) error {
	var errs Errors
	blamed := map[string]bool{}
	for _, output := range meta.Outputs {
		for _, out := range output.Imports {
			what, reachesOut := reaches(out.Path)
			if !out.External || !reachesOut {
				continue
			}
			// The modules that brought it in, and the files whose templates
			// call something of those modules.
			for _, module := range meta.modules() {
				if output.Inputs[module].Bytes == 0 || blamed[module] || !imports(meta, module, out.Path) {
					continue
				}
				blamed[module] = true
				var chains [][]string
				for _, path := range b.paths {
					if chain := meta.way(path, module); chain != nil {
						chains = append(chains, chain)
					}
				}
				if len(chains) == 0 {
					return fmt.Errorf("%s reads %s (%s), and it is a part of the code that templates call, which must load nothing", module, what, out.Path)
				}
				more := ""
				if len(chains) > 1 {
					more = fmt.Sprintf(" (%d more files do the same)", len(chains)-1)
				}
				u := b.units[b.ids[chains[0][0]]]
				errs = append(errs, u.file.Errorf(u.file.Script.Start,
					"%s → %s: the template calls %s of a module that reads %s%s. A template works on the data of its script and can load nothing itself, so that a page changes only when its data does. Move the function into a module that loads nothing, or call it in the script and show the result",
					strings.Join(chains[0], " → "), out.Path, u.calledOf(chains[0][1]), what, more))
			}
		}
	}
	if len(errs) > 0 {
		sort.Slice(errs, func(i, j int) bool { return errs[i].Path < errs[j].Path })
		return errs
	}
	return nil
}

// calledOf names what the template of the file calls of a module, as far
// as the names of the two tell: the script writes "../lib/shop" where the
// bundler says lib/shop.ts.
func (u *unit) calledOf(module string) string {
	stem := func(path string) string {
		base := path[strings.LastIndexByte(path, '/')+1:]
		if dot := strings.LastIndexByte(base, '.'); dot > 0 {
			base = base[:dot]
		}
		return base
	}
	var names []string
	for _, name := range u.reads.calls {
		if stem(u.names.imports[name]) == stem(module) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "a function"
	}
	return strings.Join(names, ", ")
}

// imports tells whether a module imports the given path itself.
func imports(meta *metafile, module, path string) bool {
	for _, imp := range meta.Inputs[module].Imports {
		if imp.Path == path {
			return true
		}
	}
	return false
}

// way returns the shortest chain of imports from one module to another, or
// nil if there is none.
func (m *metafile) way(from, to string) []string {
	if _, known := m.Inputs[from]; !known {
		return nil
	}
	before := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if at == to {
			var chain []string
			for ; at != ""; at = before[at] {
				chain = append([]string{at}, chain...)
			}
			return chain
		}
		for _, imp := range m.Inputs[at].Imports {
			if _, seen := before[imp.Path]; !seen && !imp.External {
				before[imp.Path] = at
				queue = append(queue, imp.Path)
			}
		}
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

var markupInCode = regexp.MustCompile(`<[A-Za-z/][^<>]*>`)

// diagnoseAll turns the messages of the bundler into errors in the files
// of the project.
func (b *builder) diagnoseAll(messages []api.Message, s side) error {
	var errs Errors
	for _, message := range messages {
		if message.Location == nil {
			return errors.New(message.Text)
		}
		errs = append(errs, b.diagnose(message, s))
	}
	return errs
}

// diagnose turns a message of the bundler about a place in a file into an
// error at that place. A place in a generated module becomes the place in
// the template that the code was written from.
func (b *builder) diagnose(message api.Message, s side) *syntax.Error {
	at := message.Location
	text := message.Text
	if s == renderSide && strings.Contains(text, "Top-level await") {
		text = "this module waits for data when it is imported (await at its top), and a template or a component that only shows what it is given calls its code. Such code can load nothing: load the data in the script of a page, a layout or a shared block"
	}
	id, known := b.ids[b.relative(at.File)]
	if !known {
		return &syntax.Error{Path: at.File, Line: at.Line, Column: at.Column + 1, Message: text}
	}
	u := b.units[id]
	m := u.module(s)
	if line := at.Line - 1; line < len(m.from) && m.from[line].kind == 't' && markupInCode.MatchString(m.lines[line]) {
		text += "; markup cannot stand inside {…}: put <g-if> or <g-for> around it instead"
	}
	return u.file.Errorf(m.locate(u.script, at.Line-1, at.Column), "%s", text)
}

// graph reads from the bundler which module of the render side imports
// which. It gives the order of style sheets and tells which files can be
// computed once for all pages: those that do not depend on the page
// themselves and import nothing that does.
func (b *builder) graph(meta *metafile) (shared map[string]bool, styles []style, err error) {
	names := meta.modules()

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
			for _, imp := range meta.Inputs[name].Imports {
				if own[imp.Path] && !own[name] {
					own[name], changed = true, true
				}
			}
		}
	}
	shared = map[string]bool{}
	var errs Errors
	for _, name := range names {
		id, known := b.ids[name]
		if !known || b.units[id].role == page {
			continue
		}
		shared[name] = !own[name]
		if u := b.units[id]; u.role == component && u.loads && own[name] {
			errs = append(errs, u.file.Errorf(0,
				"this component loads data — its script %s — so it is computed once for the whole site; and it reads the page it is shown on, itself or through a component it shows (Shugo.url, Shugo.params, a slot for a layout). It cannot do both: give it what it needs of the page as a plain parameter, or load the data in the page and pass it", u.why))
		}
	}
	if len(errs) > 0 {
		return nil, nil, errs
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
		for _, imp := range meta.Inputs[name].Imports {
			visit(imp.Path)
		}
		if strings.HasSuffix(name, ".css") {
			styles = append(styles, style{path: b.project.Path(filepath.FromSlash(name))})
		} else if id, known := b.ids[name]; known {
			for _, text := range b.units[id].styles {
				styles = append(styles, style{text: text, from: name})
			}
		}
	}
	for _, name := range names {
		if strings.HasSuffix(name, "shugo:entry") {
			visit(name)
		}
	}
	return shared, styles, nil
}
