package pkg

import (
	"encoding/binary"
	"fmt"
)

const dxWrapSub = 6

type DXWrap struct {
	Payload uint32
	Ver     uint8
	SRV     uint8
	CB      uint8
	Sampler uint8
	DXBC    []byte
	Extra   []byte
}

func ParseDXWrap(b []byte) (DXWrap, error) {
	// 10B 包装：u32 = 6 + DXBC.totalSize，随后 01 SRV数 CB数 采样器数 00 00
	var w DXWrap
	if len(b) < dxWrapLen+32 || string(b[dxWrapLen:dxWrapLen+4]) != dxFourCC {
		return w, fmt.Errorf("dx wrap magic")
	}
	w.Payload = binary.LittleEndian.Uint32(b[:4])
	w.Ver = b[4]
	w.SRV = b[5]
	w.CB = b[6]
	w.Sampler = b[7]
	tot := int(binary.LittleEndian.Uint32(b[dxWrapLen+24 : dxWrapLen+28]))
	if tot < 32 || dxWrapLen+tot > len(b) {
		return w, fmt.Errorf("dx wrap size %d", tot)
	}
	if int(w.Payload) != tot+dxWrapSub {
		return w, fmt.Errorf("dx wrap payload %d tot %d", w.Payload, tot)
	}
	w.DXBC = b[dxWrapLen : dxWrapLen+tot]
	w.Extra = b[dxWrapLen+tot:]
	return w, nil
}

func (w DXWrap) Binding() (DXBinding, error) {
	return ParseDXBinding(w.Extra)
}

func (w DXWrap) CountsOK(bd DXBinding) bool {
	return int(w.SRV) == len(bd.Textures)+len(bd.Buffers) && int(w.CB) == len(bd.CBuffers) && int(w.Sampler) == len(bd.Textures)
}

func DXProgramType(dx []byte) int {
	if len(dx) < 32 || string(dx[:4]) != dxFourCC {
		return -1
	}
	n := int(binary.LittleEndian.Uint32(dx[28:32]))
	if n <= 0 || n > dxMaxChunk {
		return -1
	}
	for i := 0; i < n && 36+4*i <= len(dx); i++ {
		off := int(binary.LittleEndian.Uint32(dx[32+4*i:]))
		if off < 0 || off+12 > len(dx) {
			continue
		}
		tag := string(dx[off : off+4])
		if tag == "SHDR" || tag == "SHEX" {
			return int(binary.LittleEndian.Uint32(dx[off+8:]) >> 16)
		}
	}
	return -1
}

func dxContainerLen(b []byte) int {
	if len(b) < 32 || string(b[:4]) != dxFourCC {
		return 0
	}
	if binary.LittleEndian.Uint32(b[20:24]) != 1 {
		return 0
	}
	n := int(binary.LittleEndian.Uint32(b[28:32]))
	if n <= 0 || n > dxMaxChunk {
		return 0
	}
	tot := int(binary.LittleEndian.Uint32(b[24:28]))
	if tot < 32+4*n || tot > len(b) {
		return 0
	}
	return tot
}

func ExtractAllDXBC(b []byte) [][]byte {
	var out [][]byte
	for i := 0; i+32 <= len(b); {
		if string(b[i:i+4]) != dxFourCC {
			i++
			continue
		}
		n := dxContainerLen(b[i:])
		if n == 0 {
			i++
			continue
		}
		out = append(out, b[i:i+n])
		i += n
	}
	return out
}
