package rainbow

import (
	"encoding/binary"
	"fmt"
	"math"
)

func matHead(raw []byte) (uint32, uint32, []byte, int, error) {
	if len(raw) < matMinSize {
		return 0, 0, nil, 0, fmt.Errorf("mat short")
	}
	if binary.LittleEndian.Uint32(raw[0:]) != matVer {
		return 0, 0, nil, 0, fmt.Errorf("mat ver")
	}
	pay := int(binary.LittleEndian.Uint32(raw[4:]))
	if pay < matMinSize || 8+pay > len(raw) {
		return 0, 0, nil, 0, fmt.Errorf("mat payload")
	}
	id := binary.LittleEndian.Uint32(raw[8:])
	typ := binary.LittleEndian.Uint32(raw[12:])
	if binary.LittleEndian.Uint64(raw[16:]) != 0 {
		return 0, 0, nil, 0, fmt.Errorf("mat pad")
	}
	if binary.LittleEndian.Uint32(raw[matDupOff:]) != id || binary.LittleEndian.Uint32(raw[matDupOff+4:]) != typ {
		return 0, 0, nil, 0, fmt.Errorf("mat id")
	}
	n := int(binary.LittleEndian.Uint32(raw[matTableOff:]))
	if n < 1 || n > 64 {
		return 0, 0, nil, 0, fmt.Errorf("mat table")
	}
	start := 52 + n*6
	if start%4 != 0 {
		start += 4 - start%4
	}
	if start >= 8+pay {
		return 0, 0, nil, 0, fmt.Errorf("mat table")
	}
	return id, typ, raw[matHashOff : matHashOff+8], start, nil
}

func matFindMaps(b []byte, start, pend int) ([]MatTex, []MatFloat, []MatColor, int, int, bool) {
	lim := start + matScanSpan
	if lim > pend-4 {
		lim = pend - 4
	}
	for off := start; off+4 <= lim; off += 4 {
		texs, floats, colors, end, ok := matMaps(b, off, pend)
		if ok {
			return texs, floats, colors, off, end, true
		}
	}
	return nil, nil, nil, 0, 0, false
}

func matMaps(b []byte, off, pend int) ([]MatTex, []MatFloat, []MatColor, int, bool) {
	ntex := int(u32le(b, off))
	if ntex < 1 || ntex > matMaxTex {
		return nil, nil, nil, 0, false
	}
	pos := off + 4
	texs := make([]MatTex, 0, ntex)
	for i := 0; i < ntex; i++ {
		t, npos, ok := matTex(b, pos)
		if !ok {
			return nil, nil, nil, 0, false
		}
		texs = append(texs, t)
		pos = npos
	}
	nfloat := int(u32le(b, pos))
	if nfloat < 0 || nfloat > matMaxFloat || pos+4 > pend {
		return nil, nil, nil, 0, false
	}
	pos += 4
	floats := make([]MatFloat, 0, nfloat)
	for i := 0; i < nfloat; i++ {
		name, j, ok := matStr(b, pos)
		if !ok || !matIdent(name) || j+4 > pend {
			return nil, nil, nil, 0, false
		}
		floats = append(floats, MatFloat{Name: name, Value: f32le(b, j)})
		pos = j + 4
	}
	ncol := int(u32le(b, pos))
	if ncol < 0 || ncol > matMaxColor || pos+4 > pend {
		return nil, nil, nil, 0, false
	}
	pos += 4
	colors := make([]MatColor, 0, ncol)
	for i := 0; i < ncol; i++ {
		name, j, ok := matStr(b, pos)
		if !ok || !matIdent(name) || j+16 > pend {
			return nil, nil, nil, 0, false
		}
		colors = append(colors, MatColor{Name: name, RGBA: [4]float32{f32le(b, j), f32le(b, j+4), f32le(b, j+8), f32le(b, j+12)}})
		pos = j + 16
	}
	return texs, floats, colors, pos, true
}

