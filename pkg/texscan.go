package pkg

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ScanBlockDefPkgFaces(raw []byte, r *Reader) []BlockFace {
	if r == nil || r.bases == nil {
		return nil
	}
	lines := strings.Split(string(raw), "\n")
	var out []BlockFace
	for i, ln := range lines {
		if i == 0 {
			continue
		}
		ln = strings.TrimRight(ln, "\r")
		if ln == "" || ln[0] < '0' || ln[0] > '9' {
			continue
		}
		parts := splitFaceCSV(ln)
		if len(parts) < 2 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || id < 0 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		keys := parts[1:]
		if len(parts) >= 47 {
			keys = []string{parts[45], parts[46]}
			if len(parts) > 3 {
				keys = append(keys, parts[3])
			}
		}
		var tex1, tex2 string
		for _, p := range keys {
			k := strings.ToLower(strings.TrimSpace(p))
			if k == "" || k == "-" {
				continue
			}
			hit := pickBase(r.bases, k)
			if hit == "" {
				continue
			}
			if cubeFillTex(hit) && k != hit && k != "grass" && k != "dirt" {
				continue
			}
			if tex1 == "" {
				tex1 = hit
			} else if tex2 == "" && hit != tex1 {
				tex2 = hit
			}
		}
		if tex1 == "" {
			continue
		}
		out = append(out, BlockFace{ID: id, Name: name, Tex1: tex1, Tex2: tex2})
	}
	return out
}

func LoadBlockFacesScan(basePath, patchPath string, r *Reader) ([]BlockFace, error) {
	faces, err := LoadBlockFaces(basePath, patchPath)
	if err != nil {
		return nil, err
	}
	extra := scanFacesFile(basePath, r)
	extra2 := scanFacesFile(patchPath, r)
	return MergeBlockFaces(MergeBlockFaces(faces, extra), extra2), nil
}

func scanFacesFile(path string, r *Reader) []BlockFace {
	if strings.TrimSpace(path) == "" || r == nil {
		return nil
	}
	p, err := ParseFile(path)
	if err != nil {
		return nil
	}
	plain, err := DecompressLZ4Block(p.Data, 128<<20)
	if err != nil {
		return nil
	}
	raw := SliceCsvByHeader(plain, CsvBlockDefHeader)
	return ScanBlockDefPkgFaces(raw, r)
}

func cubeFillTex(n string) bool {
	return n == "grass_top" || n == "grass_side" || n == "dirt"
}

func DefaultDefDumpPaths() []string {
	return []string{
		filepath.Join(".temp", "ida", "blocktypes_dump.txt"),
		`F:\Program Files\miniworldLauncher\blocktypes_dump.txt`,
	}
}

func LoadBlockDefDump(paths ...string) (map[int]string, map[int]string) {
	types := map[int]string{}
	tex := map[int]string{}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fs := strings.Split(sc.Text(), "\t")
			if len(fs) < 3 {
				continue
			}
			id, err := strconv.Atoi(strings.TrimSpace(fs[0]))
			if err != nil || id < 0 {
				continue
			}
			if t := NormalizeType(fs[1]); t != "" {
				if _, ok := types[id]; !ok {
					types[id] = t
				}
			}
			tx := strings.ToLower(strings.TrimSpace(fs[2]))
			if tx != "" && tx != "-" {
				if _, ok := tex[id]; !ok {
					tex[id] = tx
				}
			}
		}
		f.Close()
	}
	return types, tex
}
