package zdeflate

func (s *dstate) putByte(c byte) {
	if s.pendingN >= len(s.pendingBuf) {
		n := make([]byte, len(s.pendingBuf)*2)
		copy(n, s.pendingBuf)
		s.pendingBuf = n
	}
	s.pendingBuf[s.pendingN] = c
	s.pendingN++
}

func (s *dstate) putShort(w uint16) {
	s.putByte(byte(w))
	s.putByte(byte(w >> 8))
}

func (s *dstate) sendBits(val, length int) {
	if s.biValid > bufSize-length {
		s.biBuf |= uint16(val << s.biValid)
		s.putShort(s.biBuf)
		s.biBuf = uint16(val) >> uint(bufSize-s.biValid)
		s.biValid += length - bufSize
	} else {
		s.biBuf |= uint16(val << s.biValid)
		s.biValid += length
	}
}

func (s *dstate) sendCode(c int, tree []ct) {
	s.sendBits(int(tree[c].code), int(tree[c].len))
}

func (s *dstate) biFlush() {
	if s.biValid == 16 {
		s.putShort(s.biBuf)
		s.biBuf = 0
		s.biValid = 0
	} else if s.biValid >= 8 {
		s.putByte(byte(s.biBuf))
		s.biBuf >>= 8
		s.biValid -= 8
	}
}

func (s *dstate) biWindup() {
	if s.biValid > 8 {
		s.putShort(s.biBuf)
	} else if s.biValid > 0 {
		s.putByte(byte(s.biBuf))
	}
	s.biBuf = 0
	s.biValid = 0
}

func (s *dstate) flushPending() {
	if s.pendingN > 0 {
		s.out = append(s.out, s.pendingBuf[:s.pendingN]...)
		s.pendingN = 0
	}
}
