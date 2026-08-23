package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var pngSignature = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

type BlockTexResult struct {
	OK     int
	Failed []string
}

func DecodeRainbow(plain []byte, rk RainbowKeys) []byte {
	if bytes.HasPrefix(plain, pngSignature) {
		out := make([]byte, len(plain))
		copy(out, plain)
		return out
	}
	if rk.Magic == "" || len(rk.XORKey) == 0 {
		return nil
	}
	if !bytes.HasPrefix(plain, []byte(rk.Magic)) {
		return nil
	}
	src := plain[len(rk.Magic):]
	key := rk.XORKey
	out := make([]byte, len(src))
	for i, b := range src {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

func ExtractBlockTexture(data []byte, textureKey string, opt BlockTexOpt) ([]byte, error) {
	want := opt.Prefix + textureKey + opt.Suffix
	for _, e := range ScanZipEntries(data) {
		if e.Name != want {
			continue
		}
		plain, err := ExtractZipEntry(data, e, opt.Zip)
		if err != nil {
			return nil, err
		}
		png := DecodeRainbow(plain, opt.Rainbow)
		if png == nil {
			return nil, fmt.Errorf("ExtractBlockTexture %s: unknown container head=%x", textureKey, plain[:hexHeadLen(plain)])
		}
		return png, nil
	}
	return nil, fmt.Errorf("ExtractBlockTexture %s: not found", textureKey)
}

func DumpBlockTextures(pkgPaths []string, outDir string, filterPrefix string, opt BlockTexOpt) (int, error) {
	res, err := DumpBlockTexturesReport(pkgPaths, outDir, filterPrefix, opt)
	return res.OK, err
}

func DumpBlockTexturesReport(pkgPaths []string, outDir string, filterPrefix string, opt BlockTexOpt) (BlockTexResult, error) {
	var res BlockTexResult
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}
	entries := make(map[string][]byte)
	for _, path := range pkgPaths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		p, err := ParseFile(path)
		if err != nil {
			return res, err
		}
		for _, e := range ScanZipEntries(p.Data) {
			if !strings.HasPrefix(e.Name, opt.Prefix) {
				continue
			}
			if !strings.HasSuffix(e.Name, opt.Suffix) {
				continue
			}
			key := strings.TrimSuffix(strings.TrimPrefix(e.Name, opt.Prefix), opt.Suffix)
			if filterPrefix != "" && !strings.HasPrefix(key, filterPrefix) {
				continue
			}
			plain, err := ExtractZipEntry(p.Data, e, opt.Zip)
			if err != nil {
				res.Failed = append(res.Failed, key+": "+err.Error())
				continue
			}
			png := DecodeRainbow(plain, opt.Rainbow)
			if png == nil {
				res.Failed = append(res.Failed, key+": unknown container")
				continue
			}
			entries[key] = png
		}
	}
	names := make([]string, 0, len(entries))
	for k := range entries {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if err := os.WriteFile(filepath.Join(outDir, k+".png"), entries[k], 0o644); err != nil {
			return res, err
		}
	}
	res.OK = len(entries)
	return res, nil
}

func hexHeadLen(b []byte) int {
	if len(b) < 8 {
		return len(b)
	}
	return 8
}
