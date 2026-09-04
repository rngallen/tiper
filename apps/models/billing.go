package models

import (
	"time"

	"dfms/pkg/types"
	"dfms/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// FcfFeeBatch is an approved FCF price list (content type 61).
// Header holds the TZS/USD quote; lines are pricing models (tender/route/collection).
// Billing picks the latest approved batch whose EffectiveFrom ≤ the receipt billing date.
type FcfFeeBatch struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueFcfUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:61;not null;check:ContentType=61" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueFcfNo" json:"documentNumber"`
	Date           time.Time            `gorm:"not null;index:idx_fcfBatchDateStatus,priority:2" json:"date"`
	EffectiveFrom  time.Time            `gorm:"not null;index:idx_fcfApproved,priority:2" json:"effectiveFrom"`
	Description    string               `gorm:"type:nvarchar(200)" json:"description"`
	CurrencyCode   string               `gorm:"type:varchar(3);not null" json:"currencyCode"`
	ExchangeRate   decimal.Decimal      `gorm:"type:decimal(18,6);not null;default:0" json:"exchangeRate"`
	FxManual       bool                 `gorm:"default:0;not null" json:"fxManual"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_fcfBatchDateStatus,priority:1;index:idx_fcfApproved,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
	Lines          []FcfFee             `gorm:"foreignKey:BatchID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
}

func (m *FcfFeeBatch) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

func (m *FcfFeeBatch) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

// FcfFee is one class-of-trade pricing model: first billing (on confirmed
// reception) plus repeating second/nth billing until the parcel is finished.
type FcfFee struct {
	ID                uint                   `gorm:"primaryKey" json:"-"`
	UID               string                 `gorm:"type:varchar(26);uniqueIndex:idx_uniqueFcfLineUID;not null" json:"id"`
	BatchID           uint                   `gorm:"index;not null" json:"-"`
	ClassOfTrade      string                 `gorm:"type:varchar(60);not null;index" json:"classOfTrade"`
	ProcurementMethod types.ProcurementCode  `gorm:"type:varchar(20);not null" json:"procurementMethod"`
	DischargeRoute    types.DischargeRoute   `gorm:"type:varchar(20);not null;index" json:"dischargeRoute"`
	CollectionMethod  types.CollectionMethod `gorm:"type:varchar(20)" json:"collectionMethod"`
	IsPromotional     bool                   `gorm:"default:0;not null" json:"isPromotional"`

	FirstDays               int             `gorm:"not null;default:15" json:"firstDays"`
	FirstChargeTo           types.ChargeTo  `gorm:"type:varchar(20);not null;default:customer" json:"firstChargeTo"`
	FirstRateKind           types.RateKind  `gorm:"type:varchar(12);not null;default:flat" json:"firstRateKind"`
	FirstUnit               string          `gorm:"type:varchar(10);not null;default:M3" json:"firstUnit"`
	FirstSourceCurrencyCode string          `gorm:"type:varchar(3);not null" json:"firstSourceCurrencyCode"`
	FirstSourcePrice        decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"firstSourcePrice"`
	FirstHomePrice          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"firstHomePrice"`

	NthDays               int             `gorm:"not null;default:30" json:"nthDays"`
	NthRateKind           types.RateKind  `gorm:"type:varchar(12);not null;default:flat" json:"nthRateKind"`
	NthUnit               string          `gorm:"type:varchar(10);not null;default:M3" json:"nthUnit"`
	NthSourceCurrencyCode string          `gorm:"type:varchar(3);not null" json:"nthSourceCurrencyCode"`
	NthSourcePrice        decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"nthSourcePrice"`
	NthHomePrice          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"nthHomePrice"`

	Tiers []FcfFeeTier `gorm:"foreignKey:FeeID;constraint:OnDelete:NO ACTION;" json:"tiers,omitempty"`
}

func (m *FcfFee) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// FcfFeeTier is one volume slab on an FCF pricing model (whole-volume, not progressive).
type FcfFeeTier struct {
	ID          uint             `gorm:"primaryKey" json:"-"`
	UID         string           `gorm:"type:varchar(26);uniqueIndex:idx_uniqueFcfTierUID;not null" json:"id"`
	FeeID       uint             `gorm:"index;not null" json:"-"`
	Phase       string           `gorm:"type:varchar(8);not null;index" json:"phase"`
	FromQty     decimal.Decimal  `gorm:"type:decimal(18,3);not null;default:0" json:"fromQty"`
	ToQty       *decimal.Decimal `gorm:"type:decimal(18,3)" json:"toQty,omitempty"`
	SourcePrice decimal.Decimal  `gorm:"type:decimal(18,4);not null" json:"sourcePrice"`
	HomePrice   decimal.Decimal  `gorm:"type:decimal(18,4);not null;default:0" json:"homePrice"`
}

func (m *FcfFeeTier) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// ExchangeRate is a dated FX rate (workflow-approved).
type ExchangeRate struct {
	ID            uint                 `gorm:"primaryKey" json:"-"`
	UID           string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueFXUID;not null" json:"id"`
	ContentType   types.ContentType    `gorm:"default:78;not null;check:ContentType=78" json:"-"`
	EffectiveFrom time.Time            `gorm:"not null;index;index:idx_fxApproved,priority:2" json:"effectiveFrom"`
	FromCurrency  string               `gorm:"type:varchar(3);not null" json:"fromCurrency"`
	ToCurrency    string               `gorm:"type:varchar(3);not null" json:"toCurrency"`
	Rate          decimal.Decimal      `gorm:"type:decimal(18,6);not null" json:"rate"`
	Status        types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_fxApproved,priority:1" json:"status"`
	CreatedByID   uint                 `gorm:"index;not null" json:"-"`
	CreatedBy     User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator       *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
}

