package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	CsvBlockDefHeader = "ID,Name,PrefabName,ENName,Key,TWName,RegionType"
	CsvItemDefHeader  = "ID,Name,PrefabName,Key,EditType,EditTypeEX,IsTemplate,DropType"
)

var (
	itemIDRe     = regexp.MustCompile(`(?s)(\d{1,7})[,\x00]([^,\x00\r\n]{1,40}),,([A-Za-z][A-Za-z0-9_]{1,64})`)
	itemENBadRe  = regexp.MustCompile(`[A-Za-z]\d+[A-Za-z]`)
	itemENJunkRe = regexp.MustCompile(`(?i)^(item_|misc\.|firework_|pi\d)`)
)

func SliceCsvByHeader(blob []byte, header string) []byte {
	mark := []byte(header)
	start := bytes.Index(blob, mark)
	if start < 0 {
		return nil
	}
	rest := blob[start+len(mark):]
	next := bytes.Index(rest, []byte("\nID,Name,"))
	if next < 0 {
		return append([]byte(nil), blob[start:]...)
	}
	return append([]byte(nil), blob[start:start+len(mark)+next+1]...)
}

func ParseBlockDefIDs(raw []byte) map[int]string {
	out := make(map[int]string)
	lines := bytes.Split(raw, []byte{'\n'})
	for i, ln := range lines {
		if i == 0 {
			continue
		}
		ln = bytes.TrimRight(ln, "\r")
		if len(ln) == 0 || ln[0] < '0' || ln[0] > '9' {
			continue
		}
		if bytes.Count(ln, []byte{0}) > 2 {
			continue
		}
		text := string(ln)
		parts := strings.Split(text, ",")
		if len(parts) < 4 {
			continue
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil || id < 0 || id > 1_000_000 {
			continue
		}
		cn := strings.TrimSpace(parts[1])
		en := strings.TrimSpace(parts[3])
		if strings.Contains(cn, "预留") {
			continue
		}
		if en != "" && !asciiIdent(en) {
			en = ""
		}
		if cn == "" && en == "" {
			continue
		}
		name := cn
		if cn != "" && en != "" {
			name = cn + "|" + en
		} else if en != "" {
			name = en
		}
		if _, ok := out[id]; !ok {
			out[id] = name
		}
	}
	return out
}

func ParseItemDefIDs(raw []byte) map[int]string {
	out := make(map[int]string)
	for _, m := range itemIDRe.FindAllSubmatchIndex(raw, -1) {
		if len(m) < 8 {
			continue
		}
		if m[0] > 0 {
			prev := raw[m[0]-1]
			if prev != '\n' && prev != '\r' && prev != 0 {
				continue
			}
		}
		id, err := strconv.Atoi(string(raw[m[2]:m[3]]))
		if err != nil || id < 0 || id > 2_000_000 {
			continue
		}
		cn := string(raw[m[4]:m[5]])
		en := repairItemEN(string(raw[m[6]:m[7]]))
		if !utf8.ValidString(cn) || !utf8.ValidString(en) {
			continue
		}
		if !hasCJK(cn) || len(cn) > 30 {
			continue
		}
		if itemENBadRe.MatchString(en) || itemENJunkRe.MatchString(en) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(en), "icon") && allDigit(en[4:]) {
			continue
		}
		if strings.Contains(cn, "放置后") || strings.Contains(cn, "预留") || strings.Contains(cn, "版本") {
			continue
		}
		if _, ok := out[id]; !ok {
			out[id] = cn + "|" + en
		}
	}
	return out
}

func MergeIDMaps(base, patch map[int]string) map[int]string {
	out := make(map[int]string, len(base)+len(patch))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

func WriteIDMapCSV(path string, ids map[int]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]int, 0, len(ids))
	for k := range ids {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	b.WriteString("ID,Name\n")
	for _, k := range keys {
		b.WriteString(strconv.Itoa(k))
		b.WriteByte(',')
		b.WriteString(ids[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func ExtractDefIDs(baseData, patchData []byte) (blocks, items map[int]string, err error) {
	basePlain, err := DecompressLZ4Block(baseData, 128<<20)
	if err != nil {
		return nil, nil, fmt.Errorf("ExtractDefIDs base: %w", err)
	}
	var patchPlain []byte
	if len(patchData) > 0 {
		patchPlain, err = DecompressLZ4Block(patchData, 64<<20)
		if err != nil {
			return nil, nil, fmt.Errorf("ExtractDefIDs patch: %w", err)
		}
	}
	bBase := ParseBlockDefIDs(SliceCsvByHeader(basePlain, CsvBlockDefHeader))
	iBase := ParseItemDefIDs(SliceCsvByHeader(basePlain, CsvItemDefHeader))
	var bPatch, iPatch map[int]string
	if len(patchPlain) > 0 {
		bPatch = ParseBlockDefIDs(SliceCsvByHeader(patchPlain, CsvBlockDefHeader))
		iPatch = ParseItemDefIDs(SliceCsvByHeader(patchPlain, CsvItemDefHeader))
	}
	blocks = MergeIDMaps(bBase, bPatch)
	parsedItems := MergeIDMaps(iBase, iPatch)
	items = make(map[int]string, len(blocks)+len(parsedItems))
	for id, name := range blocks {
		items[id] = name
	}
	for id, name := range parsedItems {
		if _, ok := blocks[id]; ok {
			continue
		}
		items[id] = name
	}
	return blocks, items, nil
}

func DumpDefIDs(basePkgPath, patchPkgPath, outDir string) (nBlock, nItem int, err error) {
	basePkg, err := ParseFile(basePkgPath)
	if err != nil {
		return 0, 0, err
	}
	var patchData []byte
	if strings.TrimSpace(patchPkgPath) != "" {
		patchPkg, err := ParseFile(patchPkgPath)
		if err != nil {
			return 0, 0, err
		}
		patchData = patchPkg.Data
	}
	blocks, items, err := ExtractDefIDs(basePkg.Data, patchData)
	if err != nil {
		return 0, 0, err
	}
	if err := WriteIDMapCSV(filepath.Join(outDir, "block_ids.csv"), blocks); err != nil {
		return 0, 0, err
	}
	if err := WriteIDMapCSV(filepath.Join(outDir, "item_ids.csv"), items); err != nil {
		return 0, 0, err
	}
	return len(blocks), len(items), nil
}

func repairItemEN(en string) string {
	if strings.HasPrefix(en, "S12tic") {
		return "Static" + strings.TrimPrefix(en, "S12tic")
	}
	return en
}

func asciiIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han) {
			return true
		}
	}
	return false
}

func allDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
