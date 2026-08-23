package crn

import "encoding/binary"

func crc16(b []byte) uint16 {
	crc := ^uint16(0)
	for _, v := range b {
		q := uint16(v) ^ (crc >> 8)
		crc <<= 8
		r := (q >> 4) ^ q
		crc ^= r
		r <<= 5
		crc ^= r
		r <<= 7
		crc ^= r
	}
	return ^crc
}

func Validate(data []byte) bool {
	h, err := ParseHeader(data)
	if err != nil {
		return false
	}
	if int(h.HeaderSize) > len(data) {
		return false
	}
	if crc16(data[6:h.HeaderSize]) != binary.BigEndian.Uint16(data[4:]) {
		return false
	}
	return crc16(data[h.HeaderSize:h.DataSize]) == binary.BigEndian.Uint16(data[10:])
}
