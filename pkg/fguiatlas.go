package pkg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
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
	if aw, ah := h.atlasSize(); aw > 0 && ah > 0 && (aw != b.Dx() || ah != b.Dy()) {
		x, y, w, hgt = scaleRect(x, y, w, hgt, float64(b.Dx())/float64(aw), float64(b.Dy())/float64(ah))
	}
	cx, cy, cw, ch := clampRect(x, y, w, hgt, b.Dx(), b.Dy())
	if sp.w <= 0 || sp.h <= 0 || cw <= 0 || ch <= 0 || cx-x > 1 || cy-y > 1 || x+w-cx-cw > 1 || y+hgt-cy-ch > 1 {
		return nil, fmt.Errorf("cropFGUISprite: 矩形越界 %d,%d %dx%d atlas=%s", sp.x, sp.y, sp.w, sp.h, b)
	}
	out := image.NewNRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(out, out.Bounds(), img, image.Pt(cx, cy), draw.Src)
	if sp.rot {
		out = rotateRight(out)
	}
	if out.Bounds().Dx() != sp.w || out.Bounds().Dy() != sp.h {
		out = resizeNRGBA(out, sp.w, sp.h)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h fguiHit) atlasSize() (int, int) {
	if h.pkg == nil {
		return 0, 0
	}
	at, ok := h.pkg.byID[h.sp.atlas]
	if !ok || at.typ != fguiTypeAtlas {
		return 0, 0
	}
	return at.w, at.h
}

func scaleRect(x, y, w, h int, sx, sy float64) (int, int, int, int) {
	x0 := int(math.Round(float64(x) * sx))
	y0 := int(math.Round(float64(y) * sy))
	x1 := int(math.Round(float64(x+w) * sx))
	y1 := int(math.Round(float64(y+h) * sy))
	return x0, y0, x1 - x0, y1 - y0
}

func clampRect(x, y, w, h, maxW, maxH int) (int, int, int, int) {
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > maxW {
		w = maxW - x
	}
	if y+h > maxH {
		h = maxH - y
	}
	return x, y, w, h
}

func resizeNRGBA(src *image.NRGBA, w, h int) *image.NRGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	if sw <= 0 || sh <= 0 || w <= 0 || h <= 0 {
		return out
	}
	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(h) - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		y1 := min(y0+1, sh-1)
		y0 = max(y0, 0)
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			x1 := min(x0+1, sw-1)
			x0 = max(x0, 0)
			o := out.PixOffset(x, y)
			p00, p10 := src.PixOffset(x0, y0), src.PixOffset(x1, y0)
			p01, p11 := src.PixOffset(x0, y1), src.PixOffset(x1, y1)
			for c := 0; c < 4; c++ {
				top := float64(src.Pix[p00+c])*(1-tx) + float64(src.Pix[p10+c])*tx
				bottom := float64(src.Pix[p01+c])*(1-tx) + float64(src.Pix[p11+c])*tx
				out.Pix[o+c] = uint8(math.Round(top*(1-ty) + bottom*ty))
			}
		}
	}
	return out
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
	return flipNRGBA(out), nil
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
