package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type ZipFallbackStat struct {
	Path            string
	Names           int
	TexContainer    int
	NonTex          int
	Empty           int
	LookupFail      int
	ZipFiles        int
	ZipEntries      int
	ZipExtractOK    int
	ZipExtractFail  int
	RainbowFiles    int
	RainbowOK       int
	RainbowPNG      int
	DataZipEntries  int
	DataExtractOK   int
	DataExtractFail int
	DataRainbowOK   int
	DataRainbowPNG  int
}

func StatZipFallbackFile(path string, keys ZipKeys, rk RainbowKeys) (ZipFallbackStat, error) {
	p, err := ParseFile(path)
	if err != nil {
		return ZipFallbackStat{Path: path}, err
	}
	st, err := StatZipFallback(p, keys, rk)
	st.Path = path
	return st, err
}

func StatZipFallback(p *Pkg, keys ZipKeys, rk RainbowKeys) (ZipFallbackStat, error) {
	var st ZipFallbackStat
	if p == nil {
		return st, nil
	}
	statDataZip(&st, p.Data, keys, rk)
	r, err := OpenReader(p)
	if err != nil {
		return st, err
	}
	statLookupZip(&st, r, keys, rk)
	return st, nil
}

func statDataZip(st *ZipFallbackStat, data []byte, keys ZipKeys, rk RainbowKeys) {
	ents := ScanZipEntries(data)
	st.DataZipEntries = len(ents)
	for _, e := range ents {
		plain, err := ExtractZipEntry(data, e, keys)
		if err != nil {
			st.DataExtractFail++
			continue
		}
		st.DataExtractOK++
		if png := DecodeRainbow(plain, rk); png != nil {
			st.DataRainbowOK++
			if bytes.HasPrefix(png, pngSignature) {
				st.DataRainbowPNG++
			}
		}
	}
}

func statLookupZip(st *ZipFallbackStat, r *Reader, keys ZipKeys, rk RainbowKeys) {
	if r != nil && r.idx != nil {
		pool, err := r.ConcatPlain()
		if err == nil {
			st.Names = len(r.idx.byName)
			for _, flag := range r.idx.byName {
				body, err := sliceIdx(r, pool, flag)
				classifyBlob(st, body, err, keys, rk)
			}
			return
		}
	}
	names := r.Names("")
	st.Names = len(names)
	for _, n := range names {
		body, err := r.Lookup(n)
		classifyBlob(st, body, err, keys, rk)
	}
}

func sliceIdx(r *Reader, pool []byte, flag uint32) ([]byte, error) {
	var off, size uint32
	if flag&0x80000000 != 0 {
		i := int(flag & 0x7fffffff)
		if i < 0 || i >= len(r.idx.streams) {
			return nil, fmt.Errorf("bad stream %d", i)
		}
		off, size = r.idx.streams[i].off, r.idx.streams[i].size
	} else {
		i := int(flag)
		if i < 0 || i >= len(r.idx.files) {
			return nil, fmt.Errorf("bad index %d", i)
		}
		off, size = r.idx.files[i].off, r.idx.files[i].size
	}
	if size == 0 {
		return nil, fmt.Errorf("empty")
	}
	start := int(off) - uncompOrigin
	end := start + int(size)
	if start < 0 || end > len(pool) {
		return nil, fmt.Errorf("slice")
	}
	return pool[start:end], nil
}

func classifyBlob(st *ZipFallbackStat, body []byte, err error, keys ZipKeys, rk RainbowKeys) {
	if err != nil {
		if strings.Contains(err.Error(), "empty") {
			st.Empty++
		} else {
			st.LookupFail++
		}
		return
	}
	if texContainer(body) {
		st.TexContainer++
		return
	}
	st.NonTex++
	scanBlobZip(st, body, keys, rk)
}

func scanBlobZip(st *ZipFallbackStat, body []byte, keys ZipKeys, rk RainbowKeys) {
	if rainbowWrap(body, rk) {
		st.RainbowFiles++
		if png := DecodeRainbow(body, rk); png != nil {
			st.RainbowOK++
			if bytes.HasPrefix(png, pngSignature) {
				st.RainbowPNG++
			}
		}
	}
	ents := ScanZipEntries(body)
	if len(ents) == 0 {
		return
	}
	st.ZipFiles++
	st.ZipEntries += len(ents)
	for _, e := range ents {
		plain, err := ExtractZipEntry(body, e, keys)
		if err != nil {
			st.ZipExtractFail++
			continue
		}
		st.ZipExtractOK++
		if png := DecodeRainbow(plain, rk); png != nil {
			st.RainbowOK++
			if bytes.HasPrefix(png, pngSignature) {
				st.RainbowPNG++
			}
		}
	}
}

func texContainer(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	return binary.LittleEndian.Uint32(b[0:4]) == 2 && binary.LittleEndian.Uint32(b[4:8]) == texMagic
}

func rainbowWrap(b []byte, rk RainbowKeys) bool {
	if rk.Magic == "" {
		return false
	}
	return bytes.HasPrefix(b, []byte(rk.Magic))
}
