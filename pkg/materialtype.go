package pkg

func (h MaterialHdr) TypeTag() byte {
	return byte(h.Type)
}

func (h MaterialHdr) TypeClass() byte {
	return byte(h.Type >> 8)
}

func (h MaterialHdr) ExtraTag() byte {
	return byte(h.Extra)
}

func (h MaterialHdr) ExtraClass() byte {
	return byte(h.Extra >> 8)
}

func MaterialTypeKind(t uint16) string {
	tag := byte(t)
	if tag < 0xf0 {
		return ""
	}
	if byte(t>>8) == 0x4d {
		return "shader"
	}
	return "shader-obj"
}

func MaterialSlotKind(slot, stage uint8) string {
	if slot == 0 {
		return "depth"
	}
	if stage == 3 || stage == 4 {
		return "tess"
	}
	return "color"
}
