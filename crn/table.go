package crn

const (
	maxExpectedCodeSize = 16
	maxSupportedSyms    = 8192
	maxTableBits        = 11
)

type decoderTables struct {
	numSyms            uint32
	totalUsedSyms      uint32
	tableBits          uint32
	tableShift         uint32
	tableMaxCode       uint32
	decodeStartCodeSiz uint32
	minCodeSize        uint32
	maxCodeSize        uint32
	maxCodes           [maxExpectedCodeSize + 1]uint32
	valPtrs            [maxExpectedCodeSize + 1]int32
	lookup             []uint32
	sortedSymbolOrder  []uint16
}

func floorLog2(v uint32) uint32 {
	var l uint32
	for v > 1 {
		v >>= 1
		l++
	}
	return l
}

func ceilLog2(v uint32) uint32 {
	l := floorLog2(v)
	if l != 32 && v > (1<<l) {
		l++
	}
	return l
}

func (t *decoderTables) init(numSyms uint32, codeSizes []byte, tableBits uint32) bool {
	if numSyms == 0 || tableBits > maxTableBits {
		return false
	}
	t.numSyms = numSyms

	var numCodes [maxExpectedCodeSize + 1]uint32
	for i := uint32(0); i < numSyms; i++ {
		if c := codeSizes[i]; c != 0 {
			numCodes[c]++
		}
	}

	var minCodes [maxExpectedCodeSize]uint32
	var sortedPositions [maxExpectedCodeSize + 1]uint32
	var curCode, totalUsed, maxSize uint32
	minSize := uint32(0xFFFFFFFF)

	for i := uint32(1); i <= maxExpectedCodeSize; i++ {
		n := numCodes[i]
		if n == 0 {
			t.maxCodes[i-1] = 0
		} else {
			if i < minSize {
				minSize = i
			}
			if i > maxSize {
				maxSize = i
			}
			if curCode+n > uint32(1)<<i {
				return false
			}
			minCodes[i-1] = curCode
			mc := curCode + n - 1
			t.maxCodes[i-1] = 1 + ((mc << (16 - i)) | ((1 << (16 - i)) - 1))
			t.valPtrs[i-1] = int32(totalUsed)
			sortedPositions[i] = totalUsed
			curCode += n
			totalUsed += n
		}
		curCode <<= 1
	}
	t.totalUsedSyms = totalUsed
	if totalUsed == 0 {
		return false
	}

	if uint32(len(t.sortedSymbolOrder)) < numSyms {
		t.sortedSymbolOrder = make([]uint16, numSyms)
	}
	t.minCodeSize = minSize
	t.maxCodeSize = maxSize

	for i := uint32(0); i < numSyms; i++ {
		c := codeSizes[i]
		if c == 0 {
			continue
		}
		pos := sortedPositions[c]
		sortedPositions[c]++
		t.sortedSymbolOrder[pos] = uint16(i)
	}

	if tableBits <= minSize {
		tableBits = 0
	}
	t.tableBits = tableBits

	if tableBits != 0 {
		size := uint32(1) << tableBits
		if uint32(len(t.lookup)) < size {
			t.lookup = make([]uint32, size)
		}
		for i := uint32(0); i < size; i++ {
			t.lookup[i] = 0xFFFFFFFF
		}
		for cs := uint32(1); cs <= tableBits; cs++ {
			if numCodes[cs] == 0 {
				continue
			}
			fillSize := tableBits - cs
			fillNum := uint32(1) << fillSize
			minCode := minCodes[cs-1]
			maxCode := t.unshiftedMaxCode(cs)
			valPtr := uint32(t.valPtrs[cs-1])
			for code := minCode; code <= maxCode; code++ {
				sym := t.sortedSymbolOrder[valPtr+code-minCode]
				for j := uint32(0); j < fillNum; j++ {
					t.lookup[j+(code<<fillSize)] = uint32(sym) | (cs << 16)
				}
			}
		}
	}

	for i := 0; i < maxExpectedCodeSize; i++ {
		t.valPtrs[i] -= int32(minCodes[i])
	}

	t.tableMaxCode = 0
	t.decodeStartCodeSiz = minSize
	if tableBits != 0 {
		i := tableBits
		for ; i >= 1; i-- {
			if numCodes[i] != 0 {
				t.tableMaxCode = t.maxCodes[i-1]
				break
			}
		}
		if i >= 1 {
			t.decodeStartCodeSiz = tableBits + 1
			for j := tableBits + 1; j <= maxSize; j++ {
				if numCodes[j] != 0 {
					t.decodeStartCodeSiz = j
					break
				}
			}
		}
	}

	t.maxCodes[maxExpectedCodeSize] = 0xFFFFFFFF
	t.valPtrs[maxExpectedCodeSize] = 0xFFFFF
	t.tableShift = 32 - t.tableBits
	return true
}

func (t *decoderTables) unshiftedMaxCode(l uint32) uint32 {
	k := t.maxCodes[l-1]
	if k == 0 {
		return 0xFFFFFFFF
	}
	return (k - 1) >> (16 - l)
}
