package permissions

import "testing"

func TestSatisfies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		held     []string
		required string
		want     bool
	}{
		{"exact", []string{ILRSubmit}, ILRSubmit, true},
		{"resource manage grants submit", []string{ILRManage}, ILRSubmit, true},
		{"resource manage grants complete", []string{ILRManage}, ILRComplete, true},
		{"module manage grants document action", []string{OrdersManage}, PumpReportSubmit, true},
		{"module read grants document read", []string{OrdersRead}, ILRRead, true},
		{"module submit grants document submit", []string{OrdersSubmit}, PumpOverSubmit, true},
		{"module create grants document create", []string{InventoryCreate}, ReceiptsCreate, true},
		{"customers manage grants create", []string{CustomersManage}, CustomersCreate, true},
		{"prices manage grants update", []string{PricesManage}, PricesUpdate, true},
		{"prices manage grants submit", []string{PricesManage}, PricesSubmit, true},
		{"two-part required by module manage", []string{OrdersManage}, OrdersRead, true},
		{"report submit does not grant request submit", []string{PumpReportSubmit}, PumpOverSubmit, false},
		{"request submit does not grant report submit", []string{PumpOverSubmit}, PumpReportSubmit, false},
		{"ilr submit does not grant ilr create", []string{ILRSubmit}, ILRCreate, false},
		{"module read does not grant submit", []string{OrdersRead}, ILRSubmit, false},
		{"module read does not grant complete", []string{OrdersRead}, ILRComplete, false},
		{"balances does not grant receipts read", []string{InventoryBalances}, ReceiptsRead, false},
		{"receipts read does not grant inventory read", []string{ReceiptsRead}, InventoryRead, false},
		{"inventory read grants receipts read", []string{InventoryRead}, ReceiptsRead, true},
		{"other module manage does not grant", []string{BillingManage}, ILRRead, false},
		{"empty required", []string{OrdersManage}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Satisfies(tc.held, tc.required); got != tc.want {
				t.Fatalf("Satisfies(%v, %q) = %v, want %v", tc.held, tc.required, got, tc.want)
			}
		})
	}
}

func TestSatisfiesAny(t *testing.T) {
	t.Parallel()
	held := []string{PumpReportRead}
	if !SatisfiesAny(held, PumpOverRead, PumpReportRead) {
		t.Fatal("expected report read to satisfy OR list")
	}
	if SatisfiesAny(held, PumpOverCreate, ILRCreate) {
		t.Fatal("report read must not satisfy create codes")
	}
}
