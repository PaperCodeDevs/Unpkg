package pkg

import (
	"fmt"
	"strings"
	"sync"
)

type Reader struct {
	data     []byte
	idx      *pkgIndex
	launcher *launcherIndex
	over     *overlayPair
	alt      *Reader
	cache    map[int][]byte
	bases    map[string]string
	lower    map[string]string
	mu       sync.Mutex
}

func OpenReader(p *Pkg) (*Reader, error) {
	if p == nil {
		return nil, fmt.Errorf("nil pkg")
	}
	plain, err := decodePkgIndex(p.Index)
	if err != nil {
		return nil, err
	}
	idx, err := parsePkgIndex(plain)
	if err == nil {
		sum := idx.compAt[len(idx.compAt)-1]
		if sum == uint64(len(p.Data)) {
			r := &Reader{data: p.Data, idx: idx, cache: map[int][]byte{}, lower: map[string]string{}}
			for n := range idx.byName {
				k := strings.ToLower(n)
				if _, ok := r.lower[k]; !ok {
					r.lower[k] = n
				}
			}
			r.initBases()
			return r, nil
		}
	}
	lx, err := parseMaterialIndex(plain, p.Data)
	if err != nil {
		lx, err = parseEngineIndex(plain, p.Data)
	}
	if err != nil {
		lx, err = parseLauncherIndex(plain, len(p.Data))
	}
	if err != nil {
		return nil, err
	}
	r := &Reader{data: p.Data, launcher: lx, lower: map[string]string{}}
	for n := range lx.byName {
		k := strings.ToLower(n)
		if _, ok := r.lower[k]; !ok {
			r.lower[k] = n
		}
	}
	return r, nil
}

func (r *Reader) Lookup(name string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	if r.over != nil {
		return r.over.lookup(name)
	}
	if r.launcher != nil {
		if alt, ok := r.lower[strings.ToLower(name)]; ok {
			name = alt
		}
		return r.launcher.lookup(r.data, name)
	}
	if r.idx == nil {
		return nil, fmt.Errorf("nil reader")
	}
	flag, ok := r.idx.byName[name]
	if !ok {
		if alt, ok2 := r.lower[strings.ToLower(name)]; ok2 {
			flag, ok = r.idx.byName[alt]
		}
	}
	if !ok {
		return nil, fmt.Errorf("not found: %s", name)
	}
	if flag&0x80000000 != 0 {
		i := int(flag & 0x7fffffff)
		if i < 0 || i >= len(r.idx.streams) {
			return nil, fmt.Errorf("bad stream %d", i)
		}
		rec := r.idx.streams[i]
		return r.read(rec.flags>>6, rec.off, rec.size)
	}
	i := int(flag)
	if i < 0 || i >= len(r.idx.files) {
		return nil, fmt.Errorf("bad index %d", i)
	}
	rec := r.idx.files[i]
	return r.read(rec.blockIndex(), rec.off, rec.size)
}

func (r *Reader) Names(prefix string) []string {
	out := make([]string, 0)
	if r == nil {
		return out
	}
	if r.over != nil {
		return r.over.names(prefix)
	}
	if r.launcher != nil {
		return r.launcher.names(prefix)
	}
	if r.idx == nil {
		return out
	}
	for n := range r.idx.byName {
		if prefix == "" || strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	return out
}

func (r *Reader) ConcatPlain() ([]byte, error) {
	if r == nil || r.idx == nil {
		return nil, fmt.Errorf("nil reader")
	}
	var n uint64
	for i := range r.idx.stor {
		n += uint64(r.idx.stor[i].uncomp)
	}
	if n > concatMaxBytes {
		return nil, fmt.Errorf("concat size %d", n)
	}
	out := make([]byte, 0, n)
	for i := range r.idx.stor {
		b, err := r.blockPlain(i)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

func (r *Reader) read(block uint32, off uint32, size uint32) ([]byte, error) {
	if size == 0 {
		return nil, fmt.Errorf("empty")
	}
	out := make([]byte, 0, size)
	pos := uint64(off)
	left := uint64(size)
	bi := int(block)
	for left > 0 {
		if bi < 0 || bi >= len(r.idx.stor) {
			return nil, fmt.Errorf("block %d", bi)
		}
		start := r.idx.uncompAt[bi]
		end := r.idx.uncompAt[bi+1]
		if pos < start || pos >= end {
			bi = r.blockAt(pos)
			if bi < 0 {
				return nil, fmt.Errorf("off %d not in stor", pos)
			}
			start = r.idx.uncompAt[bi]
			end = r.idx.uncompAt[bi+1]
		}
		plain, err := r.blockPlain(bi)
		if err != nil {
			return nil, err
		}
		local := int(pos - start)
		n := int(end - pos)
		if uint64(n) > left {
			n = int(left)
		}
		if local < 0 || local+n > len(plain) {
			return nil, fmt.Errorf("slice block %d local %d n %d plain %d", bi, local, n, len(plain))
		}
		out = append(out, plain[local:local+n]...)
		pos += uint64(n)
		left -= uint64(n)
		bi++
	}
	return out, nil
}

func (r *Reader) blockAt(pos uint64) int {
	for i := 0; i < len(r.idx.stor); i++ {
		if pos >= r.idx.uncompAt[i] && pos < r.idx.uncompAt[i+1] {
			return i
		}
	}
	return -1
}

func (r *Reader) blockPlain(i int) ([]byte, error) {
	r.mu.Lock()
	if b, ok := r.cache[i]; ok {
		r.mu.Unlock()
		return b, nil
	}
	r.mu.Unlock()
	plain, err := r.decodeBlock(i)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cache[i] = plain
	r.mu.Unlock()
	return plain, nil
}

func (r *Reader) DropCache() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.cache != nil {
		r.cache = map[int][]byte{}
	}
	r.mu.Unlock()
	if r.over != nil {
		r.over.base.DropCache()
		r.over.patch.DropCache()
	}
}

func (r *Reader) blockPeek(i int) ([]byte, error) {
	r.mu.Lock()
	b, ok := r.cache[i]
	r.mu.Unlock()
	if ok {
		return b, nil
	}
	return r.decodeBlock(i)
}

func (r *Reader) decodeBlock(i int) ([]byte, error) {
	st := r.idx.stor[i]
	cs := r.idx.compAt[i]
	ce := cs + uint64(st.comp)
	if ce > uint64(len(r.data)) {
		return nil, fmt.Errorf("comp slice %d", i)
	}
	raw := r.data[cs:ce]
	var plain []byte
	var err error
	if st.comp == st.uncomp {
		plain = append([]byte(nil), raw...)
	} else {
		if !lz4RatioOK(uint64(st.comp), uint64(st.uncomp)) {
			return nil, fmt.Errorf("uncomp ratio %d/%d", st.uncomp, st.comp)
		}
		plain, err = DecompressLZ4Block(raw, int(st.uncomp))
		if err != nil {
			return nil, err
		}
	}
	if len(plain) != int(st.uncomp) {
		return nil, fmt.Errorf("uncomp %d got %d", st.uncomp, len(plain))
	}
	return plain, nil
}
