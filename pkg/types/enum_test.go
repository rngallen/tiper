package types

import "testing"

func TestRoleFamily(t *testing.T) {
	if !RoleTerminal.Valid() || !RoleFinance.Valid() || RoleFamily("reception").Valid() {
		t.Fatal("role family catalogue")
	}
	if NormalizeRoleFamily("") != RoleSystem || NormalizeRoleFamily("GANTRY") != RoleGantry {
		t.Fatal("role family normalize")
	}
	if NormalizeRoleFamily("stock") != RoleTerminal || NormalizeRoleFamily("reception") != RoleTerminal {
		t.Fatal("legacy stock/reception map to terminal")
	}
	if RoleTerminal.Label() != "Terminal" || RoleFinance.Label() != "Finance" {
		t.Fatal("role family label")
	}
}

func TestOrderAndAmendEnums(t *testing.T) {
	if !OrderInProgress.Valid() || OrderStatus("in_progress").Valid() {
		t.Fatal("order status wire value is inprogress")
	}
	if !ImmediateAmendment(AmendExtend) || ImmediateAmendment(AmendNormal) {
		t.Fatal("immediate amendment kinds")
	}
	if !AmendQtyIncrease.Valid() {
		t.Fatal("amend kind")
	}
}

func TestBillingEnums(t *testing.T) {
	if !FeeFSF.Valid() || !ChargeToBoth.Valid() || !CollectionLoading.Valid() {
		t.Fatal("billing enums")
	}
	if !ChargeToBoth.Allows("customer") || !ChargeToBoth.Allows("supplier") {
		t.Fatal("charge-to both allows customer and supplier")
	}
	if !ChargeToCustomer.Allows("customer") || ChargeToCustomer.Allows("supplier") {
		t.Fatal("charge-to customer is customer-only")
	}
	if ChargeToSupplier.Allows("customer") || !ChargeToSupplier.Allows("supplier") {
		t.Fatal("charge-to supplier is supplier-only")
	}
	if CollectionPumpOver != "PUMPOVER" || RouteSBM != "SBM" {
		t.Fatal("django collection / route codes")
	}
	if CollectionMethod("LOADING45").Valid() {
		t.Fatal("collection must not encode billing days")
	}
	if !ContractAdhoc.Valid() || ContractCode("TOP45").Valid() {
		t.Fatal("contract catalog format")
	}
	if NormalizeBillingUnit("CM") != "M3" || !BillingUnitValid("L") {
		t.Fatal("billing units")
	}
	if NormalizeBillingUnit("cu.m") != "M3" || NormalizeBillingUnit("MCM") != "M3" {
		t.Fatal("cubic metre aliases must store as M3")
	}
	if BillingUnitValid("MCM") && NormalizeBillingUnit("MCM") != "M3" {
		t.Fatal("MCM is not a fourth unit")
	}
}

func TestNormalizeVehicleType(t *testing.T) {
	if NormalizeVehicleType("") != VehicleStraight {
		t.Fatal("empty type defaults to straight for orders")
	}
	if NormalizeVehicleType("horse") != VehiclePulling {
		t.Fatal("horse is not a type; it saves as pulling")
	}
	if NormalizeVehicleType("semi") != VehicleSemi || NormalizeVehicleType("pulling") != VehiclePulling {
		t.Fatal("vehicle types")
	}
	if NormalizeVehicleType("pending") != VehiclePending {
		t.Fatal("pending stays pending")
	}
	if VehicleTypeConfigured(VehiclePending) || VehicleTypeConfigured("") {
		t.Fatal("pending is not configured")
	}
	if !VehicleTypeConfigured(VehicleStraight) || !VehicleTypeConfigured(VehiclePulling) {
		t.Fatal("straight and pulling are configured")
	}
}

func TestContentTypeFolder(t *testing.T) {
	cases := map[ContentType]string{
		ReceiptContent:              "Receipts",
		GantryLoadingRequestContent: "ILR",
		IttTransferContent:          "ITT",
		PumpOverRequestContent:      "Pump-over-request",
		PumpOverReportContent:       "Pump-over-report",
		CustomerContent:             "Customers",
		DriverContent:               "Drivers",
		TankContent:                 "Tanks",
		TruckContent:                "Trucks",
		CompartmentalizationContent: "Compartmentalization",
		OrderAmendmentContent:       "Amendments",
		FinancialHoldContent:        "Financial-hold",
		ZerolizationContent:         "Zerolization",
		ChangeOfServiceContent:      "Change-of-service",
		BillingProfileContent:       "Fixed-storage-fees",
		BillingRunContent:           "Billing",
		EwuraLicenseContent:         "EWURA-licences",
	}
	for ct, want := range cases {
		if got := ContentTypeFolder(ct); got != want {
			t.Errorf("%d folder %q want %q", ct, got, want)
		}
	}
	if ContentTypeLabel(CustomerContent) != "Customer" || ContentTypeLabel(DriverContent) != "Driver" {
		t.Fatal("master labels")
	}
	if AuditTrailContent != 11 || PasswordHistoryContent != 23 || ChangeOfServiceContent != 71 {
		t.Fatalf("packed auth types / COS: audit=%d pwd=%d cos=%d", AuditTrailContent, PasswordHistoryContent, ChangeOfServiceContent)
	}
	if ReceiptContent != 51 || GantryLoadingRequestContent != 80 {
		t.Fatal("domain content types must keep historical numbers")
	}
}

func TestNotifyAndStock(t *testing.T) {
	if EmailBadgeAction != "action" || CredentialReset != "reset" {
		t.Fatal("email badge / credential")
	}
	if StockLocal != "LOCAL" || StockTransit != "TRANSIT" {
		t.Fatal("stock roots")
	}
	if ReceiptInternal != "internal" || NpgisPumpOver != "pump_over" || TxnLoading != "loading" {
		t.Fatal("receipt / npgis / txn")
	}
	if !ReceiptInternal.PostsStock() || ReceiptExternal.PostsStock() {
		t.Fatal("only internal receipts post stock")
	}
	if !ReceiptInternal.BillsStorage() || ReceiptExternal.BillsStorage() {
		t.Fatal("only internal receipts take first-cycle storage")
	}
	if !RouteKOJ.IsKOJ() || RouteSBM.IsKOJ() {
		t.Fatal("only KOJ route is the jetty that can take the 10-inch fee")
	}
}
