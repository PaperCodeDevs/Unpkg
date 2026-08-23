package lua

import (
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
)

func (c *gen) tryForNum(pc, to int) int {
	code := c.opAt(pc)
	if code != op.OpFORI && code != op.OpJFORI {
		return -1
	}
	in := c.p.Ins[pc]
	lim := c.dest(pc)
	if lim <= pc+1 || lim > to {
		return -1
	}
	a := int(in.A)
	for i := 0; i < 4; i++ {
		n := c.p.SlotName(a+i, pc)
		if !okName(n) {
			n = c.p.SlotName(a+i, pc+1)
		}
		if okName(n) {
			c.set(a+i, n)
		}
	}
	c.line("for %s = %s, %s, %s do", c.get(a+3), c.get(a), c.get(a+1), c.get(a+2))
	c.indent++
	c.body(pc+1, lim)
	c.indent--
	c.line("end")
	c.mark(pc, lim)
	return lim - 1
}

func (c *gen) tryForIn(pc, to int) int {
	code := c.opAt(pc)
	iterPC := pc
	if code == op.OpISNEXT {
		tgt := c.dest(pc)
		if tgt > pc && tgt < to && (c.opAt(tgt) == op.OpITERC || c.opAt(tgt) == op.OpITERN) {
			iterPC = tgt
		} else {
			return pc
		}
	} else if code != op.OpITERC && code != op.OpITERN {
		return -1
	}
	iterl := -1
	for i := iterPC + 1; i < to; i++ {
		if skipIns(c.d, c.p, c.p.Ins[i], i) {
			continue
		}
		if !isIterL(c.opAt(i)) {
			continue
		}
		if c.dest(i) <= iterPC+1 {
			iterl = i
			break
		}
	}
	if iterl < 0 {
		if code == op.OpISNEXT {
			return pc
		}
		return -1
	}
	back := c.dest(iterl)
	bodyFrom := iterPC + 1
	if back >= 0 && back < iterPC {
		bodyFrom = back
	}
	if pc < bodyFrom {
		bodyFrom = pc + 1
	}
	in := c.p.Ins[iterPC]
	base := int(in.A)
	nres := int(in.B) - 1
	if nres < 1 {
		nres = 1
	}
	if nres > 4 {
		nres = 4
	}
	off := c.fr2
	vars := make([]string, nres)
	for i := 0; i < nres; i++ {
		slot := base + off + 3 + i
		name := c.localName(slot, iterPC)
		vars[i] = name
		c.set(slot, name)
	}
	c.skip[iterPC] = true
	c.skip[iterl] = true
	c.skip[pc] = true
	c.line("for %s in %s, %s, %s do", strings.Join(vars, ", "), c.get(base+off), c.get(base+off+1), c.get(base+off+2))
	c.indent++
	c.body(bodyFrom, iterl)
	c.indent--
	c.line("end")
	from := pc
	if bodyFrom < from {
		from = bodyFrom
	}
	c.mark(from, iterl+1)
	return iterl
}
