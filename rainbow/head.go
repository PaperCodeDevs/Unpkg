package rainbow

import "fmt"

func parseHead(b []byte) (string, int, error) {
	if len(b) < 24 {
		return "", 0, fmt.Errorf("mesh short")
	}
	if u32(b, 0) != meshVer || u32(b, 4) != meshMagic {
		return "", 0, fmt.Errorf("mesh magic")
	}
	if u32(b, 8) != meshType0 {
		return "", 0, fmt.Errorf("mesh type")
	}
	n := int(u32(b, 16))
	if n < 0 || 20+n > len(b) {
		return "", 0, fmt.Errorf("mesh name")
	}
	return string(b[20 : 20+n]), align4(20 + n), nil
}

func readSubs(bd []byte, nsub int) []Submesh {
	out := make([]Submesh, 0, nsub)
	if len(bd) < 28 {
		return out
	}
	out = append(out, Submesh{
		IndexStart:  0,
		IndexCount:  int(u32(bd, 8)),
		VertexStart: 0,
		VertexCount: int(u32(bd, 24)),
	})
	for i := 1; i < nsub; i++ {
		o := 52 + (i-1)*subExtra
		if o+24 > len(bd) {
			break
		}
		out = append(out, Submesh{
			IndexStart:  int(u32(bd, o)) / 2,
			IndexCount:  int(u32(bd, o+4)),
			VertexStart: int(u32(bd, o+16)),
			VertexCount: int(u32(bd, o+20)),
		})
	}
	return out
}

func findIB(bd []byte, nsub int) int {
	base := ibBase + subExtra*(nsub-1)
	best := -1
	lim := base + skinExtra + 32
	if lim > len(bd)-12 {
		lim = len(bd) - 12
	}
	for off := base; off <= lim; off += 4 {
		if off < 0 {
			continue
		}
		f := u32(bd, off)
		if f != ibRaw && f != ibPack {
			continue
		}
		if !looksIB(bd, off, f) {
			continue
		}
		return off
	}
	return best
}

func looksIB(bd []byte, off int, fmtv uint32) bool {
	if off+12 > len(bd) {
		return false
	}
	ibytes := int(u32(bd, off+8))
	if ibytes < 0 || off+12+ibytes > len(bd) {
		return false
	}
	decl := align4(off + 12 + ibytes)
	if decl+8 > len(bd) {
		return false
	}
	if int(u32(bd, decl+4)) != declSlots {
		return false
	}
	if fmtv == ibRaw {
		return ibytes >= 6 && ibytes%2 == 0
	}
	return ibytes == 0
}
