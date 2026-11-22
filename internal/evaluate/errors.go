package evaluate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/sourcemap"
)

// ScriptError is a failure of the project's own code while the site was
// being computed: an exception in a script or in an expression of a template.
type ScriptError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"` // as the JavaScript runtime wrote it
	Place   string `json:"place"` // what was being computed
	// Frames are the places in the files of the project that the failure
	// went through, the innermost first.
	Frames []Frame `json:"-"`
}

// Frame is one place in a file of the project.
type Frame struct {
	File         string // relative to the root of the project
	Line, Column int    // from 1
}

func (e *ScriptError) Error() string {
	var out strings.Builder
	out.WriteString(e.Message)
	if e.Place != "" {
		out.WriteString("\n  while computing " + e.Place)
	}
	for _, frame := range e.Frames {
		fmt.Fprintf(&out, "\n  at %s:%d:%d", frame.File, frame.Line, frame.Column)
	}
	return out.String()
}

// A line of a stack that points into the bundle: "at name (file:///…/eval.mjs:12:34)".
var bundleFrame = regexp.MustCompile(regexp.QuoteMeta(compile.Bundle) + `:(\d+):(\d+)`)

// locate finds the places in the project's files that the stack went
// through. The stack names lines of the bundle; the source map of the bundle
// says which file every one of them was made from.
func (e *ScriptError) locate(p *project.Project) {
	data, err := os.ReadFile(p.Path(project.Output, compile.Bundle+".map"))
	if err != nil {
		return
	}
	smap, err := sourcemap.Parse(data)
	if err != nil {
		return
	}
	for _, match := range bundleFrame.FindAllStringSubmatch(e.Stack, -1) {
		line, _ := strconv.Atoi(match[1])
		column, _ := strconv.Atoi(match[2])
		source, sourceLine, sourceColumn, found := smap.Find(line-1, column-1)
		// The library of the framework is not a file of the project.
		if !found || strings.Contains(source, ":") {
			continue
		}
		file, err := filepath.Rel(p.Root, filepath.Join(p.Path(project.Output), filepath.FromSlash(source)))
		if err != nil || strings.HasPrefix(file, "..") {
			continue
		}
		frame := Frame{filepath.ToSlash(file), sourceLine + 1, sourceColumn + 1}
		if n := len(e.Frames); n == 0 || e.Frames[n-1] != frame {
			e.Frames = append(e.Frames, frame)
		}
	}
	e.explain()
}

// explain adds to the message what the author of the site can do, where
// Shugo knows: a package bundled with the code of the site looked for a file
// next to its own code, and the bundle is in _build/, where that file is
// not.
func (e *ScriptError) explain() {
	if len(e.Frames) == 0 || !strings.Contains(e.Message, "ENOENT") || strings.Contains(e.Message, readsOwnFiles) {
		return
	}
	if name, _ := project.PackageOf(e.Frames[0].File); name != "" {
		e.Message += fmt.Sprintf(": the package %s %s, and its code is bundled with the code of the site into %s/%s, where that file is not: take a package that does not read its own files", name, readsOwnFiles, project.Output, compile.Bundle)
	}
}

const readsOwnFiles = "reads a file next to its own code"
