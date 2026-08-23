package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseObjMesh(raw []byte) (*BlockMesh, error) {
	lines := strings.Split(string(raw), "\n")
	var vs [][3]float32
	var vts [][2]float32
	var pos []float32
	var uv []float32
	var idx []uint32
	for _, ln := range lines {
		ln = strings.TrimSpace(strings.TrimRight(ln, "\r"))
		if ln == "" || ln[0] == '#' {
			continue
		}
		fs := strings.Fields(ln)
		if len(fs) < 2 {
			continue
		}
		switch fs[0] {
		case "v":
			if len(fs) < 4 {
				continue
			}
			x, y, z, err := threeFloat(fs)
			if err != nil {
				return nil, err
			}
			vs = append(vs, [3]float32{x / meshUnit, y / meshUnit, z / meshUnit})
		case "vt":
			if len(fs) < 2 {
				continue
			}
			u, err := parseF32(fs[1])
			if err != nil {
				return nil, err
			}
			v := float32(0)
			if len(fs) > 2 {
				v, err = parseF32(fs[2])
				if err != nil {
					return nil, err
				}
			}
			vts = append(vts, [2]float32{u, v})
		case "f":
			if len(fs) < 4 {
				continue
			}
			type corner struct{ vi, ti int }
			cs := make([]corner, 0, len(fs)-1)
			for _, tok := range fs[1:] {
				c, err := parseCorner(tok, len(vs), len(vts))
				if err != nil {
					return nil, err
				}
				cs = append(cs, c)
			}
			if len(cs) < 3 {
				continue
			}
			for i := 1; i+1 < len(cs); i++ {
				tri := [3]corner{cs[0], cs[i], cs[i+1]}
				for _, c := range tri {
					p := vs[c.vi]
					pos = append(pos, p[0], p[1], p[2])
					if c.ti >= 0 && c.ti < len(vts) {
						uv = append(uv, vts[c.ti][0], vts[c.ti][1])
					} else {
						uv = append(uv, 0, 0)
					}
					idx = append(idx, uint32(len(pos)/3-1))
				}
			}
		}
	}
	if len(pos) < 9 || len(idx) < 3 {
		return nil, fmt.Errorf("obj empty")
	}
	return &BlockMesh{Pos: pos, UV: uv, Idx: idx}, nil
}

func threeFloat(fs []string) (float32, float32, float32, error) {
	x, err := parseF32(fs[1])
	if err != nil {
		return 0, 0, 0, err
	}
	y, err := parseF32(fs[2])
	if err != nil {
		return 0, 0, 0, err
	}
	z, err := parseF32(fs[3])
	if err != nil {
		return 0, 0, 0, err
	}
	return x, y, z, nil
}

func parseF32(s string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)
	return float32(f), err
}

func parseCorner(tok string, nv, nt int) (struct{ vi, ti int }, error) {
	var z struct{ vi, ti int }
	z.ti = -1
	parts := strings.Split(tok, "/")
	vi, err := strconv.Atoi(parts[0])
	if err != nil {
		return z, err
	}
	z.vi = objIndex(vi, nv)
	if z.vi < 0 || z.vi >= nv {
		return z, fmt.Errorf("obj v %d", vi)
	}
	if len(parts) > 1 && parts[1] != "" {
		ti, err := strconv.Atoi(parts[1])
		if err != nil {
			return z, err
		}
		z.ti = objIndex(ti, nt)
	}
	return z, nil
}

func objIndex(i, n int) int {
	if i < 0 {
		return n + i
	}
	return i - 1
}
