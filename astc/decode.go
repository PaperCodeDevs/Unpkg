package astc

import (
	"fmt"
	"image"
)

const blockBytes = 16

func UnityBlockSize(format uint32) (int, int) {
	switch format {
	case 48:
		return 4, 4
	case 49:
		return 5, 5
	case 50:
		return 6, 6
	case 51:
		return 8, 8
	case 52:
		return 10, 10
	case 53:
		return 12, 12
	}
	return 0, 0
}

func Decode(data []byte, width, height, blockW, blockH int) (*image.NRGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("astc: size %dx%d", width, height)
	}
	if blockW < 4 || blockW > 12 || blockH < 4 || blockH > 12 {
		return nil, fmt.Errorf("astc: block %dx%d", blockW, blockH)
	}
	bx := (width + blockW - 1) / blockW
	by := (height + blockH - 1) / blockH
	if need := bx * by * blockBytes; len(data) < need {
		return nil, fmt.Errorf("astc: need %d bytes, have %d", need, len(data))
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	px := make([]byte, blockW*blockH*4)
	for j := 0; j < by; j++ {
		for i := 0; i < bx; i++ {
			off := (j*bx + i) * blockBytes
			if err := DecodeBlock(data[off:off+blockBytes], blockW, blockH, px); err != nil {
				return nil, fmt.Errorf("astc: block (%d,%d): %w", i, j, err)
			}
			for y := 0; y < blockH; y++ {
				iy := j*blockH + y
				if iy >= height {
					break
				}
				for x := 0; x < blockW; x++ {
					ix := i*blockW + x
					if ix >= width {
						break
					}
					s := (y*blockW + x) * 4
					d := img.PixOffset(ix, iy)
					copy(img.Pix[d:d+4], px[s:s+4])
				}
			}
		}
	}
	return img, nil
}

func DecodeBlock(blk []byte, blockW, blockH int, out []byte) error {
	if len(blk) < blockBytes || len(out) < blockW*blockH*4 {
		return fmt.Errorf("short block or output")
	}
	mode := readBits(blk, 0, 11)
	if mode&0x1FF == 0x1FC {
		return decodeVoidExtent(blk, blockW*blockH, out)
	}
	m, err := decodeMode(mode, blockW, blockH)
	if err != nil {
		return err
	}
	parts := int(readBits(blk, 11, 2)) + 1
	if m.dual && parts == 4 {
		return fmt.Errorf("dual plane with 4 partitions")
	}
	h, err := parseHeader(blk, m, parts)
	if err != nil {
		return err
	}
	vals := make([]int, h.values)
	decodeISE(blk, h.colorOff, h.values, h.colorQ, vals)
	for i := range vals {
		vals[i] = unquantColor(vals[i], h.colorQ)
	}
	var e0, e1 [4]rgba
	off := 0
	for i := 0; i < parts; i++ {
		n := (h.cem[i]>>2 + 1) * 2
		e0[i], e1[i], err = decodeEndpoints(h.cem[i], vals[off:off+n])
		if err != nil {
			return err
		}
		off += n
	}
	p0, p1 := decodeWeights(blk, m, blockW, blockH)
	small := blockW*blockH < 31
	for t := 0; t < blockH; t++ {
		for s := 0; s < blockW; s++ {
			part := 0
			if parts > 1 {
				part = selectPartition(h.seed, s, t, parts, small)
			}
			idx := t*blockW + s
			for c := 0; c < 4; c++ {
				w := p0[idx]
				if m.dual && c == h.ccs {
					w = p1[idx]
				}
				c0, c1 := e0[part][c]*257, e1[part][c]*257
				out[idx*4+c] = byte((c0*(64-w) + c1*w + 32) >> 14)
			}
		}
	}
	return nil
}

func decodeVoidExtent(blk []byte, texels int, out []byte) error {
	if readBits(blk, 9, 1) != 0 {
		return fmt.Errorf("HDR void extent")
	}
	if readBits(blk, 10, 2) != 3 {
		return fmt.Errorf("void extent reserved bits")
	}
	lowS, highS := readBits(blk, 12, 13), readBits(blk, 25, 13)
	lowT, highT := readBits(blk, 38, 13), readBits(blk, 51, 13)
	allOnes := lowS == 0x1FFF && highS == 0x1FFF && lowT == 0x1FFF && highT == 0x1FFF
	if !allOnes && (lowS >= highS || lowT >= highT) {
		return fmt.Errorf("void extent range")
	}
	for i := 0; i < texels; i++ {
		out[i*4+0] = blk[9]
		out[i*4+1] = blk[11]
		out[i*4+2] = blk[13]
		out[i*4+3] = blk[15]
	}
	return nil
}
