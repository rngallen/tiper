package catalogs

import (
	"testing"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestFlagsFromCatalog(t *testing.T) {
	s := Set{
		Tenders: map[string]models.ImportTenderType{
			"SRT": {Code: types.TenderSRT, IsSingleReceiving: true, SupplierPaysUnlessLoading: true, IsActive: true},
		},
		Routes: map[string]models.DischargeRoute{
			"KOJ": {Code: types.RouteKOJ, IsActive: true},
			"SBM": {Code: types.RouteSBM, IsActive: true},
		},
		Delivery: map[string]models.DeliveryMethod{
			"LOADING": {Code: types.CollectionLoading, IsGantryLoading: true, IsActive: true},
		},
		Pricing: map[string]models.PricingNature{
			"PROMOTIONAL": {Code: types.PricingPromotional, IsPromotional: true, IsActive: true},
			"TARIFF":      {Code: types.PricingTariff, IsActive: true},
		},
	}
	if !s.Tenders["SRT"].IsSingleReceiving || !s.SupplierPaysUnlessLoading("SRT") {
		t.Fatal("srt flags")
	}
	if !s.GantryLoading("LOADING") || s.GantryLoading("PUMPOVER") {
		t.Fatal("gantry flag")
	}
	if !s.Promotional("PROMOTIONAL") || !s.Promotional("PROMOTION") || s.Promotional("TARIFF") {
		t.Fatal("pricing flag")
	}
	if err := RequireActive(s, "tender", "SRT"); err != nil {
		t.Fatal(err)
	}
	if err := RequireActive(s, "route", "XYZ"); err == nil {
		t.Fatal("unknown route")
	}
}
