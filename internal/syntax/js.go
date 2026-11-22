package syntax

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The scanner walks JavaScript from esbuild so Shugo can separate statements
// and identify references in the scripts and generated code.

// TokenKind says what a stretch of JavaScript source is.
type TokenKind int

const (
	EOF TokenKind = iota
	Space
	Comment
	Word // an identifier or a keyword
	Number
	String   // a quoted string
	Template // a whole template literal with everything inside it
	Regexp
	Punct // one character of punctuation; "++" and "--" come as one token
)

// Token is one stretch of source: Source[Start:End].
type Token struct {
	Kind       TokenKind
	Start, End int
}

// Scanner reads JavaScript or TypeScript source token by token.
type Scanner struct {
	src string
	pos int
	// value tells whether the code so far ends in something that can be
	// divided: after it "/" is division, otherwise it starts a regular
	// expression.
	value bool
}

// NewScanner starts reading src at offset from.
func NewScanner(src string, from int) *Scanner {
	return &Scanner{src: src, pos: from}
}

// Text returns the source of a token.
func (s *Scanner) Text(t Token) string { return s.src[t.Start:t.End] }

// Words after which an expression starts, so that "/" begins a regular
// expression: `return /a/.test(x)`.
var beforeExpression = map[string]bool{
	"return": true, "typeof": true, "instanceof": true, "in": true, "of": true, "new": true,
	"delete": true, "void": true, "throw": true, "case": true, "do": true, "else": true,
	"yield": true, "await": true,
}

// Next returns the next token. At the end of the source it keeps returning EOF.
func (s *Scanner) Next() (Token, error) {
	start := s.pos
	if start >= len(s.src) {
		return Token{Kind: EOF, Start: start, End: start}, nil
	}
	token := func(kind TokenKind, end int) (Token, error) {
		s.pos = end
		return Token{Kind: kind, Start: start, End: end}, nil
	}
	c := s.src[start]
	switch {
	case isSpace(c):
		end := start + 1
		for end < len(s.src) && isSpace(s.src[end]) {
			end++
		}
		return token(Space, end)

	case c == '/' && strings.HasPrefix(s.src[start:], "//"):
		end := strings.IndexByte(s.src[start:], '\n')
		if end < 0 {
			return token(Comment, len(s.src))
		}
		return token(Comment, start+end)

	case c == '/' && strings.HasPrefix(s.src[start:], "/*"):
		end := strings.Index(s.src[start+2:], "*/")
		if end < 0 {
			return Token{}, &ScanError{Offset: start, Message: "unterminated comment"}
		}
		return token(Comment, start+2+end+2)

	case c == '"' || c == '\'':
		end, err := s.quoted(start)
		if err != nil {
			return Token{}, err
		}
		s.value = true
		return token(String, end)

	case c == '`':
		end, err := s.template(start)
		if err != nil {
			return Token{}, err
		}
		s.value = true
		return token(Template, end)

	case c == '/' && !s.value:
		end, err := s.regexp(start)
		if err != nil {
			return Token{}, err
		}
		s.value = true
		return token(Regexp, end)

	case isDigit(c) || (c == '.' && start+1 < len(s.src) && isDigit(s.src[start+1])):
		end := start + 1
		for end < len(s.src) && (isWordByte(s.src[end]) || s.src[end] == '.') {
			end++
		}
		s.value = true
		return token(Number, end)

	case isWordStart(s.src[start:]):
		end := start
		for end < len(s.src) {
			r, size := utf8.DecodeRuneInString(s.src[end:])
			if !isWordRune(r) {
				break
			}
			end += size
		}
		s.value = !beforeExpression[s.src[start:end]]
		return token(Word, end)
	}

	// Punctuation. What matters is whether a value ends here.
	end := start + 1
	switch {
	case c == ')' || c == ']' || c == '}':
		s.value = true
	case (c == '+' || c == '-') && end < len(s.src) && s.src[end] == c:
		// "a++ / b" divides, "++a" does not end a value; either way the
		// state before the operator is the right one after it.
		end++
	case c == '!' && s.value && !strings.HasPrefix(s.src[end:], "="):
		// TypeScript's "a! / b": the value goes on.
	default:
		s.value = false
	}
	return token(Punct, end)
}

func (s *Scanner) quoted(start int) (int, error) {
	quote := s.src[start]
	for i := start + 1; i < len(s.src); i++ {
		switch s.src[i] {
		case '\\':
			i++
		case '\n':
			return 0, &ScanError{Offset: start, Message: "unterminated string"}
		case quote:
			return i + 1, nil
		}
	}
	return 0, &ScanError{Offset: start, Message: "unterminated string"}
}

func (s *Scanner) template(start int) (int, error) {
	for i := start + 1; i < len(s.src); i++ {
		switch s.src[i] {
		case '\\':
			i++
		case '`':
			return i + 1, nil
		case '$':
			if i+1 < len(s.src) && s.src[i+1] == '{' {
				end, err := Balanced(s.src, i+2)
				if err != nil {
					return 0, err
				}
				i = end // the closing brace
			}
		}
	}
	return 0, &ScanError{Offset: start, Message: "unterminated template literal"}
}

func (s *Scanner) regexp(start int) (int, error) {
	class := false
	for i := start + 1; i < len(s.src); i++ {
		switch c := s.src[i]; {
		case c == '\\':
			i++
		case c == '\n':
			return 0, &ScanError{Offset: start, Message: "unterminated regular expression"}
		case c == '[':
			class = true
		case c == ']':
			class = false
		case c == '/' && !class:
			end := i + 1
			for end < len(s.src) && isWordByte(s.src[end]) {
				end++
			}
			return end, nil
		}
	}
	return 0, &ScanError{Offset: start, Message: "unterminated regular expression"}
}

// Balanced returns the offset of the brace that closes the one before `from`.
func Balanced(src string, from int) (int, error) {
	s := NewScanner(src, from)
	depth := 1
	for {
		t, err := s.Next()
		if err != nil {
			return 0, err
		}
		if t.Kind == EOF {
			return 0, &ScanError{Offset: from - 1, Message: `"{" is never closed`}
		}
		if t.Kind != Punct {
			continue
		}
		switch src[t.Start] {
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return t.Start, nil
			}
		}
	}
}

// ScanError is a place where the source stops being JavaScript.
type ScanError struct {
	Offset  int
	Message string
}

func (e *ScanError) Error() string { return fmt.Sprintf("%s at offset %d", e.Message, e.Offset) }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isWordByte(c byte) bool {
	return c == '_' || c == '$' || isDigit(c) || (c|0x20 >= 'a' && c|0x20 <= 'z')
}

func isWordRune(r rune) bool {
	return r == '_' || r == '$' || r == '#' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isWordStart(src string) bool {
	r, _ := utf8.DecodeRuneInString(src)
	return r == '_' || r == '$' || r == '#' || unicode.IsLetter(r)
}
