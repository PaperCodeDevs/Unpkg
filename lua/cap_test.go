package lua

import (
	"encoding/binary"
	"testing"
)

func TestReadCap_EmptyOrInvalid(t *testing.T) {
	_, err := ReadCap(nil)
	if err == nil {
		t.Fatalf("expected error on nil")
	}

	invalidMagic := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
	_, err = ReadCap(invalidMagic)
	if err == nil {
		t.Fatalf("expected error on invalid magic")
	}
}

func TestReadCap_ValidRecord(t *testing.T) {
	var buf []byte
	// 1. Magic
	m := make([]byte, 4)
	binary.LittleEndian.PutUint32(m, CapMagic)
	buf = append(buf, m...)

	// 2. Name: "[lj]test/hello.lua"
	name := "[lj]test/hello.lua"
	nlen := make([]byte, 4)
	binary.LittleEndian.PutUint32(nlen, uint32(len(name)))
	buf = append(buf, nlen...)
	buf = append(buf, []byte(name)...)

	// 3. Body: \x1bLJ\x90...
	body := []byte{0x1b, 'L', 'J', 0x90, 0x00}
	slen := make([]byte, 4)
	binary.LittleEndian.PutUint32(slen, uint32(len(body)))
	buf = append(buf, slen...)
	buf = append(buf, body...)

	recs, err := ReadCap(buf)
	if err != nil {
		t.Fatalf("ReadCap failed: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].Tag != "lj" {
		t.Errorf("expected tag 'lj', got %q", recs[0].Tag)
	}
	if recs[0].Name != "test/hello.lua" {
		t.Errorf("expected name 'test/hello.lua', got %q", recs[0].Name)
	}
	if len(recs[0].Body) != len(body) {
		t.Errorf("expected body len %d, got %d", len(body), len(recs[0].Body))
	}

	// 4. Test CapLastLJ deduplication
	dedup := CapLastLJ(recs)
	if len(dedup) != 1 {
		t.Fatalf("expected 1 deduplicated record, got %d", len(dedup))
	}
}

func TestRunTree_EmptyPath(t *testing.T) {
	_, err := RunTree("", "")
	if err == nil {
		t.Fatalf("expected error on empty srcRoot")
	}
}
