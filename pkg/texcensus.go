package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/crn"
)

const (
	texCensusHead    = 48
	texCensusSamples = 40
	texType0         = 0x07720004
	texType1         = 0x0002d90c
)

type TexHeader struct {
	Version  uint32
	Magic    uint32
	Type0    uint32
	Type1    uint32
	Width    int
	Height   int
	DataSize uint32
	Format   uint32
	MipCount uint32
	CRNOff   int
}

type TexFmtStat struct {
	Format     uint32 `json:"format"`
	Count      int    `json:"count"`
	DecodeOK   int    `json:"decodeOk"`
	DecodeFail int    `json:"decodeFail"`
	HasCRN     int    `json:"hasCrn"`
}

type TexPrefixStat struct {
	Prefix     string `json:"prefix"`
	Names      int    `json:"names"`
	PNG        int    `json:"png"`
	TexMagic   int    `json:"texMagic"`
	DecodeOK   int    `json:"decodeOk"`
	DecodeFail int    `json:"decodeFail"`
}

type TexFailSample struct {
	Name     string `json:"name"`
	Class    string `json:"class"`
	Format   uint32 `json:"format,omitempty"`
	W        int    `json:"w,omitempty"`
	H        int    `json:"h,omitempty"`
	Size     int    `json:"size"`
	Mip      uint32 `json:"mip,omitempty"`
	CRNOff   int    `json:"crnOff,omitempty"`
	CRNFaces int    `json:"crnFaces,omitempty"`
	CRNFmt   uint32 `json:"crnFmt,omitempty"`
	Err      string `json:"err"`
	Head     string `json:"head"`
}

type TexCensus struct {
	Names       int             `json:"names"`
	LookupOK    int             `json:"lookupOk"`
	LookupFail  int             `json:"lookupFail"`
	LookupEmpty int             `json:"lookupEmpty"`
	BlocksPNG   int             `json:"blocksPng"`
	OtherPNG    int             `json:"otherPng"`
	NonTex      int             `json:"nonTex"`
	PNGData     int             `json:"pngData"`
	TexMagic    int             `json:"texMagic"`
	RealPNG     int             `json:"realPng"`
	DecodeOK    int             `json:"decodeOk"`
	DecodeFail  int             `json:"decodeFail"`
	Exts        []kvCount       `json:"exts"`
	Prefixes    []TexPrefixStat `json:"prefixes"`
	Formats     []TexFmtStat    `json:"formats"`
	FailSample  []TexFailSample `json:"failSamples"`
}

type kvCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

func ParseTexHeader(b []byte) (TexHeader, error) {
	h := TexHeader{CRNOff: -1}
	if len(b) < 36 {
		return h, fmt.Errorf("short %d", len(b))
	}
	h.Version = binary.LittleEndian.Uint32(b[0:4])
	h.Magic = binary.LittleEndian.Uint32(b[4:8])
	if h.Version != 2 || h.Magic != texMagic {
		return h, fmt.Errorf("magic")
	}
	if len(b) >= 16 {
		h.Type0 = binary.LittleEndian.Uint32(b[8:12])
		h.Type1 = binary.LittleEndian.Uint32(b[12:16])
	}
	if h.Type0 != texType0 || h.Type1 != texType1 {
		return h, fmt.Errorf("not tex type %08x %08x", h.Type0, h.Type1)
	}
	h.Width = int(binary.LittleEndian.Uint32(b[20:24]))
	h.Height = int(binary.LittleEndian.Uint32(b[24:28]))
	h.DataSize = binary.LittleEndian.Uint32(b[28:32])
	h.Format = binary.LittleEndian.Uint32(b[32:36])
	if len(b) >= 40 {
		h.MipCount = binary.LittleEndian.Uint32(b[36:40])
	}
	if h.Width <= 0 || h.Height <= 0 || h.Width > 16384 || h.Height > 16384 {
		return h, fmt.Errorf("size %dx%d", h.Width, h.Height)
	}
	if h.Format > 80 {
		return h, fmt.Errorf("fmt %d", h.Format)
	}
	lim := 128
	if lim > len(b) {
		lim = len(b)
	}
	for i := 100; i+2 < lim; i++ {
		if b[i] == 0x48 && b[i+1] == 0x78 {
			h.CRNOff = i
			break
		}
	}
	return h, nil
}

