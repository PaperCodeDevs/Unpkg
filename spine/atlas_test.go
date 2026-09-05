package spine

import "testing"

const atlasV3 = "\r\npage.png\r\nsize: 932,88\r\nformat: RGBA8888\r\nfilter: Linear,Linear\r\nrepeat: none\r\n" +
	"sd_lan\r\n  rotate: false\r\n  xy: 732, 2\r\n  size: 65, 81\r\n  orig: 73, 87\r\n  offset: 4, 4\r\n  index: -1\r\n" +
	"sx_huang\r\n  rotate: true\r\n  xy: 2, 2\r\n  size: 81, 242\r\n  split: 1, 2, 3, 4\r\n  pad: 5, 6, 7, 8\r\n  orig: 81, 242\r\n  offset: 0, 0\r\n  index: 3\r\n" +
	"\r\npage2.png\r\nsize: 64,32\r\nformat: RGBA8888\r\nfilter: Nearest,Nearest\r\nrepeat: xy\r\n" +
	"solo\r\n  rotate: false\r\n  xy: 0, 0\r\n  size: 64, 32\r\n  orig: 64, 32\r\n  offset: 0, 0\r\n  index: -1\r\n"

const atlasV4 = "\npage.png\nsize: 256, 128\nformat: RGBA8888\nfilter: Linear, Linear\nrepeat: none\npma: true\nscale: 0.5\n" +
	"hero/idle\n  bounds: 10, 20, 30, 40\n  rotate: 90\n  offsets: 1, 2, 50, 60\n  index: 2\n" +
	"plain\n  bounds: 0, 0, 8, 8\n"

func TestParseAtlasV3(t *testing.T) {
	a, err := ParseAtlas([]byte(atlasV3))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Pages) != 2 {
		t.Fatalf("pages=%d", len(a.Pages))
	}
	p := a.Pages[0]
	if p.Name != "page.png" || p.Width != 932 || p.Height != 88 || p.Format != "RGBA8888" || p.MinFilter != "Linear" || p.MagFilter != "Linear" || p.Repeat != "none" {
		t.Fatalf("page=%+v", p)
	}
	if len(p.Regions) != 2 {
		t.Fatalf("regions=%d", len(p.Regions))
	}
	r := p.Regions[0]
	if r.Name != "sd_lan" || r.X != 732 || r.Y != 2 || r.Width != 65 || r.Height != 81 || r.OrigWidth != 73 || r.OrigHeight != 87 || r.OffsetX != 4 || r.OffsetY != 4 || r.Index != -1 || r.Rotate || r.Page != 0 {
		t.Fatalf("region=%+v", r)
	}
	r = p.Regions[1]
	if !r.Rotate || r.Degrees != 90 || r.Index != 3 || len(r.Splits) != 4 || r.Splits[3] != 4 || len(r.Pads) != 4 || r.Pads[0] != 5 {
		t.Fatalf("region=%+v", r)
	}
	if x, y, w, h := r.PageRect(); x != 2 || y != 2 || w != 242 || h != 81 {
		t.Fatalf("pageRect=%d,%d %dx%d", x, y, w, h)
	}
	if x, y, w, h := r.ScaledRect(932, 88, 1024, 64); x != 2 || y != 1 || w != 266 || h != 59 {
		t.Fatalf("scaledRect=%d,%d %dx%d", x, y, w, h)
	}
	if x, y, w, h := r.ScaledRect(932, 88, 932, 88); x != 2 || y != 2 || w != 242 || h != 81 {
		t.Fatalf("scaledRect same=%d,%d %dx%d", x, y, w, h)
	}
	p2 := a.Pages[1]
	if p2.Name != "page2.png" || p2.Repeat != "xy" || p2.MinFilter != "Nearest" || len(p2.Regions) != 1 || p2.Regions[0].Page != 1 || p2.Regions[0].Name != "solo" {
		t.Fatalf("page2=%+v", p2)
	}
}

func TestParseAtlasV4(t *testing.T) {
	a, err := ParseAtlas([]byte(atlasV4))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Pages) != 1 {
		t.Fatalf("pages=%d", len(a.Pages))
	}
	p := a.Pages[0]
	if !p.PMA || p.Scale != 0.5 || p.Width != 256 || p.Height != 128 || p.MagFilter != "Linear" {
		t.Fatalf("page=%+v", p)
	}
	if len(p.Regions) != 2 {
		t.Fatalf("regions=%d", len(p.Regions))
	}
	r := p.Regions[0]
	if r.Name != "hero/idle" || r.X != 10 || r.Y != 20 || r.Width != 30 || r.Height != 40 || !r.Rotate || r.Degrees != 90 || r.OffsetX != 1 || r.OffsetY != 2 || r.OrigWidth != 50 || r.OrigHeight != 60 || r.Index != 2 {
		t.Fatalf("region=%+v", r)
	}
	if x, y, w, h := r.PageRect(); x != 10 || y != 20 || w != 40 || h != 30 {
		t.Fatalf("pageRect=%d,%d %dx%d", x, y, w, h)
	}
	r = p.Regions[1]
	if r.Name != "plain" || r.OrigWidth != 8 || r.OrigHeight != 8 || r.Index != -1 || r.Rotate || r.Degrees != 0 {
		t.Fatalf("region=%+v", r)
	}
}

func TestParseAtlasErrors(t *testing.T) {
	cases := map[string]string{
		"orphan":  "size: 1,1\n",
		"empty":   "\r\n\r\n",
		"badint":  "p.png\nsize: a,b\n",
		"missing": "p.png\nsize: 1\n",
	}
	for name, text := range cases {
		if _, err := ParseAtlas([]byte(text)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	a, err := ParseAtlas([]byte("p.png\nsize: 4,4\nfoo: bar\nr\n  bogus: 1\n  xy: 1, 2\n  size: 1, 1\n"))
	if err != nil || len(a.Pages) != 1 || len(a.Pages[0].Regions) != 1 || a.Pages[0].Regions[0].X != 1 || a.Pages[0].Regions[0].Y != 2 {
		t.Fatalf("unknown keys must be ignored: %v %+v", err, a)
	}
}
