package billing

import (
	"testing"

	"dfms/apps/models"
	"dfms/internal/catalogs"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
)

func TestComputeVariableFee_SpreadsheetAJ(t *testing.T) {
	usdCM, usdMT, tzsCM, tzsMT := ComputeVariableFee(
		decimal.NewFromInt(2600),
		decimal.NewFromInt(2600),
		decimal.NewFromFloat(0.84),
		decimal.NewFromFloat(0.005),
	)
	if !usdCM.Equal(decimal.RequireFromString("5.00")) {
		t.Fatalf("usdCM=%s", usdCM)
	}
	if !usdMT.Equal(decimal.RequireFromString("4.20")) {
		t.Fatalf("usdMT=%s", usdMT)
	}
	if !tzsCM.Equal(decimal.RequireFromString("13000.00")) {
		t.Fatalf("tzsCM=%s", tzsCM)
	}
	if !tzsMT.Equal(decimal.RequireFromString("10920.00")) {
		t.Fatalf("tzsMT=%s", tzsMT)
	}
}

func TestHomePriceUSDToTZS(t *testing.T) {
	got := HomePrice(decimal.NewFromInt(2), "USD", "TZS", decimal.NewFromInt(2600))
	if !got.Equal(decimal.NewFromInt(5200)) {
		t.Fatalf("got %s", got)
	}
	back := HomePrice(decimal.NewFromInt(5200), "TZS", "USD", decimal.NewFromInt(2600))
	if !back.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("got %s", back)
	}
	same := HomePrice(decimal.NewFromInt(3), "USD", "USD", decimal.NewFromInt(2600))
	if !same.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("same currency: %s", same)
	}
}

func TestMatchFcfLinePrefersCollectionAndPeriod(t *testing.T) {
	lines := []models.FcfFee{
		{ClassOfTrade: "SRT", DischargeRoute: types.RouteSBM, FirstSourcePrice: decimal.NewFromInt(1), FirstUnit: "M3", FirstRateKind: types.RateFlat},
		{ClassOfTrade: "SRT", DischargeRoute: types.RouteSBM, CollectionMethod: types.CollectionPumpOver, FirstDays: 15, FirstSourcePrice: decimal.RequireFromString("3.38"), FirstUnit: "MT", FirstRateKind: types.RateFlat},
	}
	r := &models.Receipt{TenderCode: types.TenderSRT, RouteCode: types.RouteSBM}
	d := models.ReceiptDetail{CollectionMethod: types.CollectionPumpOver, NextBillingDays: 15}
	got := matchFcfLine(catalogs.Set{Pricing: map[string]models.PricingNature{}}, lines, r, d)
	if got == nil || !got.FirstSourcePrice.Equal(decimal.RequireFromString("3.38")) {
		t.Fatalf("want 3.38 MT line, got %+v", got)
	}
}

func TestCycleDaysFromNextBilling(t *testing.T) {
	if cycleDays(models.ReceiptDetail{NextBillingDays: 30, CollectionMethod: types.CollectionPumpOver}) != 30 {
		t.Fatal("explicit next billing should win")
	}
	if cycleDays(models.ReceiptDetail{CollectionMethod: types.CollectionLoading}) != 15 {
		t.Fatal("collection must not encode days")
	}
	if cycleDays(models.ReceiptDetail{NextBillingDays: 45, CollectionMethod: types.CollectionLoading}) != 45 {
		t.Fatal("45-day next billing is independent of collection")
	}
}

func TestDetailBillsKoj(t *testing.T) {
	koj := &models.Receipt{ReceiptType: types.ReceiptExternal, RouteCode: types.RouteKOJ}
	if detailBillsKoj(koj, models.ReceiptDetail{}) {
		t.Fatal("external KOJ without 10-inch pipeline must not bill")
	}
	piped := &models.Receipt{ReceiptType: types.ReceiptExternal, RouteCode: types.RouteKOJ, UsesTiperPipeline: true}
	if !detailBillsKoj(piped, models.ReceiptDetail{}) {
		t.Fatal("external KOJ on 10-inch pipeline bills KOJ fee")
	}
	if detailBillsKoj(&models.Receipt{ReceiptType: types.ReceiptInternal, RouteCode: types.RouteKOJ, UsesTiperPipeline: true},
		models.ReceiptDetail{}) {
		t.Fatal("internal receipts use FCF, not KOJ fee")
	}
	if detailBillsKoj(&models.Receipt{ReceiptType: types.ReceiptExternal, RouteCode: types.RouteSBM, UsesTiperPipeline: true},
		models.ReceiptDetail{}) {
		t.Fatal("SBM never bills KOJ fee")
	}
}

