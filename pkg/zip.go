package pkg

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	ZipLocalMagic   = 0x04034b50
	maxInflateBytes = 32 << 20
)

type ZipEntry struct {
	Name       string
	Flag       uint16
	Method     uint16
	CRC32      uint32
	CompSize   uint32
	UncompSize uint32
	DataOffset int
	HeaderAt   int
}

func ScanZipEntries(data []byte) []ZipEntry {
	var out []ZipEntry
	n := len(data)
	for i := 0; i+30 <= n; i++ {
		if binary.LittleEndian.Uint32(data[i:]) != ZipLocalMagic {
			continue
		}
		flag := binary.LittleEndian.Uint16(data[i+6:])
		method := binary.LittleEndian.Uint16(data[i+8:])
		crc := binary.LittleEndian.Uint32(data[i+14:])
		csz := binary.LittleEndian.Uint32(data[i+18:])
		usz := binary.LittleEndian.Uint32(data[i+22:])
		nlen := int(binary.LittleEndian.Uint16(data[i+26:]))
		elen := int(binary.LittleEndian.Uint16(data[i+28:]))
		if i+30+nlen+elen+int(csz) > n {
			continue
		}
		name := string(data[i+30 : i+30+nlen])
		out = append(out, ZipEntry{
			Name: name, Flag: flag, Method: method, CRC32: crc,
			CompSize: csz, UncompSize: usz,
			DataOffset: i + 30 + nlen + elen, HeaderAt: i,
		})
	}
	return out
}

type zipStream struct{ k0, k1, k2 uint32 }

func zipCrcStep(crc uint32, c byte) uint32 {
	return crc32.IEEETable[byte(crc)^c] ^ (crc >> 8)
}

func (z *zipStream) update(c byte) {
	z.k0 = zipCrcStep(z.k0, c)
	z.k1 = z.k1 + (z.k0 & 0xff)
	z.k1 = z.k1*134775813 + 1
	z.k2 = zipCrcStep(z.k2, byte(z.k1>>24))
}

func (z *zipStream) next() byte {
	t := (z.k2 | 2) & 0xffff
	return byte((t * (t ^ 1)) >> 8)
}

func DecryptZipCrypto(cipher []byte, k0, k1, k2 uint32) []byte {
	z := &zipStream{k0, k1, k2}
	out := make([]byte, len(cipher))
	for i, c := range cipher {
		p := c ^ z.next()
		out[i] = p
		z.update(p)
	}
	return out
}

func EncryptZipCrypto(plain []byte, k0, k1, k2 uint32) []byte {
	z := &zipStream{k0, k1, k2}
	out := make([]byte, len(plain))
	for i, p := range plain {
		out[i] = p ^ z.next()
		z.update(p)
	}
	return out
}

func RawDeflate(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, fmt.Errorf("RawDeflate: %w", err)
	}
	if _, err := w.Write(src); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("RawDeflate: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("RawDeflate: %w", err)
	}
	return buf.Bytes(), nil
}

func RawInflate(src []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(src))
	defer r.Close()
	body, err := io.ReadAll(io.LimitReader(r, int64(maxInflateBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("RawInflate: %w", err)
	}
	if len(body) > maxInflateBytes {
		return nil, fmt.Errorf("RawInflate: too large")
	}
	return body, nil
}

func ExtractZipEntry(data []byte, e ZipEntry, keys ZipKeys) ([]byte, error) {
	if e.Method != 8 {
		return nil, fmt.Errorf("ExtractZipEntry %s: method=%d not deflate", e.Name, e.Method)
	}
	if e.DataOffset+int(e.CompSize) > len(data) {
		return nil, fmt.Errorf("ExtractZipEntry %s: out of range", e.Name)
	}
	payload := data[e.DataOffset : e.DataOffset+int(e.CompSize)]
	var comp []byte
	if e.Flag&1 != 0 {
		if len(payload) < 12 {
			return nil, fmt.Errorf("ExtractZipEntry %s: cipher too short", e.Name)
		}
		dec := DecryptZipCrypto(payload, keys.K0, keys.K1, keys.K2)
		comp = dec[12:]
	} else {
		comp = payload
	}
	body, err := RawInflate(comp)
	if err != nil {
		return nil, fmt.Errorf("ExtractZipEntry %s: %w", e.Name, err)
	}
	return body, nil
}

func VerifyZipEntry(body []byte, e ZipEntry) (lenOK, crcOK bool) {
	return uint32(len(body)) == e.UncompSize, crc32.ChecksumIEEE(body) == e.CRC32
}
