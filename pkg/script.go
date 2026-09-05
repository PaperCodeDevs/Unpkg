package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ScriptResult struct {
	OK     int
	Failed []string
}

func DumpScripts(pkgPaths []string, outDir string, keys ZipKeys, decrypt LuaDecrypt) (ScriptResult, error) {
	var res ScriptResult
	if decrypt == nil {
		return res, fmt.Errorf("DumpScripts: nil decrypt")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}
	seen := map[string]bool{}
	for _, path := range pkgPaths {
		p, err := ParseFile(path)
		if err != nil {
			return res, err
		}
		src := openZipSource(p)
		ents, err := src.entries()
		if err != nil {
			res.Failed = append(res.Failed, filepath.Base(path)+": "+err.Error())
			continue
		}
		for _, e := range ents {
			name := strings.ReplaceAll(e.Name, "\\", "/")
			if !strings.HasSuffix(strings.ToLower(name), ".lua") {
				continue
			}
			if seen[name] {
				continue
			}
			body, err := src.extract(e, keys)
			if err != nil {
				res.Failed = append(res.Failed, name+": inflate: "+err.Error())
				continue
			}
			dec, err := decrypt(body)
			if err != nil {
				res.Failed = append(res.Failed, name+": "+err.Error())
				continue
			}
			rel := SanitizeScriptOutPath(name)
			dst := filepath.Join(outDir, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return res, err
			}
			if err := os.WriteFile(dst, dec, 0o644); err != nil {
				return res, err
			}
			seen[name] = true
			res.OK++
		}
	}
	return res, nil
}

func DumpDiskLuaScripts(root, outDir string, decrypt LuaDecrypt) (ScriptResult, error) {
	var res ScriptResult
	if decrypt == nil {
		return res, fmt.Errorf("DumpDiskLuaScripts: nil decrypt")
	}
	if strings.TrimSpace(root) == "" {
		return res, fmt.Errorf("DumpDiskLuaScripts: empty root")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".lua") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			res.Failed = append(res.Failed, path+": read: "+err.Error())
			return nil
		}
		dec, err := decrypt(body)
		if err != nil {
			res.Failed = append(res.Failed, relFail(root, path)+": "+err.Error())
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		dst := filepath.Join(outDir, SanitizeScriptOutPath(filepath.ToSlash(rel)))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, dec, 0o644); err != nil {
			return err
		}
		res.OK++
		return nil
	})
	return res, err
}

func SanitizeScriptOutPath(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	var keep []string
	for _, p := range strings.Split(name, "/") {
		p = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
				return r
			}
			return '_'
		}, p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		keep = append(keep, p)
	}
	if len(keep) == 0 {
		return "_"
	}
	return filepath.Join(keep...)
}

func relFail(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}
