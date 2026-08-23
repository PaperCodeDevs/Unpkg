package pkg

import (
	"bytes"
	"path/filepath"
	"strings"
)

var luaJITFileMagic = []byte{0x1b, 'L', 'J', 0x90}

type embeddedLua struct {
	Path string
	Body []byte
}

func extractEmbeddedLuaSources(plain []byte) []embeddedLua {
	var out []embeddedLua
	from := 0
	for {
		i := bytes.Index(plain[from:], []byte(".lua"))
		if i < 0 {
			break
		}
		pos := from + i
		start := pos
		for start > 0 && isLuaPathByte(plain[start-1]) {
			start--
		}
		path := string(plain[start : pos+4])
		from = pos + 4
		if len(path) < 8 || !validEmbeddedLuaPath(path) {
			continue
		}
		if !LooksLikeLuaSource(plain[from:minInt(len(plain), from+64)]) {
			continue
		}
		end := findLuaSourceEnd(plain, from)
		if end <= from {
			continue
		}
		body := append([]byte(nil), plain[from:end]...)
		if !LooksLikeLuaSource(body) {
			continue
		}
		out = append(out, embeddedLua{Path: filepath.ToSlash(path), Body: body})
	}
	return out
}

func extractLuaJITByPath(plain []byte) []embeddedLua {
	var out []embeddedLua
	from := 0
	for {
		i := bytes.Index(plain[from:], []byte(".lua"))
		if i < 0 {
			break
		}
		pos := from + i
		start := pos
		for start > 0 && isLuaPathByte(plain[start-1]) {
			start--
		}
		path := string(plain[start : pos+4])
		from = pos + 4
		if !validEmbeddedLuaPath(path) {
			continue
		}
		lj := findLuaJITAfter(plain, from, 1024)
		if lj < 0 {
			continue
		}
		end := findLuaJITEnd(plain, lj)
		if end <= lj+8 {
			continue
		}
		body := append([]byte(nil), plain[lj:end]...)
		if !bytes.HasPrefix(body, luaJITFileMagic) {
			continue
		}
		out = append(out, embeddedLua{Path: filepath.ToSlash(path), Body: body})
	}
	return out
}

func findLuaJITEnd(plain []byte, lj int) int {
	const maxBC = 96 * 1024
	limit := minInt(len(plain), lj+maxBC)
	// next .lua path often marks next embedded file
	from := lj + 8
	for {
		i := bytes.Index(plain[from:limit], []byte(".lua"))
		if i < 0 {
			break
		}
		pos := from + i
		start := pos
		for start > lj && isLuaPathByte(plain[start-1]) {
			start--
		}
		cand := string(plain[start : pos+4])
		if validEmbeddedLuaPath(cand) && start > lj+8 {
			return start
		}
		from = pos + 4
	}
	if n := findNextLuaJITFile(plain, lj+4); n > lj+8 && n < limit {
		return n
	}
	return limit
}

func validEmbeddedLuaPath(path string) bool {
	if len(path) < 8 || len(path) > 180 {
		return false
	}
	if !strings.Contains(path, "/") && !strings.Contains(path, "\\") {
		return false
	}
	if strings.Count(path, "luascript") > 1 {
		return false
	}
	if strings.Contains(path, "..") {
		return false
	}
	return true
}

func findLuaJITAfter(b []byte, from, limit int) int {
	end := minInt(len(b)-4, from+limit)
	for i := from; i < end; i++ {
		if bytes.Equal(b[i:i+4], luaJITFileMagic) {
			return i
		}
	}
	return -1
}

func findNextLuaJITFile(b []byte, from int) int {
	for i := from; i+4 <= len(b); i++ {
		if !bytes.Equal(b[i:i+4], luaJITFileMagic) {
			continue
		}
		if i >= 8 {
			window := b[maxInt(0, i-96):i]
			if bytes.Contains(window, []byte(".lua")) {
				return i
			}
		}
	}
	return -1
}

func findLuaSourceEnd(b []byte, from int) int {
	max := minInt(len(b), from+512*1024)
	for i := from; i < max; i++ {
		if b[i] == 0 {
			return i
		}
		if i+4 <= max && bytes.Equal(b[i:i+4], luaJITFileMagic) {
			return i
		}
	}
	return -1
}

func LooksLikeLuaSource(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	n := len(b)
	if n > 240 {
		n = 240
	}
	head := string(b[:n])
	if strings.Contains(head, "-- version") || strings.Contains(head, "--version") {
		return true
	}
	if strings.Contains(head, "function ") || strings.Contains(head, "function(") {
		return true
	}
	if strings.Contains(head, "local ") || strings.Contains(head, "return ") || strings.Contains(head, "return{") {
		return true
	}
	trim := bytes.TrimSpace(b)
	if bytes.HasPrefix(trim, []byte("return ")) || bytes.HasPrefix(trim, []byte("return{")) || bytes.HasPrefix(trim, []byte("return\n")) {
		return true
	}
	return false
}

func isLuaPathByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '/' || c == '\\' || c == '.'
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
