package spine

func (p *parser) bones() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		b := Bone{Parent: -1}
		b.Name = in.str()
		if i > 0 {
			b.Parent = in.varint(true)
		}
		b.Rotation = in.float()
		b.X = in.float()
		b.Y = in.float()
		b.ScaleX = in.float()
		b.ScaleY = in.float()
		b.ShearX = in.float()
		b.ShearY = in.float()
		b.Length = in.float()
		b.TransformMode = in.varint(true)
		if p.f.skinRequired {
			b.SkinRequired = in.bool()
		}
		if p.nonessential {
			b.Color = in.uint32()
		}
		p.sk.Bones = append(p.sk.Bones, b)
	}
}

func (p *parser) slots() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		s := Slot{}
		s.Name = in.str()
		s.Bone = in.varint(true)
		s.Color = in.uint32()
		if dark := in.int32(); dark != -1 {
			s.HasDark = true
			s.DarkColor = uint32(dark)
		}
		s.Attachment = p.readName()
		s.BlendMode = in.varint(true)
		p.sk.Slots = append(p.sk.Slots, s)
	}
}

func (p *parser) ikConstraints() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		c := IKConstraint{}
		c.Name = in.str()
		c.Order = in.varint(true)
		if p.f.skinRequired {
			c.SkinRequired = in.bool()
		}
		c.Bones = in.indices()
		c.Target = in.varint(true)
		c.Mix = in.float()
		if p.f.softness {
			c.Softness = in.float()
		}
		c.BendDirection = in.sbyte()
		if p.f.ikExtra {
			c.Compress = in.bool()
			c.Stretch = in.bool()
			c.Uniform = in.bool()
		}
		p.sk.IK = append(p.sk.IK, c)
	}
}

func (p *parser) transformConstraints() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		c := TransformConstraint{}
		c.Name = in.str()
		c.Order = in.varint(true)
		if p.f.skinRequired {
			c.SkinRequired = in.bool()
		}
		c.Bones = in.indices()
		c.Target = in.varint(true)
		c.Local = in.bool()
		c.Relative = in.bool()
		c.OffsetRotation = in.float()
		c.OffsetX = in.float()
		c.OffsetY = in.float()
		c.OffsetScaleX = in.float()
		c.OffsetScaleY = in.float()
		c.OffsetShearY = in.float()
		if p.f.v4 {
			c.MixRotate = in.float()
			c.MixX = in.float()
			c.MixY = in.float()
			c.MixScaleX = in.float()
			c.MixScaleY = in.float()
			c.MixShearY = in.float()
		} else {
			c.MixRotate = in.float()
			c.MixX = in.float()
			c.MixY = c.MixX
			c.MixScaleX = in.float()
			c.MixScaleY = c.MixScaleX
			c.MixShearY = in.float()
		}
		p.sk.Transform = append(p.sk.Transform, c)
	}
}

func (p *parser) pathConstraints() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		c := PathConstraint{}
		c.Name = in.str()
		c.Order = in.varint(true)
		if p.f.skinRequired {
			c.SkinRequired = in.bool()
		}
		c.Bones = in.indices()
		c.Target = in.varint(true)
		c.PositionMode = in.varint(true)
		c.SpacingMode = in.varint(true)
		c.RotateMode = in.varint(true)
		c.OffsetRotation = in.float()
		c.Position = in.float()
		c.Spacing = in.float()
		c.MixRotate = in.float()
		c.MixX = in.float()
		if p.f.v4 {
			c.MixY = in.float()
		} else {
			c.MixY = c.MixX
		}
		p.sk.Path = append(p.sk.Path, c)
	}
}

func (p *parser) events() {
	in := p.in
	n := in.count()
	for i := 0; i < n && in.err == nil; i++ {
		e := Event{}
		e.Name = p.readName()
		e.Int = in.varint(false)
		e.Float = in.float()
		e.String = in.str()
		if p.f.audio {
			var null bool
			e.AudioPath, null = in.strNull()
			e.HasAudio = !null
			if e.HasAudio {
				e.Volume = in.float()
				e.Balance = in.float()
			}
		}
		p.sk.Events = append(p.sk.Events, e)
	}
}
