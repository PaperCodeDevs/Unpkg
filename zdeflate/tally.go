package zdeflate

func (s *dstate) initBlock() {
	for n := 0; n < lCodes; n++ {
		s.dynL[n].freq = 0
	}
	for n := 0; n < dCodes; n++ {
		s.dynD[n].freq = 0
	}
	for n := 0; n < blCodes; n++ {
		s.blTree[n].freq = 0
	}
	s.dynL[endBlock].freq = 1
	s.optLen = 0
	s.staticLen = 0
	s.lastLit = 0
	s.matches = 0
}

func (s *dstate) tallyLit(c byte) bool {
	s.dBuf[s.lastLit] = 0
	s.lBuf[s.lastLit] = c
	s.lastLit++
	s.dynL[c].freq++
	return s.lastLit == s.litBufsize-1
}

func (s *dstate) tallyDist(distance, length uint32) bool {
	s.dBuf[s.lastLit] = uint16(distance)
	s.lBuf[s.lastLit] = byte(length)
	s.lastLit++
	distance--
	s.dynL[uint16(lengthCode[length])+literals+1].freq++
	s.dynD[distCode(distance)].freq++
	return s.lastLit == s.litBufsize-1
}

func (s *dstate) flushBlock(last bool) {
	var buf []byte
	var storedLen uint32
	if s.blockStart >= 0 {
		off := uint32(s.blockStart)
		storedLen = s.strstart - off
		buf = s.window[off : off+storedLen]
	} else {
		storedLen = uint32(int64(s.strstart) - s.blockStart)
	}
	s.trFlushBlock(buf, storedLen, last)
	s.blockStart = int64(s.strstart)
	s.flushPending()
}
