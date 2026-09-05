package pkg

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	fguiMagic         = "FGUI"
	fguiTypeImage     = 0
	fguiTypeMovieClip = 1
	fguiTypeSound     = 2
	fguiTypeComponent = 3
	fguiTypeAtlas     = 4
	fguiTypeFont      = 5
	fguiTypeMisc      = 7
)

type fguiPkg struct {
	ver    int
	id     string
	name   string
	asset  string
	items  []fguiItem
	byName map[string]fguiItem
	byID   map[string]fguiItem
	sprite map[string]fguiSprite
}

type fguiItem struct {
	typ      int
	id, name string
	file     string
	w, h     int
}

type fguiSprite struct {
	atlas string
	file  string
	x, y  int
	w, h  int
	rot   bool
}

type fguiBuf struct {
	b   []byte
	pos int
}

func (z *fguiBuf) has(n int) bool {
	return z.pos >= 0 && n >= 0 && z.pos <= len(z.b)-n
}

func (z *fguiBuf) u8() byte {
	if !z.has(1) {
		return 0
	}
	v := z.b[z.pos]
	z.pos++
	return v
}

func (z *fguiBuf) i16() int {
	if !z.has(2) {
		return 0
	}
	v := int(int16(binary.BigEndian.Uint16(z.b[z.pos:])))
	z.pos += 2
	return v
}

func (z *fguiBuf) u16() int {
	if !z.has(2) {
		return 0
	}
	v := int(binary.BigEndian.Uint16(z.b[z.pos:]))
	z.pos += 2
	return v
}

func (z *fguiBuf) i32() int {
	if !z.has(4) {
		return 0
	}
	v := int(int32(binary.BigEndian.Uint32(z.b[z.pos:])))
	z.pos += 4
	return v
}

func (z *fguiBuf) str() string {
	n := z.u16()
	if n < 0 || !z.has(n) {
		return ""
	}
	s := string(z.b[z.pos : z.pos+n])
	z.pos += n
	return s
}

func (z *fguiBuf) s(tab []string) string {
	i := z.u16()
	if i == 65534 || i == 65533 || i < 0 || i >= len(tab) {
		return ""
	}
	return tab[i]
}

func (z *fguiBuf) skip(n int) {
	z.pos += n
}

func (z *fguiBuf) seek(indexPos, block int) bool {
	old := z.pos
	if indexPos < 0 || indexPos >= len(z.b) {
		return false
	}
	z.pos = indexPos
	seg := int(z.u8())
	if block < 0 || block >= seg {
		z.pos = old
		return false
	}
	useShort := z.u8() == 1
	var np int
	if useShort {
		z.pos += 2 * block
		if z.pos+2 > len(z.b) {
			z.pos = old
			return false
		}
		np = z.i16()
	} else {
		z.pos += 4 * block
		if z.pos+4 > len(z.b) {
			z.pos = old
			return false
		}
		np = z.i32()
	}
	if np <= 0 {
		z.pos = old
		return false
	}
	z.pos = indexPos + np
	if z.pos < 0 || z.pos >= len(z.b) {
		z.pos = old
		return false
	}
	return true
}

func parseFGUI(raw []byte, asset string) (*fguiPkg, error) {
	if len(raw) < 16 || string(raw[:4]) != fguiMagic {
		return nil, fmt.Errorf("parseFGUI: magic")
	}
	z := &fguiBuf{b: raw, pos: 4}
	ver := z.i32()
	z.u8()
	id := z.str()
	name := z.str()
	z.skip(20)
	indexPos := z.pos
	if !z.seek(indexPos, 4) {
		return nil, fmt.Errorf("parseFGUI: 无字符串表")
	}
	n := z.i32()
	if n < 0 || n > 500000 {
		return nil, fmt.Errorf("parseFGUI: 字符串数 %d", n)
	}
	tab := make([]string, n)
	for i := 0; i < n; i++ {
		tab[i] = z.str()
	}
	p := &fguiPkg{
		ver: ver, id: id, name: name, asset: asset,
		byName: map[string]fguiItem{},
		byID:   map[string]fguiItem{},
		sprite: map[string]fguiSprite{},
	}
	if !z.seek(indexPos, 1) {
		return nil, fmt.Errorf("parseFGUI: 无资源表")
	}
	if err := p.readItems(z, tab); err != nil {
		return nil, err
	}
	if !z.seek(indexPos, 2) {
		return nil, fmt.Errorf("parseFGUI: 无图集切片")
	}
	if err := p.readSprites(z, tab); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *fguiPkg) readItems(z *fguiBuf, tab []string) error {
	cnt := z.i16()
	if cnt < 0 || cnt > 200000 {
		return fmt.Errorf("parseFGUI: 资源数 %d", cnt)
	}
	for i := 0; i < cnt; i++ {
		if z.pos+4 > len(z.b) {
			return fmt.Errorf("parseFGUI: 资源截断")
		}
		next := z.i32() + z.pos
		if next < z.pos || next > len(z.b) {
			return fmt.Errorf("parseFGUI: 资源边界")
		}
		it := fguiItem{typ: int(z.u8()), id: z.s(tab), name: z.s(tab)}
		z.skip(2)
		it.file = z.s(tab)
		z.u8()
		it.w = z.i32()
		it.h = z.i32()
		if it.typ == fguiTypeAtlas || it.typ == fguiTypeSound || it.typ == fguiTypeMisc {
			it.file = p.asset + "_" + it.file
		}
		p.items = append(p.items, it)
		p.byID[it.id] = it
		if it.name != "" {
			if _, ok := p.byName[it.name]; !ok {
				p.byName[it.name] = it
			}
		}
		z.pos = next
	}
	return nil
}

func (p *fguiPkg) readSprites(z *fguiBuf, tab []string) error {
	cnt := z.i16()
	if cnt < 0 || cnt > 200000 {
		return fmt.Errorf("parseFGUI: 切片数 %d", cnt)
	}
	for i := 0; i < cnt; i++ {
		if z.pos+2 > len(z.b) {
			return fmt.Errorf("parseFGUI: 切片截断")
		}
		next := z.u16() + z.pos
		if next < z.pos || next > len(z.b) {
			return fmt.Errorf("parseFGUI: 切片边界")
		}
		itemID := z.s(tab)
		atlasID := z.s(tab)
		sp := fguiSprite{atlas: atlasID, x: z.i32(), y: z.i32(), w: z.i32(), h: z.i32(), rot: z.u8() == 1}
		if at, ok := p.byID[atlasID]; ok {
			sp.file = at.file
		}
		p.sprite[itemID] = sp
		z.pos = next
	}
	return nil
}

func fguiAssetPath(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.TrimSuffix(name, ".fui")
}
