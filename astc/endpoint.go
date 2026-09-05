package astc

import "fmt"

type rgba [4]int

func decodeEndpoints(cem int, v []int) (rgba, rgba, error) {
	switch cem {
	case 0:
		return rgba{v[0], v[0], v[0], 255}, rgba{v[1], v[1], v[1], 255}, nil
	case 1:
		l0 := v[0]>>2 | v[1]&0xC0
		l1 := min(l0+v[1]&0x3F, 255)
		return rgba{l0, l0, l0, 255}, rgba{l1, l1, l1, 255}, nil
	case 4:
		return rgba{v[0], v[0], v[0], v[2]}, rgba{v[1], v[1], v[1], v[3]}, nil
	case 5:
		bitTransfer(&v[1], &v[0])
		bitTransfer(&v[3], &v[2])
		l1, a1 := v[0]+v[1], v[2]+v[3]
		return clamp(rgba{v[0], v[0], v[0], v[2]}), clamp(rgba{l1, l1, l1, a1}), nil
	case 6:
		e0, e1 := scaledRGB(v, 255, 255)
		return e0, e1, nil
	case 8:
		e0, e1 := directRGB(v, 255, 255)
		return e0, e1, nil
	case 9:
		e0, e1 := offsetRGB(v, false)
		return e0, e1, nil
	case 10:
		e0, e1 := scaledRGB(v, v[4], v[5])
		return e0, e1, nil
	case 12:
		e0, e1 := directRGB(v, v[6], v[7])
		return e0, e1, nil
	case 13:
		e0, e1 := offsetRGB(v, true)
		return e0, e1, nil
	}
	return rgba{}, rgba{}, fmt.Errorf("HDR endpoint mode %d", cem)
}

func scaledRGB(v []int, a0, a1 int) (rgba, rgba) {
	e0 := rgba{v[0] * v[3] >> 8, v[1] * v[3] >> 8, v[2] * v[3] >> 8, a0}
	return e0, rgba{v[0], v[1], v[2], a1}
}

func directRGB(v []int, a0, a1 int) (rgba, rgba) {
	if v[1]+v[3]+v[5] >= v[0]+v[2]+v[4] {
		return rgba{v[0], v[2], v[4], a0}, rgba{v[1], v[3], v[5], a1}
	}
	return blueContract(rgba{v[1], v[3], v[5], a1}), blueContract(rgba{v[0], v[2], v[4], a0})
}

func offsetRGB(v []int, alpha bool) (rgba, rgba) {
	bitTransfer(&v[1], &v[0])
	bitTransfer(&v[3], &v[2])
	bitTransfer(&v[5], &v[4])
	a0, a1 := 255, 255
	if alpha {
		bitTransfer(&v[7], &v[6])
		a0, a1 = v[6], v[6]+v[7]
	}
	base := rgba{v[0], v[2], v[4], a0}
	sum := rgba{v[0] + v[1], v[2] + v[3], v[4] + v[5], a1}
	if v[1]+v[3]+v[5] >= 0 {
		return clamp(base), clamp(sum)
	}
	return clamp(blueContract(sum)), clamp(blueContract(base))
}

func bitTransfer(a, b *int) {
	*b >>= 1
	*b |= *a & 0x80
	*a >>= 1
	*a &= 0x3F
	if *a&0x20 != 0 {
		*a -= 0x40
	}
}

func blueContract(c rgba) rgba {
	return rgba{(c[0] + c[2]) >> 1, (c[1] + c[2]) >> 1, c[2], c[3]}
}

func clamp(c rgba) rgba {
	for i := range c {
		c[i] = min(max(c[i], 0), 255)
	}
	return c
}
