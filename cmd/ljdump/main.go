package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaperCodeDevs/Unpkg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: ljdump <file-or-dir> [out-dir]\n")
		os.Exit(2)
	}
	in := os.Args[1]
	out := ""
	if len(os.Args) >= 3 {
		out = os.Args[2]
	}
	st, err := os.Stat(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if st.IsDir() {
		b, err := luajit.RunDir(in, out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(b.String())
		for _, m := range b.Failed {
			fmt.Fprintln(os.Stderr, m)
		}
		if b.ParseOK == 0 {
			os.Exit(1)
		}
		return
	}
	raw, err := os.ReadFile(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !luajit.IsDump(raw) {
		runScan(raw, out)
		return
	}
	d, err := luajit.Parse(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	src, err := luajit.Decompile(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("parse size=%d name=%q\n", d.Size, d.Name)
	if out == "" {
		os.Stdout.Write(src)
		return
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScan(blob []byte, outDir string) {
	hits := luajit.Scan(blob)
	ok, fail, luaOK, luaFail := 0, 0, 0, 0
	var cov luajit.Cover
	for i, h := range hits {
		if h.Err != nil {
			fail++
			if fail <= 20 {
				fmt.Fprintf(os.Stderr, "off=%d %v\n", h.Off, h.Err)
			}
			continue
		}
		ok++
		chunk := blob[h.Off : h.Off+h.Size]
		src, err := luajit.Decompile(chunk)
		if err != nil || len(src) == 0 {
			luaFail++
			if luaFail <= 20 {
				fmt.Fprintf(os.Stderr, "off=%d decompile %v\n", h.Off, err)
			}
			continue
		}
		luaOK++
		d, err := luajit.Parse(chunk)
		if err == nil {
			cv := luajit.Audit(d, string(src))
			cov.NeedElse += cv.NeedElse
			cov.MissElse += cv.MissElse
			cov.NeedLoop += cv.NeedLoop
			cov.MissLoop += cv.MissLoop
			cov.NeedForIn += cv.NeedForIn
			cov.MissForIn += cv.MissForIn
			cov.NeedTab += cv.NeedTab
			cov.MissTab += cv.MissTab
			cov.BadOp += cv.BadOp
			cov.NeedFn += cv.NeedFn
			cov.MissFn += cv.MissFn
			cov.NeedColon += cv.NeedColon
			cov.MissColon += cv.MissColon
			cov.NeedLogic += cv.NeedLogic
			cov.MissLogic += cv.MissLogic
			if !cv.Ok() && len(cov.Miss) < 20 {
				name := h.Name
				if name == "" {
					name = fmt.Sprintf("off=%d", h.Off)
				}
				cov.Miss = append(cov.Miss, name+": "+strings.Join(cv.Miss, ","))
			}
		}
		if outDir == "" {
			continue
		}
		name := h.Name
		if name == "" {
			name = fmt.Sprintf("dump_%d", i)
		}
		dst := filepath.Join(outDir, filepath.FromSlash(name)+".lua")
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(dst, src, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("scan hits=%d parse ok=%d fail=%d lua ok=%d fail=%d\n", len(hits), ok, fail, luaOK, luaFail)
	fmt.Println(cov.String())
	for _, m := range cov.Miss {
		fmt.Fprintln(os.Stderr, m)
	}
	if ok == 0 || luaFail > 0 || !cov.Ok() {
		os.Exit(1)
	}
}
