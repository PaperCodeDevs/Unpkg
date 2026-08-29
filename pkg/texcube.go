package pkg

import (
	"sort"
	"strings"
)

var cubeStemSuf = []string{
	"_top", "_bottom", "_side", "_up", "_down",
	"_north", "_south", "_east", "_west",
	"_front", "_back", "_left", "_right",
}

func texStem(key string) string {
	for _, s := range cubeStemSuf {
		if strings.HasSuffix(key, s) {
			return strings.TrimSuffix(key, s)
		}
	}
	return key
}

func PkgStem(full string) string {
	base := full
	if i := strings.LastIndexByte(full, '/'); i >= 0 {
		base = full[i+1:]
	}
	if len(base) >= 4 && strings.EqualFold(base[len(base)-4:], ".png") {
		return base[:len(base)-4]
	}
	return base
}

func pickTex(have map[string]string, names ...string) string {
	for _, n := range names {
		if n == "" {
			continue
		}
		if full, ok := have[strings.ToLower(n)]; ok {
			return PkgStem(full)
		}
	}
	return ""
}

var junkVariantSuf = []string{
	"_emi", "_mix", "_proto", "_matcap", "_notactivated",
	"_x", "_y", "_z",
}

func junkVariant(name string) bool {
	for _, s := range junkVariantSuf {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func pickNumVariant(have map[string]string, stem string) string {
	if stem == "" {
		return ""
	}
	for _, c := range []string{stem + "1", stem + "2", stem + "3", stem + "_1", stem + "_0"} {
		if junkVariant(c) {
			continue
		}
		if hit := pickTex(have, c); hit != "" {
			return hit
		}
	}
	return ""
}

func pickStemVariant(have map[string]string, stem string) string {
	if stem == "" {
		return ""
	}
	for _, c := range []string{stem + "_s0", stem + "_s1", stem + "_green"} {
		if hit := pickTex(have, c); hit != "" {
			return hit
		}
	}
	pre := stem + "_"
	var vars []string
	for b := range have {
		if !strings.HasPrefix(b, pre) || b == stem {
			continue
		}
		if junkVariant(b) || strings.HasSuffix(b, "_m") || strings.HasSuffix(b, "_light") {
			continue
		}
		vars = append(vars, b)
	}
	if len(vars) == 0 {
		return ""
	}
	sort.Strings(vars)
	return pickTex(have, vars[0])
}

func pickBase(have map[string]string, key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" || k == "-" {
		return ""
	}
	stem := texStem(k)
	if hit := pickBaseCore(have, k, stem); hit != "" {
		return hit
	}
	if stem == k {
		return pickStemVariant(have, stem)
	}
	return ""
}

func pickBaseCore(have map[string]string, k, stem string) string {
	if !junkVariant(k) {
		if hit := pickTex(have, k); hit != "" {
			return hit
		}
	}
	if hit := pickTex(have, stem+"_upper", stem+"_lower", k+"_upper", k+"_lower", stem+"_top", stem+"_side"); hit != "" && !junkVariant(hit) {
		return hit
	}
	return pickNumVariant(have, stem)
}

func pickPrefix(have map[string]string, key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" || k == "-" {
		return ""
	}
	pre := k + "_"
	best := ""
	n := 0
	for b := range have {
		if b == k || strings.HasPrefix(b, pre) {
			if junkVariant(b) {
				continue
			}
			n++
			if best == "" || b < best {
				best = b
			}
		}
	}
	if n == 0 {
		return ""
	}
	return pickTex(have, best)
}

func (r *Reader) CubeFaces(tex1, tex2 string) [6]string {
	return r.CubeFacesTyped(tex1, tex2, "")
}

func (r *Reader) CubeFacesTyped(tex1, tex2, typ string) [6]string {
	var z [6]string
	if r == nil || r.bases == nil {
		return z
	}
	key := strings.ToLower(strings.TrimSpace(tex1))
	t2 := strings.ToLower(strings.TrimSpace(tex2))
	if key == "" || key == "-" {
		key = t2
		t2 = ""
	}
	if key == "" || key == "-" {
		return z
	}
	have := r.bases
	if IsPlantType(typ) {
		spr := pickBase(have, key)
		if spr == "" {
			spr = pickBase(have, t2)
		}
		if spr == "" {
			spr = pickPrefix(have, key)
		}
		if spr == "" {
			return z
		}
		return [6]string{spr, spr, spr, spr, spr, spr}
	}
	stem := texStem(key)
	exact := pickBase(have, key)
	top := pickTex(have, stem+"_top", stem+"_up", exact, stem)
	bot := pickTex(have, stem+"_bottom", stem+"_down", exact, stem)
	side := pickTex(have, stem+"_side", stem+"_normal", exact, stem)
	if NormalizeType(typ) == "grass" {
		if t := pickTex(have, "grass_top", "grasstop", stem+"_top"); t != "" {
			top = t
		}
		if s := pickTex(have, "grass_side", "grass_normal", stem+"_side"); s != "" {
			side = s
		}
		if d := pickTex(have, "dirt", "soil", "grass_bottom", stem+"_bottom"); d != "" {
			bot = d
		}
	}
	if t2 != "" && t2 != "-" && t2 != key {
		alt := pickTex(have, t2, texStem(t2)+"_side", texStem(t2))
		if alt != "" && (side == "" || side == top) {
			side = alt
		}
	}
	if bot == "" || bot == top {
		if strings.Contains(stem, "grass") {
			if d := pickTex(have, "dirt"); d != "" {
				bot = d
			}
		}
	}
	px := pickTex(have, stem+"_east", stem+"_right", side)
	nx := pickTex(have, stem+"_west", stem+"_left", side)
	pz := pickTex(have, stem+"_south", stem+"_front", side)
	nz := pickTex(have, stem+"_north", stem+"_back", side)
	fill := pickTex(have, exact, top, side, bot, stem)
	out := [6]string{px, nx, top, bot, pz, nz}
	for i := range out {
		if out[i] == "" {
			out[i] = fill
		}
	}
	return out
}

func CubeAssigned(faces [6]string) bool {
	for _, n := range faces {
		if n != "" {
			return true
		}
	}
	return false
}
