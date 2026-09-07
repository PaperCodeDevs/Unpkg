package zdeflate

type ct struct {
	freq uint16
	code uint16
	dad  uint16
	len  uint16
}

var staticLtree [lCodes + 2]ct
var staticDtree [dCodes]ct

func init() {
	n := 0
	for n <= 143 {
		staticLtree[n].len = 8
		n++
	}
	for n <= 255 {
		staticLtree[n].len = 9
		n++
	}
	for n <= 279 {
		staticLtree[n].len = 7
		n++
	}
	for n <= 287 {
		staticLtree[n].len = 8
		n++
	}
	var bl [maxBits + 1]uint16
	for i := 0; i < lCodes+2; i++ {
		bl[staticLtree[i].len]++
	}
	genCodes(staticLtree[:], lCodes+1, bl)
	for i := 0; i < dCodes; i++ {
		staticDtree[i].len = 5
		staticDtree[i].code = uint16(biReverse(uint(i), 5))
	}
}

func biReverse(code uint, length int) uint {
	res := uint(0)
	for {
		res |= code & 1
		code >>= 1
		res <<= 1
		length--
		if length <= 0 {
			break
		}
	}
	return res >> 1
}