func (m *ExchangeRate) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *ExchangeRate) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

type MiLossBatch struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueMiLossUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:62;not null;check:ContentType=62" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueMiLossNo" json:"documentNumber"`
	Date           time.Time            `gorm:"not null;index:idx_milossDate" json:"date"`
	EffectiveFrom  time.Time            `gorm:"not null;index:idx_milossApproved,priority:2" json:"effectiveFrom"`
	Description    string               `gorm:"type:nvarchar(200)" json:"description"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_milossStatus;index:idx_milossList,priority:1;index:idx_milossApproved,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `gorm:"index:idx_milossCreatedAt;index:idx_milossList,priority:2" json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
	Products       []MiLossProduct      `gorm:"foreignKey:BatchID;constraint:OnDelete:NO ACTION;" json:"products,omitempty"`
	Lines          []MiLoss             `gorm:"-" json:"lines,omitempty"`
}

func (m *MiLossBatch) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

func (m *MiLossBatch) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	m.flattenRates()
	return nil
}

func (m *MiLossBatch) flattenRates() {
	if len(m.Lines) > 0 || len(m.Products) == 0 {
		return
	}
	for i := range m.Products {
		p := &m.Products[i]
		for j := range p.Rates {
			r := p.Rates[j]
			if r.Product == nil {
				r.Product = p.Product
			}
			m.Lines = append(m.Lines, r)
		}
	}
}

// MiLossProduct is one product on an MI-loss batch. Unique per batch so AGO
// appears once in the header, then many contracts under Rates.
type MiLossProduct struct {
	ID         uint     `gorm:"primaryKey" json:"-"`
	UID        string   `gorm:"type:varchar(26);uniqueIndex:idx_uniqueMiLossProductUID;not null" json:"id"`
	BatchID    uint     `gorm:"uniqueIndex:idx_uniqueMiLossProduct,priority:1;not null" json:"-"`
	ProductID  uint     `gorm:"uniqueIndex:idx_uniqueMiLossProduct,priority:2;not null" json:"-"`
	ProductUID string   `gorm:"-" json:"productId,omitempty"`
	Product    *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	Rates      []MiLoss `gorm:"foreignKey:ProductRowID;constraint:OnDelete:NO ACTION;" json:"rates,omitempty"`
}

func (m *MiLossProduct) AfterFind(*gorm.DB) error {
	if m.Product != nil {
		m.ProductUID = m.Product.UID
	}
	return nil
}

func (m *MiLossProduct) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

// MiLoss is one contract rate on a batch product (0.005 = 0.5%).
// AGO / SRT and AGO / TOP may both exist on the same batch. Across batches
// the product × contract × effective date triple is unique.
type MiLoss struct {
	ID               uint               `gorm:"primaryKey" json:"-"`
	UID              string             `gorm:"type:varchar(26);uniqueIndex:idx_uniqueMiLossLineUID;not null" json:"id"`
	ProductRowID     uint               `gorm:"uniqueIndex:idx_uniqueMiLossLine,priority:1;not null" json:"-"`
	BatchID          uint               `gorm:"index;not null" json:"-"`
	ProductID        uint               `gorm:"uniqueIndex:idx_uniqueMiLossRate,priority:1;not null" json:"-"`
	Product          *Product           `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ContractTypeCode types.ContractCode `gorm:"uniqueIndex:idx_uniqueMiLossLine,priority:2;uniqueIndex:idx_uniqueMiLossRate,priority:2;type:varchar(20);not null" json:"contractTypeCode"`
	EffectiveFrom    time.Time          `gorm:"not null;index;uniqueIndex:idx_uniqueMiLossRate,priority:3" json:"effectiveFrom"`
	Value            decimal.Decimal    `gorm:"type:decimal(12,6);not null" json:"value"`
}

