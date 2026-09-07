package zdeflate

type dstate struct {
	window                        []byte
	prev, head                    []uint16
	wSize, wMask, hashSize        uint32
	hashMask, hashShift, insH     uint32
	strstart, lookahead           uint32
	matchStart, prevMatch         uint32
	matchLength, prevLength       uint32
	matchAvailable                int
	blockStart                    int64
	insert, windowSize, highWater uint32
	maxChain, maxLazy, goodMatch  uint32
	niceMatch                     int
	level, strategy               int
	litBufsize, lastLit           uint32
	lBuf                          []byte
	dBuf                          []uint16
	dynL                          [heapSize]ct
	dynD                          [2*dCodes + 1]ct
	blTree                        [2*blCodes + 1]ct
	blCount                       [maxBits + 1]uint16
	heap                          [2*lCodes + 1]int
	heapLen, heapMax              int
	depth                         [2*lCodes + 1]byte
	optLen, staticLen, matches    uint32
	lMaxCode, dMaxCode            int
	biBuf                         uint16
	biValid                       int
	pendingBuf                    []byte
	pendingN                      int
	out                           []byte
	in                            []byte
	inOff                         int
}

func newState(cfg Config) *dstate {
	wb := cfg.WindowBits
	if wb < 8 {
		wb = 8
	}
	if wb > 15 {
		wb = 15
	}
	if wb == 8 {
		wb = 9
	}
	ml := cfg.MemLevel
	if ml < 1 {
		ml = 1
	}
	if ml > 9 {
		ml = 9
	}
	s := &dstate{level: cfg.Level, strategy: cfg.Strategy}
	s.wSize = 1 << uint(wb)
	s.wMask = s.wSize - 1
	s.hashShift = uint32((ml + 7 + minMatch - 1) / minMatch)
	s.hashSize = 1 << uint(ml+7)
	s.hashMask = s.hashSize - 1
	s.windowSize = 2 * s.wSize
	s.window = make([]byte, s.windowSize+uint32(maxMatch)+8)
	s.prev = make([]uint16, s.wSize)
	s.head = make([]uint16, s.hashSize)
	s.litBufsize = 1 << uint(ml+6)
	s.pendingBuf = make([]byte, s.litBufsize*(2+2))
	s.lBuf = make([]byte, s.litBufsize)
	s.dBuf = make([]uint16, s.litBufsize)
	lc := levelCfg[s.level]
	s.maxLazy = lc.lazy
	s.goodMatch = lc.good
	s.niceMatch = int(lc.nice)
	s.maxChain = lc.chain
	s.matchLength = minMatch - 1
	s.prevLength = minMatch - 1
	s.initBlock()
	return s
}

func (s *dstate) maxDist() uint32 {
	return s.wSize - minLookahead
}

func (s *dstate) clearHash() {
	s.head[s.hashSize-1] = nilPos
	for i := range s.head[:s.hashSize-1] {
		s.head[i] = 0
	}
}

func (s *dstate) insertString(str uint32) uint32 {
	s.insH = ((s.insH << s.hashShift) ^ uint32(s.window[str+minMatch-1])) & s.hashMask
	h := s.head[s.insH]
	s.prev[str&s.wMask] = h
	s.head[s.insH] = uint16(str)
	return uint32(h)
}
