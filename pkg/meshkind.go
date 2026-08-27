package pkg

import "strings"

func MeshAllowed(typ string) bool {
	t := NormalizeType(typ)
	if t == "" {
		return false
	}
	if _, ok := cubeNoMesh[t]; ok {
		return false
	}
	if _, ok := meshType[t]; ok {
		return true
	}
	for _, n := range meshNeed {
		if t == n || strings.HasPrefix(t, n) || strings.HasSuffix(t, n) || strings.Contains(t, n) {
			return true
		}
	}
	return false
}

func IsPlantType(typ string) bool {
	t := NormalizeType(typ)
	for _, n := range []string{"flower", "herb", "lily", "sapling", "vine", "melonstem", "waterweed", "sawtooth", "leafpile", "rainbowgrass", "homelandplant", "icecrystalfern"} {
		if strings.Contains(t, n) {
			return true
		}
	}
	return false
}

func IsFenceType(typ string) bool {
	t := NormalizeType(typ)
	return strings.Contains(t, "fence") && !strings.Contains(t, "gate")
}

func IsFenceGate(typ string) bool {
	t := NormalizeType(typ)
	return strings.Contains(t, "gate")
}

func IsDoorType(typ string) bool {
	t := NormalizeType(typ)
	return strings.Contains(t, "door") && !strings.Contains(t, "trap")
}

func IsTrapdoorType(typ string) bool {
	return strings.Contains(NormalizeType(typ), "trapdoor")
}

func meshFamilyOK(typ, path string) bool {
	p := strings.ToLower(path)
	t := NormalizeType(typ)
	if strings.Contains(p, "trapdoor") || strings.Contains(p, "trap_door") {
		return IsTrapdoorType(t)
	}
	if IsTrapdoorType(t) {
		return strings.Contains(p, "trap")
	}
	if IsDoorType(t) {
		return strings.Contains(p, "door")
	}
	if IsFenceType(t) || IsFenceGate(t) {
		return strings.Contains(p, "fence") || strings.Contains(p, "gate")
	}
	if strings.Contains(t, "stair") {
		return strings.Contains(p, "stair")
	}
	if _, ok := cubeNoMesh[t]; ok {
		return false
	}
	return true
}

var cubeNoMesh = map[string]struct{}{
	"grass": {}, "basic": {}, "soil": {}, "log": {}, "sand": {}, "gravel": {},
	"glass": {}, "snow": {}, "farmland": {}, "buryland": {}, "stone": {},
	"plank": {}, "wood": {}, "dirt": {},
}

var meshType = map[string]struct{}{
	"door": {}, "simpledoor": {}, "highdoor": {}, "centerdoor": {}, "keydoor": {},
	"trapdoor": {}, "torch": {}, "lamp": {}, "windows": {}, "glasspane": {},
	"screen": {}, "inkscreen": {}, "fence": {}, "ironfence": {}, "fencegate": {},
	"newfence": {}, "schoolfence": {}, "stair": {}, "simplestair": {},
	"colorflower": {}, "grayherbs": {}, "colorherbs": {}, "waterlily": {},
	"lilypad": {}, "lilymodel": {}, "sapling": {}, "ladder": {},
}

var meshNeed = []string{
	"trapdoor", "door", "torch", "lamp", "window", "glasspane", "screen",
	"fence", "stair", "flower", "herb", "lily", "sapling", "ladder", "gate", "vine",
	"melonstem", "waterweed", "sawtooth", "leafpile", "rainbowgrass", "homelandplant", "icecrystalfern",
	"bed",
}
