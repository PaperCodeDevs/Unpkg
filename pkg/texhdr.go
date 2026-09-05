package pkg

import (
	"encoding/binary"
	"fmt"
	"image"
	"math"
)

const (
	texFacesOff   = 44
	texFmtRHalf   = 15
	texFmtRGHalf  = 16
	texFmtRGBAH   = 17
	texFmtRFloat  = 18
	texFmtRGFloat = 19
	texFmtRGBAF   = 20
	texFmtRGB9e5  = 22
	texMaxFaces   = 6
)

func hdrPixelBytes(format uint32) (int, int) {
	switch format {
	case texFmtRHalf:
		return 2, 1
	case texFmtRGHalf:
		return 4, 2
	case texFmtRGBAH:
		return 8, 4
	case texFmtRFloat:
		return 4, 1
	case texFmtRGFloat:
		return 8, 2
	case texFmtRGBAF:
		return 16, 4
	case texFmtRGB9e5:
		return 4, 3
	}
	return 0, 0
}

func IsHDRFormat(format uint32) bool {
	n, _ := hdrPixelBytes(format)
	return n > 0
}

// 立方图 6 面按面拼成横条，每面正文 = DataSize 字节（含 mip 链），只取 mip0
func DecodeTextureHDRImage(container []byte) (*image.NRGBA, error) {
	h, err := ParseTexHeader(container)
	if err != nil {
		return nil, err
	}
	bpp, nch := hdrPixelBytes(h.Format)
	if bpp == 0 {
		return nil, fmt.Errorf("hdr fmt %d", h.Format)
	}
	faces := 1
	if len(container) >= texFacesOff+4 {
		if f := int(binary.LittleEndian.Uint32(container[texFacesOff:])); f > 1 && f <= texMaxFaces {
			faces = f
		}
	}
	face := int(h.DataSize)
	mip0 := h.Width * h.Height * bpp
	if face < mip0 {
		face = mip0
	}
	if texPayload+(faces-1)*face+mip0 > len(container) {
		return nil, fmt.Errorf("hdr short %d", len(container))
	}
	img := image.NewNRGBA(image.Rect(0, 0, h.Width*faces, h.Height))
	for f := 0; f < faces; f++ {
		pix := container[texPayload+f*face:]
		for y := 0; y < h.Height; y++ {
			for x := 0; x < h.Width; x++ {
				r, g, b, a := hdrPixel(pix[(y*h.Width+x)*bpp:], h.Format, nch)
				o := img.PixOffset(f*h.Width+x, y)
				img.Pix[o+0] = toSRGB8(r)
				img.Pix[o+1] = toSRGB8(g)
				img.Pix[o+2] = toSRGB8(b)
				img.Pix[o+3] = toLinear8(a)
			}
		}
	}
	return flipNRGBA(img), nil
}

func hdrPixel(p []byte, format uint32, nch int) (r, g, b, a float32) {
	var c [4]float32
	c[3] = 1
	switch format {
	case texFmtRHalf, texFmtRGHalf, texFmtRGBAH:
		for i := 0; i < nch; i++ {
			c[i] = halfToFloat(binary.LittleEndian.Uint16(p[i*2:]))
		}
	case texFmtRFloat, texFmtRGFloat, texFmtRGBAF:
		for i := 0; i < nch; i++ {
			c[i] = math.Float32frombits(binary.LittleEndian.Uint32(p[i*4:]))
		}
	case texFmtRGB9e5:
		c[0], c[1], c[2] = rgb9e5(binary.LittleEndian.Uint32(p))
	}
	if nch == 1 {
		c[1], c[2] = c[0], c[0]
	}
	return c[0], c[1], c[2], c[3]
}

func halfToFloat(h uint16) float32 {
	s := uint32(h>>15) & 1
	e := uint32(h>>10) & 0x1f
	m := uint32(h) & 0x3ff
	var bits uint32
	switch {
	case e == 0 && m == 0:
		bits = s << 31
	case e == 0:
		for m&0x400 == 0 {
			m <<= 1
			e--
		}
		e++
		m &= 0x3ff
		bits = s<<31 | (e+112)<<23 | m<<13
	case e == 0x1f:
		bits = s<<31 | 0xff<<23 | m<<13
	default:
		bits = s<<31 | (e+112)<<23 | m<<13
	}
	return math.Float32frombits(bits)
}

func rgb9e5(v uint32) (float32, float32, float32) {
	e := int(v>>27) - 24
	scale := float32(math.Ldexp(1, e))
	return float32(v&0x1ff) * scale, float32((v>>9)&0x1ff) * scale, float32((v>>18)&0x1ff) * scale
}

func toSRGB8(v float32) uint8 {
	if v != v || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	var s float64
	if float64(v) <= 0.0031308 {
		s = 12.92 * float64(v)
	} else {
		s = 1.055*math.Pow(float64(v), 1/2.4) - 0.055
	}
	return uint8(math.Round(s * 255))
}

func toLinear8(v float32) uint8 {
	if v != v || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(math.Round(float64(v) * 255))
}
