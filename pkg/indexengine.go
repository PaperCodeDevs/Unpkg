package pkg

import (
	"fmt"
	"strings"
)

const (
	engineMapMinGood = 1000
	engineFileSkip   = 4
)

type launcherIndex struct {
	files       []assetRec
	streams     []streamRec
	byName      map[string]uint32
	allNames    []string
	fileBase    int
	nopad       bool
	material    bool
	resolveMode int
}

func parseLauncherIndex(idx []byte, dataLen int) (*launcherIndex, error) {
	mapStart, nmap, good := findNoPadMap(idx)
	if good < engineMapMinGood {
		return nil, fmt.Errorf("launcher map not found")
	}
	mapOff := mapStart - 4
	byName := map[string]uint32{}
	pos := mapStart
	for i := 0; i < nmap; i++ {
		if pos+8 > len(idx) {
			return nil, fmt.Errorf("map short %d", i)
		}
		nl := int(u32le(idx, pos))
		pos += 4
		if nl < 1 || nl > 400 || pos+nl+4 > len(idx) {
			return nil, fmt.Errorf("map nl %d at %d", nl, i)
		}
		name := string(idx[pos : pos+nl])
		pos += nl
		flag := u32le(idx, pos)
		pos += 4
		byName[name] = flag
	}
	fileEnd := mapOff
	if fileEnd < engineFileSkip {
		return nil, fmt.Errorf("file section short")
	}
	skip, _, _, mode, files, streams, err := findLauncherLayout(idx, mapOff, byName, dataLen)
	if err != nil {
		return nil, err
	}
	return &launcherIndex{
		files:       files,
		streams:     streams,
		byName:      byName,
		fileBase:    skip + 4,
		nopad:       true,
		resolveMode: mode,
	}, nil
}

func findNoPadMap(idx []byte) (mapStart, nmap, good int) {
	bestOff, bestGood, bestN := 0, 0, 0
	for off := 0; off+8 < len(idx); off += 4 {
		n := int(u32le(idx, off))
		if n < 500 || n > 50000 {
			continue
		}
		pos := off + 4
		g := 0
		for i := 0; i < n; i++ {
			if pos+8 > len(idx) {
				g = 0
				break
			}
			nl := int(u32le(idx, pos))
			if nl < 2 || nl > 300 || pos+4+nl+4 > len(idx) {
				g = 0
				break
			}
			name := string(idx[pos+4 : pos+4+nl])
			if !asciiPathName(name) {
				g = 0
				break
			}
			if strings.HasPrefix(name, "d3d11/") || strings.Contains(name, "Material") || strings.Contains(name, "AssetsCache") {
				g++
			}
			pos = pos + 4 + nl + 4
		}
		if g > bestGood {
			bestGood = g
			bestOff = off
			bestN = n
		}
	}
	return bestOff + 4, bestN, bestGood
}

func asciiPathName(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func (lx *launcherIndex) lookup(data []byte, name string) ([]byte, error) {
	flag, ok := lx.byName[name]
	if !ok {
		if alt, ok2 := lx.byName[strings.ToLower(name)]; ok2 {
			flag, ok = alt, true
		}
	}
	if !ok {
		return nil, fmt.Errorf("not found: %s", name)
	}
	if flag == ^uint32(0) {
		return nil, fmt.Errorf("no index: %s", name)
	}
	var off, size uint32
	off, size, ok2 := launcherResolve(flag, lx.files, lx.streams, len(data), lx.resolveMode)
	if !ok2 {
		return nil, fmt.Errorf("bad flag 0x%x", flag)
	}
	if size == 0 {
		return nil, fmt.Errorf("empty")
	}
	if int(off)+int(size) > len(data) {
		return nil, fmt.Errorf("range")
	}
	return append([]byte(nil), data[off:off+size]...), nil
}

func (lx *launcherIndex) names(prefix string) []string {
	if len(lx.allNames) > 0 {
		out := make([]string, 0)
		for _, n := range lx.allNames {
			if prefix == "" || strings.HasPrefix(n, prefix) {
				out = append(out, n)
			}
		}
		return out
	}
	out := make([]string, 0, len(lx.byName))
	for n := range lx.byName {
		if prefix == "" || strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	return out
}
