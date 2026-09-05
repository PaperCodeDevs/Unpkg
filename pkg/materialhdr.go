package pkg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	MaterialHdrSize  = 30
	materialHdrSHA1  = 20
	materialPayOff   = 20
	materialItemHead = 24
)

type MaterialHdr struct {
	RawSize uint32
	SHA1    [20]byte
	Payload uint32
	Ver     uint8
	SRV     uint8
	CB      uint8
	Sampler uint8
}

type MaterialShader struct {
	Hdr    MaterialHdr
	Plain  []byte
	DXBC   []byte
	Extra  []byte
	Digest [16]byte
}

type MaterialHdrStat struct {
	Blobs     int
	Shader    int
	SHA1FP    int
	Material  int
	Other     int
	SizeOK    int
	NameOK    int
	PayloadOK int
	DXBCOk    int
	BindOK    int
	CountOK   int
	Stages    [6]int
	Kinds     map[uint8]int
}

func ParseMaterialHdr(plain []byte) (MaterialHdr, error) {
	// 解压后条目：sha1[20] u32(6+DXBC 总长) 01 SRV数 CB数 采样器数 00 00 DXBC
	var h MaterialHdr
	if len(plain) < MaterialHdrSize {
		return h, fmt.Errorf("material hdr short")
	}
	copy(h.SHA1[:], plain[:materialHdrSHA1])
	h.Payload = binary.LittleEndian.Uint32(plain[materialPayOff : materialPayOff+4])
	h.Ver = plain[materialItemHead]
	h.SRV = plain[materialItemHead+1]
	h.CB = plain[materialItemHead+2]
	h.Sampler = plain[materialItemHead+3]
	return h, nil
}

func SplitMaterialShader(blob []byte) (MaterialShader, error) {
	var out MaterialShader
	plain, ok := DecodeWrapped(blob)
	if !ok {
		return out, fmt.Errorf("material shader lz4")
	}
	if len(plain) < MaterialHdrSize+32 || string(plain[MaterialHdrSize:MaterialHdrSize+4]) != dxFourCC {
		return out, fmt.Errorf("material shader magic")
	}
	h, err := ParseMaterialHdr(plain)
	if err != nil {
		return out, err
	}
	h.RawSize = binary.LittleEndian.Uint32(blob[:4])
	w, err := ParseDXWrap(plain[materialHdrSHA1:])
	if err != nil {
		return out, err
	}
	out.Hdr = h
	out.Plain = plain
	out.DXBC = w.DXBC
	out.Extra = w.Extra
	copy(out.Digest[:], w.DXBC[4:20])
	return out, nil
}

func (h MaterialHdr) SHA1Hex() string {
	return hex.EncodeToString(h.SHA1[:])
}

func (h MaterialHdr) MatchName(name string) bool {
	sum, ok := materialNameSHA1(name)
	return ok && h.SHA1 == sum
}

func (h MaterialHdr) PayloadOK(dxbc []byte) bool {
	if len(dxbc) < 28 {
		return false
	}
	return h.Payload == binary.LittleEndian.Uint32(dxbc[24:28])+dxWrapSub
}

func (s MaterialShader) SizeOK() bool {
	return int(s.Hdr.RawSize) == len(s.Plain)
}

func (s MaterialShader) Binding() (DXBinding, error) {
	return ParseMaterialBinding(s.Extra)
}

func (s MaterialShader) CountsOK(bd DXBinding) bool {
	return int(s.Hdr.SRV) == len(bd.Textures)+len(bd.Buffers) && int(s.Hdr.CB) == len(bd.CBuffers) && int(s.Hdr.Sampler) == len(bd.Textures)
}

func ClassifyMaterialBlob(blob []byte) string {
	if len(blob) == materialHdrSHA1 {
		return "sha1fp"
	}
	plain, ok := DecodeWrapped(blob)
	if !ok {
		return "other"
	}
	if len(plain) >= MaterialHdrSize+4 && string(plain[MaterialHdrSize:MaterialHdrSize+4]) == dxFourCC {
		return "shader"
	}
	if lenStrAt(plain, 0) != "" {
		return "material"
	}
	return "other"
}

func StatMaterialHdrs(names []string, blobs [][]byte) MaterialHdrStat {
	st := MaterialHdrStat{Kinds: map[uint8]int{}}
	for i, b := range blobs {
		st.Blobs++
		switch ClassifyMaterialBlob(b) {
		case "sha1fp":
			st.SHA1FP++
			continue
		case "material":
			st.Material++
			continue
		case "other":
			st.Other++
			continue
		}
		sh, err := SplitMaterialShader(b)
		if err != nil {
			st.Other++
			continue
		}
		st.Shader++
		statMaterialShader(&st, sh, i, names)
	}
	return st
}

func statMaterialShader(st *MaterialHdrStat, sh MaterialShader, i int, names []string) {
	if sh.SizeOK() {
		st.SizeOK++
	}
	if sh.Hdr.PayloadOK(sh.DXBC) {
		st.PayloadOK++
	}
	if dxContainerLen(sh.DXBC) == len(sh.DXBC) {
		st.DXBCOk++
	}
	if pt := DXProgramType(sh.DXBC); pt >= 0 && pt < len(st.Stages) {
		st.Stages[pt]++
	}
	if i < len(names) && sh.Hdr.MatchName(names[i]) {
		st.NameOK++
	}
	bd, err := sh.Binding()
	if err != nil {
		return
	}
	st.BindOK++
	if sh.CountsOK(bd) {
		st.CountOK++
	}
	for _, t := range bd.Textures {
		st.Kinds[t.Kind]++
	}
}

func (r *Reader) StatMaterialHdrs() MaterialHdrStat {
	if r == nil {
		return MaterialHdrStat{Kinds: map[uint8]int{}}
	}
	names := r.Names("")
	blobs := make([][]byte, 0, len(names))
	okNames := make([]string, 0, len(names))
	for _, n := range names {
		b, err := r.Lookup(n)
		if err != nil {
			continue
		}
		okNames = append(okNames, n)
		blobs = append(blobs, b)
	}
	return StatMaterialHdrs(okNames, blobs)
}

func materialNameSHA1(name string) ([20]byte, bool) {
	var out [20]byte
	base := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		base = name[i+1:]
	}
	if len(base) != 40 {
		return out, false
	}
	sum, err := hex.DecodeString(base)
	if err != nil || len(sum) != 20 {
		return out, false
	}
	copy(out[:], sum)
	return out, true
}
