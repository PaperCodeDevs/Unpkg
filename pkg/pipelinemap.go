package pkg

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	ShaderStageVS uint32 = 1
	ShaderStagePS uint32 = 2
)

type ShaderStage struct {
	ShaderID [20]byte
	Type     uint32
	Name     string
}

type ShaderPipeline struct {
	Name         string
	VertexLayout string
	Keywords     [32]byte
	Stages       []ShaderStage
}

type ShaderPipelineMap struct {
	Pipelines    []ShaderPipeline
	TypeNames    []string
	LayoutNames  []string
	KeywordSets  [][32]byte
	ShaderIDs    [][20]byte
	TypeIndex    []int32
	LayoutIndex  []int32
	KeywordIndex []int32
	StageIDIndex [][]uint32
}

type pipeReader struct {
	b   []byte
	pos int
	err error
}

func (r *pipeReader) u32() uint32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.b) {
		r.err = fmt.Errorf("pipeline map: truncated at %d", r.pos)
		return 0
	}
	v := binary.LittleEndian.Uint32(r.b[r.pos:])
	r.pos += 4
	return v
}

func (r *pipeReader) count(lim int) int {
	n := int(r.u32())
	if r.err == nil && (n < 0 || n > lim) {
		r.err = fmt.Errorf("pipeline map: count %d exceeds %d", n, lim)
		return 0
	}
	return n
}

func (r *pipeReader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.pos+n > len(r.b) {
		r.err = fmt.Errorf("pipeline map: truncated at %d", r.pos)
		return nil
	}
	v := r.b[r.pos : r.pos+n]
	r.pos += n
	return v
}

func (r *pipeReader) str() string {
	n := r.count(1 << 12)
	s := r.bytes(n)
	r.bytes((4 - n%4) % 4)
	return string(s)
}

func (r *pipeReader) strs() []string {
	n := r.count(1 << 16)
	out := make([]string, 0, n)
	for i := 0; i < n && r.err == nil; i++ {
		out = append(out, r.str())
	}
	return out
}

func ParseShaderPipelineMap(raw []byte) (*ShaderPipelineMap, error) {
	b := raw
	if plain, ok := DecodeWrapped(raw); ok {
		b = plain
	}
	m, err := parseIndexedPipelineMap(b)
	if err == nil {
		return m, nil
	}
	m, namedErr := parseNamedPipelineMap(b)
	if namedErr != nil {
		return nil, fmt.Errorf("%w; named layout: %v", err, namedErr)
	}
	return m, nil
}

