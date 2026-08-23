package pkg

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
)

var fguiPackAlias = map[string]string{
	"common3":         "common",
	"ugc_common3":     "common",
	"common3common5":  "common",
	"common3common5 ": "common",
}

type fguiSet struct {
	packs   map[string]*fguiPkg
	byRes   map[string]fguiHit
	files   []*Reader
	atlases map[string]*image.NRGBA
}

type fguiHit struct {
	pack string
	item fguiItem
	sp   fguiSprite
}

func loadFGUISet(rs []*Reader) (*fguiSet, error) {
	s := &fguiSet{packs: map[string]*fguiPkg{}, byRes: map[string]fguiHit{}, files: rs, atlases: map[string]*image.NRGBA{}}
	if len(rs) == 0 {
		return nil, fmt.Errorf("loadFGUISet: 无 pkg")
	}
	for _, r := range rs {
		for _, name := range r.Names("") {
			if !strings.HasSuffix(strings.ToLower(name), ".fui") {
				continue
			}
			raw, err := r.Lookup(name)
			if err != nil {
				continue
			}
			p, err := parseFGUI(raw, fguiAssetPath(name))
			if err != nil || p == nil || p.name == "" {
				continue
			}
			s.packs[p.name] = p
			s.packs[strings.ToLower(filepath.Base(fguiAssetPath(name)))] = p
			for _, it := range p.items {
				if it.typ != fguiTypeImage || it.name == "" {
					continue
				}
				sp, ok := p.sprite[it.id]
				if !ok || sp.file == "" {
					continue
				}
				if _, exists := s.byRes[it.name]; exists {
					continue
				}
				s.byRes[it.name] = fguiHit{pack: p.name, item: it, sp: sp}
			}
		}
	}
	if len(s.packs) == 0 {
		return nil, fmt.Errorf("loadFGUISet: 无 FGUI 包")
	}
	return s, nil
}

func (s *fguiSet) lookup(pack, res string) (fguiHit, bool) {
	if s == nil || res == "" {
		return fguiHit{}, false
	}
	for _, name := range packCandidates(pack) {
		p := s.packs[name]
		if p == nil {
			continue
		}
		it, ok := p.byName[res]
		if !ok {
			continue
		}
		sp, ok := p.sprite[it.id]
		if !ok || sp.file == "" {
			continue
		}
		return fguiHit{pack: p.name, item: it, sp: sp}, true
	}
	h, ok := s.byRes[res]
	return h, ok
}

func packCandidates(pack string) []string {
	pack = strings.TrimSpace(pack)
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	if a, ok := fguiPackAlias[pack]; ok {
		add(a)
	}
	add(pack)
	if i := strings.Index(pack, "common3"); i >= 0 && pack != "common3" {
		add("common")
	}
	return out
}

func (s *fguiSet) lookupFile(name string) ([]byte, error) {
	var last error
	for i := len(s.files) - 1; i >= 0; i-- {
		b, err := s.files[i].Lookup(name)
		if err == nil {
			return b, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("not found: %s", name)
	}
	return nil, last
}

func OpenReaders(paths []string) ([]*Reader, error) {
	return openPkgReaders(paths)
}

type FGUIImages struct {
	set *fguiSet
}

func NewFGUIImages(rs []*Reader) (*FGUIImages, error) {
	s, err := loadFGUISet(rs)
	if err != nil {
		return nil, err
	}
	return &FGUIImages{set: s}, nil
}

func (g *FGUIImages) Crop(pack, res string) ([]byte, error) {
	if g == nil || g.set == nil {
		return nil, fmt.Errorf("Crop: 空图集")
	}
	hit, ok := g.set.lookup(pack, res)
	if !ok {
		return nil, fmt.Errorf("Crop: missing")
	}
	return cropFGUISprite(g.set, hit)
}

func (g *FGUIImages) Size(pack, res string) (w, h int, ok bool) {
	if g == nil || g.set == nil {
		return 0, 0, false
	}
	hit, ok := g.set.lookup(pack, res)
	if !ok {
		return 0, 0, false
	}
	return hit.sp.w, hit.sp.h, true
}

func openPkgReaders(paths []string) ([]*Reader, error) {
	var out []*Reader
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		p, err := ParseFile(path)
		if err != nil {
			return nil, err
		}
		r, err := OpenReader(p)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("openPkgReaders: 无可用 pkg")
	}
	return out, nil
}
