package spine

var boneKinds4 = []string{"rotate", "translate", "translatex", "translatey", "scale", "scalex", "scaley", "shear", "shearx", "sheary"}

var slotKinds4 = []string{"attachment", "rgba", "rgb", "rgba2", "rgb2", "alpha"}

var slotBytes4 = []int{0, 4, 3, 7, 6, 1}

func (p *parser) curve4(props int) {
	if p.in.byte() == curveBezier {
		p.in.floats(4 * props)
	}
}

func (p *parser) frames4(frames, props int, read func()) []float32 {
	in := p.in
	times := make([]float32, frames)
	for f := 0; f < frames && in.err == nil; f++ {
		times[f] = in.float()
		read()
		if f > 0 {
			p.curve4(props)
		}
	}
	return times
}

func (p *parser) animation4(name string) Animation {
	in := p.in
	b := &animBuilder{a: Animation{Name: name}}
	in.count()
	p.slotTimelines4(b)
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		bone := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := int(in.byte())
			frames := in.count()
			in.varint(true)
			if typ >= len(boneKinds4) {
				in.fail("bone timeline 类型 %d", typ)
				break
			}
			values := 1
			if typ == 1 || typ == 4 || typ == 7 {
				values = 2
			}
			b.add(boneKinds4[typ], bone, "", p.frames4(frames, values, func() { in.floats(values) }))
		}
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		frames := in.count()
		in.varint(true)
		times := make([]float32, frames)
		for f := 0; f < frames && in.err == nil; f++ {
			times[f] = in.float()
			in.floats(2)
			if f > 0 {
				p.curve4(2)
			}
			in.sbyte()
			in.bool()
			in.bool()
		}
		b.add("ik", idx, "", times)
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		frames := in.count()
		in.varint(true)
		b.add("transform", idx, "", p.frames4(frames, 6, func() { in.floats(6) }))
	}
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		idx := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := int(in.byte())
			frames := in.count()
			in.varint(true)
			switch typ {
			case 0, 1:
				b.add(pathKinds[typ], idx, "", p.frames4(frames, 1, func() { in.float() }))
			case 2:
				b.add(pathKinds[typ], idx, "", p.frames4(frames, 3, func() { in.floats(3) }))
			default:
				in.fail("path timeline 类型 %d", typ)
			}
		}
	}
	p.deformTimelines4(b)
	p.orderEventTimelines(b)
	return b.a
}

func (p *parser) slotTimelines4(b *animBuilder) {
	in := p.in
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		slot := in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			typ := int(in.byte())
			frames := in.count()
			if typ >= len(slotKinds4) {
				in.fail("slot timeline 类型 %d", typ)
				break
			}
			if typ == 0 {
				times := make([]float32, frames)
				for f := 0; f < frames && in.err == nil; f++ {
					times[f] = in.float()
					p.readName()
				}
				b.add("attachment", slot, "", times)
				continue
			}
			in.varint(true)
			nb := slotBytes4[typ]
			b.add(slotKinds4[typ], slot, "", p.frames4(frames, nb, func() {
				if in.need(nb) {
					in.pos += nb
				}
			}))
		}
	}
}

func (p *parser) deformTimelines4(b *animBuilder) {
	in := p.in
	for i, n := 0, in.count(); i < n && in.err == nil; i++ {
		in.varint(true)
		for j, nn := 0, in.count(); j < nn && in.err == nil; j++ {
			slot := in.varint(true)
			for k, nnn := 0, in.count(); k < nnn && in.err == nil; k++ {
				att := p.readName()
				typ := 0
				if p.f.sequence {
					typ = int(in.byte())
				}
				frames := in.count()
				switch typ {
				case 0:
					in.varint(true)
					b.add("deform", slot, att, p.deformFrames4(frames))
				case 1:
					times := make([]float32, frames)
					for f := 0; f < frames && in.err == nil; f++ {
						times[f] = in.float()
						in.int32()
						in.float()
					}
					b.add("sequence", slot, att, times)
				default:
					in.fail("attachment timeline 类型 %d", typ)
				}
			}
		}
	}
}

func (p *parser) deformFrames4(frames int) []float32 {
	in := p.in
	times := make([]float32, frames)
	for f := 0; f < frames && in.err == nil; f++ {
		times[f] = in.float()
		if f > 0 {
			p.curve4(1)
		}
		if end := in.count(); end != 0 {
			in.varint(true)
			in.floats(end)
		}
	}
	return times
}
