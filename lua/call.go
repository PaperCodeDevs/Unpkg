package lua

import (
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func (c *gen) call(in parse.Ins, code byte, tail bool, pc int) {
	base := int(in.A)
	fn := c.get(base)
	narg := int(in.C) - 1
	if code == op.OpCALLM || code == op.OpCALLMT {
		narg = -1
	}
	args := []string{}
	start := base + 1 + c.fr2
	if narg < 0 {
		for i := start; i < len(c.slot) && i < start+16; i++ {
			args = append(args, c.get(i))
		}
		args = append(args, "...")
	} else {
		for i := 0; i < narg; i++ {
			args = append(args, c.get(start+i))
		}
	}
	call := methodCall(fn, args)
	nres := int(in.B) - 1
	if tail {
		c.line("return %s", call)
		return
	}
	if nres < 0 {
		nres = 1
	}
	if nres == 0 {
		c.line("%s", call)
		return
	}
	if nres == 1 {
		c.store(base, pc, call)
		return
	}
	if nres > 32 {
		nres = 32
	}
	slots := make([]int, nres)
	for i := 0; i < nres; i++ {
		slots[i] = base + i
	}
	c.storeN(slots, pc, call)
}

func methodCall(fn string, args []string) string {
	if len(args) > 0 {
		obj, meth, ok := splitDot(fn)
		if ok && obj == args[0] && isIdent(meth) {
			rest := strings.Join(args[1:], ", ")
			return obj + ":" + meth + "(" + rest + ")"
		}
	}
	return fn + "(" + strings.Join(args, ", ") + ")"
}

func splitDot(s string) (string, string, bool) {
	i := strings.LastIndexByte(s, '.')
	if i <= 0 || i+1 >= len(s) {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func (c *gen) ret(in parse.Ins, code byte) {
	if code == op.OpRET0 {
		c.line("return")
		return
	}
	n := 1
	base := int(in.A)
	if code == op.OpRET1 {
		n = 1
	} else if code == op.OpRET {
		n = int(in.D) - 1
	} else {
		n = -1
	}
	if n <= 0 {
		c.line("return %s", c.get(base))
		return
	}
	if n > 32 {
		n = 32
	}
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = c.get(base + i)
	}
	c.line("return %s", strings.Join(parts, ", "))
}
