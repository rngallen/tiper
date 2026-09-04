package main

import "testing"

func TestPickPhonePrefersPhoneThenMobile(t *testing.T) {
	if got := pickPhone(" +255111 ", " +255222 "); got != "+255111" {
		t.Fatalf("phone should win: %q", got)
	}
	if got := pickPhone("  ", "+255222"); got != "+255222" {
		t.Fatalf("mobile fallback: %q", got)
	}
	if got := pickPhone("", ""); got != "" {
		t.Fatalf("both empty should stay empty: %q", got)
	}
}
