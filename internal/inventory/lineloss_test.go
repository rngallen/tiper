package inventory

import (
	"testing"

	"dfms/apps/models"

	"github.com/shopspring/decimal"
)

func TestAllocateLineLossSharesByOutturn(t *testing.T) {
	details := []models.ReceiptDetail{
		{Quantity: decimal.NewFromInt(600), CubicMeter: decimal.NewFromInt(600)},
		{Quantity: decimal.NewFromInt(400), CubicMeter: decimal.NewFromInt(400)},
	}
	AllocateLineLoss(details, []bool{false, false}, decimal.NewFromInt(-10), decimal.NewFromInt(-10), decimal.Zero)
	if !details[0].LineLoss.Equal(decimal.RequireFromString("-6")) {
		t.Fatalf("first share %s", details[0].LineLoss)
	}
	if !details[1].LineLoss.Equal(decimal.RequireFromString("-4")) {
		t.Fatalf("second share %s", details[1].LineLoss)
	}
	if !details[0].ReceivedQuantity().Equal(decimal.NewFromInt(594)) {
		t.Fatalf("received %s", details[0].ReceivedQuantity())
	}
}

func TestAllocateLineLossSkipsProration(t *testing.T) {
	details := []models.ReceiptDetail{
		{Quantity: decimal.NewFromInt(900), CubicMeter: decimal.NewFromInt(900)},
		{Quantity: decimal.NewFromInt(-100), CubicMeter: decimal.NewFromInt(-100)},
	}
	AllocateLineLoss(details, []bool{false, true}, decimal.NewFromInt(-8), decimal.NewFromInt(-8), decimal.Zero)
	if !details[0].LineLoss.Equal(decimal.NewFromInt(-8)) {
		t.Fatalf("billable parcel takes the loss: %s", details[0].LineLoss)
	}
	if !details[1].LineLoss.IsZero() {
		t.Fatalf("proration parcel must not receive line loss: %s", details[1].LineLoss)
	}
}

func TestApplyTankLineLoss(t *testing.T) {
	r := models.Receipt{
		TankQuantity:   decimal.NewFromInt(990),
		TankCubicMeter: decimal.NewFromInt(990),
		Details: []models.ReceiptDetail{
			{Quantity: decimal.NewFromInt(600), CubicMeter: decimal.NewFromInt(600)},
			{Quantity: decimal.NewFromInt(400), CubicMeter: decimal.NewFromInt(400)},
		},
	}
	ApplyTankLineLoss(&r, nil)
	if !r.LineLoss.Equal(decimal.NewFromInt(-10)) || !r.LineLossCubicMeter.Equal(decimal.NewFromInt(-10)) {
		t.Fatalf("tank − outturn: L=%s CM=%s", r.LineLoss, r.LineLossCubicMeter)
	}
}

func TestApplyTankLineLossExcludesProration(t *testing.T) {
	r := models.Receipt{
		TankQuantity:   decimal.NewFromInt(890),
		TankCubicMeter: decimal.NewFromInt(890),
		Details: []models.ReceiptDetail{
			{Quantity: decimal.NewFromInt(900), CubicMeter: decimal.NewFromInt(900)},
			{Quantity: decimal.NewFromInt(-100), CubicMeter: decimal.NewFromInt(-100)},
		},
	}
	ApplyTankLineLoss(&r, []bool{false, true})
	if !r.LineLoss.Equal(decimal.NewFromInt(-10)) || !r.LineLossCubicMeter.Equal(decimal.NewFromInt(-10)) {
		t.Fatalf("proration qty must not enter outturn: L=%s CM=%s", r.LineLoss, r.LineLossCubicMeter)
	}
}
