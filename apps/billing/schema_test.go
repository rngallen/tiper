package billing

import (
	"context"
	"strings"
	"testing"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestRateQuoteNo(t *testing.T) {
	t.Parallel()
	on := time.Date(2026, 9, 2, 15, 4, 0, 0, time.UTC)
	if got := rateQuoteNo("usd", "tzs", on); got != "USD/TZS · 2026-09-02" {
		t.Fatalf("got %q", got)
	}
}

func TestFcfBatchHeaderOnly(t *testing.T) {
	ctx := context.Background()
	s := fcfBatchSchema{Date: "2026-08-01", ExchangeRate: "2600"}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("header-only batch should be valid: %v", err)
	}
	if s.EffectiveFrom != "2026-08-01" {
		t.Fatalf("effective from default %s", s.EffectiveFrom)
	}
	s.Lines = []fcfLineSchema{{
		ClassOfTrade: "nonsrt", ProcurementMethod: "bps", DischargeRoute: "sbm",
		CollectionMethod: "pumpover", FirstDays: 30, NthDays: 30,
		FirstUnit: "cm", NthUnit: "cm",
		FirstSourceCurrencyCode: "usd", FirstSourcePrice: "3.38", FirstChargeTo: "customer",
		NthSourceCurrencyCode: "usd", NthSourcePrice: "3.38",
	}}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("line rejected: %v", err)
	}
	if s.Lines[0].ClassOfTrade != string(types.TenderNonSRT) || s.Lines[0].FirstUnit != "M3" {
		t.Fatalf("sanitize: %+v", s.Lines[0])
	}
	s.Lines[0].CollectionMethod = "LOADING45"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("LOADING45 is not a collection method")
	}
}

func TestFcfTierLineRequiresSlabs(t *testing.T) {
	ctx := context.Background()
	s := fcfBatchSchema{Date: "2026-08-01", ExchangeRate: "2600"}
	s.Lines = []fcfLineSchema{{
		ClassOfTrade: "SRT", ProcurementMethod: "BPS", DischargeRoute: "SBM",
		FirstRateKind: "tier", NthRateKind: "flat",
		FirstUnit: "M3", NthUnit: "M3",
		FirstSourceCurrencyCode: "USD", NthSourceCurrencyCode: "USD",
		NthSourcePrice: "7.5",
		Tiers: []fcfTierSchema{
			{Phase: "first", FromQty: "0", ToQty: "4,999", SourcePrice: "7"},
			{Phase: "first", FromQty: "15000", SourcePrice: "5"},
		},
	}}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("tiered line rejected: %v", err)
	}
	s.Lines[0].Tiers = nil
	if err := s.Validate(ctx); err == nil {
		t.Fatal("tier kind without slabs must fail")
	}
}

func TestPriceBatchHeaderOnly(t *testing.T) {
	ctx := context.Background()
	s := priceBatchSchema{Date: "2026-08-01"}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("header-only should pass: %v", err)
	}
	s.Fees = []priceLineSchema{{ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Unit: "L", SourceCurrencyCode: "USD", SourcePrice: "1.25"}}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	s.Fees[0].SourcePrice = "0"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("zero amount must fail")
	}
	s.Fees = []priceLineSchema{
		{ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Unit: "L", SourceCurrencyCode: "USD", SourcePrice: "1.25"},
		{ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Unit: "L", SourceCurrencyCode: "USD", SourcePrice: "2.00"},
	}
	s.Sanitize()
	if err := s.Validate(ctx); err == nil {
		t.Fatal("duplicate product, unit, and currency must fail")
	}
}

func TestVariableBatchHeaderOnly(t *testing.T) {
	ctx := context.Background()
	s := variableBatchSchema{Date: "2026-08-01", EffectiveFrom: "2026-08-01", ExchangeRate: "2600"}
	s.Sanitize()
	if err := s.Validate(ctx); err == nil {
		t.Fatal("variable fee without an MI-loss batch must fail")
	}
	s.MiLossBatchID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("header-only VCF should pass: %v", err)
	}
	s.Products = []varProductSchema{{
		ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", EwuraPrice: "2600", Density: "0.84",
	}}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("valid VCF rejected: %v", err)
	}
	s.Products[0].Density = "840"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("kg/m³ density must fail")
	}
}