func TestQtyForUnit(t *testing.T) {
	d := models.ReceiptDetail{
		Quantity:    decimal.NewFromInt(1000),
		CubicMeter:  decimal.NewFromInt(1),
		MetricTonne: decimal.RequireFromString("0.84"),
	}
	if !qtyForUnit(d, "MT").Equal(d.MetricTonne) {
		t.Fatal("MT")
	}
	if !qtyForUnit(d, "L").Equal(d.Quantity) {
		t.Fatal("L")
	}
	if !qtyForUnit(d, "CM").Equal(d.CubicMeter) {
		t.Fatal("CM alias")
	}
	d.LineLoss = decimal.NewFromInt(-50)
	if !qtyForUnit(d, "L").Equal(decimal.NewFromInt(950)) {
		t.Fatal("first billing uses received litres")
	}
}

func TestPickTierWholeVolume(t *testing.T) {
	to4999 := decimal.NewFromInt(4999)
	to9999 := decimal.NewFromInt(9999)
	to14999 := decimal.NewFromInt(14999)
	tiers := []models.FcfFeeTier{
		{Phase: "first", FromQty: decimal.Zero, ToQty: &to4999, SourcePrice: decimal.NewFromInt(7)},
		{Phase: "first", FromQty: decimal.NewFromInt(5000), ToQty: &to9999, SourcePrice: decimal.NewFromInt(6)},
		{Phase: "first", FromQty: decimal.NewFromInt(10000), ToQty: &to14999, SourcePrice: decimal.RequireFromString("5.5")},
		{Phase: "first", FromQty: decimal.NewFromInt(15000), SourcePrice: decimal.NewFromInt(5)},
		{Phase: "nth", FromQty: decimal.Zero, SourcePrice: decimal.RequireFromString("4.7")},
	}
	got := pickTier(tiers, "first", decimal.NewFromInt(12000))
	if got == nil || !got.SourcePrice.Equal(decimal.RequireFromString("5.5")) {
		t.Fatalf("12000 m³ should hit 10k–15k slab, got %+v", got)
	}
	open := pickTier(tiers, "first", decimal.NewFromInt(20000))
	if open == nil || !open.SourcePrice.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("20000 should hit open slab, got %+v", open)
	}
	nth := pickTier(tiers, "nth", decimal.NewFromInt(100))
	if nth == nil || !nth.SourcePrice.Equal(decimal.RequireFromString("4.7")) {
		t.Fatalf("nth tier %+v", nth)
	}
}

func TestApplyFcfPhaseNthUsesNthRate(t *testing.T) {
	line := &models.FcfFee{
		FirstRateKind: types.RateFlat, FirstUnit: "M3",
		FirstSourceCurrencyCode: "USD", FirstSourcePrice: decimal.RequireFromString("4.7"),
		NthRateKind: types.RateFlat, NthUnit: "M3",
		NthSourceCurrencyCode: "USD", NthSourcePrice: decimal.RequireFromString("7.5"),
		NthDays: 30,
	}
	d := models.ReceiptDetail{CubicMeter: decimal.NewFromInt(10)}
	var first, nth models.BillingRun
	if err := applyFcfPhase(nil, nil, d, line, "first", &first); err != nil {
		t.Fatal(err)
	}
	if !first.Rate.Equal(decimal.RequireFromString("4.7")) {
		t.Fatalf("first rate %s", first.Rate)
	}
	if err := applyFcfPhase(nil, nil, d, line, "nth", &nth); err != nil {
		t.Fatal(err)
	}
	if !nth.Rate.Equal(decimal.RequireFromString("7.5")) {
		t.Fatalf("nth must re-price, got %s", nth.Rate)
	}
}
