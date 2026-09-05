package pkg

import (
	"archive/zip"
	"bytes"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCommonRes(t *testing.T) (*Pkg, *Reader) {
	t.Helper()
	path := filepath.Join(testPkgDir(t), "common_res.pkg")
	if _, err := os.Stat(path); err != nil {
		t.Skip("no common_res")
	}
	p, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	return p, r
}

func TestZipPlainCoversRaw(t *testing.T) {
	p, r := testCommonRes(t)
	raw := ScanZipEntries(p.Data)
	plain, err := r.ScanZipPlain()
	if err != nil {
		t.Fatal(err)
	}
	type key struct {
		name string
		crc  uint32
		comp uint32
	}
	have := map[key]int{}
	for _, e := range plain {
		have[key{e.Name, e.CRC32, e.CompSize}]++
	}
	clean, junk := 0, 0
	for _, e := range raw {
		if !printableName(e.Name) {
			junk++
			continue
		}
		clean++
		if have[key{e.Name, e.CRC32, e.CompSize}] == 0 {
			t.Errorf("raw entry missing in plain scan: %s", e.Name)
		}
	}
	if len(plain) < clean {
		t.Fatalf("plain=%d < clean raw=%d", len(plain), clean)
	}
	t.Logf("raw=%d (clean %d, junk %d) plain=%d", len(raw), clean, junk, len(plain))
}

func printableName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

func TestZipPlainLuaCRC(t *testing.T) {
	_, r := testCommonRes(t)
	ents, err := r.ScanZipPlain()
	if err != nil {
		t.Fatal(err)
	}
	n, enc, bad := 0, 0, 0
	for _, e := range ents {
		if !strings.HasSuffix(strings.ToLower(e.Name), ".lua") {
			continue
		}
		if e.Flag&1 != 0 {
			enc++
			continue
		}
		n++
		body, err := r.ExtractZipPlain(e, ZipKeys{})
		if err != nil {
			bad++
			t.Errorf("%s: %v", e.Name, err)
			continue
		}
		if lenOK, crcOK := VerifyZipEntry(body, e); !lenOK || !crcOK {
			bad++
			t.Errorf("%s: len=%v crc=%v", e.Name, lenOK, crcOK)
		}
	}
	if n == 0 || bad != 0 {
		t.Fatalf("lua plain=%d encrypted=%d bad=%d", n, enc, bad)
	}
	t.Logf("lua plain=%d encrypted=%d", n, enc)
}

func TestZipPlainTinyBlocks(t *testing.T) {
	bodies := map[string][]byte{}
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	for i, name := range []string{"a/one.lua", "bb/two.png_", "ccc/dir/three.txt"} {
		body := bytes.Repeat([]byte{byte('a' + i)}, 300+i*37)
		bodies[name] = body
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	comp, err := RawDeflate(bytes.Repeat([]byte("fixed"), 80))
	if err != nil {
		t.Fatal(err)
	}
	fixed := bytes.Repeat([]byte("fixed"), 80)
	bodies["d/four.bin"] = fixed
	fw, err := zw.CreateRaw(&zip.FileHeader{
		Name: "d/four.bin", Method: zip.Deflate, CRC32: crc32.ChecksumIEEE(fixed),
		CompressedSize64: uint64(len(comp)), UncompressedSize64: uint64(len(fixed)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(comp); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	pool := zbuf.Bytes()
	for _, bs := range []int{7, 64, 4096, len(pool)} {
		r := fakeStoredReader(pool, bs)
		ents, err := r.ScanZipPlain()
		if err != nil {
			t.Fatalf("bs=%d: %v", bs, err)
		}
		if len(ents) != len(bodies) {
			t.Fatalf("bs=%d entries=%d want %d", bs, len(ents), len(bodies))
		}
		for _, e := range ents {
			got, err := r.ExtractZipPlain(e, ZipKeys{})
			if err != nil {
				t.Fatalf("bs=%d %s: %v", bs, e.Name, err)
			}
			if !bytes.Equal(got, bodies[e.Name]) {
				t.Fatalf("bs=%d %s: body mismatch len=%d flag=%#x", bs, e.Name, len(got), e.Flag)
			}
		}
	}
}

func fakeStoredReader(pool []byte, bs int) *Reader {
	idx := &pkgIndex{byName: map[string]uint32{}, uncompAt: []uint64{uncompOrigin}, compAt: []uint64{0}}
	for off := 0; off < len(pool); off += bs {
		n := bs
		if off+n > len(pool) {
			n = len(pool) - off
		}
		idx.stor = append(idx.stor, storRec{uncomp: uint32(n), comp: uint32(n)})
		idx.uncompAt = append(idx.uncompAt, idx.uncompAt[len(idx.uncompAt)-1]+uint64(n))
		idx.compAt = append(idx.compAt, idx.compAt[len(idx.compAt)-1]+uint64(n))
	}
	return &Reader{data: pool, idx: idx, cache: map[int][]byte{}, lower: map[string]string{}}
}
