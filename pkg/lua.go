package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	luajit "github.com/PaperCodeDevs/Unpkg"
)

type GameScriptLuaResult struct {
	SourceOK   int
	Bytecode   int
	Decompiled int
	Failed     []string
}

func DumpGameScriptLua(pkgPaths []string, outDir string) (GameScriptLuaResult, error) {
	var res GameScriptLuaResult
	srcDir := filepath.Join(outDir, "source")
	bcDir := filepath.Join(outDir, "luajit")
	decDir := filepath.Join(outDir, "decompiled")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return res, err
	}
	if err := os.MkdirAll(bcDir, 0o755); err != nil {
		return res, err
	}
	if err := os.MkdirAll(decDir, 0o755); err != nil {
		return res, err
	}

	type bcEntry struct {
		name string
		body []byte
	}
	latestBC := map[string]bcEntry{}
	seenSrc := map[string]bool{}

	for _, path := range pkgPaths {
		p, err := ParseFile(path)
		if err != nil {
			return res, err
		}
		r, err := OpenReader(p)
		if err != nil {
			res.Failed = append(res.Failed, filepath.Base(path)+": "+err.Error())
			continue
		}
		plain, err := r.ConcatPlain()
		if err != nil {
			res.Failed = append(res.Failed, filepath.Base(path)+": lz4: "+err.Error())
			continue
		}
		for _, e := range extractEmbeddedLuaSources(plain) {
			key := strings.ToLower(e.Path)
			if seenSrc[key] {
				continue
			}
			dst := filepath.Join(srcDir, SanitizeScriptOutPath(e.Path))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return res, err
			}
			if err := os.WriteFile(dst, e.Body, 0o644); err != nil {
				return res, err
			}
			seenSrc[key] = true
			res.SourceOK++
		}
		for _, n := range r.Names("") {
			ln := strings.ToLower(n)
			if !strings.HasSuffix(ln, ".lua") {
				continue
			}
			body, err := r.Lookup(n)
			if err != nil || !bytes.HasPrefix(body, luaJITFileMagic) {
				continue
			}
			latestBC[ln] = bcEntry{name: n, body: body}
		}
	}

	for _, entry := range latestBC {
		rel := SanitizeScriptOutPath(entry.name)
		bcPath := filepath.Join(bcDir, rel+".ljbc")
		if err := os.MkdirAll(filepath.Dir(bcPath), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(bcPath, entry.body, 0o644); err != nil {
			return res, err
		}
		res.Bytecode++

		src, err := luajit.Decompile(entry.body)
		if err != nil || len(src) == 0 {
			msg := "empty"
			if err != nil {
				msg = err.Error()
			}
			res.Failed = append(res.Failed, entry.name+": "+msg)
			continue
		}
		luaRel := rel
		if !strings.HasSuffix(strings.ToLower(luaRel), ".lua") {
			luaRel += ".lua"
		}
		decPath := filepath.Join(decDir, luaRel)
		if err := os.MkdirAll(filepath.Dir(decPath), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(decPath, src, 0o644); err != nil {
			return res, err
		}
		res.Decompiled++
	}

	return res, nil
}

func DumpDecompilePkg(pkgPaths []string, outDir string) (GameScriptLuaResult, error) {
	var res GameScriptLuaResult
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return res, err
	}

	type bcEntry struct {
		name string
		body []byte
	}
	latestBC := map[string]bcEntry{}
	seenSrc := map[string]bool{}

	for _, path := range pkgPaths {
		p, err := ParseFile(path)
		if err != nil {
			return res, err
		}
		r, err := OpenReader(p)
		if err != nil {
			res.Failed = append(res.Failed, filepath.Base(path)+": "+err.Error())
			continue
		}
		plain, err := r.ConcatPlain()
		if err == nil {
			for _, e := range extractEmbeddedLuaSources(plain) {
				key := strings.ToLower(e.Path)
				if seenSrc[key] {
					continue
				}
				dst := filepath.Join(outDir, SanitizeScriptOutPath(e.Path))
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err == nil {
					_ = os.WriteFile(dst, e.Body, 0o644)
					seenSrc[key] = true
					res.SourceOK++
				}
			}
		}
		for _, n := range r.Names("") {
			ln := strings.ToLower(n)
			if !strings.HasSuffix(ln, ".lua") {
				continue
			}
			body, err := r.Lookup(n)
			if err != nil || !bytes.HasPrefix(body, luaJITFileMagic) {
				continue
			}
			latestBC[ln] = bcEntry{name: n, body: body}
		}
	}

	for _, entry := range latestBC {
		src, err := luajit.Decompile(entry.body)
		if err != nil || len(src) == 0 {
			msg := "empty"
			if err != nil {
				msg = err.Error()
			}
			res.Failed = append(res.Failed, entry.name+": "+msg)
			continue
		}
		luaRel := SanitizeScriptOutPath(entry.name)
		if !strings.HasSuffix(strings.ToLower(luaRel), ".lua") {
			luaRel += ".lua"
		}
		decPath := filepath.Join(outDir, luaRel)
		if err := os.MkdirAll(filepath.Dir(decPath), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(decPath, src, 0o644); err != nil {
			return res, err
		}
		res.Decompiled++
		res.Bytecode++
	}

	return res, nil
}

func DefaultGameScriptPkgs() []string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(appdata, "miniworddata110", "pkg_assets"),
		filepath.Join(appdata, "miniworddata121", "pkg_assets"),
	}
	for _, dir := range candidates {
		var out []string
		for _, name := range []string{"game_script.pkg", "patch_game_script.pkg"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func DefaultAppDataRoot() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return ""
	}
	for _, name := range []string{"miniworddata110", "miniworddata121"} {
		p := filepath.Join(appdata, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(appdata, "miniworddata110")
}

func DefaultCommonResPkgs() []string {
	return defaultResPkgs("common_res.pkg", "patch_common_res.pkg")
}

func DefaultCoreResPkgs() []string {
	return defaultResPkgs("core_res.pkg", "patch_core_res.pkg")
}

func defaultResPkgs(names ...string) []string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(appdata, "miniworddata110", "pkg_assets"),
		filepath.Join(appdata, "miniworddata121", "pkg_assets"),
	}
	for _, dir := range candidates {
		var out []string
		for _, name := range names {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func SummarizeDump(label string, ok int, failed []string) string {
	return fmt.Sprintf("%s ok=%d fail=%d", label, ok, len(failed))
}
