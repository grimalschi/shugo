package syntax

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf8"
)

// SvelteVersion is the version of the official parser shipped with Shugo.
const SvelteVersion = "5.59.1"

//go:embed vendor/svelte-5.59.1.cjs
var compiler []byte

//go:embed svelte-ast.mjs
var adapter string

const runner = `
import { createRequire } from 'node:module';
import { createInterface } from 'node:readline';
import { Buffer } from 'node:buffer';

const [compilerFile, version] = process.argv.slice(-2);
const compiler = createRequire(compilerFile)(compilerFile);
if (compiler.VERSION !== version) throw new Error('unexpected bundled version of Svelte');

const reply = (message) => process.stdout.write(JSON.stringify(message) + '\n');
reply({ ready: true });
for await (const line of createInterface({ input: process.stdin })) {
  const { filename, source } = JSON.parse(line);
  try {
    reply({ template: parseTemplate(compiler, filename, source) });
  } catch (error) {
    const offset = error.start;
    if (!Number.isInteger(offset)) throw error;
    reply({ error: { Start: Buffer.byteLength(source.slice(0, offset)), Message: error.message } });
  }
}
`

// Parser translates the official Svelte AST into the input for Shugo lowering.
// A build reuses one parser process across templates.
type Parser struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	request  *json.Encoder
	response *json.Decoder
	stderr   bytes.Buffer
	failed   error
}

// Start starts the build-time parser. It uses the compiler bundled with Shugo.
// An empty runtime selects Bun when installed and Node otherwise.
func Start(runtime string) (*Parser, error) {
	if runtime == "" {
		runtime = "bun"
		if _, err := exec.LookPath(runtime); err != nil {
			runtime = "node"
		}
	}

	file, err := os.CreateTemp("", "shugo-svelte-*.cjs")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(compiler)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return nil, err
	}

	cmd := exec.Command(runtime, "--input-type=module", "--eval", adapter+"\n"+runner, file.Name(), SvelteVersion)
	p := &Parser{cmd: cmd}
	cmd.Stderr = &p.stderr
	p.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		p.stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("start the Svelte parser: %w", err)
	}
	p.request = json.NewEncoder(p.stdin)
	p.request.SetEscapeHTML(false)
	p.response = json.NewDecoder(stdout)

	// The compiler is loaded synchronously by the worker before it replies.
	// We may then remove the extracted copy even while it parses templates.
	var startup struct{ Ready bool }
	if err := p.response.Decode(&startup); err != nil {
		return nil, p.fail(err)
	}
	if !startup.Ready {
		return nil, p.fail(errors.New("missing startup acknowledgement"))
	}
	return p, nil
}

// Close stops the build-time parser and waits for it. It can be called more
// than once.
func (p *Parser) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stop()
}

func (p *Parser) stop() {
	if p.cmd != nil {
		p.stdin.Close()
		p.cmd.Wait()
		p.cmd = nil
	}
}

func (p *Parser) fail(err error) error {
	p.cmd.Process.Kill()
	p.stop()
	p.failed = fmt.Errorf("the Svelte parser stopped: %w", err)
	if detail := strings.TrimSpace(p.stderr.String()); detail != "" {
		p.failed = fmt.Errorf("%w\n%s", p.failed, detail)
	}
	return p.failed
}

// Parse reads one template using the build's parser.
func (p *Parser) Parse(path, source string) (*File, error) {
	file := &File{Path: path, Source: source}
	if !utf8.ValidString(source) {
		return nil, file.Errorf(0, "Svelte templates must be valid UTF-8")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failed != nil {
		return nil, p.failed
	}
	if p.cmd == nil {
		return nil, errors.New("the Svelte parser is closed")
	}
	err := p.request.Encode(struct {
		Filename string `json:"filename"`
		Source   string `json:"source"`
	}{path, source})
	if err != nil {
		return nil, p.fail(err)
	}
	var answer struct {
		Template *struct {
			Module, Script, Style *Script
			Body                  Fragment
		}
		Error *diagnostic
	}
	if err := p.response.Decode(&answer); err != nil {
		return nil, p.fail(err)
	}
	if answer.Error != nil {
		return nil, file.Errorf(answer.Error.Start, "%s", answer.Error.Message)
	}
	parsed := answer.Template
	if parsed == nil {
		return nil, p.fail(errors.New("missing parsed template"))
	}
	file.Module, file.Script, file.Style = parsed.Module, parsed.Script, parsed.Style
	file.Body = parsed.Body
	return file, nil
}

type diagnostic struct {
	Start   int
	Message string
}

// Parse reads a single template. Call Start to share a process across a build.
func Parse(path, source string) (*File, error) {
	p, err := Start("")
	if err != nil {
		return nil, err
	}
	defer p.Close()
	return p.Parse(path, source)
}

// Fragment is a list of the distinct node types in a template.
type Fragment []Node

func (f *Fragment) UnmarshalJSON(source []byte) error {
	var nodes []json.RawMessage
	if err := json.Unmarshal(source, &nodes); err != nil {
		return err
	}
	if nodes == nil {
		*f = nil
		return nil
	}
	out := make(Fragment, 0, len(nodes))
	for _, encoded := range nodes {
		node, err := decodeNode(encoded)
		if err != nil {
			return err
		}
		out = append(out, node)
	}
	*f = out
	return nil
}

func decodeNode(encoded []byte) (Node, error) {
	var header struct{ Type string }
	if err := json.Unmarshal(encoded, &header); err != nil {
		return nil, err
	}
	var node Node
	switch header.Type {
	case "Text":
		node = &Text{}
	case "HTMLComment":
		node = &HTMLComment{}
	case "Doctype":
		node = &Doctype{}
	case "Tag":
		node = &Tag{}
	case "IfBlock":
		node = &IfBlock{}
	case "EachBlock":
		node = &EachBlock{}
	case "AwaitBlock":
		node = &AwaitBlock{}
	case "KeyBlock":
		node = &KeyBlock{}
	case "SnippetBlock":
		node = &SnippetBlock{}
	case "Element":
		node = &Element{}
	default:
		return nil, fmt.Errorf("unknown template node %q from the Svelte adapter", header.Type)
	}
	return node, json.Unmarshal(encoded, node)
}
