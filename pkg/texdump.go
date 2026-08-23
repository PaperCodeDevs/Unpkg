package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const minigameBlocks = "resources/minigame/blocks/"

func DumpMinigameBlockPNGs(pkgPath, outDir, filter string) (int, error) {
	p, err := ParseFile(pkgPath)
	if err != nil {
		return 0, err
	}
	r, err := OpenReader(p)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, err
	}
	names := r.Names(minigameBlocks)
	sort.Strings(names)
	n := 0
	for _, name := range names {
		if !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(name), ".png")
		if filter != "" && !strings.HasPrefix(base, filter) {
			continue
		}
		raw, err := r.Lookup(name)
		if err != nil {
			return n, fmt.Errorf("%s: %w", name, err)
		}
		png, err := DecodeTexturePNG(raw)
		if err != nil {
			continue
		}
		out := filepath.Join(outDir, base+".png")
		if err := os.WriteFile(out, png, 0o644); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func WriteBlockFaceTSV(path string, faces []BlockFace) error {
	sorted := append([]BlockFace(nil), faces...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	var b strings.Builder
	b.WriteString("blockid\tname\ttex1\tfile1\ttex2\tfile2\tmix\tfilemix\n")
	for _, f := range sorted {
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			f.ID, f.Name, f.Tex1, fileBase(f.File1), f.Tex2, fileBase(f.File2), f.Mix, fileBase(f.FileMix))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func fileBase(name string) string {
	if name == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(name), ".png")
}

func FaceCoverage(faces []BlockFace) (all, tex1, tex2, mix int) {
	for _, f := range faces {
		all++
		if f.File1 != "" {
			tex1++
		}
		if f.File2 != "" {
			tex2++
		}
		if f.FileMix != "" {
			mix++
		}
	}
	return
}
