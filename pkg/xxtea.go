package pkg

func decryptXXTEA(data []byte, key []byte, includeLength bool) []byte {
	if len(data) == 0 {
		return data
	}
	v := xxteaBytesToUint32s(data, false)
	if len(v) < 2 {
		return data
	}
	k := xxteaBytesToUint32s(xxteaPadKey(key), false)
	if len(k) < 4 {
		return data
	}
	n := len(v) - 1
	z := v[n]
	y := v[0]
	delta := uint32(0x9e3779b9)
	q := 6 + 52/(n+1)
	sum := uint32(q) * delta
	for sum != 0 {
		e := (sum >> 2) & 3
		for p := n; p > 0; p-- {
			z = v[p-1]
			y = v[p] - xxteaMX(sum, y, z, p, int(e), k)
			v[p] = y
		}
		z = v[n]
		y = v[0] - xxteaMX(sum, y, z, 0, int(e), k)
		v[0] = y
		sum -= delta
	}
	return xxteaUint32sToBytes(v, includeLength)
}

func xxteaMX(sum, y, z uint32, p, e int, k []uint32) uint32 {
	return ((z>>5 ^ y<<2) + (y>>3 ^ z<<4)) ^ ((sum ^ y) + (k[(p&3)^e] ^ z))
}

func xxteaPadKey(key []byte) []byte {
	out := make([]byte, 16)
	copy(out, key)
	return out
}

func xxteaBytesToUint32s(data []byte, includeLength bool) []uint32 {
	length := len(data)
	n := (length + 3) >> 2
	var result []uint32
	if includeLength {
		result = make([]uint32, n+1)
		result[n] = uint32(length)
	} else {
		result = make([]uint32, n)
	}
	for i := 0; i < length; i++ {
		result[i>>2] |= uint32(data[i]) << ((i & 3) << 3)
	}
	return result
}

func xxteaUint32sToBytes(data []uint32, includeLength bool) []byte {
	n := len(data) << 2
	if includeLength {
		if len(data) == 0 {
			return nil
		}
		m := int(data[len(data)-1])
		n -= 4
		if m < 0 || m > n {
			return nil
		}
		n = m
	}
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = byte(data[i>>2] >> ((i & 3) << 3))
	}
	return result
}
