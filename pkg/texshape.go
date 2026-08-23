package pkg

import (
	"strconv"
	"strings"
)

const (
	DrawCube  = 0
	DrawCross = 1
	DrawFlat  = 2
	DrawPane  = 3
	DrawMesh  = 4
	DrawFence = 5
)

func NormalizeType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || r == '_' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}

func DrawShape(typ string) uint8 {
	t := NormalizeType(typ)
	if t == "" {
		return DrawCube
	}
	if _, ok := drawFlat[t]; ok {
		return DrawFlat
	}
	if _, ok := drawPane[t]; ok {
		return DrawPane
	}
	if prefixOf(t, panePref) {
		return DrawPane
	}
	if prefixOf(t, flatPref) {
		return DrawFlat
	}
	return DrawCube
}

func prefixOf(t string, ps []string) bool {
	for _, p := range ps {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

func isKeepType(t string) bool {
	if t == "" {
		return false
	}
	if _, ok := drawFlat[t]; ok {
		return true
	}
	if _, ok := drawPane[t]; ok {
		return true
	}
	return prefixOf(t, panePref) || prefixOf(t, flatPref)
}

func pickType(fs []string) string {
	try := func(i int) string {
		if i < 0 || i >= len(fs) {
			return ""
		}
		return NormalizeType(strings.TrimSpace(fs[i]))
	}
	if t := try(7); t != "" {
		return t
	}
	for _, i := range []int{6, 8, 4, 5} {
		if t := try(i); isKeepType(t) {
			return t
		}
	}
	return ""
}

func LoadBlockShapes(basePath, patchPath string) map[int]uint8 {
	out := map[int]uint8{}
	scanShapeFile(basePath, out)
	scanShapeFile(patchPath, out)
	return out
}

func LoadBlockTypes(basePath, patchPath string) map[int]string {
	out := map[int]string{}
	scanTypeFile(basePath, out)
	scanTypeFile(patchPath, out)
	dt, _ := LoadBlockDefDump(DefaultDefDumpPaths()...)
	for id, t := range dt {
		if out[id] == "" {
			out[id] = t
		}
	}
	return out
}

func scanTypeFile(path string, out map[int]string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	p, err := ParseFile(path)
	if err != nil {
		return
	}
	plain, err := DecompressLZ4Block(p.Data, 128<<20)
	if err != nil {
		return
	}
	raw := SliceCsvByHeader(plain, CsvBlockDefHeader)
	lines := strings.Split(string(raw), "\n")
	for i, ln := range lines {
		if i == 0 {
			continue
		}
		ln = strings.TrimRight(ln, "\r")
		if ln == "" || ln[0] < '0' || ln[0] > '9' {
			continue
		}
		fs := strings.Split(ln, ",")
		if len(fs) < 8 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(fs[0]))
		if err != nil || id < 0 {
			continue
		}
		typ := pickType(fs)
		if typ == "" {
			continue
		}
		out[id] = typ
	}
}

func scanShapeFile(path string, out map[int]uint8) {
	if strings.TrimSpace(path) == "" {
		return
	}
	p, err := ParseFile(path)
	if err != nil {
		return
	}
	plain, err := DecompressLZ4Block(p.Data, 128<<20)
	if err != nil {
		return
	}
	raw := SliceCsvByHeader(plain, CsvBlockDefHeader)
	lines := strings.Split(string(raw), "\n")
	for i, ln := range lines {
		if i == 0 {
			continue
		}
		ln = strings.TrimRight(ln, "\r")
		if ln == "" || ln[0] < '0' || ln[0] > '9' {
			continue
		}
		fs := strings.Split(ln, ",")
		if len(fs) < 8 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(fs[0]))
		if err != nil || id < 0 {
			continue
		}
		typ := pickType(fs)
		if typ == "" {
			if _, ok := out[id]; ok {
				continue
			}
		}
		shape := DrawShape(typ)
		if prev, ok := out[id]; ok && shape == DrawCube && prev != DrawCube {
			continue
		}
		out[id] = shape
	}
}

var drawPane = map[string]struct{}{
	"door": {}, "simpledoor": {}, "windows": {}, "glasspane": {}, "screen": {},
	"inkscreen": {}, "highdoor": {}, "centerdoor": {},
}

var drawFlat = map[string]struct{}{
	"waterlily": {}, "lilypad": {}, "lilymodel": {},
}

var panePref = []string{"door", "window", "screen", "glasspane"}
var flatPref = []string{"lily"}
