package pkg

import (
	"sort"
	"strings"
)

func ScanMaterialPaths(data []byte) []string {
	seen := map[string]bool{}
	var out []string
	lower := strings.ToLower(string(data))
	add := func(start, hashStart int) {
		hashEnd := hashStart
		for hashEnd < len(lower) && ((lower[hashEnd] >= '0' && lower[hashEnd] <= '9') || (lower[hashEnd] >= 'a' && lower[hashEnd] <= 'f')) {
			hashEnd++
		}
		if hashEnd-hashStart != 40 {
			return
		}
		path := lower[start:hashEnd]
		if seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	prefix := "assetscache/shaders/d3d11/"
	for i := 0; i < len(lower); {
		j := strings.Index(lower[i:], prefix)
		if j < 0 {
			break
		}
		start := i + j
		add(start, start+len(prefix))
		i = start + len(prefix)
	}
	p2 := "d3d11/"
	for i := 0; i < len(lower); {
		j := strings.Index(lower[i:], p2)
		if j < 0 {
			break
		}
		start := i + j
		if start >= len("assetscache/shaders/") && lower[start-len("assetscache/shaders/"):start] == "assetscache/shaders/" {
			i = start + len(p2)
			continue
		}
		add(start, start+len(p2))
		i = start + len(p2)
	}
	sort.Strings(out)
	return out
}

func ScanIndexPaths(indexPlain []byte) []string {
	return ScanMaterialPaths(indexPlain)
}
