package pkg

import (
	"fmt"
	"image"

	"github.com/PaperCodeDevs/Unpkg/crn"
)

func rawAlpha(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h {
		return nil, fmt.Errorf("a8 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+3] = pix[i]
	}
	return img, nil
}

func rawARGB(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*4 {
		return nil, fmt.Errorf("argb short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		o := i * 4
		img.Pix[o+0] = pix[o+1]
		img.Pix[o+1] = pix[o+2]
		img.Pix[o+2] = pix[o+3]
		img.Pix[o+3] = pix[o+0]
	}
	return img, nil
}

func rawBGRA(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*4 {
		return nil, fmt.Errorf("bgra short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		o := i * 4
		img.Pix[o+0] = pix[o+2]
		img.Pix[o+1] = pix[o+1]
		img.Pix[o+2] = pix[o+0]
		img.Pix[o+3] = pix[o+3]
	}
	return img, nil
}

func rawRGB565(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*2 {
		return nil, fmt.Errorf("rgb565 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		v := uint16(pix[i*2]) | uint16(pix[i*2+1])<<8
		r := uint8((v>>11)&31)<<3 | uint8((v>>11)&31)>>2
		g := uint8((v>>5)&63)<<2 | uint8((v>>5)&63)>>4
		b := uint8(v&31)<<3 | uint8(v&31)>>2
		o := i * 4
		img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, 255
	}
	return img, nil
}

func rawARGB4444(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*2 {
		return nil, fmt.Errorf("argb4444 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		v := uint16(pix[i*2]) | uint16(pix[i*2+1])<<8
		a := uint8((v>>12)&15) * 17
		r := uint8((v>>8)&15) * 17
		g := uint8((v>>4)&15) * 17
		b := uint8(v&15) * 17
		o := i * 4
		img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, a
	}
	return img, nil
}

func rawRGBA4444(pix []byte, w, h int) (*image.NRGBA, error) {
	if len(pix) < w*h*2 {
		return nil, fmt.Errorf("rgba4444 short")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		v := uint16(pix[i*2]) | uint16(pix[i*2+1])<<8
		r := uint8((v>>12)&15) * 17
		g := uint8((v>>8)&15) * 17
		b := uint8((v>>4)&15) * 17
		a := uint8(v&15) * 17
		o := i * 4
		img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, a
	}
	return img, nil
}

func rawDXT(pix []byte, w, h, block int) (*image.NRGBA, error) {
	bw, bh := (w+3)/4, (h+3)/4
	need := bw * bh * block
	if len(pix) < need {
		return nil, fmt.Errorf("dxt short")
	}
	if block == 8 {
		return crn.DecodeBC1(pix[:need], w, h), nil
	}
	return crn.DecodeBC3(pix[:need], w, h), nil
}
