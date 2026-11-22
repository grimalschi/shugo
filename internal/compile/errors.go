package compile

import (
	"fmt"
	"strings"

	"github.com/grimalschi/shugo/internal/svelte"
)

// Error is a mistake at a place in a file of the project.
type Error struct {
	Path    string
	Line    int // from 1
	Column  int // from 1, in characters
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Column, e.Message)
}

// Errors are the mistakes found in a project, each with its place.
type Errors []*Error

func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i, err := range e {
		lines[i] = err.Error()
	}
	return strings.Join(lines, "\n")
}

// mistakes places the mistakes Svelte found in a file.
func mistakes(path string, found svelte.Mistakes) Errors {
	out := make(Errors, len(found))
	for i, m := range found {
		out[i] = &Error{Path: path, Line: m.Line, Column: m.Column, Message: m.Text}
	}
	return out
}
