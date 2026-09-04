package inventory

import (
	"testing"

	"dfms/apps/models"

	"github.com/shopspring/decimal"
)

func TestScaleMovedAndReduceParcel(t *testing.T) {
	src := models.ReceiptDetail{
		Quantity:            decimal.NewFromInt(10000),
		CubicMeter:          decimal.NewFromInt(10),
		MetricTonne:         decimal.RequireFromString("8.4"),
		LineLoss:            decimal.NewFromInt(-100),
		LineLossCubicMeter:  decimal.RequireFromString("-0.1"),
		LineLossMetricTonne: decimal.RequireFromString("-0.084"),
		HoldQuantity:        decimal.NewFromInt(2000),
	}
	take, total := decimal.NewFromInt(4), decimal.NewFromInt(10)
	moved := scaleMoved(src, take, total)
	if !moved.CubicMeter.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("moved m³ %s", moved.CubicMeter)
	}
	if !moved.Quantity.Equal(decimal.NewFromInt(4000)) {
		t.Fatalf("moved L %s", moved.Quantity)
	}
	if !moved.MetricTonne.Equal(decimal.RequireFromString("3.360")) {
		t.Fatalf("moved MT %s", moved.MetricTonne)
	}
	reduceParcel(&src, take, total)
	if !src.CubicMeter.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("left m³ %s", src.CubicMeter)
	}
	if !src.Quantity.Equal(decimal.NewFromInt(6000)) {
		t.Fatalf("left L %s", src.Quantity)
	}
	if src.ID != 0 && moved.ID != 0 {
		t.Fatal("moved must not keep source id")
	}
}

func TestIttCubicPrefersCubicMeter(t *testing.T) {
	itt := &models.IttTransfer{Quantity: decimal.NewFromInt(7), CubicMeter: decimal.NewFromInt(4)}
	if !ittCubic(itt).Equal(decimal.NewFromInt(4)) {
		t.Fatal(ittCubic(itt))
	}
}
