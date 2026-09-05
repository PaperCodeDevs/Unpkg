package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
)

func TestTintKey(t *testing.T) {
	if got := TintKey("grass_side", 0x46a922); got != "grass_side_46a922" {
		t.Fatalf("got %q", got)
	}
	c, ok := ParseHexColor("46a922")
	if !ok || c != 0x46a922 {
		t.Fatalf("parse %v %x", ok, c)
	}
}

func testCommonOverlay(t *testing.T) *Reader {
	t.Helper()
	dir := testPkgDir(t)
	base := filepath.Join(dir, "common_res.pkg")
	patch := filepath.Join(dir, "patch_common_res.pkg")
	r, err := OpenOverlayFiles(base, patch)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCubeFacesRuntimeGrass(t *testing.T) {
	r := testCommonOverlay(t)
	gray := r.CubeFacesTyped("grass", "", "grass")
	if gray[2] != "grass_top" || gray[3] != "dirt" {
		t.Fatalf("gray faces %+v", gray)
	}
	for _, i := range []int{0, 1, 4, 5} {
		if gray[i] != "grass_side" {
			t.Fatalf("gray side %+v", gray)
		}
	}
	rt := r.CubeFacesRuntime("grass", "", "grass", 0x46a922)
	if rt[2] != "grass_top" {
		t.Fatalf("top tinted? %+v", rt)
	}
	if rt[3] != "dirt" {
		t.Fatalf("bot %+v", rt)
	}
	for _, i := range []int{0, 1, 4, 5} {
		if rt[i] != "grass_side_46a922" {
			t.Fatalf("runtime side %+v", rt)
		}
	}
}

func TestGrassSide46a922DXT(t *testing.T) {
	r := testCommonOverlay(t)
	_, raw, err := r.ResolveTex("grass_side_46a922")
	if err != nil {
		t.Fatal(err)
	}
	dxt, w, h, err := DecodeTextureDXT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if w != 256 || h != 256 || len(dxt) != 65536 {
		t.Fatalf("size %dx%d len=%d", w, h, len(dxt))
	}
	sum := sha256.Sum256(dxt)
	got := hex.EncodeToString(sum[:])
	const want = "f6e575b319d755e376c9c3cec51e908e54d9487805dac4aaf206893c23b21964"
	if got != want {
		t.Fatalf("dxt sha %s", got)
	}
}

func TestGlass64DXT(t *testing.T) {
	r := testCommonOverlay(t)
	_, raw, err := r.ResolveTex("glass")
	if err != nil {
		t.Fatal(err)
	}
	dxt, w, h, err := DecodeTextureDXT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if w != 64 || h != 64 || len(dxt) != 4096 {
		t.Fatalf("size %dx%d len=%d", w, h, len(dxt))
	}
	sum := sha256.Sum256(dxt)
	got := hex.EncodeToString(sum[:])
	const want = "b7979ed52885d0f15711d0df9a1eaf2b1a38aba6d8c37f79864486e509d7f38a"
	if got != want {
		t.Fatalf("dxt sha %s", got)
	}
}
