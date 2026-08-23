package luajit

import (
	"github.com/PaperCodeDevs/Unpkg/lua"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

type Dump = parse.Dump
type Proto = parse.Proto
type Batch = lua.Batch
type Hit = parse.Hit
type Cover = lua.Cover

func Parse(raw []byte) (*Dump, error) {
	return parse.Parse(raw)
}

func DumpSize(raw []byte) (int, error) {
	return parse.DumpSize(raw)
}

func RemapMiniWorld(raw []byte) ([]byte, error) {
	return parse.RemapMiniWorld(raw)
}

func IsDump(raw []byte) bool {
	return parse.IsDump(raw)
}

func IsMiniWorld(raw []byte) bool {
	return parse.IsMiniWorld(raw)
}

func Scan(blob []byte) []Hit {
	return parse.Scan(blob)
}

func Decompile(raw []byte) ([]byte, error) {
	return lua.Decompile(raw)
}

func Lua(d *Dump) string {
	return lua.Source(d)
}

func Disassemble(d *Dump) string {
	return lua.List(d)
}

func Audit(d *Dump, src string) Cover {
	return lua.Audit(d, src)
}

func RunDir(inDir, outDir string) (Batch, error) {
	return lua.RunDir(inDir, outDir)
}
