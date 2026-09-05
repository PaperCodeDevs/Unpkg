package spine

func (p *parser) skins() {
	in := p.in
	if def, ok := p.skin(true); ok {
		p.sk.Skins = append(p.sk.Skins, def)
	}
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		s, _ := p.skin(false)
		p.sk.Skins = append(p.sk.Skins, s)
	}
}

func (p *parser) skin(def bool) (Skin, bool) {
	in := p.in
	var s Skin
	var slotCount int
	if def {
		slotCount = in.count()
		if slotCount == 0 {
			return s, false
		}
		s.Name = "default"
	} else {
		if p.f.strTable {
			s.Name = in.strRef()
			s.Bones = in.indices()
			s.IK = in.indices()
			s.Transform = in.indices()
			s.Path = in.indices()
		} else {
			s.Name = in.str()
		}
		slotCount = in.count()
	}
	for i := 0; i < slotCount && in.err == nil; i++ {
		slot := in.varint(true)
		n := in.count()
		for j := 0; j < n && in.err == nil; j++ {
			name := p.readName()
			s.Attachments = append(s.Attachments, p.attachment(slot, name))
		}
	}
	return s, true
}

func (p *parser) attachment(slot int, attName string) Attachment {
	in := p.in
	a := Attachment{Slot: slot, Name: attName}
	if name, null := p.readNameNull(); !null {
		a.Name = name
	}
	a.Type = AttachmentType(in.byte())
	switch a.Type {
	case AttRegion:
		a.Path = p.readName()
		a.Rotation = in.float()
		a.X = in.float()
		a.Y = in.float()
		a.ScaleX = in.float()
		a.ScaleY = in.float()
		a.Width = in.float()
		a.Height = in.float()
		a.Color = in.uint32()
		if p.f.sequence {
			a.Sequence = p.sequence()
		}
	case AttBoundingBox:
		a.Vertices = p.vertices(in.count())
		if p.nonessential {
			a.Color = in.uint32()
		}
	case AttMesh:
		a.Path = p.readName()
		a.Color = in.uint32()
		count := in.count()
		a.UVs = in.floats(count << 1)
		a.Triangles = in.shorts()
		a.Vertices = p.vertices(count)
		a.HullLength = in.varint(true)
		if p.f.sequence {
			a.Sequence = p.sequence()
		}
		if p.nonessential {
			a.Edges = in.shorts()
			a.Width = in.float()
			a.Height = in.float()
		}
	case AttLinkedMesh:
		a.Path = p.readName()
		a.Color = in.uint32()
		a.SkinName = p.readName()
		a.Parent = p.readName()
		a.InheritDeform = in.bool()
		if p.f.sequence {
			a.Sequence = p.sequence()
		}
		if p.nonessential {
			a.Width = in.float()
			a.Height = in.float()
		}
	case AttPath:
		a.Closed = in.bool()
		a.ConstantSpeed = in.bool()
		count := in.count()
		a.Vertices = p.vertices(count)
		a.Lengths = in.floats(count / 3)
		if p.nonessential {
			a.Color = in.uint32()
		}
	case AttPoint:
		a.Rotation = in.float()
		a.X = in.float()
		a.Y = in.float()
		if p.nonessential {
			a.Color = in.uint32()
		}
	case AttClipping:
		a.EndSlot = in.varint(true)
		a.Vertices = p.vertices(in.count())
		if p.nonessential {
			a.Color = in.uint32()
		}
	default:
		in.fail("附件类型 %d pos=%d", a.Type, in.pos)
	}
	return a
}

func (p *parser) sequence() *Sequence {
	in := p.in
	if !in.bool() {
		return nil
	}
	s := &Sequence{}
	s.Count = in.varint(true)
	s.Start = in.varint(true)
	s.Digits = in.varint(true)
	s.SetupIndex = in.varint(true)
	return s
}

func (p *parser) vertices(count int) Vertices {
	in := p.in
	v := Vertices{Count: count}
	if !in.bool() {
		v.Values = in.floats(count << 1)
		return v
	}
	v.Weighted = true
	for i := 0; i < count && in.err == nil; i++ {
		bc := in.count()
		v.Bones = append(v.Bones, bc)
		for j := 0; j < bc && in.err == nil; j++ {
			v.Bones = append(v.Bones, in.varint(true))
			x := in.float()
			y := in.float()
			w := in.float()
			v.Values = append(v.Values, x, y, w)
		}
	}
	return v
}
