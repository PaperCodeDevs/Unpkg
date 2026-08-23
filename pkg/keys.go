package pkg

type ZipKeys struct {
	K0, K1, K2 uint32
}

type RainbowKeys struct {
	Magic  string
	XORKey []byte
}

type BlockTexOpt struct {
	Prefix  string
	Suffix  string
	Zip     ZipKeys
	Rainbow RainbowKeys
}

type LuaDecrypt func(raw []byte) ([]byte, error)
