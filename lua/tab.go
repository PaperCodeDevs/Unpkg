package lua

import (
	"math"
	"strconv"
	"strings"

	"github.com/PaperCodeDevs/Unpkg/parse"
)

func (c *gen) dup(d uint16, cc byte) string {
	k, ok := c.p.GC(c.p.GCKey(d, cc))
	if !ok || k.Tab == nil {
		return "{}"
	}
	return tabLit(k.Tab)
}

func (c *gen) tsetm(in parse.Ins) {
	tab := c.get(int(in.A) - 1)
	start := 1
	if k, ok := c.p.Num(in.D); ok {
		if k.IsInt {
			start = int(k.I) + 1
		} else {
			start = int(k.Float64()) + 1
		}
	}
	for i := 0; i < 32; i++ {
		slot := int(in.A) + i
		if slot >= len(c.slot) {
			break
		}
		val := c.get(slot)
		if i > 0 && val == "s"+strconv.Itoa(slot) {
			break
		}
		c.line("%s[%d] = %s", tab, start+i, val)
	}
}

func tabLit(t *parse.Tab) string {
	if t == nil {
		return "{}"
	}
	parts := make([]string, 0, len(t.Array)+len(t.Hash))
	for i, v := range t.Array {
		if i == 0 {
			if v.Kind != parse.TabNil {
				parts = append(parts, "[0]="+tabVal(v))
			}
			continue
		}
		parts = append(parts, tabVal(v))
	}
	for _, kv := range t.Hash {
		parts = append(parts, tabKey(kv[0])+"="+tabVal(kv[1]))
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func tabKey(k parse.TabK) string {
	if k.Kind == parse.TabStr && isIdent(k.Str) {
		return k.Str
	}
	if k.Kind == parse.TabStr {
		return "[" + quote(k.Str) + "]"
	}
	if k.Kind == parse.TabInt {
		return "[" + strconv.Itoa(int(k.I)) + "]"
	}
	return "[" + tabVal(k) + "]"
}

func tabVal(k parse.TabK) string {
	switch k.Kind {
	case parse.TabNil:
		return "nil"
	case parse.TabFalse:
		return "false"
	case parse.TabTrue:
		return "true"
	case parse.TabInt:
		return strconv.Itoa(int(k.I))
	case parse.TabNum:
		u := uint64(k.Lo) | uint64(k.Hi)<<32
		return strconv.FormatFloat(math.Float64frombits(u), 'g', -1, 64)
	case parse.TabStr:
		return quote(k.Str)
	default:
		return "nil"
	}
}

func tabNeedles(t *parse.Tab) []string {
	if t == nil {
		return nil
	}
	var out []string
	add := func(s string) {
		if s == "" || len(s) > 80 {
			return
		}
		for i := 0; i < len(s); i++ {
			if s[i] < 32 {
				return
			}
		}
		out = append(out, s)
	}
	for _, v := range t.Array {
		if v.Kind == parse.TabStr {
			add(v.Str)
		}
	}
	for _, kv := range t.Hash {
		if kv[0].Kind == parse.TabStr {
			add(kv[0].Str)
		}
		if kv[1].Kind == parse.TabStr {
			add(kv[1].Str)
		}
	}
	return out
}
