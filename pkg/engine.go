package pkg

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	engineObjMark    = "\x01\x00\x01\x00\x04\x0c\x00"
	engineMinStrLen  = 4
	engineMaxStrLen  = 260
	engineInterestFn = "catalog.txt"
)

func ScanRainbowStrings(data []byte) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		if !utf8.ValidString(s) {
			return
		}
		if !rainbowStringInteresting(s) {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for i := 0; i+4 <= len(data); i++ {
		n := int(binary.LittleEndian.Uint32(data[i:]))
		if n < engineMinStrLen || n > engineMaxStrLen || i+4+n > len(data) {
			continue
		}
		raw := data[i+4 : i+4+n]
		if !asciiPathLike(raw) {
			continue
		}
		add(string(raw))
	}
	for i := 0; i < len(data); i++ {
		if data[i] < 32 || data[i] > 126 {
			continue
		}
		j := i
		for j < len(data) && data[j] >= 32 && data[j] <= 126 {
			j++
		}
		if j-i >= 8 {
			add(string(data[i:j]))
		}
		i = j
	}
	sort.Strings(out)
	return out
}

func rainbowStringInteresting(s string) bool {
	if strings.Contains(s, "Materials/") || strings.Contains(s, "Material/") {
		return true
	}
	if strings.HasPrefix(s, "d3d11/") || strings.Contains(s, "/d3d11/") {
		return true
	}
	if strings.Contains(s, "#include") || strings.Contains(s, "DECLARE_") {
		return true
	}
	if strings.Contains(s, "VS") || strings.Contains(s, "PS") {
		if strings.Contains(s, "Postprocess") || strings.Contains(s, "Forward") || strings.Contains(s, "Bloom") {
			return true
		}
	}
	if strings.HasPrefix(s, "UI/") {
		return true
	}
	if strings.Contains(s, ".templatemat") || strings.Contains(s, ".hlsli") {
		return true
	}
	return false
}

func asciiPathLike(b []byte) bool {
	if len(b) < engineMinStrLen {
		return false
	}
	slash := false
	for _, c := range b {
		if c == '/' || c == '\\' {
			slash = true
		}
		if c != '/' && c != '\\' && c != '_' && c != '.' && c != '-' {
			if c < 32 || c > 126 {
				return false
			}
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return false
			}
		}
	}
	return slash || strings.HasSuffix(string(b), ".templatemat")
}

func CountEngineObjectMarks(data []byte) int {
	return strings.Count(string(data), engineObjMark)
}

func DumpEngineCatalog(pkgPath, outDir string) (string, int, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", 0, err
	}
	p, err := ParseFile(pkgPath)
	if err != nil {
		return "", 0, err
	}
	strs := ScanRainbowStrings(p.Data)
	out := filepath.Join(outDir, engineInterestFn)
	var sb strings.Builder
	for _, s := range strs {
		sb.WriteString(s)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(out, []byte(sb.String()), 0o644); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "data.bin"), p.Data, 0o644); err != nil {
		return "", 0, err
	}
	plain, err := decodePkgIndex(p.Index)
	if err == nil {
		_ = os.WriteFile(filepath.Join(outDir, "index_plain.bin"), plain, 0o644)
	}
	var objs []EngineObject
	lookOK, lookN := 0, 0
	if rd, e := OpenReader(p); e == nil {
		objs = rd.EngineObjects()
		ns := rd.Names("")
		lookN = len(ns)
		for _, n := range ns {
			if _, e2 := rd.Lookup(n); e2 == nil {
				lookOK++
			}
		}
	}
	var ob strings.Builder
	for _, o := range objs {
		fmt.Fprintf(&ob, "%d\t0x%x\t%d\t%s\t%s\t%d\n", o.Offset, o.Type, o.Count, o.ID, o.Name, len(o.Fields))
	}
	_ = os.WriteFile(filepath.Join(outDir, "objects.tsv"), []byte(ob.String()), 0o644)
	meta := fmt.Sprintf("version=%d subtype=%d data=%d index=%d strings=%d objmark=%d objects=%d lookup=%d/%d ratio=%.3f\n",
		p.Version, p.Subtype, len(p.Data), len(p.Index), len(strs), CountEngineObjectMarks(p.Data), len(objs), lookOK, lookN, p.ReadableRatio(8000))
	if err := os.WriteFile(filepath.Join(outDir, "meta.txt"), []byte(meta), 0o644); err != nil {
		return "", 0, err
	}
	return out, len(strs), nil
}
