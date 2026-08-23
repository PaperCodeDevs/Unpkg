package crn

var dxt5FromLinear = [8]uint32{0, 2, 3, 4, 5, 6, 7, 1}

func (u *unpacker) slice(p palette) ([]byte, bool) {
	if p.Ofs > uint32(len(u.data)) || p.Ofs+p.Size > uint32(len(u.data)) {
		return nil, false
	}
	return u.data[p.Ofs : p.Ofs+p.Size], true
}

func (u *unpacker) decodeColorEndpoints() error {
	n := u.h.ColorEndpoints.Num
	u.colorEndpoints = make([]uint32, n)
	src, ok := u.slice(u.h.ColorEndpoints)
	if !ok || !u.codec.start(src) {
		return errStream
	}
	var dm [2]dataModel
	for i := 0; i < 2; i++ {
		if err := u.codec.receiveModel(&dm[i]); err != nil {
			return err
		}
	}
	var a, b, c, d, e, f uint32
	for i := uint32(0); i < n; i++ {
		a = (a + u.codec.decode(&dm[0])) & 31
		b = (b + u.codec.decode(&dm[1])) & 63
		c = (c + u.codec.decode(&dm[0])) & 31
		d = (d + u.codec.decode(&dm[0])) & 31
		e = (e + u.codec.decode(&dm[1])) & 63
		f = (f + u.codec.decode(&dm[0])) & 31
		u.colorEndpoints[i] = c | (b << 5) | (a << 11) | (f << 16) | (e << 21) | (d << 27)
	}
	return nil
}

func (u *unpacker) decodeColorSelectors() error {
	n := u.h.ColorSelectors.Num
	src, ok := u.slice(u.h.ColorSelectors)
	if !ok || !u.codec.start(src) {
		return errStream
	}
	var dm dataModel
	if err := u.codec.receiveModel(&dm); err != nil {
		return err
	}
	u.colorSelectors = make([]uint32, n)
	var s uint32
	for i := uint32(0); i < n; i++ {
		for j := uint32(0); j < 32; j += 4 {
			s ^= u.codec.decode(&dm) << j
		}
		u.colorSelectors[i] = ((s ^ s<<1) & 0xAAAAAAAA) | (s >> 1 & 0x55555555)
	}
	return nil
}

func (u *unpacker) decodeAlphaEndpoints() error {
	n := u.h.AlphaEndpoints.Num
	src, ok := u.slice(u.h.AlphaEndpoints)
	if !ok || !u.codec.start(src) {
		return errStream
	}
	var dm dataModel
	if err := u.codec.receiveModel(&dm); err != nil {
		return err
	}
	u.alphaEndpoints = make([]uint16, n)
	var a, b uint32
	for i := uint32(0); i < n; i++ {
		a = (a + u.codec.decode(&dm)) & 255
		b = (b + u.codec.decode(&dm)) & 255
		u.alphaEndpoints[i] = uint16(a | (b << 8))
	}
	return nil
}

func (u *unpacker) decodeAlphaSelectors() error {
	n := u.h.AlphaSelectors.Num
	src, ok := u.slice(u.h.AlphaSelectors)
	if !ok || !u.codec.start(src) {
		return errStream
	}
	var dm dataModel
	if err := u.codec.receiveModel(&dm); err != nil {
		return err
	}
	u.alphaSelectors = make([]uint16, n*3)
	var fromLin [64]uint32
	for i := 0; i < 64; i++ {
		fromLin[i] = dxt5FromLinear[i&7] | dxt5FromLinear[i>>3]<<3
	}
	var s0lin, s1lin uint32
	for i := 0; i < len(u.alphaSelectors); {
		var s0, s1 uint32
		for j := uint32(0); j < 24; j += 6 {
			s0lin ^= u.codec.decode(&dm) << j
			s0 |= fromLin[s0lin>>j&0x3F] << j
		}
		for j := uint32(0); j < 24; j += 6 {
			s1lin ^= u.codec.decode(&dm) << j
			s1 |= fromLin[s1lin>>j&0x3F] << j
		}
		u.alphaSelectors[i] = uint16(s0)
		i++
		u.alphaSelectors[i] = uint16(s0>>16 | s1<<8)
		i++
		u.alphaSelectors[i] = uint16(s1 >> 8)
		i++
	}
	return nil
}
