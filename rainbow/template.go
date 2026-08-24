package rainbow

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/pkg"
)

const (
	KindLZ4  = "lz4"
	KindVer1 = "ver1"
	maxPlain = 8 << 20
	minStr   = 2
	maxStr   = 512
)

type TemplateMat struct {
	Name     string
	Kind     string
	HLSL     []string
	Slots    []string
	Keywords []string
}

func ParseTemplateMat(raw []byte) (*TemplateMat, error) {
	plain, kind, err := openTemplate(raw)
	if err != nil {
		return nil, err
	}
	t := &TemplateMat{Kind: kind}
	strs := u32Strings(plain)
	t.HLSL = hlslPaths(plain, strs)
	for _, s := range strs {
		switch {
		case isAssetName(s):
			if t.Name == "" {
				t.Name = s
			}
		case isKeyword(s):
			t.Keywords = appendUniq(t.Keywords, s)
		case isSlot(s):
			t.Slots = appendUniq(t.Slots, s)
		}
	}
	for _, s := range declareTex(plain) {
		t.Slots = appendUniq(t.Slots, s)
	}
	if t.Name == "" {
		t.Name = firstGUID(strs)
	}
	return t, nil
}

func openTemplate(raw []byte) ([]byte, string, error) {
	if len(raw) < 8 {
		return nil, "", fmt.Errorf("templatemat short")
	}
	n := int(binary.LittleEndian.Uint32(raw[:4]))
	if n == 1 {
		return raw, KindVer1, nil
	}
	if n < 16 || n > maxPlain {
		return nil, "", fmt.Errorf("templatemat size")
	}
	dst, err := pkg.DecompressLZ4Block(raw[4:], n+64)
	if err != nil {
		return nil, "", fmt.Errorf("templatemat lz4: %w", err)
	}
	if len(dst) > n {
		dst = dst[:n]
	}
	if len(dst) < n*8/10 {
		return nil, "", fmt.Errorf("templatemat lz4 short")
	}
	return dst, KindLZ4, nil
}

func u32Strings(b []byte) []string {
	var out []string
	seen := map[string]struct{}{}
	for i := 0; i+4 <= len(b); {
		n := int(binary.LittleEndian.Uint32(b[i:]))
		if n < minStr || n > maxStr || i+4+n > len(b) || !printable(b[i+4:i+4+n]) {
			i++
			continue
		}
		s := string(b[i+4 : i+4+n])
		if !usableStr(s) {
			i++
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
		i += 4 + n
	}
	return out
}

func printable(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return len(b) > 0
}

func usableStr(s string) bool {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			n++
		}
	}
	return n >= 2
}

func isAssetName(s string) bool {
	low := strings.ToLower(s)
	return strings.HasSuffix(low, ".templatemat") || strings.HasSuffix(low, ".mat")
}

func isIdent(s string) bool {
	if len(s) < 2 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	c0 := s[0]
	return c0 == '_' || c0 >= 'A' && c0 <= 'Z' || c0 >= 'a' && c0 <= 'z'
}

func isSlot(s string) bool {
	if !isIdent(s) {
		return false
	}
	if strings.HasPrefix(s, "g_") || strings.HasPrefix(s, "_") {
		return len(s) >= 4
	}
	for _, suf := range []string{"Tex", "Texture", "Color", "Float", "Scale", "Map"} {
		if strings.HasSuffix(s, suf) && len(s) > len(suf) {
			return true
		}
	}
	return false
}

func isKeyword(s string) bool {
	if !isIdent(s) {
		return false
	}
	if strings.HasPrefix(s, "ENABLE_") || strings.HasPrefix(s, "SHADING_MODEL_") || strings.HasPrefix(s, "MATERIAL_") {
		return true
	}
	if s == "EMISSIVE" {
		return true
	}
	if len(s) < 6 || !strings.Contains(s, "_") {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

func firstGUID(strs []string) string {
	for _, s := range strs {
		if len(s) == 32 && isHex(s) {
			return s
		}
	}
	return ""
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
			continue
		}
		return false
	}
	return true
}

func appendUniq(dst []string, s string) []string {
	for _, x := range dst {
		if x == s {
			return dst
		}
	}
	return append(dst, s)
}

var (
	reInclude = regexp.MustCompile(`#include\s+"([^"]+\.hlsl[i]?)"`)
	reHLSL    = regexp.MustCompile(`(?i)[A-Za-z0-9_./\\-]+\.hlsli?`)
	reDecl    = regexp.MustCompile(`DECLARE_TEX2D\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)`)
)

func hlslPaths(plain []byte, strs []string) []string {
	var out []string
	add := func(s string) {
		s = strings.ReplaceAll(strings.TrimSpace(s), `\`, `/`)
		if s == "" {
			return
		}
		out = appendUniq(out, s)
	}
	for _, m := range reInclude.FindAllSubmatch(plain, -1) {
		add(string(m[1]))
	}
	for _, s := range strs {
		low := strings.ToLower(s)
		if !strings.Contains(low, ".hlsl") {
			continue
		}
		if m := reInclude.FindSubmatch([]byte(s)); len(m) == 2 {
			add(string(m[1]))
			continue
		}
		for _, h := range reHLSL.FindAllString(s, -1) {
			add(h)
		}
	}
	return out
}

func declareTex(plain []byte) []string {
	var out []string
	for _, m := range reDecl.FindAllSubmatch(plain, -1) {
		out = appendUniq(out, string(m[1]))
	}
	return out
}
