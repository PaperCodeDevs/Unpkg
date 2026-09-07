package zdeflate

func (s *dstate) scanTree(tree []ct, maxCode int) {
	prevlen := -1
	nextlen := int(tree[0].len)
	count := 0
	maxCount := 7
	minCount := 4
	if nextlen == 0 {
		maxCount = 138
		minCount = 3
	}
	tree[maxCode+1].len = 0xffff
	for n := 0; n <= maxCode; n++ {
		curlen := nextlen
		nextlen = int(tree[n+1].len)
		count++
		if count < maxCount && curlen == nextlen {
			continue
		}
		if count < minCount {
			s.blTree[curlen].freq += uint16(count)
		} else if curlen != 0 {
			if curlen != prevlen {
				s.blTree[curlen].freq++
			}
			s.blTree[rep36].freq++
		} else if count <= 10 {
			s.blTree[repz310].freq++
		} else {
			s.blTree[repz11138].freq++
		}
		count = 0
		prevlen = curlen
		if nextlen == 0 {
			maxCount = 138
			minCount = 3
		} else if curlen == nextlen {
			maxCount = 6
			minCount = 3
		} else {
			maxCount = 7
			minCount = 4
		}
	}
}

func (s *dstate) sendTree(tree []ct, maxCode int) {
	prevlen := -1
	nextlen := int(tree[0].len)
	count := 0
	maxCount := 7
	minCount := 4
	if nextlen == 0 {
		maxCount = 138
		minCount = 3
	}
	for n := 0; n <= maxCode; n++ {
		curlen := nextlen
		nextlen = int(tree[n+1].len)
		count++
		if count < maxCount && curlen == nextlen {
			continue
		}
		if count < minCount {
			for {
				s.sendCode(curlen, s.blTree[:])
				count--
				if count == 0 {
					break
				}
			}
		} else if curlen != 0 {
			if curlen != prevlen {
				s.sendCode(curlen, s.blTree[:])
				count--
			}
			s.sendCode(rep36, s.blTree[:])
			s.sendBits(count-3, 2)
		} else if count <= 10 {
			s.sendCode(repz310, s.blTree[:])
			s.sendBits(count-3, 3)
		} else {
			s.sendCode(repz11138, s.blTree[:])
			s.sendBits(count-11, 7)
		}
		count = 0
		prevlen = curlen
		if nextlen == 0 {
			maxCount = 138
			minCount = 3
		} else if curlen == nextlen {
			maxCount = 6
			minCount = 3
		} else {
			maxCount = 7
			minCount = 4
		}
	}
}

func (s *dstate) buildBlTree() int {
	s.scanTree(s.dynL[:], s.lMaxCode)
	s.scanTree(s.dynD[:], s.dMaxCode)
	s.buildTree(s.blTree[:], extraBlbits[:], 0, blCodes, maxBlBits, nil)
	maxBlindex := blCodes - 1
	for maxBlindex >= 3 {
		if s.blTree[blOrder[maxBlindex]].len != 0 {
			break
		}
		maxBlindex--
	}
	s.optLen += 3*uint32(maxBlindex+1) + 5 + 5 + 4
	return maxBlindex
}

func (s *dstate) sendAllTrees(lcodes, dcodes, blcodes int) {
	s.sendBits(lcodes-257, 5)
	s.sendBits(dcodes-1, 5)
	s.sendBits(blcodes-4, 4)
	for rank := 0; rank < blcodes; rank++ {
		s.sendBits(int(s.blTree[blOrder[rank]].len), 3)
	}
	s.sendTree(s.dynL[:], lcodes-1)
	s.sendTree(s.dynD[:], dcodes-1)
}