func (m *MiLoss) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

type VariableFeeBatch struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueVarFeeUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:63;not null;check:ContentType=63" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueVarFeeNo" json:"documentNumber"`
	Date           time.Time            `gorm:"not null;index:idx_varFeeDate;index:idx_varFeeList,priority:2" json:"date"`
	EffectiveFrom  time.Time            `gorm:"not null;index:idx_varFeeApproved,priority:2" json:"effectiveFrom"`
	Description    string               `gorm:"type:nvarchar(200)" json:"description"`
	CurrencyCode   string               `gorm:"type:varchar(3);not null;default:USD" json:"currencyCode"`
	ExchangeRate   decimal.Decimal      `gorm:"type:decimal(18,6);not null" json:"exchangeRate"`
	FxManual       bool                 `gorm:"default:0;not null" json:"fxManual"`
	MiLossBatchID  *uint                `gorm:"index" json:"-"`
	MiLossBatch    *MiLossBatch         `gorm:"foreignKey:MiLossBatchID;constraint:OnDelete:NO ACTION;" json:"miLossBatch,omitempty"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_varFeeStatus;index:idx_varFeeList,priority:1;index:idx_varFeeApproved,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	Products       []ProductConfig      `gorm:"foreignKey:BatchID;constraint:OnDelete:NO ACTION;" json:"products,omitempty"`
}

func (m *VariableFeeBatch) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

func (m *VariableFeeBatch) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

type ProductConfig struct {
	ID          uint                  `gorm:"primaryKey" json:"-"`
	UID         string                `gorm:"type:varchar(26);uniqueIndex:idx_uniqueVarProductUID;not null" json:"id"`
	BatchID     uint                  `gorm:"uniqueIndex:idx_uniqueVarProduct,priority:1;not null" json:"-"`
	ProductID   uint                  `gorm:"uniqueIndex:idx_uniqueVarProduct,priority:2;not null" json:"-"`
	Product     *Product              `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	EwuraPrice  decimal.Decimal       `gorm:"type:decimal(18,4);not null" json:"ewuraPrice"`
	Density     decimal.Decimal       `gorm:"type:decimal(12,6);not null" json:"density"`
	WholesaleCM decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0" json:"wholesaleCm"`
	WholesaleMT decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0" json:"wholesaleMt"`
	Contracts   []ProductContractRate `gorm:"foreignKey:ProductConfigID;constraint:OnDelete:NO ACTION;" json:"contracts,omitempty"`
}

