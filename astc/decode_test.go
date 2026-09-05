package astc

import (
	"bytes"
	"image/color"
	"testing"
)

func setBits(blk []byte, off, n int, v uint32) {
	for i := 0; i < n; i++ {
		p := off + i
		if v>>uint(i)&1 != 0 {
			blk[p>>3] |= 1 << uint(p&7)
		}
	}
}

func voidBlock(r, g, b, a uint16) []byte {
	blk := []byte{0xfc, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}
	for i, v := range []uint16{r, g, b, a} {
		blk[8+2*i] = byte(v)
		blk[9+2*i] = byte(v >> 8)
	}
	return blk
}

func TestDecodeBlockVoidExtent(t *testing.T) {
	out := make([]byte, 64)
	if err := DecodeBlock(voidBlock(0x1234, 0x5678, 0x9abc, 0xdef0), 4, 4, out); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		if got := out[i*4 : i*4+4]; !bytes.Equal(got, []byte{0x12, 0x56, 0x9a, 0xde}) {
			t.Fatalf("texel %d = %v", i, got)
		}
	}
	hdr := voidBlock(0, 0, 0, 0)
	hdr[1] |= 0x02
	if err := DecodeBlock(hdr, 4, 4, out); err == nil {
		t.Fatal("HDR void extent accepted")
	}
	bad := voidBlock(0, 0, 0, 0)
	bad[1] &^= 0x04
	if err := DecodeBlock(bad, 4, 4, out); err == nil {
		t.Fatal("reserved bits accepted")
	}
}

func syntheticRGB(cem uint32) []byte {
	blk := make([]byte, 16)
	setBits(blk, 0, 11, 0x042)
	setBits(blk, 13, 4, cem)
	for i, v := range []uint32{0x10, 0xf0, 0x20, 0xe0, 0x30, 0xd0} {
		setBits(blk, 17+8*i, 8, v)
	}
	for k := 0; k < 16; k++ {
		w := uint32(k & 3)
		setBits(blk, 127-2*k, 1, w&1)
		setBits(blk, 126-2*k, 1, w>>1)
	}
	return blk
}

func TestDecodeBlockSyntheticRGB(t *testing.T) {
	out := make([]byte, 64)
	if err := DecodeBlock(syntheticRGB(8), 4, 4, out); err != nil {
		t.Fatal(err)
	}
	e0 := [4]int{0x10, 0x20, 0x30, 255}
	e1 := [4]int{0xf0, 0xe0, 0xd0, 255}
	weights := [4]int{0, 21, 43, 64}
	for k := 0; k < 16; k++ {
		w := weights[k&3]
		for c := 0; c < 4; c++ {
			want := byte((e0[c]*257*(64-w) + e1[c]*257*w + 32) >> 14)
			if out[k*4+c] != want {
				t.Fatalf("texel %d ch %d = %d want %d", k, c, out[k*4+c], want)
			}
		}
	}
	if err := DecodeBlock(syntheticRGB(11), 4, 4, out); err == nil {
		t.Fatal("HDR endpoint mode accepted")
	}
	if err := DecodeBlock(make([]byte, 16), 4, 4, out); err == nil {
		t.Fatal("reserved block mode accepted")
	}
}

func TestTritQuintUnpack(t *testing.T) {
	trits := map[uint32][5]int{
		3: {0, 0, 2, 0, 0}, 7: {1, 0, 2, 0, 0}, 28: {0, 0, 0, 2, 2},
		91: {2, 1, 2, 2, 0}, 123: {2, 1, 2, 0, 2}, 255: {2, 1, 2, 2, 2},
	}
	for in, want := range trits {
		if got := tritsOf(in); got != want {
			t.Errorf("trits %d = %v want %v", in, got, want)
		}
	}
	quints := map[uint32][3]int{5: {0, 4, 0}, 6: {4, 4, 0}, 7: {4, 4, 4}, 38: {4, 0, 4}, 127: {1, 3, 4}}
	for in, want := range quints {
		if got := quintsOf(in); got != want {
			t.Errorf("quints %d = %v want %v", in, got, want)
		}
	}
}

