// Package svelte runs Svelte for Shugo: its compiler, which turns the
// templates of a project into code for a server and the islands into code
// for the browser too, and its runtimes, which the bundler takes from the
// package. Svelte is a dependency of the package shugo, of the one version
// Shugo is made for; the command of that package tells the program where it
// is. Nothing of Svelte is a part of the program.
package svelte

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/evanw/esbuild/pkg/api"
)

// Version is the version of Svelte that Shugo is made for: the package
// shugo depends on this one, and Shugo takes no other. Its server code and
// the comments of hydration are what the blocks of Shugo and the programs
// of islands are made of.
const Version = "5.59.1"

//go:embed compile.mjs
var compilerScript string

//go:embed island-program.mjs
var islandProgram string

// Package is the package svelte that Shugo runs.
type Package struct {
	// Dir is the folder of the package, inside a folder node_modules.
	Dir string
}

// Find finds the package svelte: where the command of the package shugo
// says it is, in SHUGO_SVELTE, or else in node_modules of the folder of the
// project or of one around it, as the repository of Shugo has it for its
// examples. It must be of Version.
func Find(root string) (*Package, error) {
	dir := os.Getenv("SHUGO_SVELTE")
	if dir == "" {
		dir = Installed(root, "svelte")
	}
	if dir == "" {
		return nil, errors.New("Svelte is not installed: Shugo takes it from the package shugo, which depends on it; install shugo: bun add shugo")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if filepath.Base(dir) != "svelte" || filepath.Base(filepath.Dir(dir)) != "node_modules" {
		return nil, fmt.Errorf("%s is not a package svelte in node_modules", dir)
	}
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("the package svelte at %s: %w", dir, err)
	}
	var manifest struct{ Version string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("the package svelte at %s: %w", dir, err)
	}
	if manifest.Version != Version {
		return nil, fmt.Errorf("the package svelte at %s is of version %s, and Shugo is made for Svelte %s: install shugo again, bun add shugo, and nothing else of Svelte", dir, manifest.Version, Version)
	}
	return &Package{Dir: dir}, nil
}

// Installed returns the folder of a package of npm in node_modules of a
// folder or of one around it, where the runtime of JavaScript would look for
// it; nothing when there is none.
func Installed(dir, name string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		folder := filepath.Join(dir, "node_modules", filepath.FromSlash(name))
		if info, err := os.Stat(folder); err == nil && info.IsDir() {
			return folder
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Template is a template compiled for a server.
type Template struct {
	Code string // an ES module whose default export is the component
	Map  string // the source map of Code, as JSON
	CSS  string // the styles of the component, scoped to it by Svelte
	// Warnings are what Svelte warns about in the template.
	Warnings []Message
	Facts    `json:"facts"`
}

// Facts are what the framework needs to know of a template.
type Facts struct {
	// Page tells that the template imports page or paginate from the module
	// shugo: it shows what differs from page to page.
	Page bool `json:"page"`
	// Context tells that the template uses the context of Svelte, ID that
	// it calls $props.id(), and Select that it shows a snippet inside a
	// <select>, whose value chooses the <option> the snippet shows: what it
	// shows depends on where it stands.
	Context bool `json:"context"`
	ID      bool `json:"id"`
	Select  bool `json:"select"`
	// Renders tells that the template shows a snippet, as a layout shows the
	// page with {@render children()}.
	Renders bool `json:"renders"`
	// Paths tells that its <script module> exports getStaticPaths.
	Paths bool `json:"paths"`
	// Islands are the islands the template imports.
	Islands []Island `json:"islands"`
}

// Island is an island a template imports.
type Island struct {
	Source string `json:"source"` // as the import names it
	// Request tells that a tag of the island binds a prop to the data of a
	// request: request:initial={search(…)}.
	Request bool `json:"request"`
}

// Message is a mistake or a warning at a place of a template.
type Message struct {
	Line   int    `json:"line"`   // from 1
	Column int    `json:"column"` // from 1, in characters
	Text   string `json:"message"`
}

// Mistakes are the mistakes found in a template.
type Mistakes []Message

func (m Mistakes) Error() string {
	lines := make([]string, len(m))
	for i, one := range m {
		lines[i] = fmt.Sprintf("%d:%d: %s", one.Line, one.Column, one.Text)
	}
	return strings.Join(lines, "\n")
}

// Compiler compiles templates with the compiler of Svelte, run by Bun or
// Node. A build uses one for all its templates.
type Compiler struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	request  *json.Encoder
	response *json.Decoder
	stderr   bytes.Buffer
	failed   error
}

// Start starts the compiler of the package with the JavaScript runtime
// given, "bun" or "node", in the folder dir.
func (p *Package) Start(runtime, dir string) (*Compiler, error) {
	cmd := exec.Command(runtime, "--input-type=module", "--eval", islandProgram+"\n"+compilerScript, p.Dir, Version)
	cmd.Dir = dir
	c := &Compiler{cmd: cmd}
	cmd.Stderr = &c.stderr
	var err error
	if c.stdin, err = cmd.StdinPipe(); err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		c.stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("start the compiler of Svelte: %w", err)
	}
	c.request = json.NewEncoder(c.stdin)
	c.request.SetEscapeHTML(false)
	c.response = json.NewDecoder(stdout)

	// The compiler says it is ready once it is loaded.
	var startup struct{ Ready bool }
	if err := c.response.Decode(&startup); err != nil {
		return nil, c.fail(err)
	}
	if !startup.Ready {
		return nil, c.fail(errors.New("it did not say it is ready"))
	}
	return c, nil
}

