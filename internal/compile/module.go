package compile

import (
	"encoding/json"
	"regexp"
	"strings"
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
