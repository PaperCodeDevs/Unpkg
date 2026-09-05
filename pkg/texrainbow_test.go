package pkg

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testEngineResPath(t *testing.T) string {
	t.Helper()
	cands := []string{
		filepath.Join(os.Getenv("APPDATA"), "miniworldgameguan110", "engine_res.pkg"),
		filepath.Join(DefaultLauncherRoot(), "engine_res.pkg"),
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no engine_res.pkg")
	return ""
}

func TestIsRainbowTexReject(t *testing.T) {
	if IsRainbowTex(nil) || IsRainbowTex(make([]byte, 94)) {
		t.Fatal("short accepted")
	}
	b := make([]byte, 200)
	binary.LittleEndian.PutUint32(b[8:], 0x59A21C2C)
	if IsRainbowTex(b) {
		t.Fatal("unity magic accepted")
	}
	binary.LittleEndian.PutUint32(b[8:], rbTexHash)
	if !IsRainbowTex(b) {
		t.Fatal("rainbow hash rejected")
	}
	if _, err := DecodeRainbowTex(b); err == nil {
		t.Fatal("zero size decoded")
	}
	binary.LittleEndian.PutUint32(b[14:], 4)
	binary.LittleEndian.PutUint32(b[18:], 4)
	binary.LittleEndian.PutUint32(b[22:], 100)
	binary.LittleEndian.PutUint32(b[91:], 100)
	binary.LittleEndian.PutUint32(b[26:], 3)
	_, err := DecodeRainbowTex(b)
	if err == nil || !strings.Contains(err.Error(), "fmt 3") {
		t.Fatalf("want fmt error, got %v", err)
	}
}

func TestRainbowTexEngineRes(t *testing.T) {
	p, err := ParseFile(testEngineResPath(t))
	if err != nil {
		t.Fatal(err)
	}
	rd, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]int{
		"resources/ministudio/ui/icon_chat.png":                           {32, 32},
		"resources/ministudio/entity/player/player01/body/malebody01.png": {256, 256},
		"systemdefault/textures/default_env.png":                          {4096, 4096},
		"systemdefault/textures/area_tex.png":                             {128, 512},
		"systemdefault/dotted.png":                                        {16, 8},
	}
	n := 0
	for _, name := range rd.Names("") {
		if !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}
		n++
		raw, err := rd.Lookup(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !IsRainbowTex(raw) {
			t.Fatalf("%s: not rainbow tex", name)
		}
		h, err := ParseRainbowTex(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if h.Format != rbTexFmtCRN || h.Levels < 1 || h.SrcWidth < 1 || h.SrcHeight < 1 {
			t.Fatalf("%s: header %+v", name, h)
		}
		img, err := DecodeRainbowTex(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if img.Bounds().Dx() != h.Width || img.Bounds().Dy() != h.Height {
			t.Fatalf("%s: got %v want %dx%d", name, img.Bounds(), h.Width, h.Height)
		}
		if wh, ok := want[name]; ok && (wh[0] != h.Width || wh[1] != h.Height) {
			t.Fatalf("%s: %dx%d want %dx%d", name, h.Width, h.Height, wh[0], wh[1])
		}
		if _, err := ParseTexHeader(raw); err == nil {
			t.Fatalf("%s: unity header accepted", name)
		}
	}
	if n != 20 {
		t.Fatalf("png entries %d want 20", n)
	}
}
