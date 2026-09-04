package settings

import (
	"context"
	"testing"
)

func TestPrecisionUpdateBounds(t *testing.T) {
	ctx := context.Background()
	ok := precisionUpdateRequest{
		QuantityPrecision: 3, CubicMeterPrecision: 3, MetricTonnePrecision: 3, DensityPrecision: 5,
		PricePrecision: 4, MiLossPrecision: 4, IloExpiryDays: 14,
	}
	if err := ok.Validate(ctx); err != nil {
		t.Fatal(err)
	}
	ok.QuantityPrecision = 7
	if err := ok.Validate(ctx); err == nil {
		t.Fatal("places > 6 must fail")
	}
	ok.QuantityPrecision = 3
	ok.CubicMeterPrecision = 7
	if err := ok.Validate(ctx); err == nil {
		t.Fatal("m³ places > 6 must fail")
	}
	ok.CubicMeterPrecision = 3
	ok.IloExpiryDays = 0
	if err := ok.Validate(ctx); err == nil {
		t.Fatal("ilo days 0 must fail")
	}
}

func TestSessionUpdateBounds(t *testing.T) {
	ctx := context.Background()
	idle := 1
	if err := (sessionUpdateRequest{IdleMinutes: &idle}).Validate(ctx); err == nil {
		t.Fatal("idle 1 must fail")
	}
	idle = 10
	warn := 2
	if err := (sessionUpdateRequest{IdleMinutes: &idle, WarnMinutes: &warn}).Validate(ctx); err != nil {
		t.Fatalf("valid session: %v", err)
	}
	warn = 11
	if err := (sessionUpdateRequest{WarnMinutes: &warn}).Validate(ctx); err == nil {
		t.Fatal("warn 11 must fail")
	}
}

func TestNpgisLicenseURLValidation(t *testing.T) {
	ctx := context.Background()
	if err := (npgisUpdateRequest{LicenseURL: "not-a-url"}).Validate(ctx); err == nil {
		t.Fatal("invalid license URL must fail")
	}
	if err := (npgisUpdateRequest{LicenseURL: "https://ewura.example/licenses"}).Validate(ctx); err != nil {
		t.Fatalf("valid license URL: %v", err)
	}
	if err := (npgisUpdateRequest{
		LicenseURL:  "https://www.ewura.go.tz/licensees/fetch-licensees/Petroleum",
		LicenseNo:   "PSBL-2018-003",
		APISourceID: "TIPER_DEPOT",
	}).Validate(ctx); err != nil {
		t.Fatalf("TIPER EWURA defaults: %v", err)
	}
}

func TestMailAndSageSeedValuesValidate(t *testing.T) {
	ctx := context.Background()
	port := 465
	ssl := true
	if err := (mailUpdateRequest{
		Host:      "smtp.gmail.com",
		Port:      &port,
		User:      "tipergantry@gmail.com",
		FromEmail: "tipergantry@gmail.com",
		UseSSL:    &ssl,
	}).Validate(ctx); err != nil {
		t.Fatalf("TIPER mail defaults: %v", err)
	}
	if err := (smsUpdateRequest{
		APIURL:   "https://api.notify.africa/api/v1/api/messages/send",
		SenderID: "563",
	}).Validate(ctx); err != nil {
		t.Fatalf("TIPER SMS defaults: %v", err)
	}
	if err := (sageUpdateRequest{
		Host: "173.249.63.248",
		Port: "1343",
		User: "sa",
		Name: "TIPER LIMITED",
	}).Validate(ctx); err != nil {
		t.Fatalf("TIPER Sage defaults: %v", err)
	}
}

func TestUploadsDirectoryRejectsTraversal(t *testing.T) {
	ctx := context.Background()
	bad := "../etc"
	if err := (uploadsUpdateRequest{Directory: &bad}).Validate(ctx); err == nil {
		t.Fatal(".. in directory must fail")
	}
	ok := `D:\dfms\uploads`
	r := uploadsUpdateRequest{Directory: &ok}
	r.Sanitize()
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("absolute directory rejected: %v", err)
	}
}

func TestCurrencyCreateRequiresCodeAndSymbol(t *testing.T) {
	ctx := context.Background()
	if err := (currencyCreateRequest{Code: "KES"}).Validate(ctx); err == nil {
		t.Fatal("expected symbol required")
	}
	if err := (currencyCreateRequest{Code: "USD", Symbol: "$"}).Validate(ctx); err != nil {
		t.Fatalf("valid currency: %v", err)
	}
}