func (m *ProductConfig) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// ProductContractRate is MI-loss and computed VCF for one product × contract type.
type ProductContractRate struct {
	ID               uint               `gorm:"primaryKey" json:"-"`
	UID              string             `gorm:"type:varchar(26);uniqueIndex:idx_uniqueVarRateUID;not null" json:"id"`
	ProductConfigID  uint               `gorm:"index;not null" json:"-"`
	BatchID          uint               `gorm:"uniqueIndex:idx_uniqueVarRate,priority:1;not null" json:"-"`
	ProductID        uint               `gorm:"uniqueIndex:idx_uniqueVarRate,priority:2;not null" json:"-"`
	ContractTypeCode types.ContractCode `gorm:"uniqueIndex:idx_uniqueVarRate,priority:3;type:varchar(20);not null" json:"contractTypeCode"`
	MiLossValue      decimal.Decimal    `gorm:"type:decimal(12,6);not null;default:0" json:"miLossValue"`
	FeeUSDCM         decimal.Decimal    `gorm:"type:decimal(18,4);not null;default:0" json:"feeUsdCm"`
	FeeUSDMT         decimal.Decimal    `gorm:"type:decimal(18,4);not null;default:0" json:"feeUsdMt"`
	FeeTZSCM         decimal.Decimal    `gorm:"type:decimal(18,4);not null;default:0" json:"feeTzsCm"`
	FeeTZSMT         decimal.Decimal    `gorm:"type:decimal(18,4);not null;default:0" json:"feeTzsMt"`
}

func (m *ProductContractRate) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

type KojFeeBatch struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueKojUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:64;not null;check:ContentType=64" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueKojNo" json:"documentNumber"`
	Date           time.Time            `gorm:"not null;index:idx_kojBatchDateStatus,priority:2" json:"date"`
	EffectiveFrom  time.Time            `gorm:"not null;index:idx_kojApproved,priority:2" json:"effectiveFrom"`
	Description    string               `gorm:"type:nvarchar(200)" json:"description"`
	CurrencyCode   string               `gorm:"type:varchar(3);not null" json:"currencyCode"`
	ExchangeRate   decimal.Decimal      `gorm:"type:decimal(18,6);not null;default:0" json:"exchangeRate"`
	FxManual       bool                 `gorm:"default:0;not null" json:"fxManual"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_kojBatchDateStatus,priority:1;index:idx_kojApproved,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	Fees           []KojFee             `gorm:"foreignKey:BatchID;constraint:OnDelete:NO ACTION;" json:"fees,omitempty"`
}

func (m *KojFeeBatch) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

func (m *KojFeeBatch) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

