package lua

import (
	"fmt"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func List(d *parse.Dump) string {
	var b strings.Builder
	fmt.Fprintf(&b, "; ver=0x%02x flags=0x%x %s\n", d.Version, d.Flags, d.Name)
	disasmProto(&b, d, d.Main, 0)
	return b.String()
}

func disasmProto(b *strings.Builder, d *parse.Dump, p *parse.Proto, depth int) {
	if p == nil {
		return
	}
	pad := strings.Repeat("  ", depth)
	fmt.Fprintf(b, "%s; proto params=%d frame=%d kgc=%d kn=%d bc=%d\n",
		pad, p.Params, p.Frame, len(p.KGC), len(p.KNum), len(p.Ins))
	for i, k := range p.KGC {
		switch k.Kind {
		case parse.KStr:
			fmt.Fprintf(b, "%s; k%d %q\n", pad, i, k.Str)
		case parse.KChild:
			fmt.Fprintf(b, "%s; k%d child\n", pad, i)
		case parse.KTab:
			fmt.Fprintf(b, "%s; k%d table\n", pad, i)
		default:
			fmt.Fprintf(b, "%s; k%d kind=%d\n", pad, i, k.Kind)
		}
	}
	for pc, in := range p.Ins {
		code := op.Norm(d.Version, in.Op)
		name := op.NameOf(code)
		if op.ABC(code) {
			fmt.Fprintf(b, "%s%04d  %-6s %d %d %d\n", pad, pc, name, in.A, in.B, in.C)
		} else if code == op.OpJMP || code == op.OpFORI || code == op.OpFORL || code == op.OpITERL || code == op.OpLOOP || code == op.OpUCLO || code == op.OpISNEXT {
			fmt.Fprintf(b, "%s%04d  %-6s %d => %d\n", pad, pc, name, in.A, pc+1+in.J())
		} else {
			extra := ""
			if code == op.OpKSTR || code == op.OpGGET || code == op.OpGSET {
				if s := p.StrC(in.D, in.C); s != "" {
					extra = " ; " + quote(s)
				}
			} else if code == op.OpTGETS || code == op.OpTSETS {
				if s := p.Str(uint16(in.C)); s != "" {
					extra = " ; " + quote(s)
				}
			}
			fmt.Fprintf(b, "%s%04d  %-6s %d %d%s\n", pad, pc, name, in.A, in.D, extra)
		}
	}
	for _, k := range p.KGC {
		if k.Kind == parse.KChild && k.Child != nil {
			disasmProto(b, d, k.Child, depth+1)
		}
	}
}
