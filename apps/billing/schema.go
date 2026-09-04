package billing

import (
	"context"
	"strings"

	"dfms/pkg/types"

	"github.com/jellydator/validation"
	"github.com/shopspring/decimal"
)

func compact(s string) string { return strings.TrimSpace(s) }
func upper(s string) string   { return strings.ToUpper(strings.TrimSpace(s)) }

func stripNum(s string) string { return strings.ReplaceAll(compact(s), ",", "") }

func optionalDecimal(value any) error {
	s, _ := value.(string)
	s = stripNum(s)
	if s == "" {
		return nil
	}
	if _, err := decimal.NewFromString(s); err != nil {
		return validation.NewError("validation_decimal", "must be a number")
	}
	return nil
}

func requiredDecimal(value any) error {
	s, _ := value.(string)
	s = stripNum(s)
	if s == "" {
		return validation.ErrRequired
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

func requiredDate(value any) error {
	s, _ := value.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return validation.ErrRequired
	}
	if parseDate(s).IsZero() {
		return validation.NewError("validation_date", "must be a date (YYYY-MM-DD)")
	}
	return nil
}

func effectiveOnOrAfterDoc(date, effective string) error {
	d := dateOnly(parseDate(date))
	e := dateOnly(parseDate(effective))
	if d.IsZero() || e.IsZero() {
		return nil
	}
	if e.Before(d) {
		return validation.NewError("validation_effective", "effective from cannot be before the document date")
	}
	return nil
}

func requiredPositiveDecimal(value any) error {
	if err := requiredDecimal(value); err != nil {
		return err
	}
	s, _ := value.(string)
	if parseDec(s).IsZero() {
		return validation.NewError("validation_decimal", "must be greater than zero")
	}
	return nil
}

func billingUnit(s string) string { return types.NormalizeBillingUnit(s) }

func validBillingUnit(value any) error {
	s, _ := value.(string)
	if !types.BillingUnitValid(s) {
		return validation.NewError("validation_unit", "must be L, MT, or M3")
	}
	return nil
}

// ── Fixed storage fee batch ──────────────────────────────────

type fcfTierSchema struct {
	Phase       string `json:"phase"`
	FromQty     string `json:"fromQty"`
	ToQty       string `json:"toQty"`
	SourcePrice string `json:"sourcePrice"`
}

func (s *fcfTierSchema) Sanitize() {
	s.Phase = strings.ToLower(compact(s.Phase))
	s.FromQty = stripNum(s.FromQty)
	s.ToQty = stripNum(s.ToQty)
	s.SourcePrice = stripNum(s.SourcePrice)
}

func (s fcfTierSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Phase, validation.Required, validation.In(string(types.PhaseFirst), string(types.PhaseNth))),
		validation.Field(&s.FromQty, validation.By(requiredDecimal)),
		validation.Field(&s.ToQty, validation.By(optionalDecimal)),
		validation.Field(&s.SourcePrice, validation.By(requiredDecimal)),
	); err != nil {
		return err
	}
	if s.ToQty != "" {
		from := parseDec(s.FromQty)
		to := parseDec(s.ToQty)
		if to.LessThan(from) {
			return validation.NewError("validation_tier", "to quantity must be at least from quantity")
		}
	}
	return nil
}

type fcfLineSchema struct {
	ClassOfTrade            string          `json:"classOfTrade"`
	ProcurementMethod       string          `json:"procurementMethod"`
	DischargeRoute          string          `json:"dischargeRoute"`
	CollectionMethod        string          `json:"collectionMethod"`
	IsPromotional           bool            `json:"isPromotional"`
	FirstDays               int             `json:"firstDays"`
	FirstChargeTo           string          `json:"firstChargeTo"`
	FirstRateKind           string          `json:"firstRateKind"`
	FirstUnit               string          `json:"firstUnit"`
	FirstSourceCurrencyCode string          `json:"firstSourceCurrencyCode"`
	FirstSourcePrice        string          `json:"firstSourcePrice"`
	NthDays                 int             `json:"nthDays"`
	NthRateKind             string          `json:"nthRateKind"`
	NthUnit                 string          `json:"nthUnit"`
	NthSourceCurrencyCode   string          `json:"nthSourceCurrencyCode"`
	NthSourcePrice          string          `json:"nthSourcePrice"`
	Tiers                   []fcfTierSchema `json:"tiers"`
}

