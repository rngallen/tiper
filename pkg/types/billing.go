package types

import "strings"

// FeeCode is a billed fee (FSF / VSF / KOJ / TBS).
type FeeCode string

const (
	FeeFSF FeeCode = "FSF"
	FeeVSF FeeCode = "VSF"
	FeeKOJ FeeCode = "KOJ"
	FeeTBS FeeCode = "TBS"
)

func (c FeeCode) Valid() bool {
	switch c {
	case FeeFSF, FeeVSF, FeeKOJ, FeeTBS:
		return true
	}
	return false
}

// ChargeTo is who a fee is billed to.
type ChargeTo string

const (
	ChargeToCustomer ChargeTo = "customer"
	ChargeToSupplier ChargeTo = "supplier"
	ChargeToBoth     ChargeTo = "both"
)

func (c ChargeTo) Valid() bool {
	return c == ChargeToCustomer || c == ChargeToSupplier || c == ChargeToBoth
}

// Allows reports whether this fee may be mapped to a customer or supplier
// billing account. party is "customer" or "supplier".
func (c ChargeTo) Allows(party string) bool {
	switch c {
	case ChargeToBoth:
		return party == "customer" || party == "supplier"
	case ChargeToCustomer:
		return party == "customer"
	case ChargeToSupplier:
		return party == "supplier"
	default:
		return false
	}
}

// CollectionMethod is how product leaves the depot (delivery-method catalog).
type CollectionMethod string

const (
	CollectionPumpOver CollectionMethod = "PUMPOVER"
	CollectionLoading  CollectionMethod = "LOADING"
)

func (c CollectionMethod) Valid() bool { return CatalogCodeOK(string(c)) }

// NormalizeBillingUnit maps every cubic-metre spelling to M3.
// Store only L, MT, or M3 — aliases (CM, CBM, CU.M, MCM, M³) are input-only.
// MCM here is cubic metre (TIPER / Django), not “thousand cubic metres”.
func NormalizeBillingUnit(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.NewReplacer(".", "", " ", "", "³", "3").Replace(s)
	switch s {
	case "CM", "CBM", "CUM", "MCM", "M3", "CUBICMETRE", "CUBICMETER":
		return "M3"
	case "L", "LTR", "LITRE", "LITER":
		return "L"
	case "MT", "TONNE", "TON":
		return "MT"
	default:
		return s
	}
}

func BillingUnitValid(s string) bool {
	switch NormalizeBillingUnit(s) {
	case "L", "MT", "M3":
		return true
	}
	return false
}

// TenderCode is the import-tender class on a receipt (catalog).
type TenderCode string

const (
	TenderSRT    TenderCode = "SRT"
	TenderNonSRT TenderCode = "NON-SRT"
)

func (c TenderCode) Valid() bool { return CatalogCodeOK(string(c)) }

// ProcurementCode is how the cargo was bought (catalog).
type ProcurementCode string

const (
	ProcurementBPS     ProcurementCode = "BPS"
	ProcurementPrivate ProcurementCode = "PRIVATE"
)

func (c ProcurementCode) Valid() bool { return CatalogCodeOK(string(c)) }

// DischargeRoute is a vessel berth (catalog; SBM / KOJ are seeds).
type DischargeRoute string

const (
	RouteSBM DischargeRoute = "SBM"
	RouteKOJ DischargeRoute = "KOJ"
)

func (c DischargeRoute) Valid() bool { return CatalogCodeOK(string(c)) }

// IsKOJ is the Kurasini Oil Jetty. KOJ fee is decided on the receipt
// (external + 10-inch pipeline), not by a second route code.
func (c DischargeRoute) IsKOJ() bool {
	return DischargeRoute(strings.ToUpper(strings.TrimSpace(string(c)))) == RouteKOJ
}

// ContractCode is the commercial contract of a parcel (catalog).
type ContractCode string

const (
	ContractAdhoc ContractCode = "ADHOC"
	ContractSRT   ContractCode = "SRT"
	ContractTop   ContractCode = "TOP"
)

func (c ContractCode) Valid() bool { return CatalogCodeOK(string(c)) }

// PricingNature is tariff vs promotional (catalog).
type PricingNature string

const (
	PricingTariff      PricingNature = "TARIFF"
	PricingPromotional PricingNature = "PROMOTIONAL"
	PricingStandard    PricingNature = ""
	PricingPromotion   PricingNature = "PROMOTION" // legacy alias of PROMOTIONAL
)

func (c PricingNature) Valid() bool {
	if c == "" {
		return true
	}
	return CatalogCodeOK(string(c))
}

// CatalogCodeOK is a format check for lookup codes stored on fact rows.
// Digits are rejected so collection cannot encode billing days (LOADING45).
func CatalogCodeOK(s string) bool {
	if s == "" || len(s) > 20 {
		return false
	}
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

// BillingSource identifies which engine created a BillingRun.
type BillingSource string

const (
	BillSrcFirst      BillingSource = "first"
	BillSrcNth        BillingSource = "nth"
	BillSrcKOJ        BillingSource = "koj"
	BillSrcTBSDaily   BillingSource = "tbs-daily"
	BillSrcVCFMonthly BillingSource = "vsf-monthly"
)

// RateKind is a flat unit price or a volume-slab tariff.
type RateKind string

const (
	RateFlat RateKind = "flat"
	RateTier RateKind = "tier"
)

func NormalizeRateKind(s string) RateKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "tier", "volume", "slab":
		return RateTier
	default:
		return RateFlat
	}
}

// BillingPhase is first reception billing vs repeating nth billing.
type BillingPhase string

const (
	PhaseFirst BillingPhase = "first"
	PhaseNth   BillingPhase = "nth"
)
