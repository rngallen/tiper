// Package catalogs loads billing classification tables used on documents.
// Fact rows store the code; flags on the row drive engine behavior.
package catalogs

import (
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

type Set struct {
	Tenders   map[string]models.ImportTenderType
	Routes    map[string]models.DischargeRoute
	Delivery  map[string]models.DeliveryMethod
	Procure   map[string]models.ProcurementMethod
	Contracts map[string]models.ContractType
	Pricing   map[string]models.PricingNature
	Cycles    map[int]models.BillingCycle
}

func Load(db *gorm.DB) (Set, error) {
	s := Set{
		Tenders:   map[string]models.ImportTenderType{},
		Routes:    map[string]models.DischargeRoute{},
		Delivery:  map[string]models.DeliveryMethod{},
		Procure:   map[string]models.ProcurementMethod{},
		Contracts: map[string]models.ContractType{},
		Pricing:   map[string]models.PricingNature{},
		Cycles:    map[int]models.BillingCycle{},
	}
	var tenders []models.ImportTenderType
	var routes []models.DischargeRoute
	var delivery []models.DeliveryMethod
	var procs []models.ProcurementMethod
	var contracts []models.ContractType
	var pricing []models.PricingNature
	var cycles []models.BillingCycle
	if err := db.Find(&tenders).Error; err != nil {
		return s, err
	}
	if err := db.Find(&routes).Error; err != nil {
		return s, err
	}
	if err := db.Find(&delivery).Error; err != nil {
		return s, err
	}
	if err := db.Find(&procs).Error; err != nil {
		return s, err
	}
	if err := db.Find(&contracts).Error; err != nil {
		return s, err
	}
	if err := db.Find(&pricing).Error; err != nil {
		return s, err
	}
	if err := db.Find(&cycles).Error; err != nil {
		return s, err
	}
	for _, r := range tenders {
		s.Tenders[string(r.Code)] = r
	}
	for _, r := range routes {
		s.Routes[string(r.Code)] = r
	}
	for _, r := range delivery {
		s.Delivery[string(r.Code)] = r
	}
	for _, r := range procs {
		s.Procure[string(r.Code)] = r
	}
	for _, r := range contracts {
		s.Contracts[string(r.Code)] = r
	}
	for _, r := range pricing {
		s.Pricing[string(r.Code)] = r
	}
	for _, r := range cycles {
		s.Cycles[r.Days] = r
	}
	return s, nil
}

func normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (s Set) SupplierPaysUnlessLoading(tender string) bool {
	r, ok := s.Tenders[normalize(tender)]
	return ok && r.SupplierPaysUnlessLoading
}

func (s Set) GantryLoading(collection string) bool {
	r, ok := s.Delivery[normalize(collection)]
	return ok && r.IsGantryLoading
}

func (s Set) Promotional(nature string) bool {
	n := normalize(nature)
	if n == "" || n == string(types.PricingTariff) {
		return false
	}
	if n == string(types.PricingPromotion) {
		n = string(types.PricingPromotional)
	}
	r, ok := s.Pricing[n]
	return ok && r.IsPromotional
}

func RequireActive(s Set, kind, code string) error {
	code = normalize(code)
	if code == "" {
		return fmt.Errorf("%s is required", kind)
	}
	if !types.CatalogCodeOK(code) {
		return fmt.Errorf("invalid %s code", kind)
	}
	switch kind {
	case "tender":
		r, ok := s.Tenders[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown tender %s", code)
		}
	case "route":
		r, ok := s.Routes[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown discharge route %s", code)
		}
	case "delivery":
		r, ok := s.Delivery[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown delivery method %s", code)
		}
	case "procurement":
		r, ok := s.Procure[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown procurement method %s", code)
		}
	case "contract":
		r, ok := s.Contracts[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown contract type %s", code)
		}
	case "pricing":
		if code == string(types.PricingPromotion) {
			code = string(types.PricingPromotional)
		}
		r, ok := s.Pricing[code]
		if !ok || !r.IsActive {
			return fmt.Errorf("unknown pricing nature %s", code)
		}
	default:
		return fmt.Errorf("unknown catalog %s", kind)
	}
	return nil
}

func RequireCycle(s Set, days int) error {
	if days <= 0 {
		return nil
	}
	r, ok := s.Cycles[days]
	if !ok || !r.IsActive {
		return fmt.Errorf("unknown billing cycle %d days", days)
	}
	return nil
}
