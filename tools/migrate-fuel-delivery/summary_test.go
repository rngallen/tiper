package main

import (
	"testing"

	"dfms/apps/models"

	"github.com/shopspring/decimal"
)

func TestAggregateLoadingSummariesSplitsRequestedAcrossGrades(t *testing.T) {
	qty := decimal.NewFromInt(100)
	loads := []models.GantryLoading{{
		ID: 1, Year: 2026, Month: 8, RequestedQty: qty, StockStatusID: 2,
	}}
	prods := []models.GantryLoadingProduct{
		{LoadingID: 1, ProductID: 10, ProductCode: "AGO", StandardVolume: decimal.NewFromInt(60), Weight: decimal.NewFromInt(50)},
		{LoadingID: 1, ProductID: 11, ProductCode: "PMS", StandardVolume: decimal.NewFromInt(40), Weight: decimal.NewFromInt(33)},
	}
	rows := aggregateLoadingSummaries(loads, prods, map[uint]bool{2: false})
	if len(rows) != 2 {
		t.Fatalf("want 2 summary rows, got %d", len(rows))
	}
	byCode := map[string]models.GantryLoadingSummary{}
	for _, r := range rows {
		byCode[r.ProductCode] = r
	}
	ago := byCode["AGO"]
	if !ago.LocalLoaded.Equal(decimal.NewFromInt(60)) || !ago.LocalRequested.Equal(decimal.NewFromInt(60)) {
		t.Fatalf("AGO local %s / %s", ago.LocalLoaded, ago.LocalRequested)
	}
	pms := byCode["PMS"]
	if !pms.LocalLoaded.Equal(decimal.NewFromInt(40)) || !pms.LocalRequested.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("PMS local %s / %s", pms.LocalLoaded, pms.LocalRequested)
	}
}

func TestAggregateLoadingSummariesTransitBucket(t *testing.T) {
	loads := []models.GantryLoading{{
		ID: 1, Year: 2026, Month: 1, RequestedQty: decimal.NewFromInt(10), StockStatusID: 9,
	}}
	prods := []models.GantryLoadingProduct{
		{LoadingID: 1, ProductID: 10, ProductCode: "AGO", StandardVolume: decimal.NewFromInt(10), Weight: decimal.NewFromInt(8)},
	}
	rows := aggregateLoadingSummaries(loads, prods, map[uint]bool{9: true})
	if len(rows) != 1 {
		t.Fatalf("got %d", len(rows))
	}
	if !rows[0].TransitLoaded.Equal(decimal.NewFromInt(10)) || !rows[0].LocalLoaded.IsZero() {
		t.Fatalf("transit bucket: %+v", rows[0])
	}
}
