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

func zipEntryAt(buf []byte, i int) (ZipEntry, bool) {
	if i < 0 || i+30 > len(buf) || binary.LittleEndian.Uint32(buf[i:]) != ZipLocalMagic {
		return ZipEntry{}, false
	}
	nlen := int(binary.LittleEndian.Uint16(buf[i+26:]))
	elen := int(binary.LittleEndian.Uint16(buf[i+28:]))
	if i+30+nlen+elen > len(buf) {
		return ZipEntry{}, false
	}
	return ZipEntry{
		Name:       string(buf[i+30 : i+30+nlen]),
		Flag:       binary.LittleEndian.Uint16(buf[i+6:]),
		Method:     binary.LittleEndian.Uint16(buf[i+8:]),
		CRC32:      binary.LittleEndian.Uint32(buf[i+14:]),
		CompSize:   binary.LittleEndian.Uint32(buf[i+18:]),
		UncompSize: binary.LittleEndian.Uint32(buf[i+22:]),
		DataOffset: i + 30 + nlen + elen,
		HeaderAt:   i,
	}, true
}

func ScanZipEntries(data []byte) []ZipEntry {
	var out []ZipEntry
	for i := 0; i+30 <= len(data); i++ {
		if binary.LittleEndian.Uint32(data[i:]) != ZipLocalMagic {
			continue
		}
		e, ok := zipEntryAt(data, i)
		if !ok || e.DataOffset+int(e.CompSize) > len(data) {
			continue
		}
		out = append(out, e)
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

func inflateZipPayload(payload []byte, e ZipEntry, keys ZipKeys) ([]byte, error) {
	if e.Method != 0 && e.Method != 8 {
		return nil, fmt.Errorf("method=%d unsupported", e.Method)
	}
	comp := payload
	if e.Flag&1 != 0 {
		if len(payload) < 12 {
			return nil, fmt.Errorf("cipher too short")
		}
		comp = DecryptZipCrypto(payload, keys.K0, keys.K1, keys.K2)[12:]
	}
	if e.Method == 0 {
		return append([]byte(nil), comp...), nil
	}
	return RawInflate(comp)
}

func ExtractZipEntry(data []byte, e ZipEntry, keys ZipKeys) ([]byte, error) {
	if e.DataOffset < 0 || e.DataOffset+int(e.CompSize) > len(data) {
		return nil, fmt.Errorf("ExtractZipEntry %s: out of range", e.Name)
	}
	body, err := inflateZipPayload(data[e.DataOffset:e.DataOffset+int(e.CompSize)], e, keys)
	if err != nil {
		return nil, fmt.Errorf("ExtractZipEntry %s: %w", e.Name, err)
	}
	return body, nil
}

func VerifyZipEntry(body []byte, e ZipEntry) (lenOK, crcOK bool) {
	return uint32(len(body)) == e.UncompSize, crc32.ChecksumIEEE(body) == e.CRC32
}
