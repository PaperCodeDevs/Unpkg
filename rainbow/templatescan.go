package rainbow

import (
	"strings"

	"github.com/PaperCodeDevs/Unpkg/pkg"
)

type TemplateStat struct {
	Total    int
	Parsed   int
	Failed   int
	WithHLSL int
	Mats     int
	MatOK    int
	Kinds    map[string]int
	HLSL     map[string]int
	Slots    map[string]int
	Keywords map[string]int
}

func ScanReader(rd *pkg.Reader) TemplateStat {
	st := TemplateStat{
		Kinds:    map[string]int{},
		HLSL:     map[string]int{},
		Slots:    map[string]int{},
		Keywords: map[string]int{},
	}
	if rd == nil {
		return st
	}
	for _, n := range rd.Names("") {
		low := strings.ToLower(strings.ReplaceAll(n, `\`, `/`))
		tmpl := strings.HasSuffix(low, ".templatemat")
		mat := strings.HasSuffix(low, ".mat")
		if !tmpl && !mat {
			continue
		}
		b, err := rd.Lookup(n)
		if err != nil {
			if tmpl {
				st.Total++
				st.Failed++
			}
			continue
		}
		t, err := ParseTemplateMat(b)
		if tmpl {
			st.Total++
			if err != nil {
				st.Failed++
				continue
			}
			if t.Name == "" {
				t.Name = n
			}
			st.Parsed++
			addStat(&st, t)
			continue
		}
		st.Mats++
		if err == nil {
			st.MatOK++
		}
	}
	return st
}

func ScanPkgFile(path string) (TemplateStat, error) {
	p, err := pkg.ParseFile(path)
	if err != nil {
		return TemplateStat{}, err
	}
	rd, err := pkg.OpenReader(p)
	if err != nil {
		return TemplateStat{}, err
	}
	return ScanReader(rd), nil
}

func Overlap(have, want []string) (hit, miss []string) {
	set := map[string]struct{}{}
	for _, s := range have {
		set[s] = struct{}{}
	}
	for _, s := range want {
		if _, ok := set[s]; ok {
			hit = append(hit, s)
		} else {
			miss = append(miss, s)
		}
	}
	return hit, miss
}

func CloudKeys(raw []byte) (slots, keywords []string) {
	t, err := ParseTemplateMat(raw)
	if err != nil {
		return nil, nil
	}
	return t.Slots, t.Keywords
}

func addStat(st *TemplateStat, t *TemplateMat) {
	st.Kinds[t.Kind]++
	if len(t.HLSL) > 0 {
		st.WithHLSL++
	}
	for _, s := range t.HLSL {
		st.HLSL[s]++
	}
	for _, s := range t.Slots {
		st.Slots[s]++
	}
	for _, s := range t.Keywords {
		st.Keywords[s]++
	}
}
