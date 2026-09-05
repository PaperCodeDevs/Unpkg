package pkg

import "fmt"

const (
	matBindUnit  = 21
	matBindUnits = 6
)

func (r *bindReader) matVarTail(v *DXVar) {
	// material_res 旧版：多一个 u32，再 u8 class(1 向量 4 矩阵) u8 列数 u8 行标
	r.u32()
	t := r.bytes(3)
	if t == nil {
		return
	}
	v.Type, v.Cols, v.Flag = t[0], t[1], uint16(t[2])
}

func ParseMaterialBinding(ex []byte) (DXBinding, error) {
	var bd DXBinding
	r := &bindReader{b: ex}
	r.u16()
	n := r.count(bindMaxAttr)
	for i := 0; i < n && r.ok(); i++ {
		bd.Attrs = append(bd.Attrs, [2]uint32{r.u32(), r.u32()})
	}
	bd.AttrMask = r.u32()
	r.u32()
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		t := DXTexture{Name: r.str(), Slot: r.u32(), Sampler: r.u32()}
		r.bytes(1)
		t.Kind = uint8(r.u32())
		bd.Textures = append(bd.Textures, t)
	}
	r.align()
	r.u32()
	r.u32()
	r.u32()
	n = r.count(bindMaxRes)
	for i := 0; i < n && r.ok(); i++ {
		cb := DXCBuffer{Name: r.str()}
		cb.Vars = r.vars(bindMaxVar, r.matVarTail)
		r.align()
		r.u32()
		cb.Size = r.u32()
		cb.Slot = r.u32()
		r.u32()
		bd.CBuffers = append(bd.CBuffers, cb)
	}
	r.u32()
	r.u32()
	n = r.count(matBindUnits)
	bd.Tail = r.bytes(n * matBindUnit)
	if r.err != nil {
		return bd, r.err
	}
	if r.pos != len(ex) {
		return bd, fmt.Errorf("material binding: trailing %d", len(ex)-r.pos)
	}
	bd.Consumed = r.pos
	return bd, nil
}
