package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func testPkgDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.Getenv("APPDATA"), "miniworddata110", "pkg_assets")
	if _, err := os.Stat(filepath.Join(dir, "game_script.pkg")); err != nil {
		t.Skip("no pkg_assets")
	}
	return dir
}

func TestParseOpenGameScript(t *testing.T) {
	path := filepath.Join(testPkgDir(t), "game_script.pkg")
	p, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	names := rd.Names("")
	if len(names) == 0 {
		t.Fatal("no names")
	}
	plain, err := rd.ConcatPlain()
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) == 0 {
		t.Fatal("empty concat")
	}
}

func TestParseHeaders(t *testing.T) {
	dir := testPkgDir(t)
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".pkg" {
			continue
		}
		n++
		p, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s parse: %v", e.Name(), err)
		}
		if _, err := OpenReader(p); err != nil {
			t.Fatalf("%s open: %v", e.Name(), err)
		}
	}
	if n == 0 {
		t.Fatal("no pkg files")
	}
}

func TestOverlayResolveUltrastone(t *testing.T) {
	dir := testPkgDir(t)
	base := filepath.Join(dir, "common_res.pkg")
	patch := filepath.Join(dir, "patch_common_res.pkg")
	if _, err := os.Stat(base); err != nil {
		t.Skip("no common_res")
	}
	r, err := OpenOverlayFiles(base, patch)
	if err != nil {
		t.Fatal(err)
	}
	name, raw, err := r.ResolveTex("ultrastone")
	if err != nil {
		t.Fatal(err)
	}
	if name == "" || len(raw) == 0 {
		t.Fatalf("empty tex name=%q n=%d", name, len(raw))
	}
	png, err := DecodeTexturePNG(raw)
	if err != nil || len(png) == 0 {
		t.Fatalf("decode %v n=%d", err, len(png))
	}
}
