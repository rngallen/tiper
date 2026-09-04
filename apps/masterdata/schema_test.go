package masterdata

import (
	"context"
	"strings"
	"testing"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestCustomerRequestRequiresLicenseAndName(t *testing.T) {
	ctx := context.Background()
	var r customerRequest
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("expected validation error for empty customer")
	}
	r.Name = "PUMA Energy"
	r.EwuraLicense = "TL-123"
	r.KycNumber = "10001"
	r.Sanitize()
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("valid customer rejected: %v", err)
	}
}

func TestCustomerRequestRequiresKyc(t *testing.T) {
	ctx := context.Background()
	r := customerRequest{Name: "PUMA Energy", EwuraLicense: "TL-123"}
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("expected validation error for empty KYC number")
	}
}

func TestCustomerRequestRejectsBadEmail(t *testing.T) {
	ctx := context.Background()
	r := customerRequest{Name: "PUMA Energy", EwuraLicense: "TL-123", KycNumber: "10001", Email: "not-an-email"}
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("expected invalid email to fail")
	}
	r.Email = "ops@example.com"
	r.Sanitize()
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("valid email rejected: %v", err)
	}
	if r.Email != "ops@example.com" {
		t.Fatalf("email not lowercased: %q", r.Email)
	}
}

func TestCustomerRequestAllowsEmptyEmailAndTin(t *testing.T) {
	ctx := context.Background()
	r := customerRequest{Name: "PUMA Energy", EwuraLicense: "TL-123", KycNumber: "10001"}
	r.Sanitize()
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("empty email/tin should be allowed so licence can fill them: %v", err)
	}
}

func TestFillFromLicenseOnlyWhenEmpty(t *testing.T) {
	lic := &models.EwuraPetroleumLicense{Email: "licence@ewura.go.tz", TinNumber: "100-111-222", Phone: "255755000000"}
	email, tin, phone := "keep@omc.tz", "", ""
	fillFromLicense(lic, &email, &tin, &phone)
	if email != "keep@omc.tz" {
		t.Fatalf("submitted email was overwritten: %s", email)
	}
	if tin != "100-111-222" {
		t.Fatalf("empty tin was not filled: %s", tin)
	}
	if phone != "255755000000" {
		t.Fatalf("empty phone was not filled: %s", phone)
	}
	email2, tin2, phone2 := "", "", "keep-phone"
	fillFromLicense(lic, &email2, &tin2, &phone2)
	if email2 != "licence@ewura.go.tz" || tin2 != "100-111-222" {
		t.Fatalf("empty fields were not filled: %s %s", email2, tin2)
	}
	if phone2 != "keep-phone" {
		t.Fatalf("submitted phone was overwritten: %s", phone2)
	}
}

func TestDriverRequestRequiresLicence(t *testing.T) {
	ctx := context.Background()
	r := driverRequest{Name: "Jane Doe"}
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("expected licence number required")
	}
	r.LicenseNumber = "DL-1"
	r.Sanitize()
	if r.LicenseNumber != "DL1" {
		t.Fatalf("licence should be alphanumeric uppercase, got %q", r.LicenseNumber)
	}
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("valid driver rejected: %v", err)
	}
}

func TestValidateTruckShapeHorseOnlyCreate(t *testing.T) {
	ctx := context.Background()
	in := truckRequest{PlateNumber: "t 124 abc"}
	in.Sanitize()
	if err := in.Validate(ctx); err != nil {
		t.Fatalf("horse-only create rejected: %v", err)
	}
	if in.Trailer != "" || in.TrailerTwo != "" {
		t.Fatal("pending create must not infer tank plates")
	}
	if resolveVehicleType(in) != types.VehiclePending {
		t.Fatal("create without type stays pending")
	}
}

