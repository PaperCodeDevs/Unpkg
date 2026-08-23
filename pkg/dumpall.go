package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PkgKind string

const (
	PkgIndexed  PkgKind = "indexed"
	PkgLZ4Plain PkgKind = "lz4_plain"
	PkgEngine   PkgKind = "engine_stream"
	PkgMaterial PkgKind = "material_cache"
	PkgUnknown  PkgKind = "unknown"
)

type ProbeResult struct {
	Path       string  `json:"path"`
	Base       string  `json:"base"`
	Kind       PkgKind `json:"kind"`
	Version    uint32  `json:"version"`
	Subtype    uint32  `json:"subtype"`
	DataBytes  int     `json:"dataBytes"`
	IndexBytes int     `json:"indexBytes"`
	IndexFiles int     `json:"indexFiles,omitempty"`
	Names      int     `json:"names,omitempty"`
	LZ4Plain   int     `json:"lz4Plain,omitempty"`
	Readable   float64 `json:"readable"`
	Note       string  `json:"note,omitempty"`
}

type DumpAllResult struct {
	OutDir string        `json:"outDir"`
	Probes []ProbeResult `json:"probes"`
	OK     int           `json:"ok"`
	Failed []string      `json:"failed,omitempty"`
}

func ProbePkg(path string) (ProbeResult, error) {
	res := ProbeResult{Path: path, Base: filepath.Base(path)}
	p, err := ParseFile(path)
	if err != nil {
		return res, err
	}
	res.Version = p.Version
	res.Subtype = p.Subtype
	res.DataBytes = len(p.Data)
	res.IndexBytes = len(p.Index)
	res.Readable = p.ReadableRatio(8000)
	res.Kind = classifyPkg(p, path)
	if n, ok := p.IndexFileCount(); ok {
		res.IndexFiles = n
	}
	if rd, err := OpenReader(p); err == nil {
		res.Names = len(rd.Names(""))
		if res.Kind == PkgUnknown {
			res.Kind = PkgIndexed
		}
	}
	if plain, err := p.DecompressData(len(p.Data)*2 + 64); err == nil && len(plain) > len(p.Data)/2 {
		res.LZ4Plain = len(plain)
		if res.Kind == PkgUnknown || res.Kind == PkgEngine {
			res.Kind = PkgLZ4Plain
		}
	}
	switch res.Kind {
	case PkgEngine:
		res.Note = "Rainbow serialized stream; index layout differs from common_res"
	case PkgMaterial:
		res.Note = "D3D11 shader cache; 28B file table; Data origin-16 plaintext"
	}
	if strings.Contains(strings.ToLower(res.Base), "dx_res") {
		res.Note = "D3D11 DXBC cache; 10B wrap+DXBC+bindings; not ASTC/GLSL source"
	}
	return res, nil
}

func classifyPkg(p *Pkg, path string) PkgKind {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.Contains(base, "engine_res"):
		return PkgEngine
	case strings.Contains(base, "material_res"):
		return PkgMaterial
	}
	if rd, err := OpenReader(p); err == nil && len(rd.Names("")) > 0 {
		return PkgIndexed
	}
	if plain, err := p.DecompressData(len(p.Data)*2 + 64); err == nil && len(plain) > 0 {
		if plain[0] == '<' || plain[0] == '{' || plain[0] == '#' {
			return PkgLZ4Plain
		}
	}
	return PkgUnknown
}

func DumpAll(outDir string, extract bool) (DumpAllResult, error) {
	var res DumpAllResult
	res.OutDir = outDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}
	paths := append(DiscoverPkgPaths(), DiscoverHotfixPkgPaths()...)
	seenBase := map[string]bool{}
	for _, path := range paths {
		base := filepath.Base(path)
		if seenBase[base] {
			continue
		}
		seenBase[base] = true
		probe, err := ProbePkg(path)
		if err != nil {
			res.Failed = append(res.Failed, base+": probe: "+err.Error())
			continue
		}
		res.Probes = append(res.Probes, probe)
		sub := filepath.Join(outDir, strings.TrimSuffix(base, ".pkg"))
		if err := dumpOne(path, sub, probe.Kind, extract); err != nil {
			res.Failed = append(res.Failed, base+": "+err.Error())
			continue
		}
		res.OK++
	}
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), raw, 0o644); err != nil {
		return res, err
	}
	return res, nil
}

