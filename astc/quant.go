package astc

type quant struct {
	bits   int
	trits  int
	quints int
	max    int
}

const (
	quant2 = iota
	quant3
	quant4
	quant5
	quant6
	quant8
	quant10
	quant12
	quant16
	quant20
	quant24
	quant32
	quant40
	quant48
	quant64
	quant80
	quant96
	quant128
	quant160
	quant192
	quant256
)

var quants = [...]quant{
	{1, 0, 0, 1}, {0, 1, 0, 2}, {2, 0, 0, 3}, {0, 0, 1, 4}, {1, 1, 0, 5},
	{3, 0, 0, 7}, {1, 0, 1, 9}, {2, 1, 0, 11}, {4, 0, 0, 15}, {2, 0, 1, 19},
	{3, 1, 0, 23}, {5, 0, 0, 31}, {3, 0, 1, 39}, {4, 1, 0, 47}, {6, 0, 0, 63},
	{4, 0, 1, 79}, {5, 1, 0, 95}, {7, 0, 0, 127}, {5, 0, 1, 159}, {6, 1, 0, 191},
	{8, 0, 0, 255},
}

type unquantSpec struct {
	layout string
	c      int
}

// 规范 Table 158/168 的 B 位布局，MSB 在前，字母对应量化值第几位（a=bit0）
var colorUnquant = map[int]unquantSpec{
	quant6: {"000000000", 204}, quant10: {"000000000", 113},
	quant12: {"b000b0bb0", 93}, quant20: {"b0000bb00", 54},
	quant24: {"cb000cbcb", 44}, quant40: {"cb0000cbc", 26},
	quant48: {"dcb000dcb", 22}, quant80: {"dcb0000dc", 13},
	quant96: {"edcb000ed", 11}, quant160: {"edcb0000e", 6},
	quant192: {"fedcb000f", 5},
}

var weightUnquant = map[int]unquantSpec{
	quant6: {"0000000", 50}, quant10: {"0000000", 28},
	quant12: {"b000b0b", 23}, quant20: {"b0000b0", 13},
	quant24: {"cb000cb", 11},
}

func iseBits(count, q int) int {
	qi := quants[q]
	return (count*8*qi.trits+4)/5 + (count*7*qi.quints+2)/3 + count*qi.bits
}

func layoutBits(layout string, m int) int {
	v := 0
	n := len(layout)
	for i := 0; i < n; i++ {
		ch := layout[i]
		if ch == '0' {
			continue
		}
		v |= (m >> uint(ch-'a') & 1) << uint(n-1-i)
	}
	return v
}

func replicate(v, bits, width int) int {
	out := 0
	for shift := width; shift > 0; {
		shift -= bits
		if shift >= 0 {
			out |= v << uint(shift)
		} else {
			out |= v >> uint(-shift)
		}
	}
	return out
}

func unquantColor(v, q int) int {
	qi := quants[q]
	if qi.trits == 0 && qi.quints == 0 {
		return replicate(v, qi.bits, 8)
	}
	m := v & (1<<uint(qi.bits) - 1)
	d := v >> uint(qi.bits)
	spec := colorUnquant[q]
	a := 0
	if m&1 != 0 {
		a = 0x1FF
	}
	unq := d*spec.c + layoutBits(spec.layout, m)
	unq ^= a
	return a&0x80 | unq>>2
}

func unquantWeight(v, q int) int {
	qi := quants[q]
	var unq int
	switch {
	case qi.trits == 0 && qi.quints == 0:
		unq = replicate(v, qi.bits, 6)
	case qi.bits == 0 && qi.trits == 1:
		unq = [3]int{0, 32, 63}[v]
	case qi.bits == 0:
		unq = [5]int{0, 16, 32, 47, 63}[v]
	default:
		m := v & (1<<uint(qi.bits) - 1)
		d := v >> uint(qi.bits)
		spec := weightUnquant[q]
		a := 0
		if m&1 != 0 {
			a = 0x7F
		}
		unq = d*spec.c + layoutBits(spec.layout, m)
		unq ^= a
		unq = a&0x20 | unq>>2
	}
	if unq > 32 {
		unq++
	}
	return unq
}
