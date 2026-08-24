package rainbow

import (
	"fmt"
	"math"
)

func parsePacked(name string, bd []byte, ib int, subs []Submesh) (*Mesh, error) {
	if ib+12 > len(bd) || u32(bd, ib+8) != 0 {
		return nil, fmt.Errorf("mesh pack ib")
	}
	nidxH := int(u32(bd, 8))
	nvertH := int(u32(bd, 24))
	if nvertH <= 0 || nvertH > maxVert || nidxH <= 0 || nidxH > maxIndex {
		return nil, fmt.Errorf("mesh pack count")
	}
	decl := ib + 12
	if decl+8 > len(bd) || int(u32(bd, decl+4)) != declSlots {
		return nil, fmt.Errorf("mesh pack decl")
	}
	off := decl + 8 + declSlots*4 + 4
	var pos, nrm, uv []float32
	var idx []uint32
	for off+8 <= len(bd) {
		off = skipPad(bd, off, nvertH, nidxH)
		if off+8 > len(bd) {
			break
		}
		count := int(u32(bd, off))
		if count <= 0 || count > maxIndex {
			off += 4
			continue
		}
		if q, sc, bi, width := quantHdr(bd, off, count); q {
			vals := unpackF(bd[off+16:], count, width, sc, bi)
			off += 16 + (count*width)/8
			assignF(&pos, &nrm, &uv, vals, count, nvertH, sc, bi)
			continue
		}
		plen := int(u32(bd, off+4))
		width := bitWidth(plen, count)
		if plen == 0 {
			off += 8
			continue
		}
		if width < 1 || width > 16 {
			off += 4
			continue
		}
		if count == nidxH && idx == nil {
			idx = unpackU(bd[off+8:], count, width)
		}
		off += 8 + (count*width)/8
	}
	if len(pos) != nvertH*3 || len(idx) != nidxH {
		return nil, fmt.Errorf("mesh pack geom")
	}
	for _, v := range idx {
		if int(v) >= nvertH {
			return nil, fmt.Errorf("mesh pack idx")
		}
	}
	if nidxH%3 != 0 {
		return nil, fmt.Errorf("mesh pack tri")
	}
	return &Mesh{Name: name, Submeshes: clipSubs(subs, nidxH, nvertH), Positions: pos, Normals: nrm, UV: uv, Indices: idx}, nil
}

func skipPad(bd []byte, off, nvert, nidx int) int {
	for off+4 <= len(bd) {
		if off%4 != 0 {
			off++
			continue
		}
		v := int(u32(bd, off))
		if v == 0 || (v > 0 && v < 16 && v != nvert && v != nidx) {
			off += 4
			continue
		}
		break
	}
	return off
}

func quantHdr(bd []byte, off, count int) (bool, float32, float32, int) {
	if off+16 > len(bd) || count <= 0 {
		return false, 0, 0, 0
	}
	if !looksF32(u32(bd, off+4)) || !looksF32(u32(bd, off+8)) {
		return false, 0, 0, 0
	}
	sc, bi := f32(bd, off+4), f32(bd, off+8)
	if math.Abs(float64(sc)) < 1e-8 && math.Abs(float64(bi)) < 1e-8 {
		return false, 0, 0, 0
	}
	plen := int(u32(bd, off+12))
	width := bitWidth(plen, count)
	if width < 1 || width > 16 || plen <= 0 {
		return false, 0, 0, 0
	}
	want := (count*width + 7) / 8
	if plen < want-1 || plen > want+1 {
		return false, 0, 0, 0
	}
	return true, sc, bi, width
}

func looksF32(u uint32) bool {
	f := math.Float32frombits(u)
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		return false
	}
	a := math.Abs(float64(f))
	return a >= 1e-8 && a < 1e6
}

func bitWidth(plen, count int) int {
	if count <= 0 || plen < 0 {
		return 0
	}
	return int(math.Round(float64(plen) * 8 / float64(count)))
}

func assignF(pos, nrm, uv *[]float32, vals []float32, count, nvert int, sc, bi float32) {
	switch {
	case count == nvert*3 && *pos == nil:
		*pos = vals
	case count == nvert*2 && isUV(sc, bi) && *uv == nil:
		*uv = vals
	case count == nvert*2 && isNrm(sc, bi) && *nrm == nil:
		*nrm = reconZ(vals, nvert)
	}
}

func isUV(sc, bi float32) bool {
	return sc > 0.4 && sc < 2.2 && bi > -0.5 && bi < 0.5
}

func isNrm(sc, bi float32) bool {
	return sc > 1.4 && sc < 2.6 && bi > -1.2 && bi < -0.7
}

func reconZ(xy []float32, nvert int) []float32 {
	out := make([]float32, nvert*3)
	for i := 0; i < nvert; i++ {
		x, y := xy[i*2], xy[i*2+1]
		z2 := 1 - x*x - y*y
		if z2 < 0 {
			z2 = 0
		}
		out[i*3] = x
		out[i*3+1] = y
		out[i*3+2] = float32(math.Sqrt(float64(z2)))
	}
	return out
}

func unpackF(d []byte, count, width int, sc, bi float32) []float32 {
	maxq := float32(int(1<<uint(width)) - 1)
	if maxq <= 0 {
		maxq = 1
	}
	out := make([]float32, count)
	for i := 0; i < count; i++ {
		out[i] = float32(bitLE(d, i*width, width))*sc/maxq + bi
	}
	return out
}

func unpackU(d []byte, count, width int) []uint32 {
	out := make([]uint32, count)
	for i := 0; i < count; i++ {
		out[i] = bitLE(d, i*width, width)
	}
	return out
}

func bitLE(d []byte, bit, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := bit + i
		if p/8 >= len(d) {
			break
		}
		if d[p/8]&(1<<uint(p%8)) != 0 {
			v |= 1 << uint(i)
		}
	}
	return v
}
