package lua

import (
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func (c *gen) tryAndOr(pc, to int) int {
	code := c.opAt(pc)
	if code != op.OpISTC && code != op.OpISFC {
		return -1
	}
	if !cmpOK(c.p, c.p.Ins[pc], code) {
		return -1
	}
	nins := len(c.p.Ins)
	if pc+1 >= nins || c.opAt(pc+1) != op.OpJMP {
		return -1
	}
	tgt := c.dest(pc + 1)
	if tgt <= pc+2 || tgt > nins {
		return -1
	}
	slot := int(c.p.Ins[pc].A)
	if !rhsWritesSlot(c.d.Version, c.p, pc+2, tgt, slot) {
		return -1
	}
	left := c.get(int(c.p.Ins[pc].D))
	c.set(slot, left)
	c.bodyAssign(pc+2, tgt)
	rhs := c.get(slot)
	join := " or "
	if code == op.OpISFC {
		join = " and "
	}
	expr := left + join + rhs
	c.store(slot, pc, expr)
	c.mark(pc, tgt)
	return tgt - 1
}

func (c *gen) bodyAssign(from, to int) {
	if c.skip == nil {
		c.skip = map[int]bool{}
	}
	nins := len(c.p.Ins)
	if to > nins {
		to = nins
	}
	for pc := from; pc < to; pc++ {
		if c.skip[pc] || pc >= nins {
			continue
		}
		if skipIns(c.d, c.p, c.p.Ins[pc], pc) {
			continue
		}
		if n := c.tryAndOr(pc, to); n >= 0 {
			pc = n
			continue
		}
		in := c.p.Ins[pc]
		c.stmt(in, c.code(in), pc)
	}
}

func (c *gen) tryUclo(pc int) int {
	if c.opAt(pc) != op.OpUCLO {
		return -1
	}
	dest := c.dest(pc)
	n := len(c.p.Ins)
	if dest <= pc+1 || dest >= n {
		return -1
	}
	in := c.p.Ins[dest]
	code := c.code(in)
	if code == op.OpISNES && op.MagicRet(in.A, in.D) {
		c.line("return")
		return pc
	}
	switch code {
	case op.OpRET, op.OpRETM, op.OpRET0, op.OpRET1:
		c.ret(in, code)
		return pc
	}
	return -1
}

func (c *gen) capture(from, to int) string {
	var b strings.Builder
	old := c.out
	c.out = &b
	c.body(from, to)
	c.out = old
	return b.String()
}

func rhsWritesSlot(ver byte, p *parse.Proto, from, to, slot int) bool {
	wrote := false
	n := len(p.Ins)
	if to > n {
		to = n
	}
	for pc := from; pc < to; pc++ {
		code := op.Norm(ver, p.Ins[pc].Op)
		if isCmp(code) && pc+1 < to && op.Norm(ver, p.Ins[pc+1].Op) == op.OpJMP {
			if code != op.OpISTC && code != op.OpISFC {
				return false
			}
			tgt := pc + 2 + p.Ins[pc+1].J()
			if tgt <= pc+2 || tgt > to {
				return false
			}
			if int(p.Ins[pc].A) == slot {
				wrote = true
			}
			if rhsWritesSlot(ver, p, pc+2, tgt, slot) {
				wrote = true
			}
			pc = tgt - 1
			continue
		}
		if isLoop(code) || isIter(code) || isIterL(code) || code == op.OpJMP {
			return false
		}
		if slotWrite(p.Ins[pc], code, slot) {
			wrote = true
		}
	}
	return wrote
}

func slotWrite(in parse.Ins, code byte, slot int) bool {
	a := int(in.A)
	switch code {
	case op.OpMOV, op.OpNOT, op.OpUNM, op.OpLEN, op.OpBNOT,
		op.OpKSTR, op.OpKSHORT, op.OpKNUM, op.OpKPRI, op.OpUGET, op.OpGGET,
		op.OpTGETV, op.OpTGETS, op.OpTGETB, op.OpTGETR,
		op.OpFNEW, op.OpTNEW, op.OpTDUP, op.OpCAT, op.OpVARG,
		op.OpADDVV, op.OpSUBVV, op.OpMULVV, op.OpDIVVV, op.OpMODVV, op.OpPOW,
		op.OpADDVN, op.OpSUBVN, op.OpMULVN, op.OpDIVVN, op.OpMODVN,
		op.OpADDNV, op.OpSUBNV, op.OpMULNV, op.OpDIVNV, op.OpMODNV,
		op.OpBAND, op.OpBOR, op.OpBXOR, op.OpBSHL, op.OpBSHR, op.OpBSAR,
		op.OpISTC, op.OpISFC:
		return a == slot
	case op.OpKNIL:
		return slot >= a && slot <= int(in.D)
	case op.OpCALL, op.OpCALLM:
		return a == slot && int(in.B) != 1
	}
	return false
}

func logicJump(d *parse.Dump, p *parse.Proto, pc int) bool {
	n := len(p.Ins)
	if d == nil || p == nil || pc+1 >= n {
		return false
	}
	if skipIns(d, p, p.Ins[pc], pc) {
		return false
	}
	code := op.Norm(d.Version, p.Ins[pc].Op)
	if code != op.OpISTC && code != op.OpISFC {
		return false
	}
	if op.Norm(d.Version, p.Ins[pc+1].Op) != op.OpJMP {
		return false
	}
	tgt := pc + 2 + p.Ins[pc+1].J()
	if tgt <= pc+2 || tgt > n {
		return false
	}
	return rhsWritesSlot(d.Version, p, pc+2, tgt, int(p.Ins[pc].A))
}

func protoHasLogic(d *parse.Dump, p *parse.Proto) bool {
	n := len(p.Ins)
	for pc := 0; pc+1 < n; pc++ {
		if logicJump(d, p, pc) {
			return true
		}
	}
	return false
}
