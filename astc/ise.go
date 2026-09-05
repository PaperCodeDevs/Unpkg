package astc

func tritsOf(t uint32) [5]int {
	var out [5]int
	var c uint32
	if t>>2&7 == 7 {
		c = (t>>5&7)<<2 | t&3
		out[4], out[3] = 2, 2
	} else {
		c = t & 0x1F
		if t>>5&3 == 3 {
			out[4] = 2
			out[3] = int(t >> 7 & 1)
		} else {
			out[4] = int(t >> 7 & 1)
			out[3] = int(t >> 5 & 3)
		}
	}
	switch {
	case c&3 == 3:
		out[2] = 2
		out[1] = int(c >> 4 & 1)
		out[0] = int((c>>3&1)<<1 | (c>>2&1)&^(c>>3&1))
	case c>>2&3 == 3:
		out[2], out[1] = 2, 2
		out[0] = int(c & 3)
	default:
		out[2] = int(c >> 4 & 1)
		out[1] = int(c >> 2 & 3)
		out[0] = int((c>>1&1)<<1 | (c&1)&^(c>>1&1))
	}
	return out
}

func quintsOf(q uint32) [3]int {
	var out [3]int
	if q>>1&3 == 3 && q>>5&3 == 0 {
		q0 := q & 1
		out[2] = int(q0<<2 | ((q>>4&1)&^q0)<<1 | (q>>3&1)&^q0)
		out[1], out[0] = 4, 4
		return out
	}
	var c uint32
	if q>>1&3 == 3 {
		out[2] = 4
		c = (q>>3&3)<<3 | (^q>>5&3)<<1 | q&1
	} else {
		out[2] = int(q >> 5 & 3)
		c = q & 0x1F
	}
	if c&7 == 5 {
		out[1] = 4
		out[0] = int(c >> 3 & 3)
	} else {
		out[1] = int(c >> 3 & 3)
		out[0] = int(c & 7)
	}
	return out
}

func decodeISE(b []byte, off, count, q int, out []int) {
	qi := quants[q]
	var tq [22]uint32
	lc, hc := 0, 0
	for i := 0; i < count; i++ {
		out[i] = int(readBits(b, off, qi.bits))
		off += qi.bits
		if qi.trits != 0 {
			n := [5]int{2, 2, 1, 2, 1}[lc]
			tq[hc] |= readBits(b, off, n) << [5]uint{0, 2, 4, 5, 7}[lc]
			off += n
			lc++
			if lc == 5 {
				lc = 0
				hc++
			}
		}
		if qi.quints != 0 {
			n := [3]int{3, 2, 2}[lc]
			tq[hc] |= readBits(b, off, n) << [3]uint{0, 3, 5}[lc]
			off += n
			lc++
			if lc == 3 {
				lc = 0
				hc++
			}
		}
	}
	if qi.trits != 0 {
		for i := 0; i*5 < count; i++ {
			t := tritsOf(tq[i])
			for j := 0; j < 5 && i*5+j < count; j++ {
				out[i*5+j] |= t[j] << uint(qi.bits)
			}
		}
	}
	if qi.quints != 0 {
		for i := 0; i*3 < count; i++ {
			t := quintsOf(tq[i])
			for j := 0; j < 3 && i*3+j < count; j++ {
				out[i*3+j] |= t[j] << uint(qi.bits)
			}
		}
	}
}
