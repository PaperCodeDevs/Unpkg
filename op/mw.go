package op

func ToStd(code byte) byte {
	switch {
	case code <= 0x0F:
		return code ^ 1
	case code >= 0x12 && code <= 0x17:
		return code - 0x12 + 0x27
	case code >= 0x18 && code <= 0x2C:
		return code - 0x18 + 0x12
	case code >= 0x2D && code <= 0x39:
		return code - 0x2D + 0x34
	case code >= 0x3A && code <= 0x40:
		return code - 0x3A + 0x2D
	case code >= 0x41 && code <= 0x5F:
		return code
	default:
		return code
	}
}

func Norm(version, code byte) byte {
	if version == 0x90 {
		return ToStd(code)
	}
	return code
}

func Name(version, code byte) string {
	return NameOf(Norm(version, code))
}

func DumpOK(std byte) bool {
	if int(std) >= OpMax {
		return false
	}
	if std >= OpFUNCF || std == OpILOOP || std == OpJLOOP {
		return false
	}
	return true
}

func MagicRet(a byte, d uint16) bool {
	return a == 13 && d == 5120
}
