package pkg

import (
	"fmt"
)

const materialRecBytes = 28

func parseMaterialIndex(idx []byte, data []byte) (*launcherIndex, error) {
	files, byName, names, err := parseMaterial28(idx, data)
	if err != nil {
		return nil, err
	}
	return &launcherIndex{
		files:       files,
		byName:      byName,
		allNames:    names,
		fileBase:    4,
		nopad:       true,
		material:    true,
		resolveMode: 0,
	}, nil
}

func parseMaterial28(idx []byte, data []byte) ([]assetRec, map[string]uint32, []string, error) {
	if len(idx) < 8 {
		return nil, nil, nil, fmt.Errorf("material short")
	}
	nfile := int(u32le(idx, 0))
	if nfile < 1000 || nfile > 50000 {
		return nil, nil, nil, fmt.Errorf("material nfile %d", nfile)
	}
	mapOff := 4 + nfile*materialRecBytes
	if mapOff+4 > len(idx) {
		return nil, nil, nil, fmt.Errorf("material mapOff")
	}
	nmap := int(u32le(idx, mapOff))
	if nmap != nfile {
		return nil, nil, nil, fmt.Errorf("material nmap %d nfile %d", nmap, nfile)
	}
	files, err := readMaterialFiles(idx, data, nfile)
	if err != nil {
		return nil, nil, nil, err
	}
	byName, names, end, err := parseNoPadMapEnd(idx, mapOff+4, nmap)
	if err != nil {
		return nil, nil, nil, err
	}
	if end != len(idx) {
		return nil, nil, nil, fmt.Errorf("material map tail %d", len(idx)-end)
	}
	for _, flag := range byName {
		if int(flag) < 0 || int(flag) >= nfile {
			return nil, nil, nil, fmt.Errorf("material flag %d", flag)
		}
	}
	if err := verifyMaterialMD5(idx, data, nfile); err != nil {
		return nil, nil, nil, err
	}
	return files, byName, names, nil
}

func parseNoPadMapEnd(idx []byte, start, nmap int) (map[string]uint32, []string, int, error) {
	byName := make(map[string]uint32, nmap)
	names := make([]string, 0, nmap)
	pos := start
	for i := 0; i < nmap; i++ {
		if pos+8 > len(idx) {
			return nil, nil, 0, fmt.Errorf("map short %d", i)
		}
		nl := int(u32le(idx, pos))
		pos += 4
		if nl < 1 || nl > 400 || pos+nl+4 > len(idx) {
			return nil, nil, 0, fmt.Errorf("map nl %d at %d", nl, i)
		}
		name := string(idx[pos : pos+nl])
		pos += nl
		flag := u32le(idx, pos)
		pos += 4
		byName[name] = flag
		names = append(names, name)
	}
	return byName, names, pos, nil
}