type KojFee struct {
	ID                 uint            `gorm:"primaryKey" json:"-"`
	UID                string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueKojLineUID;not null" json:"id"`
	BatchID            uint            `gorm:"index;not null" json:"-"`
	ProductID          uint            `gorm:"uniqueIndex:idx_uniqueKojRate,priority:1;not null" json:"-"`
	Product            *Product        `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	Unit               string          `gorm:"type:varchar(10);uniqueIndex:idx_uniqueKojRate,priority:2;not null" json:"unit"`
	SourceCurrencyCode string          `gorm:"type:varchar(3);uniqueIndex:idx_uniqueKojRate,priority:3;not null" json:"sourceCurrencyCode"`
	EffectiveFrom      time.Time       `gorm:"uniqueIndex:idx_uniqueKojRate,priority:4;not null" json:"effectiveFrom"`
	SourcePrice        decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"sourcePrice"`
	HomePrice          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"homePrice"`
}

func (m *KojFee) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

type TbsFeeBatch struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTbsUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:65;not null;check:ContentType=65" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueTbsNo" json:"documentNumber"`
	Date           time.Time            `gorm:"not null;index:idx_tbsBatchDateStatus,priority:2" json:"date"`
	EffectiveFrom  time.Time            `gorm:"not null;index:idx_tbsApproved,priority:2" json:"effectiveFrom"`
	Description    string               `gorm:"type:nvarchar(200)" json:"description"`
	CurrencyCode   string               `gorm:"type:varchar(3);not null" json:"currencyCode"`
	ExchangeRate   decimal.Decimal      `gorm:"type:decimal(18,6);not null;default:0" json:"exchangeRate"`
	FxManual       bool                 `gorm:"default:0;not null" json:"fxManual"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_tbsBatchDateStatus,priority:1;index:idx_tbsApproved,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	Fees           []TbsFee             `gorm:"foreignKey:BatchID;constraint:OnDelete:NO ACTION;" json:"fees,omitempty"`
}

func (m *TbsFeeBatch) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

func (m *TbsFeeBatch) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

type TbsFee struct {
	ID                 uint            `gorm:"primaryKey" json:"-"`
	UID                string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTbsLineUID;not null" json:"id"`
	BatchID            uint            `gorm:"index;not null" json:"-"`
	ProductID          uint            `gorm:"uniqueIndex:idx_uniqueTbsRate,priority:1;not null" json:"-"`
	Product            *Product        `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	Unit               string          `gorm:"type:varchar(10);uniqueIndex:idx_uniqueTbsRate,priority:2;not null" json:"unit"`
	SourceCurrencyCode string          `gorm:"type:varchar(3);uniqueIndex:idx_uniqueTbsRate,priority:3;not null" json:"sourceCurrencyCode"`
	EffectiveFrom      time.Time       `gorm:"uniqueIndex:idx_uniqueTbsRate,priority:4;not null" json:"effectiveFrom"`
	SourcePrice        decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"sourcePrice"`
	HomePrice          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"homePrice"`
}

func (m *TbsFee) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

type BillingRun struct {
	ID              uint                 `gorm:"primaryKey" json:"-"`
	UID             string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueBillRunUID;not null" json:"id"`
	ContentType     types.ContentType    `gorm:"default:66;not null;check:ContentType=66" json:"-"`
	DocumentNumber  string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueBillRunNo" json:"documentNumber"`
	ReceiptDetailID *uint                `gorm:"index" json:"-"`
	CustomerID      *uint                `gorm:"index" json:"-"`
	SupplierID      *uint                `gorm:"index" json:"-"`
	FeeCode         types.FeeCode        `gorm:"type:varchar(10);not null;index;index:idx_billReport,priority:2;index:idx_billRunDue,priority:1" json:"feeCode"`
	BillingSequence int                  `gorm:"not null;default:1" json:"billingSequence"`
	PeriodStart     time.Time            `gorm:"not null;index;index:idx_billReport,priority:1" json:"periodStart"`
	PeriodEnd       time.Time            `gorm:"not null;index:idx_billRunDue,priority:3" json:"periodEnd"`
	ChargeTo        types.ChargeTo       `gorm:"type:varchar(20);not null" json:"chargeTo"`
	CurrencyCode    string               `gorm:"type:varchar(3);not null" json:"currencyCode"`
	Quantity        decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"quantity"`
	Rate            decimal.Decimal      `gorm:"type:decimal(18,4);not null;default:0" json:"rate"`
	Amount          decimal.Decimal      `gorm:"type:decimal(18,2);not null;default:0" json:"amount"`
	Status          types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index;index:idx_billRunDue,priority:2;index:idx_billRunList,priority:1" json:"status"`
	Source          types.BillingSource  `gorm:"type:varchar(30)" json:"source"`
	ExceptionReason string               `gorm:"type:nvarchar(300)" json:"exceptionReason,omitempty"`
	CreatedByID     uint                 `gorm:"index;not null" json:"-"`
	CreatedBy       User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator         *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt       time.Time            `gorm:"index:idx_billRunCreatedAt;index:idx_billRunList,priority:2" json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
	Lines           []ChargeLine         `gorm:"foreignKey:BillingRunID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
}

func (m *BillingRun) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *BillingRun) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

type ChargeLine struct {
	ID                  uint            `gorm:"primaryKey" json:"-"`
	UID                 string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueChargeLineUID;not null" json:"id"`
	BillingRunID        uint            `gorm:"index;not null" json:"-"`
	FeeCode             types.FeeCode   `gorm:"type:varchar(10);not null" json:"feeCode"`
	InventoryEventLogID *uint           `gorm:"index" json:"-"`
	Quantity            decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"quantity"`
	Rate                decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"rate"`
	Amount              decimal.Decimal `gorm:"type:decimal(18,2);not null" json:"amount"`
	CurrencyCode        string          `gorm:"type:varchar(3);not null" json:"currencyCode"`
	TruckReference      string          `gorm:"type:varchar(40)" json:"truckReference"`
}

func (m *ChargeLine) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

