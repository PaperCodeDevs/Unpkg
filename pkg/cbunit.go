package pkg

import "encoding/binary"

type MaterialCBUnit struct {
	Index int
	Flags uint32
	Hash  uint32
	Size  uint32
	Slot  uint32
	Name  string
}

var engineCBNames = []string{"ViewCB", "PrimitiveCB"}

var engineCBByHash = func() map[uint32]string {
	m := make(map[uint32]string, len(engineCBNames))
	for _, n := range engineCBNames {
		m[RainbowNameHash(n)] = n
	}
	return m
}()

func EngineCBName(hash uint32) string {
	return engineCBByHash[hash]
}

func (bd DXBinding) CBUnits() []MaterialCBUnit {
	var out []MaterialCBUnit
	for i := 0; i+matBindUnit <= len(bd.Tail); i += matBindUnit {
		u := bd.Tail[i : i+matBindUnit]
		unit := MaterialCBUnit{
			Index: i / matBindUnit,
			Flags: binary.LittleEndian.Uint32(u),
			Hash:  binary.LittleEndian.Uint32(u[8:]),
			Size:  binary.LittleEndian.Uint32(u[12:]),
			Slot:  binary.LittleEndian.Uint32(u[16:]),
		}
		if unit.Flags == 0xFFFFFFFF && unit.Hash == 0 {
			continue
		}
		unit.Name = EngineCBName(unit.Hash)
		out = append(out, unit)
	}
	return out
}

func (bd DXBinding) CBufferOf(u MaterialCBUnit) (DXCBuffer, bool) {
	for _, cb := range bd.CBuffers {
		if cb.Size == u.Size && cb.Slot == u.Slot {
			return cb, true
		}
	}
	return DXCBuffer{}, false
}
