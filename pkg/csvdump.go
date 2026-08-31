package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadCsvDef(r *Reader, name string) (path string, raw []byte, tab *CsvDefTable, err error) {
	path, raw, err = r.LookupCsvDef(name)
	if err != nil {
		return path, nil, nil, err
	}
	tab, err = ParseCsvDef(raw)
	if err != nil {
		return path, raw, nil, err
	}
	if tab == nil || tab.Col("ID") < 0 {
		return path, raw, nil, fmt.Errorf("csvdef %s: no ID column", name)
	}
	return path, raw, tab, nil
}

func ExtractBlockItemIDs(r *Reader) (blocks, items map[int]string, err error) {
	_, _, bt, err := LoadCsvDef(r, "blockdef")
	if err != nil {
		return nil, nil, err
	}
	_, _, it, err := LoadCsvDef(r, "itemdef")
	if err != nil {
		return nil, nil, err
	}
	blocks = bt.IDName()
	parsed := it.IDName()
	items = make(map[int]string, len(blocks)+len(parsed))
	for id, name := range blocks {
		items[id] = name
	}
	for id, name := range parsed {
		if _, ok := blocks[id]; ok {
			continue
		}
		items[id] = name
	}
	return blocks, items, nil
}

func DumpCsvDefIDs(basePkgPath, patchPkgPath, outDir string) (nBlock, nItem int, err error) {
	r, err := OpenOverlayFiles(basePkgPath, patchPkgPath)
	if err != nil {
		return 0, 0, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, 0, err
	}
	names := r.ListCsvDefs()
	if err := os.WriteFile(filepath.Join(outDir, "csvdef_names.txt"), []byte(strings.Join(names, "\n")+"\n"), 0o644); err != nil {
		return 0, 0, err
	}
	bPath, bRaw, bt, err := LoadCsvDef(r, "blockdef")
	if err != nil {
		return 0, 0, err
	}
	iPath, iRaw, it, err := LoadCsvDef(r, "itemdef")
	if err != nil {
		return 0, 0, err
	}
	_ = bPath
	_ = iPath
	if err := os.WriteFile(filepath.Join(outDir, "blockdef.csv"), bRaw, 0o644); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "itemdef.csv"), iRaw, 0o644); err != nil {
		return 0, 0, err
	}
	blocks := bt.IDName()
	parsed := it.IDName()
	items := make(map[int]string, len(blocks)+len(parsed))
	for id, name := range blocks {
		items[id] = name
	}
	for id, name := range parsed {
		if _, ok := items[id]; !ok {
			items[id] = name
		}
	}
	if err := WriteIDMapCSV(filepath.Join(outDir, "block_ids.csv"), blocks); err != nil {
		return 0, 0, err
	}
	if err := WriteIDMapCSV(filepath.Join(outDir, "item_ids.csv"), items); err != nil {
		return 0, 0, err
	}
	if err := writeCsvSlim(filepath.Join(outDir, "block_ui.tsv"), bt, []string{"ID", "Name", "ENName", "Texture1", "Texture2", "PlaceDir", "Directionable"}); err != nil {
		return 0, 0, err
	}
	if err := writeCsvSlim(filepath.Join(outDir, "item_ui.tsv"), it, []string{"ID", "Name", "Icon", "CreateType", "ClassificationType", "Type"}); err != nil {
		return 0, 0, err
	}
	if err := dumpRelatedCsvDefs(r, outDir); err != nil {
		return 0, 0, err
	}
	return len(blocks), len(items), nil
}

func dumpRelatedCsvDefs(r *Reader, outDir string) error {
	dir := filepath.Join(outDir, "csvdef")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{
		"blockeffectdef", "colormixdef", "dropitems", "iteminhanddef",
		"oredef", "chestdef", "material", "tooldef", "iconbank",
		"plant", "planttrees",
	} {
		path, raw, err := r.LookupCsvDef(name)
		if err != nil {
			continue
		}
		_ = path
		if err := os.WriteFile(filepath.Join(dir, csvBase(name)+".csv"), raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeCsvSlim(path string, t *CsvDefTable, cols []string) error {
	if t == nil {
		return fmt.Errorf("nil table")
	}
	idx := make([]int, len(cols))
	for i, c := range cols {
		idx[i] = t.Col(c)
	}
	var b strings.Builder
	b.WriteString(strings.Join(cols, "\t") + "\n")
	idI := t.Col("ID")
	seen := map[int]bool{}
	for _, row := range t.Rows {
		id := -1
		if idI >= 0 && idI < len(row) {
			n, err := strconv.Atoi(strings.TrimSpace(row[idI]))
			if err != nil {
				continue
			}
			id = n
		}
		if id >= 0 {
			if seen[id] {
				continue
			}
			seen[id] = true
		}
		fields := make([]string, len(cols))
		for i, ci := range idx {
			if ci >= 0 && ci < len(row) {
				fields[i] = strings.ReplaceAll(strings.TrimSpace(row[ci]), "\t", " ")
			}
		}
		b.WriteString(strings.Join(fields, "\t") + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