func TestValidateTruckShapePendingKeepsPlates(t *testing.T) {
	in := truckRequest{PlateNumber: "T1", Trailer: "T2", TrailerTwo: "T3"}
	in.Sanitize()
	if err := validateTruckShape(in, true); err != nil {
		t.Fatal(err)
	}
	if in.Trailer != "T2" || in.TrailerTwo != "T3" {
		t.Fatalf("pending create must keep tank plates, got %q %q", in.Trailer, in.TrailerTwo)
	}
	if resolveVehicleType(in) != types.VehiclePending {
		t.Fatal("create without type stays pending")
	}
}

func TestValidateTruckShapePendingSkipsTanks(t *testing.T) {
	in := truckRequest{PlateNumber: "T124ABC", VehicleType: "pending"}
	in.Sanitize()
	if err := validateTruckShape(in, true); err != nil {
		t.Fatal(err)
	}
}

func TestApplyVehiclePlateRulesOnTypeChange(t *testing.T) {
	in := truckRequest{PlateNumber: "T1", Trailer: "T2", TrailerTwo: "T3", VehicleType: "straight"}
	in.Sanitize()
	if in.Trailer != "T1" || in.TrailerTwo != "" {
		t.Fatalf("straight must use horse as tank one and empty tank two, got %q %q", in.Trailer, in.TrailerTwo)
	}
	in.Trailer, in.TrailerTwo, in.VehicleType = "T2", "T3", "semi"
	in.Sanitize()
	if in.TrailerTwo != "" {
		t.Fatal("semi must clear tank two")
	}
	if in.Trailer != "T2" {
		t.Fatalf("semi must keep tank one, got %q", in.Trailer)
	}
}

