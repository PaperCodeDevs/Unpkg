package rainbow

import (
	"encoding/binary"
	"fmt"
)

type Ref struct {
	ID   uint32
	Type uint32
}

const (
	Ver1       = 1
	TypeAnim   = 0x0798BD94
	TypeNode   = 0x0798BD95
	TypeVFX    = 0x0798BD96
	TypeMat2   = 0x0798BD98
	TypeExtra  = 0x0798BD9A
	KindNode   = "node"
	KindAnim   = "anim"
	KindVFX    = "vfx"
	KindExtra  = "extra"
	meshTypeLo = 0x079481BD
	meshTypeHi = 0x079481C8
	meshTypeA  = 0x079482F8
	meshTypeB  = 0x079482F9
	meshTypeC  = 0x0775D1DD
	meshTypeD  = 0x0797855B
	typeMin    = 0x07400000
	typeMax    = 0x07A00000
)

type Header struct {
	Ver     uint32
	Payload uint32
	ID      uint32
	Type    uint32
}

func (r Ref) ResID() uint64 {
	return uint64(r.ID) | uint64(r.Type)<<32
}

func RefFromResID(id uint64) Ref {
	return Ref{ID: uint32(id), Type: uint32(id >> 32)}
}

func IDs(rr []Ref) []uint32 {
	if len(rr) == 0 {
		return nil
	}
	out := make([]uint32, len(rr))
	for i, r := range rr {
		out[i] = r.ID
	}
	return out
}

func ParseHeader(raw []byte) (Header, error) {
	var h Header
	if len(raw) < 16 {
		return h, fmt.Errorf("prefab short")
	}
	h.Ver = binary.LittleEndian.Uint32(raw[0:])
	h.Payload = binary.LittleEndian.Uint32(raw[4:])
	h.ID = binary.LittleEndian.Uint32(raw[8:])
	h.Type = binary.LittleEndian.Uint32(raw[12:])
	if h.Ver != Ver1 {
		return h, fmt.Errorf("prefab ver")
	}
	if int(h.Payload) > len(raw)-8 {
		return h, fmt.Errorf("prefab payload")
	}
	return h, nil
}

func ParseRef(b []byte) (Ref, bool) {
	if len(b) < 16 {
		return Ref{}, false
	}
	id := binary.LittleEndian.Uint32(b[0:])
	typ := binary.LittleEndian.Uint32(b[4:])
	z := binary.LittleEndian.Uint64(b[8:])
	if z != 0 || id == 0 || typ < typeMin || typ > typeMax {
		return Ref{}, false
	}
	return Ref{ID: id, Type: typ}, true
}

func ScanRefs(b []byte) []Ref {
	var out []Ref
	seen := map[uint64]struct{}{}
	for i := 0; i+16 <= len(b); i += 4 {
		r, ok := ParseRef(b[i:])
		if !ok {
			continue
		}
		k := r.ResID()
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	return out
}

func IsMeshType(t uint32) bool {
	if t >= meshTypeLo && t <= meshTypeHi {
		return true
	}
	return t == meshTypeA || t == meshTypeB || t == meshTypeC || t == meshTypeD
}

func IsMatType(t uint32) bool {
	switch t {
	case TypeAnim, TypeNode, TypeMat2, TypeExtra:
		return true
	}
	return false
}

func IsPrefabType(t uint32) bool {
	switch t {
	case TypeAnim, TypeNode, TypeVFX, TypeExtra:
		return true
	}
	return false
}

func PrefabKind(t uint32) string {
	switch t {
	case TypeNode:
		return KindNode
	case TypeAnim:
		return KindAnim
	case TypeVFX:
		return KindVFX
	case TypeExtra:
		return KindExtra
	default:
		return ""
	}
}
