package orders

import (
	"testing"
	"time"

	"dfms/pkg/types"
)

func TestIloExpireBefore(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 5, 0, 0, time.UTC)
	got := IloExpireBefore(now)
	want := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("cutoff=%v want %v", got, want)
	}
	expOnDay := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	if !expOnDay.Before(got) {
		t.Fatal("ILO expiring today must be selected at midnight")
	}
	tomorrow := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	if tomorrow.Before(got) {
		t.Fatal("ILO expiring tomorrow must stay open")
	}
}

func TestOrderExpiredValid(t *testing.T) {
	if !types.OrderExpired.Valid() {
		t.Fatal("expired must be a valid order status")
	}
}

func TestExpireIlosNoDB(t *testing.T) {
	s := &Service{}
	n, err := s.ExpireIlos(time.Now())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
