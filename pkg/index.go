package pkg

import (
	"encoding/binary"
	"fmt"
)

const (
	assetRecBytes  = 44
	streamRecBytes = 12
	storRecBytes   = 8
	uncompOrigin   = 16
)

type assetRec struct {
	off   uint32
	size  uint32
	flags uint32
}

func (r assetRec) blockIndex() uint32 {
	return r.flags >> 6
}

type storRec struct {
	uncomp uint32
	comp   uint32
}

type streamRec struct {
	off   uint32
	size  uint32
	flags uint32
}

type pkgIndex struct {
	files    []assetRec
	streams  []streamRec
	byName   map[string]uint32
	stor     []storRec
	uncompAt []uint64
	compAt   []uint64
}

func u32le(b []byte, o int) uint32 {
	return binary.LittleEndian.Uint32(b[o:])
}

func decodePkgIndex(raw []byte) ([]byte, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("index too short")
	}
	want := int(u32le(raw, 0))
	if want < 4 || want > 64<<20 {
		return nil, fmt.Errorf("index want %d", want)
	}
	return DecompressLZ4Block(raw[4:], want+64)
}

func parsePkgIndex(idx []byte) (*pkgIndex, error) {
	if len(idx) < 8 {
		return nil, fmt.Errorf("plain index too short")
	}
	nfile := int(u32le(idx, 0))
	if nfile < 0 || nfile > 500000 {
		return nil, fmt.Errorf("nfile %d", nfile)
	}
	endFiles := 4 + nfile*assetRecBytes
	if endFiles+4 > len(idx) {
		return nil, fmt.Errorf("files overrun")
	}
	files := make([]assetRec, nfile)
	for i := 0; i < nfile; i++ {
		b := 4 + i*assetRecBytes
		files[i] = assetRec{
			off:   u32le(idx, b+16),
			size:  u32le(idx, b+20),
			flags: u32le(idx, b+40),
		}
	}
	off := endFiles
	nstream := int(u32le(idx, off))
	off += 4
	if nstream < 0 || nstream > 500000 {
		return nil, fmt.Errorf("nstream %d", nstream)
	}
	if off+nstream*streamRecBytes+4 > len(idx) {
		return nil, fmt.Errorf("stream overrun")
	}
	streams := make([]streamRec, nstream)
	for i := 0; i < nstream; i++ {
		b := off + i*streamRecBytes
		streams[i] = streamRec{
			off:   u32le(idx, b),
			size:  u32le(idx, b+4),
			flags: u32le(idx, b+8),
		}
	}
	off += nstream * streamRecBytes
	nmap := int(u32le(idx, off))
	off += 4
	if nmap <= 0 || nmap > 500000 {
		return nil, fmt.Errorf("nmap %d", nmap)
	}
	byName := make(map[string]uint32, nmap)
	for i := 0; i < nmap; i++ {
		if off+8 > len(idx) {
			return nil, fmt.Errorf("name hdr %d", i)
		}
		nl := int(u32le(idx, off))
		off += 4
		if nl < 1 || nl > 400 || off+nl+4 > len(idx) {
			return nil, fmt.Errorf("name len %d at %d", nl, i)
		}
		raw := idx[off : off+nl]
		off += nl
		for off%4 != 0 {
			if off >= len(idx) {
				return nil, fmt.Errorf("name pad %d", i)
			}
			off++
		}
		if off+4 > len(idx) {
			return nil, fmt.Errorf("name pad %d", i)
		}
		flag := u32le(idx, off)
		off += 4
		if raw[nl-1] == 0 {
			raw = raw[:nl-1]
		}
		byName[string(raw)] = flag
	}
	if off+4 > len(idx) {
		return nil, fmt.Errorf("stor missing")
	}
	nstor := int(u32le(idx, off))
	off += 4
	if nstor <= 0 || nstor > 20000 {
		return nil, fmt.Errorf("nstor %d", nstor)
	}
	need := off + nstor*storRecBytes
	if need > len(idx) {
		return nil, fmt.Errorf("stor overrun")
	}
	stor := make([]storRec, nstor)
	uncompAt := make([]uint64, nstor+1)
	compAt := make([]uint64, nstor+1)
	uncompAt[0] = uncompOrigin
	for i := 0; i < nstor; i++ {
		b := off + i*storRecBytes
		stor[i] = storRec{uncomp: u32le(idx, b), comp: u32le(idx, b+4)}
		uncompAt[i+1] = uncompAt[i] + uint64(stor[i].uncomp)
		compAt[i+1] = compAt[i] + uint64(stor[i].comp)
	}
	return &pkgIndex{files: files, streams: streams, byName: byName, stor: stor, uncompAt: uncompAt, compAt: compAt}, nil
}

func (p *Pkg) IndexFileCount() (int, bool) {
	plain, err := decodePkgIndex(p.Index)
	if err != nil {
		return 0, false
	}
	n := int(u32le(plain, 0))
	if n > 0 {
		return n, true
	}
	if len(plain) >= 8 {
		n = int(u32le(plain, 4))
		if n > 0 {
			return n, true
		}
	}
	return 0, false
}
