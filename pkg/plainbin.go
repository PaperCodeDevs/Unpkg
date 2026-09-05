package pkg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"
)

func PBDescriptorMessages(b []byte) ([]string, error) {
	var out []string
	err := pbEach(b, func(field, wt int, v []byte) error {
		if field != 1 || wt != 2 {
			return nil
		}
		pkg, msgs := "", []string{}
		err := pbEach(v, func(f, w int, vv []byte) error {
			switch {
			case f == 2 && w == 2:
				pkg = string(vv)
			case f == 4 && w == 2:
				return pbEach(vv, func(f2, w2 int, v2 []byte) error {
					if f2 == 1 && w2 == 2 {
						msgs = append(msgs, string(v2))
					}
					return nil
				})
			}
			return nil
		})
		for _, m := range msgs {
			out = append(out, pkg+"."+m)
		}
		return err
	})
	if err == nil && len(out) == 0 {
		err = errors.New("pbdesc: no messages")
	}
	return out, err
}

func pbEach(b []byte, fn func(field, wt int, v []byte) error) error {
	for len(b) > 0 {
		tag, k := binary.Uvarint(b)
		if k <= 0 {
			return errors.New("pb: bad tag")
		}
		b = b[k:]
		field, wt := int(tag>>3), int(tag&7)
		size := 0
		switch wt {
		case 0:
			if _, size = binary.Uvarint(b); size <= 0 {
				return errors.New("pb: bad varint")
			}
		case 1:
			size = 8
		case 5:
			size = 4
		case 2:
			l, k := binary.Uvarint(b)
			if k <= 0 || l > uint64(len(b)-k) {
				return errors.New("pb: bad length")
			}
			b = b[k:]
			size = int(l)
		default:
			return fmt.Errorf("pb: wire type %d", wt)
		}
		if size > len(b) {
			return errors.New("pb: truncated")
		}
		if err := fn(field, wt, b[:size]); err != nil {
			return err
		}
		b = b[size:]
	}
	return nil
}

var singleChunkMagic = []byte{0x0c, 0x1c, 0x07, 0x06}

func regionLocCount(b []byte) (int, string, error) {
	if bytes.HasPrefix(b, singleChunkMagic) {
		if len(b) < 23 || 23+int(binary.LittleEndian.Uint32(b[15:])) > len(b) {
			return 0, "", errors.New("region: single chunk truncated")
		}
		return 1, "single", nil
	}
	if len(b) < 8192 || len(b)%4096 != 0 {
		return 0, "", fmt.Errorf("region: size %d", len(b))
	}
	n := 0
	for i := 0; i < 4096; i += 4 {
		if binary.LittleEndian.Uint32(b[i:]) != 0 {
			n++
		}
	}
	if n == 0 {
		return 0, "", errors.New("region: empty loc table")
	}
	return n, "sectors", nil
}

func flatBufferRootOK(b []byte) error {
	if len(b) < 8 || int(binary.LittleEndian.Uint32(b))+4 > len(b) {
		return errors.New("flatbuffer: bad root offset")
	}
	return nil
}

// 头部：u32 版本表条数 N，随后 N 组 (u32 类哈希, u16 版本)，再接 u32 SandboxTimelineTypeId。
func timelineHeader(b []byte) (int, string, error) {
	if len(b) < 8 {
		return 0, "", errors.New("timeline: short")
	}
	n := int(binary.LittleEndian.Uint32(b))
	if n < 1 || n > 64 || len(b) < 4+6*n+4 {
		return 0, "", fmt.Errorf("timeline: version map %d", n)
	}
	for i := 0; i < n; i++ {
		if v := binary.LittleEndian.Uint16(b[4+6*i+4:]); v == 0 || v > 255 {
			return 0, "", fmt.Errorf("timeline: version %d", v)
		}
	}
	return n, fmt.Sprintf("type=%d", binary.LittleEndian.Uint32(b[4+6*n:])), nil
}

func verifyZipMembers(body []byte, keys ZipKeys) (int, []string, error) {
	ents := ScanZipEntries(body)
	if len(ents) == 0 {
		return 0, nil, errors.New("zip: no entries")
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		plain, err := ExtractZipEntry(body, e, keys)
		if err != nil {
			return len(ents), names, err
		}
		if lenOK, crcOK := VerifyZipEntry(plain, e); !lenOK || !crcOK {
			return len(ents), names, fmt.Errorf("zip: crc %s", e.Name)
		}
		names = append(names, e.Name)
	}
	return len(ents), names, nil
}

func luaJITHeaderOK(b []byte) error {
	if len(b) < 5 || b[3] != 0x90 && b[3] != 1 && b[3] != 2 {
		return fmt.Errorf("luajit: version %#x", b[minInt(3, len(b)-1)])
	}
	return nil
}

func binJSONHeaderOK(b []byte) error {
	if len(b) < 5 || binary.LittleEndian.Uint32(b) > 1<<16 {
		return errors.New("binjson: bad string table count")
	}
	return nil
}

func isTextual(b []byte) bool {
	n := minInt(len(b), 256)
	for _, c := range b[:n] {
		if c < 0x09 || (c > 0x0d && c < 0x20) {
			return false
		}
	}
	return utf8.Valid(b[:n]) || decodeCsvGBK(b[:n]) != nil
}