// Compile compiles one template. path is how messages name the file. The
// mistakes in the template come as Mistakes.
func (c *Compiler) Compile(path, source string) (*Template, error) {
	var t Template
	return &t, c.ask(request{Path: path, Source: source}, &t)
}

// CompileModule compiles a module with runes, a .svelte.js file, for the
// server: the Template has Code, Map and Warnings. A .svelte.ts file is
// given with its types taken away.
func (c *Compiler) CompileModule(path, source string) (*Template, error) {
	var t Template
	return &t, c.ask(request{Path: path, Source: source, Module: true}, &t)
}

// IslandCode is a component of an island, or a module with runes in its
// graph, compiled for the browser and for the server.
type IslandCode struct {
	Client, Server string
	Map            string // the source map of Server, as JSON
	CSS            string // the styles of the component, scoped to it
	Warnings       []Message
}

// CompileIsland compiles a component of an island, or a module with runes
// when module is set; a .svelte.ts file is given with its types taken away.
func (c *Compiler) CompileIsland(path, source string, module bool) (*IslandCode, error) {
	var code IslandCode
	return &code, c.ask(request{Path: path, Source: source, Island: true, Module: module}, &code)
}

// LowerIsland makes the program by which Go builds the HTML of an island
// bound to a request, from the server code of its component, as JSON. A
// mistake is told at its place in that code.
func (c *Compiler) LowerIsland(server string) (json.RawMessage, error) {
	var answer struct{ Program json.RawMessage }
	return answer.Program, c.ask(request{Source: server, Lower: true}, &answer)
}

// request is what the compiler is asked; see compile.mjs.
type request struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	Module bool   `json:"module,omitempty"`
	Island bool   `json:"island,omitempty"`
	Lower  bool   `json:"lower,omitempty"`
}

// ask asks the compiler and reads its answer into out; the mistakes it
// found come as Mistakes.
func (c *Compiler) ask(r request, out any) error {
	if !utf8.ValidString(r.Source) {
		return Mistakes{{Line: 1, Column: 1, Text: "a source file is text in UTF-8"}}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failed != nil {
		return c.failed
	}
	if c.cmd == nil {
		return errors.New("the compiler of Svelte is closed")
	}
	if err := c.request.Encode(r); err != nil {
		return c.fail(err)
	}
	var answer json.RawMessage
	if err := c.response.Decode(&answer); err != nil {
		return c.fail(err)
	}
	var mistakes struct{ Errors Mistakes }
	if err := json.Unmarshal(answer, &mistakes); err != nil {
		return c.fail(err)
	}
	if len(mistakes.Errors) > 0 {
		return mistakes.Errors
	}
	if err := json.Unmarshal(answer, out); err != nil {
		return c.fail(err)
	}
	return nil
}

// Close stops the compiler and waits for it. It can be called more than once.
func (c *Compiler) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stop()
}

func (c *Compiler) stop() {
	if c.cmd != nil {
		c.stdin.Close()
		c.cmd.Wait()
		c.cmd = nil
	}
}

func (c *Compiler) fail(err error) error {
	c.cmd.Process.Kill()
	c.stop()
	c.failed = fmt.Errorf("the compiler of Svelte stopped: %w", err)
	if detail := strings.TrimSpace(c.stderr.String()); detail != "" {
		c.failed = fmt.Errorf("%w\n%s", c.failed, detail)
	}
	return c.failed
}

// Pool is a few compilers that work at once; each compiles one file at a
// time. The calls of a pool may come from several goroutines.
type Pool struct {
	free chan *Compiler
}

