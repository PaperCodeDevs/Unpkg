package zdeflate_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/pkg"
	"github.com/PaperCodeDevs/Unpkg/zdeflate"
)

var ugcZipKeys = pkg.ZipKeys{K0: 0x1f85d854, K1: 0xdbaf3374, K2: 0xc4d7be09}

type gameMem struct {
	Path, Name, Kind string
	Plain, Comp      []byte
}

func sampleRoots() []string {
	var out []string
	if d := os.Getenv("APPDATA"); d != "" {
		out = append(out, filepath.Join(d, "miniworddata110", "data"))
	}
	_, file, _, ok := runtime.Caller(0)
	if ok {
		out = append(out, filepath.Join(filepath.Dir(file), "..", "..", "..", ".temp", "rev-reslist2", "samples"))
	}
	return out
}

func isUgcZipName(name string) bool {
	n := strings.ToLower(name)
	if n == "resourcelist.data" {
		return true
	}
	for _, s := range []string{".script", ".trigger", ".uprefab", ".uscene"} {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

func loadGameMembers() (mems []gameMem, files int) {
	seen := map[string]struct{}{}
	for _, root := range sampleRoots() {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || !isUgcZipName(info.Name()) {
				return err
			}
			raw, err := os.ReadFile(p)
			if err != nil || len(raw) < 30 || binary.LittleEndian.Uint32(raw) != pkg.ZipLocalMagic {
				return nil
			}
			abs, _ := filepath.Abs(p)
			if _, ok := seen[abs]; ok {
				return nil
			}
			seen[abs] = struct{}{}
			ms := parseGameZip(p, raw)
			if len(ms) == 0 {
				return nil
			}
			files++
			mems = append(mems, ms...)
			return nil
		})
	}
	return mems, files
}

func parseGameZip(path string, raw []byte) []gameMem {
	var out []gameMem
	n := len(raw)
	i := 0
	for i+30 <= n {
		if binary.LittleEndian.Uint32(raw[i:]) != pkg.ZipLocalMagic {
			break
		}
		ver := binary.LittleEndian.Uint16(raw[i+4:])
		flag := binary.LittleEndian.Uint16(raw[i+6:])
		method := binary.LittleEndian.Uint16(raw[i+8:])
		dost := binary.LittleEndian.Uint16(raw[i+10:])
		dosd := binary.LittleEndian.Uint16(raw[i+12:])
		crc := binary.LittleEndian.Uint32(raw[i+14:])
		csz := int(binary.LittleEndian.Uint32(raw[i+18:]))
		nlen := int(binary.LittleEndian.Uint16(raw[i+26:]))
		elen := int(binary.LittleEndian.Uint16(raw[i+28:]))
		if i+30+nlen+elen+csz > n {
			break
		}
		name := string(raw[i+30 : i+30+nlen])
		off := i + 30 + nlen + elen
		payload := raw[off : off+csz]
		i = off + csz
		if ver != 20 || flag != 3 || method != 8 || dost != 0 || dosd != 32 || elen != 0 || csz < 12 {
			continue
		}
		dec := pkg.DecryptZipCrypto(payload, ugcZipKeys.K0, ugcZipKeys.K1, ugcZipKeys.K2)
		zero := true
		for _, b := range dec[:11] {
			if b != 0 {
				zero = false
				break
			}
		}
		if zero || spawnEnc10(crc, dec[:10]) {
			continue
		}
		plain, err := pkg.RawInflate(dec[12:])
		if err != nil {
			continue
		}
		out = append(out, gameMem{
			Path: path, Name: name, Kind: memberKind(name),
			Plain: plain, Comp: append([]byte(nil), dec[12:]...),
		})
	}
	return out
}

func memberKind(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".script"):
		return "script"
	case strings.HasSuffix(n, ".trigger"):
		return "trigger"
	case strings.HasSuffix(n, ".json"):
		return "json"
	case strings.HasSuffix(n, ".lua"):
		return "lua"
	case strings.HasSuffix(n, ".txt"):
		return "txt"
	default:
		return "bin"
	}
}

func spawnEnc10(crc uint32, got []byte) bool {
	if len(got) < 10 {
		return false
	}
	s := crc ^ 0x9e3779b9
	var hdr [10]byte
	for i := 0; i < 10; i++ {
		s = s*1664525 + 1013904223
		hdr[i] = byte(s >> 24)
	}
	zero := true
	for i := 0; i < 10; i++ {
		if hdr[i] != 0 {
			zero = false
			break
		}
	}
	if zero {
		hdr[0] = 1
	}
	for i := 0; i < 10; i++ {
		if got[i] != hdr[i] {
			return false
		}
	}
	return true
}

func recompressOK(m gameMem) bool {
	got := zdeflate.Compress(m.Plain)
	if len(got) != len(m.Comp) {
		return false
	}
	for i := range got {
		if got[i] != m.Comp[i] {
			return false
		}
	}
	return true
}
