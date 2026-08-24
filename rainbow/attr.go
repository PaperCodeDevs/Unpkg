package rainbow

type attr struct {
	n, fmt, off, use int
}

func parseAttrs(bd []byte, decl, nslot int) []attr {
	out := make([]attr, 0, 8)
	for i := 0; i < nslot; i++ {
		o := decl + 8 + i*4
		if o+4 > len(bd) {
			break
		}
		r := u32(bd, o)
		if r == 0 {
			continue
		}
		out = append(out, attr{
			n:   int(r >> 24),
			fmt: int((r >> 16) & 0xFF),
			off: int((r >> 8) & 0xFF),
			use: int(r & 0xFF),
		})
	}
	return out
}

func attrSize(a attr) int {
	switch a.fmt {
	case 0:
		return a.n * 4
	case 2:
		return a.n
	case 11:
		return 4
	default:
		if a.n <= 0 {
			return 0
		}
		return a.n * 4
	}
}

func readVB(bd []byte, vb, nvert int, attrs []attr) (pos, nrm, uv []float32) {
	main := 0
	var extra []attr
	uvOff, nrmOff := -1, -1
	for _, a := range attrs {
		if a.use != 0 {
			extra = append(extra, a)
			continue
		}
		end := a.off + attrSize(a)
		if end > main {
			main = end
		}
		if a.n == 3 && a.off == 0 && a.fmt == 0 {
			continue
		}
		if a.n == 3 && a.off == 12 && a.fmt == 0 {
			nrmOff = 12
		}
		if a.n == 2 && a.fmt == 0 && uvOff < 0 {
			uvOff = a.off
		}
	}
	if main < 12 {
		main = 12
	}
	need := nvert * main
	for _, a := range extra {
		need += nvert * attrSize(a)
	}
	if vb < 0 || nvert <= 0 || vb+need > len(bd) {
		return nil, nil, nil
	}
	pos = make([]float32, nvert*3)
	for i := 0; i < nvert; i++ {
		o := vb + i*main
		pos[i*3] = f32(bd, o)
		pos[i*3+1] = f32(bd, o+4)
		pos[i*3+2] = f32(bd, o+8)
	}
	if nrmOff >= 0 {
		nrm = make([]float32, nvert*3)
		for i := 0; i < nvert; i++ {
			o := vb + i*main + nrmOff
			nrm[i*3] = f32(bd, o)
			nrm[i*3+1] = f32(bd, o+4)
			nrm[i*3+2] = f32(bd, o+8)
		}
	}
	if uvOff >= 0 {
		uv = make([]float32, nvert*2)
		for i := 0; i < nvert; i++ {
			o := vb + i*main + uvOff
			uv[i*2] = f32(bd, o)
			uv[i*2+1] = f32(bd, o+4)
		}
		return pos, nrm, uv
	}
	cur := vb + nvert*main
	for _, a := range extra {
		sz := attrSize(a)
		if a.n == 2 && a.fmt == 0 && uv == nil {
			uv = make([]float32, nvert*2)
			for i := 0; i < nvert; i++ {
				o := cur + i*sz
				uv[i*2] = f32(bd, o)
				uv[i*2+1] = f32(bd, o+4)
			}
		}
		cur += nvert * sz
	}
	return pos, nrm, uv
}
