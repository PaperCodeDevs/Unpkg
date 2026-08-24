package rainbow

import "fmt"

func parseRaw(name string, bd []byte, ib int, subs []Submesh) (*Mesh, error) {
	ibytes := int(u32(bd, ib+8))
	ibdata := ib + 12
	if ibytes < 6 || ibytes%2 != 0 || ibdata+ibytes > len(bd) {
		return nil, fmt.Errorf("mesh ibytes")
	}
	nidx := ibytes / 2
	if nidx > maxIndex {
		return nil, fmt.Errorf("mesh nidx")
	}
	decl := align4(ibdata + ibytes)
	if decl+12 > len(bd) {
		return nil, fmt.Errorf("mesh decl")
	}
	nvert := int(u32(bd, decl))
	nslot := int(u32(bd, decl+4))
	if nvert <= 0 || nvert > maxVert || nslot <= 0 || nslot > 32 {
		return nil, fmt.Errorf("mesh decl %d/%d", nvert, nslot)
	}
	if decl+8+nslot*4+4 > len(bd) {
		return nil, fmt.Errorf("mesh attr")
	}
	attrs := parseAttrs(bd, decl, nslot)
	vb := decl + 8 + nslot*4 + 4
	pos, nrm, uv := readVB(bd, vb, nvert, attrs)
	if len(pos) != nvert*3 {
		return nil, fmt.Errorf("mesh vb")
	}
	idx := make([]uint32, nidx)
	for i := 0; i < nidx; i++ {
		v := uint32(u16(bd, ibdata+i*2))
		if int(v) >= nvert {
			return nil, fmt.Errorf("mesh idx")
		}
		idx[i] = v
	}
	if nidx%3 != 0 {
		return nil, fmt.Errorf("mesh not tri")
	}
	return &Mesh{Name: name, Submeshes: clipSubs(subs, nidx, nvert), Positions: pos, Normals: nrm, UV: uv, Indices: idx}, nil
}

func clipSubs(subs []Submesh, nidx, nvert int) []Submesh {
	out := make([]Submesh, 0, len(subs))
	for _, s := range subs {
		if s.IndexCount <= 0 || s.VertexCount <= 0 {
			continue
		}
		if s.IndexStart < 0 || s.VertexStart < 0 {
			continue
		}
		if s.IndexStart+s.IndexCount > nidx {
			continue
		}
		if s.VertexStart+s.VertexCount > nvert {
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return []Submesh{{IndexCount: nidx, VertexCount: nvert}}
	}
	return out
}
