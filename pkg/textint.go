package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

func TintKey(stem string, color uint32) string {
	return fmt.Sprintf("%s_%x", strings.ToLower(strings.TrimSpace(stem)), color)
}

func ParseHexColor(s string) (uint32, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "#")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func (r *Reader) HasBlockTex(stem string) bool {
	if r == nil || r.bases == nil {
		return false
	}
	_, ok := r.bases[strings.ToLower(strings.TrimSpace(stem))]
	return ok
}

func (r *Reader) TintFace(stem string, color uint32) string {
	if stem == "" {
		return stem
	}
	tinted := TintKey(stem, color)
	if r.HasBlockTex(tinted) {
		return tinted
	}
	return stem
}

func (r *Reader) CubeFacesRuntime(tex1, tex2, typ string, grassColor uint32) [6]string {
	faces := r.CubeFacesTyped(tex1, tex2, typ)
	if grassColor == 0 {
		return faces
	}
	for i := range faces {
		faces[i] = r.TintFace(faces[i], grassColor)
	}
	return faces
}
