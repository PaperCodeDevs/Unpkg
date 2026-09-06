package pkg

import "errors"

var ErrServerListKey = errors.New("serverlist: 密钥为空或超过 16 字节")

var desIP = [64]uint8{57, 49, 41, 33, 25, 17, 9, 1, 59, 51, 43, 35, 27, 19, 11, 3, 61, 53, 45, 37, 29, 21, 13, 5, 63, 55, 47, 39, 31, 23, 15, 7, 56, 48, 40, 32, 24, 16, 8, 0, 58, 50, 42, 34, 26, 18, 10, 2, 60, 52, 44, 36, 28, 20, 12, 4, 62, 54, 46, 38, 30, 22, 14, 6}
var desFP = [64]uint8{39, 7, 47, 15, 55, 23, 63, 31, 38, 6, 46, 14, 54, 22, 62, 30, 37, 5, 45, 13, 53, 21, 61, 29, 36, 4, 44, 12, 52, 20, 60, 28, 35, 3, 43, 11, 51, 19, 59, 27, 34, 2, 42, 10, 50, 18, 58, 26, 33, 1, 41, 9, 49, 17, 57, 25, 32, 0, 40, 8, 48, 16, 56, 24}
var desE = [48]uint8{31, 0, 1, 2, 3, 4, 3, 4, 5, 6, 7, 8, 7, 8, 9, 10, 11, 12, 11, 12, 13, 14, 15, 16, 15, 16, 17, 18, 19, 20, 19, 20, 21, 22, 23, 24, 23, 24, 25, 26, 27, 28, 27, 28, 29, 30, 31, 0}
var desP = [32]uint8{15, 6, 19, 20, 28, 11, 27, 16, 0, 14, 22, 25, 4, 17, 30, 9, 1, 7, 23, 13, 31, 26, 2, 8, 18, 12, 29, 5, 21, 10, 3, 24}
var desPC1 = [56]uint8{56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1, 58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35, 62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5, 60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3}
var desPC2 = [48]uint8{13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1, 40, 51, 30, 36, 46, 54, 29, 39, 50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31}
var desShifts = [16]uint8{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}
var desS = [8][64]uint8{
	{14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7, 0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8, 4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0, 15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13},
	{15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10, 3, 13, 4, 7, 15, 2, 8, 14, 12, 0, 1, 10, 6, 9, 11, 5, 0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15, 13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9},
	{10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8, 13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1, 13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7, 1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12},
	{7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15, 13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9, 10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4, 3, 15, 0, 6, 10, 1, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14},
	{2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9, 14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6, 4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14, 11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3},
	{12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11, 10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8, 9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6, 4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13},
	{4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1, 13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6, 1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2, 6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12},
	{13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7, 1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2, 7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8, 2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11},
}

type desSchedule [16][48]uint8

func desKeySchedule(key8 []byte) *desSchedule {
	var kb [64]uint8
	for i := range kb {
		kb[i] = key8[i>>3] >> (i & 7) & 1
	}
	var cd [56]uint8
	for i, p := range desPC1 {
		cd[i] = kb[p]
	}
	ks := new(desSchedule)
	for r := range 16 {
		s := int(desShifts[r])
		var rot [56]uint8
		copy(rot[:28], cd[s:28])
		copy(rot[28-s:28], cd[:s])
		copy(rot[28:], cd[28+s:])
		copy(rot[56-s:], cd[28:28+s])
		cd = rot
		for i, p := range desPC2 {
			ks[r][i] = cd[p]
		}
	}
	return ks
}

func desFeistel(r []uint8, k *[48]uint8) {
	var e [48]uint8
	for i, p := range desE {
		e[i] = r[p] ^ k[i]
	}
	var s [32]uint8
	for g := range 8 {
		b := e[g*6 : g*6+6]
		row := int(b[0])<<1 | int(b[5])
		col := int(b[1])<<3 | int(b[2])<<2 | int(b[3])<<1 | int(b[4])
		v := desS[g][row*16+col]
		for j := range 4 {
			s[g*4+j] = v >> j & 1
		}
	}
	for i, p := range desP {
		r[i] = s[p]
	}
}

func desBlock(in []byte, ks *desSchedule, reverse bool) [8]byte {
	var bits, st [64]uint8
	for i := range bits {
		bits[i] = in[i>>3] >> (i & 7) & 1
	}
	for i, p := range desIP {
		st[i] = bits[p]
	}
	a, b := st[32:], st[:32]
	if reverse {
		a, b = st[:32], st[32:]
	}
	for r := range 16 {
		k := &ks[r]
		if reverse {
			k = &ks[15-r]
		}
		var tmp [32]uint8
		copy(tmp[:], a)
		desFeistel(a, k)
		for i := range a {
			a[i] ^= b[i]
		}
		copy(b, tmp[:])
	}
	var out [8]byte
	for i, p := range desFP {
		out[i>>3] |= st[p] << (i & 7)
	}
	return out
}

func serverListSchedules(key []byte) (*desSchedule, *desSchedule, error) {
	if len(key) == 0 || len(key) > 16 {
		return nil, nil, ErrServerListKey
	}
	var k16 [16]byte
	copy(k16[:], key)
	if len(key) == 8 {
		return desKeySchedule(k16[:8]), nil, nil
	}
	return desKeySchedule(k16[:8]), desKeySchedule(k16[8:]), nil
}

func serverListEDE(src []byte, k1, k2 *desSchedule, decrypt bool) []byte {
	out := make([]byte, (len(src)+7)&^7)
	for i := 0; i+8 <= len(out); i += 8 {
		var blk [8]byte
		copy(blk[:], src[i:])
		blk = desBlock(blk[:], k1, !decrypt)
		if k2 != nil {
			blk = desBlock(blk[:], k2, decrypt)
			blk = desBlock(blk[:], k1, !decrypt)
		}
		copy(out[i:], blk[:])
	}
	return out
}

func DecryptServerList(body, key []byte) ([]byte, error) {
	k1, k2, err := serverListSchedules(key)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || len(body)%8 != 0 {
		return nil, errors.New("serverlist: 长度不是 8 的倍数")
	}
	plain := serverListEDE(body, k1, k2, true)
	end := len(plain)
	for end > 0 && plain[end-1] == 0 {
		end--
	}
	return plain[:end], nil
}

func EncryptServerList(plain, key []byte) ([]byte, error) {
	k1, k2, err := serverListSchedules(key)
	if err != nil {
		return nil, err
	}
	return serverListEDE(plain, k1, k2, false), nil
}
