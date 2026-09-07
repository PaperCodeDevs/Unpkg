package zdeflate

func (s *dstate) readBuf(dst []byte) uint32 {
	n := uint32(len(dst))
	left := uint32(len(s.in) - s.inOff)
	if n > left {
		n = left
	}
	if n == 0 {
		return 0
	}
	copy(dst[:n], s.in[s.inOff:s.inOff+int(n)])
	s.inOff += int(n)
	return n
}

func (s *dstate) slideHash() {
	wsize := s.wSize
	n := s.hashSize
	p := s.head
	for n > 0 {
		n--
		m := uint32(p[n])
		if m >= wsize {
			p[n] = uint16(m - wsize)
		} else {
			p[n] = nilPos
		}
	}
	n = wsize
	p = s.prev
	for n > 0 {
		n--
		m := uint32(p[n])
		if m >= wsize {
			p[n] = uint16(m - wsize)
		} else {
			p[n] = nilPos
		}
	}
}

func (s *dstate) fillWindow() {
	wsize := s.wSize
	for {
		more := s.windowSize - s.lookahead - s.strstart
		if s.strstart >= wsize+s.maxDist() {
			copy(s.window[:wsize], s.window[wsize:wsize+wsize])
			s.matchStart -= wsize
			s.strstart -= wsize
			s.blockStart -= int64(wsize)
			s.slideHash()
			more += wsize
		}
		if s.inOff >= len(s.in) {
			break
		}
		n := s.readBuf(s.window[s.strstart+s.lookahead : s.strstart+s.lookahead+more])
		s.lookahead += n
		if s.lookahead+s.insert >= minMatch {
			str := s.strstart - s.insert
			s.insH = uint32(s.window[str])
			s.insH = ((s.insH << s.hashShift) ^ uint32(s.window[str+1])) & s.hashMask
			for s.insert > 0 {
				s.insH = ((s.insH << s.hashShift) ^ uint32(s.window[str+minMatch-1])) & s.hashMask
				s.prev[str&s.wMask] = s.head[s.insH]
				s.head[s.insH] = uint16(str)
				str++
				s.insert--
				if s.lookahead+s.insert < minMatch {
					break
				}
			}
		}
		if s.lookahead >= minLookahead || s.inOff >= len(s.in) {
			break
		}
	}
	if s.highWater < s.windowSize {
		curr := s.strstart + s.lookahead
		if s.highWater < curr {
			initN := s.windowSize - curr
			if initN > winInit {
				initN = winInit
			}
			for i := uint32(0); i < initN; i++ {
				s.window[curr+i] = 0
			}
			s.highWater = curr + initN
		} else if s.highWater < curr+winInit {
			initN := curr + winInit - s.highWater
			if initN > s.windowSize-s.highWater {
				initN = s.windowSize - s.highWater
			}
			for i := uint32(0); i < initN; i++ {
				s.window[s.highWater+i] = 0
			}
			s.highWater += initN
		}
	}
}
