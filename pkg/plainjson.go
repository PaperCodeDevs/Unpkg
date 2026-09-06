package pkg

import "bytes"

func looksJSONText(b []byte) bool {
	t := bytes.TrimSpace(bytes.TrimPrefix(b, utf8BOM))
	return len(t) > 0 && (t[0] == '{' || t[0] == '[')
}

func JSONLenient(b []byte) []byte {
	b = bytes.TrimPrefix(b, utf8BOM)
	out := make([]byte, 0, len(b))
	inStr := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case inStr:
			out = append(out, c)
			if c == '\\' && i+1 < len(b) {
				i++
				out = append(out, b[i])
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i+1 < len(b) && b[i+1] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			if end := bytes.Index(b[i+2:], []byte("*/")); end >= 0 {
				i += end + 3
			} else {
				i = len(b)
			}
		case c == ',' && nextNonSpaceCloses(b, i+1):
		default:
			out = append(out, c)
		}
	}
	return out
}

func nextNonSpaceCloses(b []byte, i int) bool {
	for i < len(b) {
		switch {
		case b[i] == ' ' || b[i] == '\t' || b[i] == '\r' || b[i] == '\n':
			i++
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return false
			}
			i += end + 4
		default:
			return b[i] == '}' || b[i] == ']'
		}
	}
	return false
}
