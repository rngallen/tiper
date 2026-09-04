package masterdata

import (
	"context"
	"strings"
	"time"

	"dfms/pkg/types"

	"github.com/jellydator/validation"
	"github.com/jellydator/validation/is"
	"github.com/shopspring/decimal"
)

func compact(s string) string { return strings.TrimSpace(s) }
func upper(s string) string   { return strings.ToUpper(strings.TrimSpace(s)) }
func lower(s string) string   { return strings.ToLower(strings.TrimSpace(s)) }

// alphaNumUpper keeps letters and digits only, uppercased (plates, licences).
func alphaNumUpper(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func optionalEmail(value string) []validation.Rule {
	return []validation.Rule{
		validation.Length(0, 160),
		validation.When(value != "", is.EmailFormat),
	}
}

func requiredDecimal(value any) error {
	s, _ := value.(string)
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return validation.NewError("validation_required", "is required")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return validation.NewError("validation_decimal", "must be a number")
	}
	if d.IsNegative() {
		return validation.NewError("validation_decimal", "must not be negative")
	}
	return nil
}

func capacityGreaterThanDead(cap, dead string) error {
	c, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(cap), ",", ""))
	if err != nil {
		return nil
	}
	d, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(dead), ",", ""))
	if err != nil {
		return nil
	}
	if !c.GreaterThan(d) {
		return validation.NewError("validation_capacity", "must be greater than dead stock")
	}
	return nil
}

func optionalDecimal(value any) error {
	s, _ := value.(string)
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return nil
	}
	if _, err := decimal.NewFromString(s); err != nil {
		return validation.NewError("validation_decimal", "must be a number")
	}
	return nil
}

func optionalDate(value any) error {
	s, _ := value.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return nil
	}
	return validation.NewError("validation_date", "must be a date (YYYY-MM-DD)")
}

// ── Categories ──────────────────────────────────────────────

type categoryRequest struct {
	Name string `json:"name"`
}

func (r *categoryRequest) Sanitize() { r.Name = compact(r.Name) }

func (r categoryRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
	)
}

type categoryUpdateRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"isActive"`
}

func (r *categoryUpdateRequest) Sanitize() { r.Name = compact(r.Name) }

func (r categoryUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
	)
}

// ── Products ────────────────────────────────────────────────

type productRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	StockCategoryID string `json:"stockCategoryId"`
	IsActive        *bool  `json:"isActive"`
}

func (r *productRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.StockCategoryID = compact(r.StockCategoryID)
}

func (r productRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Code, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.Name, validation.Required, validation.Length(2, 120)),
		validation.Field(&r.StockCategoryID, validation.Required, validation.Length(1, 26)),
	)
}

type productUpdateRequest struct {
	Name            string `json:"name"`
	StockCategoryID string `json:"stockCategoryId"`
	IsActive        *bool  `json:"isActive"`
}

func (r *productUpdateRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.StockCategoryID = compact(r.StockCategoryID)
}

func (r productUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 120)),
		validation.Field(&r.StockCategoryID, validation.Required, validation.Length(1, 26)),
	)
}

// ── Stock statuses ──────────────────────────────────────────

type statusRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	IsTransit   bool   `json:"isTransit"`
	IsLocal     bool   `json:"isLocal"`
	IsMining    bool   `json:"isMining"`
	IsProration bool   `json:"isProration"`
	ParentID    string `json:"parentId"`
	IsActive    *bool  `json:"isActive"`
}

func (r *statusRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.ParentID = compact(r.ParentID)
}

func (r statusRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
		validation.Field(&r.Code, validation.Length(0, 20)),
		validation.Field(&r.ParentID, validation.Length(0, 26)),
	)
}

// ── Tanks ───────────────────────────────────────────────────

type tankRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	MaximumCapacity string `json:"maximumCapacity"`
	DeadStock       string `json:"deadStock"`
	ProductID       string `json:"productId"`
	IsActive        *bool  `json:"isActive"`
}

func (r *tankRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.MaximumCapacity = strings.ReplaceAll(compact(r.MaximumCapacity), ",", "")
	r.DeadStock = strings.ReplaceAll(compact(r.DeadStock), ",", "")
	r.ProductID = compact(r.ProductID)
}

func (r tankRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Code, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
		validation.Field(&r.MaximumCapacity, validation.By(requiredDecimal), validation.By(func(any) error {
			return capacityGreaterThanDead(r.MaximumCapacity, r.DeadStock)
		})),
		validation.Field(&r.DeadStock, validation.By(requiredDecimal)),
		validation.Field(&r.ProductID, validation.Required, validation.Length(1, 26)),
	)
}

type tankUpdateRequest struct {
	Name            string `json:"name"`
	MaximumCapacity string `json:"maximumCapacity"`
	DeadStock       string `json:"deadStock"`
	ProductID       string `json:"productId"`
	IsActive        *bool  `json:"isActive"`
}

func (r *tankUpdateRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.MaximumCapacity = strings.ReplaceAll(compact(r.MaximumCapacity), ",", "")
	r.DeadStock = strings.ReplaceAll(compact(r.DeadStock), ",", "")
	r.ProductID = compact(r.ProductID)
}

func (r tankUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
		validation.Field(&r.MaximumCapacity, validation.By(requiredDecimal), validation.By(func(any) error {
			return capacityGreaterThanDead(r.MaximumCapacity, r.DeadStock)
		})),
		validation.Field(&r.DeadStock, validation.By(requiredDecimal)),
		validation.Field(&r.ProductID, validation.Required, validation.Length(1, 26)),
	)
}

// ── Vessels ─────────────────────────────────────────────────

type vesselRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ImoNumber string `json:"imoNumber"`
	IsActive  *bool  `json:"isActive"`
}

func (r *vesselRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.ImoNumber = upper(r.ImoNumber)
}

func (r vesselRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 120)),
		validation.Field(&r.Code, validation.Length(0, 20)),
		validation.Field(&r.ImoNumber, validation.Length(0, 20)),
	)
}

// ── Depots ──────────────────────────────────────────────────

type depotRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	EwuraLicense string `json:"ewuraLicense"`
	IsInternal   bool   `json:"isInternal"`
	CustomerID   string `json:"customerId"`
	IsActive     *bool  `json:"isActive"`
}

func (r *depotRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.EwuraLicense = compact(r.EwuraLicense)
	r.CustomerID = compact(r.CustomerID)
}

func (r depotRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Code, validation.Length(0, 20)),
		validation.Field(&r.Name, validation.Required, validation.Length(2, 120)),
		validation.Field(&r.EwuraLicense, validation.Length(0, 40)),
		validation.Field(&r.CustomerID, validation.Length(0, 26)),
	)
}

type depotUpdateRequest struct {
	Name         string `json:"name"`
	EwuraLicense string `json:"ewuraLicense"`
	IsInternal   bool   `json:"isInternal"`
	CustomerID   string `json:"customerId"`
	IsActive     *bool  `json:"isActive"`
}

func (r *depotUpdateRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.EwuraLicense = compact(r.EwuraLicense)
	r.CustomerID = compact(r.CustomerID)
}

func (r depotUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 120)),
		validation.Field(&r.EwuraLicense, validation.Length(0, 40)),
		validation.Field(&r.CustomerID, validation.Length(0, 26)),
	)
}

// ── Customers / suppliers ───────────────────────────────────

type customerRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	TinNumber    string `json:"tinNumber"`
	KycNumber    string `json:"kycNumber"`
	VrnNumber    string `json:"vrnNumber"`
	EwuraLicense string `json:"ewuraLicense"`
	IsActive     *bool  `json:"isActive"`
}

func (r *customerRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.Email = lower(r.Email)
	r.Phone = compact(r.Phone)
	r.TinNumber = compact(r.TinNumber)
	r.KycNumber = compact(r.KycNumber)
	r.VrnNumber = upper(r.VrnNumber)
	r.EwuraLicense = compact(r.EwuraLicense)
}

func (r customerRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 160)),
		validation.Field(&r.Code, validation.Length(0, 20)),
		validation.Field(&r.Email, optionalEmail(r.Email)...),
		validation.Field(&r.Phone, validation.Length(0, 24)),
		validation.Field(&r.TinNumber, validation.Length(0, 40)),
		validation.Field(&r.KycNumber, validation.Required, validation.Length(1, 40)),
		validation.Field(&r.VrnNumber, validation.Length(0, 40)),
		validation.Field(&r.EwuraLicense, validation.Required, validation.Length(1, 40)),
	)
}

type supplierRequest struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Mobile        string `json:"mobile"`
	ContactPerson string `json:"contactPerson"`
	TinNumber     string `json:"tinNumber"`
	CountryCode   string `json:"countryCode"`
	Address       string `json:"address"`
	Address2      string `json:"address2"`
	IsActive      *bool  `json:"isActive"`
}

func (r *supplierRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
	r.Email = lower(r.Email)
	r.Phone = compact(r.Phone)
	r.Mobile = compact(r.Mobile)
	r.ContactPerson = compact(r.ContactPerson)
	r.TinNumber = compact(r.TinNumber)
	r.CountryCode = upper(r.CountryCode)
	r.Address = compact(r.Address)
	r.Address2 = compact(r.Address2)
}

func (r supplierRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 160)),
		validation.Field(&r.Code, validation.Length(0, 20)),
		validation.Field(&r.Email, optionalEmail(r.Email)...),
		validation.Field(&r.Phone, validation.Length(0, 24)),
		validation.Field(&r.Mobile, validation.Length(0, 24)),
		validation.Field(&r.ContactPerson, validation.Length(0, 120)),
		validation.Field(&r.TinNumber, validation.Length(0, 40)),
		validation.Field(&r.CountryCode, validation.Length(0, 2)),
		validation.Field(&r.Address, validation.Length(0, 120)),
		validation.Field(&r.Address2, validation.Length(0, 120)),
	)
}

type billingAccountRequest struct {
	FeeCode     string `json:"feeCode"`
	SageAccount string `json:"sageAccount"`
	BillingUnit string `json:"billingUnit"`
	IsActive    *bool  `json:"isActive"`
}

func (r *billingAccountRequest) Sanitize() {
	r.FeeCode = upper(r.FeeCode)
	r.SageAccount = compact(r.SageAccount)
	r.BillingUnit = types.NormalizeBillingUnit(r.BillingUnit)
}

func (r billingAccountRequest) Validate(ctx context.Context) error {
	unit := r.BillingUnit
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.FeeCode, validation.Required, validation.In(
			string(types.FeeFSF), string(types.FeeVSF), string(types.FeeKOJ), string(types.FeeTBS),
		)),
		validation.Field(&r.SageAccount, validation.Required, validation.Length(1, 40)),
		validation.Field(&r.BillingUnit, validation.When(unit != "", validation.By(func(any) error {
			if !types.BillingUnitValid(unit) {
				return validation.NewError("validation_unit", "must be L, MT, or M3")
			}
			return nil
		}))),
	)
}

// ── Transporters ────────────────────────────────────────────

type transporterRequest struct {
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	ContactPerson string `json:"contactPerson"`
	TinNumber     string `json:"tinNumber"`
	VrnNumber     string `json:"vrnNumber"`
	License       string `json:"license"`
	CountryCode   string `json:"countryCode"`
	Address       string `json:"address"`
	Address2      string `json:"address2"`
	AeoEndDate    string `json:"aeoEndDate"`
	IsActive      *bool  `json:"isActive"`
}

func (r *transporterRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.Phone = compact(r.Phone)
	r.Email = lower(r.Email)
	r.ContactPerson = compact(r.ContactPerson)
	r.TinNumber = compact(r.TinNumber)
	r.VrnNumber = upper(r.VrnNumber)
	r.License = compact(r.License)
	r.CountryCode = upper(r.CountryCode)
	r.Address = compact(r.Address)
	r.Address2 = compact(r.Address2)
	r.AeoEndDate = compact(r.AeoEndDate)
}

func (r transporterRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 180)),
		validation.Field(&r.Phone, validation.Length(0, 24)),
		validation.Field(&r.Email, optionalEmail(r.Email)...),
		validation.Field(&r.ContactPerson, validation.Length(0, 120)),
		validation.Field(&r.TinNumber, validation.Length(0, 24)),
		validation.Field(&r.VrnNumber, validation.Length(0, 24)),
		validation.Field(&r.License, validation.Length(0, 40)),
		validation.Field(&r.CountryCode, validation.Length(0, 2)),
		validation.Field(&r.Address, validation.Length(0, 120)),
		validation.Field(&r.Address2, validation.Length(0, 120)),
		validation.Field(&r.AeoEndDate, validation.By(optionalDate)),
	)
}

// ── Drivers ─────────────────────────────────────────────────

type driverRequest struct {
	Name           string `json:"name"`
	LicenseNumber  string `json:"licenseNumber"`
	LicenseExpires string `json:"licenseExpires"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	IsActive       *bool  `json:"isActive"`
}

