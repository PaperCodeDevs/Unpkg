package rainbow

import (
	"encoding/binary"
	"fmt"
)

type Prefab struct {
	Name string
	ID   uint32
	Type uint32
	Mesh []Ref
	Mat  []Ref
}

func (p *Prefab) Kind() string {
	if p == nil {
		return ""
	}
	return PrefabKind(p.Type)
}

func (p *Prefab) MeshIDs() []uint32 {
	if p == nil {
		return nil
	}
	return IDs(p.Mesh)
}

func (p *Prefab) MatIDs() []uint32 {
	if p == nil {
		return nil
	}
	return IDs(p.Mat)
}

func (p *Prefab) Self() Ref {
	if p == nil {
		return Ref{}
	}
	return Ref{ID: p.ID, Type: p.Type}
}

func ParsePrefab(raw []byte) (*Prefab, error) {
	h, err := ParseHeader(raw)
	if err != nil {
		return nil, err
	}
	if !IsPrefabType(h.Type) {
		return nil, fmt.Errorf("prefab type")
	}
	p := &Prefab{ID: h.ID, Type: h.Type, Name: prefabName(raw)}
	self := p.Self().ResID()
	for _, r := range ScanRefs(raw) {
		if r.ResID() == self {
			continue
		}
		switch {
		case IsMeshType(r.Type):
			p.Mesh = append(p.Mesh, r)
		case IsMatType(r.Type):
			p.Mat = append(p.Mat, r)
		}
	}
	return p, nil
}

func prefabName(b []byte) string {
	if len(b) < 56 {
		return ""
	}
	nfield := int(binary.LittleEndian.Uint32(b[48:]))
	if nfield < 1 || nfield > 128 {
		return ""
	}
	off := 52 + nfield*6
	if off%4 != 0 {
		off += 4 - off%4
	}
	if off+4 > len(b) {
		return ""
	}
	ncomp := int(binary.LittleEndian.Uint32(b[off:]))
	if ncomp < 0 || ncomp > 512 {
		return ""
	}
	off += 4 + ncomp*28
	return u32name(b, off)
}

func u32name(b []byte, off int) string {
	if off < 0 || off+4 > len(b) {
		return ""
	}
	n := int(binary.LittleEndian.Uint32(b[off:]))
	if n < 1 || n > 96 || off+4+n > len(b) {
		return ""
	}
	s := b[off+4 : off+4+n]
	if n > 0 && s[n-1] == 0 {
		s = s[:n-1]
	}
	if len(s) == 0 {
		return ""
	}
	for _, c := range s {
		if c < 0x20 || c >= 0x7f {
			return ""
		}
	}
	return string(s)
}
