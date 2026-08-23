package pkg

import (
	"encoding/csv"
	"strconv"
	"strings"
)

type BlockFace struct {
	ID      int
	Name    string
	Tex1    string
	Tex2    string
	Mix     string
	Group   string
	File1   string
	File2   string
	FileMix string
}

func ParseBlockDefFaces(raw []byte) []BlockFace {
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
		if len(parts) != 85 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || id < 0 {
			continue
		}
		out = append(out, BlockFace{
			ID:    id,
			Name:  strings.TrimSpace(parts[1]),
			Group: strings.TrimSpace(parts[13]),
			Tex1:  strings.TrimSpace(parts[45]),
			Tex2:  strings.TrimSpace(parts[46]),
			Mix:   strings.TrimSpace(parts[81]),
		})
	}
	return out
}

func splitFaceCSV(ln string) []string {
	rd := csv.NewReader(strings.NewReader(ln))
	rd.FieldsPerRecord = -1
	rd.LazyQuotes = true
	rec, err := rd.Read()
	if err != nil {
		return nil
	}
	return rec
}

func BindBlockFaces(r *Reader, faces []BlockFace) []BlockFace {
	for i := range faces {
		if n, _, err := r.ResolveTex(faces[i].Tex1); err == nil {
			faces[i].File1 = n
		}
		if n, _, err := r.ResolveTex(faces[i].Tex2); err == nil {
			faces[i].File2 = n
		}
		if n, _, err := r.ResolveTex(faces[i].Mix); err == nil {
			faces[i].FileMix = n
		}
	}
	return faces
}

func loadFacesPkg(path string) ([]BlockFace, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	p, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := DecompressLZ4Block(p.Data, 128<<20)
	if err != nil {
		return nil, err
	}
	raw := SliceCsvByHeader(plain, CsvBlockDefHeader)
	return ParseBlockDefFaces(raw), nil
}

func LoadBlockFaces(basePath, patchPath string) ([]BlockFace, error) {
	base, err := loadFacesPkg(basePath)
	if err != nil {
		return nil, err
	}
	patch, err := loadFacesPkg(patchPath)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]BlockFace, len(base)+len(patch))
	for _, f := range base {
		byID[f.ID] = f
	}
	for _, f := range patch {
		byID[f.ID] = f
	}
	out := make([]BlockFace, 0, len(byID))
	for _, f := range byID {
		out = append(out, f)
	}
	return out, nil
}
