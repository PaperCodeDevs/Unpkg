package pkg

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestRainbowNameHash(t *testing.T) {
	if XXH32(nil, 0) != 0x02CC5D05 {
		t.Fatalf("xxh32 empty %08x", XXH32(nil, 0))
	}
	for name, want := range map[string]uint32{"ViewCB": 0x1BB89DD0, "PrimitiveCB": 0x6298D38A} {
		if got := RainbowNameHash(name); got != want {
			t.Fatalf("%s %08x != %08x", name, got, want)
		}
		if EngineCBName(want) != name {
			t.Fatalf("%08x -> %q", want, EngineCBName(want))
		}
	}
}

func TestMaterialCBUnits(t *testing.T) {
	rd := shaderPkg(t, "miniworldgameguan110", "material_res.pkg")
	shaders, units, named, matched := 0, 0, 0, 0
	sizes := map[string]map[uint32]int{}
	for _, n := range rd.Names("") {
		if !strings.Contains(n, "d3d11/") {
			continue
		}
		b, err := rd.Lookup(n)
		if err != nil {
			t.Fatal(err)
		}
		sh, err := SplitMaterialShader(b)
		if err != nil {
			continue
		}
		bd, err := ParseMaterialBinding(sh.Extra)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		shaders++
		for _, u := range bd.CBUnits() {
			units++
			if u.Name != "" {
				named++
				if sizes[u.Name] == nil {
					sizes[u.Name] = map[uint32]int{}
				}
				sizes[u.Name][u.Size]++
			}
			if cb, ok := bd.CBufferOf(u); ok && cb.Name == u.Name {
				matched++
			}
		}
	}
	if shaders != 15322 || units != 10877 || named != units || matched != units {
		t.Fatalf("shaders %d units %d named %d matched %d", shaders, units, named, matched)
	}
	if len(sizes["ViewCB"]) != 1 || sizes["ViewCB"][320] == 0 || len(sizes["PrimitiveCB"]) != 1 || sizes["PrimitiveCB"][144] == 0 {
		t.Fatalf("sizes %v", sizes)
	}
}

func pipelinePkg(t *testing.T, sub, name string) (*ShaderPipelineMap, map[[20]byte][20]byte, *Reader) {
	t.Helper()
	rd := shaderPkg(t, sub, name)
	raw, err := rd.Lookup("effectshaderpipelinemap.bin")
	if err != nil {
		t.Skipf("%s: no effectshaderpipelinemap.bin", name)
	}
	m, err := ParseShaderPipelineMap(raw)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	idl, err := rd.Lookup("d3d11/shaderidlist.bin")
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	ids, err := ParseShaderIDList(idl)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return m, ids, rd
}

func checkPipelineIDs(t *testing.T, m *ShaderPipelineMap, ids map[[20]byte][20]byte, rd *Reader) int {
	t.Helper()
	seen := map[[20]byte]bool{}
	for _, p := range m.Pipelines {
		if p.Name == "" || !strings.HasSuffix(p.Name, "Pipeline") {
			t.Fatalf("pipeline name %q", p.Name)
		}
		for _, st := range p.Stages {
			if st.Type != 1 && st.Type != 2 {
				t.Fatalf("%s: stage type %d", p.Name, st.Type)
			}
			if seen[st.ShaderID] {
				continue
			}
			seen[st.ShaderID] = true
			dst, ok := ids[st.ShaderID]
			if !ok {
				t.Fatalf("%s: shader id %x not in idlist", p.Name, st.ShaderID[:4])
			}
			if _, err := rd.Lookup("d3d11/" + hex.EncodeToString(dst[:])); err != nil {
				t.Fatalf("%s: file id %x missing", p.Name, dst[:4])
			}
		}
	}
	return len(seen)
}

func TestShaderPipelineMapDX(t *testing.T) {
	m, ids, rd := pipelinePkg(t, "miniworddata110/pkg_assets", "dx_res.pkg")
	if len(m.Pipelines) != 646 || len(m.TypeNames) != 42 || len(m.LayoutNames) != 7 || len(m.KeywordSets) != 350 || len(m.ShaderIDs) != 950 {
		t.Fatalf("counts %d %d %d %d %d", len(m.Pipelines), len(m.TypeNames), len(m.LayoutNames), len(m.KeywordSets), len(m.ShaderIDs))
	}
	if n := checkPipelineIDs(t, m, ids, rd); n != 950 {
		t.Fatalf("distinct ids %d", n)
	}
	if m.Pipelines[1].Name != "PreZPassPositionOnlyPipeline" || m.Pipelines[1].VertexLayout != "BlockVertexLayout" || m.Pipelines[0].VertexLayout != "" {
		t.Fatalf("pipelines %+v %+v", m.Pipelines[0], m.Pipelines[1])
	}
	for _, p := range m.Pipelines {
		if p.VertexLayout != "" && !strings.HasSuffix(p.VertexLayout, "VertexLayout") {
			t.Fatalf("layout %q", p.VertexLayout)
		}
	}
}

