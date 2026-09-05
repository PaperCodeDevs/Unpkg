package pkg

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBlockMeshStreams(t *testing.T) {
	dir := testPkgDir(t)
	base := filepath.Join(dir, "common_res.pkg")
	patch := filepath.Join(dir, "patch_common_res.pkg")
	if _, err := os.Stat(base); err != nil {
		t.Skip("no common_res")
	}
	if _, err := os.Stat(patch); err != nil {
		patch = ""
	}
	rd, err := OpenOverlayFiles(base, patch)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name                 string
		streams, nvert, nidx int
	}{
		{"countryflowerpot_2_base.blockmesh", 2, 266, 924},
		{"guomugs_base.blockmesh", 2, 180, 588},
		{"stone_core_base1.blockmesh", 2, 780, 1320},
		{"countryflowerpot_1_base.blockmesh", 1, 156, 408},
	}
	for _, c := range cases {
		raw, err := rd.Lookup(minigameBlocks + c.name)
		if err != nil {
			t.Fatal(err)
		}
		if got := int(binary.LittleEndian.Uint32(raw[0x1C:])); got != c.streams {
			t.Fatalf("%s streams %d", c.name, got)
		}
		m, err := ParseBlockMesh(raw)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		checkMeshShape(t, c.name, m, c.nvert, c.nidx)
	}
}

func checkMeshShape(t *testing.T, name string, m *BlockMesh, nvert, nidx int) {
	t.Helper()
	if len(m.Pos) != nvert*3 || len(m.UV) != nvert*2 || len(m.Idx) != nidx || nidx%3 != 0 {
		t.Fatalf("%s pos=%d uv=%d idx=%d", name, len(m.Pos), len(m.UV), len(m.Idx))
	}
	for _, v := range m.Idx {
		if int(v) >= nvert {
			t.Fatalf("%s idx %d >= %d", name, v, nvert)
		}
	}
	for i, v := range m.Pos {
		if v < -1 || v > 2 {
			t.Fatalf("%s pos[%d]=%g", name, i, v)
		}
	}
	for i, v := range m.UV {
		if v < -0.01 || v > 1.01 {
			t.Fatalf("%s uv[%d]=%g", name, i, v)
		}
	}
}
