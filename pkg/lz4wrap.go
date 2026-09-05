package pkg

import "encoding/binary"

const lz4WrapMax = 64 << 20

func DecodeWrapped(b []byte) ([]byte, bool) {
	if len(b) < 8 {
		return nil, false
	}
	size := int(binary.LittleEndian.Uint32(b[:4]))
	if size <= 0 || size > lz4WrapMax || size < len(b)/lz4MaxRatio {
		return nil, false
	}
	out, err := DecompressLZ4Block(b[4:], size)
	if err != nil || len(out) != size {
		return nil, false
	}
	return out, true
}

func IsWrapped(b []byte) bool {
	_, ok := DecodeWrapped(b)
	return ok
}
