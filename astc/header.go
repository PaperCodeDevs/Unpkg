package astc

import "fmt"

type blockHeader struct {
	cem      [4]int
	seed     int
	colorOff int
	colorQ   int
	ccs      int
	values   int
}

func parseHeader(blk []byte, m blockMode, parts int) (blockHeader, error) {
	var h blockHeader
	below := 128 - m.weightBits
	configBits := 17
	extra := 0
	if parts == 1 {
		h.cem[0] = int(readBits(blk, 13, 4))
	} else {
		h.seed = int(readBits(blk, 13, 10))
		configBits = 29
		enc := int(readBits(blk, 23, 6))
		if enc&3 == 0 {
			for i := 0; i < parts; i++ {
				h.cem[i] = enc >> 2 & 0xF
			}
		} else {
			extra = 3*parts - 4
			below -= extra
			enc |= int(readBits(blk, below, extra)) << 6
			base := enc&3 - 1
			for i := 0; i < parts; i++ {
				h.cem[i] = (base + enc>>uint(2+i)&1) << 2
			}
			pos := 2 + parts
			for i := 0; i < parts; i++ {
				h.cem[i] |= enc >> uint(pos) & 3
				pos += 2
			}
		}
	}
	h.colorOff = configBits
	for i := 0; i < parts; i++ {
		h.values += (h.cem[i]>>2 + 1) * 2
	}
	if h.values > 18 {
		return h, fmt.Errorf("%d color integers", h.values)
	}
	colorBits := 128 - configBits - m.weightBits - extra
	if m.dual {
		colorBits -= 2
		h.ccs = int(readBits(blk, below-2, 2))
	}
	h.colorQ = -1
	for q := quant256; q >= quant6; q-- {
		if iseBits(h.values, q) <= colorBits {
			h.colorQ = q
			break
		}
	}
	if h.colorQ < 0 {
		return h, fmt.Errorf("%d color bits for %d integers", colorBits, h.values)
	}
	return h, nil
}