func (s *fcfLineSchema) Sanitize() {
	s.ClassOfTrade = upper(s.ClassOfTrade)
	s.ProcurementMethod = upper(s.ProcurementMethod)
	s.DischargeRoute = upper(s.DischargeRoute)
	s.CollectionMethod = upper(s.CollectionMethod)
	s.FirstChargeTo = strings.ToLower(compact(s.FirstChargeTo))
	s.FirstRateKind = string(types.NormalizeRateKind(s.FirstRateKind))
	s.FirstUnit = billingUnit(upper(s.FirstUnit))
	s.FirstSourceCurrencyCode = upper(s.FirstSourceCurrencyCode)
	s.FirstSourcePrice = stripNum(s.FirstSourcePrice)
	s.NthRateKind = string(types.NormalizeRateKind(s.NthRateKind))
	s.NthUnit = billingUnit(upper(s.NthUnit))
	s.NthSourceCurrencyCode = upper(s.NthSourceCurrencyCode)
	s.NthSourcePrice = stripNum(s.NthSourcePrice)
	if s.ClassOfTrade == "NONSRT" {
		s.ClassOfTrade = string(types.TenderNonSRT)
	}
	if s.ProcurementMethod == "" {
		s.ProcurementMethod = string(types.ProcurementBPS)
	}
	if s.FirstDays <= 0 {
		s.FirstDays = 15
	}
	if s.NthDays <= 0 {
		s.NthDays = 30
	}
	if s.FirstUnit == "" {
		s.FirstUnit = "M3"
	}
	if s.NthUnit == "" {
		s.NthUnit = "M3"
	}
	if s.FirstSourceCurrencyCode == "" {
		s.FirstSourceCurrencyCode = "USD"
	}
	if s.NthSourceCurrencyCode == "" {
		s.NthSourceCurrencyCode = "USD"
	}
	for i := range s.Tiers {
		s.Tiers[i].Sanitize()
	}
}

func catalogCode(required bool) validation.Rule {
	return validation.By(func(value any) error {
		s, _ := value.(string)
		if s == "" {
			if required {
				return validation.NewError("validation_required", "is required")
			}
			return nil
		}
		if !types.CatalogCodeOK(s) {
			return validation.NewError("validation_catalog", "must be a catalog code without encoded days")
		}
		return nil
	})
}

func (s fcfLineSchema) phaseTiers(phase string) int {
	n := 0
	for _, t := range s.Tiers {
		if t.Phase == phase {
			n++
		}
	}
	return n
}

