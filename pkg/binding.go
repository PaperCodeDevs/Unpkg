package pkg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const (
	bindMaxAttr = 32
	bindMaxRes  = 64
	bindMaxVar  = 512
	bindMaxStr  = 512
	bindTail    = 12
)

type DXVar struct {
	Name     string
	Offset   uint32
	Elements uint32
	Type     uint8
	Cols     uint8
	Flag     uint16
}

type DXTexture struct {
	Name    string
	Slot    uint32
	Sampler uint32
	Kind    uint8
}

type DXBuffer struct {
	Name string
	Slot uint32
}

type DXCBuffer struct {
	Name   string
	Engine []DXVar
	Vars   []DXVar
	Size   uint32
	Slot   uint32
}

type DXBinding struct {
	Hash     string
	Attrs    [][2]uint32
	AttrMask uint32
	Textures []DXTexture
	Buffers  []DXBuffer
	CBuffers []DXCBuffer
	Slots    []DXBuffer
	Tail     []byte
	Consumed int
}

type bindReader struct {
	b   []byte
	pos int
	err error
}

func (r *bindReader) fail(f string, a ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("binding: "+f, a...)
	}
}

func (r *bindReader) u32() uint32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.b) {
		r.fail("u32 at %d", r.pos)
		return 0
	}
	v := binary.LittleEndian.Uint32(r.b[r.pos:])
	r.pos += 4
	return v
}

func (r *bindReader) count(lim int) int {
	n := int(r.u32())
	if n < 0 || n > lim {
		r.fail("count %d at %d", n, r.pos)
		return 0
	}
	return n
}

func (r *bindReader) u16() uint16 {
	if r.err != nil {
		return 0
	}
	if r.pos+2 > len(r.b) {
		r.fail("u16 at %d", r.pos)
		return 0
	}
	v := binary.LittleEndian.Uint16(r.b[r.pos:])
	r.pos += 2
	return v
}

func (r *bindReader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.pos+n > len(r.b) {
		r.fail("bytes %d at %d", n, r.pos)
		return nil
	}
	v := r.b[r.pos : r.pos+n]
	r.pos += n
	return v
}

func (r *bindReader) align() {
	// 表前有 u16，字段在 blob 空间按 4 对齐，落到表内即 pos%4==2
	for r.err == nil && r.pos%4 != 2 {
		r.pos++
	}
	if r.pos > len(r.b) {
		r.fail("align past end")
	}
}

func (r *bindReader) str() string {
	n := r.count(bindMaxStr)
	s := string(r.bytes(n))
	r.align()
	return s
}

func (r *bindReader) ok() bool {
	return r.err == nil
}

func (r *bindReader) vars(lim int, tail func(*DXVar)) []DXVar {
	n := r.count(lim)
	out := make([]DXVar, 0, n)
	for i := 0; i < n && r.ok(); i++ {
		v := DXVar{Name: r.str(), Offset: r.u32(), Elements: r.u32()}
		tail(&v)
		out = append(out, v)
	}
	return out
}

func (r *bindReader) dxVarTail(v *DXVar) {
	t := r.bytes(4)
	if t == nil {
		return
	}
	v.Type, v.Cols = t[0], t[1]
	v.Flag = binary.LittleEndian.Uint16(t[2:4])
}

func ParseDXBinding(ex []byte) (DXBinding, error) {
	var bd DXBinding
	r := &bindReader{b: ex}
	r.u16()
	if len(ex) >= 6 && binary.LittleEndian.Uint32(ex[2:6]) == dxHashLen {
		r.u32()
		bd.Hash = hex.EncodeToString(r.bytes(dxHashLen))
	} else {
		bd.Hash = hex.EncodeToString(r.bytes(dxHashLen))
	}
	n := r.count(bindMaxAttr)
	for i := 0; i < n && r.ok(); i++ {
		bd.Attrs = append(bd.Attrs, [2]uint32{r.u32(), r.u32()})
	}
	bd.AttrMask = r.u32()
	r.u32()
	r.u32()
	r.u32()
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		t := DXTexture{Name: r.str(), Slot: r.u32(), Sampler: r.u32()}
		if k := r.bytes(4); k != nil {
			t.Kind = k[1]
		}
		bd.Textures = append(bd.Textures, t)
	}
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		bd.Buffers = append(bd.Buffers, DXBuffer{Name: r.str(), Slot: r.u32()})
	}
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		cb := DXCBuffer{Name: r.str()}
		cb.Engine = r.vars(bindMaxVar, r.dxVarTail)
		cb.Vars = r.vars(bindMaxVar, r.dxVarTail)
		r.u32()
		cb.Size = r.u32()
		bd.CBuffers = append(bd.CBuffers, cb)
	}
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		bd.Slots = append(bd.Slots, DXBuffer{Name: r.str(), Slot: r.u32()})
	}
	for i := range bd.Slots {
		if i < len(bd.CBuffers) && bd.CBuffers[i].Name == bd.Slots[i].Name {
			bd.CBuffers[i].Slot = bd.Slots[i].Slot
		}
	}
	bd.Tail = r.bytes(bindTail)
	if r.err != nil {
		return bd, r.err
	}
	if r.pos != len(ex) {
		return bd, fmt.Errorf("binding: trailing %d", len(ex)-r.pos)
	}
	bd.Consumed = r.pos
	return bd, nil
}

func (bd DXBinding) Names() []string {
	var out []string
	for _, t := range bd.Textures {
		out = append(out, t.Name)
	}
	for _, b := range bd.Buffers {
		out = append(out, b.Name)
	}
	for _, c := range bd.CBuffers {
		out = append(out, c.Name)
	}
	return uniqDX(out)
}

func TextureKindName(k uint8) string {
	switch k {
	case 2:
		return "2d"
	case 3:
		return "3d"
	case 4:
		return "cube"
	case 5:
		return "2darray"
	}
	return ""
}
