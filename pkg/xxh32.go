package pkg

import (
	"encoding/binary"
	"math/bits"
)

const (
	RainbowNameSeed uint32 = 0x8F37154B

	xxPrime1 uint32 = 0x9E3779B1
	xxPrime2 uint32 = 0x85EBCA77
	xxPrime3 uint32 = 0xC2B2AE3D
	xxPrime4 uint32 = 0x27D4EB2F
	xxPrime5 uint32 = 0x165667B1
)

func xxRound(acc, input uint32) uint32 {
	return bits.RotateLeft32(acc+input*xxPrime2, 13) * xxPrime1
}

func XXH32(b []byte, seed uint32) uint32 {
	n := len(b)
	i := 0
	var h uint32
	if n >= 16 {
		v1, v2, v3, v4 := seed+xxPrime1+xxPrime2, seed+xxPrime2, seed, seed-xxPrime1
		for ; i+16 <= n; i += 16 {
			v1 = xxRound(v1, binary.LittleEndian.Uint32(b[i:]))
			v2 = xxRound(v2, binary.LittleEndian.Uint32(b[i+4:]))
			v3 = xxRound(v3, binary.LittleEndian.Uint32(b[i+8:]))
			v4 = xxRound(v4, binary.LittleEndian.Uint32(b[i+12:]))
		}
		h = bits.RotateLeft32(v1, 1) + bits.RotateLeft32(v2, 7) + bits.RotateLeft32(v3, 12) + bits.RotateLeft32(v4, 18)
	} else {
		h = seed + xxPrime5
	}
	h += uint32(n)
	for ; i+4 <= n; i += 4 {
		h = bits.RotateLeft32(h+binary.LittleEndian.Uint32(b[i:])*xxPrime3, 17) * xxPrime4
	}
	for ; i < n; i++ {
		h = bits.RotateLeft32(h+uint32(b[i])*xxPrime5, 11) * xxPrime1
	}
	h ^= h >> 15
	h *= xxPrime2
	h ^= h >> 13
	h *= xxPrime3
	h ^= h >> 16
	return h
}

func RainbowNameHash(s string) uint32 {
	return XXH32([]byte(s), RainbowNameSeed)
}