func TestMiLossEffectiveOnOrBefore(t *testing.T) {
	feeEff := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	same := models.MiLossBatch{
		DocumentNumber: "MIL-1",
		Date:           time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		EffectiveFrom:  feeEff,
	}
	if err := miLossEffectiveOnOrBefore(same, feeEff); err != nil {
		t.Fatalf("same effective day: %v", err)
	}
	prior := models.MiLossBatch{
		DocumentNumber: "MIL-1",
		Date:           time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		EffectiveFrom:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	if err := miLossEffectiveOnOrBefore(prior, feeEff); err != nil {
		t.Fatalf("earlier effective date: %v", err)
	}
	// Created after the variable fee document date is still valid if already effective.
	createdLater := models.MiLossBatch{
		DocumentNumber: "MIL-2",
		Date:           time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC),
		EffectiveFrom:  time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := miLossEffectiveOnOrBefore(createdLater, feeEff); err != nil {
		t.Fatalf("later batch date must still pass on effective date: %v", err)
	}
	later := models.MiLossBatch{
		DocumentNumber: "MIL-3",
		Date:           time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		EffectiveFrom:  time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
	}
	if err := miLossEffectiveOnOrBefore(later, feeEff); err == nil {
		t.Fatal("later MI-loss effective date must fail")
	}
	viaDate := models.MiLossBatch{
		DocumentNumber: "MIL-4",
		Date:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := miLossEffectiveOnOrBefore(viaDate, feeEff); err != nil {
		t.Fatalf("date fallback: %v", err)
	}
}

func TestMiLossBatchLines(t *testing.T) {
	ctx := context.Background()
	s := miLossBatchSchema{Date: "2026-08-01", Description: "August card"}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("header-only MI loss should pass: %v", err)
	}
	if s.EffectiveFrom != "2026-08-01" {
		t.Fatalf("effective from %s", s.EffectiveFrom)
	}
	s.Lines = []miLossLineSchema{{
		ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", ContractTypeCode: "adhoc", Value: "0.005",
	}}
	s.Products = nil
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("valid MI loss rejected: %v", err)
	}
	if s.Products[0].Rates[0].ContractTypeCode != string(types.ContractAdhoc) {
		t.Fatalf("contract: %s", s.Products[0].Rates[0].ContractTypeCode)
	}
	s.Products[0].Rates[0].Value = "1.5"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("MI-loss above 100% must fail")
	}
	s.Products[0].Rates[0].Value = "1"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("MI-loss of 1 (100%) must fail")
	}
	s.Products[0].Rates[0].Value = "0"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("zero MI-loss must fail")
	}
	s.Products = nil
	s.Lines[0].Value = "0.005"
	s.Lines = append(s.Lines, miLossLineSchema{
		ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", ContractTypeCode: "ADHOC", Value: "0.002",
	})
	s.Sanitize()
	if err := s.Validate(ctx); err == nil {
		t.Fatal("duplicate product and contract must fail")
	}
	s.Products = nil
	s.Lines[1].ContractTypeCode = "SRT"
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("same product with a different contract should pass: %v", err)
	}
	if len(s.Products) != 1 || len(s.Products[0].Rates) != 2 {
		t.Fatalf("grouped products=%d rates=%d", len(s.Products), len(s.Products[0].Rates))
	}
}

func TestBatchEffectiveNotBeforeDate(t *testing.T) {
	ctx := context.Background()
	mi := miLossBatchSchema{Date: "2026-08-10", EffectiveFrom: "2026-08-01"}
	mi.Sanitize()
	if err := mi.Validate(ctx); err == nil {
		t.Fatal("MI-loss effective before document date must fail")
	}
	fcf := fcfBatchSchema{Date: "2026-08-10", EffectiveFrom: "2026-08-01"}
	fcf.Sanitize()
	if err := fcf.Validate(ctx); err == nil {
		t.Fatal("FSF effective before document date must fail")
	}
	ok := fcfBatchSchema{Date: "2026-08-01", EffectiveFrom: "2026-08-10"}
	ok.Sanitize()
	if err := ok.Validate(ctx); err != nil {
		t.Fatalf("effective on or after document date should pass: %v", err)
	}
}

func TestMiLossBatchProducts(t *testing.T) {
	ctx := context.Background()
	s := miLossBatchSchema{
		Date: "2026-08-01",
		Products: []miLossProductSchema{{
			ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			Rates: []miLossRateSchema{
				{ContractTypeCode: "SRT", Value: "0.005"},
				{ContractTypeCode: "TOP", Value: "0.004"},
				{ContractTypeCode: "ADHOC", Value: "0.003"},
			},
		}},
	}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("one product with three contracts should pass: %v", err)
	}
	s.Products = append(s.Products, miLossProductSchema{ProductID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"})
	if err := s.Validate(ctx); err == nil {
		t.Fatal("duplicate product on the batch must fail")
	}
}

func TestChangeOfServiceParcel(t *testing.T) {
	ctx := context.Background()
	s := changeOfServiceSchema{
		EffectiveDate: "2026-09-01",
		CustomerID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		ParcelID:      "01ARZ3NDEKTSV4RRFFQ69G5FB0",
		ToCollection:  "loading",
	}
	s.Sanitize()
	if err := s.Validate(ctx); err != nil {
		t.Fatalf("valid COS rejected: %v", err)
	}
	if s.ToCollection != "LOADING" {
		t.Fatalf("collection: %s", s.ToCollection)
	}
	s.ToCollection = ""
	if err := s.Validate(ctx); err == nil {
		t.Fatal("delivery method required")
	}
	s.ToCollection = "LOADING45"
	if err := s.Validate(ctx); err == nil {
		t.Fatal("LOADING45 is not a delivery method")
	}
}

func TestRequiredDecimalRejectsNegative(t *testing.T) {
	if err := requiredDecimal("-1"); err == nil {
		t.Fatal("negative should fail")
	}
	if err := requiredDecimal("0"); err != nil {
		t.Fatalf("zero should be allowed: %v", err)
	}
	if err := requiredPositiveDecimal("0"); err == nil {
		t.Fatal("zero should fail positive check")
	}
}

func TestCompactUpper(t *testing.T) {
	if compact("  a  ") != "a" {
		t.Fatal("compact")
	}
	if !strings.EqualFold(upper("usd"), "USD") {
		t.Fatal("upper")
	}
}
