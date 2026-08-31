package pkg

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"unicode/utf8"
)

type CsvDefTable struct {
	Header []string
	Rows   [][]string
	col    map[string]int
}

func ParseCsvDef(raw []byte) (*CsvDefTable, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		raw = raw[3:]
	}
	raw = bytes.ReplaceAll(raw, []byte{0}, nil)
	if !utf8.Valid(raw) {
		return nil, nil
	}
	lines := splitCsvLines(raw)
	if len(lines) == 0 {
		return &CsvDefTable{col: map[string]int{}}, nil
	}
	start := 1
	header := splitCsvHeader(lines[0])
	if len(lines) > 1 && isCsvEnglishHeader(lines[1]) {
		header = splitCsvHeader(lines[1])
		start = 2
	} else if isCsvEnglishHeader(lines[0]) {
		start = 1
	}
	t := &CsvDefTable{Header: header, col: map[string]int{}}
	for i, h := range header {
		t.col[normCsvCol(h)] = i
	}
	want := len(header)
	for _, ln := range lines[start:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		rec := splitCsvRow(ln, want)
		if len(rec) == 0 {
			continue
		}
		t.Rows = append(t.Rows, rec)
	}
	return t, nil
}

func (t *CsvDefTable) Col(name string) int {
	if t == nil || t.col == nil {
		return -1
	}
	if i, ok := t.col[strings.ToLower(strings.TrimSpace(name))]; ok {
		return i
	}
	return -1
}

func (t *CsvDefTable) Cell(row []string, name string) string {
	i := t.Col(name)
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func (t *CsvDefTable) IDName() map[int]string {
	out := map[int]string{}
	if t == nil {
		return out
	}
	idI := t.Col("ID")
	nameI := t.Col("Name")
	enI := t.Col("ENName")
	if idI < 0 {
		return out
	}
	for _, row := range t.Rows {
		if idI >= len(row) {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(row[idI]))
		if err != nil || id < 0 {
			continue
		}
		cn := ""
		if nameI >= 0 && nameI < len(row) {
			cn = strings.TrimSpace(row[nameI])
		}
		en := ""
		if enI >= 0 && enI < len(row) {
			en = strings.TrimSpace(row[enI])
		}
		name := cn
		if cn != "" && en != "" {
			name = cn + "|" + en
		} else if en != "" {
			name = en
		}
		if name == "" {
			continue
		}
		if _, ok := out[id]; !ok {
			out[id] = name
		}
	}
	return out
}

func isCsvEnglishHeader(line string) bool {
	s := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(s, "id,") && strings.Contains(s, ",name,")
}

func normCsvCol(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "(\uff08"); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(strings.TrimSpace(s))
}

func splitCsvLines(raw []byte) []string {
	s := string(raw)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func splitCsvHeader(line string) []string {
	rec, err := csv.NewReader(strings.NewReader(line)).Read()
	if err == nil && len(rec) >= 4 {
		return rec
	}
	return strings.Split(line, ",")
}

func splitCsvRow(line string, want int) []string {
	r := csv.NewReader(strings.NewReader(line))
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	rec, err := r.Read()
	if err == nil && (want <= 0 || absCsv(len(rec)-want) <= 8 || len(rec) >= 8) {
		return rec
	}
	return strings.Split(line, ",")
}

func absCsv(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
