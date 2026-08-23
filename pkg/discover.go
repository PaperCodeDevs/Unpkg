package pkg

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var launcherRoots = []string{
	`F:\Program Files\miniworldLauncher`,
	`C:\Program Files\miniworldLauncher`,
}

func DiscoverPkgPaths() []string {
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
	for _, root := range launcherRoots {
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".pkg") {
				add(filepath.Join(root, e.Name()))
			}
		}
	}
	appdata := os.Getenv("APPDATA")
	if appdata != "" {
		for _, ver := range []string{"miniworddata110", "miniworddata121", "miniworldgameguan110"} {
			dir := filepath.Join(appdata, ver, "pkg_assets")
			ents, err := os.ReadDir(dir)
			if err != nil {
				dir = filepath.Join(appdata, ver)
				ents, err = os.ReadDir(dir)
				if err != nil {
					continue
				}
			}
			for _, e := range ents {
				if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".pkg") {
					add(filepath.Join(dir, e.Name()))
				}
			}
		}
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

func DefaultLauncherRoot() string {
	for _, root := range launcherRoots {
		if _, err := os.Stat(filepath.Join(root, "engine_res.pkg")); err == nil {
			return root
		}
	}
	return launcherRoots[0]
}
