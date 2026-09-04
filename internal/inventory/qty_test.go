package inventory

import (
	"testing"

	"dfms/pkg/precision"

	"github.com/shopspring/decimal"
)

func TestQtyFromLitres(t *testing.T) {
	cm, mt := QtyFromLitres(decimal.NewFromInt(1000), decimal.RequireFromString("0.84"))
	if !cm.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("m³=%s", cm)
	}
	if !mt.Equal(decimal.RequireFromString("0.84")) {
		t.Fatalf("MT=%s", mt)
	}
}

func TestQtyFromLitresUsesCubicPlaces(t *testing.T) {
	cm, _ := QtyFromLitresRounded(
		decimal.RequireFromString("1234"),
		decimal.RequireFromString("0.84"),
		precision.Settings{CubicMeter: 2, MetricTonne: 3},
	)
	if !cm.Equal(decimal.RequireFromString("1.23")) {
		t.Fatalf("m³ places: %s", cm)
	}
}
