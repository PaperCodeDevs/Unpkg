package spine

import "testing"

func TestParseSkeleton38(t *testing.T) {
	raw := skeleton38()
	if h, v, err := Probe(raw); err != nil || v != "3.8.99" || h != "NNl6b7LWQCOjpT/S6FC56pkhPbQ" {
		t.Fatalf("probe %q %q %v", h, v, err)
	}
	sk, err := ParseSkeleton(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sk.X != -10 || sk.Y != -20 || sk.Width != 100 || sk.Height != 200 || len(sk.Bones) != 2 || len(sk.Slots) != 1 {
		t.Fatalf("header %+v", sk)
	}
	b := sk.Bones[1]
	if b.Name != "child" || b.Parent != 0 || b.Rotation != 45 || b.X != 3 || b.Length != 50 || b.TransformMode != 1 || !b.SkinRequired || sk.Bones[0].Parent != -1 {
		t.Fatalf("bone %+v", b)
	}
	s := sk.Slots[0]
	if s.Name != "slot" || s.Bone != 1 || s.Color != 0xFFFFFFFF || s.HasDark || s.Attachment != "img" {
		t.Fatalf("slot %+v", s)
	}
	if len(sk.Skins) != 1 || sk.Skins[0].Name != "default" || len(sk.Skins[0].Attachments) != 1 {
		t.Fatalf("skins %+v", sk.Skins)
	}
	a := sk.Skins[0].Attachments[0]
	if a.Type != AttRegion || a.Name != "img" || a.Slot != 0 || a.X != 1 || a.Y != 2 || a.Width != 64 || a.Height != 32 || a.Color != 0xFFFFFFFF {
		t.Fatalf("attachment %+v", a)
	}
	if len(sk.Events) != 1 || sk.Events[0].Name != "img" || sk.Events[0].Int != -3 || sk.Events[0].Float != 1.5 || sk.Events[0].String != "hi" || sk.Events[0].HasAudio {
		t.Fatalf("events %+v", sk.Events)
	}
	if len(sk.Animations) != 1 || sk.Animations[0].Name != "idle" || sk.Animations[0].Duration != 1 || len(sk.Animations[0].Timelines) != 3 {
		t.Fatalf("animations %+v", sk.Animations)
	}
	tl := sk.Animations[0].Timelines
	if tl[0].Kind != "attachment" || tl[0].Target != 0 || tl[0].Frames != 2 || tl[1].Kind != "rotate" || tl[1].Target != 1 || tl[1].Times[1] != 1 || tl[2].Kind != "event" || tl[2].Times[0] != 0.25 {
		t.Fatalf("timelines %+v", tl)
	}
}

func TestParseSkeleton37(t *testing.T) {
	sk, err := ParseSkeleton(skeleton37())
	if err != nil {
		t.Fatal(err)
	}
	if sk.Version != "3.7.94" || sk.Width != 10 || sk.Height != 20 || sk.FPS != 30 || sk.ImagesPath != "./images/" || sk.AudioPath != "./audio/" {
		t.Fatalf("header %+v", sk)
	}
	if len(sk.Bones) != 1 || sk.Bones[0].Color != 0x11223344 || sk.Bones[0].SkinRequired {
		t.Fatalf("bones %+v", sk.Bones)
	}
	s := sk.Slots[0]
	if !s.HasDark || s.DarkColor != 0x00FF00FF || s.Attachment != "a" || s.BlendMode != 2 {
		t.Fatalf("slot %+v", s)
	}
	a := sk.Skins[0].Attachments[0]
	if a.Type != AttBoundingBox || a.Name != "a" || a.Vertices.Count != 3 || a.Vertices.Weighted || len(a.Vertices.Values) != 6 || a.Color != 0xFFFFFFFF {
		t.Fatalf("attachment %+v", a)
	}
}

func TestParseSkeleton40(t *testing.T) {
	raw := skeleton40()
	if h, v, err := Probe(raw); err != nil || v != "4.0.64" || h != "12345" {
		t.Fatalf("probe %q %q %v", h, v, err)
	}
	sk, err := ParseSkeleton(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sk.Hash != "12345" || len(sk.Bones) != 1 || len(sk.Skins) != 0 || len(sk.Animations) != 1 {
		t.Fatalf("skeleton %+v", sk)
	}
	a := sk.Animations[0]
	if a.Name != "run" || a.Duration != 1.25 || len(a.Timelines) != 1 || a.Timelines[0].Kind != "translate" || a.Timelines[0].Frames != 3 || a.Timelines[0].Times[1] != 0.5 {
		t.Fatalf("animation %+v", a)
	}
}

func TestSkeletonErrors(t *testing.T) {
	if _, _, err := Probe([]byte("\x05abcd\x03xy")); err == nil {
		t.Fatal("garbage probe")
	}
	old := &bin{}
	old.str("h")
	old.str("2.1.27")
	if _, err := ParseSkeleton(old.b); err == nil {
		t.Fatal("unsupported version")
	}
	raw := skeleton38()
	if _, err := ParseSkeleton(raw[:len(raw)-5]); err == nil {
		t.Fatal("truncated")
	}
	if _, err := ParseSkeleton(append(append([]byte(nil), raw...), 0)); err == nil {
		t.Fatal("trailing")
	}
}
