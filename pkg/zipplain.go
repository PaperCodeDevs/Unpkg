package pkg

import (
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
)

const (
	zipCarry          = 4096
	zipFlagDescriptor = 1 << 3
	zipDescBack       = 8
	zipDescWin        = 32
)

// Data 池是 stor 块拼接且相邻块可能 LZ4 压缩，扫原始 Data 会在跨块处读到压缩字节，只能在明文块空间扫。
func (r *Reader) ScanZipPlain() ([]ZipEntry, error) {
	if r == nil || r.idx == nil {
		return nil, fmt.Errorf("ScanZipPlain: no index")
	}
	total := r.idx.uncompAt[len(r.idx.uncompAt)-1]
	var out []ZipEntry
	var carry []byte
	for i := range r.idx.stor {
		plain, err := r.blockPeek(i)
		if err != nil {
			return nil, fmt.Errorf("ScanZipPlain block %d: %w", i, err)
		}
		base := r.idx.uncompAt[i]
		if len(carry) > 0 {
			head := plain
			if len(head) > zipCarry {
				head = head[:zipCarry]
			}
			join := make([]byte, 0, len(carry)+len(head))
			join = append(append(join, carry...), head...)
			scanZipBuf(join, len(carry), len(carry), base-uint64(len(carry)), total, &out)
		}
		scanZipBuf(plain, len(plain), 0, base, total, &out)
		carry = zipTail(carry, plain)
	}
	return out, nil
}

// 尾巴跨块累积到 zipCarry 字节，空块或比头还短的块不会把前面的尾巴冲掉。
func zipTail(carry, plain []byte) []byte {
	if len(plain) >= zipCarry {
		return append(carry[:0], plain[len(plain)-zipCarry:]...)
	}
	carry = append(carry, plain...)
	if n := len(carry) - zipCarry; n > 0 {
		copy(carry, carry[n:])
		carry = carry[:zipCarry]
	}
	return carry
}

// boundary 是拼接缓冲里前段尾巴的长度；数据区不越过它的头整个落在前一块，前一块整块扫时已收。
func scanZipBuf(buf []byte, limit, boundary int, base, total uint64, out *[]ZipEntry) {
	for j := 0; j < limit && j+30 <= len(buf); j++ {
		if binary.LittleEndian.Uint32(buf[j:]) != ZipLocalMagic {
			continue
		}
		e, ok := zipEntryAt(buf, j)
		if !ok || e.DataOffset <= boundary {
			continue
		}
		if base+uint64(e.DataOffset)+uint64(e.CompSize) > total {
			continue
		}
		e.HeaderAt += int(base)
		e.DataOffset += int(base)
		*out = append(*out, e)
	}
}

func (r *Reader) ExtractZipPlain(e ZipEntry, keys ZipKeys) ([]byte, error) {
	if r == nil || r.idx == nil {
		return nil, fmt.Errorf("ExtractZipPlain %s: no index", e.Name)
	}
	if e.DataOffset < 0 {
		return nil, fmt.Errorf("ExtractZipPlain %s: bad offset", e.Name)
	}
	if e.CompSize == 0 && e.Flag&zipFlagDescriptor != 0 {
		body, err := r.inflateDescriptor(e, keys)
		if err != nil {
			return nil, fmt.Errorf("ExtractZipPlain %s: %w", e.Name, err)
		}
		return body, nil
	}
	var payload []byte
	if e.CompSize > 0 {
		var err error
		if payload, err = r.readPlain(uint64(e.DataOffset), e.CompSize); err != nil {
			return nil, fmt.Errorf("ExtractZipPlain %s: %w", e.Name, err)
		}
	}
	body, err := inflateZipPayload(payload, e, keys)
	if err != nil {
		return nil, fmt.Errorf("ExtractZipPlain %s: %w", e.Name, err)
	}
	return body, nil
}

func (r *Reader) readPlain(off uint64, n uint32) ([]byte, error) {
	if off > math.MaxUint32 {
		return nil, fmt.Errorf("offset %d beyond uint32 pool", off)
	}
	bi := r.blockAt(off)
	if bi < 0 {
		return nil, fmt.Errorf("offset %d not in stor", off)
	}
	return r.read(uint32(bi), uint32(off), n)
}

