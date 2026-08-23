package pkg

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadMeshLogPaths(paths ...string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
		for sc.Scan() {
			if name := meshLogName(sc.Text()); name != "" {
				out[name] = struct{}{}
			}
		}
		f.Close()
	}
	return out
}

func meshLogName(ln string) string {
	i := strings.Index(ln, "blocks/")
	if i < 0 {
		return ""
	}
	s := ln[i:]
	for j, r := range s {
		if r == ' ' || r == '\t' || r == '"' {
			s = s[:j]
			break
		}
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasSuffix(s, ".obj") || strings.HasSuffix(s, ".blockmesh") {
		return s
	}
	return ""
}

func ResolveMeshLog(tex, typ string, logs map[string]struct{}) string {
	if !MeshAllowed(typ) {
		return ""
	}
	if hit := meshExact(tex, logs); hit != "" && meshFamilyOK(typ, hit) {
		return hit
	}
	if hit := meshPrefix(tex, logs); hit != "" && meshFamilyOK(typ, hit) {
		return hit
	}
	if typ != "" && !strings.EqualFold(typ, tex) {
		if hit := meshExact(typ, logs); hit != "" && meshFamilyOK(typ, hit) {
			return hit
		}
		if hit := meshPrefixLoose(typ, logs); hit != "" && meshFamilyOK(typ, hit) {
			return hit
		}
	}
	if IsTrapdoorType(typ) {
		if hit := meshPrefixLoose("trap", logs); hit != "" && meshFamilyOK(typ, hit) {
			return hit
		}
	}
	return ""
}

func meshExact(key string, logs map[string]struct{}) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" || key == "-" {
		return ""
	}
	for _, s := range []string{key + ".obj", key + ".blockmesh", key + "_base.blockmesh"} {
		p := "blocks/" + s
		if _, ok := logs[p]; ok {
			return p
		}
	}
	return ""
}

func meshPrefix(key string, logs map[string]struct{}) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return ""
	}
	pre := "blocks/" + key
	best := ""
	for p := range logs {
		if !strings.HasPrefix(p, pre) {
			continue
		}
		if strings.Contains(p, "trapdoor") && !strings.Contains(key, "trapdoor") {
			continue
		}
		rest := strings.TrimPrefix(p, pre)
		if !meshVariant(rest) {
			continue
		}
		if best == "" || p < best {
			best = p
		}
	}
	return best
}

func meshPrefixLoose(key string, logs map[string]struct{}) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return ""
	}
	pre := "blocks/" + key
	best := ""
	for p := range logs {
		if !strings.HasPrefix(p, pre) {
			continue
		}
		if strings.Contains(p, "trapdoor") && !strings.Contains(key, "trap") {
			continue
		}
		rest := strings.TrimPrefix(p, pre)
		if !strings.HasPrefix(rest, "_") {
			continue
		}
		if best == "" || p < best {
			best = p
		}
	}
	return best
}

func meshVariant(rest string) bool {
	for _, s := range []string{
		"_close", "_open", "_base", "_side", "_top", "_bottom",
		"_upper", "_lower", "_bar", "_pillow",
	} {
		if strings.HasPrefix(rest, s) {
			return true
		}
	}
	return false
}

func PkgMeshName(logName string) string {
	n := strings.TrimPrefix(strings.ToLower(logName), "blocks/")
	return minigameBlocks + n
}

func ParseMeshFile(name string, raw []byte) (*BlockMesh, error) {
	low := strings.ToLower(name)
	if strings.HasSuffix(low, ".obj") {
		return ParseObjMesh(raw)
	}
	if strings.HasSuffix(low, ".blockmesh") {
		return ParseBlockMesh(raw)
	}
	return nil, fmt.Errorf("mesh kind")
}
