package spine

import (
	"fmt"
	"strconv"
	"strings"
)

type format struct {
	major        int
	minor        int
	v4           bool
	xy           bool
	strTable     bool
	audio        bool
	ikExtra      bool
	softness     bool
	skinRequired bool
	sequence     bool
}

func newFormat(version string) (format, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return format{}, fmt.Errorf("spine: 版本串 %q", version)
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return format{}, fmt.Errorf("spine: 版本串 %q", version)
	}
	f := format{major: major, minor: minor}
	switch {
	case major == 3 && minor >= 6 && minor <= 8:
		f.audio = minor >= 7
		f.ikExtra = minor >= 7
		f.xy = minor >= 8
		f.strTable = minor >= 8
		f.softness = minor >= 8
		f.skinRequired = minor >= 8
	case major == 4 && minor <= 1:
		f.v4, f.xy, f.strTable, f.audio, f.ikExtra, f.softness, f.skinRequired = true, true, true, true, true, true, true
		f.sequence = minor >= 1
	default:
		return f, fmt.Errorf("spine: 不支持的版本 %s", version)
	}
	return f, nil
}

func looksVersion(v string) bool {
	if len(v) < 3 || len(v) > 16 {
		return false
	}
	dots := 0
	for _, c := range v {
		if c == '.' {
			dots++
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return dots >= 1
}

func Probe(raw []byte) (hash, version string, err error) {
	in := &input{b: raw}
	h, _ := in.strNull()
	v, _ := in.strNull()
	if in.err == nil && looksVersion(v) {
		return h, v, nil
	}
	in = &input{b: raw}
	hv := in.int64()
	v, _ = in.strNull()
	if in.err == nil && looksVersion(v) {
		if hv != 0 {
			h = strconv.FormatInt(hv, 10)
		}
		return h, v, nil
	}
	return "", "", fmt.Errorf("spine: 不是 Spine 二进制骨骼")
}

type parser struct {
	in           *input
	f            format
	sk           *Skeleton
	nonessential bool
}

func ParseSkeleton(raw []byte) (*Skeleton, error) {
	hash, version, err := Probe(raw)
	if err != nil {
		return nil, err
	}
	f, err := newFormat(version)
	if err != nil {
		return nil, err
	}
	in := &input{b: raw}
	if f.v4 {
		in.int64()
	} else {
		in.str()
	}
	in.str()
	sk := &Skeleton{Hash: hash, Version: version}
	p := &parser{in: in, f: f, sk: sk}
	p.header()
	p.bones()
	p.slots()
	p.ikConstraints()
	p.transformConstraints()
	p.pathConstraints()
	p.skins()
	p.events()
	p.animations()
	if in.err != nil {
		return nil, in.err
	}
	if in.pos != len(raw) {
		return nil, fmt.Errorf("spine: 尾部剩余 %d 字节", len(raw)-in.pos)
	}
	return sk, nil
}

func (p *parser) header() {
	in := p.in
	if p.f.xy {
		p.sk.X = in.float()
		p.sk.Y = in.float()
	}
	p.sk.Width = in.float()
	p.sk.Height = in.float()
	p.nonessential = in.bool()
	if p.nonessential {
		p.sk.FPS = in.float()
		p.sk.ImagesPath = in.str()
		if p.f.audio {
			p.sk.AudioPath = in.str()
		}
	}
	if p.f.strTable {
		n := in.count()
		for i := 0; i < n && in.err == nil; i++ {
			in.strings = append(in.strings, in.str())
		}
	}
}

func (p *parser) readName() string {
	if p.f.strTable {
		return p.in.strRef()
	}
	return p.in.str()
}

func (p *parser) readNameNull() (string, bool) {
	if p.f.strTable {
		return p.in.strRefNull()
	}
	return p.in.strNull()
}
