package crn

import (
	"fmt"
	"image"
)

type bc7bits struct {
	b []byte
	n int
}

func (r *bc7bits) get(k int) uint32 {
	var v uint32
	for i := 0; i < k; i++ {
		if r.n/8 < len(r.b) && r.b[r.n/8]&(1<<uint(r.n%8)) != 0 {
			v |= 1 << uint(i)
		}
		r.n++
	}
	return v
}

func bc7expand(v, bits int) int {
	if bits <= 0 {
		return 0
	}
	v &= (1 << bits) - 1
	if bits >= 8 {
		return v
	}
	out := 0
	for shift := 8; shift > 0; {
		shift -= bits
		if shift >= 0 {
			out |= v << shift
		} else {
			out |= v >> -shift
		}
	}
	return out
}

func bc7lerp(a, b, iw int) uint8 {
	return uint8((a*(64-iw) + b*iw + 32) >> 6)
}

func bc7subset(ns, part, px int) int {
	if ns == 2 {
		return int((bc7part2[part] >> uint(px)) & 1)
	}
	if ns == 3 {
		s := int(bc7part3[part][px])
		if s > 2 {
			return 2
		}
		return s
	}
	return 0
}

func bc7anchor(ns, part, subset int) int {
	if subset == 0 {
		return 0
	}
	if ns == 2 {
		return int(bc7anchor2[part])
	}
	if subset == 1 {
		return int(bc7anchor3a[part])
	}
	return int(bc7anchor3b[part])
}

func decodeBC7Block(src []byte, dst []byte, pitch int) {
	var r bc7bits
	r.b = src
	mode := 0
	for mode < 8 && r.get(1) == 0 {
		mode++
	}
	if mode >= 8 {
		return
	}
	m := bc7modes[mode]
	ns := int(m.ns)
	part := 0
	if m.pb > 0 {
		part = int(r.get(int(m.pb)))
	}
	rot := 0
	if m.rb > 0 {
		rot = int(r.get(int(m.rb)))
	}
	sel := 0
	if m.isb > 0 {
		sel = int(r.get(int(m.isb)))
	}
	nend := ns * 2
	var ep [3][6]int
	cb := int(m.cb)
	for c := 0; c < 3; c++ {
		for i := 0; i < nend; i++ {
			ep[c][i] = int(r.get(cb))
		}
	}
	var ea [6]int
	ab := int(m.ab)
	hasA := ab > 0
	if hasA {
		for i := 0; i < nend; i++ {
			ea[i] = int(r.get(ab))
		}
	}
	var pbit [6]int
	if m.epb != 0 {
		for i := 0; i < nend; i++ {
			pbit[i] = int(r.get(1))
		}
	} else if m.spb != 0 {
		for s := 0; s < ns; s++ {
			v := int(r.get(1))
			pbit[s*2] = v
			pbit[s*2+1] = v
		}
	}
	cbits, abits := cb, ab
	if m.epb != 0 || m.spb != 0 {
		cbits++
		if hasA {
			abits++
		}
		for i := 0; i < nend; i++ {
			for c := 0; c < 3; c++ {
				ep[c][i] = (ep[c][i] << 1) | pbit[i]
			}
			if hasA {
				ea[i] = (ea[i] << 1) | pbit[i]
			}
		}
	}
	for i := 0; i < nend; i++ {
		for c := 0; c < 3; c++ {
			ep[c][i] = bc7expand(ep[c][i], cbits)
		}
		if hasA {
			ea[i] = bc7expand(ea[i], abits)
		} else {
			ea[i] = 255
		}
	}
	ib, iba := int(m.ib), int(m.iba)
	var idx, idx2 [16]int
	for i := 0; i < 16; i++ {
		bits := ib
		s := bc7subset(ns, part, i)
		if i == bc7anchor(ns, part, s) {
			bits--
		}
		idx[i] = int(r.get(bits))
	}
	if iba > 0 {
		for i := 0; i < 16; i++ {
			bits := iba
			if i == 0 {
				bits--
			}
			idx2[i] = int(r.get(bits))
		}
	}
	var wt, wtA []int
	switch ib {
	case 2:
		wt = bc7w2[:]
	case 3:
		wt = bc7w3[:]
	default:
		wt = bc7w4[:]
	}
	wtA = wt
	if iba == 2 {
		wtA = bc7w2[:]
	} else if iba == 3 {
		wtA = bc7w3[:]
	}
	for i := 0; i < 16; i++ {
		s := bc7subset(ns, part, i)
		e0, e1 := s*2, s*2+1
		ci, ai := idx[i], idx[i]
		cw, aw := wt, wtA
		if iba > 0 {
			if sel != 0 {
				ci, ai = idx2[i], idx[i]
				cw, aw = wtA, wt
			} else {
				ai = idx2[i]
				aw = wtA
			}
		}
		if e1 >= 6 {
			e0, e1 = 0, 1
		}
		if ci >= len(cw) {
			ci = len(cw) - 1
		}
		if ai >= len(aw) {
			ai = len(aw) - 1
		}
		r8 := bc7lerp(ep[0][e0], ep[0][e1], cw[ci])
		g8 := bc7lerp(ep[1][e0], ep[1][e1], cw[ci])
		b8 := bc7lerp(ep[2][e0], ep[2][e1], cw[ci])
		a8 := bc7lerp(ea[e0], ea[e1], aw[ai])
		switch rot {
		case 1:
			r8, a8 = a8, r8
		case 2:
			g8, a8 = a8, g8
		case 3:
			b8, a8 = a8, b8
		}
		x, y := i&3, i>>2
		o := y*pitch + x*4
		dst[o+0], dst[o+1], dst[o+2], dst[o+3] = r8, g8, b8, a8
	}
}

func DecodeBC7(blocks []byte, w, h int) (*image.NRGBA, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("bc7 size")
	}
	bw, bh := (w+3)/4, (h+3)/4
	need := bw * bh * 16
	if len(blocks) < need {
		return nil, fmt.Errorf("bc7 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	tmp := make([]byte, 16*4)
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			off := (by*bw + bx) * 16
			decodeBC7Block(blocks[off:off+16], tmp, 16)
			for y := 0; y < 4; y++ {
				iy := by*4 + y
				if iy >= h {
					break
				}
				for x := 0; x < 4; x++ {
					ix := bx*4 + x
					if ix >= w {
						break
					}
					s := y*16 + x*4
					d := img.PixOffset(ix, iy)
					copy(img.Pix[d:d+4], tmp[s:s+4])
				}
			}
		}
	}
	return img, nil
}
