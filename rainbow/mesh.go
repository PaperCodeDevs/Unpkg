package rainbow

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	meshVer   = 2
	meshMagic = 0x59A21C2C
	meshType0 = 0x4A740004
	ibRaw     = 256
	ibPack    = 259
	declSlots = 14
	maxSub    = 64
	maxVert   = 2_000_000
	maxIndex  = 4_000_000
	ibBase    = 84
	subExtra  = 48
	skinExtra = 96
)

type Mesh struct {
	Name      string
	Submeshes []Submesh
	Positions []float32
	Normals   []float32
	UV        []float32
	Indices   []uint32
}

type Submesh struct {
	IndexStart  int
	IndexCount  int
	VertexStart int
	VertexCount int
}

func ParseMesh(raw []byte) (*Mesh, error) {
	name, body, err := parseHead(raw)
	if err != nil {
		return nil, err
	}
	bd := raw[body:]
	if len(bd) < ibBase {
		return nil, fmt.Errorf("mesh body")
	}
	nsub := int(u32(bd, 0))
	if nsub < 1 || nsub > maxSub {
		return nil, fmt.Errorf("mesh nsub %d", nsub)
	}
	subs := readSubs(bd, nsub)
	ib := findIB(bd, nsub)
	if ib < 0 {
		return nil, fmt.Errorf("mesh ib")
	}
	switch u32(bd, ib) {
	case ibRaw:
		return parseRaw(name, bd, ib, subs)
	case ibPack:
		return parsePacked(name, bd, ib, subs)
	default:
		return nil, fmt.Errorf("mesh ibfmt %d", u32(bd, ib))
	}
}

func u32(b []byte, o int) uint32 {
	return binary.LittleEndian.Uint32(b[o:])
}

func u16(b []byte, o int) uint16 {
	return binary.LittleEndian.Uint16(b[o:])
}

func f32(b []byte, o int) float32 {
	return math.Float32frombits(u32(b, o))
}

func align4(n int) int {
	return (n + 3) &^ 3
}
