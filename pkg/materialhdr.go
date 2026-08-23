package pkg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	MaterialHdrSize   = 36
	materialHdrSHA1   = 20
	materialHdrPayOff = 26
)

type MaterialHdr struct {
	CodeSize uint32
	Type     uint16
	SHA1     [20]byte
	Payload  uint32
	Ver      uint8
	Stage    uint8
	Slot     uint8
	Stage2   uint8
	Extra    uint16
}

type MaterialShader struct {
	Hdr    MaterialHdr
	DXBC   []byte
	Digest [16]byte
}

type MaterialHdrStat struct {
	Blobs     int
	Shader    int
	SHA1FP    int
	Other     int
	NameOK    int
	StageDup  int
	PayloadOK int
	DXBCOk    int
	Stages    [6]int
	Types     map[uint16]int
	Extras    map[uint16]int
}

func ParseMaterialHdr(b []byte) (MaterialHdr, error) {
	var h MaterialHdr
	if len(b) < MaterialHdrSize {
		return h, fmt.Errorf("material hdr short")
	}
	h.CodeSize = binary.LittleEndian.Uint32(b[0:4])
	h.Type = binary.LittleEndian.Uint16(b[4:6])
	copy(h.SHA1[:], b[6:26])
	h.Payload = binary.LittleEndian.Uint32(b[materialHdrPayOff:30])
	h.Ver = b[30]
	h.Stage = b[31]
	h.Slot = b[32]
	h.Stage2 = b[33]
	h.Extra = binary.LittleEndian.Uint16(b[34:36])
	return h, nil
}

func SplitMaterialShader(blob []byte) (MaterialShader, error) {
	var out MaterialShader
	if len(blob) < MaterialHdrSize+32 {
		return out, fmt.Errorf("material shader short")
	}
	if string(blob[MaterialHdrSize:MaterialHdrSize+4]) != "DXBC" {
		return out, fmt.Errorf("material shader magic")
	}
	h, err := ParseMaterialHdr(blob[:MaterialHdrSize])
	if err != nil {
		return out, err
	}
	dx := blob[MaterialHdrSize:]
	out.Hdr = h
	out.DXBC = dx
	copy(out.Digest[:], dx[4:20])
	return out, nil
}

func (h MaterialHdr) SHA1Hex() string {
	return hex.EncodeToString(h.SHA1[:])
}

func (h MaterialHdr) MatchName(name string) bool {
	sum, ok := materialNameSHA1(name)
	if !ok {
		return false
	}
	return h.SHA1 == sum
}

func (h MaterialHdr) StageDupOK() bool {
	return h.Stage == h.Stage2
}

func (h MaterialHdr) PayloadOK(dxbc []byte) bool {
	if len(dxbc) < 26 {
		return false
	}
	return h.Payload == uint32(binary.LittleEndian.Uint16(dxbc[24:26]))+6
}

func MaterialStageName(st uint8) string {
	switch st {
	case 0:
		return "ps"
	case 1:
		return "vs"
	case 2:
		return "gs"
	case 3:
		return "hs"
	case 4:
		return "ds"
	case 5:
		return "cs"
	default:
		return ""
	}
}

func ClassifyMaterialBlob(blob []byte) string {
	if len(blob) == materialHdrSHA1 {
		return "sha1fp"
	}
	if len(blob) >= MaterialHdrSize+4 && string(blob[MaterialHdrSize:MaterialHdrSize+4]) == "DXBC" {
		return "shader"
	}
	return "other"
}

func StatMaterialHdrs(names []string, blobs [][]byte) MaterialHdrStat {
	st := MaterialHdrStat{Types: map[uint16]int{}, Extras: map[uint16]int{}}
	for i, b := range blobs {
		st.Blobs++
		kind := ClassifyMaterialBlob(b)
		switch kind {
		case "sha1fp":
			st.SHA1FP++
			continue
		case "other":
			st.Other++
			continue
		}
		st.Shader++
		sh, err := SplitMaterialShader(b)
		if err != nil {
			st.Other++
			st.Shader--
			continue
		}
		if sh.Hdr.StageDupOK() {
			st.StageDup++
		}
		if sh.Hdr.PayloadOK(sh.DXBC) {
			st.PayloadOK++
		}
		if len(sh.DXBC) >= 32 && string(sh.DXBC[:4]) == "DXBC" {
			st.DXBCOk++
		}
		if sh.Hdr.Stage < 6 {
			st.Stages[sh.Hdr.Stage]++
		}
		st.Types[sh.Hdr.Type]++
		st.Extras[sh.Hdr.Extra]++
		if i < len(names) && sh.Hdr.MatchName(names[i]) {
			st.NameOK++
		}
	}
	return st
}

func (r *Reader) StatMaterialHdrs() MaterialHdrStat {
	if r == nil {
		return MaterialHdrStat{Types: map[uint16]int{}, Extras: map[uint16]int{}}
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
