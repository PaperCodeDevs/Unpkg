package spine

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func ParseAtlas(raw []byte) (*Atlas, error) {
	a := &Atlas{}
	pi, ri := -1, -1
	for ln, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			pi, ri = -1, -1
			continue
		}
		colon := strings.Index(t, ":")
		if colon < 0 {
			if pi < 0 {
				a.Pages = append(a.Pages, Page{Name: t, Scale: 1})
				pi, ri = len(a.Pages)-1, -1
				continue
			}
			pg := &a.Pages[pi]
			pg.Regions = append(pg.Regions, Region{Page: pi, Name: t, Index: -1})
			ri = len(pg.Regions) - 1
			continue
		}
		if pi < 0 {
			return nil, fmt.Errorf("spine atlas: 第 %d 行属性无归属", ln+1)
		}
		key := strings.TrimSpace(t[:colon])
		vals := splitValues(t[colon+1:])
		var err error
		if ri >= 0 {
			err = a.Pages[pi].Regions[ri].set(key, vals)
		} else {
			err = a.Pages[pi].set(key, vals)
		}
		if err != nil {
			return nil, fmt.Errorf("spine atlas: 第 %d 行 %w", ln+1, err)
		}
	}
	if len(a.Pages) == 0 {
		return nil, fmt.Errorf("spine atlas: 无 page")
	}
	for i := range a.Pages {
		for j := range a.Pages[i].Regions {
			r := &a.Pages[i].Regions[j]
			if r.OrigWidth == 0 && r.OrigHeight == 0 {
				r.OrigWidth, r.OrigHeight = r.Width, r.Height
			}
		}
	}
	return a, nil
}

func splitValues(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func atoi(vals []string, i int) (int, error) {
	if i >= len(vals) {
		return 0, fmt.Errorf("缺少第 %d 个值", i+1)
	}
	return strconv.Atoi(vals[i])
}

func ints(vals []string, n int) ([]int, error) {
	out := make([]int, n)
	for i := range out {
		v, err := atoi(vals, i)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (p *Page) set(key string, vals []string) error {
	switch key {
	case "size":
		v, err := ints(vals, 2)
		if err != nil {
			return err
		}
		p.Width, p.Height = v[0], v[1]
	case "format":
		p.Format = vals[0]
	case "filter":
		p.MinFilter = vals[0]
		p.MagFilter = vals[len(vals)-1]
	case "repeat":
		p.Repeat = vals[0]
	case "pma":
		p.PMA = vals[0] == "true"
	case "scale":
		f, err := strconv.ParseFloat(vals[0], 32)
		if err != nil {
			return err
		}
		p.Scale = float32(f)
	}
	return nil
}

func (r *Region) set(key string, vals []string) error {
	var v []int
	var err error
	switch key {
	case "rotate":
		switch vals[0] {
		case "true":
			r.Degrees = 90
		case "false":
			r.Degrees = 0
		default:
			if r.Degrees, err = strconv.Atoi(vals[0]); err != nil {
				return err
			}
		}
		r.Rotate = r.Degrees == 90
	case "xy":
		if v, err = ints(vals, 2); err == nil {
			r.X, r.Y = v[0], v[1]
		}
	case "size":
		if v, err = ints(vals, 2); err == nil {
			r.Width, r.Height = v[0], v[1]
		}
	case "bounds":
		if v, err = ints(vals, 4); err == nil {
			r.X, r.Y, r.Width, r.Height = v[0], v[1], v[2], v[3]
		}
	case "orig":
		if v, err = ints(vals, 2); err == nil {
			r.OrigWidth, r.OrigHeight = v[0], v[1]
		}
	case "offset":
		if v, err = ints(vals, 2); err == nil {
			r.OffsetX, r.OffsetY = v[0], v[1]
		}
	case "offsets":
		if v, err = ints(vals, 4); err == nil {
			r.OffsetX, r.OffsetY, r.OrigWidth, r.OrigHeight = v[0], v[1], v[2], v[3]
		}
	case "index":
		r.Index, err = atoi(vals, 0)
	case "split":
		r.Splits, err = ints(vals, 4)
	case "pad":
		r.Pads, err = ints(vals, 4)
	}
	return err
}

func (r Region) PageRect() (x, y, w, h int) {
	if r.Rotate {
		return r.X, r.Y, r.Height, r.Width
	}
	return r.X, r.Y, r.Width, r.Height
}

func (r Region) ScaledRect(pageW, pageH, texW, texH int) (x, y, w, h int) {
	x, y, w, h = r.PageRect()
	if pageW <= 0 || pageH <= 0 || texW <= 0 || texH <= 0 || (pageW == texW && pageH == texH) {
		return x, y, w, h
	}
	sx := float64(texW) / float64(pageW)
	sy := float64(texH) / float64(pageH)
	x0 := int(math.Round(float64(x) * sx))
	y0 := int(math.Round(float64(y) * sy))
	x1 := int(math.Round(float64(x+w) * sx))
	y1 := int(math.Round(float64(y+h) * sy))
	return x0, y0, max(x1-x0, 1), max(y1-y0, 1)
}
