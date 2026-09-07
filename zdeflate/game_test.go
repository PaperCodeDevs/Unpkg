package zdeflate_test

import (
	"testing"
	"time"

	"github.com/PaperCodeDevs/Unpkg/zdeflate"
)

func TestGameFamilyMembers(t *testing.T) {
	mems, files := loadGameMembers()
	if len(mems) == 0 {
		t.Skip("no game-family UGC zip members under miniworddata110/data or .temp/rev-reslist2/samples")
	}
	by := map[string][2]int{}
	pass := 0
	var first string
	for _, m := range mems {
		ok := recompressOK(m)
		c := by[m.Kind]
		c[1]++
		if ok {
			pass++
			c[0]++
		} else if first == "" {
			first = m.Path + " " + m.Name
		}
		by[m.Kind] = c
	}
	t.Logf("files=%d members=%d pass=%d", files, len(mems), pass)
	for k, c := range by {
		t.Logf("%s %d/%d", k, c[0], c[1])
	}
	if files != 4976 || len(mems) != 9952 || pass != 9952 {
		t.Fatalf("files=%d members=%d pass=%d want 4976/9952/9952 first=%s", files, len(mems), pass, first)
	}
	want := map[string][2]int{
		"json": {4976, 4976}, "bin": {3773, 3773}, "script": {707, 707}, "trigger": {496, 496},
	}
	for k, w := range want {
		if by[k] != w {
			t.Fatalf("%s %d/%d want %d/%d", k, by[k][0], by[k][1], w[0], w[1])
		}
	}
}

func TestSpeed1MiB(t *testing.T) {
	src := make([]byte, 1<<20)
	for i := range src {
		src[i] = byte(i * 3)
	}
	start := time.Now()
	got := zdeflate.Compress(src)
	d := time.Since(start)
	if len(got) == 0 {
		t.Fatal("empty")
	}
	if d > 2*time.Second {
		t.Fatalf("1MiB %s", d)
	}
	t.Logf("1MiB %s", d)
}
