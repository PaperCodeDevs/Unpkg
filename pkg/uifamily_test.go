package pkg

import (
	"bytes"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaperCodeDevs/Unpkg/spine"
)

func openCoreOverlay(t *testing.T) *Reader {
	t.Helper()
	dir := testPkgDir(t)
	base := filepath.Join(dir, "core_res.pkg")
	if _, err := os.Stat(base); err != nil {
		t.Skip("no core_res")
	}
	patch := filepath.Join(dir, "patch_core_res.pkg")
	if _, err := os.Stat(patch); err != nil {
		patch = ""
	}
	rd, err := OpenOverlayFiles(base, patch)
	if err != nil {
		t.Fatal(err)
	}
	return rd
}

func namesWithExt(rd *Reader, ext string) []string {
	var out []string
	for _, n := range rd.Names("") {
		if strings.HasSuffix(strings.ToLower(n), ext) {
			out = append(out, n)
		}
	}
	return out
}

func TestUIFamily(t *testing.T) {
	rd := openCoreOverlay(t)
	t.Run("FGUI", func(t *testing.T) { checkUIFamilyFGUI(t, rd) })
	t.Run("Spine", func(t *testing.T) { checkUIFamilySpine(t, rd) })
	t.Run("CropScaled", func(t *testing.T) { checkUIFamilyCrop(t, rd) })
}

func checkUIFamilyFGUI(t *testing.T, rd *Reader) {
	names := namesWithExt(rd, ".fui")
	if len(names) == 0 {
		t.Fatal("no .fui")
	}
	vers := map[int]int{}
	for _, n := range names {
		raw, err := rd.Lookup(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		info, err := ParseFGUIPack(raw, n)
		if err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		vers[info.Version]++
	}
	t.Logf("fui=%d versions=%v", len(names), vers)
}

func checkUIFamilySpine(t *testing.T, rd *Reader) {
	skels := namesWithExt(rd, ".skel")
	atlases := namesWithExt(rd, ".atlas")
	if len(skels) == 0 || len(atlases) == 0 {
		t.Fatal("no spine files")
	}
	vers := map[string]int{}
	for _, n := range skels {
		raw, err := rd.Lookup(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		sk, err := spine.ParseSkeleton(raw)
		if err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		vers[sk.Version]++
	}
	pageMiss, regionBad, pages := 0, 0, 0
	for _, n := range atlases {
		raw, err := rd.Lookup(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		at, err := spine.ParseAtlas(raw)
		if err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		dir := path.Dir(n)
		stem := strings.TrimSuffix(path.Base(n), path.Ext(n))
		for _, pg := range at.Pages {
			pages++
			if _, err := rd.Lookup(dir + "/" + pg.Name); err != nil {
				if _, err2 := rd.Lookup(dir + "/" + stem + ".png"); err2 != nil {
					pageMiss++
					t.Logf("page missing %s -> %s", n, pg.Name)
				}
			}
			for _, r := range pg.Regions {
				x, y, w, h := r.PageRect()
				if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > pg.Width || y+h > pg.Height {
					regionBad++
				}
			}
		}
	}
	if regionBad != 0 || pageMiss != 0 {
		t.Errorf("regionBad=%d pageMiss=%d", regionBad, pageMiss)
	}
	t.Logf("skel=%d versions=%v atlas=%d pages=%d", len(skels), vers, len(atlases), pages)
}

func checkUIFamilyCrop(t *testing.T, rd *Reader) {
	imgs, err := NewFGUIImages([]*Reader{rd})
	if err != nil {
		t.Fatal(err)
	}
	raw, asset, err := lookupFGUIBytes([]*Reader{rd}, "commonWidgetPic")
	if err != nil {
		t.Skip("no commonWidgetPic")
	}
	info, err := ParseFGUIPack(raw, asset)
	if err != nil {
		t.Fatal(err)
	}
	hasSprite := map[string]bool{}
	for _, sp := range info.Sprites {
		hasSprite[sp.ItemID] = sp.File != ""
	}
	tried, failed := 0, 0
	for _, it := range info.Items {
		if it.Type != fguiTypeImage || it.Name == "" || !hasSprite[it.ID] {
			continue
		}
		tried++
		b, err := imgs.Crop(info.Name, it.Name)
		if err != nil {
			failed++
			t.Logf("crop %s: %v", it.Name, err)
			continue
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Errorf("png %s: %v", it.Name, err)
			continue
		}
		if w, h, ok := imgs.Size(info.Name, it.Name); !ok || img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			t.Errorf("size %s: png=%v size=%dx%d", it.Name, img.Bounds(), w, h)
		}
	}
	if tried == 0 || failed != 0 {
		t.Fatalf("crop tried=%d failed=%d", tried, failed)
	}
	t.Logf("crop tried=%d", tried)
}
