package astc

func readBits(b []byte, off, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := off + i
		if p < 0 || p >= len(b)*8 {
			continue
		}
		v |= uint32(b[p>>3]>>uint(p&7)&1) << uint(i)
	}
	return v
}

func bitrev8(v byte) byte {
	v = v>>4 | v<<4
	v = v&0xCC>>2 | v&0x33<<2
	v = v&0xAA>>1 | v&0x55<<1
	return v
}

func reverseBlock(src []byte) []byte {
	out := make([]byte, 16)
	for i := range out {
		out[i] = bitrev8(src[15-i])
	}
	return out
}
