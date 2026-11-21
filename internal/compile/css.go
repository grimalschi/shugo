package compile

import (
	"regexp"
	"strconv"
	"strings"
)

// The <style> of a component applies to the component's own elements, as in
// Svelte: every one of its elements gets a class made of a hash, and every
// compound selector is made to need that class — the first with .svelte-…,
// the ones after it with :where(.svelte-…), so that they weigh no more than
// written. :global(…) is left as it is, and so is a :global { … } block. The
// names of @keyframes get the class as their prefix, unless written with
// -global-.
//
// Svelte also drops the selectors that match none of the component's
// elements and puts the class only on the elements some selector matches.
// Here every element gets it; what a page looks like is the same.

// scopeClass returns the class that scopes the styles of a file. Svelte
// makes it of the name of the file the same way.
func scopeClass(path string) string {
	return "svelte-" + svelteHash(path)
}

// svelteHash is the hash of Svelte: djb2 over the characters from the last,
// in base 36.
func svelteHash(text string) string {
	text = strings.ReplaceAll(text, "\r", "")
	hash := uint32(5381)
	units := []rune(text)
	for i := len(units) - 1; i >= 0; i-- {
		hash = ((hash << 5) - hash) ^ uint32(units[i])
	}
	return strconv.FormatUint(uint64(hash), 36)
}

var keyframes = regexp.MustCompile(`@(?:-[a-z]+-)?keyframes\s+([A-Za-z_-][\w-]*)`)

// scopeCSS returns the styles of a component made to apply to it alone.
func scopeCSS(css, hash string) string {
	names := map[string]bool{}
	for _, m := range keyframes.FindAllStringSubmatch(css, -1) {
		if !strings.HasPrefix(m[1], "-global-") {
			names[m[1]] = true
		}
	}
	s := &scoper{src: css, hash: hash, names: names}
	return s.rules(0, len(css), true)
}

type scoper struct {
	src   string
	hash  string
	names map[string]bool // keyframes of the component
}

// rules rewrites the statements in src[from:to]: rules and at-rules.
func (s *scoper) rules(from, to int, scoped bool) string {
	var out strings.Builder
	at := from
	for at < to {
		// White space and comments go as they are.
		next := s.skip(at, to)
		out.WriteString(s.src[at:next])
		at = next
		if at >= to {
			break
		}
		end, open := s.statement(at, to)
		if open < 0 {
			// A statement without a block: @import …; or a stray piece.
			out.WriteString(s.src[at:end])
			at = end
			continue
		}
		prelude := s.src[at:open]
		bodyStart, bodyEnd := open+1, end-1
		trimmed := strings.TrimSpace(prelude)
		switch {
		case strings.HasPrefix(trimmed, "@"):
			name := strings.ToLower(strings.SplitN(strings.TrimPrefix(trimmed, "@"), " ", 2)[0])
			name = strings.TrimSpace(strings.SplitN(name, "(", 2)[0])
			switch {
			case strings.HasSuffix(name, "keyframes"):
				out.WriteString(keyframes.ReplaceAllStringFunc(prelude, func(m string) string {
					sub := keyframes.FindStringSubmatch(m)
					if strings.HasPrefix(sub[1], "-global-") {
						return strings.Replace(m, sub[1], strings.TrimPrefix(sub[1], "-global-"), 1)
					}
					return strings.Replace(m, sub[1], s.hash+"-"+sub[1], 1)
				}))
				out.WriteString(s.src[open:end])
			case name == "media" || name == "supports" || name == "container" || name == "layer" || name == "document" || name == "scope" || name == "starting-style":
				out.WriteString(prelude + "{" + s.rules(bodyStart, bodyEnd, scoped) + "}")
			default:
				out.WriteString(s.src[at:end])
			}
		case trimmed == ":global":
			// :global { … }: what is inside applies to the whole page.
			out.WriteString(s.rules(bodyStart, bodyEnd, false))
		default:
			if scoped {
				out.WriteString(s.selectors(prelude))
			} else {
				out.WriteString(prelude)
			}
			out.WriteString("{" + s.declarations(s.src[bodyStart:bodyEnd]) + "}")
		}
		at = end
	}
	return out.String()
}

// skip returns where the white space and the comments from at end.
func (s *scoper) skip(at, to int) int {
	for at < to {
		switch {
		case isCSSSpace(s.src[at]):
			at++
		case strings.HasPrefix(s.src[at:], "/*"):
			end := strings.Index(s.src[at+2:], "*/")
			if end < 0 {
				return to
			}
			at += end + 4
		default:
			return at
		}
	}
	return at
}

