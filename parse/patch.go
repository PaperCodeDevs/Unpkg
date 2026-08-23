package parse

import (
	"fmt"

	"github.com/PaperCodeDevs/Unpkg/op"
)

func RemapMiniWorld(raw []byte) ([]byte, error) {
	if !IsDump(raw) {
		return nil, fmt.Errorf("luajit: magic")
	}
	out := append([]byte(nil), raw...)
	out[3] = VerStd
	r := &reader{b: out, i: 4}
	flags, err := r.uleb32()
	if err != nil {
		return nil, err
	}
	if flags&FlagStrip == 0 {
		n, err := r.uleb32()
		if err != nil {
			return nil, err
		}
		if _, err = r.bytes(int(n)); err != nil {
			return nil, err
		}
	}
	for {
		if r.i >= len(out) {
			return nil, fmt.Errorf("luajit: truncated")
		}
		if out[r.i] == 0 {
			r.i++
			break
		}
		psz, err := r.uleb32()
		if err != nil {
			return nil, err
		}
		if psz == 0 {
			break
		}
		start := r.i
		if start+int(psz) > len(out) {
			return nil, fmt.Errorf("luajit: proto overflow")
		}
		if err := remapOps(out[start:start+int(psz)], flags); err != nil {
			return nil, err
		}
		r.i = start + int(psz)
	}
	return out[:r.i], nil
}

func remapOps(proto []byte, flags uint32) error {
	r := &reader{b: proto}
	if _, err := r.bytes(4); err != nil {
		return err
	}
	if _, err := r.uleb(); err != nil {
		return err
	}
	if _, err := r.uleb(); err != nil {
		return err
	}
	nbc, err := r.uleb32()
	if err != nil {
		return err
	}
	if flags&FlagStrip == 0 {
		dbg, err := r.uleb32()
		if err != nil {
			return err
		}
		if dbg != 0 {
			if _, err = r.uleb(); err != nil {
				return err
			}
			if _, err = r.uleb(); err != nil {
				return err
			}
		}
	}
	if r.i+int(nbc)*4 > len(proto) {
		return fmt.Errorf("luajit: bc overflow")
	}
	for i := 0; i < int(nbc); i++ {
		off := r.i + i*4
		code := op.ToStd(proto[off])
		switch code {
		case op.OpTGETR:
			code = op.OpTGETV
		case op.OpTSETR:
			code = op.OpTSETV
		}
		proto[off] = code
	}
	return nil
}