type BillingException struct {
	ID          uint                 `gorm:"primaryKey" json:"-"`
	UID         string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueBillExUID;not null" json:"id"`
	ContentType types.ContentType    `gorm:"default:73;not null;check:ContentType=73" json:"-"`
	RunID       *uint                `gorm:"index" json:"-"`
	Reason      string               `gorm:"type:nvarchar(400);not null" json:"reason"`
	ValidUntil  time.Time            `gorm:"not null" json:"validUntil"`
	Status      types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft" json:"status"`
	CreatedByID uint                 `gorm:"index;not null" json:"-"`
	CreatedAt   time.Time            `json:"createdAt"`
}

func (m *BillingException) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

type SagePostingLog struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	ContentType  types.ContentType `gorm:"default:76;not null;check:ContentType=76" json:"-"`
	BillingRunID uint              `gorm:"index;not null" json:"-"`
	AttemptedAt  time.Time         `gorm:"not null" json:"attemptedAt"`
	Success      bool              `gorm:"not null" json:"success"`
	Reference    string            `gorm:"type:varchar(80)" json:"reference"`
	Error        string            `gorm:"type:nvarchar(500)" json:"error"`
}

type ReportSnapshot struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueSnapUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:70;not null;check:ContentType=70" json:"-"`
	ReportCode  string            `gorm:"type:varchar(60);not null;index" json:"reportCode"`
	PeriodStart time.Time         `gorm:"not null" json:"periodStart"`
	PeriodEnd   time.Time         `gorm:"not null" json:"periodEnd"`
	Payload     JSONMap           `gorm:"type:nvarchar(max)" json:"payload"`
	CreatedByID uint              `gorm:"index;not null" json:"-"`
	CreatedAt   time.Time         `json:"createdAt"`
}

func (m *ReportSnapshot) BeforeCreate(*gorm.DB) error {
	if m.UID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.UID = uid
	return nil
}

// ChangeOfService switches delivery method on one customer parcel
// (vessel + vessel date). Fees stay mapped; FCF billing follows the new method.
type ChangeOfService struct {
	ID              uint                   `gorm:"primaryKey" json:"-"`
	UID             string                 `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCOSUID;not null" json:"id"`
	ContentType     types.ContentType      `gorm:"default:71;not null;check:ContentType=71" json:"-"`
	DocumentNumber  string                 `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueCOSNo" json:"documentNumber"`
	EffectiveDate   time.Time              `gorm:"not null;index:idx_cosList,priority:2" json:"effectiveDate"`
	CustomerID      uint                   `gorm:"index;index:idx_cosCustomer;not null" json:"-"`
	Customer        Customer               `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer"`
	ReceiptDetailID uint                   `gorm:"index;not null" json:"-"`
	ReceiptDetail   ReceiptDetail          `gorm:"foreignKey:ReceiptDetailID;constraint:OnDelete:NO ACTION;" json:"-"`
	ParcelID        string                 `gorm:"-" json:"parcelId,omitempty"`
	VesselID        uint                   `gorm:"index;not null" json:"-"`
	Vessel          Vessel                 `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel"`
	VesselDate      time.Time              `gorm:"not null;index:idx_cosVesselDate" json:"vesselDate"`
	ProductID       uint                   `gorm:"index;not null" json:"-"`
	Product         Product                `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	FromCollection  types.CollectionMethod `gorm:"type:varchar(20);not null" json:"fromCollection"`
	ToCollection    types.CollectionMethod `gorm:"type:varchar(20);not null" json:"toCollection"`
	Notes           string                 `gorm:"type:nvarchar(400)" json:"notes"`
	Status          types.DocumentStatus   `gorm:"type:varchar(20);not null;default:draft;index:idx_cosList,priority:1" json:"status"`
	CreatedByID     uint                   `gorm:"index;not null" json:"-"`
	CreatedBy       User                   `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator         *CreatedByRef          `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	DjangoID        uint                   `gorm:"index" json:"-"`
	ApprovalTrail   ApprovalTrail          `gorm:"type:nvarchar(max)" json:"-"`
}

func (m *ChangeOfService) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *ChangeOfService) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *ChangeOfService) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	MarkHasData(tx, &Vessel{}, m.VesselID)
	return nil
}

func (m ChangeOfService) TableName() string { return "ChangeOfService" }
