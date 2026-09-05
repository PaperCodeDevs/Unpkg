package pkg

import (
	"encoding/binary"
	"fmt"
)

const (
	meshDescBase = 0x24
	meshDesc     = 48
	meshMaxIdx   = 200000
	meshMaxVert  = 50000
)

type meshStream struct {
	nidx  int
	base  int
	nvert int
}

func readMeshStreams(b []byte) ([]meshStream, error) {
	n := int(binary.LittleEndian.Uint32(b[0x1C:0x20]))
	if n < 1 || meshDescBase+n*meshDesc+4 > len(b) {
		return nil, fmt.Errorf("blockmesh streams %d", n)
	}
	out := make([]meshStream, n)
	total := 0
	for i := range out {
		d := meshDescBase + i*meshDesc
		s := meshStream{
			nidx:  int(binary.LittleEndian.Uint32(b[d:])),
			base:  int(binary.LittleEndian.Uint32(b[d+12:])),
			nvert: int(binary.LittleEndian.Uint32(b[d+16:])),
		}
		if s.nidx <= 0 || s.nidx%3 != 0 || s.nvert <= 0 || s.base+s.nvert > meshMaxVert {
			return nil, fmt.Errorf("blockmesh stream %d nidx=%d base=%d nvert=%d", i, s.nidx, s.base, s.nvert)
		}
		total += s.nidx
		if total > meshMaxIdx || int(binary.LittleEndian.Uint32(b[d+44:])) != total*2 {
			return nil, fmt.Errorf("blockmesh stream %d idx bytes", i)
		}
		out[i] = s
	}
	return out, nil
}

func meshStreamTotals(ss []meshStream) (nidx, nvert int) {
	for _, s := range ss {
		nidx += s.nidx
		if e := s.base + s.nvert; e > nvert {
			nvert = e
		}
	}
	return nidx, nvert
}

func readMeshIndex(b []byte, ss []meshStream, nidx int) ([]uint32, error) {
	idx := make([]uint32, 0, nidx)
	for i, s := range ss {
		end := s.base + s.nvert
		for k := 0; k < s.nidx; k++ {
			v := int(binary.LittleEndian.Uint16(b[len(idx)*2:]))
			if v < s.base || v >= end {
				return nil, fmt.Errorf("blockmesh stream %d idx %d", i, v)
			}
			idx = append(idx, uint32(v))
		}
	}
	return idx, nil
}
