package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
)

const (
	dxFourCC     = "DXBC"
	dxWrapLen    = 10
	dxHashLen    = 20
	dxTexMagic   = 0x59A21C2C
	dxMaxChunk   = 16
	dxMaxSign    = 32
	dxMaxName    = 96
	dxMaxBinding = 80
)

type DXKind string

const (
	DXKindDXBC     DXKind = "dxbc"
	DXKindSHA1FP   DXKind = "sha1fp"
	DXKindTemplate DXKind = "templatemat"
	DXKindCompute  DXKind = "compute"
	DXKindIDList   DXKind = "idlist"
	DXKindPipeline DXKind = "pipeline"
	DXKindUnknown  DXKind = "unknown"
)

type DXInfo struct {
	Name     string   `json:"name"`
	Kind     DXKind   `json:"kind"`
	Size     int      `json:"size"`
	DXOff    int      `json:"dxOff,omitempty"`
	DXSize   int      `json:"dxSize,omitempty"`
	DXCount  int      `json:"dxCount,omitempty"`
	Stage    int      `json:"stage"`
	Chunks   []string `json:"chunks,omitempty"`
	Hash     string   `json:"hash,omitempty"`
	SRV      int      `json:"srv"`
	CB       int      `json:"cb"`
	Sampler  int      `json:"sampler"`
	Wrapped  bool     `json:"wrapped,omitempty"`
	BindOK   bool     `json:"bindOk"`
	Bindings []string `json:"bindings,omitempty"`
	Signs    []string `json:"signs,omitempty"`
	HLSL     []string `json:"hlsl,omitempty"`
}

func ExtractDXBC(b []byte) ([]byte, []byte, bool) {
	off := dxBCOff(b)
	if off < 0 || off+28 > len(b) {
		return nil, nil, false
	}
	n := int(binary.LittleEndian.Uint32(b[off+24 : off+28]))
	if n < 32 || off+n > len(b) {
		return nil, nil, false
	}
	dx := b[off : off+n]
	var extra []byte
	if off+n < len(b) {
		extra = b[off+n:]
	}
	return dx, extra, true
}

func ClassifyDXBlob(name string, b []byte) DXInfo {
	info := DXInfo{Name: name, Size: len(b), Kind: DXKindUnknown, DXOff: -1, Stage: -1}
	low := strings.ToLower(name)
	if len(b) == dxHashLen {
		info.Kind = DXKindSHA1FP
		info.Hash = hex.EncodeToString(b)
		return info
	}
	if len(b) == dxHashLen+4 && binary.LittleEndian.Uint32(b[:4]) == dxHashLen {
		info.Kind = DXKindSHA1FP
		info.Hash = hex.EncodeToString(b[4:])
		return info
	}
	if strings.HasSuffix(low, ".templatemat") {
		info.Kind = DXKindTemplate
		info.HLSL = dxHLSLNames(b)
		info.Bindings = dxLenNames(b, 8)
		return info
	}
	if strings.HasSuffix(low, "shaderidlist.bin") {
		info.Kind = DXKindIDList
		return info
	}
	if strings.HasSuffix(low, "effectshaderpipelinemap.bin") {
		info.Kind = DXKindPipeline
		info.Bindings = dxPipeNames(b)
		return info
	}
	if strings.HasSuffix(low, ".compute") || dxIsCompute(b) {
		info.Kind = DXKindCompute
		fillDXBC(&info, b)
		info.DXCount = len(ExtractAllDXBC(b))
		off := dxBCOff(b)
		info.Bindings = dxASCIIIdents(b[:clamp(off, 0, len(b))], 4)
		return info
	}
	if fillWrap(&info, b, false) {
		return info
	}
	if plain, ok := DecodeWrapped(b); ok && len(plain) > MaterialHdrSize && string(plain[MaterialHdrSize:MaterialHdrSize+4]) == dxFourCC {
		if fillWrap(&info, plain[materialHdrSHA1:], true) {
			info.Wrapped = true
			info.Hash = hex.EncodeToString(plain[:materialHdrSHA1])
			return info
		}
	}
	if fillDXBC(&info, b) {
		info.Kind = DXKindDXBC
	}
	return info
}

