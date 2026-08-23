package pkg

import (
	"regexp"
	"strconv"
	"strings"
)

var spawnTexRe = regexp.MustCompile(`(?m)^\t// blockid (\d+) / Name=([^/]*?) / EN=([^/]*?) / Key=([^/]*?) / Tex=([^/]*?) /`)

func ParseSpawnProTexComments(src []byte) []BlockFace {
	ms := spawnTexRe.FindAllSubmatch(src, -1)
	out := make([]BlockFace, 0, len(ms))
	for _, m := range ms {
		id, err := strconv.Atoi(string(m[1]))
		if err != nil || id < 0 {
			continue
		}
		out = append(out, BlockFace{
			ID:   id,
			Name: strings.TrimSpace(string(m[2])),
			Tex1: strings.TrimSpace(string(m[5])),
		})
	}
	return out
}

func MergeBlockFaces(def, sp []BlockFace) []BlockFace {
	byID := make(map[int]BlockFace, len(def)+len(sp))
	for _, f := range def {
		byID[f.ID] = f
	}
	for _, f := range sp {
		cur := byID[f.ID]
		cur.ID = f.ID
		if f.Name != "" {
			cur.Name = f.Name
		}
		if f.Tex1 != "" && f.Tex1 != "-" {
			cur.Tex1 = f.Tex1
		}
		byID[f.ID] = cur
	}
	out := make([]BlockFace, 0, len(byID))
	for _, f := range byID {
		out = append(out, f)
	}
	return out
}
