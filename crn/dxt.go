package crn

import (
	"encoding/binary"
	"image"
)

func rgb565(v uint16) (r, g, b uint8) {
	rr := uint32(v>>11) & 31
	gg := uint32(v>>5) & 63
	bb := uint32(v) & 31
	return uint8(rr<<3 | rr>>2), uint8(gg<<2 | gg>>4), uint8(bb<<3 | bb>>2)
}

func colorTable(c0, c1 uint16, opaque bool) (out [4][3]uint8) {
	r0, g0, b0 := rgb565(c0)
	r1, g1, b1 := rgb565(c1)
	out[0] = [3]uint8{r0, g0, b0}
	out[1] = [3]uint8{r1, g1, b1}
	if opaque || c0 > c1 {
		for i := 0; i < 3; i++ {
			out[2][i] = uint8((2*uint32(out[0][i]) + uint32(out[1][i])) / 3)
			out[3][i] = uint8((uint32(out[0][i]) + 2*uint32(out[1][i])) / 3)
		}
	} else {
		for i := 0; i < 3; i++ {
			out[2][i] = uint8((uint32(out[0][i]) + uint32(out[1][i])) / 2)
			out[3][i] = 0
		}
	}
	return
}

func alphaTable(a0, a1 uint8) (t [8]uint8) {
	t[0], t[1] = a0, a1
	if a0 > a1 {
		for i := 2; i < 8; i++ {
			t[i] = uint8((uint32(8-i)*uint32(a0) + uint32(i-1)*uint32(a1)) / 7)
		}
	} else {
		for i := 2; i < 6; i++ {
			t[i] = uint8((uint32(6-i)*uint32(a0) + uint32(i-1)*uint32(a1)) / 5)
		}
		t[6], t[7] = 0, 255
	}
	return
}

func blitBlock(img *image.NRGBA, bx, by int, px *[16][4]uint8) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
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
			p := px[x+y*4]
			o := img.PixOffset(ix, iy)
			img.Pix[o] = p[0]
			img.Pix[o+1] = p[1]
			img.Pix[o+2] = p[2]
			img.Pix[o+3] = p[3]
		}
	}
}

func decodeDXT5(blocks []byte, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bw, bh := (w+3)/4, (h+3)/4
	var px [16][4]uint8
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			o := (by*bw + bx) * 16
			if o+16 > len(blocks) {
				break
			}
			b := blocks[o : o+16]
			at := alphaTable(b[0], b[1])
			abits := uint64(b[2]) | uint64(b[3])<<8 | uint64(b[4])<<16 |
				uint64(b[5])<<24 | uint64(b[6])<<32 | uint64(b[7])<<40
			ct := colorTable(binary.LittleEndian.Uint16(b[8:]), binary.LittleEndian.Uint16(b[10:]), true)
			cbits := binary.LittleEndian.Uint32(b[12:])
			for i := 0; i < 16; i++ {
				c := ct[(cbits>>(uint(i)*2))&3]
				px[i] = [4]uint8{c[0], c[1], c[2], at[(abits>>(uint(i)*3))&7]}
			}
			blitBlock(img, bx, by, &px)
		}
	}
	return img
}

func decodeDXT1(blocks []byte, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bw, bh := (w+3)/4, (h+3)/4
	var px [16][4]uint8
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			o := (by*bw + bx) * 8
			if o+8 > len(blocks) {
				break
			}
			b := blocks[o : o+8]
			c0 := binary.LittleEndian.Uint16(b)
			c1 := binary.LittleEndian.Uint16(b[2:])
			ct := colorTable(c0, c1, false)
			cbits := binary.LittleEndian.Uint32(b[4:])
			for i := 0; i < 16; i++ {
				idx := (cbits >> (uint(i) * 2)) & 3
				c := ct[idx]
				a := uint8(255)
				if c0 <= c1 && idx == 3 {
					a = 0
				}
				px[i] = [4]uint8{c[0], c[1], c[2], a}
			}
			blitBlock(img, bx, by, &px)
		}
	}
	return img
}

func DecodeBC3(blocks []byte, w, h int) *image.NRGBA {
	return decodeDXT5(blocks, w, h)
}

func DecodeBC1(blocks []byte, w, h int) *image.NRGBA {
	return decodeDXT1(blocks, w, h)
}
