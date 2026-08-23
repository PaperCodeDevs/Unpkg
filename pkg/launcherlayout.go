package pkg

import "fmt"

func findLauncherLayout(idx []byte, mapOff int, byName map[string]uint32, dataLen int) (skip, nfile, nstream, mode int, files []assetRec, streams []streamRec, err error) {
	bestHit := 0
	var best struct {
		skip, nf, ns, mode int
	}
	var bestFiles []assetRec
	var bestStreams []streamRec
	for ns := 0; ns < 20000; ns++ {
		endFiles := mapOff - 4 - ns*streamRecBytes
		if endFiles <= 20 {
			break
		}
		for sk := 0; sk <= 32; sk++ {
			nf := (endFiles - sk - 4) / assetRecBytes
			if nf <= 0 || sk+4+nf*assetRecBytes != endFiles {
				continue
			}
			fs := make([]assetRec, nf)
			for i := 0; i < nf; i++ {
				b := sk + 4 + i*assetRecBytes
				fs[i] = assetRec{
					off:   u32le(idx, b+16),
					size:  u32le(idx, b+20),
					flags: u32le(idx, b+40),
				}
			}
			ss := make([]streamRec, ns)
			for i := 0; i < ns; i++ {
				b := endFiles + 4 + i*streamRecBytes
				ss[i] = streamRec{
					off:   u32le(idx, b),
					size:  u32le(idx, b+4),
					flags: u32le(idx, b+8),
				}
			}
			for m := 0; m < 3; m++ {
				hit := scoreLauncherMap(byName, fs, ss, dataLen, m)
				if hit > bestHit {
					bestHit = hit
					best.skip, best.nf, best.ns, best.mode = sk, nf, ns, m
					bestFiles = fs
					bestStreams = ss
				}
			}
		}
	}
	if bestHit < 100 {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("launcher layout hit %d", bestHit)
	}
	return best.skip, best.nf, best.ns, best.mode, bestFiles, bestStreams, nil
}

func scoreLauncherMap(byName map[string]uint32, files []assetRec, streams []streamRec, dataLen, mode int) int {
	hit := 0
	for _, flag := range byName {
		if _, _, ok := launcherResolve(flag, files, streams, dataLen, mode); ok {
			hit++
		}
	}
	return hit
}

func launcherResolve(flag uint32, files []assetRec, streams []streamRec, dataLen, mode int) (off, size uint32, ok bool) {
	switch mode {
	case 1:
		i := int(flag)
		if i >= 0 && i < len(streams) {
			off, size = streams[i].off, streams[i].size
			if size > 0 && int(off)+int(size) <= dataLen {
				return off, size, true
			}
		}
		return 0, 0, false
	case 2:
		i := int(flag)
		if i >= len(files) {
			i -= len(files)
		}
		if i >= 0 && i < len(streams) {
			off, size = streams[i].off, streams[i].size
			if size > 0 && int(off)+int(size) <= dataLen {
				return off, size, true
			}
		}
		if int(flag) >= 0 && int(flag) < len(files) {
			rec := files[int(flag)]
			if rec.size > 0 && int(rec.off)+int(rec.size) <= dataLen {
				return rec.off, rec.size, true
			}
		}
		return 0, 0, false
	default:
		return materialResolveChecked(flag, files, streams, dataLen)
	}
}

func materialResolveChecked(flag uint32, files []assetRec, streams []streamRec, dataLen int) (off, size uint32, ok bool) {
	if flag&0x80000000 != 0 {
		i := int(flag & 0x7fffffff)
		if i >= 0 && i < len(streams) {
			off, size = streams[i].off, streams[i].size
			if size > 0 && int(off)+int(size) <= dataLen {
				return off, size, true
			}
		}
		return 0, 0, false
	}
	i := int(flag)
	if i >= 0 && i < len(files) {
		rec := files[i]
		if rec.size > 1 && int(rec.off)+int(rec.size) <= dataLen {
			return rec.off, rec.size, true
		}
	}
	j := i - len(files)
	if j >= 0 && j < len(streams) {
		rec := streams[j]
		if rec.size > 0 && int(rec.off)+int(rec.size) <= dataLen {
			return rec.off, rec.size, true
		}
	}
	if i >= 0 && i < len(streams) {
		rec := streams[i]
		if rec.size > 0 && int(rec.off)+int(rec.size) <= dataLen {
			return rec.off, rec.size, true
		}
	}
	if i >= 0 && i < len(files) {
		rec := files[i]
		if rec.size > 0 && int(rec.off)+int(rec.size) <= dataLen {
			return rec.off, rec.size, true
		}
	}
	return 0, 0, false
}
