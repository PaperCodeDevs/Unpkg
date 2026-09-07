package zdeflate

func (s *dstate) storedBlock(buf []byte, storedLen uint32, last bool) {
	flag := storedBlock << 1
	if last {
		flag++
	}
	s.sendBits(flag, 3)
	s.biWindup()
	s.putShort(uint16(storedLen))
	s.putShort(uint16(^storedLen))
	if storedLen > 0 && buf != nil {
		for i := uint32(0); i < storedLen; i++ {
			s.putByte(buf[i])
		}
	}
}

func (s *dstate) compressBlock(ltree, dtree []ct) {
	if s.lastLit != 0 {
		lx := uint32(0)
		for lx < s.lastLit {
			dist := uint32(s.dBuf[lx])
			lc := int(s.lBuf[lx])
			lx++
			if dist == 0 {
				s.sendCode(lc, ltree)
			} else {
				code := int(lengthCode[lc])
				s.sendCode(code+literals+1, ltree)
				extra := extraLbits[code]
				if extra != 0 {
					lc -= baseLength[code]
					s.sendBits(lc, extra)
				}
				dist--
				code = distCode(dist)
				s.sendCode(code, dtree)
				extra = extraDbits[code]
				if extra != 0 {
					dist -= uint32(baseDist[code])
					s.sendBits(int(dist), extra)
				}
			}
		}
	}
	s.sendCode(endBlock, ltree)
}

func (s *dstate) trFlushBlock(buf []byte, storedLen uint32, last bool) {
	optLenb := storedLen + 5
	staticLenb := storedLen + 5
	maxBlindex := 0
	if s.level > 0 {
		s.lMaxCode = s.buildTree(s.dynL[:], extraLbits[:], literals+1, lCodes, maxBits, staticLtree[:])
		s.dMaxCode = s.buildTree(s.dynD[:], extraDbits[:], 0, dCodes, maxBits, staticDtree[:])
		maxBlindex = s.buildBlTree()
		optLenb = (s.optLen + 3 + 7) >> 3
		staticLenb = (s.staticLen + 3 + 7) >> 3
		if staticLenb <= optLenb {
			optLenb = staticLenb
		}
	}
	if storedLen+4 <= optLenb && buf != nil {
		s.storedBlock(buf, storedLen, last)
	} else if s.strategy == zFixed || staticLenb == optLenb {
		flag := staticTrees << 1
		if last {
			flag++
		}
		s.sendBits(flag, 3)
		s.compressBlock(staticLtree[:], staticDtree[:])
	} else {
		flag := dynTrees << 1
		if last {
			flag++
		}
		s.sendBits(flag, 3)
		s.sendAllTrees(s.lMaxCode+1, s.dMaxCode+1, maxBlindex+1)
		s.compressBlock(s.dynL[:], s.dynD[:])
	}
	s.initBlock()
	if last {
		s.biWindup()
	}
}
