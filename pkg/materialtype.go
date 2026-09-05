package pkg

import "encoding/binary"

const lenStrMax = 260

func MaterialStageName(st int) string {
	switch st {
	case 0:
		return "ps"
	case 1:
		return "vs"
	case 2:
		return "gs"
	case 3:
		return "hs"
	case 4:
		return "ds"
	case 5:
		return "cs"
	}
	return ""
}

func MaterialVarClass(v DXVar) string {
	switch v.Type {
	case 1:
		return "vector"
	case 4:
		return "matrix"
	}
	return ""
}

func DXVarType(v DXVar) string {
	switch v.Type {
	case 0:
		return "float"
	case 1:
		return "int"
	}
	return ""
}

func lenStrAt(b []byte, off int) string {
	if off < 0 || off+4 > len(b) {
		return ""
	}
	n := int(binary.LittleEndian.Uint32(b[off:]))
	if n < 1 || n > lenStrMax || off+4+n > len(b) {
		return ""
	}
	s := b[off+4 : off+4+n]
	for _, c := range s {
		if c < 32 || c > 126 {
			return ""
		}
	}
	return string(s)
}

func lenStrings(b []byte, lim int) []string {
	var out []string
	for i := 0; i+4 <= len(b) && len(out) < lim; {
		s := lenStrAt(b, i)
		if s == "" || len(s) < 2 {
			i++
			continue
		}
		out = append(out, s)
		i += 4 + len(s)
	}
	return uniqDX(out)
}
