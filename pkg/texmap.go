package pkg

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
)

type BlockFace struct {
	ID      int
	Name    string
	Type    string
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
		if !blockDefColsOK(len(parts)) {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || id < 0 {
			continue
		}
		f := BlockFace{
			ID:    id,
			Name:  strings.TrimSpace(parts[1]),
			Group: strings.TrimSpace(parts[13]),
			Tex1:  strings.TrimSpace(parts[45]),
			Tex2:  strings.TrimSpace(parts[46]),
			Mix:   strings.TrimSpace(parts[81]),
		}
		if len(parts) > 7 {
			f.Type = strings.TrimSpace(parts[7])
		}
		out = append(out, f)
	}
	return out
}

func blockDefColsOK(n int) bool {
	return n == 85 || n == 86
}

func ParseBlockDefNamedCSV(raw []byte) []BlockFace {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	csvr := csv.NewReader(bytes.NewReader(raw))
	csvr.LazyQuotes = true
	csvr.FieldsPerRecord = -1
	recs, err := csvr.ReadAll()
	if err != nil || len(recs) < 2 {
		return ParseBlockDefFaces(raw)
	}
	hdrRow := 0
	tex1, tex2, mix, typ, grp := 45, 46, 81, 7, 13
	find := func(row []string) bool {
		t1, t2, mx, tp, g := -1, -1, -1, -1, -1
		for i, h := range row {
			switch strings.TrimSpace(h) {
			case "Texture1":
				t1 = i
			case "Texture2":
				t2 = i
			case "MixTexture":
				mx = i
			case "Type":
				tp = i
			case "TextureGroup":
				g = i
			}
		}
		if t1 < 0 {
			return false
		}
		tex1, tex2, mix, typ, grp = t1, t2, mx, tp, g
		return true
	}
	if !find(recs[0]) && len(recs) > 1 && find(recs[1]) {
		hdrRow = 1
	}
	var out []BlockFace
	for _, rec := range recs[hdrRow+1:] {
		if len(rec) == 0 {
			continue
		}
		idStr := strings.TrimSpace(rec[0])
		if idStr == "" || idStr[0] < '0' || idStr[0] > '9' {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 0 {
			continue
		}
		at := func(i int) string {
			if i < 0 || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		out = append(out, BlockFace{
			ID:    id,
			Name:  at(1),
			Type:  at(typ),
			Group: at(grp),
			Tex1:  at(tex1),
			Tex2:  at(tex2),
			Mix:   at(mix),
		})
	}
	return out
}

func LoadNamedBlockDef(gsPaths []string) ([]byte, []BlockFace, error) {
	if len(gsPaths) == 0 {
		return nil, nil, nil
	}
	base, patch := gsPaths[0], ""
	if len(gsPaths) > 1 {
		patch = gsPaths[len(gsPaths)-1]
	}
	rd, err := OpenOverlayFiles(base, patch)
	if err != nil {
		return nil, nil, err
	}
	_, raw, err := rd.LookupCsvDef("blockdef")
	if err != nil {
		return nil, nil, err
	}
	return raw, ParseBlockDefNamedCSV(raw), nil
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
		cur, ok := byID[f.ID]
		if !ok {
			byID[f.ID] = f
			continue
		}
		if f.Name != "" {
			cur.Name = f.Name
		}
		if f.Tex1 != "" && f.Tex1 != "-" {
			cur.Tex1 = f.Tex1
		}
		if f.Tex2 != "" && f.Tex2 != "-" {
			cur.Tex2 = f.Tex2
		}
		if f.Mix != "" {
			cur.Mix = f.Mix
		}
		if f.Group != "" {
			cur.Group = f.Group
		}
		if f.Type != "" {
			cur.Type = f.Type
		}
		byID[f.ID] = cur
	}
	out := make([]BlockFace, 0, len(byID))
	for _, f := range byID {
		out = append(out, f)
	}
	return out, nil
}