func (s fcfLineSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ClassOfTrade, catalogCode(true)),
		validation.Field(&s.ProcurementMethod, catalogCode(true)),
		validation.Field(&s.DischargeRoute, catalogCode(true)),
		validation.Field(&s.CollectionMethod, catalogCode(false)),
		validation.Field(&s.FirstDays, validation.Min(1), validation.Max(480)),
		validation.Field(&s.NthDays, validation.Min(1), validation.Max(480)),
		validation.Field(&s.FirstChargeTo, validation.When(s.FirstChargeTo != "", validation.In(
			string(types.ChargeToCustomer),
			string(types.ChargeToSupplier),
			string(types.ChargeToBoth),
		))),
		validation.Field(&s.FirstRateKind, validation.In(string(types.RateFlat), string(types.RateTier))),
		validation.Field(&s.NthRateKind, validation.In(string(types.RateFlat), string(types.RateTier))),
		validation.Field(&s.FirstUnit, validation.Required, validation.By(validBillingUnit)),
		validation.Field(&s.NthUnit, validation.Required, validation.By(validBillingUnit)),
		validation.Field(&s.FirstSourceCurrencyCode, validation.Required, validation.Length(3, 3)),
		validation.Field(&s.NthSourceCurrencyCode, validation.Required, validation.Length(3, 3)),
		validation.Field(&s.FirstSourcePrice, validation.When(s.FirstRateKind != string(types.RateTier), validation.By(requiredDecimal))),
		validation.Field(&s.NthSourcePrice, validation.When(s.NthRateKind != string(types.RateTier), validation.By(requiredDecimal))),
	); err != nil {
		return err
	}
	if s.FirstRateKind == string(types.RateTier) && s.phaseTiers(string(types.PhaseFirst)) == 0 {
		return validation.NewError("validation_tier", "first billing volume tiers are required")
	}
	if s.NthRateKind == string(types.RateTier) && s.phaseTiers(string(types.PhaseNth)) == 0 {
		return validation.NewError("validation_tier", "second billing volume tiers are required")
	}
	for i := range s.Tiers {
		if err := s.Tiers[i].Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

type fcfBatchSchema struct {
	Date          string          `json:"date"`
	EffectiveFrom string          `json:"effectiveFrom"`
	Description   string          `json:"description"`
	ExchangeRate  string          `json:"exchangeRate"`
	FxManual      bool            `json:"fxManual"`
	Lines         []fcfLineSchema `json:"lines"`
}

func (s *fcfBatchSchema) Sanitize() {
	s.Date = compact(s.Date)
	s.EffectiveFrom = compact(s.EffectiveFrom)
	s.Description = compact(s.Description)
	s.ExchangeRate = stripNum(s.ExchangeRate)
	if s.EffectiveFrom == "" {
		s.EffectiveFrom = s.Date
	}
	for i := range s.Lines {
		s.Lines[i].Sanitize()
	}
}

func (s fcfBatchSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Date, validation.By(requiredDate)),
		validation.Field(&s.EffectiveFrom, validation.By(requiredDate)),
		validation.Field(&s.Description, validation.Length(0, 200)),
		validation.Field(&s.ExchangeRate, validation.By(optionalDecimal)),
		validation.Field(&s.Lines, validation.Length(0, 200)),
	); err != nil {
		return err
	}
	if err := effectiveOnOrAfterDoc(s.Date, s.EffectiveFrom); err != nil {
		return err
	}
	for i := range s.Lines {
		if err := s.Lines[i].Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ── KOJ / TBS price batch ───────────────────────────────────

type priceLineSchema struct {
	ProductID          string `json:"productId"`
	Unit               string `json:"unit"`
	SourceCurrencyCode string `json:"sourceCurrencyCode"`
	SourcePrice        string `json:"sourcePrice"`
}

func (s *priceLineSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
	s.Unit = billingUnit(upper(s.Unit))
	s.SourceCurrencyCode = upper(s.SourceCurrencyCode)
	s.SourcePrice = stripNum(s.SourcePrice)
}

func (s priceLineSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&s.Unit, validation.Required, validation.By(validBillingUnit)),
		validation.Field(&s.SourceCurrencyCode, validation.Required, validation.Length(3, 3)),
		validation.Field(&s.SourcePrice, validation.By(requiredPositiveDecimal)),
	)
}

type priceBatchSchema struct {
	Date          string            `json:"date"`
	EffectiveFrom string            `json:"effectiveFrom"`
	Description   string            `json:"description"`
	ExchangeRate  string            `json:"exchangeRate"`
	FxManual      bool              `json:"fxManual"`
	Fees          []priceLineSchema `json:"fees"`
}

