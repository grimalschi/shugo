package compile

import (
	"fmt"
	"strings"

	"github.com/grimalschi/shugo/internal/syntax"
)

// Errors are the mistakes found in a project, each with its place.
type Errors []*syntax.Error

func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i, err := range e {
		lines[i] = err.Error()
	}
	return strings.Join(lines, "\n")
}

// internalError reports something the compiler did not expect of its own
// tools. The project's author can do nothing about it but tell us.
func internalError(f *syntax.File, format string, arguments ...any) error {
	return f.Errorf(f.Script.Start, "internal error, please report it: %s", fmt.Sprintf(format, arguments...))
}
