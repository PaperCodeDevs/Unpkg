package zdeflate

func (s *dstate) longestMatch(curMatch uint32) uint32 {
	chainLength := s.maxChain
	bestLen := int(s.prevLength)
	niceMatch := s.niceMatch
	var limit uint32
	if s.strstart > s.maxDist() {
		limit = s.strstart - s.maxDist()
	}
	wmask := s.wMask
	scan := s.strstart
	scanEnd1 := s.window[scan+uint32(bestLen)-1]
	scanEnd := s.window[scan+uint32(bestLen)]
	if s.prevLength >= s.goodMatch {
		chainLength >>= 2
	}
	if uint32(niceMatch) > s.lookahead {
		niceMatch = int(s.lookahead)
	}
	strend := scan + maxMatch
	for {
		m := curMatch
		if s.window[m+uint32(bestLen)] == scanEnd &&
			s.window[m+uint32(bestLen)-1] == scanEnd1 &&
			s.window[m] == s.window[scan] &&
			s.window[m+1] == s.window[scan+1] {
			si := scan + 2
			mi := m + 2
			for {
				ok := true
				for k := 0; k < 8; k++ {
					si++
					mi++
					if s.window[si] != s.window[mi] {
						ok = false
						break
					}
				}
				if !ok || si >= strend {
					break
				}
			}
			lenN := maxMatch - int(strend-si)
			if lenN > bestLen {
				s.matchStart = curMatch
				bestLen = lenN
				if lenN >= niceMatch {
					break
				}
				scanEnd1 = s.window[scan+uint32(bestLen)-1]
				scanEnd = s.window[scan+uint32(bestLen)]
			}
		}
		curMatch = uint32(s.prev[curMatch&wmask])
		if curMatch <= limit {
			break
		}
		chainLength--
		if chainLength == 0 {
			break
		}
	}
	if uint32(bestLen) <= s.lookahead {
		return uint32(bestLen)
	}
	return s.lookahead
}
