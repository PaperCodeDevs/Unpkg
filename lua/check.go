package lua

import (
	"fmt"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

type Cover struct {
	NeedElse  int
	MissElse  int
	NeedLoop  int
	MissLoop  int
	NeedForIn int
	MissForIn int
	NeedTab   int
	MissTab   int
	BadOp     int
	NeedFn    int
	MissFn    int
	NeedColon int
	MissColon int
	NeedLogic int
	MissLogic int
	Miss      []string
}

func (c Cover) Ok() bool {
	return c.MissElse == 0 && c.MissLoop == 0 && c.MissForIn == 0 && c.MissTab == 0 && c.BadOp == 0 && c.MissFn == 0 && c.MissColon == 0 && c.MissLogic == 0
}

func (c Cover) String() string {
	return fmt.Sprintf("else %d/%d loop %d/%d forin %d/%d tab %d/%d badop=%d fn %d/%d colon %d/%d logic %d/%d",
		c.NeedElse-c.MissElse, c.NeedElse,
		c.NeedLoop-c.MissLoop, c.NeedLoop,
		c.NeedForIn-c.MissForIn, c.NeedForIn,
		c.NeedTab-c.MissTab, c.NeedTab,
		c.BadOp,
		c.NeedFn-c.MissFn, c.NeedFn,
		c.NeedColon-c.MissColon, c.NeedColon,
		c.NeedLogic-c.MissLogic, c.NeedLogic)
}

func Audit(d *parse.Dump, src string) Cover {
	var cov Cover
	if d == nil || d.Main == nil {
		return cov
	}
	hasElse := lineHas(src, "else") || strings.Contains(src, "elseif ")
	hasLoop := lineHas(src, "repeat") || strings.Contains(src, "while ")
	hasForIn := strings.Contains(src, " for ") || strings.HasPrefix(strings.TrimSpace(src), "for ") || strings.Contains(src, "\nfor ")
	hasIn := strings.Contains(src, " in ")
	hasColon := hasMethodColon(src)
	hasLogic := strings.Contains(src, " and ") || strings.Contains(src, " or ")
	walkProto(d.Main, func(p *parse.Proto) {
		if protoHasElse(d, p) {
			cov.NeedElse++
			if !hasElse {
				cov.MissElse++
				cov.note("else")
			}
		}
		if protoHasLoop(d, p) {
			cov.NeedLoop++
			if !hasLoop {
				cov.MissLoop++
				cov.note("loop")
			}
		}
		if protoHasForIn(d, p) {
			cov.NeedForIn++
			if !hasForIn || !hasIn {
				cov.MissForIn++
				cov.note("forin")
			}
		}
		if protoHasMethod(d, p) {
			cov.NeedColon++
			if !hasColon {
				cov.MissColon++
				cov.note("colon")
			}
		}
		if protoHasLogic(d, p) {
			cov.NeedLogic++
			if !hasLogic {
				cov.MissLogic++
				cov.note("logic")
			}
		}
		for pc, in := range p.Ins {
			if skipIns(d, p, in, pc) {
				continue
			}
			if op.Norm(d.Version, in.Op) != op.OpTDUP {
				continue
			}
			k, ok := p.GC(p.GCKey(in.D, in.C))
			if !ok || k.Tab == nil {
				continue
			}
			needles := tabNeedles(k.Tab)
			if len(needles) == 0 {
				if len(k.Tab.Array) > 1 || len(k.Tab.Hash) > 0 {
					cov.NeedTab++
					if !strings.Contains(src, "{ ") {
						cov.MissTab++
						cov.note("tab")
					}
				}
				continue
			}
			cov.NeedTab++
			okn := false
			for _, n := range needles {
				if strings.Contains(src, n) || strings.Contains(src, quote(n)) {
					okn = true
					break
				}
			}
			if !okn {
				cov.MissTab++
				cov.note("tab")
			}
		}
	})
	cov.BadOp = countBadOp(src)
	if cov.BadOp > 0 {
		cov.note("badop")
	}
	auditFn(d, src, &cov)
	return cov
}

func (c *Cover) note(msg string) {
	if len(c.Miss) < 8 {
		c.Miss = append(c.Miss, msg)
	}
}

func walkProto(p *parse.Proto, fn func(*parse.Proto)) {
	if p == nil {
		return
	}
	fn(p)
	for _, k := range p.KGC {
		if k.Kind == parse.KChild {
			walkProto(k.Child, fn)
		}
	}
}

func protoHasElse(d *parse.Dump, p *parse.Proto) bool {
	n := len(p.Ins)
	for pc := 0; pc+1 < n; pc++ {
		if skipIns(d, p, p.Ins[pc], pc) {
			continue
		}
		if !isCmp(op.Norm(d.Version, p.Ins[pc].Op)) {
			continue
		}
		if logicJump(d, p, pc) {
			continue
		}
		if op.Norm(d.Version, p.Ins[pc+1].Op) != op.OpJMP {
			continue
		}
		tgt := pc + 2 + p.Ins[pc+1].J()
		if tgt <= pc+2 || tgt > n {
			continue
		}
		last := tgt - 1
		for last > pc+2 && op.Norm(d.Version, p.Ins[last].Op) == op.OpUCLO {
			last--
		}
		if last < pc+2 || last >= n {
			continue
		}
		if op.Norm(d.Version, p.Ins[last].Op) != op.OpJMP {
			continue
		}
		if tgt < n && isIter(op.Norm(d.Version, p.Ins[tgt].Op)) {
			continue
		}
		if last > 0 && isCmp(op.Norm(d.Version, p.Ins[last-1].Op)) {
			continue
		}
		after := last + 1 + p.Ins[last].J()
		if after > tgt && after <= n && !elseSpill(d, p, pc, after) {
			return true
		}
	}
	return false
}

func elseSpill(d *parse.Dump, p *parse.Proto, pc, tgt int) bool {
	n := len(p.Ins)
	for e := 0; e+1 < pc; e++ {
		if !isCmp(op.Norm(d.Version, p.Ins[e].Op)) {
			continue
		}
		if op.Norm(d.Version, p.Ins[e+1].Op) != op.OpJMP {
			continue
		}
		te := e + 2 + p.Ins[e+1].J()
		if te <= e+2 || te > n {
			continue
		}
		if pc > e+1 && pc < te && tgt > te {
			return true
		}
	}
	return false
}

func protoHasLoop(d *parse.Dump, p *parse.Proto) bool {
	n := len(p.Ins)
	for pc, in := range p.Ins {
		if skipIns(d, p, in, pc) {
			continue
		}
		if !isLoop(op.Norm(d.Version, in.Op)) {
			continue
		}
		after := pc + 1 + in.J()
		if after <= pc+2 || after > n {
			continue
		}
		if loopBack(p, d.Version, pc, after) {
			return true
		}
		lim := pc
		if lim > 64 {
			lim = pc - 64
		} else {
			lim = 0
		}
		for h := lim; h <= pc; h++ {
			if loopBack(p, d.Version, h, after) && loopBackTo(p, d.Version, pc, after, h) {
				return true
			}
		}
	}
	return false
}

func loopBack(p *parse.Proto, ver byte, header, after int) bool {
	return loopBackTo(p, ver, header, after, header)
}

func loopBackTo(p *parse.Proto, ver byte, l, after, header int) bool {
	if l+1 >= after || after > len(p.Ins) {
		return false
	}
	for i := after - 1; i > l; i-- {
		if op.Norm(ver, p.Ins[i].Op) != op.OpJMP {
			continue
		}
		d := i + 1 + p.Ins[i].J()
		if d >= header && d <= l {
			return true
		}
	}
	return false
}

func protoHasForIn(d *parse.Dump, p *parse.Proto) bool {
	n := len(p.Ins)
	for pc, in := range p.Ins {
		if skipIns(d, p, in, pc) {
			continue
		}
		code := op.Norm(d.Version, in.Op)
		if code != op.OpITERC && code != op.OpITERN {
			continue
		}
		for i := pc + 1; i < n; i++ {
			if skipIns(d, p, p.Ins[i], i) {
				continue
			}
			if !isIterL(op.Norm(d.Version, p.Ins[i].Op)) {
				continue
			}
			back := i + 1 + p.Ins[i].J()
			if back <= pc+1 {
				return true
			}
		}
	}
	return false
}

func lineHas(src, word string) bool {
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) == word {
			return true
		}
	}
	return false
}