func dumpOne(path, outDir string, kind PkgKind, extract bool) error {
	switch kind {
	case PkgEngine:
		if err := dumpEngineNames(path, outDir); err != nil {
			return err
		}
		_, _, err := DumpEngineCatalog(path, outDir)
		return err
	case PkgMaterial:
		return dumpMaterial(path, outDir)
	case PkgLZ4Plain:
		return dumpLZ4Plain(path, outDir)
	default:
		return dumpIndexed(path, outDir, extract)
	}
}

func dumpLZ4Plain(path, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	p, err := ParseFile(path)
	if err != nil {
		return err
	}
	plain, err := p.DecompressData(len(p.Data)*2 + 64)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "plain.bin"), plain, 0o644); err != nil {
		return err
	}
	idx, err := decodePkgIndex(p.Index)
	if err == nil {
		_ = os.WriteFile(filepath.Join(outDir, "index_plain.bin"), idx, 0o644)
	}
	return nil
}

func dumpIndexed(path, outDir string, extract bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	p, err := ParseFile(path)
	if err != nil {
		return err
	}
	idx, err := decodePkgIndex(p.Index)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "index_plain.bin"), idx, 0o644); err != nil {
		return err
	}
	rd, err := OpenReader(p)
	if err != nil {
		return err
	}
	names := rd.Names("")
	sortNames := append([]string(nil), names...)
	sort.Strings(sortNames)
	listPath := filepath.Join(outDir, "names.txt")
	var sb strings.Builder
	for _, n := range sortNames {
		sb.WriteString(n)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(listPath, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(filepath.Base(path)), "dx_res") {
		if _, err := DumpDXResReader(rd, outDir); err != nil {
			return err
		}
	}
	if !extract {
		return nil
	}
	filesDir := filepath.Join(outDir, "files")
	for _, name := range sortNames {
		raw, err := rd.Lookup(name)
		if err != nil {
			continue
		}
		dst := filepath.Join(filesDir, filepath.FromSlash(sanitizePkgOut(name)))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func dumpEngineNames(path, outDir string) error {
	p, err := ParseFile(path)
	if err != nil {
		return err
	}
	rd, err := OpenReader(p)
	if err != nil {
		return err
	}
	names := rd.Names("")
	sortNames := append([]string(nil), names...)
	sort.Strings(sortNames)
	var sb strings.Builder
	for _, n := range sortNames {
		sb.WriteString(n)
		sb.WriteByte('\n')
	}
	return os.WriteFile(filepath.Join(outDir, "names.txt"), []byte(sb.String()), 0o644)
}

func dumpMaterial(path, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	p, err := ParseFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "data.bin"), p.Data, 0o644); err != nil {
		return err
	}
	idx, err := decodePkgIndex(p.Index)
	if err == nil {
		_ = os.WriteFile(filepath.Join(outDir, "index_plain.bin"), idx, 0o644)
	}
	rd, err := OpenReader(p)
	if err != nil {
		return err
	}
	names := rd.Names("")
	sortNames := append([]string(nil), names...)
	sort.Strings(sortNames)
	var sb strings.Builder
	for _, n := range sortNames {
		sb.WriteString(n)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(outDir, "names.txt"), []byte(sb.String()), 0o644); err != nil {
		return err
	}
	return nil
}

func sanitizePkgOut(name string) string {
	name = strings.TrimPrefix(name, "../")
	name = strings.TrimPrefix(name, "./")
	return filepath.Clean(strings.ReplaceAll(name, "..", "_"))
}
