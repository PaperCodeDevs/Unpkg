package pkg

import (
	"encoding/binary"
	"fmt"
)

var (
	lz4Dec32 = [8]int{4, 1, 2, 1, 4, 4, 4, 4}
	lz4Dec64 = [8]int{0, 0, 0, -1, 0, 1, 2, 3}
)

func DecompressLZ4Block(src []byte, maxSize int) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("DecompressLZ4Block: empty")
	}
	if maxSize <= 0 {
		maxSize = 128 << 20
	}
	dst := make([]byte, maxSize+32)
	iend := len(src)
	oend := maxSize
	ip := 0
	op := 0
	for {
		if ip >= iend {
			return nil, fmt.Errorf("DecompressLZ4Block: ip past end at op=%d", op)
		}
		token := int(src[ip])
		ip++
		litLen := token >> 4
		if litLen == 15 {
			for {
				if ip >= iend {
					return nil, fmt.Errorf("DecompressLZ4Block: lit-ext past end")
				}
				s := int(src[ip])
				ip++
				litLen += s
				if s != 255 {
					break
				}
				if ip >= iend-15 {
					break
				}
			}
		}
		cpy := op + litLen
		if cpy > oend-12 || ip+litLen > iend-8 {
			if ip+litLen != iend || cpy > oend {
				return nil, fmt.Errorf("DecompressLZ4Block: bad final ip=%d litLen=%d iend=%d cpy=%d oend=%d", ip, litLen, iend, cpy, oend)
			}
			copy(dst[op:cpy], src[ip:ip+litLen])
			op = cpy
			out := make([]byte, op)
			copy(out, dst[:op])
			return out, nil
		}
		{
			d := op
			s := ip
			for {
				dst[d+0] = src[s+0]
				dst[d+1] = src[s+1]
				dst[d+2] = src[s+2]
				dst[d+3] = src[s+3]
				dst[d+4] = src[s+4]
				dst[d+5] = src[s+5]
				dst[d+6] = src[s+6]
				dst[d+7] = src[s+7]
				d += 8
				s += 8
				if d >= cpy {
					break
				}
			}
		}
		ip += litLen
		op = cpy
		if ip+2 > iend {
			return nil, fmt.Errorf("DecompressLZ4Block: offset past end")
		}
		offset := int(binary.LittleEndian.Uint16(src[ip:]))
		ip += 2
		match := op - offset
		if match < 0 {
			return nil, fmt.Errorf("DecompressLZ4Block: offset > op: off=%d op=%d", offset, op)
		}
		mLen := token & 15
		if mLen == 15 {
			for {
				if ip > iend-5 {
					return nil, fmt.Errorf("DecompressLZ4Block: mLen-ext past end")
				}
				s := int(src[ip])
				ip++
				mLen += s
				if s != 255 {
					break
				}
			}
		}
		mLen += 4
		cpy = op + mLen
		if op-match < 8 {
			diff := op - match
			dst[op+0] = dst[match+0]
			dst[op+1] = dst[match+1]
			dst[op+2] = dst[match+2]
			dst[op+3] = dst[match+3]
			match += lz4Dec32[diff]
			dst[op+4] = dst[match+0]
			dst[op+5] = dst[match+1]
			dst[op+6] = dst[match+2]
			dst[op+7] = dst[match+3]
			op += 8
			match -= lz4Dec64[diff]
		} else {
			v := binary.LittleEndian.Uint64(dst[match : match+8])
			binary.LittleEndian.PutUint64(dst[op:op+8], v)
			op += 8
			match += 8
		}
		if cpy > oend-12 {
			if cpy > oend-5 {
				return nil, fmt.Errorf("DecompressLZ4Block: match crosses last-5 cpy=%d oend=%d", cpy, oend)
			}
			if op < oend-8 {
				d := op
				s := match
				e := oend - 8
				for {
					v := binary.LittleEndian.Uint64(dst[s : s+8])
					binary.LittleEndian.PutUint64(dst[d:d+8], v)
					d += 8
					s += 8
					if d >= e {
						break
					}
				}
				match += (oend - 8) - op
				op = oend - 8
			}
			for op < cpy {
				dst[op] = dst[match]
				op++
				match++
			}
		} else {
			d := op
			s := match
			for {
				v := binary.LittleEndian.Uint64(dst[s : s+8])
				binary.LittleEndian.PutUint64(dst[d:d+8], v)
				d += 8
				s += 8
				if d >= cpy {
					break
				}
			}
		}
		op = cpy
	}
}

func (p *Pkg) DecompressData(maxSize int) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("DecompressData: nil pkg")
	}
	return DecompressLZ4Block(p.Data, maxSize)
}
