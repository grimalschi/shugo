package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/project"
)

// importsPage tells whether code imports from the module shugo what differs
// from page to page: page, or paginate, which cuts a list by the number of
// the page. A file that does cannot be computed once for all pages. A
// namespace import counts as both; an import of types alone as neither.
func importsPage(code string) bool {
	return imports(code, "shugo", "page", "paginate")
}

// importsContext tells whether code imports the context of Svelte, which the
// components around a component give it.
func importsContext(code string) bool {
	return imports(code, "svelte", "getContext", "setContext", "hasContext", "getAllContexts", "createContext")
}

// imports tells whether code imports one of the names from a module, or
// all of it.
func imports(code, module string, names ...string) bool {
	for _, m := range importFrom.FindAllStringSubmatch(code, -1) {
		if m[1] != "" || m[3] != module {
			continue // import type { … }
		}
		clause := m[2]
		if strings.Contains(clause, "*") {
			return true
		}
		open, close := strings.IndexByte(clause, '{'), strings.LastIndexByte(clause, '}')
		if open < 0 || close < open {
			continue
		}
		for _, spec := range strings.Split(clause[open+1:close], ",") {
			fields := strings.Fields(spec)
			if len(fields) == 0 || fields[0] == "type" {
				continue
			}
			for _, name := range names {
				if fields[0] == name {
					return true
				}
			}
		}
	}
	return false
}

// importFrom finds what an import or an export takes from a module.
var importFrom = regexp.MustCompile(`\b(?:import|export)\s+(type\s+)?([^;'"]*?)\s*from\s*["']([^"']+)["']`)

func pathBase(path string) string {
	return path[strings.LastIndexByte(path, '/')+1:]
}

// quote writes a text as a string of JavaScript.
func quote(text string) string {
	out, _ := json.Marshal(text)
	return string(out)
}

// siteModule checks a module of the site as the bundler loads it: the code
// that computes pages does not wait, at the top of a module either, which
// would wait when the code is loaded. A module of a package is not looked
// at. Nothing comes back when the module is fine: the bundler loads it as
// it does.
func (b *builder) siteModule(path string) (api.OnLoadResult, error) {
	rel := b.relative(path)
	if !strings.HasPrefix(rel, project.Source+"/") {
		return api.OnLoadResult{}, nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return api.OnLoadResult{}, err
	}
	if at := waits(rel, string(source)); at != nil {
		return api.OnLoadResult{Errors: []api.Message{*at}}, nil
	}
	return api.OnLoadResult{}, nil
}

// waits returns where a module awaits at its top, or nil when it does not.
// The bundler finds it: told that the code cannot wait at the top, it
// names the first place that does.
func waits(path, source string) *api.Message {
	loader := api.LoaderJS
	switch filepath.Ext(path) {
	case ".ts", ".mts", ".cts":
		loader = api.LoaderTS
	case ".tsx":
		loader = api.LoaderTSX
	case ".jsx":
		loader = api.LoaderJSX
	}
	result := api.Transform(source, api.TransformOptions{
		Loader: loader, Format: api.FormatESModule, Target: api.ESNext,
		Supported:  map[string]bool{"top-level-await": false},
		Sourcefile: path, LogLevel: api.LogLevelSilent,
	})
	for _, message := range result.Errors {
		if strings.HasPrefix(message.Text, "Top-level await is not available") {
			message.Text = topLevelAwait
			return &message
		}
	}
	return nil
}

// topLevelAwait is what Shugo says of a module of the site that waits at its
// top, as of await in a template.
const topLevelAwait = "a module of the site does not wait at its top: a page is computed without waiting, and its data comes from query of the module shugo, which answers at once, so there is nothing to wait for"
