package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/parse"
)

type Batch struct {
	ParseOK   int
	ParseFail int
	LuaOK     int
	LuaFail   int
	Failed    []string
}

func RunDir(inDir, outDir string) (Batch, error) {
	var b Batch
	err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		low := strings.ToLower(path)
		if !strings.HasSuffix(low, ".lua") && !strings.HasSuffix(low, ".ljbc") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			b.fail(&b.ParseFail, path, "read")
			return nil
		}
		if !parse.IsDump(raw) {
			return nil
		}
		if _, err := parse.Parse(raw); err != nil {
			b.fail(&b.ParseFail, path, err.Error())
			return nil
		}
		b.ParseOK++
		src, err := Decompile(raw)
		if err != nil {
			b.fail(&b.LuaFail, path, err.Error())
			return nil
		}
		if len(src) == 0 {
			b.fail(&b.LuaFail, path, "empty")
			return nil
		}
		b.LuaOK++
		if outDir == "" {
			return nil
		}
		rel, err := filepath.Rel(inDir, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".lua"
		dst := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, src, 0o644)
	})
	return b, err
}

func (b *Batch) fail(n *int, path, msg string) {
	*n++
	if len(b.Failed) < 40 {
		b.Failed = append(b.Failed, path+": "+msg)
	}
}

func (b Batch) String() string {
	return fmt.Sprintf("parse ok=%d fail=%d lua ok=%d fail=%d", b.ParseOK, b.ParseFail, b.LuaOK, b.LuaFail)
}