func (s *priceBatchSchema) Sanitize() {
	s.Date = compact(s.Date)
	s.EffectiveFrom = compact(s.EffectiveFrom)
	s.Description = compact(s.Description)
	s.ExchangeRate = stripNum(s.ExchangeRate)
	if s.EffectiveFrom == "" {
		s.EffectiveFrom = s.Date
	}
	for i := range s.Fees {
		s.Fees[i].Sanitize()
	}
}

func (s priceBatchSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Date, validation.By(requiredDate)),
		validation.Field(&s.EffectiveFrom, validation.By(requiredDate)),
		validation.Field(&s.Description, validation.Length(0, 200)),
		validation.Field(&s.ExchangeRate, validation.By(optionalDecimal)),
		validation.Field(&s.Fees, validation.Length(0, 200)),
	); err != nil {
		return err
	}
	if err := effectiveOnOrAfterDoc(s.Date, s.EffectiveFrom); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for i := range s.Fees {
		if err := s.Fees[i].Validate(ctx); err != nil {
			return err
		}
		key := s.Fees[i].ProductID + "|" + s.Fees[i].Unit + "|" + s.Fees[i].SourceCurrencyCode
		if _, ok := seen[key]; ok {
			return validation.NewError("validation_unique", "each product, unit, and currency can appear only once on the batch")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func requiredDensity(value any) error {
	if err := requiredPositiveDecimal(value); err != nil {
		return err
	}
	s, _ := value.(string)
	d := parseDec(s)
	if d.GreaterThanOrEqual(decimal.NewFromInt(2)) {
		return validation.NewError("validation_density", "enter density as MT per m³ (e.g. 0.84), not kg/m³")
	}
	return nil
}

func requiredMiLossFraction(value any) error {
	if err := requiredDecimal(value); err != nil {
		return err
	}
	s, _ := value.(string)
	d := parseDec(s)
	if !d.GreaterThan(decimal.Zero) || !d.LessThan(decimal.NewFromInt(1)) {
		return validation.NewError("validation_miloss", "MI-loss must be greater than 0 and less than 1 (0.005 = 0.5%)")
	}
	return nil
}

// ── MI-loss batch ───────────────────────────────────────────

type miLossRateSchema struct {
	ContractTypeCode string `json:"contractTypeCode"`
	Value            string `json:"value"`
}

func (s *miLossRateSchema) Sanitize() {
	s.ContractTypeCode = upper(s.ContractTypeCode)
	s.Value = stripNum(s.Value)
}

func (s miLossRateSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ContractTypeCode, catalogCode(true)),
		validation.Field(&s.Value, validation.By(requiredMiLossFraction)),
	)
}

type miLossProductAddSchema struct {
	ProductID string `json:"productId"`
}

func (s *miLossProductAddSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
}

func (s miLossProductAddSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
	)
}

type miLossRateAddSchema struct {
	ProductID        string `json:"productId"`
	ContractTypeCode string `json:"contractTypeCode"`
	Value            string `json:"value"`
}

func (s *miLossRateAddSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
	s.ContractTypeCode = upper(s.ContractTypeCode)
	s.Value = stripNum(s.Value)
}

func (s miLossRateAddSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&s.ContractTypeCode, catalogCode(true)),
		validation.Field(&s.Value, validation.By(requiredMiLossFraction)),
	)
}

type miLossLineSchema struct {
	ProductID        string `json:"productId"`
	ContractTypeCode string `json:"contractTypeCode"`
	Value            string `json:"value"`
}

func (s *miLossLineSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
	s.ContractTypeCode = upper(s.ContractTypeCode)
	s.Value = stripNum(s.Value)
}

func (s miLossLineSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&s.ContractTypeCode, catalogCode(true)),
		validation.Field(&s.Value, validation.By(requiredMiLossFraction)),
	)
}

type miLossProductSchema struct {
	ProductID string             `json:"productId"`
	Rates     []miLossRateSchema `json:"rates"`
}

