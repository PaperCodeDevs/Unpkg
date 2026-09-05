package pkg

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestJSONLenientCommentsAndTrailingComma(t *testing.T) {
	src := "\xEF\xBB\xBF{\r\n  // note\r\n  \"a\": [1, 2, /* x */],\r\n  \"b\": \"//not comment\",\r\n}"
	info, err := VerifyPlain("x.json", []byte(src), PlainOpt{})
	if err != nil || info.Kind != PlainJSON || info.Detail != "lenient" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if _, err := VerifyPlain("bad.json", []byte("{\"a\":}"), PlainOpt{}); err == nil {
		t.Fatal("expected invalid json")
	}
}

func TestXMLCharsetGB2312(t *testing.T) {
	raw, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("<?xml version=\"1.0\" encoding=\"gb2312\"?><a><b>中文</b></a>"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := VerifyPlain("a.xml", raw, PlainOpt{})
	if err != nil || info.Count != 2 {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestCSVGB18030AndLua(t *testing.T) {
	raw, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("ID,名称\n1,石头\n"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := VerifyPlain("animmap.csv", raw, PlainOpt{})
	if err != nil || info.Detail != "gb18030" || info.Count != 2 {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if k := ClassifyPlain("playermodule.bil", []byte("if not CommonModule then\n  return\nend\n")); k != PlainLua {
		t.Fatalf("kind=%s", k)
	}
	if !IsLuaText([]byte("if not CommonModule then\n  require('x')\nend\n")) {
		t.Fatal("IsLuaText")
	}
}

func pbString(field int, s string) []byte {
	out := binary.AppendUvarint(nil, uint64(field<<3|2))
	out = binary.AppendUvarint(out, uint64(len(s)))
	return append(out, s...)
}

func TestPBDescriptorMessages(t *testing.T) {
	msg := pbString(1, "Foo")
	file := append(pbString(1, "a.proto"), pbString(2, "ugc")...)
	file = append(file, pbString(4, string(msg))...)
	file = append(file, pbString(4, string(pbString(1, "Bar")))...)
	set := pbString(1, string(file))
	info, err := VerifyPlain("x.pb", set, PlainOpt{})
	if err != nil || strings.Join(info.Names, ",") != "ugc.Foo,ugc.Bar" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestGIFZipTimelineServerList(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	var gbuf bytes.Buffer
	if err := gif.EncodeAll(&gbuf, &gif.GIF{Image: []*image.Paletted{img, img}, Delay: []int{1, 1}}); err != nil {
		t.Fatal(err)
	}
	if info, err := VerifyPlain("a.gif", gbuf.Bytes(), PlainOpt{}); err != nil || info.Count != 2 {
		t.Fatalf("gif info=%+v err=%v", info, err)
	}
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	for _, n := range []string{"a.txt", "b.txt"} {
		data := bytes.Repeat([]byte(n), 40)
		w, err := zw.CreateRaw(&zip.FileHeader{Name: n, Method: zip.Store, CRC32: crc32.ChecksumIEEE(data), CompressedSize64: uint64(len(data)), UncompressedSize64: uint64(len(data))})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(data)
	}
	_ = zw.Close()
	if info, err := VerifyPlain("c.zip", zbuf.Bytes(), PlainOpt{}); err != nil || info.Count != 2 || info.Names[1] != "b.txt" {
		t.Fatalf("zip info=%+v err=%v", info, err)
	}
	tl := binary.LittleEndian.AppendUint32(nil, 2)
	tl = binary.LittleEndian.AppendUint32(tl, 0xac031125)
	tl = binary.LittleEndian.AppendUint16(tl, 1)
	tl = binary.LittleEndian.AppendUint32(tl, 0xf019cc75)
	tl = binary.LittleEndian.AppendUint16(tl, 3)
	tl = binary.LittleEndian.AppendUint32(tl, 2007)
	if info, err := VerifyPlain("tl1.asset", tl, PlainOpt{}); err != nil || info.Kind != PlainTimeline || info.Count != 2 || info.Detail != "type=2007" {
		t.Fatalf("timeline info=%+v err=%v", info, err)
	}
	sl := append([]byte{0x5a, 0x12, 0xbe, 0x34, 0xc0, 0x18, 0xc3, 0x15}, make([]byte, 8)...)
	if info, err := VerifyPlain("serverlist.data", sl, PlainOpt{}); !errors.Is(err, ErrCipherUnknown) || info.Kind != PlainServerList {
		t.Fatalf("serverlist info=%+v err=%v", info, err)
	}
	rk := RainbowKeys{Magic: "Rainbow\x00", XORKey: []byte{4, 8, 10, 26, 13, 29, 14, 0}}
	body := append([]byte(rk.Magic), 0x89^4, 0x50^8, 0x4e^10, 0x47^26, 0x0d^13, 0x0a^29, 0x1a^14, 0x0a^0)
	if info, err := VerifyPlain("x.png_", body, PlainOpt{Rainbow: rk}); err != nil || info.Kind != PlainRainbow {
		t.Fatalf("rainbow info=%+v err=%v", info, err)
	}
}

func TestVerifyPlainRealPkgs(t *testing.T) {
	appdata := os.Getenv("APPDATA")
	paths := []string{
		filepath.Join(appdata, "miniworldgameguan110", "first_res.pkg"),
		filepath.Join(appdata, "miniworddata110", "pkg_assets", "patch_game_script.pkg"),
		filepath.Join(appdata, "miniworddata110", "pkg_assets", "patch_common_res.pkg"),
	}
	opt := PlainOpt{
		Zip:     ZipKeys{K0: 0x1f85d854, K1: 0xdbaf3374, K2: 0xc4d7be09},
		Rainbow: RainbowKeys{Magic: "Rainbow\x00", XORKey: []byte{4, 8, 10, 26, 13, 29, 14, 0}},
	}
	skip := map[PlainKind]bool{PlainUnknown: true, PlainEmpty: true}
	seen := 0
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		rd, err := OpenOverlayFiles(p, "")
		if err != nil {
			t.Fatal(err)
		}
		seen++
		total := map[PlainKind]int{}
		for _, n := range rd.Names("") {
			body, err := rd.Lookup(n)
			if err != nil {
				continue
			}
			k := ClassifyPlain(n, body)
			if skip[k] {
				continue
			}
			total[k]++
			if _, err := VerifyPlain(n, body, opt); err != nil && !(k == PlainServerList && errors.Is(err, ErrCipherUnknown)) {
				t.Errorf("%s %s: %v", filepath.Base(p), n, err)
			}
		}
		t.Logf("%s kinds=%v", filepath.Base(p), total)
	}
	if seen == 0 {
		t.Skip("no pkg files")
	}
}
