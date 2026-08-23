package crn

func DebugPalettes(data []byte) ([]uint32, []uint32, []uint16, []uint16, error) {
	u, err := newUnpacker(data)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return u.colorEndpoints, u.colorSelectors, u.alphaEndpoints, u.alphaSelectors, nil
}

func DebugBlocks(data []byte, level uint32) ([]byte, int, int, error) {
	u, err := newUnpacker(data)
	if err != nil {
		return nil, 0, 0, err
	}
	b, w, h, err := u.unpackLevel(level)
	return b, int(w), int(h), err
}
