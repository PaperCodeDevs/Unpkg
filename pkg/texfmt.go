package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"

	"github.com/PaperCodeDevs/Unpkg/crn"
)

const (
	texMagic   = 0x59A21C2C
	texPayload = 108
)

func TextureCRN(container []byte) ([]byte, error) {
	if len(container) < 110 {
		return nil, fmt.Errorf("container short %d", len(container))
	}
	if binary.LittleEndian.Uint32(container[0:4]) != 2 {
		return nil, fmt.Errorf("ver %d", binary.LittleEndian.Uint32(container[0:4]))
	}
	if binary.LittleEndian.Uint32(container[4:8]) != texMagic {
		return nil, fmt.Errorf("magic")
	}
	lim := 128
	if lim > len(container) {
		lim = len(container)
	}
	for i := 100; i+2 < lim; i++ {
		if container[i] == 0x48 && container[i+1] == 0x78 {
			return container[i:], nil
		}
	}
	return nil, fmt.Errorf("no crn")
}

func DecodeTextureImage(container []byte) (*image.NRGBA, error) {
	if src, err := TextureCRN(container); err == nil {
		if img, err := crn.Decode(src); err == nil {
			return flipNRGBA(img), nil
		}
	}
	img, err := decodeRawTex(container)
	if err != nil {
		return nil, err
	}
	return flipNRGBA(img), nil
}

func flipNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(out.Pix[y*out.Stride:(y+1)*out.Stride], src.Pix[(h-1-y)*src.Stride:(h-y)*src.Stride])
	}
	return out
}

func DecodeTexturePNG(container []byte) ([]byte, error) {
	img, err := DecodeTextureImage(container)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeTextureDXT(container []byte) ([]byte, int, int, error) {
	src, err := TextureCRN(container)
	if err != nil {
		return nil, 0, 0, err
	}
	dxt, w, h, err := crn.DebugBlocks(src, 0)
	return dxt, w, h, err
}

func decodeRawTex(b []byte) (*image.NRGBA, error) {
	if len(b) < texPayload+1 {
		return nil, fmt.Errorf("container short %d", len(b))
	}
	if binary.LittleEndian.Uint32(b[0:4]) != 2 || binary.LittleEndian.Uint32(b[4:8]) != texMagic {
		return nil, fmt.Errorf("magic")
	}
	w := int(binary.LittleEndian.Uint32(b[20:24]))
	h := int(binary.LittleEndian.Uint32(b[24:28]))
	format := binary.LittleEndian.Uint32(b[32:36])
	if w <= 0 || h <= 0 || w > 16384 || h > 16384 {
		return nil, fmt.Errorf("size %dx%d", w, h)
	}
	pix := b[texPayload:]
	switch format {
	case 1:
		return rawAlpha(pix, w, h)
	case 2:
		return rawARGB4444(pix, w, h)
	case 3:
		return rawColor(pix, w, h)
	case 4:
		return rawRGBA(pix, w, h)
	case 5:
		return rawARGB(pix, w, h)
	case 7:
		return rawRGB565(pix, w, h)
	case 10:
		return rawDXT(pix, w, h, 8)
	case 12:
		return rawDXT(pix, w, h, 16)
	case 13:
		return rawRGBA4444(pix, w, h)
	case 14:
		return rawBGRA(pix, w, h)
	case 25:
		return crn.DecodeBC7(pix, w, h)
	case 63:
		return rawGray(pix, w, h)
	default:
		return nil, fmt.Errorf("raw fmt %d", format)
	}
}

func rawColor(pix []byte, w, h int) (*image.NRGBA, error) {
	need3 := w * h * 3
	if len(pix) < need3 {
		return nil, fmt.Errorf("rgb short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = pix[i*3+0]
		img.Pix[i*4+1] = pix[i*3+1]
		img.Pix[i*4+2] = pix[i*3+2]
		img.Pix[i*4+3] = 255
	}
	return img, nil
}

func rawRGBA(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*4 {
		return nil, fmt.Errorf("rgba short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, pix[:w*h*4])
	return img, nil
}

func rawGray(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h {
		return nil, fmt.Errorf("r8 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		v := pix[i]
		img.Pix[i*4+0] = v
		img.Pix[i*4+1] = v
		img.Pix[i*4+2] = v
		img.Pix[i*4+3] = 255
	}
	return img, nil
}
