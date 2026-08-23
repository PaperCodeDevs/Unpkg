package pkg

import (
	"crypto/md5"
	"fmt"
)

func readMaterialFiles(idx []byte, data []byte, nfile int) ([]assetRec, error) {
	files := make([]assetRec, nfile)
	for i := 0; i < nfile; i++ {
		b := 4 + i*materialRecBytes
		off := u32le(idx, b+16)
		size := u32le(idx, b+20)
		extra := u32le(idx, b+24)
		if off < uncompOrigin {
			return nil, fmt.Errorf("material off %d", off)
		}
		rel := int(off - uncompOrigin) // 偏移含 16B pkg 头
		if size == 0 || rel+int(size) > len(data) {
			return nil, fmt.Errorf("material range %d", i)
		}
		files[i] = assetRec{off: uint32(rel), size: size, flags: extra}
	}
	return files, nil
}

func verifyMaterialMD5(idx []byte, data []byte, nfile int) error {
	if nfile <= 0 {
		return fmt.Errorf("material empty")
	}
	checks := []int{0, nfile / 2, nfile - 1}
	for _, i := range checks {
		if i < 0 || i >= nfile {
			continue
		}
		b := 4 + i*materialRecBytes
		var hash [16]byte
		copy(hash[:], idx[b:b+16])
		off := int(u32le(idx, b+16) - uncompOrigin)
		size := int(u32le(idx, b+20))
		if off < 0 || off+size > len(data) {
			return fmt.Errorf("material md5 range %d", i)
		}
		sum := md5.Sum(data[off : off+size])
		if sum != hash {
			return fmt.Errorf("material md5 %d", i)
		}
	}
	return nil
}
