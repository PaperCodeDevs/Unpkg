package lua

import (
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func countBadOp(src string) int {
	n := 0
	for _, line := range strings.Split(src, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "for ") {
			rest := strings.TrimSpace(s[4:])
			eq := strings.Index(rest, " = ")
			in := strings.Index(rest, " in ")
			var head string
			if eq >= 0 && (in < 0 || eq < in) {
				head = rest[:eq]
			} else if in >= 0 {
				head = rest[:in]
			}
			for _, part := range strings.Split(head, ",") {
				if !okName(strings.TrimSpace(part)) {
					n++
				}
			}
		}
		if !strings.HasPrefix(s, "-- ") {
			continue
		}
		name := strings.Fields(s)
		if len(name) < 2 {
			continue
		}
		switch name[1] {
		case "LOOP", "ILOOP", "JLOOP", "ITERC", "ITERN", "ITERL", "IITERL",
			"TSETM", "VARG", "BAND", "BOR", "BXOR", "BSHL", "BSHR", "BSAR",
			"IFUNCV", "FUNCV", "FUNCF", "IFUNCF", "JFUNCF", "JFUNCV", "FUNCC", "FUNCCW", "?",
			"ISLT", "ISGE", "ISLE", "ISGT", "ISEQV", "ISNEV", "ISEQS", "ISNES",
			"ISEQN", "ISNEN", "ISEQP", "ISNEP", "ISTC", "ISFC", "IST", "ISF",
			"ISTYPE", "ISNUM", "KCDATA", "FORI", "JFORI":
			n++
		}
	}
	return n
}

func auditFn(d *parse.Dump, src string, cov *Cover) {
	need := 0
	walkProto(d.Main, func(p *parse.Proto) {
		for _, in := range p.Ins {
			if op.Norm(d.Version, in.Op) != op.OpFNEW {
				continue
			}
			if p.FNew(in.D, in.C) != nil {
				need++
			}
		}
	})
	if need == 0 {
		return
	}
	cov.NeedFn += need
	if countFnEmit(src) < need {
		cov.MissFn += need
		cov.note("fnew")
	}
}

func countFnEmit(src string) int {
	n := 0
	for _, line := range strings.Split(src, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "function ") || strings.HasPrefix(s, "function(") || strings.HasPrefix(s, "local function ") {
			n++
			continue
		}
		if strings.Contains(s, " = function(") || strings.Contains(s, " = function ") {
			n++
		}
	}
	return n
}

func protoHasMethod(d *parse.Dump, p *parse.Proto) bool {
	fr2 := 0
	if d.Flags&parse.FlagFR2 != 0 {
		fr2 = 1
	}
	n := len(p.Ins)
	for pc := 0; pc+3 < n; pc++ {
		in0 := p.Ins[pc]
		if skipIns(d, p, in0, pc) || skipIns(d, p, p.Ins[pc+1], pc+1) || skipIns(d, p, p.Ins[pc+2], pc+2) || skipIns(d, p, p.Ins[pc+3], pc+3) {
			continue
		}
		if op.Norm(d.Version, in0.Op) != op.OpTGETS {
			continue
		}
		a := int(in0.A)
		in1 := p.Ins[pc+1]
		if op.Norm(d.Version, in1.Op) != op.OpMOV {
			continue
		}
		if int(in1.A) != a+1+fr2 || int(in1.D) != a {
			continue
		}
		in2 := p.Ins[pc+2]
		if op.Norm(d.Version, in2.Op) != op.OpTGETS {
			continue
		}
		if int(in2.A) != a || int(in2.B) != a {
			continue
		}
		if !isIdent(p.Str(uint16(in2.C))) {
			continue
		}
		in3 := p.Ins[pc+3]
		code := op.Norm(d.Version, in3.Op)
		if code != op.OpCALL && code != op.OpCALLT && code != op.OpCALLM && code != op.OpCALLMT {
			continue
		}
		if int(in3.A) != a || in3.C < 2 {
			continue
		}
		return true
	}
	return false
}

func hasMethodColon(src string) bool {
	for i := 0; i < len(src); i++ {
		if src[i] != ':' {
			continue
		}
		j := i + 1
		if j >= len(src) {
			continue
		}
		c := src[j]
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			continue
		}
		j++
		for j < len(src) {
			c = src[j]
			ok := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
			if !ok {
				break
			}
			j++
		}
		if j < len(src) && src[j] == '(' {
			return true
		}
	}
	return false
}