func (s *miLossProductSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
	for i := range s.Rates {
		s.Rates[i].Sanitize()
	}
}

func (s miLossProductSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&s.Rates, validation.Length(0, 40)),
	); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for i := range s.Rates {
		if err := s.Rates[i].Validate(ctx); err != nil {
			return err
		}
		if _, ok := seen[s.Rates[i].ContractTypeCode]; ok {
			return validation.NewError("validation_unique", "this product, contract, and effective date already have a rate")
		}
		seen[s.Rates[i].ContractTypeCode] = struct{}{}
	}
	return nil
}

type miLossBatchSchema struct {
	Date          string                `json:"date"`
	EffectiveFrom string                `json:"effectiveFrom"`
	Description   string                `json:"description"`
	Products      []miLossProductSchema `json:"products"`
	Lines         []miLossLineSchema    `json:"lines"`
}

func (s *miLossBatchSchema) Sanitize() {
	s.Date = compact(s.Date)
	s.EffectiveFrom = compact(s.EffectiveFrom)
	s.Description = compact(s.Description)
	if s.EffectiveFrom == "" {
		s.EffectiveFrom = s.Date
	}
	if s.Date == "" {
		s.Date = s.EffectiveFrom
	}
	for i := range s.Products {
		s.Products[i].Sanitize()
	}
	for i := range s.Lines {
		s.Lines[i].Sanitize()
	}
	if len(s.Products) == 0 && len(s.Lines) > 0 {
		byProduct := map[string]int{}
		for _, line := range s.Lines {
			i, ok := byProduct[line.ProductID]
			if !ok {
				i = len(s.Products)
				byProduct[line.ProductID] = i
				s.Products = append(s.Products, miLossProductSchema{ProductID: line.ProductID})
			}
			s.Products[i].Rates = append(s.Products[i].Rates, miLossRateSchema{
				ContractTypeCode: line.ContractTypeCode, Value: line.Value,
			})
		}
	}
}

func (s miLossBatchSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Date, validation.By(requiredDate)),
		validation.Field(&s.EffectiveFrom, validation.By(requiredDate)),
		validation.Field(&s.Description, validation.Length(0, 200)),
		validation.Field(&s.Products, validation.Length(0, 80)),
	); err != nil {
		return err
	}
	seenProduct := map[string]struct{}{}
	for i := range s.Products {
		if err := s.Products[i].Validate(ctx); err != nil {
			return err
		}
		if _, ok := seenProduct[s.Products[i].ProductID]; ok {
			return validation.NewError("validation_unique", "each product can appear only once on the batch")
		}
		seenProduct[s.Products[i].ProductID] = struct{}{}
	}
	return effectiveOnOrAfterDoc(s.Date, s.EffectiveFrom)
}

// ── Variable storage fee batch ──────────────────────────────

type varProductSchema struct {
	ProductID  string `json:"productId"`
	EwuraPrice string `json:"ewuraPrice"`
	Density    string `json:"density"`
}

func (s *varProductSchema) Sanitize() {
	s.ProductID = compact(s.ProductID)
	s.EwuraPrice = stripNum(s.EwuraPrice)
	s.Density = stripNum(s.Density)
}

func (s varProductSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&s.EwuraPrice, validation.By(requiredDecimal)),
		validation.Field(&s.Density, validation.By(requiredDensity)),
	)
}

type variableBatchSchema struct {
	Date          string             `json:"date"`
	EffectiveFrom string             `json:"effectiveFrom"`
	Description   string             `json:"description"`
	ExchangeRate  string             `json:"exchangeRate"`
	FxManual      bool               `json:"fxManual"`
	MiLossBatchID string             `json:"miLossBatchId"`
	Products      []varProductSchema `json:"products"`
}

