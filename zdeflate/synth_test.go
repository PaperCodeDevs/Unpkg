package zdeflate

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func inflateRaw(t *testing.T, comp, want []byte) {
	t.Helper()
	r := flate.NewReader(bytes.NewReader(comp))
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("inflate: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("inflate mismatch len %d want %d", len(got), len(want))
	}
}

func TestSynthRoundtrip(t *testing.T) {
	cases := [][]byte{
		{},
		{0},
		{255},
		[]byte("a"),
		[]byte("hello zlib level nine"),
		bytes.Repeat([]byte{0}, 64<<10),
		bytes.Repeat([]byte("abcXYZ\n"), 4000),
	}
	for i, src := range cases {
		comp := Compress(src)
		inflateRaw(t, comp, src)
		if i == 0 && len(comp) == 0 {
			t.Fatalf("empty produced no bytes")
		}
	}
	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = byte(i * 3)
	}
	comp := Compress(big)
	inflateRaw(t, comp, big)
}

func TestRepeatTextLen(t *testing.T) {
	src := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog\n"), 80)
	comp := Compress(src)
	inflateRaw(t, comp, src)
	if len(comp) >= len(src) {
		t.Fatalf("repeat not compressed %d >= %d", len(comp), len(src))
	}
}
