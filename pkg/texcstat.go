package pkg

import (
	"path/filepath"
	"sort"
	"strings"
)

func censusClass(name string) string {
	n := strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	if strings.HasPrefix(n, minigameBlocks) && strings.HasSuffix(n, ".png") {
		return "blocks"
	}
	if strings.HasSuffix(n, ".png") {
		return "other_png"
	}
	return "non_tex"
}

func censusExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return "(none)"
	}
	return ext
}

func censusPrefix(name string) string {
	n := strings.ReplaceAll(name, "\\", "/")
	parts := strings.Split(n, "/")
	cut := 3
	if len(parts) < cut {
		cut = len(parts)
	}
	if cut > 1 && strings.Contains(parts[cut-1], ".") {
		cut--
	}
	if cut < 1 {
		return n
	}
	return strings.Join(parts[:cut], "/")
}

func censusPrefixStat(m map[string]*TexPrefixStat, name string) *TexPrefixStat {
	k := censusPrefix(name)
	p := m[k]
	if p == nil {
		p = &TexPrefixStat{Prefix: k}
		m[k] = p
	}
	return p
}

func censusSample(c *TexCensus, s TexFailSample, seen map[uint32]int) {
	if len(c.FailSample) >= texCensusSamples {
		return
	}
	if s.Class != "lookup" {
		n := seen[s.Format]
		if n >= 2 {
			return
		}
		seen[s.Format] = n + 1
	}
	c.FailSample = append(c.FailSample, s)
}

func sortKV(m map[string]int) []kvCount {
	out := make([]kvCount, 0, len(m))
	for k, v := range m {
		out = append(out, kvCount{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func sortPrefix(m map[string]*TexPrefixStat) []TexPrefixStat {
	out := make([]TexPrefixStat, 0, len(m))
	for _, p := range m {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PNG != out[j].PNG {
			return out[i].PNG > out[j].PNG
		}
		if out[i].Names != out[j].Names {
			return out[i].Names > out[j].Names
		}
		return out[i].Prefix < out[j].Prefix
	})
	return out
}

func sortFmt(m map[uint32]*TexFmtStat) []TexFmtStat {
	out := make([]TexFmtStat, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Format < out[j].Format
	})
	return out
}