func CensusCommonRes(pkgPath string) (*TexCensus, error) {
	if strings.TrimSpace(pkgPath) == "" {
		pkgs := DefaultCommonResPkgs()
		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no common_res.pkg")
		}
		pkgPath = pkgs[0]
	}
	p, err := ParseFile(pkgPath)
	if err != nil {
		return nil, err
	}
	r, err := OpenReader(p)
	if err != nil {
		return nil, err
	}
	return r.CensusTextures()
}

func (r *Reader) CensusTextures() (*TexCensus, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	names := r.Names("")
	sort.Strings(names)
	c := &TexCensus{}
	c.Names = len(names)
	exts := map[string]int{}
	pref := map[string]*TexPrefixStat{}
	fmts := map[uint32]*TexFmtStat{}
	fmtSeen := map[uint32]int{}
	for _, name := range names {
		censusOne(r, c, name, exts, pref, fmts, fmtSeen)
	}
	c.Exts = sortKV(exts)
	c.Prefixes = sortPrefix(pref)
	c.Formats = sortFmt(fmts)
	return c, nil
}

func censusOne(r *Reader, c *TexCensus, name string, exts map[string]int, pref map[string]*TexPrefixStat, fmts map[uint32]*TexFmtStat, fmtSeen map[uint32]int) {
	exts[censusExt(name)]++
	ps := censusPrefixStat(pref, name)
	ps.Names++
	cls := censusClass(name)
	isPNG := cls != "non_tex"
	switch cls {
	case "blocks":
		c.BlocksPNG++
		ps.PNG++
	case "other_png":
		c.OtherPNG++
		ps.PNG++
	default:
		c.NonTex++
	}
	raw, err := r.Lookup(name)
	if err != nil {
		c.LookupFail++
		if err.Error() == "empty" {
			c.LookupEmpty++
		}
		return
	}
	c.LookupOK++
	if bytes.HasPrefix(raw, pngSignature) {
		c.RealPNG++
	}
	if !isPNG {
		return
	}
	c.PNGData++
	hdr, herr := ParseTexHeader(raw)
	if herr != nil {
		c.DecodeFail++
		ps.DecodeFail++
		censusSample(c, TexFailSample{
			Name: name, Class: cls, Size: len(raw),
			Err: "not container: " + herr.Error(), Head: censusHead(raw),
		}, fmtSeen)
		return
	}
	c.TexMagic++
	ps.TexMagic++
	st := fmts[hdr.Format]
	if st == nil {
		st = &TexFmtStat{Format: hdr.Format}
		fmts[hdr.Format] = st
	}
	st.Count++
	if hdr.CRNOff >= 0 {
		st.HasCRN++
	}
	_, derr := DecodeTextureImage(raw)
	if derr == nil {
		c.DecodeOK++
		ps.DecodeOK++
		st.DecodeOK++
		return
	}
	c.DecodeFail++
	ps.DecodeFail++
	st.DecodeFail++
	smp := TexFailSample{
		Name: name, Class: cls, Format: hdr.Format,
		W: hdr.Width, H: hdr.Height, Size: len(raw),
		Mip: hdr.MipCount, CRNOff: hdr.CRNOff,
		Err: derr.Error(), Head: censusHead(raw),
	}
	if hdr.CRNOff >= 0 && hdr.CRNOff < len(raw) {
		if info, err := crn.Probe(raw[hdr.CRNOff:]); err == nil {
			smp.CRNFaces = info.Faces
			smp.CRNFmt = info.Format
		}
	}
	censusSample(c, smp, fmtSeen)
}

func censusHead(b []byte) string {
	n := texCensusHead
	if n > len(b) {
		n = len(b)
	}
	return hex.EncodeToString(b[:n])
}
