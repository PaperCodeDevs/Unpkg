package crn

import (
	"encoding/binary"
	"errors"
)

const (
	FmtDXT1  = 0
	FmtDXT3  = 1
	FmtDXT5  = 2
	FmtDXN   = 7
	FmtDXT5A = 9
)

const sigValue = 0x4878

type palette struct {
	Ofs  uint32
	Size uint32
	Num  uint32
}

type Header struct {
	HeaderSize uint32
	DataSize   uint32
	Width      uint32
	Height     uint32
	Levels     uint32
	Faces      uint32
	Format     uint32
	Flags      uint32

	ColorEndpoints palette
	ColorSelectors palette
	AlphaEndpoints palette
	AlphaSelectors palette

	TablesSize uint32
	TablesOfs  uint32
	LevelOfs   []uint32
}

var ErrNotCRN = errors.New("crn: 不是 crn 数据")

func be24(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

func readPalette(b []byte) palette {
	return palette{Ofs: be24(b), Size: be24(b[3:]), Num: uint32(binary.BigEndian.Uint16(b[6:]))}
}

func ParseHeader(b []byte) (*Header, error) {
	if len(b) < 74 || binary.BigEndian.Uint16(b) != sigValue {
		return nil, ErrNotCRN
	}
	h := &Header{
		HeaderSize: uint32(binary.BigEndian.Uint16(b[2:])),
		DataSize:   binary.BigEndian.Uint32(b[6:]),
		Width:      uint32(binary.BigEndian.Uint16(b[12:])),
		Height:     uint32(binary.BigEndian.Uint16(b[14:])),
		Levels:     uint32(b[16]),
		Faces:      uint32(b[17]),
		Format:     uint32(b[18]),
		Flags:      uint32(binary.BigEndian.Uint16(b[19:])),
	}
	h.ColorEndpoints = readPalette(b[33:])
	h.ColorSelectors = readPalette(b[41:])
	h.AlphaEndpoints = readPalette(b[49:])
	h.AlphaSelectors = readPalette(b[57:])
	h.TablesSize = uint32(binary.BigEndian.Uint16(b[65:]))
	h.TablesOfs = be24(b[67:])

	if h.Levels == 0 || h.Levels > 32 || h.Faces == 0 || h.Faces > 6 {
		return nil, ErrNotCRN
	}
	need := 70 + int(h.Levels)*4
	if int(h.HeaderSize) < need || len(b) < need {
		return nil, ErrNotCRN
	}
	h.LevelOfs = make([]uint32, h.Levels)
	for i := uint32(0); i < h.Levels; i++ {
		h.LevelOfs[i] = binary.BigEndian.Uint32(b[70+i*4:])
	}
	if int(h.DataSize) > len(b) {
		return nil, ErrNotCRN
	}
	return h, nil
}

func (h *Header) blockSize() uint32 {
	if h.Format == FmtDXT1 || h.Format == FmtDXT5A {
		return 8
	}
	return 16
}
