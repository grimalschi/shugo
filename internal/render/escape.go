package render

// appendEscaped appends text so that it is text in HTML, between tags and
// inside a quoted attribute alike.
func appendEscaped(dst, text []byte) []byte {
	from := 0
	for i, c := range text {
		var entity string
		switch c {
		case '&':
			entity = "&amp;"
		case '<':
			entity = "&lt;"
		case '>':
			entity = "&gt;"
		case '"':
			entity = "&#34;"
		case '\'':
			entity = "&#39;"
		default:
			continue
		}
		dst = append(append(dst, text[from:i]...), entity...)
		from = i + 1
	}
	return append(dst, text[from:]...)
}

// What an address is replaced with when following it could run code.
var blockedAddress = []byte("about:invalid#blocked")

// Schemes that only name a place to go to or a thing to show.
var safeSchemes = map[string]bool{"http": true, "https": true, "mailto": true, "tel": true, "sms": true, "ftp": true}

// safeAddress tells whether an address can go into href or src: it has no
// scheme at all, or one that cannot run code. A browser drops spaces, tabs
// and line breaks from a scheme before it reads it, so they are dropped here
// too: "java\nscript:" is "javascript:".
func safeAddress(address []byte) bool {
	var scheme []byte
	for _, c := range address {
		switch {
		case c == ':':
			name := string(scheme)
			return safeSchemes[name] || (name == "data" && dataImage(address))
		case c == '/' || c == '?' || c == '#':
			return true
		case c <= ' ':
		case c >= 'A' && c <= 'Z':
			scheme = append(scheme, c+('a'-'A'))
		default:
			scheme = append(scheme, c)
		}
	}
	return true
}

// dataImage tells whether a data: address holds a picture that is not SVG:
// an SVG file can carry scripts.
func dataImage(address []byte) bool {
	const prefix = "data:image/"
	rest := string(address)
	if len(rest) < len(prefix) || rest[:len(prefix)] != prefix {
		return false
	}
	rest = rest[len(prefix):]
	for _, kind := range []string{"png", "jpeg", "gif", "webp", "avif"} {
		if len(rest) > len(kind) && rest[:len(kind)] == kind && (rest[len(kind)] == ';' || rest[len(kind)] == ',') {
			return true
		}
	}
	return false
}
