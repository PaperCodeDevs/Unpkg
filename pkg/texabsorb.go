package pkg

import (
	"fmt"
	"strings"
)

func (r *Reader) AbsorbBases(extra *Reader) {
	if r == nil || extra == nil || r.bases == nil || extra.bases == nil {
		return
	}
	for b, n := range extra.bases {
		r.bases[b] = n
	}
	r.alt = extra
}

func (r *Reader) LookupTex(name string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	if r.alt != nil {
		if b, err := r.alt.Lookup(name); err == nil {
			return b, nil
		}
	}
	return r.Lookup(name)
}

func (r *Reader) HasBase(base string) bool {
	if r == nil || r.bases == nil {
		return false
	}
	_, ok := r.bases[base]
	return ok
}

func (r *Reader) BaseNames() map[string]bool {
	out := map[string]bool{}
	if r == nil || r.bases == nil {
		return out
	}
	for b := range r.bases {
		out[b] = true
	}
	return out
}

func TexStemName(key string) string {
	return texStem(strings.ToLower(strings.TrimSpace(key)))
}
