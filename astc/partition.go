package astc

func hash52(p uint32) uint32 {
	p ^= p >> 15
	p *= 0xEEDE0891
	p ^= p >> 5
	p += p << 16
	p ^= p >> 7
	p ^= p >> 3
	p ^= p << 6
	p ^= p >> 17
	return p
}

func selectPartition(seed, x, y, count int, small bool) int {
	if small {
		x <<= 1
		y <<= 1
	}
	seed += (count - 1) * 1024
	rnum := hash52(uint32(seed))
	var s [8]int
	for i := range s {
		v := int(rnum >> uint(4*i) & 0xF)
		s[i] = v * v
	}
	sh1, sh2 := 5, 5
	if seed&1 != 0 {
		if seed&2 != 0 {
			sh1 = 4
		}
		if count == 3 {
			sh2 = 6
		}
	} else {
		if count == 3 {
			sh1 = 6
		}
		if seed&2 != 0 {
			sh2 = 4
		}
	}
	for i := 0; i < 8; i += 2 {
		s[i] >>= uint(sh1)
		s[i+1] >>= uint(sh2)
	}
	a := (s[0]*x + s[1]*y + int(rnum>>14)) & 0x3F
	b := (s[2]*x + s[3]*y + int(rnum>>10)) & 0x3F
	c := (s[4]*x + s[5]*y + int(rnum>>6)) & 0x3F
	d := (s[6]*x + s[7]*y + int(rnum>>2)) & 0x3F
	if count < 4 {
		d = 0
	}
	if count < 3 {
		c = 0
	}
	switch {
	case a >= b && a >= c && a >= d:
		return 0
	case b >= c && b >= d:
		return 1
	case c >= d:
		return 2
	}
	return 3
}
