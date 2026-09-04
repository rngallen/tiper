package inventory

import (
	"testing"
	"time"
)

func TestDayUTC_UsesEastAfricaCalendar(t *testing.T) {
	eat := time.FixedZone("EAT", 3*3600)
	in := time.Date(2026, 8, 23, 1, 15, 0, 0, eat)
	got := dayUTC(in)
	want := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("dayUTC(%v) = %v, want %v", in, got, want)
	}
	utcPrev := time.Date(2026, 8, 22, 22, 15, 0, 0, time.UTC) // 01:15 EAT next day
	if !dayUTC(utcPrev).Equal(want) {
		t.Fatalf("dayUTC of 22:15 UTC should be 23 Aug EAT, got %v", dayUTC(utcPrev))
	}
}