func TestDecodeISE(t *testing.T) {
	var b [16]byte
	m := []uint32{1, 0, 1, 1, 0}
	tb := []struct {
		n     int
		shift uint
	}{{2, 0}, {2, 2}, {1, 4}, {2, 5}, {1, 7}}
	off := 0
	for i := 0; i < 5; i++ {
		setBits(b[:], off, 1, m[i])
		off++
		setBits(b[:], off, tb[i].n, 91>>tb[i].shift)
		off += tb[i].n
	}
	got := make([]int, 5)
	decodeISE(b[:], 0, 5, quant6, got)
	if want := []int{5, 2, 5, 5, 0}; !equalInts(got, want) {
		t.Fatalf("trit ise %v want %v", got, want)
	}
	var q [16]byte
	setBits(q[:], 0, 7, 38)
	got = make([]int, 3)
	decodeISE(q[:], 0, 3, quant5, got)
	if want := []int{4, 0, 4}; !equalInts(got, want) {
		t.Fatalf("quint ise %v want %v", got, want)
	}
	if unquantWeight(4, quant5) != 64 || unquantWeight(3, quant5) != 48 || unquantWeight(5, quant6) != 39 {
		t.Fatal("weight unquant")
	}
	if unquantColor(3, quant6) != 204 || unquantColor(5, quant12) != 232 || unquantColor(9, quant10) != 142 {
		t.Fatal("color unquant")
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDecodeCrop(t *testing.T) {
	var data []byte
	colors := []color.NRGBA{{10, 20, 30, 255}, {40, 50, 60, 255}, {70, 80, 90, 255}, {100, 110, 120, 255}}
	for _, c := range colors {
		data = append(data, voidBlock(uint16(c.R)<<8, uint16(c.G)<<8, uint16(c.B)<<8, 0xffff)...)
	}
	img, err := Decode(data, 6, 5, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 6 || img.Bounds().Dy() != 5 {
		t.Fatalf("bounds %v", img.Bounds())
	}
	for _, c := range []struct{ x, y, idx int }{{3, 0, 0}, {4, 0, 1}, {0, 4, 2}, {5, 4, 3}} {
		if got := img.NRGBAAt(c.x, c.y); got != colors[c.idx] {
			t.Fatalf("pixel (%d,%d) = %v want %v", c.x, c.y, got, colors[c.idx])
		}
	}
	if _, err := Decode(data[:63], 6, 5, 4, 4); err == nil {
		t.Fatal("short data accepted")
	}
}

func TestDecodeBlockGolden(t *testing.T) {
	cases := []struct {
		name string
		blk  []byte
		want []byte
	}{
		{"single", []byte{0x43, 0x82, 0x02, 0x6a, 0x01, 0x02, 0x40, 0x4f, 0x00, 0xbd, 0x02, 0x80, 0x01, 0x00, 0x00, 0x00}, []byte{
			0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00,
			0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x0f, 0x0f, 0x0f, 0x00, 0x0f, 0x0f, 0x0f, 0x00,
			0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x77, 0x77, 0x77, 0x00, 0x61, 0x61, 0x61, 0x00,
			0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0xb0, 0xb0, 0xb0, 0x00, 0x61, 0x61, 0x61, 0x00,
		}},
		{"two", []byte{0x53, 0xc8, 0x5f, 0x01, 0xb8, 0x00, 0x00, 0x28, 0x00, 0x40, 0x44, 0x26, 0xa0, 0x06, 0x30, 0x00}, []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x08, 0x08, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x27, 0x27, 0x27, 0x00,
			0x07, 0x07, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1a, 0x1a, 0x1a, 0x00,
			0x50, 0x50, 0x50, 0x00, 0x05, 0x05, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00,
		}},
		{"three", []byte{0x23, 0x90, 0x28, 0x0d, 0x0a, 0x10, 0x0b, 0x00, 0x9e, 0x77, 0x00, 0xa0, 0x72, 0x88, 0x01, 0x01}, []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2b, 0x2b, 0x2b, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x3d, 0x3d, 0x3d, 0x00, 0xae, 0xae, 0xae, 0x00, 0x1d, 0x1d, 0x1d, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x54, 0x1e, 0x54, 0x00, 0xef, 0x56, 0xef, 0x00, 0x30, 0x11, 0x30, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x30, 0x11, 0x30, 0x00, 0x81, 0x00, 0x81, 0x00, 0x61, 0x00, 0x61, 0x00,
		}},
		{"dual", []byte{0x51, 0xec, 0x04, 0x0a, 0x73, 0x01, 0x00, 0x00, 0x00, 0x00, 0x7f, 0x06, 0x00, 0x00, 0x00, 0x00}, []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x21, 0x21, 0x21, 0x00, 0x21, 0x21, 0x21, 0x00, 0x21, 0x21, 0x21, 0x00, 0x23, 0x23, 0x23, 0x00,
		}},
	}
	out := make([]byte, 64)
	for _, c := range cases {
		if err := DecodeBlock(c.blk, 4, 4, out); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !bytes.Equal(out, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, out, c.want)
		}
	}
}