// statement finds the end of the statement at at: after its block, or after
// its ";". It returns where the block opens, or -1 when it has none.
func (s *scoper) statement(at, to int) (end, open int) {
	depth := 0
	open = -1
	for i := at; i < to; i++ {
		c := s.src[i]
		switch {
		case c == '"' || c == '\'':
			i = s.stringEnd(i, to)
		case strings.HasPrefix(s.src[i:], "/*"):
			close := strings.Index(s.src[i+2:to], "*/")
			if close < 0 {
				return to, open
			}
			i += close + 3
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == '{':
			if depth == 0 && open < 0 {
				open = i
			}
			depth++
		case c == '}':
			depth--
			if depth == 0 && open >= 0 {
				return i + 1, open
			}
			if depth < 0 {
				return i + 1, -1
			}
		case c == ';' && depth == 0 && open < 0:
			return i + 1, -1
		}
	}
	return to, open
}

// stringEnd returns the place of the quote that closes the string at i.
func (s *scoper) stringEnd(i, to int) int {
	quote := s.src[i]
	for j := i + 1; j < to; j++ {
		switch s.src[j] {
		case '\\':
			j++
		case quote:
			return j
		}
	}
	return to
}

// selectors scopes a list of selectors: a, b > c.
func (s *scoper) selectors(list string) string {
	var out []string
	for _, one := range splitTop(list, ',') {
		out = append(out, s.selector(one))
	}
	return strings.Join(out, ",")
}

// selector scopes one complex selector, compound by compound.
func (s *scoper) selector(text string) string {
	var out strings.Builder
	first := true
	global := false
	depth := 0
	start := 0
	flush := func(end int) {
		compound := text[start:end]
		switch {
		case compound == "":
		case global:
			out.WriteString(compound)
		case compound == ":global":
			global = true
		case strings.HasPrefix(compound, ":global(") && strings.HasSuffix(compound, ")") && matchingClose(compound, len(":global")) == len(compound)-1:
			out.WriteString(compound[len(":global(") : len(compound)-1])
		default:
			out.WriteString(s.compound(compound, first))
			first = false
		}
		start = end
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c == '"' || c == '\'':
			for j := i + 1; j < len(text); j++ {
				if text[j] == '\\' {
					j++
				} else if text[j] == c {
					i = j
					break
				}
			}
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case depth == 0 && (isCSSSpace(c) || c == '>' || c == '+' || c == '~'):
			flush(i)
			// The combinator and the white space around it go as they are.
			j := i
			for j < len(text) && (isCSSSpace(text[j]) || text[j] == '>' || text[j] == '+' || text[j] == '~') {
				j++
			}
			out.WriteString(text[i:j])
			start = j
			i = j - 1
		}
	}
	flush(len(text))
	return out.String()
}

// compound puts the scoping class into one compound selector: before its
// pseudo-classes and pseudo-elements.
func (s *scoper) compound(text string, first bool) string {
	mark := "." + s.hash
	if !first {
		mark = ":where(." + s.hash + ")"
	}
	if text == "*" {
		return mark
	}
	depth := 0
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == ':' && depth == 0:
			return text[:i] + mark + text[i:]
		}
	}
	return text + mark
}

// declarations renames the keyframes of the component where animations name them.
func (s *scoper) declarations(text string) string {
	if len(s.names) == 0 {
		return text
	}
	var out []string
	for _, declaration := range strings.Split(text, ";") {
		name, value, found := strings.Cut(declaration, ":")
		property := strings.ToLower(strings.TrimSpace(name))
		if found && (property == "animation" || property == "animation-name") {
			for keyframe := range s.names {
				value = regexp.MustCompile(`(^|[\s,])`+regexp.QuoteMeta(keyframe)+`($|[\s,])`).ReplaceAllString(value, "${1}"+s.hash+"-"+keyframe+"${2}")
			}
			declaration = name + ":" + value
		}
		out = append(out, declaration)
	}
	return strings.Join(out, ";")
}

// splitTop splits text at the separator where it stands outside of brackets and strings.
func splitTop(text string, separator byte) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == separator && depth == 0:
			out = append(out, text[start:i])
			start = i + 1
		}
	}
	return append(out, text[start:])
}

// matchingClose returns the place of the ")" that closes the "(" at open.
func matchingClose(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isCSSSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }
