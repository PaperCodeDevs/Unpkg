package pkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FGUIItemInfo struct {
	Type int
	ID   string
	Name string
	File string
	W    int
	H    int
}

type FGUISpriteInfo struct {
	ItemID string
	Atlas  string
	File   string
	X      int
	Y      int
	W      int
	H      int
	Rot    bool
}

type FGUIPackInfo struct {
	Name    string
	ID      string
	Asset   string
	Source  string
	Version int
	Items   []FGUIItemInfo
	Sprites []FGUISpriteInfo
}

var OfficialFGUINeed = []struct {
	Pack string
	Comp string
}{
	{Pack: "login", Comp: "login_homepage"},
	{Pack: "login", Comp: "login_miniaccount"},
	{Pack: "login", Comp: "login_captcha"},
	{Pack: "c_login", Comp: ""},
	{Pack: "commonWidgetPic", Comp: ""},
	{Pack: "common", Comp: ""},
	{Pack: "commonIconPic", Comp: ""},
	{Pack: "common_comp", Comp: ""},
	{Pack: "c_AccountMgr", Comp: ""},
	{Pack: "AccountMgr", Comp: "acc_mgr_bf"},
	{Pack: "homepage_v2", Comp: "HomePage"},
	{Pack: "c_hpm", Comp: ""},
	{Pack: "SimpleLobbyChatView", Comp: ""},
}

func toFGUIPack(p *fguiPkg, src string) FGUIPackInfo {
	info := FGUIPackInfo{Name: p.name, ID: p.id, Asset: p.asset, Source: src, Version: p.ver}
	for _, it := range p.items {
		info.Items = append(info.Items, FGUIItemInfo{
			Type: it.typ, ID: it.id, Name: it.name, File: it.file, W: it.w, H: it.h,
		})
		if sp, ok := p.sprite[it.id]; ok {
			info.Sprites = append(info.Sprites, FGUISpriteInfo{
				ItemID: it.id, Atlas: sp.atlas, File: sp.file, X: sp.x, Y: sp.y, W: sp.w, H: sp.h, Rot: sp.rot,
			})
		}
	}
	return info
}

func ParseFGUIPack(raw []byte, asset string) (FGUIPackInfo, error) {
	p, err := parseFGUI(raw, fguiAssetPath(asset))
	if err != nil {
		return FGUIPackInfo{}, err
	}
	if p.name == "" {
		return FGUIPackInfo{}, fmt.Errorf("ParseFGUIPack: 无包名")
	}
	return toFGUIPack(p, ""), nil
}

func ListFGUI(pkgPaths []string) ([]FGUIPackInfo, error) {
	rs, err := openPkgReaders(pkgPaths)
	if err != nil {
		return nil, err
	}
	byKey := map[string]*FGUIPackInfo{}
	order := []string{}
	for si, r := range rs {
		src := ""
		if si < len(pkgPaths) {
			src = filepath.Base(pkgPaths[si])
		}
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
			info := toFGUIPack(p, src)
			key := strings.ToLower(p.name)
			if _, ok := byKey[key]; !ok {
				order = append(order, key)
			}
			byKey[key] = &info
		}
	}
	if len(byKey) == 0 {
		return nil, fmt.Errorf("ListFGUI: 无 FGUI 包")
	}
	out := make([]FGUIPackInfo, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (p FGUIPackInfo) ComponentNames() []string {
	var out []string
	seen := map[string]bool{}
	for _, it := range p.Items {
		if it.Type != fguiTypeComponent || it.Name == "" || seen[it.Name] {
			continue
		}
		seen[it.Name] = true
		out = append(out, it.Name)
	}
	sort.Strings(out)
	return out
}

func FindFGUIPack(list []FGUIPackInfo, name string) (FGUIPackInfo, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, p := range list {
		if strings.ToLower(p.Name) == want {
			return p, true
		}
	}
	return FGUIPackInfo{}, false
}

func HasFGUIComponent(p FGUIPackInfo, name string) bool {
	want := strings.TrimSpace(name)
	if want == "" {
		return true
	}
	for _, it := range p.Items {
		if it.Type == fguiTypeComponent && it.Name == want {
			return true
		}
	}
	return false
}

func CheckOfficialFGUI(list []FGUIPackInfo) []string {
	var miss []string
	for _, need := range OfficialFGUINeed {
		p, ok := FindFGUIPack(list, need.Pack)
		if !ok {
			miss = append(miss, "pack "+need.Pack)
			continue
		}
		if need.Comp != "" && !HasFGUIComponent(p, need.Comp) {
			miss = append(miss, need.Pack+"/"+need.Comp)
		}
	}
	return miss
}

func CheckShippedFGUI(dir string, list []FGUIPackInfo) []string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return []string{"ship dir"}
	}
	var miss []string
	seen := map[string]bool{}
	for _, need := range OfficialFGUINeed {
		if seen[need.Pack] {
			continue
		}
		seen[need.Pack] = true
		fui := filepath.Join(dir, need.Pack, need.Pack+".fui")
		st, err := os.Stat(fui)
		if err != nil || st.Size() <= 0 {
			miss = append(miss, "ship "+need.Pack+".fui")
			continue
		}
		p, ok := FindFGUIPack(list, need.Pack)
		if !ok {
			continue
		}
		raw, err := os.ReadFile(fui)
		if err != nil {
			miss = append(miss, "read "+need.Pack+".fui")
			continue
		}
		got, err := parseFGUI(raw, p.Asset)
		if err != nil || got == nil {
			miss = append(miss, "parse "+need.Pack+".fui")
			continue
		}
		info := toFGUIPack(got, "")
		if need.Comp != "" && !HasFGUIComponent(info, need.Comp) {
			miss = append(miss, "ship "+need.Pack+"/"+need.Comp)
		}
	}
	return miss
}

func WriteOfficialFGUIDump(w io.Writer, list []FGUIPackInfo) error {
	if w == nil {
		return fmt.Errorf("WriteOfficialFGUIDump: 无输出")
	}
	seen := map[string]bool{}
	for _, need := range OfficialFGUINeed {
		key := strings.ToLower(need.Pack)
		if seen[key] {
			continue
		}
		seen[key] = true
		p, ok := FindFGUIPack(list, need.Pack)
		if !ok {
			if _, err := fmt.Fprintf(w, "MISS pack %s\n", need.Pack); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "PACK %s id=%s src=%s items=%d\n", p.Name, p.ID, p.Source, len(p.Items)); err != nil {
			return err
		}
		for _, name := range p.ComponentNames() {
			if _, err := fmt.Fprintf(w, "  COMP %s %s\n", p.Name, name); err != nil {
				return err
			}
		}
	}
	return nil
}
