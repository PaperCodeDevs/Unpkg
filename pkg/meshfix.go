package pkg

func FixPlantY(m *BlockMesh) {
	if m == nil || len(m.Pos) < 6 {
		return
	}
	minY, maxY := m.Pos[1], m.Pos[1]
	for i := 1; i < len(m.Pos); i += 3 {
		if m.Pos[i] < minY {
			minY = m.Pos[i]
		}
		if m.Pos[i] > maxY {
			maxY = m.Pos[i]
		}
	}
	if maxY < 0.2 {
		for i := 1; i < len(m.Pos); i += 3 {
			m.Pos[i] = -m.Pos[i]
		}
		minY, maxY = -maxY, -minY
	}
	cy := (minY + maxY) / 2
	if cy > 0.55 && maxY-minY > 0.3 {
		sum := minY + maxY
		for i := 1; i < len(m.Pos); i += 3 {
			m.Pos[i] = sum - m.Pos[i]
		}
		minY, maxY = sum-maxY, sum-minY
	}
	if minY < -0.01 || minY > 0.12 {
		dy := -minY
		for i := 1; i < len(m.Pos); i += 3 {
			m.Pos[i] += dy
		}
	}
}

func ScaleMeshY(m *BlockMesh, s float32) {
	if m == nil || s <= 0 {
		return
	}
	minY := float32(0)
	if len(m.Pos) >= 2 {
		minY = m.Pos[1]
		for i := 1; i < len(m.Pos); i += 3 {
			if m.Pos[i] < minY {
				minY = m.Pos[i]
			}
		}
	}
	for i := 1; i < len(m.Pos); i += 3 {
		m.Pos[i] = (m.Pos[i]-minY)*s + minY
	}
}

func MeshHeight(m *BlockMesh) float32 {
	if m == nil || len(m.Pos) < 3 {
		return 0
	}
	minY, maxY := m.Pos[1], m.Pos[1]
	for i := 1; i < len(m.Pos); i += 3 {
		if m.Pos[i] < minY {
			minY = m.Pos[i]
		}
		if m.Pos[i] > maxY {
			maxY = m.Pos[i]
		}
	}
	return maxY - minY
}
