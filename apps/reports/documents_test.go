package reports

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFmtLThousands(t *testing.T) {
	if got := fmtL(decimal.NewFromInt(187500)); got != "187,500.00" {
		t.Fatalf("got %s", got)
	}
	if got := fmtL(decimal.RequireFromString("0.5")); got != "0.50" {
		t.Fatalf("got %s", got)
	}
}
