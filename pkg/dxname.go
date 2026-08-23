package pkg

import (
	"bytes"
	"encoding/binary"
	"strings"
)

func dxLenNames(b []byte, lim int) []string {
	var out []string
	for i := 0; i+4 <= len(b) && len(out) < lim; {
		n := int(binary.LittleEndian.Uint32(b[i:]))
		if n < 2 || n > dxMaxName || i+4+n > len(b) || !dxIdent(b[i+4:i+4+n]) {
			i++
			continue
		}
		out = append(out, string(b[i+4:i+4+n]))
		i += 4 + n
	}
	return uniqDX(out)
}

func dxIdent(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for i, c := range b {
		if c == '$' && i == 0 {
			continue
		}
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.' {
			continue
		}
		return false
	}
	c0 := b[0]
	return (c0 >= 'A' && c0 <= 'Z') || (c0 >= 'a' && c0 <= 'z') || c0 == '$' || c0 == '_'
}

func dxZString(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		n = len(b)
	}
	if n == 0 || n > dxMaxName || !dxIdent(b[:n]) {
		return ""
	}
	return string(b[:n])
}

func dxHLSLNames(b []byte) []string {
	var out []string
	for _, s := range dxASCIIIdents(b, 6) {
		low := strings.ToLower(s)
		if strings.HasSuffix(low, ".hlsl") || strings.HasSuffix(low, ".hlsli") {
			out = append(out, s)
		}
	}
	return uniqDX(out)
}

func dxASCIIIdents(b []byte, minn int) []string {
	var out []string
	i := 0
	for i < len(b) {
		c := b[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_') {
			i++
			continue
		}
		j := i + 1
		for j < len(b) {
			d := b[j]
			if (d >= 'A' && d <= 'Z') || (d >= 'a' && d <= 'z') || (d >= '0' && d <= '9') || d == '_' || d == '.' {
				j++
				continue
			}
			break
		}
		if j-i >= minn {
			out = append(out, string(b[i:j]))
		}
		i = j
	}
	return uniqDX(out)
}

func dxPipeNames(b []byte) []string {
	var out []string
	for _, s := range dxLenNames(b, 200) {
		if strings.Contains(s, "Pipeline") || strings.HasSuffix(s, "VS") || strings.HasSuffix(s, "PS") || strings.HasSuffix(s, "CS") {
			out = append(out, s)
		}
	}
	return out
}

func uniqDX(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
