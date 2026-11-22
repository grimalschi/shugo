// Package svelte is the Svelte that Shugo ships: its compiler, which turns
// the templates of a project into code for a server, and its server runtime,
// which that code runs on. Both are of one version, vendor/README.md tells
// where they come from, so a project does not install Svelte to have pages.
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

//go:embed compile.mjs
var compilerScript string

//go:embed vendor/runtime/*.js
var runtime embed.FS

// Template is a template compiled for a server.
type Template struct {
	Code string // an ES module whose default export is the component
	Map  string // the source map of Code, as JSON
	CSS  string // the styles of the component, scoped to it by Svelte
	// Warnings are what Svelte warns about in the template.
	Warnings []Message
	Facts
}

// Facts are what the framework needs to know of a template.
type Facts struct {
	// Page tells that the template imports page or paginate from the module
	// shugo: it shows what differs from page to page.
	Page bool `json:"page"`
	// Context tells that the template uses the context of Svelte, and ID
	// that it calls $props.id(): what it shows depends on where it stands.
	Context bool `json:"context"`
	ID      bool `json:"id"`
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
	// createRequire needs an absolute path, even when TMPDIR is relative.
	temp, err := filepath.Abs(os.TempDir())
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(temp, "shugo-svelte-*.cjs")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(compilerBundle)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return nil, err
	}

	cmd := exec.Command(runtime, "--input-type=module", "--eval", compilerScript, file.Name(), Version)
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

// Compile compiles one template. path is how messages name the file. The
// mistakes in the template come as Mistakes.
func (c *Compiler) Compile(path, source string) (*Template, error) {
	if !utf8.ValidString(source) {
		return nil, Mistakes{{Line: 1, Column: 1, Text: "a template is text in UTF-8"}}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failed != nil {
		return nil, c.failed
	}
	if c.cmd == nil {
		return nil, errors.New("the compiler of Svelte is closed")
	}
	if err := c.request.Encode(struct {
		Path   string `json:"path"`
		Source string `json:"source"`
	}{path, source}); err != nil {
		return nil, c.fail(err)
	}
	var answer struct {
		Code, Map, CSS string
		Warnings       []Message
		Facts          Facts
		Errors         Mistakes
	}
	if err := c.response.Decode(&answer); err != nil {
		return nil, c.fail(err)
	}
	if len(answer.Errors) > 0 {
		return nil, answer.Errors
	}
	return &Template{Code: answer.Code, Map: answer.Map, CSS: answer.CSS, Warnings: answer.Warnings, Facts: answer.Facts}, nil
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

// Runtime has the bundler take the modules of Svelte that a page imports,
// svelte, svelte/internal/server, svelte/store and the others, from the
// server runtime Shugo ships. A module of Svelte that a server has no use
// for is an error.
func Runtime(build api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: `^svelte(/|$)`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		name := "svelte.js"
		if rest, sub := strings.CutPrefix(args.Path, "svelte/"); sub {
			name = strings.ReplaceAll(rest, "/", "-") + ".js"
		}
		if _, err := runtime.Open("vendor/runtime/" + name); err != nil {
			return api.OnResolveResult{}, fmt.Errorf("a page is computed with the server runtime of Svelte %s that Shugo ships, and %s is not a part of it: it has %s", Version, args.Path, strings.Join(Modules(), ", "))
		}
		return api.OnResolveResult{Path: name, Namespace: "svelte"}, nil
	})
	build.OnResolve(api.OnResolveOptions{Filter: `^\./`, Namespace: "svelte"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		return api.OnResolveResult{Path: path.Join(path.Dir(args.Importer), args.Path), Namespace: "svelte"}, nil
	})
	build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "svelte"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		data, err := runtime.ReadFile("vendor/runtime/" + args.Path)
		if err != nil {
			return api.OnLoadResult{}, err
		}
		contents := string(data)
		return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
	})
}

// Modules lists the modules of Svelte that a page may import.
func Modules() []string {
	entries, _ := runtime.ReadDir("vendor/runtime")
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
