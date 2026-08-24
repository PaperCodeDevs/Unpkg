package pkg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	HeaderSize   = 16
	SubtypeBase  = 9
	SubtypePatch = 12
)

type Pkg struct {
	Version uint32
	Subtype uint32
	IdxOff  uint32
	IdxSize uint32
	Data    []byte
	Index   []byte
	Raw     []byte
}

func Parse(raw []byte) (*Pkg, error) {
	if len(raw) < HeaderSize {
		return nil, fmt.Errorf("Parse: too short")
	}
	version := binary.LittleEndian.Uint32(raw[0:4])
	subtype := binary.LittleEndian.Uint32(raw[4:8])
	idxOff := binary.LittleEndian.Uint32(raw[8:12])
	idxSize := binary.LittleEndian.Uint32(raw[12:16])
	if uint64(idxOff)+uint64(idxSize) != uint64(len(raw)) {
		return nil, fmt.Errorf("Parse: idx_off+idx_size (%d) != size (%d)", idxOff+idxSize, len(raw))
	}
	if idxOff < HeaderSize {
		return nil, fmt.Errorf("Parse: idx_off too small")
	}
	return &Pkg{
		Version: version,
		Subtype: subtype,
		IdxOff:  idxOff,
		IdxSize: idxSize,
		Data:    append([]byte(nil), raw[HeaderSize:idxOff]...),
		Index:   append([]byte(nil), raw[idxOff:]...),
		Raw:     append([]byte(nil), raw...),
	}, nil
}

func ParseFile(path string) (*Pkg, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ParseFile: %w", err)
	}
	p, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("ParseFile %s: %w", filepath.Base(path), err)
	}
	return p, nil
}

func (p *Pkg) ReadableRatio(n int) float64 {
	if p == nil || len(p.Data) == 0 {
		return 0
	}
	if n <= 0 || n > len(p.Data) {
		n = len(p.Data)
		if n > 8000 {
			n = 8000
		}
	}
	good := 0
	for _, c := range p.Data[:n] {
		if (c >= 9 && c <= 126) || c >= 0x80 {
			good++
		}
	}
	return float64(good) / float64(n)
}

func (p *Pkg) WriteData(path string) error {
	if p == nil {
		return fmt.Errorf("WriteData: nil pkg")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, p.Data, 0o644)
}

func (p *Pkg) WriteIndex(path string) error {
	if p == nil {
		return fmt.Errorf("WriteIndex: nil pkg")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, p.Index, 0o644)
}

func DumpFile(srcPath, outDataPath string) (*Pkg, error) {
	p, err := ParseFile(srcPath)
	if err != nil {
		return nil, err
	}
	if err := p.WriteData(outDataPath); err != nil {
		return p, err
	}
	return p, nil
}

func DecryptXXTEAUnzip(b64 []byte, key []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(b64)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("DecryptXXTEAUnzip: empty")
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("DecryptXXTEAUnzip: empty key")
	}
	raw, err := base64.StdEncoding.DecodeString(string(trimmed))
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(string(trimmed) + "====")
		if err != nil {
			return nil, fmt.Errorf("DecryptXXTEAUnzip: base64: %w", err)
		}
	}
	dec := DecryptXXTEA(raw, key, false)
	if len(dec) < 4 {
		return nil, fmt.Errorf("DecryptXXTEAUnzip: too short")
	}
	n := binary.BigEndian.Uint32(dec[:4])
	payload := dec[4:]
	if int(n) > 0 && int(n) <= len(payload) {
		payload = payload[:n]
	}
	zr, err := zlib.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("DecryptXXTEAUnzip: zlib: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("DecryptXXTEAUnzip: inflate: %w", err)
	}
	return out, nil
}