func TestSanitizeHorseBecomesPulling(t *testing.T) {
	in := truckRequest{PlateNumber: "T1", Trailer: "T2", TrailerTwo: "T3", VehicleType: "horse"}
	in.Sanitize()
	if in.VehicleType != string(types.VehiclePulling) {
		t.Fatalf("horse must save as pulling, got %q", in.VehicleType)
	}
	if err := in.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTruckShapeStraightOnly(t *testing.T) {
	in := truckRequest{PlateNumber: "T123", VehicleType: "semi"}
	in.Sanitize()
	if err := validateTruckShape(in, false); err == nil {
		t.Fatal("semi with horse only should fail")
	}
	in.VehicleType = "straight"
	in.Sanitize()
	if err := validateTruckShape(in, false); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTruckShapeStraightPlatesMustMatch(t *testing.T) {
	ctx := context.Background()
	raw := truckRequest{PlateNumber: "T124ABC", Trailer: "T121ABC", VehicleType: "straight"}
	if err := validateTruckShape(raw, false); err == nil {
		t.Fatal("straight with different horse and tank one must fail before sanitize")
	}
	in := raw
	in.Sanitize()
	if err := in.Validate(ctx); err != nil {
		t.Fatalf("sanitize should align straight plates: %v", err)
	}
	if in.Trailer != in.PlateNumber {
		t.Fatal("straight must force tank one to the horse plate")
	}
	if in.TrailerTwo != "" {
		t.Fatal("straight must have empty tank two")
	}
}

func TestValidateTruckShapeSemiAndPulling(t *testing.T) {
	ctx := context.Background()
	raw := truckRequest{PlateNumber: "T1", Trailer: "T2", TrailerTwo: "T3", VehicleType: "semi"}
	if err := validateTruckShape(raw, false); err == nil {
		t.Fatal("semi with tank two must fail before sanitize")
	}
	semi := raw
	semi.Sanitize()
	if err := semi.Validate(ctx); err != nil {
		t.Fatalf("sanitize should drop tank two for semi: %v", err)
	}
	if semi.TrailerTwo != "" {
		t.Fatal("semi must clear tank two")
	}
	same := truckRequest{PlateNumber: "T1", Trailer: "T1", VehicleType: "semi"}
	same.Sanitize()
	if err := same.Validate(ctx); err != nil {
		t.Fatalf("semi may use the same horse and tank one: %v", err)
	}
	pull := truckRequest{PlateNumber: "T1", Trailer: "T2", VehicleType: "pulling"}
	pull.Sanitize()
	if err := pull.Validate(ctx); err == nil {
		t.Fatal("pulling without tank two must fail")
	}
	pull.TrailerTwo = "T3"
	pull.Sanitize()
	if err := pull.Validate(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTruckPlatesAlphanumericUpper(t *testing.T) {
	ctx := context.Background()
	in := truckRequest{PlateNumber: "t 124 abc", Trailer: "t 124 abc", VehicleType: "straight"}
	in.Sanitize()
	if in.PlateNumber != "T124ABC" {
		t.Fatalf("plate %q", in.PlateNumber)
	}
	if err := in.Validate(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredDecimal(t *testing.T) {
	if err := requiredDecimal(""); err == nil {
		t.Fatal("empty must fail")
	}
	if err := requiredDecimal("abc"); err == nil {
		t.Fatal("letters must fail")
	}
	if err := requiredDecimal("-1"); err == nil {
		t.Fatal("negative must fail")
	}
	if err := requiredDecimal("1,200.5"); err != nil {
		t.Fatal(err)
	}
}

func TestProductRequiresCategory(t *testing.T) {
	ctx := context.Background()
	r := productRequest{Code: "AGO", Name: "Automotive Gas Oil"}
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("category required")
	}
	r.StockCategoryID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if err := r.Validate(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTankRequiresNumericCapacityAndProduct(t *testing.T) {
	ctx := context.Background()
	r := tankRequest{Code: "T1", Name: "Tank 1", MaximumCapacity: "abc", DeadStock: "0"}
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("letters in capacity must fail")
	}
	r.MaximumCapacity = "50000"
	r.DeadStock = "200"
	if err := r.Validate(ctx); err == nil {
		t.Fatal("product required")
	}
	r.ProductID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if err := r.Validate(ctx); err != nil {
		t.Fatal(err)
	}
	r.DeadStock = "50000"
	if err := r.Validate(ctx); err == nil {
		t.Fatal("dead stock equal to capacity must fail")
	}
	r.DeadStock = "50001"
	if err := r.Validate(ctx); err == nil {
		t.Fatal("dead stock above capacity must fail")
	}
	r.DeadStock = "49999.999"
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("capacity above dead stock rejected: %v", err)
	}
}

func TestOptionalDecimalRejectsLetters(t *testing.T) {
	if err := optionalDecimal("12.5"); err != nil {
		t.Fatalf("12.5 should parse: %v", err)
	}
	if err := optionalDecimal("abc"); err == nil {
		t.Fatal("letters should fail")
	}
	if err := optionalDecimal(""); err != nil {
		t.Fatal("empty decimal is optional")
	}
}

func TestCompactHelpers(t *testing.T) {
	if upper("  ab ") != "AB" {
		t.Fatal(upper("  ab "))
	}
	if lower("  AB@X.COM ") != "ab@x.com" {
		t.Fatal(lower("  AB@X.COM "))
	}
	if compact("  x  ") != "x" {
		t.Fatal(compact("  x  "))
	}
	if !strings.EqualFold("TL-1", "tl-1") {
		t.Fatal("sanity")
	}
}

func TestBillingAccountRequestRequiresFeeAndSageAccount(t *testing.T) {
	ctx := context.Background()
	var r billingAccountRequest
	r.Sanitize()
	if err := r.Validate(ctx); err == nil {
		t.Fatal("expected validation error")
	}
	r.FeeCode = "vsf"
	r.SageAccount = " 4000 "
	r.BillingUnit = "cm"
	r.Sanitize()
	if r.FeeCode != "VSF" || r.SageAccount != "4000" || r.BillingUnit != "M3" {
		t.Fatalf("sanitize: %+v", r)
	}
	if err := r.Validate(ctx); err != nil {
		t.Fatalf("valid mapping rejected: %v", err)
	}
	r.FeeCode = "XYZ"
	if err := r.Validate(ctx); err == nil {
		t.Fatal("unknown fee should fail")
	}
}
