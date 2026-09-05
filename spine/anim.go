package spine

const (
	curveStepped = 1
	curveBezier  = 2
)

var boneKinds3 = []string{"rotate", "translate", "scale", "shear"}

var pathKinds = []string{"pathposition", "pathspacing", "pathmix"}

type animBuilder struct {
	a Animation
}

func (b *animBuilder) add(kind string, target int, att string, times []float32) {
	for _, t := range times {
		if t > b.a.Duration {
			b.a.Duration = t
		}
	}
	b.a.Timelines = append(b.a.Timelines, Timeline{Kind: kind, Target: target, Attachment: att, Frames: len(times), Times: times})
}

func (p *parser) animations() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		name := in.str()
		if p.f.v4 {
			p.sk.Animations = append(p.sk.Animations, p.animation4(name))
		} else {
			p.sk.Animations = append(p.sk.Animations, p.animation3(name))
		}
	}
}

func (p *parser) curve3() {
	if p.in.byte() == curveBezier {
		p.in.floats(4)
	}
}

func (p *parser) frames3(frames, values int) []float32 {
	in := p.in
	times := make([]float32, frames)
	for f := 0; f < frames && in.err == nil; f++ {
		times[f] = in.float()
		in.floats(values)
		if f < frames-1 {
			p.curve3()
		}
	}
	return times
}

func (p *parser) animation3(name string) Animation {
	in := p.in
	b := &animBuilder{a: Animation{Name: name}}
	p.slotTimelines3(b)
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		bone := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := int(in.byte())
			frames := in.count()
			if typ >= len(boneKinds3) {
				in.fail("bone timeline 类型 %d", typ)
				break
			}
			values := 2
			if typ == 0 {
				values = 1
			}
			b.add(boneKinds3[typ], bone, "", p.frames3(frames, values))
		}
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		frames := in.count()
		times := make([]float32, frames)
		for f := 0; f < frames && in.err == nil; f++ {
			times[f] = in.float()
			in.float()
			if p.f.softness {
				in.float()
			}
			in.sbyte()
			if p.f.ikExtra {
				in.bool()
				in.bool()
			}
			if f < frames-1 {
				p.curve3()
			}
		}
		b.add("ik", idx, "", times)
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		b.add("transform", idx, "", p.frames3(in.count(), 4))
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := int(in.byte())
			frames := in.count()
			switch typ {
			case 0, 1:
				b.add(pathKinds[typ], idx, "", p.frames3(frames, 1))
			case 2:
				b.add(pathKinds[typ], idx, "", p.frames3(frames, 2))
			default:
				in.fail("path timeline 类型 %d", typ)
			}
		}
	}
	p.deformTimelines3(b)
	p.orderEventTimelines(b)
	return b.a
}

func (p *parser) slotTimelines3(b *animBuilder) {
	in := p.in
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		slot := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := in.byte()
			frames := in.count()
			times := make([]float32, frames)
			switch typ {
			case 0:
				for f := 0; f < frames && in.err == nil; f++ {
					times[f] = in.float()
					p.readName()
				}
				b.add("attachment", slot, "", times)
			case 1, 2:
				for f := 0; f < frames && in.err == nil; f++ {
					times[f] = in.float()
					in.int32()
					if typ == 2 {
						in.int32()
					}
					if f < frames-1 {
						p.curve3()
					}
				}
				kind := "color"
				if typ == 2 {
					kind = "twocolor"
				}
				b.add(kind, slot, "", times)
			default:
				in.fail("slot timeline 类型 %d", typ)
			}
		}
	}
}

func (p *parser) deformTimelines3(b *animBuilder) {
	in := p.in
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			slot := in.varint(true)
			for k, nnn := 0, in.count(); k < nnn && in.err == nil; k++ {
				att := p.readName()
				frames := in.count()
				times := make([]float32, frames)
				for f := 0; f < frames && in.err == nil; f++ {
					times[f] = in.float()
					if end := in.count(); end != 0 {
						in.varint(true)
						in.floats(end)
					}
					if f < frames-1 {
						p.curve3()
					}
				}
				b.add("deform", slot, att, times)
			}
		}
	}
}

func (p *parser) orderEventTimelines(b *animBuilder) {
	in := p.in
	if n := in.count(); n > 0 {
		times := make([]float32, n)
		for f := 0; f < n && in.err == nil; f++ {
			times[f] = in.float()
			for k, oc := 0, in.count(); k < oc && in.err == nil; k++ {
				in.varint(true)
				in.varint(true)
			}
		}
		b.add("draworder", -1, "", times)
	}
	if n := in.count(); n > 0 {
		times := make([]float32, n)
		for f := 0; f < n && in.err == nil; f++ {
			times[f] = in.float()
			idx := in.varint(true)
			if idx < 0 || idx >= len(p.sk.Events) {
				in.fail("event 索引 %d/%d pos=%d", idx, len(p.sk.Events), in.pos)
				break
			}
			in.varint(false)
			in.float()
			if in.bool() {
				in.str()
			}
			if p.f.audio && p.sk.Events[idx].HasAudio {
				in.float()
				in.float()
			}
		}
		b.add("event", -1, "", times)
	}
}
