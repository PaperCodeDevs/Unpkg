package spine

import (
	"encoding/binary"
	"fmt"
	"math"
)

type input struct {
	b       []byte
	pos     int
	strings []string
	err     error
}

func (in *input) fail(format string, a ...any) {
	if in.err == nil {
		in.err = fmt.Errorf("spine: "+format, a...)
	}
}

func (in *input) need(n int) bool {
	if in.err != nil {
		return false
	}
	if n < 0 || in.pos+n > len(in.b) {
		in.fail("数据截断 pos=%d need=%d len=%d", in.pos, n, len(in.b))
		return false
	}
	return true
}

func (in *input) byte() byte {
	if !in.need(1) {
		return 0
	}
	v := in.b[in.pos]
	in.pos++
	return v
}

func (in *input) sbyte() int {
	return int(int8(in.byte()))
}

func (in *input) bool() bool {
	return in.byte() != 0
}

func (in *input) int32() int32 {
	if !in.need(4) {
		return 0
	}
	v := int32(binary.BigEndian.Uint32(in.b[in.pos:]))
	in.pos += 4
	return v
}

func (in *input) uint32() uint32 {
	return uint32(in.int32())
}

func (in *input) int64() int64 {
	if !in.need(8) {
		return 0
	}
	v := int64(binary.BigEndian.Uint64(in.b[in.pos:]))
	in.pos += 8
	return v
}

func (in *input) short() int {
	if !in.need(2) {
		return 0
	}
	v := int(binary.BigEndian.Uint16(in.b[in.pos:]))
	in.pos += 2
	return v
}

func (in *input) float() float32 {
	return math.Float32frombits(in.uint32())
}

func (in *input) varint(optimizePositive bool) int {
	var result uint32
	shift := 0
	for i := 0; i < 5; i++ {
		b := in.byte()
		result |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	if optimizePositive {
		return int(int32(result))
	}
	return int(int32(result>>1) ^ -int32(result&1))
}

func (in *input) count() int {
	n := in.varint(true)
	if n < 0 || n > len(in.b)-in.pos {
		in.fail("计数越界 %d pos=%d", n, in.pos)
		return 0
	}
	return n
}

func (in *input) strNull() (string, bool) {
	n := in.varint(true)
	if n == 0 {
		return "", true
	}
	n--
	if n < 0 || !in.need(n) {
		in.fail("字符串长度 %d pos=%d", n, in.pos)
		return "", true
	}
	s := string(in.b[in.pos : in.pos+n])
	in.pos += n
	return s, false
}

func (in *input) str() string {
	s, _ := in.strNull()
	return s
}

func (in *input) strRefNull() (string, bool) {
	i := in.varint(true)
	if i == 0 {
		return "", true
	}
	if i < 0 || i > len(in.strings) {
		in.fail("字符串引用 %d/%d pos=%d", i, len(in.strings), in.pos)
		return "", true
	}
	return in.strings[i-1], false
}

func (in *input) strRef() string {
	s, _ := in.strRefNull()
	return s
}

func (in *input) floats(n int) []float32 {
	if n < 0 || !in.need(n*4) {
		return nil
	}
	out := make([]float32, n)
	for i := range out {
		out[i] = in.float()
	}
	return out
}

func (in *input) shorts() []uint16 {
	n := in.count()
	if !in.need(n * 2) {
		return nil
	}
	out := make([]uint16, n)
	for i := range out {
		out[i] = uint16(in.short())
	}
	return out
}

func (in *input) indices() []int {
	n := in.count()
	var out []int
	for i := 0; i < n && in.err == nil; i++ {
		out = append(out, in.varint(true))
	}
	return out
}
