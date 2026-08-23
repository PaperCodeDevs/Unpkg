package pkg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
)

func cropFGUISprite(s *fguiSet, h fguiHit) ([]byte, error) {
	if s == nil || h.sp.file == "" {
		return nil, fmt.Errorf("cropFGUISprite: 空切片")
	}
	img, err := s.atlasImage(h.sp.file)
	if err != nil {
		return nil, err
	}
	sp := h.sp
	x, y, w, hgt := sp.x, sp.y, sp.w, sp.h
	if sp.rot {
		w, hgt = sp.h, sp.w
	}
	b := img.Bounds()
	if x < b.Min.X || y < b.Min.Y || x+w > b.Max.X || y+hgt > b.Max.Y || w <= 0 || hgt <= 0 {
		return nil, fmt.Errorf("cropFGUISprite: 矩形越界 %d,%d %dx%d atlas=%s", x, y, w, hgt, b)
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, hgt))
	draw.Draw(out, out.Bounds(), img, image.Pt(x, y), draw.Src)
	if sp.rot {
		out = rotateRight(out)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *fguiSet) atlasImage(name string) (*image.NRGBA, error) {
	if img, ok := s.atlases[name]; ok {
		return img, nil
	}
	raw, err := s.lookupFile(name)
	if err != nil {
		return nil, err
	}
	img, err := decodeAtlasImage(raw)
	if err != nil {
		return nil, err
	}
	img = flipNRGBA(img)
	s.atlases[name] = img
	return img, nil
}

func decodeAtlasImage(raw []byte) (*image.NRGBA, error) {
	if img, err := DecodeTextureImage(raw); err == nil {
		return img, nil
	}
	im, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := im.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), im, b.Min, draw.Src)
	return out, nil
}

func flipNRGBA(src *image.NRGBA) *image.NRGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(out.Pix[y*out.Stride:(y+1)*out.Stride], src.Pix[(h-1-y)*src.Stride:(h-y)*src.Stride])
	}
	return out
}

func rotateRight(src *image.NRGBA) *image.NRGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(h-1-y, x, src.NRGBAAt(x, y))
		}
	}
	return out
}
