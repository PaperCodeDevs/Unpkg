package pkg

import (
	"fmt"
	"strings"
)

func (r *Reader) initBases() {
	r.bases = make(map[string]string)
	if r.idx == nil {
		return
	}
	for n, flag := range r.idx.byName {
		if flag&0x80000000 != 0 {
			continue
		}
		if !strings.HasPrefix(n, minigameBlocks) {
			continue
		}
		low := strings.ToLower(n)
		if !strings.HasSuffix(low, ".png") {
			continue
		}
		base := strings.TrimSuffix(low[len(minigameBlocks):], ".png")
		if _, ok := r.bases[base]; !ok {
			r.bases[base] = n
		}
	}
}

func (r *Reader) ResolveTex(key string) (string, []byte, error) {
	if r == nil || r.bases == nil {
		return "", nil, fmt.Errorf("nil reader")
	}
	base := strings.ToLower(strings.TrimSpace(key))
	if base == "" || base == "-" {
		return "", nil, fmt.Errorf("empty")
	}
	cands := []string{
		base, base + "_top", base + "_side", base + "_bottom",
		base + "_normal", base + "_still", base + "_s0", base + "_s1",
		base + "_0", base + "_1", base + "1", base + "_lower",
		base + "_empty", base + "_middle", base + "_full",
	}
	for _, suf := range []string{"_flow", "_still", "_normal"} {
		if strings.HasSuffix(base, suf) {
			cands = append(cands, strings.TrimSuffix(base, suf))
		}
	}
	for _, c := range cands {
		if name, ok := r.bases[c]; ok {
			b, err := r.Lookup(name)
			if err == nil {
				return name, b, nil
			}
		}
	}
	prefix := base + "_"
	var hit string
	n := 0
	for b, name := range r.bases {
		if !strings.HasPrefix(b, prefix) {
			continue
		}
		if strings.HasSuffix(b, "_mix") || strings.HasSuffix(b, "_emi") {
			continue
		}
		n++
		hit = name
		if n > 1 {
			break
		}
	}
	if n == 1 {
		b, err := r.Lookup(hit)
		if err == nil {
			return hit, b, nil
		}
	}
	return "", nil, fmt.Errorf("no png for %s", key)
}
