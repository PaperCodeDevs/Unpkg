package pkg

import (
	"encoding/binary"
	"fmt"
	"image"

	"github.com/PaperCodeDevs/Unpkg/crn"
)

const (
	rbTexHash    = 0xD90C0772
	rbTexFmtCRN  = 29
	rbTexHeader  = 95
	rbTexMaxSide = 16384
)

type RainbowTexHeader struct {
	Width     int
	Height    int
	Format    int
	Levels    int
	SrcWidth  int
	SrcHeight int
	DataLen   int
}

func IsRainbowTex(raw []byte) bool {
	return len(raw) >= rbTexHeader && binary.LittleEndian.Uint32(raw[8:12]) == rbTexHash
}

func ParseRainbowTex(raw []byte) (RainbowTexHeader, error) {
	var h RainbowTexHeader
	if !IsRainbowTex(raw) {
		return h, fmt.Errorf("rainbow tex magic")
	}
	le := binary.LittleEndian
	h.Width = int(le.Uint32(raw[14:18]))
	h.Height = int(le.Uint32(raw[18:22]))
	h.DataLen = int(le.Uint32(raw[22:26]))
	h.Format = int(le.Uint32(raw[26:30]))
	h.Levels = int(le.Uint32(raw[30:34]))
	h.SrcWidth = int(le.Uint32(raw[71:75]))
	h.SrcHeight = int(le.Uint32(raw[75:79]))
	if h.Width <= 0 || h.Height <= 0 || h.Width > rbTexMaxSide || h.Height > rbTexMaxSide {
		return h, fmt.Errorf("rainbow tex size %dx%d", h.Width, h.Height)
	}
	if h.DataLen <= 0 || rbTexHeader+h.DataLen > len(raw) {
		return h, fmt.Errorf("rainbow tex data %d of %d", h.DataLen, len(raw))
	}
	if int(le.Uint32(raw[91:95])) != h.DataLen {
		return h, fmt.Errorf("rainbow tex data len mismatch")
	}
	return h, nil
}

func DecodeRainbowTex(raw []byte) (*image.NRGBA, error) {
	h, err := ParseRainbowTex(raw)
	if err != nil {
		return nil, err
	}
	if h.Format != rbTexFmtCRN {
		return nil, fmt.Errorf("rainbow tex fmt %d", h.Format)
	}
	img, err := crn.Decode(raw[rbTexHeader : rbTexHeader+h.DataLen])
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != h.Width || img.Bounds().Dy() != h.Height {
		return nil, fmt.Errorf("rainbow tex crn %dx%d want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), h.Width, h.Height)
	}
	return flipNRGBA(img), nil
}