func TestShaderPipelineMapFirst(t *testing.T) {
	m, ids, rd := pipelinePkg(t, "miniworldgameguan110", "first_res.pkg")
	if len(m.Pipelines) != 646 || len(m.TypeNames) != 42 || len(m.LayoutNames) != 7 || len(m.KeywordSets) != 350 || len(m.ShaderIDs) != 950 {
		t.Fatalf("counts %d %d %d %d %d", len(m.Pipelines), len(m.TypeNames), len(m.LayoutNames), len(m.KeywordSets), len(m.ShaderIDs))
	}
	if n := checkPipelineIDs(t, m, ids, rd); n != 950 {
		t.Fatalf("distinct ids %d", n)
	}
}

func TestShaderPipelineMapEngine(t *testing.T) {
	m, ids, rd := pipelinePkg(t, "miniworldgameguan110", "engine_res.pkg")
	if len(m.Pipelines) != 124 || len(m.TypeNames) != 29 || len(m.LayoutNames) != 0 || len(m.KeywordSets) != 124 || len(m.ShaderIDs) != 172 {
		t.Fatalf("counts %d %d %d %d %d", len(m.Pipelines), len(m.TypeNames), len(m.LayoutNames), len(m.KeywordSets), len(m.ShaderIDs))
	}
	stages := 0
	for _, p := range m.Pipelines {
		stages += len(p.Stages)
		for _, st := range p.Stages {
			c, ok := EngineShaderCodeOf(st.Name)
			if !ok || uint32(c.Type) != st.Type {
				t.Fatalf("%s: stage %q type %d", p.Name, st.Name, st.Type)
			}
		}
	}
	if stages != 248 {
		t.Fatalf("stages %d", stages)
	}
	if n := checkPipelineIDs(t, m, ids, rd); n != 172 {
		t.Fatalf("distinct ids %d", n)
	}
	if stageType("FooVS") != ShaderStageVS || stageType("FooPS") != ShaderStagePS || stageType("Foo") != 0 {
		t.Fatal("stageType suffix fallback")
	}
}

func TestEngineShaderSources(t *testing.T) {
	rd := shaderPkg(t, "miniworddata110/pkg_assets", "dx_res.pkg")
	byFile := map[string]map[string]int{}
	found := 0
	for _, n := range rd.Names("") {
		bn := n[strings.LastIndex(n, "/")+1:]
		if strings.Contains(bn, ".") || strings.HasPrefix(n, "d3d11/") {
			continue
		}
		c, ok := EngineShaderCodeOf(bn)
		if !ok {
			continue
		}
		b, err := rd.Lookup(n)
		if err != nil || len(b) != 24 {
			t.Fatalf("%s: %d bytes %v", n, len(b), err)
		}
		found++
		if byFile[c.File] == nil {
			byFile[c.File] = map[string]int{}
		}
		byFile[c.File][hex.EncodeToString(b[4:])]++
	}
	if found < 80 || len(EngineShaderCodes()) != 84 {
		t.Fatalf("found %d of %d", found, len(EngineShaderCodes()))
	}
	fps := map[string]string{}
	for file, set := range byFile {
		if len(set) != 1 {
			t.Fatalf("%s: %d fingerprints", file, len(set))
		}
		for fp := range set {
			if other, dup := fps[fp]; dup {
				t.Fatalf("%s and %s share %s", file, other, fp[:8])
			}
			fps[fp] = file
		}
	}
}

func TestShaderPipelineMapMaterial(t *testing.T) {
	rd := shaderPkg(t, "miniworldgameguan110", "material_res.pkg")
	raw, err := rd.Lookup("effectshaderpipelinemap.bin")
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseShaderPipelineMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Pipelines) != 10 || len(m.ShaderIDs) != 20 || m.Pipelines[0].Name != "GUIMainPipeline" || m.Pipelines[0].Stages[0].Name != "GUIMainVS" {
		t.Fatalf("pipelines %d ids %d first %+v", len(m.Pipelines), len(m.ShaderIDs), m.Pipelines[0])
	}
	names := map[string]bool{}
	for _, n := range rd.Names("") {
		names[n[strings.LastIndex(n, "/")+1:]] = true
	}
	for _, id := range m.ShaderIDs {
		if !names[hex.EncodeToString(id[:])] {
			t.Fatalf("file id %x missing", id[:4])
		}
	}
}
