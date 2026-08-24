package pkg

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func ExtractFGUI(pkgPaths []string, packName, outDir string) error {
	packName = strings.TrimSpace(packName)
	if packName == "" {
		return fmt.Errorf("ExtractFGUI: 无包名")
	}
	rs, err := openPkgReaders(pkgPaths)
	if err != nil {
		return err
	}
	raw, asset, err := lookupFGUIBytes(rs, packName)
	if err != nil {
		return err
	}
	p, err := parseFGUI(raw, fguiAssetPath(asset))
	if err != nil {
		return err
	}
	dir := filepath.Join(outDir, p.name)
	if err := os.MkdirAll(filepath.Join(dir, "atlas"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "font"), 0o755); err != nil {
		return err
	}
	fuiName := p.name + ".fui"
	if err := os.WriteFile(filepath.Join(dir, fuiName), raw, 0o644); err != nil {
		return err
	}
	done := map[string]bool{}
	for _, it := range p.items {
		if it.typ != fguiTypeAtlas && it.typ != fguiTypeMisc && it.typ != fguiTypeFont && it.typ != fguiTypeSound {
			continue
		}
		file := strings.TrimSpace(it.file)
		if file == "" {
			continue
		}
		data, got, err := lookupPkgFile(rs, file, p.asset, filepath.Base(asset))
		if err != nil {
			continue
		}
		if done[got] {
			continue
		}
		done[got] = true
		base := filepath.Base(got)
		low := strings.ToLower(base)
		if it.typ == fguiTypeAtlas || strings.HasSuffix(low, ".png") || strings.HasSuffix(low, ".jpg") {
			img, err := decodeAtlasImage(data)
			if err != nil {
				_ = os.WriteFile(filepath.Join(dir, "atlas", base+".raw"), data, 0o644)
				continue
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return err
			}
			pngName := strings.TrimSuffix(base, filepath.Ext(base)) + ".png"
			if err := os.WriteFile(filepath.Join(dir, "atlas", pngName), buf.Bytes(), 0o644); err != nil {
				return err
			}
			continue
		}
		if strings.HasSuffix(low, ".ttf") || strings.HasSuffix(low, ".otf") || strings.HasSuffix(low, ".fnt") {
			if err := os.WriteFile(filepath.Join(dir, "font", base), data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func ExtractUIFile(pkgPaths []string, name, dest string) error {
	rs, err := openPkgReaders(pkgPaths)
	if err != nil {
		return err
	}
	want := strings.ReplaceAll(strings.ToLower(name), "\\", "/")
	for i := len(rs) - 1; i >= 0; i-- {
		for _, n := range rs[i].Names("") {
			k := strings.ReplaceAll(strings.ToLower(n), "\\", "/")
			if k == want || strings.HasSuffix(k, "/"+want) || strings.HasSuffix(k, want) {
				raw, err := rs[i].Lookup(n)
				if err != nil {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return err
				}
				img, decErr := decodeAtlasImage(raw)
				if decErr == nil {
					var buf bytes.Buffer
					if err := png.Encode(&buf, img); err != nil {
						return err
					}
					return os.WriteFile(dest, buf.Bytes(), 0o644)
				}
				return os.WriteFile(dest, raw, 0o644)
			}
		}
	}
	return fmt.Errorf("ExtractUIFile: 无 %s", name)
}

func ExtractOfficialFGUI(pkgPaths []string, outDir string) error {
	if strings.TrimSpace(outDir) == "" {
		return fmt.Errorf("ExtractOfficialFGUI: 无输出目录")
	}
	seen := map[string]bool{}
	for _, need := range OfficialFGUINeed {
		if seen[need.Pack] {
			continue
		}
		seen[need.Pack] = true
		if err := ExtractFGUI(pkgPaths, need.Pack, outDir); err != nil {
			return fmt.Errorf("ExtractOfficialFGUI %s: %w", need.Pack, err)
		}
	}
	ext := filepath.Join(outDir, "login", "external")
	for _, n := range []string{"ui/login_bg.jpg", "ui/login_logo.png", "ui/hotfix_loading.jpg"} {
		dest := filepath.Join(ext, strings.TrimSuffix(filepath.Base(n), filepath.Ext(n))+".png")
		_ = ExtractUIFile(pkgPaths, n, dest)
	}
	return nil
}

func lookupFGUIBytes(rs []*Reader, packName string) ([]byte, string, error) {
	want := strings.ToLower(packName)
	var found []byte
	var asset string
	for i := len(rs) - 1; i >= 0; i-- {
		for _, name := range rs[i].Names("") {
			if !strings.HasSuffix(strings.ToLower(name), ".fui") {
				continue
			}
			base := strings.ToLower(strings.TrimSuffix(filepath.Base(name), ".fui"))
			if base != want && !strings.HasSuffix(strings.ToLower(fguiAssetPath(name)), want) {
				continue
			}
			raw, err := rs[i].Lookup(name)
			if err != nil {
				continue
			}
			found, asset = raw, name
			break
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		return nil, "", fmt.Errorf("lookupFGUIBytes: 无 %s", packName)
	}
	return found, asset, nil
}

func lookupPkgFile(rs []*Reader, file, asset, fuiBase string) ([]byte, string, error) {
	stem := strings.TrimSuffix(fuiBase, ".fui")
	cands := []string{
		file,
		asset + "_" + filepath.Base(file),
		stem + "_" + filepath.Base(file),
		filepath.Base(file),
	}
	var last error
	for _, cand := range cands {
		if cand == "" {
			continue
		}
		for i := len(rs) - 1; i >= 0; i-- {
			d, err := rs[i].Lookup(cand)
			if err == nil {
				return d, cand, nil
			}
			last = err
		}
	}
	if last == nil {
		last = fmt.Errorf("not found: %s", file)
	}
	return nil, "", last
}
