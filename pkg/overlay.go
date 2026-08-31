package pkg

import (
	"fmt"
	"strings"
)

type overlayPair struct {
	base  *Reader
	patch *Reader
}

func OpenOverlay(base, patch *Pkg) (*Reader, error) {
	if base == nil {
		return nil, fmt.Errorf("nil pkg")
	}
	br, err := OpenReader(base)
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return br, nil
	}
	pr, err := OpenReader(patch)
	if err != nil {
		return nil, err
	}
	r := &Reader{over: &overlayPair{base: br, patch: pr}, bases: map[string]string{}, lower: map[string]string{}}
	mergeBases(r, br)
	mergeBases(r, pr)
	return r, nil
}

func mergeBases(dst, src *Reader) {
	if dst == nil || src == nil {
		return
	}
	if dst.bases == nil {
		dst.bases = map[string]string{}
	}
	for k, v := range src.bases {
		dst.bases[k] = v
	}
	if dst.lower == nil {
		dst.lower = map[string]string{}
	}
	for k, v := range src.lower {
		if _, ok := dst.lower[k]; !ok {
			dst.lower[k] = v
		}
	}
}

func OpenOverlayFiles(basePath, patchPath string) (*Reader, error) {
	bp, err := ParseFile(basePath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(patchPath) == "" {
		return OpenReader(bp)
	}
	pp, err := ParseFile(patchPath)
	if err != nil {
		return nil, err
	}
	return OpenOverlay(bp, pp)
}

func (o *overlayPair) lookup(name string) ([]byte, error) {
	if o == nil {
		return nil, fmt.Errorf("nil reader")
	}
	if o.patch != nil {
		b, err := o.patch.Lookup(name)
		if err == nil || o.base == nil || !overlayMiss(err) {
			return b, err
		}
	}
	if o.base != nil {
		return o.base.Lookup(name)
	}
	return nil, fmt.Errorf("not found: %s", name)
}

func overlayMiss(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "not found:")
}

func (o *overlayPair) names(prefix string) []string {
	if o == nil {
		return nil
	}
	seen := make(map[string]string)
	add := func(r *Reader) {
		if r == nil {
			return
		}
		for _, n := range r.Names(prefix) {
			seen[strings.ToLower(n)] = n
		}
	}
	add(o.base)
	add(o.patch)
	out := make([]string, 0, len(seen))
	for _, n := range seen {
		out = append(out, n)
	}
	return out
}
