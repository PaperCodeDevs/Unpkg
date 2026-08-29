package crn

import "errors"

type dataModel struct {
	totalSyms uint32
	codeSizes []byte
	tables    *decoderTables
}

func (m *dataModel) valid() bool { return m.tables != nil }

func (m *dataModel) prepare() bool {
	m.totalSyms = uint32(len(m.codeSizes))
	if m.totalSyms == 0 || m.totalSyms > maxSupportedSyms {
		return false
	}
	if m.tables == nil {
		m.tables = &decoderTables{}
	}
	return m.tables.init(m.totalSyms, m.codeSizes, m.decoderTableBits())
}

func (m *dataModel) decoderTableBits() uint32 {
	if m.totalSyms <= 16 {
		return 0
	}
	b := 1 + ceilLog2(m.totalSyms)
	if b > maxTableBits {
		b = maxTableBits
	}
	return b
}

type codec struct {
	buf    []byte
	next   int
	end    int
	bitBuf uint32
	bitCnt int
}

func (c *codec) start(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	c.buf = buf
	c.next = 0
	c.end = len(buf)
	c.bitBuf = 0
	c.bitCnt = 0
	return true
}

func (c *codec) getBits(n uint32) uint32 {
	for c.bitCnt < int(n) {
		var b uint32
		if c.next != c.end {
			b = uint32(c.buf[c.next])
			c.next++
		}
		c.bitCnt += 8
		c.bitBuf |= b << (32 - uint(c.bitCnt))
	}
	r := c.bitBuf >> (32 - n)
	c.bitBuf <<= n
	c.bitCnt -= int(n)
	return r
}

func (c *codec) decodeBits(n uint32) uint32 {
	if n == 0 {
		return 0
	}
	if n > 16 {
		a := c.getBits(n - 16)
		b := c.getBits(16)
		return (a << 16) | b
	}
	return c.getBits(n)
}

func (c *codec) decode(m *dataModel) uint32 {
	if m == nil || m.tables == nil {
		return 0
	}
	t := m.tables
	if c.bitCnt < 24 {
		if c.bitCnt < 16 {
			var c0, c1 uint32
			if c.next < c.end {
				c0 = uint32(c.buf[c.next])
				c.next++
			}
			if c.next < c.end {
				c1 = uint32(c.buf[c.next])
				c.next++
			}
			c.bitCnt += 16
			c.bitBuf |= ((c0 << 8) | c1) << (32 - uint(c.bitCnt))
		} else {
			var b uint32
			if c.next < c.end {
				b = uint32(c.buf[c.next])
				c.next++
			}
			c.bitCnt += 8
			c.bitBuf |= b << (32 - uint(c.bitCnt))
		}
	}

	k := (c.bitBuf >> 16) + 1
	var sym, l uint32
	if k <= t.tableMaxCode {
		v := t.lookup[c.bitBuf>>(32-t.tableBits)]
		sym = v & 0xFFFF
		l = v >> 16
	} else {
		l = t.decodeStartCodeSiz
		for {
			if l > maxExpectedCodeSize {
				return 0
			}
			if k <= t.maxCodes[l-1] {
				break
			}
			l++
		}
		vp := t.valPtrs[l-1] + int32(c.bitBuf>>(32-l))
		if vp < 0 || uint32(vp) >= m.totalSyms || int(vp) >= len(t.sortedSymbolOrder) {
			return 0
		}
		sym = uint32(t.sortedSymbolOrder[vp])
	}
	c.bitBuf <<= l
	c.bitCnt -= int(l)
	return sym
}

const (
	maxCodelengthCodes = 21
	smallZeroRunCode   = 17
	largeZeroRunCode   = 18
	smallRepeatCode    = 19
	largeRepeatCode    = 20
)

var mostProbableCodelengthCodes = []byte{
	smallZeroRunCode, largeZeroRunCode,
	smallRepeatCode, largeRepeatCode,
	0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15, 16,
}

var errStream = errors.New("crn: 码流损坏")

func (c *codec) receiveModel(m *dataModel) error {
	total := c.decodeBits(14)
	if total == 0 {
		m.tables = nil
		m.codeSizes = nil
		m.totalSyms = 0
		return nil
	}
	if total > maxSupportedSyms {
		return errStream
	}
	m.codeSizes = make([]byte, total)

	numCl := c.decodeBits(5)
	if numCl < 1 || numCl > maxCodelengthCodes {
		return errStream
	}
	var dm dataModel
	dm.codeSizes = make([]byte, maxCodelengthCodes)
	for i := uint32(0); i < numCl; i++ {
		dm.codeSizes[mostProbableCodelengthCodes[i]] = byte(c.decodeBits(3))
	}
	if !dm.prepare() {
		return errStream
	}

	var ofs uint32
	for ofs < total {
		remain := total - ofs
		code := c.decode(&dm)
		switch {
		case code <= 16:
			m.codeSizes[ofs] = byte(code)
			ofs++
		case code == smallZeroRunCode:
			n := c.decodeBits(3) + 3
			if n > remain {
				return errStream
			}
			ofs += n
		case code == largeZeroRunCode:
			n := c.decodeBits(7) + 11
			if n > remain {
				return errStream
			}
			ofs += n
		case code == smallRepeatCode || code == largeRepeatCode:
			var n uint32
			if code == smallRepeatCode {
				n = c.decodeBits(2) + 3
			} else {
				n = c.decodeBits(6) + 7
			}
			if ofs == 0 || n > remain {
				return errStream
			}
			prev := m.codeSizes[ofs-1]
			if prev == 0 {
				return errStream
			}
			for end := ofs + n; ofs < end; ofs++ {
				m.codeSizes[ofs] = prev
			}
		default:
			return errStream
		}
	}
	if ofs != total || !m.prepare() {
		return errStream
	}
	return nil
}
