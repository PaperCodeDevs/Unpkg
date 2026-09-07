package zdeflate

func (s *dstate) deflateSlow(flush int) {
	for {
		if s.lookahead < minLookahead {
			s.fillWindow()
			if s.lookahead < minLookahead && flush != zFinish {
				return
			}
			if s.lookahead == 0 {
				break
			}
		}
		hashHead := uint32(nilPos)
		if s.lookahead >= minMatch {
			hashHead = s.insertString(s.strstart)
		}
		s.prevLength = s.matchLength
		s.prevMatch = s.matchStart
		s.matchLength = minMatch - 1
		if hashHead != nilPos && s.prevLength < s.maxLazy && s.strstart-hashHead <= s.maxDist() {
			s.matchLength = s.longestMatch(hashHead)
			if s.matchLength <= 5 && (s.strategy == 1 ||
				(s.matchLength == minMatch && s.strstart-s.matchStart > tooFar)) {
				s.matchLength = minMatch - 1
			}
		}
		if s.prevLength >= minMatch && s.matchLength <= s.prevLength {
			maxInsert := s.strstart + s.lookahead - minMatch
			bflush := s.tallyDist(s.strstart-1-s.prevMatch, s.prevLength-minMatch)
			s.lookahead -= s.prevLength - 1
			s.prevLength -= 2
			for {
				s.strstart++
				if s.strstart <= maxInsert {
					hashHead = s.insertString(s.strstart)
				}
				s.prevLength--
				if s.prevLength == 0 {
					break
				}
			}
			s.matchAvailable = 0
			s.matchLength = minMatch - 1
			s.strstart++
			if bflush {
				s.flushBlock(false)
			}
		} else if s.matchAvailable != 0 {
			bflush := s.tallyLit(s.window[s.strstart-1])
			if bflush {
				s.flushBlock(false)
			}
			s.strstart++
			s.lookahead--
		} else {
			s.matchAvailable = 1
			s.strstart++
			s.lookahead--
		}
	}
	if s.matchAvailable != 0 {
		s.tallyLit(s.window[s.strstart-1])
		s.matchAvailable = 0
	}
	if s.strstart < minMatch-1 {
		s.insert = s.strstart
	} else {
		s.insert = minMatch - 1
	}
	if flush == zFinish {
		s.flushBlock(true)
	} else if s.lastLit != 0 {
		s.flushBlock(false)
	}
}