// StartPool starts n compilers of the package, as Start does.
func (pkg *Package) StartPool(runtime, dir string, n int) (*Pool, error) {
	p := &Pool{free: make(chan *Compiler, max(n, 1))}
	for range cap(p.free) {
		c, err := pkg.Start(runtime, dir)
		if err != nil {
			p.Close()
			return nil, err
		}
		p.free <- c
	}
	return p, nil
}

// Compile compiles a template with a compiler that is free.
func (p *Pool) Compile(path, source string) (*Template, error) {
	c := <-p.free
	defer func() { p.free <- c }()
	return c.Compile(path, source)
}

// CompileModule compiles a module with runes with a compiler that is free.
func (p *Pool) CompileModule(path, source string) (*Template, error) {
	c := <-p.free
	defer func() { p.free <- c }()
	return c.CompileModule(path, source)
}

// CompileIsland compiles a part of an island with a compiler that is free.
func (p *Pool) CompileIsland(path, source string, module bool) (*IslandCode, error) {
	c := <-p.free
	defer func() { p.free <- c }()
	return c.CompileIsland(path, source, module)
}

// LowerIsland makes the program of an island with a compiler that is free.
func (p *Pool) LowerIsland(server string) (json.RawMessage, error) {
	c := <-p.free
	defer func() { p.free <- c }()
	return c.LowerIsland(server)
}

// Close stops the compilers. Nothing may be compiling.
func (p *Pool) Close() {
	for {
		select {
		case c := <-p.free:
			c.Close()
		default:
			return
		}
	}
}

// The modules of Svelte that the code a page is computed with may import,
// and those an island may import in the browser, with the internal modules
// that the code Svelte compiled imports.
var (
	serverModules = []string{
		"svelte", "svelte/internal/server", "svelte/server", "svelte/store", "svelte/reactivity",
		"svelte/reactivity/window", "svelte/easing", "svelte/transition", "svelte/animate", "svelte/motion",
		"svelte/events", "svelte/attachments",
	}
	browserModules = []string{
		"svelte", "svelte/internal/client", "svelte/internal/init-operations", "svelte/store", "svelte/reactivity",
		"svelte/reactivity/window", "svelte/easing", "svelte/transition", "svelte/animate", "svelte/motion",
		"svelte/events", "svelte/attachments",
	}
)

// Modules lists the modules of Svelte that a page may import.
func Modules() []string {
	return slices.Clone(serverModules)
}

// Resolving marks a resolution that a plugin of Shugo asks the bundler for,
// to find a file as the bundler itself would: every plugin of Shugo leaves
// it to the bundler, and none asks again.
type Resolving struct{}

// Runtime has the bundler take the modules of Svelte from the package,
// whatever node_modules of the project holds: those a page may import on a
// server, or, for the browser, those an island may import there. Another
// module of Svelte is an error. Svelte imports itself by its name too; what
// it imports is not looked at.
func (p *Package) Runtime(build api.PluginBuild, browser bool) {
	modules, which := serverModules, "a page is computed on a server"
	if browser {
		modules, which = browserModules, "an island runs in the browser"
	}
	from := filepath.Dir(filepath.Dir(p.Dir)) // the folder whose node_modules holds the package
	inside := p.Dir + string(filepath.Separator)
	build.OnResolve(api.OnResolveOptions{Filter: `^svelte(/|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if _, again := args.PluginData.(Resolving); again {
			return api.OnResolveResult{}, nil
		}
		if !strings.HasPrefix(args.Importer, inside) && !slices.Contains(modules, args.Path) {
			return api.OnResolveResult{}, fmt.Errorf("%s with Svelte %s, and %s is not among the modules it takes: %s", which, Version, args.Path, strings.Join(modules, ", "))
		}
		found := build.Resolve(args.Path, api.ResolveOptions{ResolveDir: from, Kind: args.Kind, PluginData: Resolving{}})
		if len(found.Errors) > 0 {
			return api.OnResolveResult{Errors: found.Errors}, nil
		}
		sideEffects := api.SideEffectsFalse
		if found.SideEffects {
			sideEffects = api.SideEffectsTrue
		}
		return api.OnResolveResult{Path: found.Path, Namespace: found.Namespace, External: found.External, SideEffects: sideEffects, Suffix: found.Suffix}, nil
	})
}

// Contains tells whether a file is a part of the package.
func (p *Package) Contains(path string) bool {
	return strings.HasPrefix(path, p.Dir+string(filepath.Separator))
}
