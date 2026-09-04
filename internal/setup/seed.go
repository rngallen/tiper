// Package setup seeds reference/master data that the rest of the system depends
// on: currencies, ISO countries, DFMS lookups and the company profile.
package setup

import (
	"fmt"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

type currencySeed struct {
	Code   string
	Name   string
	Symbol string
}

// defaultCurrencies is TIPER's Sage Currency table (code / description / symbol)
// plus TZS for company profile and local billing. Sage 200 and Sage 300 both
// identify a currency by CurrencyCode — DFMS does not store Sage numeric ids.
// Client.iCurrencyID is mapped at read time: 0=TZS (home, no Currency row),
// 1=USD, 2=GBP, 3=EUR, 4=ZAR, 5=CHF. See internal/sage.CurrencyCode.
var defaultCurrencies = []currencySeed{
	{Code: "USD", Name: "US Dollar", Symbol: "$"},
	{Code: "GBP", Name: "Pound", Symbol: "£"},
	{Code: "EUR", Name: "Euro", Symbol: "€"},
	{Code: "ZAR", Name: "South African Rand", Symbol: "R"},
	{Code: "CHF", Name: "Swiss Franc", Symbol: "CHf"},
	{Code: "TZS", Name: "Tanzanian Shilling", Symbol: "TSh"},
}

// CurrencyCodes is the seeded ISO catalogue (tests / diagnostics).
func CurrencyCodes() []string {
	out := make([]string, len(defaultCurrencies))
	for i, c := range defaultCurrencies {
		out[i] = c.Code
	}
	return out
}

// SeedReference inserts currencies, countries, DFMS lookups and the company profile.
// Currencies and countries are matched by Code (PK).
func SeedReference(db *gorm.DB) error {
	if err := seedCurrencies(db); err != nil {
		return err
	}
	if err := seedCountries(db); err != nil {
		return err
	}
	if err := seedLookups(db); err != nil {
		return err
	}
	if err := seedCompanyDetails(db); err != nil {
		return err
	}
	return nil
}

func seedCountries(db *gorm.DB) error {
	for _, def := range defaultCountries {
		var row models.Country
		if err := db.Where(models.Country{Code: def.Code}).
			Attrs(models.Country{
				Name:     def.Name,
				Alpha3:   def.Alpha3,
				Numeric:  def.Numeric,
				IsActive: true,
			}).
			Assign(models.Country{
				Name:     def.Name,
				Alpha3:   def.Alpha3,
				Numeric:  def.Numeric,
				IsActive: true,
			}).
			FirstOrCreate(&row, models.Country{Code: def.Code}).Error; err != nil {
			return fmt.Errorf("seed country %s: %w", def.Code, err)
		}
	}
	return nil
}

func seedCurrencies(db *gorm.DB) error {
	adminID, err := systemAdminID(db)
	if err != nil {
		return fmt.Errorf("seed currencies: %w", err)
	}
	for _, def := range defaultCurrencies {
		var row models.Currency
		if err := db.Where(models.Currency{Code: def.Code}).
			Attrs(models.Currency{
				Name:        def.Name,
				Symbol:      def.Symbol,
				IsActive:    true,
				CreatedByID: adminID,
			}).
			Assign(models.Currency{
				Name:     def.Name,
				Symbol:   def.Symbol,
				IsActive: true,
			}).
			FirstOrCreate(&row, models.Currency{Code: def.Code}).Error; err != nil {
			return fmt.Errorf("seed currency %s: %w", def.Code, err)
		}
	}
	// Previous seed invented AUD/INR and mapped Sage ids that are not on this
	// TIPER company. Leave the rows but hide them from pickers.
	if err := db.Model(&models.Currency{}).Where("Code IN ?", []string{"AUD", "INR"}).
		Update("IsActive", false).Error; err != nil {
		return fmt.Errorf("retire leftover currencies: %w", err)
	}
	return nil
}

// systemAdminID returns the bootstrap administrator (IsSuperUser) used as
// CreatedBy for seeded reference rows. Prefer the oldest super-user so the
// link stays stable across restarts.
func systemAdminID(db *gorm.DB) (uint, error) {
	var admin models.User
	err := db.Where("IsSuperUser = ? AND IsActive = ?", true, true).
		First(&admin).Error
	if err != nil {
		return 0, fmt.Errorf("admin user not found (run auth seed / ensure a super-user exists): %w", err)
	}
	return admin.ID, nil
}

func seedLookups(db *gorm.DB) error {
	for _, def := range []models.UnitOfMeasure{
		{Code: "L", Description: "Litre @ 20°C", IsActive: true},
		{Code: "M3", Description: "Cubic metre @ 20°C", IsActive: true},
		{Code: "MT", Description: "Metric tonne", IsActive: true},
	} {
		row := def
		if err := db.Where(models.UnitOfMeasure{Code: def.Code}).Attrs(def).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	for _, f := range []models.Fee{
		{Code: types.FeeFSF, Name: "Fixed storage fee", ChargeTo: types.ChargeToBoth, IsActive: true},
		{Code: types.FeeVSF, Name: "Variable storage fee", ChargeTo: types.ChargeToBoth, IsActive: true},
		{Code: types.FeeKOJ, Name: "KOJ facility fee", ChargeTo: types.ChargeToCustomer, IsActive: true},
		{Code: types.FeeTBS, Name: "TBS truck loading fee", ChargeTo: types.ChargeToCustomer, IsActive: true},
	} {
		if err := db.Where(models.Fee{Code: f.Code}).Assign(models.Fee{
			Name: f.Name, ChargeTo: f.ChargeTo, IsActive: f.IsActive,
		}).FirstOrCreate(&f).Error; err != nil {
			return err
		}
	}
	for _, r := range []models.DischargeRoute{
		{Code: types.RouteSBM, Name: "Single Buoy Mooring", IsActive: true},
		{Code: types.RouteKOJ, Name: "Kurasini Oil Jetty", IsActive: true},
	} {
		if err := db.Where(models.DischargeRoute{Code: r.Code}).Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	for _, t := range []models.ImportTenderType{
		{Code: types.TenderSRT, Name: "Single Receiving Terminal", IsSingleReceiving: true, SupplierPaysUnlessLoading: true, IsActive: true},
		{Code: types.TenderNonSRT, Name: "Non-SRT", IsActive: true},
	} {
		if err := db.Where(models.ImportTenderType{Code: t.Code}).Attrs(t).FirstOrCreate(&t).Error; err != nil {
			return err
		}
	}
	for _, p := range []models.ProcurementMethod{
		{Code: types.ProcurementBPS, Name: "Bulk Procurement System", IsActive: true},
		{Code: types.ProcurementPrivate, Name: "Private / non-BPS vessel", IsActive: true},
	} {
		if err := db.Where(models.ProcurementMethod{Code: p.Code}).Attrs(p).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	for _, m := range []models.DeliveryMethod{
		{Code: types.CollectionLoading, Name: "Gantry / one-stop loading", IsGantryLoading: true, IsActive: true},
		{Code: types.CollectionPumpOver, Name: "Pump-over", IsActive: true},
	} {
		if err := db.Where(models.DeliveryMethod{Code: m.Code}).Attrs(m).FirstOrCreate(&m).Error; err != nil {
			return err
		}
	}
	for _, p := range []models.PricingNature{
		{Code: types.PricingTariff, Name: "Tariff", IsActive: true},
		{Code: types.PricingPromotional, Name: "Promotional", IsPromotional: true, IsActive: true},
	} {
		if err := db.Where(models.PricingNature{Code: p.Code}).Attrs(p).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	if err := seedStockStatuses(db); err != nil {
		return err
	}
	for _, c := range []models.ContractType{
		{Code: types.ContractAdhoc, Name: "Ad hoc", IsActive: true},
		{Code: types.ContractSRT, Name: "SRT", IsActive: true},
		{Code: types.ContractTop, Name: "Take or Pay", IsActive: true},
	} {
		row := c
		if err := db.Where(models.ContractType{Code: c.Code}).Assign(models.ContractType{
			Name: c.Name, IsActive: c.IsActive,
		}).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	for _, c := range []models.BillingCycle{
		{Days: 15, Description: "15-day next billing", IsActive: true},
		{Days: 30, Description: "30-day next billing", IsActive: true},
		{Days: 45, Description: "45-day next billing", IsActive: true},
	} {
		row := c
		if err := db.Where(models.BillingCycle{Days: c.Days}).Attrs(c).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	adminID, err := systemAdminID(db)
	if err != nil {
		return err
	}
	petrol := models.StockCategory{Name: "Petroleum products", CreatedByID: adminID, IsActive: true}
	if err := db.Where(models.StockCategory{Name: petrol.Name}).Attrs(petrol).FirstOrCreate(&petrol).Error; err != nil {
		return err
	}
	for _, p := range []models.Product{
		{Code: "1002", Name: "Automotive Gas Oil", Unit: "L", StockCategoryID: petrol.ID, CreatedByID: adminID, IsActive: true},
		{Code: "1001", Name: "Premium Motor Spirit", Unit: "L", StockCategoryID: petrol.ID, CreatedByID: adminID, IsActive: true},
	} {
		row := p
		legacy := "AGO"
		if p.Code == "1001" {
			legacy = "PMS"
		}
		if err := db.Where("Code IN ?", []string{p.Code, legacy}).
			Assign(models.Product{Code: p.Code, Unit: "L", StockCategoryID: petrol.ID}).
			FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	if err := db.Model(&models.Product{}).Where("Unit <> ?", "L").Update("Unit", "L").Error; err != nil {
		return err
	}
	if err := db.Model(&models.Product{}).Where("StockCategoryID <> ?", petrol.ID).Update("StockCategoryID", petrol.ID).Error; err != nil {
		return err
	}
	if err := db.Where("ID <> ?", petrol.ID).Delete(&models.StockCategory{}).Error; err != nil {
		if err := db.Model(&models.StockCategory{}).Where("ID <> ?", petrol.ID).Update("IsActive", false).Error; err != nil {
			return err
		}
	}
	depot := models.Depot{Code: "TIPER", Name: "TIPER Kigamboni", IsInternal: true, IsActive: true, CreatedByID: adminID}
	if err := db.Where(models.Depot{Code: depot.Code}).Attrs(depot).FirstOrCreate(&depot).Error; err != nil {
		return err
	}
	var kojCust uint
	db.Model(&models.CustomerBillingAccount{}).Select("CustomerID").
		Where("FeeCode = ? AND IsActive = ?", types.FeeKOJ, true).
		Order("ID ASC").Limit(1).Scan(&kojCust)
	if kojCust > 0 {
		_ = db.Model(&models.Depot{}).Where("CustomerID IS NULL").Update("CustomerID", kojCust).Error
	}
	return nil
}

const (
	defaultCompanyTIN = "100-103-362"
	defaultCompanyVRN = "10-00115-Z"
	defaultCompanyISO = "ISO 9001: 2015 and ISO 45001: 2018"
)

func seedCompanyDetails(db *gorm.DB) error {
	var company models.Company
	result := db.Where(models.Company{ID: 1}).
		Attrs(models.Company{
			Name:         "Tanzania International Petroleum Reserves Limited",
			TinNumber:    defaultCompanyTIN,
			VrnNumber:    defaultCompanyVRN,
			IsoNumber:    defaultCompanyISO,
			CurrencyCode: new("TZS"),
			Address:      "Kigamboni Depot Site - Plot 1",
			Address2:     "Kigamboni Industrial Area",
			PostalCode:   "P.O. Box 2608",
			City:         "Dar es Salaam",
			Country:      "Tanzania",
			Phone:        "+255 (0) 22 5511 500",
			PortalURL:    "http://127.0.0.1:3000",
			Email:        "info@tiper.co.tz",
			Website:      "https://tiper.co.tz",
		}).
		FirstOrCreate(&company)
	return result.Error
}

func seedStockStatuses(db *gorm.DB) error {
	roots := []models.StockStatus{
		{Code: string(types.StockLocal), Name: "Local", IsLocal: true, IsActive: true},
		{Code: string(types.StockTransit), Name: "Transit", IsTransit: true, IsActive: true},
		{Code: string(types.StockProration), Name: "Proration", IsProration: true, IsActive: true},
	}
	for _, s := range roots {
		row := s
		if err := db.Where(models.StockStatus{Code: s.Code}).
			Assign(models.StockStatus{
				Name: s.Name, IsTransit: s.IsTransit, IsLocal: s.IsLocal,
				IsProration: s.IsProration, IsActive: true,
			}).
			FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	var local, transit models.StockStatus
	if err := db.Where("Code = ?", types.StockLocal).First(&local).Error; err != nil {
		return err
	}
	if err := db.Where("Code = ?", types.StockTransit).First(&transit).Error; err != nil {
		return err
	}
	if err := db.Model(&models.StockStatus{}).Where("Code = ?", "MINES").
		Updates(map[string]any{"Code": types.StockMining, "Name": "Mining"}).Error; err != nil {
		return err
	}
	children := []models.StockStatus{
		{Code: string(types.StockMining), Name: "Mining", IsLocal: true, IsMining: true, ParentID: &local.ID, IsActive: true},
		{Code: "CONGO", Name: "Congo", IsTransit: true, ParentID: &transit.ID, IsActive: true},
		{Code: "RWANDA", Name: "Rwanda", IsTransit: true, ParentID: &transit.ID, IsActive: true},
		{Code: "BURUNDI", Name: "Burundi", IsTransit: true, ParentID: &transit.ID, IsActive: true},
		{Code: "MALAWI", Name: "Malawi", IsTransit: true, ParentID: &transit.ID, IsActive: true},
		{Code: "ZAMBIA", Name: "Zambia", IsTransit: true, ParentID: &transit.ID, IsActive: true},
	}
	for _, s := range children {
		row := s
		if err := db.Where(models.StockStatus{Code: s.Code}).
			Assign(models.StockStatus{
				Name: s.Name, IsTransit: s.IsTransit, IsLocal: s.IsLocal, IsMining: s.IsMining,
				IsProration: s.IsProration, ParentID: s.ParentID, IsActive: true,
			}).
			FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
