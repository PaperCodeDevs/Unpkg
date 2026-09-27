package lua

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const CapMagic = 0x0141554C

type CapRec struct {
	Tag  string
	Name string
	Body []byte
}

func ReadCap(raw []byte) ([]CapRec, error) {
	var out []CapRec
	i := 0
	for i+8 <= len(raw) {
		if binary.LittleEndian.Uint32(raw[i:]) != CapMagic {
			if i == 0 {
				return nil, fmt.Errorf("lua cap magic")
			}
			break
		}
		i += 4
		nlen := int(binary.LittleEndian.Uint32(raw[i:]))
		i += 4
		if nlen < 0 || i+nlen+4 > len(raw) {
			return nil, fmt.Errorf("lua cap name")
		}
		nm := string(raw[i : i+nlen])
		i += nlen
		slen := int(binary.LittleEndian.Uint32(raw[i:]))
		i += 4
		if slen < 0 || i+slen > len(raw) {
			return nil, fmt.Errorf("lua cap body")
		}
		body := append([]byte(nil), raw[i:i+slen]...)
		i += slen
		tag, name := "", nm
		if strings.HasPrefix(nm, "[") {
			if j := strings.Index(nm, "]"); j >= 0 {
				tag = nm[1:j]
				name = strings.TrimSpace(nm[j+1:])
			}
		}
		out = append(out, CapRec{Tag: tag, Name: name, Body: body})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("lua cap empty")
	}
	return out, nil
}

func CapLastLJ(recs []CapRec) []CapRec {
	idx := map[string]int{}
	var out []CapRec
	for _, r := range recs {
		if r.Tag != "lj" || len(r.Body) < 4 || r.Body[3] != 0x90 {
			continue
		}
		key := filepath.ToSlash(r.Name)
		if key == "" {
			continue
		}
		if i, ok := idx[key]; ok {
			out[i] = r
			continue
		}
		idx[key] = len(out)
		out = append(out, r)
	}
	return out
}

func DumpDecompileCap(srcFile, outRoot string) (Batch, error) {
	var b Batch
	raw, err := os.ReadFile(srcFile)
	if err != nil {
		return b, err
	}
	recs, err := ReadCap(raw)
	if err != nil {
		return b, err
	}
	if err := os.MkdirAll(outRoot, 0o755); err != nil {
		return b, err
	}
	hintDir := filepath.Join(outRoot, "_hint")
	for _, r := range recs {
		if r.Tag != "buff" || !looksLikeLuaSource(r.Body) {
			continue
		}
		rel := hintScriptName(r.Name)
		if rel == "" {
			continue
		}
		dst := filepath.Join(hintDir, sanitizeCapPath(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			continue
		}
		_ = os.WriteFile(dst, r.Body, 0o644)
	}
	for _, r := range CapLastLJ(recs) {
		src, err := Decompile(r.Body)
		if err != nil || len(src) == 0 {
			msg := "empty"
			if err != nil {
				msg = err.Error()
			}
			b.fail(&b.LuaFail, r.Name, msg)
			continue
		}
		rel := strings.TrimSuffix(filepath.ToSlash(r.Name), ".lua") + ".lua"
		dst := filepath.Join(outRoot, sanitizeCapPath(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return b, err
		}
		if err := os.WriteFile(dst, src, 0o644); err != nil {
			return b, err
		}
		b.LuaOK++
	}
	if b.LuaOK == 0 {
		return b, fmt.Errorf("no LuaJIT dump in %s", srcFile)
	}
	return b, nil
}

func looksLikeLuaSource(b []byte) bool {
	if len(b) < 8 || len(b) > 4<<20 {
		return false
	}
	if bytes.HasPrefix(b, []byte{0x1b, 'L', 'J'}) {
		return false
	}
	s := bytes.TrimSpace(b)
	if len(s) == 0 || s[0] == '{' || s[0] == '[' {
		return false
	}
	if !utf8.Valid(s) {
		return false
	}
	head := string(s)
	if strings.HasPrefix(head, "do local ret") {
		return false
	}
	if len(head) > 256 {
		head = head[:256]
	}
	return strings.Contains(head, "function") || strings.Contains(head, "local ") ||
		strings.HasPrefix(head, "return ") ||
		strings.Contains(head, "\nfunction") || strings.Contains(head, "\nlocal ")
}

func hintScriptName(name string) string {
	n := strings.TrimSpace(filepath.ToSlash(name))
	if n == "" || strings.ContainsAny(n, "\n\r") || len(n) > 180 {
		return ""
	}
	low := strings.ToLower(n)
	ok := strings.Contains(low, "luascript/") || strings.Contains(low, "sandboxengine/") ||
		strings.Contains(low, "miniui/") || strings.Contains(low, "ui/") || strings.HasSuffix(low, ".lua")
	if !ok {
		return ""
	}
	if !strings.HasSuffix(low, ".lua") {
		n += ".lua"
	}
	return n
}

func sanitizeCapPath(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	var keep []string
	for _, p := range strings.Split(name, "/") {
		p = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
				return r
			}
			return '_'
		}, p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		keep = append(keep, p)
	}
	if len(keep) == 0 {
		return "_"
	}
	return filepath.Join(keep...)
}
