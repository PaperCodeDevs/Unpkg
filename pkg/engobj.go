package pkg

import (
	"encoding/binary"
	"encoding/hex"
)

const (
	engineObjTypeOff = 7
	engineObjIDOff   = 28
	engineObjCntOff  = 44
	engineObjFldOff  = 47
	engineObjIDLen   = 16
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
	if len(data) == 0 {
		return nil
	}
	marks := findEngineMarks(data)
	out := make([]EngineObject, 0, len(marks))
	for i, m := range marks {
		end := len(data)
		if i+1 < len(marks) {
			end = marks[i+1]
		} else if m+1<<20 < end {
			end = m + 1<<20
		}
		out = append(out, parseEngineObj(data, m, end))
	}
	return out
}

func findEngineMarks(data []byte) []int {
	mark := []byte(engineObjMark)
	var pos []int
	for i := 0; i+len(mark) <= len(data); i++ {
		ok := true
		for j := 0; j < len(mark); j++ {
			if data[i+j] != mark[j] {
				ok = false
				break
			}
		}
		if ok {
			pos = append(pos, i)
			i += len(mark) - 1
		}
	}
	return pos
}

func parseEngineObj(data []byte, m, end int) EngineObject {
	obj := EngineObject{Offset: m}
	if m+engineObjTypeOff+4 <= len(data) {
		obj.Type = binary.LittleEndian.Uint32(data[m+engineObjTypeOff:])
	}
	if m+engineObjIDOff+engineObjIDLen <= len(data) {
		obj.ID = hex.EncodeToString(data[m+engineObjIDOff : m+engineObjIDOff+engineObjIDLen])
	}
	if m+engineObjCntOff < len(data) {
		obj.Count = int(data[m+engineObjCntOff])
	}
	start := m + engineObjFldOff
	if start > end {
		start = m
	}
	obj.Fields = scanFFFields(data, start, end)
	if len(obj.Fields) > 0 {
		obj.Name = obj.Fields[0].Text
	}
	return obj
}

func scanFFFields(data []byte, lo, hi int) []EngineField {
	var out []EngineField
	i := lo
	for i+6 < hi {
		if data[i] != 0xff {
			i++
			continue
		}
		kind := data[i+1]
		n := int(binary.LittleEndian.Uint32(data[i+2 : i+6]))
		s0 := i + 6
		if n < 2 || n > 300 || s0+n > len(data) || s0+n > hi+32 {
			i++
			continue
		}
		s := data[s0 : s0+n]
		if !ffPrintable(s) {
			i++
			continue
		}
		out = append(out, EngineField{Kind: kind, Text: string(s)})
		i = s0 + n
	}
	return out
}

func ffPrintable(s []byte) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
