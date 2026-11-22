// Package svelte is the Svelte that Shugo ships: its compiler, which turns
// the templates of a project into code for a server and the islands into
// code for the browser too, and its runtimes, for the server and for the
// browser. All are of one version, vendor/README.md tells where they come
// from, so a project does not install Svelte.
package svelte

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/evanw/esbuild/pkg/api"
)

// Version is the version of Svelte that Shugo ships.
const Version = "5.59.1"

//go:embed vendor/compiler-5.59.1.cjs
var compilerBundle []byte

//go:embed vendor/acorn.cjs
var acornBundle []byte

//go:embed compile.mjs
var compilerScript string

//go:embed island-program.mjs
var islandProgram string

//go:embed vendor/runtime/*.js vendor/runtime-browser/*.js
var runtime embed.FS

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

// Start starts the compiler with the JavaScript runtime given, "bun" or
// "node", in the folder dir.
func Start(runtime, dir string) (*Compiler, error) {
	compiler, err := temporary("shugo-svelte-*.cjs", compilerBundle)
	if err != nil {
		return nil, err
	}
	defer os.Remove(compiler)
	acorn, err := temporary("shugo-acorn-*.cjs", acornBundle)
	if err != nil {
		return nil, err
	}
	defer os.Remove(acorn)

	cmd := exec.Command(runtime, "--input-type=module", "--eval", islandProgram+"\n"+compilerScript, compiler, acorn, Version)
	cmd.Dir = dir
	c := &Compiler{cmd: cmd}
	cmd.Stderr = &c.stderr
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

	// The compiler is loaded before the first answer, so the copy of it can
	// go while it works.
	var startup struct{ Ready bool }
	if err := c.response.Decode(&startup); err != nil {
		return nil, c.fail(err)
	}
	if !startup.Ready {
		return nil, c.fail(errors.New("it did not say it is ready"))
	}
	return c, nil
}

// temporary writes data into a new file of the folder of temporary files
// and returns its absolute path: createRequire needs one, even when TMPDIR
// is relative.
func temporary(pattern string, data []byte) (string, error) {
	folder, err := filepath.Abs(os.TempDir())
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(folder, pattern)
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
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

// StartPool starts n compilers, as Start does.
func StartPool(runtime, dir string, n int) (*Pool, error) {
	p := &Pool{free: make(chan *Compiler, max(n, 1))}
	for range cap(p.free) {
		c, err := Start(runtime, dir)
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

// Runtime has the bundler take the modules of Svelte that a page imports,
// svelte, svelte/internal/server, svelte/store and the others, from the
// server runtime Shugo ships. A module of Svelte that a server has no use
// for is an error.
func Runtime(build api.PluginBuild) {
	runtimeOf(build, "runtime", "a page is computed with the server runtime")
}

// BrowserRuntime has the bundler take the modules of Svelte that an island
// imports in the browser from the browser runtime Shugo ships.
func BrowserRuntime(build api.PluginBuild) {
	runtimeOf(build, "runtime-browser", "an island runs in the browser with the browser runtime")
}

// runtimeOf resolves the modules of Svelte to the files of a runtime under
// vendor/.
func runtimeOf(build api.PluginBuild, folder, which string) {
	build.OnResolve(api.OnResolveOptions{Filter: `^svelte(/|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		name := "svelte.js"
		if rest, sub := strings.CutPrefix(args.Path, "svelte/"); sub {
			name = strings.ReplaceAll(rest, "/", "-") + ".js"
		}
		if _, err := runtime.Open("vendor/" + folder + "/" + name); err != nil {
			return api.OnResolveResult{}, fmt.Errorf("%s of Svelte %s that Shugo ships, and %s is not a part of it: it has %s", which, Version, args.Path, strings.Join(modules(folder), ", "))
		}
		return api.OnResolveResult{Path: name, Namespace: "svelte"}, nil
	})
	build.OnResolve(api.OnResolveOptions{Filter: `^\./`, Namespace: "svelte"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		return api.OnResolveResult{Path: path.Join(path.Dir(args.Importer), args.Path), Namespace: "svelte"}, nil
	})
	build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "svelte"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		data, err := runtime.ReadFile("vendor/" + folder + "/" + args.Path)
		if err != nil {
			return api.OnLoadResult{}, err
		}
		contents := string(data)
		return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
	})
}

// Modules lists the modules of Svelte that a page may import.
func Modules() []string {
	return modules("runtime")
}

// modules lists the modules of Svelte in a runtime under vendor/.
func modules(folder string) []string {
	entries, _ := runtime.ReadDir("vendor/" + folder)
	var out []string
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".js")
		switch {
		case strings.HasPrefix(name, "chunk-"):
		case name == "svelte":
			out = append(out, "svelte")
		default:
			out = append(out, "svelte/"+strings.ReplaceAll(name, "-", "/"))
		}
	}
	return out
}
