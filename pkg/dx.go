package pkg

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DXResStats struct {
	Path       string         `json:"path,omitempty"`
	Names      int            `json:"names"`
	LookupOK   int            `json:"lookupOk"`
	LookupFail int            `json:"lookupFail"`
	Kinds      map[string]int `json:"kinds"`
	Chunks     map[string]int `json:"chunks,omitempty"`
	DXBC       int            `json:"dxbc"`
	Wrap10     int            `json:"wrap10"`
	Bindings   map[string]int `json:"bindings,omitempty"`
	IDListN    int            `json:"idListN,omitempty"`
	IDListDst  int            `json:"idListDstHit,omitempty"`
	IDListSrc  int            `json:"idListSrcHit,omitempty"`
}

func ScanDXResFile(path string) (DXResStats, error) {
	p, err := ParseFile(path)
	if err != nil {
		return DXResStats{}, err
	}
	rd, err := OpenReader(p)
	if err != nil {
		return DXResStats{}, err
	}
	st := ScanDXRes(rd)
	st.Path = path
	return st, nil
}

func ScanDXRes(rd *Reader) DXResStats {
	st := DXResStats{Kinds: map[string]int{}, Chunks: map[string]int{}, Bindings: map[string]int{}}
	if rd == nil {
		return st
	}
	names := rd.Names("")
	st.Names = len(names)
	compiled := map[string]struct{}{}
	var idlist []byte
	for _, n := range names {
		b, err := rd.Lookup(n)
		if err != nil {
			st.LookupFail++
			continue
		}
		st.LookupOK++
		info := ClassifyDXBlob(n, b)
		st.Kinds[string(info.Kind)]++
		if info.Kind == DXKindDXBC {
			st.DXBC++
			if info.DXOff == dxWrapLen {
				st.Wrap10++
			}
		}
		for _, c := range info.Chunks {
			st.Chunks[c]++
		}
		for _, s := range info.Bindings {
			if len(st.Bindings) < 400 || st.Bindings[s] > 0 {
				st.Bindings[s]++
			}
		}
		if info.Kind == DXKindDXBC && info.Hash != "" {
			compiled[info.Hash] = struct{}{}
		} else if isDXHashName(n) {
			compiled[strings.ToLower(filepath.Base(n))] = struct{}{}
		}
		if info.Kind == DXKindIDList {
			idlist = b
		}
	}
	if len(idlist) > 0 {
		st.IDListN, st.IDListSrc, st.IDListDst = dxCountIDList(idlist, compiled)
	}
	return st
}

func DumpDXRes(path, outDir string) (DXResStats, error) {
	p, err := ParseFile(path)
	if err != nil {
		return DXResStats{}, err
	}
	rd, err := OpenReader(p)
	if err != nil {
		return DXResStats{}, err
	}
	st := ScanDXRes(rd)
	st.Path = path
	if err := writeDXResStats(st, outDir); err != nil {
		return st, err
	}
	return st, nil
}

func DumpDXResReader(rd *Reader, outDir string) (DXResStats, error) {
	st := ScanDXRes(rd)
	return st, writeDXResStats(st, outDir)
}

func writeDXResStats(st DXResStats, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "dxstats.json"), raw, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "dxkinds.txt"), []byte(dxKindText(st)), 0o644)
}

func dxKindText(st DXResStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "names=%d ok=%d fail=%d dxbc=%d wrap10=%d idlist=%d dstHit=%d srcHit=%d\n",
		st.Names, st.LookupOK, st.LookupFail, st.DXBC, st.Wrap10, st.IDListN, st.IDListDst, st.IDListSrc)
	keys := make([]string, 0, len(st.Kinds))
	for k := range st.Kinds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "kind %s %d\n", k, st.Kinds[k])
	}
	ck := make([]string, 0, len(st.Chunks))
	for k := range st.Chunks {
		ck = append(ck, k)
	}
	sort.Strings(ck)
	for _, k := range ck {
		fmt.Fprintf(&b, "chunk %s %d\n", k, st.Chunks[k])
	}
	return b.String()
}

func dxCountIDList(b []byte, compiled map[string]struct{}) (n, srcHit, dstHit int) {
	if len(b) < 8 {
		return 0, 0, 0
	}
	n = int(binary.LittleEndian.Uint32(b[4:8]))
	off := 8
	for i := 0; i < n && off+48 <= len(b); i++ {
		if binary.LittleEndian.Uint32(b[off:off+4]) != dxHashLen || binary.LittleEndian.Uint32(b[off+24:off+28]) != dxHashLen {
			off += 48
			continue
		}
		a := hex.EncodeToString(b[off+4 : off+24])
		d := hex.EncodeToString(b[off+28 : off+48])
		if _, ok := compiled[a]; ok {
			srcHit++
		}
		if _, ok := compiled[d]; ok {
			dstHit++
		}
		off += 48
	}
	return n, srcHit, dstHit
}

func isDXHashName(n string) bool {
	n = strings.ToLower(strings.ReplaceAll(n, "\\", "/"))
	if !strings.HasPrefix(n, "d3d11/") {
		return false
	}
	base := n[6:]
	if len(base) != 40 {
		return false
	}
	for i := 0; i < 40; i++ {
		c := base[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func FindDXResPath() string {
	for _, p := range DiscoverPkgPaths() {
		if strings.EqualFold(filepath.Base(p), "dx_res.pkg") {
			return p
		}
	}
	return ""
}
