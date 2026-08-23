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
	Chunks   []string `json:"chunks,omitempty"`
	Hash     string   `json:"hash,omitempty"`
	WrapA    int      `json:"wrapA,omitempty"`
	WrapB    int      `json:"wrapB,omitempty"`
	Bindings []string `json:"bindings,omitempty"`
	Signs    []string `json:"signs,omitempty"`
	HLSL     []string `json:"hlsl,omitempty"`
}

func ExtractDXBC(b []byte) ([]byte, []byte, bool) {
	// 包装头 10B：u32=DXBC.Size+6，随后 01 A B A 00 00，再接 DXBC 与绑定表
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
	info := DXInfo{Name: name, Size: len(b), Kind: DXKindUnknown, DXOff: -1}
	low := strings.ToLower(name)
	if len(b) == 24 && binary.LittleEndian.Uint32(b[:4]) == dxHashLen {
		info.Kind = DXKindSHA1FP
		info.Hash = hex.EncodeToString(b[4:24])
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
		off := dxBCOff(b)
		info.Bindings = dxASCIIIdents(b[:clamp(off, 0, len(b))], 4)
		return info
	}
	if fillDXBC(&info, b) {
		info.Kind = DXKindDXBC
		if info.DXOff == dxWrapLen && len(b) >= dxWrapLen && b[4] == 1 {
			info.WrapA = int(b[5])
			info.WrapB = int(b[6])
		}
		return info
	}
	return info
}

func fillDXBC(info *DXInfo, b []byte) bool {
	dx, extra, ok := ExtractDXBC(b)
	if !ok {
		return false
	}
	info.DXOff = dxBCOff(b)
	info.DXSize = len(dx)
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