func (r *driverRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.LicenseNumber = alphaNumUpper(r.LicenseNumber)
	r.LicenseExpires = compact(r.LicenseExpires)
	r.Phone = compact(r.Phone)
	r.Email = lower(r.Email)
}

func (r driverRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 160)),
		validation.Field(&r.LicenseNumber, validation.Required, is.Alphanumeric, validation.Length(2, 40)),
		validation.Field(&r.LicenseExpires, validation.By(optionalDate)),
		validation.Field(&r.Phone, validation.Length(0, 24)),
		validation.Field(&r.Email, optionalEmail(r.Email)...),
	)
}

type driverUpdateRequest struct {
	Name           string `json:"name"`
	LicenseExpires string `json:"licenseExpires"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	IsActive       *bool  `json:"isActive"`
}

func (r *driverUpdateRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.LicenseExpires = compact(r.LicenseExpires)
	r.Phone = compact(r.Phone)
	r.Email = lower(r.Email)
}

func (r driverUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 160)),
		validation.Field(&r.LicenseExpires, validation.By(optionalDate)),
		validation.Field(&r.Phone, validation.Length(0, 24)),
		validation.Field(&r.Email, optionalEmail(r.Email)...),
	)
}

// ── Trucks ──────────────────────────────────────────────────

type truckRequest struct {
	PlateNumber string `json:"plateNumber"`
	Trailer     string `json:"trailer"`
	TrailerTwo  string `json:"trailerTwo"`
	VehicleType string `json:"vehicleType"`
	LoadingType string `json:"loadingType"`
	LngCng      bool   `json:"lngCng"`
	Mplw        string `json:"mplw"`
	Gcwr        string `json:"gcwr"`
	TareWeight  string `json:"tareWeight"`
	IsActive    *bool  `json:"isActive"`
}

func (r *truckRequest) Sanitize() {
	r.PlateNumber = alphaNumUpper(r.PlateNumber)
	r.Trailer = alphaNumUpper(r.Trailer)
	r.TrailerTwo = alphaNumUpper(r.TrailerTwo)
	r.VehicleType = lower(r.VehicleType)
	if r.VehicleType == "horse" {
		r.VehicleType = string(types.VehiclePulling)
	}
	r.LoadingType = lower(r.LoadingType)
	r.Mplw = strings.ReplaceAll(compact(r.Mplw), ",", "")
	r.Gcwr = strings.ReplaceAll(compact(r.Gcwr), ",", "")
	r.TareWeight = strings.ReplaceAll(compact(r.TareWeight), ",", "")
	applyVehiclePlateRules(r)
}

func (r truckRequest) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.PlateNumber, validation.Required, is.Alphanumeric, validation.Length(2, 20)),
		validation.Field(&r.Trailer, validation.When(r.Trailer != "", is.Alphanumeric, validation.Length(2, 20))),
		validation.Field(&r.TrailerTwo, validation.When(r.TrailerTwo != "", is.Alphanumeric, validation.Length(2, 20))),
		validation.Field(&r.VehicleType, validation.In(
			"", string(types.VehiclePending), string(types.VehicleStraight),
			string(types.VehicleSemi), string(types.VehiclePulling),
		)),
		validation.Field(&r.LoadingType, validation.In("", string(types.LoadingTop), string(types.LoadingBottom))),
		validation.Field(&r.Mplw, validation.By(optionalDecimal)),
		validation.Field(&r.Gcwr, validation.By(optionalDecimal)),
		validation.Field(&r.TareWeight, validation.By(optionalDecimal)),
	); err != nil {
		return err
	}
	return validateTruckShape(r, !types.VehicleTypeConfigured(types.VehicleType(r.VehicleType)))
}

// ── Destinations / districts ────────────────────────────────

type destinationRequest struct {
	Name      string `json:"name"`
	IsCountry bool   `json:"isCountry"`
	IsActive  *bool  `json:"isActive"`
}

func (r *destinationRequest) Sanitize() { r.Name = compact(r.Name) }

func (r destinationRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
	)
}

type districtRequest struct {
	Name          string `json:"name"`
	DestinationID string `json:"destinationId"`
	IsActive      *bool  `json:"isActive"`
}

func (r *districtRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.DestinationID = compact(r.DestinationID)
}

func (r districtRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
		validation.Field(&r.DestinationID, validation.Required, validation.Length(1, 26)),
	)
}

type districtUpdateRequest struct {
	Name          string `json:"name"`
	DestinationID string `json:"destinationId"`
	IsActive      *bool  `json:"isActive"`
}

func (r *districtUpdateRequest) Sanitize() {
	r.Name = compact(r.Name)
	r.DestinationID = compact(r.DestinationID)
}

func (r districtUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
		validation.Field(&r.DestinationID, validation.Length(0, 26)),
	)
}

// ── Billing catalogs (code PK) ──────────────────────────────

type catalogNamedRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive *bool  `json:"isActive"`
}

func (r *catalogNamedRequest) Sanitize() {
	r.Code = upper(r.Code)
	r.Name = compact(r.Name)
}

func (r catalogNamedRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Code, validation.Required, validation.By(func(value any) error {
			s, _ := value.(string)
			if !types.CatalogCodeOK(s) {
				return validation.NewError("validation_catalog", "must be a code without digits")
			}
			return nil
		})),
		validation.Field(&r.Name, validation.Required, validation.Length(2, 80)),
	)
}

type tenderRequest struct {
	catalogNamedRequest
	IsSingleReceiving         bool `json:"isSingleReceiving"`
	SupplierPaysUnlessLoading bool `json:"supplierPaysUnlessLoading"`
}

func (r *tenderRequest) Sanitize() { r.catalogNamedRequest.Sanitize() }

func (r tenderRequest) Validate(ctx context.Context) error {
	return r.catalogNamedRequest.Validate(ctx)
}

type deliveryRequest struct {
	catalogNamedRequest
	IsGantryLoading bool `json:"isGantryLoading"`
}

func (r *deliveryRequest) Sanitize() { r.catalogNamedRequest.Sanitize() }

func (r deliveryRequest) Validate(ctx context.Context) error {
	return r.catalogNamedRequest.Validate(ctx)
}

type pricingRequest struct {
	catalogNamedRequest
	IsPromotional bool `json:"isPromotional"`
}

func (r *pricingRequest) Sanitize() { r.catalogNamedRequest.Sanitize() }

func (r pricingRequest) Validate(ctx context.Context) error {
	return r.catalogNamedRequest.Validate(ctx)
}

type cycleRequest struct {
	Days        int    `json:"days"`
	Description string `json:"description"`
	IsActive    *bool  `json:"isActive"`
}

func (r *cycleRequest) Sanitize() { r.Description = compact(r.Description) }

func (r cycleRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Days, validation.Required, validation.Min(1), validation.Max(480)),
		validation.Field(&r.Description, validation.Required, validation.Length(2, 80)),
	)
}
