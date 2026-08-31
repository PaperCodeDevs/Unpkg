package pkg

import (
	"fmt"
	"sort"
	"strings"
)

const csvDefDir = "../script/csvdef/utf8/"

func CsvDefPath(name string) string {
	n := strings.TrimSpace(name)
	n = strings.ReplaceAll(n, "\\", "/")
	n = strings.TrimSuffix(n, ".csv")
	if i := strings.LastIndex(n, "/"); i >= 0 {
		n = n[i+1:]
	}
	return csvDefDir + strings.ToLower(n) + ".csv"
}

func csvBase(name string) string {
	n := strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(n, "/"); i >= 0 {
		n = n[i+1:]
	}
	return strings.ToLower(n)
}

func (r *Reader) LookupCsvDef(name string) (string, []byte, error) {
	if r == nil {
		return "", nil, fmt.Errorf("nil reader")
	}
	path := CsvDefPath(name)
	b, err := r.Lookup(path)
	if err == nil && len(b) > 32 {
		return path, b, nil
	}
	want := csvBase(path)
	for _, n := range r.Names("") {
		if csvBase(n) != want {
			continue
		}
		b, err = r.Lookup(n)
		if err == nil && len(b) > 32 {
			return n, b, nil
		}
		if err != nil {
			return n, nil, err
		}
	}
	if err != nil {
		return path, nil, err
	}
	return path, nil, fmt.Errorf("not found: %s", path)
}

func (r *Reader) ListCsvDefs() []string {
	if r == nil {
		return nil
	}
	var out []string
	for _, n := range r.Names("") {
		ln := strings.ToLower(strings.ReplaceAll(n, "\\", "/"))
		if strings.Contains(ln, "csvdef/") && strings.HasSuffix(ln, ".csv") {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
