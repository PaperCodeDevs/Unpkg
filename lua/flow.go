package lua

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) {
			continue
		}
		if i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
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
		if n := c.tryNamedFn(pc, to); n >= 0 {
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

func (c *gen) tryNamedFn(pc, to int) int {
	if c.opAt(pc) != op.OpFNEW {
		return -1
	}
	in := c.p.Ins[pc]
	ch := c.p.FNew(in.D, in.C)
	if ch == nil {
		return -1
	}
	a := int(in.A)
	nxt := pc + 1
	for nxt < to && c.opAt(nxt) == op.OpUCLO {
		nxt++
	}
	if nxt >= to || c.skip[nxt] || skipIns(c.d, c.p, c.p.Ins[nxt], nxt) {
		return -1
	}
	n := c.p.Ins[nxt]
	title := ""
	switch c.code(n) {
	case op.OpGSET:
		if int(n.A) != a {
			return -1
		}
		s := c.p.Str(c.p.GCKey(n.D, n.C))
		if !okName(s) {
			return -1
		}
		title = s
	case op.OpTSETS:
		if int(n.A) != a {
			return -1
		}
		field, ok := c.p.StrOK(uint16(n.C))
		if !ok || !okName(field) {
			return -1
		}
		obj := c.get(int(n.B))
		if !identPath(obj) {
			return -1
		}
		title = obj + "." + field
	default:
		return -1
	}
	if c.used == nil {
		c.used = map[*parse.Proto]bool{}
	}
	c.used[ch] = true
	emitFn(c.out, c.d, ch, c.indent, false, title, false)
	c.set(a, title)
	c.skip[pc] = true
	c.skip[nxt] = true
	return nxt
}

func identPath(s string) bool {
	if s == "" {
		return false
	}
	for _, p := range strings.Split(s, ".") {
		if !okName(p) || slotName(p) {
			return false
		}
	}
	return true
}

func slotName(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "_c") {
		rest := s[2:]
		if rest == "" {
			return false
		}
		for _, c := range rest {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	if s[0] != 's' && s[0] != 'a' {
		return false
	}
	rest := s[1:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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
		return order("<", l, r)
	case op.OpISGE:
		return order(">=", l, r)
	case op.OpISLE:
		return order("<=", l, r)
	case op.OpISGT:
		return order(">", l, r)
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

func order(op, l, r string) string {
	if litNum(l) && !litNum(r) {
		switch op {
		case "<":
			op, l, r = ">", r, l
		case "<=":
			op, l, r = ">=", r, l
		case ">":
			op, l, r = "<", r, l
		case ">=":
			op, l, r = "<=", r, l
		}
	}
	return l + " " + op + " " + r
}

func litNum(s string) bool {
	i := 0
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		i = 1
	}
	if i >= len(s) {
		return false
	}
	dot := false
	for ; i < len(s); i++ {
		if s[i] == '.' {
			if dot {
				return false
			}
			dot = true
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
