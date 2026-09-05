package astc

func decodeWeights(blk []byte, m blockMode, blockW, blockH int) ([]int, []int) {
	count := m.gridW * m.gridH
	n := count
	if m.dual {
		n *= 2
	}
	raw := make([]int, n)
	decodeISE(reverseBlock(blk), 0, n, m.quant, raw)
	grid0 := make([]int, count)
	var grid1 []int
	if m.dual {
		grid1 = make([]int, count)
		for i := 0; i < count; i++ {
			grid0[i] = unquantWeight(raw[2*i], m.quant)
			grid1[i] = unquantWeight(raw[2*i+1], m.quant)
		}
	} else {
		for i := 0; i < count; i++ {
			grid0[i] = unquantWeight(raw[i], m.quant)
		}
	}
	plane0 := infill(grid0, m.gridW, m.gridH, blockW, blockH)
	var plane1 []int
	if m.dual {
		plane1 = infill(grid1, m.gridW, m.gridH, blockW, blockH)
	}
	return plane0, plane1
}

func infill(grid []int, gw, gh, bw, bh int) []int {
	out := make([]int, bw*bh)
	ds := (1024 + bw/2) / (bw - 1)
	dt := (1024 + bh/2) / (bh - 1)
	at := func(i int) int {
		if i < 0 || i >= len(grid) {
			return 0
		}
		return grid[i]
	}
	for t := 0; t < bh; t++ {
		gt := (dt*t*(gh-1) + 32) >> 6
		jt, ft := gt>>4, gt&0xF
		for s := 0; s < bw; s++ {
			gs := (ds*s*(gw-1) + 32) >> 6
			js, fs := gs>>4, gs&0xF
			v0 := js + jt*gw
			w11 := (fs*ft + 8) >> 4
			w10 := ft - w11
			w01 := fs - w11
			w00 := 16 - fs - ft + w11
			out[t*bw+s] = (at(v0)*w00 + at(v0+1)*w01 + at(v0+gw)*w10 + at(v0+gw+1)*w11 + 8) >> 4
		}
	}
	return out
}
