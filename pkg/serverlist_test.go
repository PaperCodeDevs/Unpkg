package pkg

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const serverListTestKey = "@$#^!1345^&()"

func TestServerListKnownAnswer(t *testing.T) {
	key := []byte(serverListTestKey)
	plain := []byte("<Config>\r\n    <!")
	want, _ := hex.DecodeString("5a12be34c018c3154efe822ba3d3b238")
	got, err := EncryptServerList(plain, key)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("encrypt=%x err=%v", got, err)
	}
	back, err := DecryptServerList(want, key)
	if err != nil || !bytes.Equal(back, plain) {
		t.Fatalf("decrypt=%q err=%v", back, err)
	}
	padded, _ := EncryptServerList([]byte("<Config>\n<a/>"), key)
	if len(padded) != 16 {
		t.Fatalf("padded len=%d", len(padded))
	}
	back, _ = DecryptServerList(padded, key)
	if string(back) != "<Config>\n<a/>" {
		t.Fatalf("round trip=%q", back)
	}
	if _, err := DecryptServerList(want, nil); err != ErrServerListKey {
		t.Fatalf("no key err=%v", err)
	}
	if _, err := DecryptServerList(want, make([]byte, 17)); err != ErrServerListKey {
		t.Fatalf("long key err=%v", err)
	}
	single, _ := EncryptServerList(plain, key[:8])
	triple, _ := EncryptServerList(plain, append(key[:8:8], make([]byte, 8)...))
	if bytes.Equal(single, triple) {
		t.Fatal("8 字节密钥应走单 DES")
	}
	if back, _ = DecryptServerList(single, key[:8]); !bytes.Equal(back, plain) {
		t.Fatalf("single des round trip=%q", back)
	}
	if _, err := DecryptServerList(want[:7], key); err == nil {
		t.Fatal("expected length error")
	}
}

func TestServerListRealPkg(t *testing.T) {
	p := filepath.Join(os.Getenv("APPDATA"), "miniworldgameguan110", "first_res.pkg")
	if _, err := os.Stat(p); err != nil {
		t.Skip("no first_res.pkg")
	}
	rd, err := OpenOverlayFiles(p, "")
	if err != nil {
		t.Fatal(err)
	}
	key := []byte(serverListTestKey)
	seen, sameAsXML := 0, 0
	for _, n := range rd.Names("") {
		if !strings.HasSuffix(n, ".data") || !strings.Contains(n, "serverlist") {
			continue
		}
		body, err := rd.Lookup(n)
		if err != nil {
			t.Fatal(err)
		}
		plain, err := DecryptServerList(body, key)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if !bytes.HasPrefix(plain, []byte("<Config>")) {
			t.Fatalf("%s: head=%q", n, plain[:minInt(len(plain), 16)])
		}
		if _, err := walkXML(plain); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		again, _ := EncryptServerList(plain, key)
		if !bytes.Equal(again, body) {
			t.Fatalf("%s: re-encrypt mismatch", n)
		}
		seen++
		if xmlBody, err := rd.Lookup(strings.TrimSuffix(n, ".data") + ".xml"); err == nil {
			lf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
			if bytes.Equal(lf(plain), lf(xmlBody)) {
				sameAsXML++
			}
		}
	}
	if seen == 0 || sameAsXML == 0 {
		t.Fatalf("seen=%d sameAsXML=%d", seen, sameAsXML)
	}
	t.Logf("serverlist.data=%d sameAsXML=%d", seen, sameAsXML)
}
