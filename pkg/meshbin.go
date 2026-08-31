package pkg

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	meshUnit    = 100
	meshHead    = 0x54
	meshDecl    = 64
	meshSlots   = 14
	meshTypePos = 3
	meshTypeUV  = 2
)

type BlockMesh struct {
	Pos []float32
	UV  []float32
	Idx []uint32
}

func ParseBlockMesh(b []byte) (*BlockMesh, error) {
	if len(b) < meshHead+2 {
		return nil, fmt.Errorf("blockmesh short")
	}
	ver := math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	if ver < 0.5 || ver > 2 {
		return nil, fmt.Errorf("blockmesh ver %v", ver)
	}
	nidx := int(binary.LittleEndian.Uint32(b[0x24:0x28]))
	nvert := int(binary.LittleEndian.Uint32(b[0x34:0x38]))
	idxBytes := int(binary.LittleEndian.Uint32(b[0x50:0x54]))
	streams := int(binary.LittleEndian.Uint32(b[0x1C:0x20]))
	if streams != 1 {
		return nil, fmt.Errorf("blockmesh streams %d", streams)
	}
	if nidx <= 0 || nvert <= 0 || nidx > 200000 || nvert > 50000 {
		return nil, fmt.Errorf("blockmesh nidx=%d nvert=%d", nidx, nvert)
	}
	if idxBytes != nidx*2 {
		return nil, fmt.Errorf("blockmesh idx bytes %d", idxBytes)
	}
	idxEnd := meshHead + idxBytes
	if idxEnd%4 != 0 {
		idxEnd = (idxEnd + 3) &^ 3
	}
	declEnd := idxEnd + meshDecl
	if declEnd+4 > len(b) {
		return nil, fmt.Errorf("blockmesh decl")
	}
	dn := int(binary.LittleEndian.Uint32(b[idxEnd : idxEnd+4]))
	nslot := int(binary.LittleEndian.Uint32(b[idxEnd+4 : idxEnd+8]))
	if dn != nvert || nslot != meshSlots {
		return nil, fmt.Errorf("blockmesh decl %d/%d", dn, nslot)
	}
	posOff, uvOff := -1, -1
	for i := 0; i < meshSlots; i++ {
		e := binary.LittleEndian.Uint32(b[idxEnd+8+i*4 : idxEnd+12+i*4])
		if e == 0 {
			continue
		}
		typ := int(e >> 24)
		off := int((e >> 8) & 0xFFFF)
		if typ == meshTypePos && posOff < 0 {
			posOff = off
		}
		if typ == meshTypeUV && uvOff < 0 {
			uvOff = off
		}
	}
	if posOff < 0 {
		return nil, fmt.Errorf("blockmesh no pos")
	}
	vbBytes := int(binary.LittleEndian.Uint32(b[declEnd : declEnd+4]))
	vb := declEnd + 4
	if vb+vbBytes != len(b) || vbBytes%nvert != 0 {
		return nil, fmt.Errorf("blockmesh vb %d", vbBytes)
	}
	stride := vbBytes / nvert
	if posOff+12 > stride || (uvOff >= 0 && uvOff+8 > stride) {
		return nil, fmt.Errorf("blockmesh stride %d", stride)
	}
	pos := make([]float32, nvert*3)
	uv := make([]float32, nvert*2)
	for i := 0; i < nvert; i++ {
		o := vb + i*stride
		pos[i*3] = f32le(b, o+posOff) / meshUnit
		pos[i*3+1] = f32le(b, o+posOff+4) / meshUnit
		pos[i*3+2] = f32le(b, o+posOff+8) / meshUnit
		if uvOff >= 0 {
			uv[i*2] = f32le(b, o+uvOff)
			uv[i*2+1] = f32le(b, o+uvOff+4)
		}
	}
	idx := make([]uint32, nidx)
	maxI := uint32(0)
	for i := 0; i < nidx; i++ {
		v := uint32(binary.LittleEndian.Uint16(b[meshHead+i*2:]))
		idx[i] = v
		if v > maxI {
			maxI = v
		}
	}
	if int(maxI) >= nvert {
		return nil, fmt.Errorf("blockmesh idx")
	}
	if nidx%3 != 0 {
		return nil, fmt.Errorf("blockmesh not tri")
	}
	return &BlockMesh{Pos: pos, UV: uv, Idx: idx}, nil
}

func f32le(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}
