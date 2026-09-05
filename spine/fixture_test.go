package spine

import (
	"encoding/binary"
	"math"
)

type bin struct{ b []byte }

func (w *bin) varint(v int) {
	u := uint32(int32(v))
	for {
		c := byte(u & 0x7f)
		u >>= 7
		if u == 0 {
			w.b = append(w.b, c)
			return
		}
		w.b = append(w.b, c|0x80)
	}
}

func (w *bin) zig(v int) {
	w.varint(int((int32(v) << 1) ^ (int32(v) >> 31)))
}

func (w *bin) str(s string) {
	w.varint(len(s) + 1)
	w.b = append(w.b, s...)
}

func (w *bin) null() { w.varint(0) }

func (w *bin) f32(f float32) {
	w.b = binary.BigEndian.AppendUint32(w.b, math.Float32bits(f))
}

func (w *bin) i32(v int32) { w.b = binary.BigEndian.AppendUint32(w.b, uint32(v)) }

func (w *bin) i64(v int64) { w.b = binary.BigEndian.AppendUint64(w.b, uint64(v)) }

func (w *bin) u8(v byte) { w.b = append(w.b, v) }

func (w *bin) bool(v bool) {
	if v {
		w.u8(1)
		return
	}
	w.u8(0)
}

func (w *bin) floats(n int, v float32) {
	for i := 0; i < n; i++ {
		w.f32(v)
	}
}

func (w *bin) zeros(n int) {
	for i := 0; i < n; i++ {
		w.varint(0)
	}
}

func skeleton38() []byte {
	w := &bin{}
	w.str("NNl6b7LWQCOjpT/S6FC56pkhPbQ")
	w.str("3.8.99")
	w.f32(-10)
	w.f32(-20)
	w.f32(100)
	w.f32(200)
	w.bool(false)
	w.varint(1)
	w.str("img")
	w.varint(2)
	w.str("root")
	w.floats(8, 0)
	w.varint(0)
	w.bool(false)
	w.str("child")
	w.varint(0)
	w.f32(45)
	w.f32(3)
	w.f32(4)
	w.f32(1)
	w.f32(1)
	w.f32(0)
	w.f32(0)
	w.f32(50)
	w.varint(1)
	w.bool(true)
	w.varint(1)
	w.str("slot")
	w.varint(1)
	w.i32(-1)
	w.i32(-1)
	w.varint(1)
	w.varint(0)
	w.zeros(3)
	w.varint(1)
	w.varint(0)
	w.varint(1)
	w.varint(1)
	w.varint(0)
	w.u8(0)
	w.varint(0)
	w.f32(0)
	w.f32(1)
	w.f32(2)
	w.f32(1)
	w.f32(1)
	w.f32(64)
	w.f32(32)
	w.i32(-1)
	w.varint(0)
	w.varint(1)
	w.varint(1)
	w.zig(-3)
	w.f32(1.5)
	w.str("hi")
	w.null()
	w.varint(1)
	w.str("idle")
	w.varint(1)
	w.varint(0)
	w.varint(1)
	w.u8(0)
	w.varint(2)
	w.f32(0)
	w.varint(1)
	w.f32(0.5)
	w.varint(0)
	w.varint(1)
	w.varint(1)
	w.varint(1)
	w.u8(0)
	w.varint(2)
	w.f32(0)
	w.f32(10)
	w.u8(2)
	w.floats(4, 0.25)
	w.f32(1)
	w.f32(20)
	w.zeros(5)
	w.varint(1)
	w.f32(0.25)
	w.varint(0)
	w.zig(7)
	w.f32(2)
	w.bool(false)
	return w.b
}

func skeleton37() []byte {
	w := &bin{}
	w.str("hash")
	w.str("3.7.94")
	w.f32(10)
	w.f32(20)
	w.bool(true)
	w.f32(30)
	w.str("./images/")
	w.str("./audio/")
	w.varint(1)
	w.str("root")
	w.floats(8, 0)
	w.varint(0)
	w.i32(0x11223344)
	w.varint(1)
	w.str("s")
	w.varint(0)
	w.i32(-1)
	w.i32(0x00FF00FF)
	w.str("a")
	w.varint(2)
	w.zeros(3)
	w.varint(1)
	w.varint(0)
	w.varint(1)
	w.str("a")
	w.null()
	w.u8(1)
	w.varint(3)
	w.bool(false)
	w.floats(6, 1)
	w.i32(-1)
	w.zeros(3)
	return w.b
}

func skeleton40() []byte {
	w := &bin{}
	w.i64(12345)
	w.str("4.0.64")
	w.f32(0)
	w.f32(0)
	w.f32(64)
	w.f32(64)
	w.bool(false)
	w.varint(0)
	w.varint(1)
	w.str("root")
	w.floats(8, 0)
	w.varint(0)
	w.bool(false)
	w.zeros(7)
	w.varint(1)
	w.str("run")
	w.varint(1)
	w.varint(0)
	w.varint(1)
	w.varint(0)
	w.varint(1)
	w.u8(1)
	w.varint(3)
	w.varint(1)
	w.f32(0)
	w.f32(1)
	w.f32(2)
	w.f32(0.5)
	w.f32(3)
	w.f32(4)
	w.u8(2)
	w.floats(8, 0.1)
	w.f32(1.25)
	w.f32(5)
	w.f32(6)
	w.u8(1)
	w.zeros(6)
	return w.b
}
