package pkg

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func IsHotfixPkgName(name string) bool {
	n := strings.ToLower(filepath.Base(name))
	if !strings.HasSuffix(n, ".pkg") {
		return false
	}
	if strings.Contains(n, "merge") {
		return true
	}
	return strings.Contains(n, "hotfix")
}

func DiscoverHotfixPkgPaths() []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		if _, err := os.Stat(p); err != nil {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, dir := range hotfixSearchDirs() {
		walkHotfixDir(dir, add)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := filepath.Base(out[i]), filepath.Base(out[j])
		if a != b {
			return a < b
		}
		return out[i] < out[j]
	})
	return out
}

func hotfixSearchDirs() []string {
	var dirs []string
	push := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		dirs = append(dirs, filepath.Clean(p))
	}
	subs := []string{"update", "maindata", "download", "hotfix", "patch", "cache"}
	for _, root := range launcherRoots {
		push(root)
		for _, sub := range subs {
			push(filepath.Join(root, sub))
		}
	}
	appdata := os.Getenv("APPDATA")
	vers := []string{
		"miniworddata110", "miniworddata121", "miniworddata999",
		"miniworddata1", "miniworldgameguan110", "MiniStudio",
	}
	dataSubs := []string{
		"pkg_assets", "update", "hotfix", "patch", "cache", "download",
		"asset_cache", "AssetsCache",
		filepath.Join("data", "http", "update"),
		filepath.Join("data", "sandbox", "download"),
	}
	if appdata != "" {
		for _, ver := range vers {
			base := filepath.Join(appdata, ver)
			push(base)
			for _, sub := range dataSubs {
				push(filepath.Join(base, sub))
			}
		}
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		push(filepath.Join(local, "MiniWorld"))
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		push(filepath.Join(home, "Downloads"))
		push(filepath.Join(home, "Desktop", "MiniWorldBackup"))
		push(filepath.Join(home, "Desktop", "MiniWorld"))
	}
	push(`F:\MiniWorldRE`)
	return dirs
}

func skipHotfixWalk(name string) bool {
	switch strings.ToLower(name) {
	case "gpucache", "blob_storage", ".sentry-native", "anticheatexpert",
		"crash", "locales", "swiftshader", "webedit", "screenshots",
		"pluginres", "tqm", "devices", "pc4399sdk_res", ".git", "node_modules":
		return true
	default:
		return false
	}
}

func walkHotfixDir(dir string, add func(string)) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && skipHotfixWalk(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if IsHotfixPkgName(d.Name()) {
			add(path)
		}
		return nil
	})
}
