package lua

import (
	"strconv"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9'
		if !ok {
			return false
		}
	}
	return true
}

func okName(s string) bool {
	if !isIdent(s) {
		return false
	}
	switch s {
	case "and", "break", "do", "else", "elseif", "end", "false", "for", "function",
		"goto", "if", "in", "local", "nil", "not", "or", "repeat", "return",
		"then", "true", "until", "while":
		return false
	}
	return true
}

func (c *gen) body(from, to int) {
	if c.skip == nil {
		c.skip = map[int]bool{}
	}
	nins := len(c.p.Ins)
	if from < 0 {
		from = 0
	}
	if to > nins {
		to = nins
	}
	for pc := from; pc < to; pc++ {
		if c.skip[pc] {
			continue
		}
		if pc >= nins {
			return
		}
		if skipIns(c.d, c.p, c.p.Ins[pc], pc) {
			continue
		}
		if n := c.tryWhile(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryLoop(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryForIn(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryAndOr(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryIf(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryForNum(pc, to); n >= 0 {
			pc = n
			continue
		}
		if n := c.tryUclo(pc); n >= 0 {
			pc = n
			continue
		}
		in := c.p.Ins[pc]
		c.stmt(in, c.code(in), pc)
	}
}

func (c *gen) opAt(pc int) byte {
	if pc < 0 || pc >= len(c.p.Ins) {
		return 255
	}
	return c.code(c.p.Ins[pc])
}

func (c *gen) dest(pc int) int {
	if pc < 0 || pc >= len(c.p.Ins) {
		return pc
	}
	return pc + 1 + c.p.Ins[pc].J()
}

func (c *gen) mark(from, to int) {
	if to > len(c.p.Ins) {
		to = len(c.p.Ins)
	}
	for i := from; i < to; i++ {
		c.skip[i] = true
	}
}

func (c *gen) skipNoise(pc, lim int) int {
	for pc < lim && pc < len(c.p.Ins) {
		if c.opAt(pc) == op.OpUCLO {
			pc++
			continue
		}
		return pc
	}
	return pc
}

func isLoop(code byte) bool {
	return code == op.OpLOOP
}

func isIterL(code byte) bool {
	return code == op.OpITERL || code == op.OpIITERL || code == op.OpJITERL
}

func isIter(code byte) bool {
	return code == op.OpITERC || code == op.OpITERN
}

func isCmp(code byte) bool {
	return code <= op.OpISNEP || code == op.OpIST || code == op.OpISF || code == op.OpISTC || code == op.OpISFC
}

func invert(s string) string {
	if strings.HasPrefix(s, "not ") && !strings.Contains(s[4:], " ") {
		return s[4:]
	}
	pairs := [][2]string{
		{" >= ", " < "},
		{" <= ", " > "},
		{" ~= ", " == "},
		{" == ", " ~= "},
		{" > ", " <= "},
		{" < ", " >= "},
	}
	for _, p := range pairs {
		if strings.Count(s, p[0]) == 1 {
			return strings.Replace(s, p[0], p[1], 1)
		}
	}
	return "not (" + s + ")"
}

func (c *gen) cmp(code byte, in parse.Ins) string {
	l, r := c.get(int(in.A)), c.get(int(in.D))
	switch code {
	case op.OpISLT:
		return l + " < " + r
	case op.OpISGE:
		return l + " >= " + r
	case op.OpISLE:
		return l + " <= " + r
	case op.OpISGT:
		return l + " > " + r
	case op.OpISEQV:
		return l + " == " + r
	case op.OpISNEV:
		return l + " ~= " + r
	case op.OpISEQS:
		return l + " == " + c.gcstr(in.D, in.C)
	case op.OpISNES:
		return l + " ~= " + c.gcstr(in.D, in.C)
	case op.OpISEQN:
		return l + " == " + c.numAD(in.D, in.C)
	case op.OpISNEN:
		return l + " ~= " + c.numAD(in.D, in.C)
	case op.OpISEQP:
		return l + " == " + pri(in.D)
	case op.OpISNEP:
		return l + " ~= " + pri(in.D)
	case op.OpIST:
		return r
	case op.OpISF:
		return "not " + r
	case op.OpISTC:
		c.set(int(in.A), r)
		return r
	case op.OpISFC:
		c.set(int(in.A), r)
		return "not " + r
	default:
		return "true"
	}
}

func (c *gen) numD(d uint16) string {
	k, ok := c.p.Num(d)
	if !ok {
		return "n" + strconv.Itoa(int(d))
	}
	return numLit(k)
}

func (c *gen) numAD(d uint16, cc byte) string {
	return c.numD(c.p.NumKey(d, cc))
}