func parseIndexedPipelineMap(b []byte) (*ShaderPipelineMap, error) {
	r := &pipeReader{b: b}
	m := &ShaderPipelineMap{}
	n := r.count(1 << 20)
	for i := 0; i < n && r.err == nil; i++ {
		ns := r.count(6)
		ids := make([]uint32, 0, ns)
		typs := make([]uint32, 0, ns)
		for j := 0; j < ns && r.err == nil; j++ {
			ids = append(ids, r.u32())
			typs = append(typs, r.u32())
		}
		m.KeywordIndex = append(m.KeywordIndex, int32(r.u32()))
		m.TypeIndex = append(m.TypeIndex, int32(r.u32()))
		m.LayoutIndex = append(m.LayoutIndex, int32(r.u32()))
		m.StageIDIndex = append(m.StageIDIndex, ids)
		p := ShaderPipeline{Stages: make([]ShaderStage, ns)}
		for j := range p.Stages {
			p.Stages[j].Type = typs[j]
		}
		m.Pipelines = append(m.Pipelines, p)
	}
	m.TypeNames = r.strs()
	m.LayoutNames = r.strs()
	nk := r.count(1 << 20)
	for i := 0; i < nk && r.err == nil; i++ {
		if l := r.u32(); l != 32 && r.err == nil {
			return nil, fmt.Errorf("pipeline map: keyword set len %d", l)
		}
		var k [32]byte
		copy(k[:], r.bytes(32))
		m.KeywordSets = append(m.KeywordSets, k)
	}
	ni := r.count(1 << 24)
	for i := 0; i < ni && r.err == nil; i++ {
		if l := r.u32(); l != 20 && r.err == nil {
			return nil, fmt.Errorf("pipeline map: shader id len %d", l)
		}
		var k [20]byte
		copy(k[:], r.bytes(20))
		m.ShaderIDs = append(m.ShaderIDs, k)
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.pos != len(b) {
		return nil, fmt.Errorf("pipeline map: trailing %d bytes", len(b)-r.pos)
	}
	if err := m.resolve(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *ShaderPipelineMap) resolve() error {
	for i := range m.Pipelines {
		p := &m.Pipelines[i]
		ti, li, ki := m.TypeIndex[i], m.LayoutIndex[i], m.KeywordIndex[i]
		if ti < 0 || int(ti) >= len(m.TypeNames) || ki < 0 || int(ki) >= len(m.KeywordSets) || int(li) >= len(m.LayoutNames) {
			return fmt.Errorf("pipeline %d: index out of range (type %d layout %d keyword %d)", i, ti, li, ki)
		}
		p.Name = m.TypeNames[ti]
		p.Keywords = m.KeywordSets[ki]
		if li >= 0 {
			p.VertexLayout = m.LayoutNames[li]
		}
		for j, id := range m.StageIDIndex[i] {
			if int(id) >= len(m.ShaderIDs) {
				return fmt.Errorf("pipeline %d: shader id index %d out of range", i, id)
			}
			p.Stages[j].ShaderID = m.ShaderIDs[id]
		}
	}
	return nil
}

func ParseShaderIDList(b []byte) (map[[20]byte][20]byte, error) {
	if len(b) < 8 {
		return nil, fmt.Errorf("shader id list: %d bytes", len(b))
	}
	if ver := binary.LittleEndian.Uint32(b); ver != 0 {
		return nil, fmt.Errorf("shader id list: version %d", ver)
	}
	n := int(binary.LittleEndian.Uint32(b[4:]))
	if n <= 0 || (len(b)-8)%n != 0 {
		return nil, fmt.Errorf("shader id list: %d entries in %d bytes", n, len(b))
	}
	stride := (len(b) - 8) / n
	var srcOff, dstOff int
	switch stride {
	case 40:
		srcOff, dstOff = 0, 20
	case 48:
		srcOff, dstOff = 4, 28
	default:
		return nil, fmt.Errorf("shader id list: stride %d", stride)
	}
	out := make(map[[20]byte][20]byte, n)
	for i := 0; i < n; i++ {
		e := b[8+i*stride:]
		var s, d [20]byte
		copy(s[:], e[srcOff:])
		copy(d[:], e[dstOff:])
		out[s] = d
	}
	return out, nil
}

func stageType(name string) uint32 {
	if c, ok := EngineShaderCodeOf(name); ok {
		return uint32(c.Type)
	}
	if strings.HasSuffix(name, "VS") {
		return ShaderStageVS
	}
	if strings.HasSuffix(name, "PS") {
		return ShaderStagePS
	}
	return 0
}

func parseNamedPipelineMap(b []byte) (*ShaderPipelineMap, error) {
	r := &pipeReader{b: b}
	m := &ShaderPipelineMap{}
	n := r.count(1 << 20)
	names := map[string]bool{}
	ids := map[[20]byte]bool{}
	kw := 0
	for i := 0; i < n && r.err == nil; i++ {
		ns := r.count(6)
		p := ShaderPipeline{Stages: make([]ShaderStage, ns)}
		for j := 0; j < ns && r.err == nil; j++ {
			copy(p.Stages[j].ShaderID[:], r.bytes(20))
			p.Stages[j].Name = r.str()
			p.Stages[j].Type = stageType(p.Stages[j].Name)
			if !ids[p.Stages[j].ShaderID] {
				ids[p.Stages[j].ShaderID] = true
				m.ShaderIDs = append(m.ShaderIDs, p.Stages[j].ShaderID)
			}
		}
		if kw == 0 {
			kw = namedKeywordWidth(b, r.pos)
		}
		copy(p.Keywords[:], r.bytes(kw))
		m.KeywordSets = append(m.KeywordSets, p.Keywords)
		p.Name = r.str()
		p.VertexLayout = r.str()
		if !names[p.Name] {
			names[p.Name] = true
			m.TypeNames = append(m.TypeNames, p.Name)
		}
		m.Pipelines = append(m.Pipelines, p)
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.pos != len(b) {
		return nil, fmt.Errorf("pipeline map: trailing %d bytes", len(b)-r.pos)
	}
	return m, nil
}

func namedKeywordWidth(b []byte, pos int) int {
	for _, w := range []int{32, 24} {
		at := pos + w
		if at+4 > len(b) {
			continue
		}
		n := int(binary.LittleEndian.Uint32(b[at:]))
		if n < 1 || n > 128 || at+4+n > len(b) {
			continue
		}
		ok := true
		for _, c := range b[at+4 : at+4+n] {
			if c < '0' || c > 'z' {
				ok = false
				break
			}
		}
		if ok {
			return w
		}
	}
	return 32
}
