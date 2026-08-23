package lua

import (
	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func skipIns(d *parse.Dump, p *parse.Proto, in parse.Ins, pc int) bool {
	if p == nil {
		return true
	}
	code := in.Op
	if d != nil {
		code = op.Norm(d.Version, in.Op)
	}
	if code == op.OpISNES && op.MagicRet(in.A, in.D) {
		return false
	}
	if !legal(d, p, in, code, pc) {
		return true
	}
	return holeSkip(d, p, in, code, pc)
}

func legal(d *parse.Dump, p *parse.Proto, in parse.Ins, code byte, pc int) bool {
	if !op.DumpOK(code) {
		return false
	}
	fr := int(p.Frame)
	a, b, c, dd := int(in.A), int(in.B), int(in.C), int(in.D)
	inFr := func(s int) bool { return s >= 0 && s < fr }
	n := len(p.Ins)
	hasJMP := false
	if d != nil && pc >= 0 && pc+1 < n && op.Norm(d.Version, p.Ins[pc+1].Op) == op.OpJMP {
		t := pc + 2 + p.Ins[pc+1].J()
		hasJMP = t >= 0 && t <= n
	}
	needJMP := pc >= 0
	destOK := pc < 0
	if pc >= 0 {
		t := pc + 1 + in.J()
		destOK = t >= 0 && t <= n
	}
	str := p.Str(p.GCKey(in.D, in.C)) != ""
	_, numC := p.Num(uint16(in.C))
	_, numAD := p.Num(p.NumKey(in.D, in.C))
	switch code {
	case op.OpISLT, op.OpISGE, op.OpISLE, op.OpISGT, op.OpISEQV, op.OpISNEV, op.OpISTC, op.OpISFC:
		return inFr(a) && inFr(dd) && (!needJMP || hasJMP)
	case op.OpISEQS, op.OpISNES:
		return inFr(a) && str && (!needJMP || hasJMP)
	case op.OpISEQN, op.OpISNEN:
		return inFr(a) && numAD && (!needJMP || hasJMP)
	case op.OpISEQP, op.OpISNEP:
		return inFr(a) && (!needJMP || hasJMP)
	case op.OpIST, op.OpISF:
		return inFr(dd) && (!needJMP || hasJMP)
	case op.OpMOV, op.OpNOT, op.OpUNM, op.OpLEN, op.OpBNOT:
		return inFr(a) && inFr(dd)
	case op.OpADDVN, op.OpSUBVN, op.OpMULVN, op.OpDIVVN, op.OpMODVN,
		op.OpADDNV, op.OpSUBNV, op.OpMULNV, op.OpDIVNV, op.OpMODNV:
		return inFr(a) && inFr(b) && numC
	case op.OpADDVV, op.OpSUBVV, op.OpMULVV, op.OpDIVVV, op.OpMODVV, op.OpPOW,
		op.OpBAND, op.OpBOR, op.OpBXOR, op.OpBSHL, op.OpBSHR, op.OpBSAR:
		return inFr(a) && inFr(b) && inFr(c)
	case op.OpCAT:
		return inFr(a) && inFr(b) && inFr(c) && b <= c
	case op.OpKSTR:
		return inFr(a) && str
	case op.OpKSHORT, op.OpKPRI:
		return inFr(a)
	case op.OpKNUM:
		return inFr(a) && numAD
	case op.OpKNIL:
		return inFr(a) && inFr(dd) && a <= dd
	case op.OpUGET:
		k := p.UVKey(in.D, in.C)
		return inFr(a) && k >= 0 && k < len(p.UV)
	case op.OpUSETV:
		return a < len(p.UV) && inFr(dd)
	case op.OpUSETS:
		return a < len(p.UV) && str
	case op.OpUSETN:
		return a < len(p.UV) && numAD
	case op.OpUSETP:
		return a < len(p.UV)
	case op.OpUCLO, op.OpISNEXT, op.OpJMP, op.OpLOOP:
		return destOK
	case op.OpFNEW:
		return p.FNew(in.D, in.C) != nil
	case op.OpTNEW:
		return inFr(a)
	case op.OpTDUP:
		k, ok := p.GC(p.GCKey(in.D, in.C))
		return inFr(a) && ok && k.Tab != nil
	case op.OpGGET, op.OpGSET:
		return inFr(a) && str
	case op.OpTGETV, op.OpTSETV, op.OpTGETR, op.OpTSETR:
		return inFr(a) && inFr(b) && inFr(c)
	case op.OpTGETS, op.OpTSETS:
		// B>=frame is payload, not an index: pack FNEW+TSETS B-fail=176, B==FNEW.A=0 (SSMgrBase Instance 36 01 07 05 B=5 FA=1). Fake 36 14 0e 61 B=97 stays skipped.
		_, ok := p.StrOK(uint16(in.C))
		return inFr(a) && inFr(b) && ok
	case op.OpTGETB, op.OpTSETB:
		return inFr(a) && inFr(b)
	case op.OpTSETM:
		return inFr(a) && numAD
	case op.OpCALL, op.OpCALLM, op.OpCALLT, op.OpCALLMT, op.OpITERC, op.OpITERN, op.OpVARG:
		return inFr(a)
	case op.OpRET0:
		return true
	case op.OpRET1, op.OpRET, op.OpRETM:
		return inFr(a)
	case op.OpFORI, op.OpJFORI, op.OpFORL, op.OpIFORL, op.OpJFORL:
		return a+3 < fr && destOK
	case op.OpITERL, op.OpIITERL, op.OpJITERL:
		return inFr(a) && destOK
	case op.OpISTYPE, op.OpISNUM, op.OpKCDATA:
		return false
	default:
		return false
	}
}

func cmpOK(p *parse.Proto, in parse.Ins, code byte) bool {
	if p == nil {
		return false
	}
	fr := int(p.Frame)
	switch code {
	case op.OpISLT, op.OpISGE, op.OpISLE, op.OpISGT, op.OpISEQV, op.OpISNEV, op.OpISTC, op.OpISFC:
		return int(in.A) < fr && int(in.D) < fr
	case op.OpISEQS, op.OpISNES, op.OpISEQN, op.OpISNEN, op.OpISEQP, op.OpISNEP:
		return int(in.A) < fr
	case op.OpIST, op.OpISF:
		return int(in.D) < fr
	default:
		return true
	}
}
