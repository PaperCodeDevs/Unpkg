package lua

import (
	"github.com/PaperCodeDevs/Unpkg/op"
)

func (c *gen) tryWhile(pc, to int) int {
	if c.opAt(pc) != op.OpJMP || pc+1 >= to || !isLoop(c.opAt(pc+1)) {
		return -1
	}
	condPC := c.skipNoise(c.dest(pc), to)
	if condPC <= pc+2 || condPC+1 >= to {
		return -1
	}
	if !isCmp(c.opAt(condPC)) || c.opAt(condPC+1) != op.OpJMP {
		return -1
	}
	back := c.dest(condPC + 1)
	if back > pc+2 {
		return -1
	}
	after := c.dest(pc + 1)
	if after < condPC+2 {
		after = condPC + 2
	}
	if after > to {
		after = to
	}
	cond := invert(c.cmp(c.opAt(condPC), c.p.Ins[condPC]))
	c.line("while %s do", cond)
	c.indent++
	c.body(pc+2, condPC)
	c.indent--
	c.line("end")
	c.mark(pc, after)
	return after - 1
}

func (c *gen) tryLoop(pc, to int) int {
	if n := c.emitLoop(pc, pc, to); n >= 0 {
		return n
	}
	if isLoop(c.opAt(pc)) {
		return -1
	}
	lim := pc + 64
	if lim > to {
		lim = to
	}
	for l := pc + 1; l < lim; l++ {
		if c.skip[l] || !isLoop(c.opAt(l)) {
			continue
		}
		if n := c.emitLoop(pc, l, to); n >= 0 {
			return n
		}
	}
	return -1
}

func (c *gen) emitLoop(header, l, to int) int {
	if !isLoop(c.opAt(l)) || c.skip[l] {
		return -1
	}
	after := c.dest(l)
	nins := len(c.p.Ins)
	if after > nins {
		after = nins
	}
	if after <= l+2 {
		return -1
	}
	backPC := -1
	condPC := -1
	for i := after - 1; i > l; i-- {
		if c.opAt(i) != op.OpJMP {
			continue
		}
		d := c.dest(i)
		if d < header || d > l {
			continue
		}
		backPC = i
		if i > 0 && isCmp(c.opAt(i-1)) {
			condPC = i - 1
		}
		break
	}
	if backPC < 0 {
		return -1
	}
	c.skip[l] = true
	bodyTo := backPC
	if bodyTo > to {
		bodyTo = to
	}
	endMark := after
	if endMark > to {
		endMark = to
	}
	if header == l && condPC >= 0 && condPC <= bodyTo {
		until := c.cmp(c.opAt(condPC), c.p.Ins[condPC])
		c.line("repeat")
		c.indent++
		c.body(l+1, condPC)
		c.indent--
		c.line("until %s", until)
	} else {
		c.line("while true do")
		c.indent++
		from := header
		if header == l {
			from = l + 1
		}
		c.body(from, bodyTo)
		c.indent--
		c.line("end")
	}
	c.mark(header, endMark)
	return endMark - 1
}
