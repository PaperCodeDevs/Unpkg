package rainbow

import (
	"encoding/binary"
	"fmt"
)

const (
	matVer      = 1
	matMinSize  = 52
	matTableOff = 48
	matDupOff   = 24
	matHashOff  = 40
	matTexBody  = 44
	matKeyLead  = 48
	matMaxStr   = 256
	matMaxTex   = 40
	matMaxFloat = 200
	matMaxColor = 80
	matMaxKey   = 40
	matScanSpan = 720
)

type Mat struct {
	ID       uint32
	Type     uint32
	Name     string
	Hash     [8]byte
	Shader   [16]byte
	Texs     []MatTex
	Floats   []MatFloat
	Colors   []MatColor
	Enabled  []string
	Keywords []MatKey
}

type MatTex struct {
	Name   string
	Scale  [2]float32
	Offset [2]float32
	Tag    uint32
	Ref    [16]byte
}

type MatFloat struct {
	Name  string
	Value float32
}

type MatColor struct {
	Name string
	RGBA [4]float32
}

type MatKey struct {
	Name string
	On   bool
}

func (m *Mat) ResID() uint64 {
	if m == nil {
		return 0
	}
	return uint64(m.ID) | uint64(m.Type)<<32
}

func TexResID(ref [16]byte) uint64 {
	return binary.LittleEndian.Uint64(ref[:8])
}

func (t MatTex) ResID() uint64 {
	return TexResID(t.Ref)
}

func (t MatTex) Empty() bool {
	return TexResID(t.Ref) == 0
}

func ParseMat(raw []byte) (*Mat, error) {
	id, typ, hash, start, err := matHead(raw)
	if err != nil {
		return nil, err
	}
	pend := 8 + int(binary.LittleEndian.Uint32(raw[4:]))
	texs, floats, colors, texOff, end, ok := matFindMaps(raw, start, pend)
	if !ok {
		return nil, fmt.Errorf("mat props")
	}
	m := &Mat{
		ID:     id,
		Type:   typ,
		Name:   matName(raw, start, texOff),
		Texs:   texs,
		Floats: floats,
		Colors: colors,
	}
	copy(m.Hash[:], hash)
	matKeys(raw, end, pend, m)
	return m, nil
}
