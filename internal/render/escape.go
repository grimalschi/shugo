package render

// appendEscaped appends text the way Svelte escapes it: "&" and "<"
// between tags, and the double quote too inside an attribute, which is
// always written in double quotes.
func appendEscaped(dst, text []byte, attribute bool) []byte {
	from := 0
	for i, c := range text {
		var entity string
		switch {
		case c == '&':
			entity = "&amp;"
		case c == '<':
			entity = "&lt;"
		case c == '"' && attribute:
			entity = "&quot;"
		default:
			continue
		}
		dst = append(append(dst, text[from:i]...), entity...)
		from = i + 1
	}
	return append(dst, text[from:]...)
}
