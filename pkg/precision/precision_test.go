package precision

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestClampPlaces(t *testing.T) {
	if ClampPlaces(-1) != 0 || ClampPlaces(9) != 6 || ClampPlaces(3) != 3 {
		t.Fatal("clamp")
	}
}

func TestRound(t *testing.T) {
	got := Round(decimal.RequireFromString("1.23456"), 3)
	if !got.Equal(decimal.RequireFromString("1.235")) {
		t.Fatalf("got %s", got)
	}
}
