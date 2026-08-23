package crn

import "encoding/binary"

type unpacker struct {
	data  []byte
	h     *Header
	codec codec

	referenceDM     dataModel
	endpointDeltaDM [2]dataModel
	selectorDeltaDM [2]dataModel

	colorEndpoints []uint32
	colorSelectors []uint32
	alphaEndpoints []uint16
	alphaSelectors []uint16
}

type blockBuf struct {
	ref    uint8
	color  uint16
	alpha0 uint16
}

func (u *unpacker) initTables() error {
	if u.h.TablesOfs+u.h.TablesSize > uint32(len(u.data)) {
		return errStream
	}
	if !u.codec.start(u.data[u.h.TablesOfs : u.h.TablesOfs+u.h.TablesSize]) {
		return errStream
	}
	if err := u.codec.receiveModel(&u.referenceDM); err != nil {
		return err
	}
	if u.h.ColorEndpoints.Num == 0 && u.h.AlphaEndpoints.Num == 0 {
		return errStream
	}
	if u.h.ColorEndpoints.Num != 0 {
		if err := u.codec.receiveModel(&u.endpointDeltaDM[0]); err != nil {
			return err
		}
		if err := u.codec.receiveModel(&u.selectorDeltaDM[0]); err != nil {
			return err
		}
	}
	if u.h.AlphaEndpoints.Num != 0 {
		if err := u.codec.receiveModel(&u.endpointDeltaDM[1]); err != nil {
			return err
		}
		if err := u.codec.receiveModel(&u.selectorDeltaDM[1]); err != nil {
			return err
		}
	}
	return nil
}

func (u *unpacker) decodePalettes() error {
	if u.h.ColorEndpoints.Num != 0 {
		if err := u.decodeColorEndpoints(); err != nil {
			return err
		}
		if err := u.decodeColorSelectors(); err != nil {
			return err
		}
	}
	if u.h.AlphaEndpoints.Num != 0 {
		if err := u.decodeAlphaEndpoints(); err != nil {
			return err
		}
		if err := u.decodeAlphaSelectors(); err != nil {
			return err
		}
	}
	return nil
}

func (u *unpacker) unpackLevel(level uint32) ([]byte, uint32, uint32, error) {
	cur := u.h.LevelOfs[level]
	next := u.h.DataSize
	if level+1 < u.h.Levels {
		next = u.h.LevelOfs[level+1]
	}
	if cur >= next || next > uint32(len(u.data)) {
		return nil, 0, 0, errStream
	}
	w := u.h.Width >> level
	if w < 1 {
		w = 1
	}
	hh := u.h.Height >> level
	if hh < 1 {
		hh = 1
	}
	blocksX := (w + 3) >> 2
	blocksY := (hh + 3) >> 2
	pitch := u.h.blockSize() * blocksX
	dst := make([]byte, pitch*blocksY)
	if !u.codec.start(u.data[cur:next]) {
		return nil, 0, 0, errStream
	}
	switch u.h.Format {
	case FmtDXT1:
		u.unpackDXT1(dst, pitch, blocksX, blocksY)
	case FmtDXT5, 3, 4, 5, 6:
		u.unpackDXT5(dst, pitch, blocksX, blocksY)
	default:
		return nil, 0, 0, errStream
	}
	return dst, w, hh, nil
}

func put32(dst []byte, off int, v uint32) {
	if off < 0 || off+4 > len(dst) {
		return
	}
	binary.LittleEndian.PutUint32(dst[off:], v)
}

func wrapIdx(x, n uint32) uint32 {
	if n == 0 {
		return 0
	}
	for x >= n {
		x -= n
	}
	return x
}

func (u *unpacker) unpackDXT5(dst []byte, pitch, blocksX, blocksY uint32) {
	numC := uint32(len(u.colorEndpoints))
	numA := uint32(len(u.alphaEndpoints))
	numCS := uint32(len(u.colorSelectors))
	numAS := uint32(len(u.alphaSelectors) / 3)
	width := (blocksX + 1) &^ 1
	height := (blocksY + 1) &^ 1
	buf := make([]blockBuf, width)
	var colorEP, alphaEP uint32
	var refGroup uint8
	off := 0
	for y := uint32(0); y < height; y++ {
		row := off
		for x := uint32(0); x < width; x++ {
			vis := y < blocksY && x < blocksX
			if y&1 == 0 && x&1 == 0 {
				refGroup = uint8(u.codec.decode(&u.referenceDM))
			}
			b := &buf[x]
			var eref uint8
			if y&1 != 0 {
				eref = b.ref
			} else {
				eref = refGroup & 3
				refGroup >>= 2
				b.ref = refGroup & 3
				refGroup >>= 2
			}
			if eref == 0 {
				colorEP = wrapIdx(colorEP+u.codec.decode(&u.endpointDeltaDM[0]), numC)
				b.color = uint16(colorEP)
				alphaEP = wrapIdx(alphaEP+u.codec.decode(&u.endpointDeltaDM[1]), numA)
				b.alpha0 = uint16(alphaEP)
			} else if eref == 1 {
				b.color = uint16(colorEP)
				b.alpha0 = uint16(alphaEP)
			} else {
				colorEP = uint32(b.color)
				alphaEP = uint32(b.alpha0)
			}
			cs := wrapIdx(u.codec.decode(&u.selectorDeltaDM[0]), numCS)
			as := wrapIdx(u.codec.decode(&u.selectorDeltaDM[1]), numAS)
			if vis {
				pd := row + int(x)*16
				ap := u.alphaSelectors[as*3:]
				put32(dst, pd, uint32(u.alphaEndpoints[alphaEP])|uint32(ap[0])<<16)
				put32(dst, pd+4, uint32(ap[1])|uint32(ap[2])<<16)
				put32(dst, pd+8, u.colorEndpoints[colorEP])
				put32(dst, pd+12, u.colorSelectors[cs])
			}
		}
		off += int(pitch)
	}
}

func (u *unpacker) unpackDXT1(dst []byte, pitch, blocksX, blocksY uint32) {
	numC := uint32(len(u.colorEndpoints))
	numCS := uint32(len(u.colorSelectors))
	width := (blocksX + 1) &^ 1
	height := (blocksY + 1) &^ 1
	buf := make([]blockBuf, width)
	var colorEP uint32
	var refGroup uint8
	off := 0
	for y := uint32(0); y < height; y++ {
		row := off
		for x := uint32(0); x < width; x++ {
			vis := y < blocksY && x < blocksX
			if y&1 == 0 && x&1 == 0 {
				refGroup = uint8(u.codec.decode(&u.referenceDM))
			}
			b := &buf[x]
			var eref uint8
			if y&1 != 0 {
				eref = b.ref
			} else {
				eref = refGroup & 3
				refGroup >>= 2
				b.ref = refGroup & 3
				refGroup >>= 2
			}
			if eref == 0 {
				colorEP = wrapIdx(colorEP+u.codec.decode(&u.endpointDeltaDM[0]), numC)
				b.color = uint16(colorEP)
			} else if eref == 1 {
				b.color = uint16(colorEP)
			} else {
				colorEP = uint32(b.color)
			}
			cs := wrapIdx(u.codec.decode(&u.selectorDeltaDM[0]), numCS)
			if vis {
				pd := row + int(x)*8
				put32(dst, pd, u.colorEndpoints[colorEP])
				put32(dst, pd+4, u.colorSelectors[cs])
			}
		}
		off += int(pitch)
	}
}
