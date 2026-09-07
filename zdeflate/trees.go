package zdeflate

func (s *dstate) smaller(tree []ct, n, m int) bool {
	return tree[n].freq < tree[m].freq ||
		(tree[n].freq == tree[m].freq && s.depth[n] <= s.depth[m])
}

func (s *dstate) pqdownheap(tree []ct, k int) {
	v := s.heap[k]
	j := k << 1
	for j <= s.heapLen {
		if j < s.heapLen && s.smaller(tree, s.heap[j+1], s.heap[j]) {
			j++
		}
		if s.smaller(tree, v, s.heap[j]) {
			break
		}
		s.heap[k] = s.heap[j]
		k = j
		j <<= 1
	}
	s.heap[k] = v
}

func (s *dstate) pqremove(tree []ct) int {
	top := s.heap[smallest]
	s.heap[smallest] = s.heap[s.heapLen]
	s.heapLen--
	s.pqdownheap(tree, smallest)
	return top
}

func genCodes(tree []ct, maxCode int, blCount [maxBits + 1]uint16) {
	var next [maxBits + 1]uint16
	code := uint16(0)
	for bits := 1; bits <= maxBits; bits++ {
		code = (code + blCount[bits-1]) << 1
		next[bits] = code
	}
	for n := 0; n <= maxCode; n++ {
		lenN := int(tree[n].len)
		if lenN == 0 {
			continue
		}
		tree[n].code = uint16(biReverse(uint(next[lenN]), lenN))
		next[lenN]++
	}
}

func (s *dstate) genBitlen(tree []ct, maxCode int, extra []int, extraBase, maxLength int, stree []ct) {
	overflow := 0
	for bits := 0; bits <= maxBits; bits++ {
		s.blCount[bits] = 0
	}
	tree[s.heap[s.heapMax]].len = 0
	for h := s.heapMax + 1; h < heapSize; h++ {
		n := s.heap[h]
		bits := int(tree[tree[n].dad].len) + 1
		if bits > maxLength {
			bits = maxLength
			overflow++
		}
		tree[n].len = uint16(bits)
		if n > maxCode {
			continue
		}
		s.blCount[bits]++
		xbits := 0
		if n >= extraBase && extra != nil {
			xbits = extra[n-extraBase]
		}
		f := uint32(tree[n].freq)
		s.optLen += f * uint32(bits+xbits)
		if stree != nil {
			s.staticLen += f * uint32(int(stree[n].len)+xbits)
		}
	}
	if overflow == 0 {
		return
	}
	doOverflow := overflow
	for doOverflow > 0 {
		bits := maxLength - 1
		for s.blCount[bits] == 0 {
			bits--
		}
		s.blCount[bits]--
		s.blCount[bits+1] += 2
		s.blCount[maxLength]--
		doOverflow -= 2
	}
	h := heapSize
	for bits := maxLength; bits != 0; bits-- {
		n := int(s.blCount[bits])
		for n != 0 {
			h--
			m := s.heap[h]
			if m > maxCode {
				continue
			}
			if int(tree[m].len) != bits {
				s.optLen += (uint32(bits) - uint32(tree[m].len)) * uint32(tree[m].freq)
				tree[m].len = uint16(bits)
			}
			n--
		}
	}
}

func (s *dstate) buildTree(tree []ct, extra []int, extraBase, elems, maxLength int, stree []ct) int {
	maxCode := -1
	s.heapLen = 0
	s.heapMax = heapSize
	for n := 0; n < elems; n++ {
		if tree[n].freq != 0 {
			s.heapLen++
			s.heap[s.heapLen] = n
			maxCode = n
			s.depth[n] = 0
		} else {
			tree[n].len = 0
		}
	}
	for s.heapLen < 2 {
		var node int
		if maxCode < 2 {
			maxCode++
			node = maxCode
		} else {
			node = 0
		}
		s.heapLen++
		s.heap[s.heapLen] = node
		tree[node].freq = 1
		s.depth[node] = 0
		s.optLen--
		if stree != nil {
			s.staticLen -= uint32(stree[node].len)
		}
	}
	for n := s.heapLen / 2; n >= 1; n-- {
		s.pqdownheap(tree, n)
	}
	node := elems
	for {
		n := s.pqremove(tree)
		m := s.heap[smallest]
		s.heapMax--
		s.heap[s.heapMax] = n
		s.heapMax--
		s.heap[s.heapMax] = m
		tree[node].freq = tree[n].freq + tree[m].freq
		dn, dm := s.depth[n], s.depth[m]
		if dn >= dm {
			s.depth[node] = dn + 1
		} else {
			s.depth[node] = dm + 1
		}
		tree[n].dad = uint16(node)
		tree[m].dad = uint16(node)
		s.heap[smallest] = node
		node++
		s.pqdownheap(tree, smallest)
		if s.heapLen < 2 {
			break
		}
	}
	s.heapMax--
	s.heap[s.heapMax] = s.heap[smallest]
	s.genBitlen(tree, maxCode, extra, extraBase, maxLength, stree)
	genCodes(tree, maxCode, s.blCount)
	return maxCode
}
