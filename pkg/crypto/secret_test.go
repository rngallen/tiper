package crypto

import "testing"

func TestSealOpenRoundTrip(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	sealed, err := Seal("super-secret", key)
	if err != nil {
		t.Fatal(err)
	}
	if !IsSealed(sealed) {
		t.Fatalf("expected sealed prefix, got %q", sealed)
	}
	plain, err := Open(sealed, key)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "super-secret" {
		t.Fatalf("got %q", plain)
	}
}