func (s *variableBatchSchema) Sanitize() {
	s.Date = compact(s.Date)
	s.EffectiveFrom = compact(s.EffectiveFrom)
	s.Description = compact(s.Description)
	s.ExchangeRate = stripNum(s.ExchangeRate)
	s.MiLossBatchID = compact(s.MiLossBatchID)
	if s.EffectiveFrom == "" {
		s.EffectiveFrom = s.Date
	}
	for i := range s.Products {
		s.Products[i].Sanitize()
	}
}

func (s variableBatchSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Date, validation.By(requiredDate)),
		validation.Field(&s.EffectiveFrom, validation.By(requiredDate)),
		validation.Field(&s.Description, validation.Length(0, 200)),
		validation.Field(&s.ExchangeRate, validation.By(optionalDecimal)),
		validation.Field(&s.MiLossBatchID, validation.Required.Error("select an approved MI-loss batch"), validation.Length(1, 26)),
		validation.Field(&s.Products, validation.Length(0, 50)),
	); err != nil {
		return err
	}
	if err := effectiveOnOrAfterDoc(s.Date, s.EffectiveFrom); err != nil {
		return err
	}
	for i := range s.Products {
		if err := s.Products[i].Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ── Engine simulate ─────────────────────────────────────────

type simulateSchema struct {
	FeeCode  string `json:"feeCode"`
	Quantity string `json:"quantity"`
	Rate     string `json:"rate"`
}

func (s *simulateSchema) Sanitize() {
	s.FeeCode = upper(s.FeeCode)
	s.Quantity = stripNum(s.Quantity)
	s.Rate = stripNum(s.Rate)
}

func (s simulateSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.FeeCode, validation.Required, validation.In(
			string(types.FeeFSF), string(types.FeeVSF), string(types.FeeKOJ), string(types.FeeTBS),
		)),
		validation.Field(&s.Quantity, validation.By(requiredDecimal)),
		validation.Field(&s.Rate, validation.By(requiredDecimal)),
	)
}

type fxSchema struct {
	EffectiveFrom string `json:"effectiveFrom"`
	FromCurrency  string `json:"fromCurrency"`
	ToCurrency    string `json:"toCurrency"`
	Rate          string `json:"rate"`
}

func (s *fxSchema) Sanitize() {
	s.EffectiveFrom = compact(s.EffectiveFrom)
	s.FromCurrency = upper(s.FromCurrency)
	s.ToCurrency = upper(s.ToCurrency)
	s.Rate = compact(s.Rate)
}

func (s fxSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.EffectiveFrom, validation.By(requiredDate)),
		validation.Field(&s.FromCurrency, validation.Required, validation.Length(3, 3)),
		validation.Field(&s.ToCurrency, validation.Required, validation.Length(3, 3)),
		validation.Field(&s.Rate, validation.By(requiredPositiveDecimal)),
	)
}

type changeOfServiceSchema struct {
	EffectiveDate string `json:"effectiveDate"`
	CustomerID    string `json:"customerId"`
	ParcelID      string `json:"parcelId"`
	ToCollection  string `json:"toCollection"`
	Notes         string `json:"notes"`
}

func (s *changeOfServiceSchema) Sanitize() {
	s.EffectiveDate = compact(s.EffectiveDate)
	s.CustomerID = compact(s.CustomerID)
	s.ParcelID = compact(s.ParcelID)
	s.ToCollection = upper(s.ToCollection)
	s.Notes = compact(s.Notes)
}

func (s changeOfServiceSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.EffectiveDate, validation.By(requiredDate)),
		validation.Field(&s.CustomerID, validation.Required),
		validation.Field(&s.ParcelID, validation.Required),
		validation.Field(&s.ToCollection, validation.By(requiredCollection)),
	)
}

func requiredCollection(value any) error {
	s, _ := value.(string)
	if s == "" {
		return validation.NewError("validation_required", "delivery method is required")
	}
	if !types.CollectionMethod(s).Valid() {
		return validation.NewError("validation_collection", "unknown delivery method")
	}
	return nil
}