func fillWrap(info *DXInfo, b []byte, material bool) bool {
	w, err := ParseDXWrap(b)
	if err != nil {
		return false
	}
	info.Kind = DXKindDXBC
	info.DXOff = dxWrapLen
	info.DXSize = len(w.DXBC)
	info.DXCount = 1
	info.SRV, info.CB, info.Sampler = int(w.SRV), int(w.CB), int(w.Sampler)
	info.Stage = DXProgramType(w.DXBC)
	info.Chunks, info.Signs = dxChunks(w.DXBC)
	var bd DXBinding
	if material {
		bd, err = ParseMaterialBinding(w.Extra)
	} else {
		bd, err = ParseDXBinding(w.Extra)
	}
	if err == nil {
		info.BindOK = w.CountsOK(bd)
		info.Hash = bd.Hash
		info.Bindings = bd.Names()
		return true
	}
	if h, names := dxParseExtra(w.Extra); h != "" || len(names) > 0 {
		info.Hash = h
		info.Bindings = names
	}
	return true
}

func fillDXBC(info *DXInfo, b []byte) bool {
	dx, extra, ok := ExtractDXBC(b)
	if !ok {
		return false
	}
	info.DXOff = dxBCOff(b)
	info.DXSize = len(dx)
	info.Stage = DXProgramType(dx)
	info.Chunks, info.Signs = dxChunks(dx)
	if h, names := dxParseExtra(extra); h != "" || len(names) > 0 {
		info.Hash = h
		info.Bindings = names
	}
	return true
}

func dxBCOff(b []byte) int {
	if len(b) >= dxWrapLen+4 && string(b[dxWrapLen:dxWrapLen+4]) == dxFourCC {
		return dxWrapLen
	}
	return bytes.Index(b, []byte(dxFourCC))
}

func dxIsCompute(b []byte) bool {
	if len(b) < 32 {
		return false
	}
	if binary.LittleEndian.Uint32(b[:4]) != 1 || binary.LittleEndian.Uint32(b[4:8]) != dxTexMagic {
		return false
	}
	off := bytes.Index(b, []byte(dxFourCC))
	return off > 64 && off < 4096
}

func dxChunks(dx []byte) ([]string, []string) {
	if len(dx) < 32 || string(dx[:4]) != dxFourCC {
		return nil, nil
	}
	n := int(binary.LittleEndian.Uint32(dx[28:32]))
	if n <= 0 || n > dxMaxChunk || 32+4*n > len(dx) {
		return nil, nil
	}
	tags := make([]string, 0, n)
	var signs []string
	total := int(binary.LittleEndian.Uint32(dx[24:28]))
	if total > len(dx) {
		total = len(dx)
	}
	for i := 0; i < n; i++ {
		off := int(binary.LittleEndian.Uint32(dx[32+4*i : 36+4*i]))
		if off < 0 || off+8 > total {
			continue
		}
		tag := string(dx[off : off+4])
		tags = append(tags, tag)
		sz := int(binary.LittleEndian.Uint32(dx[off+4 : off+8]))
		if (tag == "ISGN" || tag == "OSGN") && off+8+sz <= total {
			signs = append(signs, dxSignNames(dx[off+8:off+8+sz])...)
		}
	}
	return tags, uniqDX(signs)
}

func dxSignNames(payload []byte) []string {
	if len(payload) < 8 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(payload[:4]))
	if n <= 0 || n > dxMaxSign {
		return nil
	}
	var out []string
	for i := 0; i < n; i++ {
		e := 8 + i*24
		if e+4 > len(payload) {
			break
		}
		noff := int(binary.LittleEndian.Uint32(payload[e : e+4]))
		if noff < 0 || noff >= len(payload) {
			continue
		}
		s := dxZString(payload[noff:])
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func dxParseExtra(ex []byte) (string, []string) {
	if len(ex) < 26 {
		return "", nil
	}
	hash := ""
	start := 0
	if binary.LittleEndian.Uint32(ex[2:6]) == dxHashLen {
		hash = hex.EncodeToString(ex[6:26])
		start = 46
		if start > len(ex) {
			start = 26
		}
	}
	return hash, dxLenNames(ex[start:], dxMaxBinding)
}
