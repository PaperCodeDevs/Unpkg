package lua

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/op"
	"github.com/PaperCodeDevs/Unpkg/parse"
)

func Decompile(raw []byte) (out []byte, err error) {
	defer func() {
		if x := recover(); x != nil {
			out = nil
			err = fmt.Errorf("luajit: panic %v", x)
		}
	}()
	d, e := parse.Parse(raw)
	if e != nil {
		return nil, e
	}
	src := Source(d)
	if src == "" {
		return nil, fmt.Errorf("luajit: empty")
	}
	return []byte(src), nil
}

func Source(d *parse.Dump) string {
	var b strings.Builder
	if d.Name != "" {
		fmt.Fprintf(&b, "-- %s\n", d.Name)
	}
	emitFn(&b, d, d.Main, 0, true, "", false)
	return b.String()
}

func emitFn(b *strings.Builder, d *parse.Dump, p *parse.Proto, indent int, top bool, named string, local bool) {
	if p == nil {
		return
	}
	need := int(p.Frame) + 8
	if n := int(p.Params) + 8; n > need {
		need = n
	}
	if d.Flags&parse.FlagFR2 != 0 {
		need++
	}
	c := &gen{
		d:      d,
		p:      p,
		slot:   make([]string, need),
		indent: indent,
		out:    b,
		used:   map[*parse.Proto]bool{},
		loc:    map[int]string{},
	}
	if d.Flags&parse.FlagFR2 != 0 {
		c.fr2 = 1
	}
	for i := range c.slot {
		c.slot[i] = "s" + strconv.Itoa(i)
	}
	for i := 0; i < int(p.Params); i++ {
		c.set(i, "a"+strconv.Itoa(i))
	}
	c.bindParams()
	if !top {
		if named != "" && local {
			c.line("local function %s(%s)", named, strings.Join(c.params(), ", "))
		} else if named != "" {
			c.line("function %s(%s)", named, strings.Join(c.params(), ", "))
		} else {
			c.line("function(%s)", strings.Join(c.params(), ", "))
		}
		c.indent++
	}
	c.body(0, len(p.Ins))
	c.emitUnused()
	if !top {
		c.indent--
		c.line("end")
	}
}

type gen struct {
	d      *parse.Dump
	p      *parse.Proto
	slot   []string
	indent int
	fr2    int
	out    *strings.Builder
	skip   map[int]bool
	used   map[*parse.Proto]bool
	loc    map[int]string
}

func (c *gen) emitUnused() {
	if c.used == nil {
		c.used = map[*parse.Proto]bool{}
	}
	for i, k := range c.p.KGC {
		if k.Kind != parse.KChild || k.Child == nil || c.used[k.Child] {
			continue
		}
		var inner strings.Builder
		emitFn(&inner, c.d, k.Child, c.indent, false, "", false)
		c.line("local _c%d = %s", i, strings.TrimSpace(inner.String()))
	}
}

func (c *gen) params() []string {
	n := int(c.p.Params)
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = c.get(i)
	}
	if c.p.Flags&2 != 0 {
		out = append(out, "...")
	}
	return out
}

func (c *gen) bindParams() {
	if c.loc == nil {
		c.loc = map[int]string{}
	}
	for i := 0; i < int(c.p.Params); i++ {
		if n := c.p.SlotName(i, 0); okName(n) {
			c.set(i, n)
			c.loc[i] = n
			continue
		}
		n := "a" + strconv.Itoa(i)
		c.set(i, n)
		c.loc[i] = n
	}
}

func (c *gen) store(slot, pc int, expr string) {
	c.storeNamed(slot, pc, expr, false)
}

func (c *gen) storeNamed(slot, pc int, expr string, tnew bool) {
	if c.loc == nil {
		c.loc = map[int]string{}
	}
	name := c.freshName(slot, pc, tnew)
	prev := c.loc[slot]
	if prev != "" && okName(prev) && !c.varStartsAt(name, pc) {
		c.line("%s = %s", prev, expr)
		c.set(slot, prev)
		return
	}
	c.line("local %s = %s", name, expr)
	c.set(slot, name)
	c.loc[slot] = name
}

func (c *gen) storeN(slots []int, pc int, expr string) {
	if c.loc == nil {
		c.loc = map[int]string{}
	}
	names := make([]string, len(slots))
	all := len(slots) > 0
	for i, slot := range slots {
		name := c.localName(slot, pc)
		prev := c.loc[slot]
		if prev != "" && okName(prev) && !c.varStartsAt(name, pc) {
			names[i] = prev
		} else {
			names[i] = name
			all = false
		}
	}
	if all {
		c.line("%s = %s", strings.Join(names, ", "), expr)
	} else {
		c.line("local %s = %s", strings.Join(names, ", "), expr)
	}
	for i, slot := range slots {
		c.set(slot, names[i])
		c.loc[slot] = names[i]
	}
}

func (c *gen) localName(slot, pc int) string {
	return c.freshName(slot, pc, false)
}

func (c *gen) freshName(slot, pc int, tnew bool) string {
	cur := c.get(slot)
	try := func(n string) string {
		if !okName(n) || n == cur {
			return ""
		}
		if c.nameTaken(n, slot) {
			return ""
		}
		if tnew && !c.varStartsAt(n, pc) {
			return ""
		}
		return n
	}
	if n := try(c.p.SlotName(slot, pc+1)); n != "" {
		return n
	}
	if n := try(c.p.SlotName(slot, pc)); n != "" {
		return n
	}
	return "s" + strconv.Itoa(slot)
}

func (c *gen) nameTaken(name string, except int) bool {
	for i, s := range c.slot {
		if i != except && s == name {
			return true
		}
	}
	return false
}

func (c *gen) varStartsAt(name string, pc int) bool {
	if c.p == nil {
		return false
	}
	for _, v := range c.p.Var {
		if v.Name != name {
			continue
		}
		if int(v.Start) == pc || int(v.Start) == pc+1 {
			return true
		}
	}
	return false
}

func (c *gen) line(format string, args ...any) {
	c.out.WriteString(strings.Repeat("  ", c.indent))
	fmt.Fprintf(c.out, format, args...)
	c.out.WriteByte('\n')
}

func (c *gen) code(in parse.Ins) byte {
	return op.Norm(c.d.Version, in.Op)
}

func (c *gen) get(i int) string {
	if i < 0 || i >= len(c.slot) || c.slot[i] == "" {
		return "s" + strconv.Itoa(i)
	}
	return c.slot[i]
}

func (c *gen) set(i int, expr string) {
	for i >= len(c.slot) {
		c.slot = append(c.slot, "s"+strconv.Itoa(len(c.slot)))
	}
	c.slot[i] = expr
}

func (c *gen) gcstr(d uint16, cc byte) string {
	key := c.p.GCKey(d, cc)
	s := c.p.Str(key)
	if s == "" {
		return "k" + strconv.Itoa(int(key))
	}
	return quote(s)
}

func (c *gen) gcname(d uint16, cc byte) string {
	key := c.p.GCKey(d, cc)
	s := c.p.Str(key)
	if s != "" && isIdent(s) {
		return s
	}
	if s != "" {
		return "_G[" + quote(s) + "]"
	}
	return "_G[k" + strconv.Itoa(int(key)) + "]"
}
