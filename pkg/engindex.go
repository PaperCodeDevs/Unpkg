package pkg

import (
	"crypto/md5"
	"fmt"
	"strings"
)

func parseEngineIndex(idx []byte, data []byte) (*launcherIndex, error) {
	mapStart, nmap, good := findNoPadMap(idx)
	if good < engineMapMinGood {
		return nil, fmt.Errorf("engine map not found")
	}
	mapOff := mapStart - 4
	byName, names, end, err := parseNoPadMapEnd(idx, mapStart, nmap)
	if err != nil {
		return nil, err
	}
	if end != len(idx) {
		return nil, fmt.Errorf("engine map tail")
	}
	start, shader, err := findEngine28(idx, data, mapOff)
	if err != nil {
		return nil, err
	}
	pre := readEngine44(idx, data, 4, start)
	lx := &launcherIndex{
		byName:   byName,
		allNames: names,
		fileBase: start,
		nopad:    true,
		engine:   true,
		shader:   shader,
		pre:      pre,
	}
	skip, _, _, mode, files, streams, err := findLauncherLayout(idx, mapOff, byName, len(data))
	if err == nil {
		lx.files = files
		lx.streams = streams
		lx.fileBase = skip + 4
		lx.resolveMode = mode
	}
	return lx, nil
}

func findEngine28(idx, data []byte, mapOff int) (int, []assetRec, error) {
	maxN := (mapOff - 4) / 28
	for n := maxN; n >= 1000; n-- {
		start := mapOff - n*28
		if start < 4 {
			continue
		}
		if !engine28Probe(idx, data, start, n) {
			continue
		}
		files, hit := readEngine28(idx, data, start, n)
		if hit < n*9/10 {
			continue
		}
		return start, files, nil
	}
	return 0, nil, fmt.Errorf("engine 28B table")
}

func engine28Probe(idx, data []byte, start, n int) bool {
	if n < 1000 {
		return false
	}
	for _, i := range []int{0, 1, n / 2, n - 1} {
		if !engineRecMD5(idx, data, start+i*28, 28) {
			return false
		}
	}
	return true
}

func engineRecMD5(idx, data []byte, b, rec int) bool {
	if b < 0 || b+rec > len(idx) {
		return false
	}
	var h [16]byte
	copy(h[:], idx[b:b+16])
	off := u32le(idx, b+16)
	sz := u32le(idx, b+20)
	if off < uncompOrigin || sz == 0 || sz > 16<<20 {
		return false
	}
	s := int(off - uncompOrigin)
	if s+int(sz) > len(data) {
		return false
	}
	return md5.Sum(data[s:s+int(sz)]) == h
}

func readEngine28(idx, data []byte, start, n int) ([]assetRec, int) {
	files := make([]assetRec, n)
	hit := 0
	for i := 0; i < n; i++ {
		b := start + i*28
		off := u32le(idx, b+16)
		sz := u32le(idx, b+20)
		if off >= uncompOrigin && sz > 0 && int(off-uncompOrigin)+int(sz) <= len(data) {
			files[i] = assetRec{off: off - uncompOrigin, size: sz, flags: u32le(idx, b+24)}
			var h [16]byte
			copy(h[:], idx[b:b+16])
			if md5.Sum(data[files[i].off:files[i].off+files[i].size]) == h {
				hit++
			}
		}
	}
	return files, hit
}

func readEngine44(idx, data []byte, start, end int) []assetRec {
	n := (end - start) / assetRecBytes
	if n <= 0 {
		return nil
	}
	files := make([]assetRec, n)
	for i := 0; i < n; i++ {
		b := start + i*assetRecBytes
		off := u32le(idx, b+16)
		sz := u32le(idx, b+20)
		if off < uncompOrigin || sz == 0 || sz > 16<<20 {
			continue
		}
		s := int(off - uncompOrigin)
		if s+int(sz) > len(data) {
			continue
		}
		var h [16]byte
		copy(h[:], idx[b:b+16])
		if md5.Sum(data[s:s+int(sz)]) != h {
			continue
		}
		files[i] = assetRec{off: uint32(s), size: sz, flags: u32le(idx, b+40)}
	}
	return files
}

func (lx *launcherIndex) lookupEngine(data []byte, name string, flag uint32) ([]byte, error) {
	if lx == nil || !lx.engine {
		return nil, fmt.Errorf("no engine")
	}
	if strings.HasPrefix(strings.ToLower(name), "d3d11/") {
		return sliceFlag(data, lx.shader, flag)
	}
	i := int(flag)
	if i >= 0 && i < len(lx.pre) && lx.pre[i].size > 0 {
		rec := lx.pre[i]
		if int(rec.off)+int(rec.size) <= len(data) {
			return append([]byte(nil), data[rec.off:rec.off+rec.size]...), nil
		}
	}
	return nil, fmt.Errorf("engine miss")
}

func sliceFlag(data []byte, files []assetRec, flag uint32) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("empty table")
	}
	i := int(flag)
	if i >= len(files) {
		i -= len(files)
	}
	if i < 0 || i >= len(files) {
		return nil, fmt.Errorf("flag")
	}
	rec := files[i]
	if rec.size == 0 || int(rec.off)+int(rec.size) > len(data) {
		return nil, fmt.Errorf("range")
	}
	return append([]byte(nil), data[rec.off:rec.off+rec.size]...), nil
}
