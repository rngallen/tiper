package inventory

import (
	"context"
	"testing"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
)

func TestReceiptInputValidate_InternalRequired(t *testing.T) {
	in := receiptInput{ReceiptType: "internal"}
	in.Sanitize()
	if err := in.Validate(context.Background()); err == nil {
		t.Fatal("expected missing header fields")
	}
	in = receiptInput{
		Date:                  time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselDate:            time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselID:              "01VESSEL",
		ProductID:             "01PRODUCT",
		SupplierID:            "01SUPPLIER",
		RouteCode:             "sbm",
		TenderCode:            "srt",
		ProcurementMethodCode: "bps",
		ReceiptType:           "internal",
		Density:               "0.84",
		Notes:                 "  discharge complete  ",
	}
	in.Sanitize()
	if in.RouteCode != "SBM" || in.Notes != "discharge complete" {
		t.Fatalf("sanitize: %+v", in)
	}
	if err := in.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReceiptInputValidate_DensityScale(t *testing.T) {
	base := receiptInput{
		Date:                  time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselDate:            time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselID:              "01VESSEL",
		ProductID:             "01PRODUCT",
		SupplierID:            "01SUPPLIER",
		RouteCode:             "SBM",
		TenderCode:            "SRT",
		ProcurementMethodCode: "BPS",
		ReceiptType:           "internal",
		Notes:                 "ok",
	}
	base.Density = "840"
	if err := base.Validate(context.Background()); err == nil {
		t.Fatal("expected kg/m³ density rejected")
	}
	base.Density = "0"
	if err := base.Validate(context.Background()); err == nil {
		t.Fatal("expected zero density rejected")
	}
	base.Density = "0.84"
	if err := base.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReceiptInputValidate_ExternalSkipsCommercial(t *testing.T) {
	in := receiptInput{
		Date:        time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselDate:  time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselID:    "01VESSEL",
		ProductID:   "01PRODUCT",
		RouteCode:   "KOJ",
		ReceiptType: "external",
		Notes:       "third-party discharge",
	}
	if err := in.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAssertReceiptHeader_Internal(t *testing.T) {
	sid := uint(9)
	row := &models.Receipt{
		Date:                  time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselDate:            time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		VesselID:              1,
		ProductID:             2,
		SupplierID:            &sid,
		RouteCode:             "SBM",
		TenderCode:            "SRT",
		ProcurementMethodCode: "BPS",
		ReceiptType:           types.ReceiptInternal,
		Density:               decimal.RequireFromString("0.84"),
		Notes:                 "ok",
	}
	if err := assertReceiptHeader(row); err != nil {
		t.Fatal(err)
	}
	row.SupplierID = nil
	row.Notes = ""
	if err := assertReceiptHeader(row); err == nil {
		t.Fatal("expected supplier and notes")
	}
}

func TestHoldReleaseInputValidate(t *testing.T) {
	in := holdReleaseInput{}
	if err := in.Validate(context.Background()); err == nil {
		t.Fatal("expected required fields")
	}
	in = holdReleaseInput{
		ReleaseDate: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Description: "Invoice paid",
		Lines:       []holdLineInput{{CustomerID: "01", Quantity: "1000"}},
	}
	if err := in.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}
