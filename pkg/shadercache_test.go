package pkg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shaderPkg(t *testing.T, sub, name string) *Reader {
	t.Helper()
	path := filepath.Join(os.Getenv("APPDATA"), sub, name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no %s", path)
	}
	p, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	return rd
}

func TestMaterialShaderCache(t *testing.T) {
	rd := shaderPkg(t, "miniworldgameguan110", "material_res.pkg")
	st := rd.StatMaterialHdrs()
	if st.Blobs != 15423 || st.Shader != 15322 || st.SHA1FP != 35 || st.Material != 63 || st.Other != 3 {
		t.Fatalf("classes %+v", st)
	}
	for k, v := range map[string]int{"size": st.SizeOK, "name": st.NameOK, "payload": st.PayloadOK, "dxbc": st.DXBCOk, "bind": st.BindOK, "count": st.CountOK} {
		if v != st.Shader {
			t.Fatalf("%s %d != %d", k, v, st.Shader)
		}
	}
	if st.Stages[0]+st.Stages[1] != st.Shader || st.Stages[1] != 5549 {
		t.Fatalf("stages %v", st.Stages)
	}
}

func noExtKinds(t *testing.T, rd *Reader) (map[DXKind]int, int) {
	t.Helper()
	kinds := map[DXKind]int{}
	bad := 0
	for _, n := range rd.Names("") {
		bn := n[strings.LastIndex(n, "/")+1:]
		if strings.Contains(bn, ".") {
			continue
		}
		b, err := rd.Lookup(n)
		if err != nil {
			t.Fatalf("lookup %s: %v", n, err)
		}
		info := ClassifyDXBlob(n, b)
		kinds[info.Kind]++
		if info.Kind == DXKindDXBC && !info.BindOK {
			bad++
		}
	}
	return kinds, bad
}

func TestShaderNoExt(t *testing.T) {
	cases := []struct {
		sub, name    string
		fp, dx, hash int
	}{
		{"miniworldgameguan110", "first_res.pkg", 93, 1739, 1739},
		{"miniworldgameguan110", "engine_res.pkg", 69, 7116, 0},
		{"miniworddata110/pkg_assets", "dx_res.pkg", 93, 69098, 69098},
	}
	for _, c := range cases {
		rd := shaderPkg(t, c.sub, c.name)
		kinds, bad := noExtKinds(t, rd)
		if kinds[DXKindUnknown] != 0 || kinds[DXKindSHA1FP] != c.fp || kinds[DXKindDXBC] != c.dx || bad != 0 {
			t.Fatalf("%s kinds %v bad %d", c.name, kinds, bad)
		}
		hit := 0
		for _, n := range rd.Names("d3d11/") {
			if !isDXHashName(n) {
				continue
			}
			b, _ := rd.Lookup(n)
			if info := ClassifyDXBlob(n, b); info.Hash == strings.ToLower(n[6:]) {
				hit++
			}
		}
		if hit != c.hash {
			t.Fatalf("%s hash hits %d", c.name, hit)
		}
	}
}

func TestDXCompute(t *testing.T) {
	rd := shaderPkg(t, "miniworddata110/pkg_assets", "dx_res.pkg")
	want := map[string]int{"vxgiupdate.compute": 6, "astcencodecs.compute": 3, "ssprcomputeshader.compute": 3, "clearinstancecountcs.compute": 1, "instancecullingcs.compute": 1}
	seen := 0
	for _, n := range rd.Names("") {
		if !strings.HasSuffix(n, ".compute") {
			continue
		}
		seen++
		b, _ := rd.Lookup(n)
		info := ClassifyDXBlob(n, b)
		if info.Kind != DXKindCompute || info.DXCount < 1 {
			t.Fatalf("%s %+v", n, info)
		}
		if w, ok := want[n[strings.LastIndex(n, "/")+1:]]; ok && w != info.DXCount {
			t.Fatalf("%s dxcount %d want %d", n, info.DXCount, w)
		}
	}
	if seen != 6 {
		t.Fatalf("compute entries %d", seen)
	}
}

func TestEngineAssets(t *testing.T) {
	rd := shaderPkg(t, "miniworldgameguan110", "engine_res.pkg")
	objs := rd.EngineObjects()
	if len(objs) != 106 {
		t.Fatalf("objects %d", len(objs))
	}
	typed, marks := 0, 0
	names := map[string]bool{}
	for _, n := range rd.Names("") {
		names[n] = true
	}
	for _, o := range objs {
		if o.Type == 0xd8485716 {
			typed++
		}
		if o.Offset < 0 || !names[o.Name] {
			t.Fatalf("offset %d name %s", o.Offset, o.Name)
		}
		b, _ := rd.Lookup(o.Name)
		if plain, ok := DecodeWrapped(b); ok {
			marks += CountEngineObjectMarks(plain)
		}
	}
	if typed != 56 || marks != 0 {
		t.Fatalf("typed %d marks %d", typed, marks)
	}
	b, err := rd.Lookup("d3d11/shaderidlist.bin")
	if err != nil || len(b) < 8 || bytes.HasPrefix(b[10:], []byte(dxFourCC)) {
		t.Fatalf("shaderidlist %v len %d", err, len(b))
	}
	if n, rec := dxIDListLayout(b); n != 322797 || rec != 40 {
		t.Fatalf("shaderidlist n %d rec %d len %d", n, rec, len(b))
	}
}

func TestFirstResPair(t *testing.T) {
	a := shaderPkg(t, "miniworldgameguan110", "first_res.pkg")
	alt := `F:\Program Files\miniworldLauncher\first_res.pkg`
	if _, err := os.Stat(alt); err != nil {
		t.Skip("no launcher first_res")
	}
	p, err := ParseFile(alt)
	if err != nil {
		t.Fatal(err)
	}
	c, err := OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	na, nc := a.Names(""), c.Names("")
	if len(na) != len(nc) {
		t.Fatalf("names %d %d", len(na), len(nc))
	}
	diff := 0
	for _, n := range na {
		ba, ea := a.Lookup(n)
		bc, ec := c.Lookup(n)
		if ea != nil || ec != nil || len(ba) != len(bc) {
			t.Fatalf("%s %v %v %d %d", n, ea, ec, len(ba), len(bc))
		}
		if !bytes.Equal(ba, bc) {
			diff++
			if !isDXHashName(n) {
				t.Fatalf("diff outside d3d11: %s", n)
			}
		}
	}
	if diff != 14 {
		t.Fatalf("diff entries %d", diff)
	}
}
