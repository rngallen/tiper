package config

import "testing"

func TestClampIdleMinutes(t *testing.T) {
	t.Parallel()
	if ClampIdleMinutes(0) != 10 {
		t.Fatal("zero must default to 10 minutes")
	}
	if ClampIdleMinutes(3) != 3 {
		t.Fatal("explicit minutes")
	}
	if ClampIdleMinutes(1) != 10 {
		t.Fatal("below 2 must default")
	}
	if ClampIdleMinutes(999) != 480 {
		t.Fatal("cap at 480")
	}
}

func TestClampIdleWarnSeconds(t *testing.T) {
	t.Parallel()
	if ClampIdleWarnSeconds(10, 0) != 120 {
		t.Fatal("default warn 120")
	}
	got := ClampIdleWarnSeconds(3, 600)
	if got != 165 {
		t.Fatalf("warn cannot consume the whole 3m window: got %d", got)
	}
}

func TestDefaultSession(t *testing.T) {
	t.Parallel()
	s := DefaultSession()
	if s.IdleMinutes != 10 || s.WarnSeconds != 120 {
		t.Fatalf("seed default: %+v", s)
	}
}

func TestSessionConfigWarnMinutes(t *testing.T) {
	t.Parallel()
	s := SessionConfig{IdleMinutes: 10, WarnSeconds: 120}.Clamp()
	if s.WarnMinutes() != 2 {
		t.Fatalf("120s → 2 minutes, got %d", s.WarnMinutes())
	}
}
