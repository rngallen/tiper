package docsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"dfms/apps/auth/middleware"
	"dfms/pkg/config"
)

func setMasterKey(t *testing.T, master []byte) {
	t.Helper()
	middleware.SymmetricKey = append([]byte(nil), master...)
	config.Conf = config.Config{App: config.AppConfig{
		SymmetricKey: hex.EncodeToString(master),
	}}
}

func TestSignValidRoundTrip(t *testing.T) {
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i + 1)
	}
	setMasterKey(t, master)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	uid := "01KZGSYS09BR5DX2JWTWDCJKDB"
	sig := Sign(KindILR, uid)
	if sig == "" {
		t.Fatal("empty sig")
	}
	if !Valid(KindILR, uid, sig) {
		t.Fatal("expected valid")
	}
	if Valid(KindILR, uid, sig+"x") {
		t.Fatal("tampered sig must fail")
	}
	if Valid(KindITT, uid, sig) {
		t.Fatal("wrong kind must fail")
	}
	if Valid(KindILR, "other-uid", sig) {
		t.Fatal("wrong uid must fail")
	}
}

func TestKeyIsPurposeBound(t *testing.T) {
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i + 3)
	}
	setMasterKey(t, master)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	mac := hmac.New(sha256.New, master)
	mac.Write([]byte(purpose))
	derived := mac.Sum(nil)

	mu.RLock()
	got := append([]byte(nil), key...)
	mu.RUnlock()
	if !hmac.Equal(got, derived) {
		t.Fatal("derived key mismatch")
	}
	if hmac.Equal(got, master) {
		t.Fatal("must not use raw symmetric key")
	}
}

func TestPath(t *testing.T) {
	master := make([]byte, 32)
	setMasterKey(t, master)
	_ = Init()
	uid := "01TESTUID00000000000000000"
	p := Path(KindReceipt, uid)
	sig := Sign(KindReceipt, uid)
	want := "/verify/document/receipt/" + uid + "/" + sig
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}
