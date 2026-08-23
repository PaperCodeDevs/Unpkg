package crn

import (
	"fmt"
	"image"
)

type Info struct {
	Width  int
	Height int
	Levels int
	Faces  int
	Format uint32
}

func Probe(data []byte) (Info, error) {
	h, err := ParseHeader(data)
	if err != nil {
		return Info{}, err
	}
	return Info{
		Width:  int(h.Width),
		Height: int(h.Height),
		Levels: int(h.Levels),
		Faces:  int(h.Faces),
		Format: h.Format,
	}, nil
}

func newUnpacker(data []byte) (*unpacker, error) {
	h, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}
	if h.Faces < 1 || h.Faces > 6 {
		return nil, fmt.Errorf("crn: 暂不支持 %d 面纹理", h.Faces)
	}
	u := &unpacker{data: data[:h.DataSize], h: h}
	if err := u.initTables(); err != nil {
		return nil, err
	}
	if err := u.decodePalettes(); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *unpacker) level(i uint32) (*image.NRGBA, error) {
	blocks, w, h, err := u.unpackLevel(i)
	if err != nil {
		return nil, err
	}
	switch u.h.Format {
	case FmtDXT1:
		return decodeDXT1(blocks, int(w), int(h)), nil
	case FmtDXT5, 3, 4, 5, 6:
		return decodeDXT5(blocks, int(w), int(h)), nil
	}
	return nil, fmt.Errorf("crn: 不支持的格式 %d", u.h.Format)
}

func Decode(data []byte) (*image.NRGBA, error) {
	u, err := newUnpacker(data)
	if err != nil {
		return nil, err
	}
	return u.level(0)
}

func DecodeLevels(data []byte) ([]*image.NRGBA, error) {
	u, err := newUnpacker(data)
	if err != nil {
		return nil, err
	}
	out := make([]*image.NRGBA, 0, u.h.Levels)
	for i := uint32(0); i < u.h.Levels; i++ {
		img, err := u.level(i)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			break
		}
		out = append(out, img)
	}
	return out, nil
}
