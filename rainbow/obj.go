package rainbow

import (
	"fmt"
	"io"
	"strings"
)

func (m *Mesh) WriteOBJ(w io.Writer) error {
	if m == nil || len(m.Positions) < 9 || len(m.Indices) < 3 {
		return fmt.Errorf("mesh empty")
	}
	nvert := len(m.Positions) / 3
	hasN := len(m.Normals) >= nvert*3
	hasT := len(m.UV) >= nvert*2
	var b strings.Builder
	if m.Name != "" {
		fmt.Fprintf(&b, "o %s\n", m.Name)
	}
	for i := 0; i < nvert; i++ {
		fmt.Fprintf(&b, "v %g %g %g\n", m.Positions[i*3], m.Positions[i*3+1], m.Positions[i*3+2])
	}
	if hasN {
		for i := 0; i < nvert; i++ {
			fmt.Fprintf(&b, "vn %g %g %g\n", m.Normals[i*3], m.Normals[i*3+1], m.Normals[i*3+2])
		}
	}
	if hasT {
		for i := 0; i < nvert; i++ {
			fmt.Fprintf(&b, "vt %g %g\n", m.UV[i*2], m.UV[i*2+1])
		}
	}
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, c, d := m.Indices[i]+1, m.Indices[i+1]+1, m.Indices[i+2]+1
		switch {
		case hasN && hasT:
			fmt.Fprintf(&b, "f %d/%d/%d %d/%d/%d %d/%d/%d\n", a, a, a, c, c, c, d, d, d)
		case hasT:
			fmt.Fprintf(&b, "f %d/%d %d/%d %d/%d\n", a, a, c, c, d, d)
		case hasN:
			fmt.Fprintf(&b, "f %d//%d %d//%d %d//%d\n", a, a, c, c, d, d)
		default:
			fmt.Fprintf(&b, "f %d %d %d\n", a, c, d)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
