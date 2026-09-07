package zdeflate

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func zlibOracle(t *testing.T) string {
	t.Helper()
	p := filepath.Join(os.TempDir(), "zlib127", "oracle.exe")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func oracleCompress(t *testing.T, exe string, src []byte) []byte {
	t.Helper()
	cmd := exec.Command(exe)
	cmd.Stdin = bytes.NewReader(src)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	return out
}

func TestMatchZlib127(t *testing.T) {
	exe := zlibOracle(t)
	if exe == "" {
		t.Skip("no zlib127 oracle")
	}
	cases := [][]byte{
		{},
		{0},
		{255},
		[]byte("a"),
		[]byte("hello zlib level nine"),
		bytes.Repeat([]byte{0}, 64<<10),
		bytes.Repeat([]byte("abcXYZ\n"), 4000),
		bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog\n"), 80),
	}
	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = byte(i * 3)
	}
	cases = append(cases, big)
	for i, src := range cases {
		want := oracleCompress(t, exe, src)
		got := Compress(src)
		if !bytes.Equal(got, want) {
			n := len(got)
			if len(want) < n {
				n = len(want)
			}
			off := 0
			for off < n && got[off] == want[off] {
				off++
			}
			t.Fatalf("case %d len(src)=%d got=%d want=%d firstDiff=%d", i, len(src), len(got), len(want), off)
		}
	}
}
