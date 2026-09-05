package pkg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	engineTypeMax   = 8
	engineTypeBytes = 6
	engineGUIDLen   = 16
	engineOmodMagic = 0x23456789
	engineMaxField  = 64
)

type EngineField struct {
	Kind byte
	Text string
}

type EngineObject struct {
	Offset int
	Type   uint32
	ID     string
	Name   string
	Count  int
	Fields []EngineField
}

func ParseEngineObjects(data []byte) []EngineObject {
	plain, ok := DecodeWrapped(data)
	if !ok {
		plain = data
	}
	obj, ok := ParseEngineAsset("", plain)
	if !ok {
		return nil
	}
	return []EngineObject{obj}
}

func ParseEngineAsset(name string, plain []byte) (EngineObject, bool) {
	obj := EngineObject{Name: name}
	if len(plain) < 8 {
		return obj, false
	}
	inner := ""
	switch {
	case engineTypeTable(&obj, plain):
	case binary.LittleEndian.Uint32(plain[:4]) == engineOmodMagic:
		obj.Type = engineOmodMagic
	case plain[0] == '<':
		inner = engineXMLRoot(plain)
	default:
		inner = lenStrAt(plain, 0)
	}
	if inner != "" {
		obj.Fields = append(obj.Fields, EngineField{Kind: 'n', Text: inner})
	}
	for _, s := range lenStrings(plain, engineMaxField) {
		obj.Fields = append(obj.Fields, EngineField{Kind: 's', Text: s})
		if inner == "" && (strings.Contains(s, "/") || strings.Contains(s, ".")) {
			inner = s
		}
	}
	if obj.Name == "" {
		obj.Name = inner
	}
	return obj, true
}

func engineTypeTable(obj *EngineObject, plain []byte) bool {
	// 解压后 Rainbow 资源头：u32 0 | u32 n | n×{u32 类型 id, u16 版本} | guid[16]
	if binary.LittleEndian.Uint32(plain[:4]) != 0 {
		return false
	}
	n := int(binary.LittleEndian.Uint32(plain[4:8]))
	if n < 1 || n > engineTypeMax || 8+n*engineTypeBytes+engineGUIDLen > len(plain) {
		return false
	}
	for i := 0; i < n; i++ {
		off := 8 + i*engineTypeBytes
		id := binary.LittleEndian.Uint32(plain[off:])
		ver := binary.LittleEndian.Uint16(plain[off+4:])
		if i == 0 {
			obj.Type = id
		}
		obj.Fields = append(obj.Fields, EngineField{Kind: 't', Text: fmt.Sprintf("%08x:v%d", id, ver)})
	}
	obj.Count = n
	g := 8 + n*engineTypeBytes
	obj.ID = hex.EncodeToString(plain[g : g+engineGUIDLen])
	return true
}

func engineXMLRoot(plain []byte) string {
	end := strings.IndexAny(string(plain[:min(len(plain), 128)]), " >\r\n")
	if end < 0 {
		return ""
	}
	return strings.TrimPrefix(string(plain[:end]), "<")
}

func (r *Reader) EngineObjects() []EngineObject {
	if r == nil {
		return nil
	}
	var out []EngineObject
	for _, n := range r.Names("") {
		if isDXHashName(n) {
			continue
		}
		b, err := r.Lookup(n)
		if err != nil || len(b) == dxHashLen {
			continue
		}
		plain, ok := DecodeWrapped(b)
		if !ok {
			plain = b
		}
		obj, ok := ParseEngineAsset(n, plain)
		if !ok {
			continue
		}
		obj.Offset = r.entryOffset(n)
		out = append(out, obj)
	}
	return out
}

func (r *Reader) entryOffset(name string) int {
	if r.launcher == nil {
		return -1
	}
	if alt, ok := r.lower[strings.ToLower(name)]; ok {
		name = alt
	}
	flag, ok := r.launcher.byName[name]
	if !ok {
		return -1
	}
	return r.launcher.recordOffset(name, flag)
}
