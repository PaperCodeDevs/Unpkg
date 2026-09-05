package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

const testASTCTex = "resources/ui/mobile/texture0/bigtex/actcenter/actct_61_img_shuoming.png"

func TestDecodeTextureASTC(t *testing.T) {
	path := filepath.Join(testPkgDir(t), "core_res.pkg")
	if _, err := os.Stat(path); err != nil {
		t.Skip("no core_res.pkg")
	}
	p, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rd.Lookup(testASTCTex)
	if err != nil {
		t.Fatal(err)
	}
	h, err := ParseTexHeader(raw)
	if err != nil {
		t.Fatal(err)
	}
	if h.Format != 48 {
		t.Skipf("format %d", h.Format)
	}
	img, err := DecodeTextureImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 1024 {
		t.Fatalf("size %v", img.Bounds())
	}
	colors := map[[4]byte]int{}
	opaque, transparent := 0, 0
	for i := 0; i+4 <= len(img.Pix); i += 4 {
		var c [4]byte
		copy(c[:], img.Pix[i:i+4])
		colors[c]++
		switch c[3] {
		case 255:
			opaque++
		case 0:
			transparent++
		}
	}
	if len(colors) < 1000 {
		t.Fatalf("only %d distinct colors", len(colors))
	}
	if opaque == 0 || transparent == 0 {
		t.Fatalf("opaque=%d transparent=%d", opaque, transparent)
	}
}