func matTex(b []byte, i int) (MatTex, int, bool) {
	var t MatTex
	name, j, ok := matStr(b, i)
	if !ok || !matIdent(name) || j+matTexBody > len(b) {
		return t, i, false
	}
	if u32le(b, j+20) != 1 || u32le(b, j+24) != 0 {
		return t, i, false
	}
	t.Name = name
	t.Scale = [2]float32{f32le(b, j), f32le(b, j+4)}
	t.Offset = [2]float32{f32le(b, j+8), f32le(b, j+12)}
	t.Tag = u32le(b, j+16)
	copy(t.Ref[:], b[j+28:j+44])
	return t, j + matTexBody, true
}

func matKeys(b []byte, pos, pend int, m *Mat) {
	if pos+matKeyLead+12 > pend {
		return
	}
	copy(m.Shader[:], b[pos+8:pos+24])
	p := pos + matKeyLead
	if u32le(b, p) != 2 {
		return
	}
	p += 4
	p += 4
	n0 := int(u32le(b, p))
	p += 4
	if n0 < 1 || n0 > matMaxKey {
		return
	}
	en := make([]string, 0, n0)
	for i := 0; i < n0; i++ {
		s, np, ok := matCStr(b, p)
		if !ok {
			return
		}
		en = append(en, s)
		p = np
	}
	p = matAlign(p)
	if p+12 > pend {
		return
	}
	p += 8
	n1 := int(u32le(b, p))
	p += 4
	if n1 < 1 || n1 > matMaxKey {
		return
	}
	keys := make([]MatKey, 0, n1)
	for i := 0; i < n1; i++ {
		s, j, ok := matStr(b, p)
		if !ok || j >= len(b) {
			return
		}
		keys = append(keys, MatKey{Name: s, On: b[j] != 0})
		p = j + 1
	}
	m.Enabled = en
	m.Keywords = keys
}

func matName(b []byte, start, end int) string {
	name := ""
	for i := start; i+4 < end; i += 4 {
		s, j, ok := matStr(b, i)
		if !ok || j > end || !matDCC(s) {
			continue
		}
		name = s
		i = j - 4
	}
	return name
}

func matStr(b []byte, i int) (string, int, bool) {
	if i < 0 || i+4 > len(b) {
		return "", i, false
	}
	n := int(u32le(b, i))
	if n < 1 || n > matMaxStr || i+4+n > len(b) {
		return "", i, false
	}
	s := b[i+4 : i+4+n]
	if !matPrint(s) {
		return "", i, false
	}
	return string(s), matAlign(i + 4 + n), true
}

func matCStr(b []byte, i int) (string, int, bool) {
	if i < 0 || i+4 > len(b) {
		return "", i, false
	}
	n := int(u32le(b, i))
	if n < 1 || n > matMaxStr || i+4+n >= len(b) {
		return "", i, false
	}
	s := b[i+4 : i+4+n]
	if !matPrint(s) || b[i+4+n] != 0 {
		return "", i, false
	}
	return string(s), i + 4 + n + 1, true
}

func matPrint(s []byte) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func matIdent(s string) bool {
	if len(s) < 2 || len(s) > 80 {
		return false
	}
	c0 := s[0]
	if !(c0 == '_' || c0 >= 'A' && c0 <= 'Z' || c0 >= 'a' && c0 <= 'z') {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func matDCC(s string) bool {
	if len(s) < 3 || len(s) > 80 {
		return false
	}
	sep := false
	low := false
	us := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '#' || c == '.' || c == '-' {
			sep = true
			continue
		}
		if c >= 'a' && c <= 'z' {
			low = true
			continue
		}
		if c == '_' {
			us = true
			continue
		}
		if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	if sep {
		return true
	}
	return low && !us && s[0] >= 'A' && s[0] <= 'Z'
}

func u32le(b []byte, i int) uint32 {
	if i+4 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[i:])
}

func f32le(b []byte, i int) float32 {
	return math.Float32frombits(u32le(b, i))
}

func matAlign(n int) int {
	return (n + 3) &^ 3
}
