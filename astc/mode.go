package astc

import "fmt"

type blockMode struct {
	gridW      int
	gridH      int
	dual       bool
	quant      int
	weightBits int
}

func decodeMode(mode uint32, blockW, blockH int) (blockMode, error) {
	var m blockMode
	r := int(mode >> 4 & 1)
	p := int(mode >> 9 & 1)
	d := mode>>10&1 != 0
	a := int(mode >> 5 & 3)
	if mode&3 != 0 {
		r |= int(mode&3) << 1
		b := int(mode >> 7 & 3)
		switch mode >> 2 & 3 {
		case 0:
			m.gridW, m.gridH = b+4, a+2
		case 1:
			m.gridW, m.gridH = b+8, a+2
		case 2:
			m.gridW, m.gridH = a+2, b+8
		default:
			b &= 1
			if mode&0x100 != 0 {
				m.gridW, m.gridH = b+2, a+2
			} else {
				m.gridW, m.gridH = a+2, b+6
			}
		}
	} else {
		if mode>>2&3 == 0 {
			return m, fmt.Errorf("reserved block mode %#x", mode)
		}
		r |= int(mode>>2&3) << 1
		b := int(mode >> 9 & 3)
		switch mode >> 7 & 3 {
		case 0:
			m.gridW, m.gridH = 12, a+2
		case 1:
			m.gridW, m.gridH = a+2, 12
		case 2:
			m.gridW, m.gridH = a+6, b+6
			d, p = false, 0
		default:
			switch mode >> 5 & 3 {
			case 0:
				m.gridW, m.gridH = 6, 10
			case 1:
				m.gridW, m.gridH = 10, 6
			default:
				return m, fmt.Errorf("reserved block mode %#x", mode)
			}
		}
	}
	m.dual = d
	m.quant = r - 2 + 6*p
	count := m.gridW * m.gridH
	if d {
		count *= 2
	}
	if count > 64 {
		return m, fmt.Errorf("block mode %#x needs %d weights", mode, count)
	}
	m.weightBits = iseBits(count, m.quant)
	if m.weightBits < 24 || m.weightBits > 96 {
		return m, fmt.Errorf("block mode %#x weight bits %d", mode, m.weightBits)
	}
	if m.gridW > blockW || m.gridH > blockH {
		return m, fmt.Errorf("weight grid %dx%d exceeds block %dx%d", m.gridW, m.gridH, blockW, blockH)
	}
	return m, nil
}