// 描述符形态头内 comp/crc 为 0：按块喂 flate 到流自然结束（pos 恰停在流尾），再在 pos 前后小窗口找 CRC 与长度同时命中的描述符。
func (r *Reader) inflateDescriptor(e ZipEntry, keys ZipKeys) ([]byte, error) {
	if e.Method != 8 || e.Flag&1 != 0 {
		return nil, fmt.Errorf("descriptor method=%d flag=%#x unsupported", e.Method, e.Flag)
	}
	start := uint64(e.DataOffset)
	total := r.idx.uncompAt[len(r.idx.uncompAt)-1]
	if start >= total {
		return nil, fmt.Errorf("offset %d beyond pool", start)
	}
	src := &plainStream{r: r, pos: start, end: total}
	fr := flate.NewReader(src)
	body, err := io.ReadAll(io.LimitReader(fr, int64(maxInflateBytes)+1))
	_ = fr.Close()
	if err != nil {
		return nil, fmt.Errorf("RawInflate: %w", err)
	}
	if len(body) > maxInflateBytes {
		return nil, fmt.Errorf("RawInflate: too large")
	}
	if e.UncompSize != 0 && int(e.UncompSize) != len(body) {
		return nil, fmt.Errorf("descriptor len %d want %d", len(body), e.UncompSize)
	}
	from := src.pos - zipDescBack
	if from < start {
		from = start
	}
	n := uint64(zipDescWin)
	if from+n > total {
		n = total - from
	}
	win, err := r.readPlain(from, uint32(n))
	if err != nil {
		return nil, fmt.Errorf("descriptor: %w", err)
	}
	want := crc32.ChecksumIEEE(body)
	for k := 0; k+12 <= len(win); k++ {
		if binary.LittleEndian.Uint32(win[k:]) == want && binary.LittleEndian.Uint32(win[k+8:]) == uint32(len(body)) {
			return body, nil
		}
	}
	return nil, fmt.Errorf("descriptor not found after %d bytes, crc=%08x", src.pos-start, want)
}

// 必须实现 io.ByteReader，否则 flate 会套 4K 预读 bufio，pos 会越过描述符。
type plainStream struct {
	r        *Reader
	pos, end uint64
	buf      []byte
}

func (s *plainStream) fill() error {
	if len(s.buf) > 0 {
		return nil
	}
	if s.pos >= s.end {
		return io.EOF
	}
	bi := s.r.blockAt(s.pos)
	if bi < 0 {
		return fmt.Errorf("offset %d not in stor", s.pos)
	}
	plain, err := s.r.blockPlain(bi)
	if err != nil {
		return err
	}
	s.buf = plain[s.pos-s.r.idx.uncompAt[bi]:]
	return nil
}

func (s *plainStream) Read(p []byte) (int, error) {
	if err := s.fill(); err != nil {
		return 0, err
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	s.pos += uint64(n)
	return n, nil
}

func (s *plainStream) ReadByte() (byte, error) {
	if err := s.fill(); err != nil {
		return 0, err
	}
	b := s.buf[0]
	s.buf = s.buf[1:]
	s.pos++
	return b, nil
}

type zipSource struct {
	rd   *Reader
	data []byte
}

// material / engine / launcher 布局没有 stor 表，退回原始 Data 线性扫。
func openZipSource(p *Pkg) zipSource {
	if p == nil {
		return zipSource{}
	}
	if rd, err := OpenReader(p); err == nil && rd.idx != nil {
		return zipSource{rd: rd}
	}
	return zipSource{data: p.Data}
}

func (s zipSource) entries() ([]ZipEntry, error) {
	if s.rd != nil {
		return s.rd.ScanZipPlain()
	}
	return ScanZipEntries(s.data), nil
}

func (s zipSource) extract(e ZipEntry, keys ZipKeys) ([]byte, error) {
	if s.rd != nil {
		return s.rd.ExtractZipPlain(e, keys)
	}
	return ExtractZipEntry(s.data, e, keys)
}
