package lua

import (
	"strconv"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/parse"
)

func quote(s string) string {
	var r strings.Builder
	r.WriteByte('\'')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			r.WriteString(`\\`)
		case '\'':
			r.WriteString(`\'`)
		case '\n':
			r.WriteString(`\n`)
		case '\r':
			r.WriteString(`\r`)
		case '\t':
			r.WriteString(`\t`)
		default:
			if c < 32 || c == 127 {
				r.WriteByte('\\')
				r.WriteString(strconv.Itoa(int(c)))
			} else {
				r.WriteByte(c)
			}
		}
	}
	r.WriteByte('\'')
	return r.String()
}

func pri(d uint16) string {
	switch d {
	case 0:
		return "nil"
	case 1:
		return "false"
	case 2:
		return "true"
	default:
		return "nil"
	}
}

func shortInt(d uint16) string {
	return strconv.Itoa(int(int16(d)))
}

func numLit(k parse.KNum) string {
	if k.IsInt {
		return strconv.Itoa(int(k.I))
	}
	return strconv.FormatFloat(k.Float64(), 'g', -1, 64)
}
