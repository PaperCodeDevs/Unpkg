package zdeflate

const (
	minMatch     = 3
	maxMatch     = 258
	minLookahead = maxMatch + minMatch + 1
	tooFar       = 4096
	nilPos       = 0
	winInit      = maxMatch

	lengthCodes = 29
	literals    = 256
	lCodes      = literals + 1 + lengthCodes
	dCodes      = 30
	blCodes     = 19
	heapSize    = 2*lCodes + 1
	maxBits     = 15
	maxBlBits   = 7
	bufSize     = 16

	endBlock  = 256
	rep36     = 16
	repz310   = 17
	repz11138 = 18

	storedBlock = 0
	staticTrees = 1
	dynTrees    = 2

	zFinish = 4
	zFixed  = 4

	smallest = 1
)

type Config struct {
	Level      int
	WindowBits int
	MemLevel   int
	Strategy   int
}

func Compress(src []byte) []byte {
	return CompressCfg(src, Config{Level: 9, WindowBits: 15, MemLevel: 8, Strategy: 0})
}

func CompressCfg(src []byte, cfg Config) []byte {
	if cfg.WindowBits == 0 {
		cfg.WindowBits = 15
	}
	if cfg.MemLevel == 0 {
		cfg.MemLevel = 8
	}
	if cfg.Level < 0 || cfg.Level > 9 {
		cfg.Level = 9
	}
	s := newState(cfg)
	s.in = src
	s.deflateSlow(zFinish)
	return s.out
}
