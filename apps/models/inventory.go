package models

import (
	"time"

	"dfms/pkg/types"
	"dfms/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Receipt is a vessel discharge (stock in).
type Receipt struct {
	ID                    uint                  `gorm:"primaryKey" json:"-"`
	UID                   string                `gorm:"type:varchar(26);uniqueIndex:idx_uniqueReceiptUID;not null" json:"id"`
	ContentType           types.ContentType     `gorm:"default:51;not null;check:ContentType=51" json:"-"`
	DocumentNumber        string                `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueReceiptNo" json:"documentNumber"`
	Date                  time.Time             `gorm:"not null;index:idx_receiptDate;index:idx_receiptReport,priority:3;index:idx_receiptList,priority:2" json:"date"`
	VesselDate            time.Time             `gorm:"not null;index:idx_receiptVessel" json:"vesselDate"`
	BillingEffectiveDate  time.Time             `gorm:"not null;index" json:"billingEffectiveDate"`
	VesselID              uint                  `gorm:"index:idx_receiptVessel;not null" json:"-"`
	Vessel                Vessel                `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel"`
	SupplierID            *uint                 `gorm:"index" json:"-"`
	Supplier              *Supplier             `gorm:"foreignKey:SupplierID;constraint:OnDelete:NO ACTION;" json:"supplier,omitempty"`
	ProductID             uint                  `gorm:"index;not null" json:"-"`
	Product               Product               `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	RouteCode             types.DischargeRoute  `gorm:"type:varchar(20);not null;index:idx_receiptReport,priority:1;index:idx_receiptRoute" json:"routeCode"`
	TenderCode            types.TenderCode      `gorm:"type:varchar(20);not null;default:'';index:idx_receiptTender" json:"tenderCode"`
	ProcurementMethodCode types.ProcurementCode `gorm:"type:varchar(20);not null;default:''" json:"procurementMethodCode"`
	ReceiptType           types.ReceiptType     `gorm:"type:varchar(20);not null;index:idx_receiptType" json:"receiptType"`
	// UsesTiperPipeline: this KOJ external session used TIPER's 10-inch line.
	// One receipt, one channel — the whole session takes the KOJ service fee.
	UsesTiperPipeline bool            `gorm:"default:0;not null" json:"usesTiperPipeline"`
	Density           decimal.Decimal `gorm:"type:decimal(12,6);not null;default:0" json:"density"`
	// Tank* is the measured volume in TIPER tanks. LineLoss* is Tank − outturn
	// (signed). Reception + line loss = actual received on each parcel.
	TankQuantity        decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"tankQuantity"`
	TankCubicMeter      decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"tankCubicMeter"`
	TankMetricTonne     decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"tankMetricTonne"`
	LineLoss            decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"lineLoss"`
	LineLossCubicMeter  decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossCubicMeter"`
	LineLossMetricTonne decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossMetricTonne"`
	IsProvision         bool                `gorm:"default:0;not null;index" json:"isProvision"`
	IsFinal             bool                `gorm:"default:0;not null" json:"isFinal"`
	ProvisionReceiptID  *uint               `gorm:"index" json:"-"`
	Status              types.ReceiptStatus `gorm:"type:varchar(20);not null;default:draft;index;index:idx_receiptReport,priority:2;index:idx_receiptList,priority:1" json:"status"`
	Notes               string              `gorm:"type:nvarchar(500)" json:"notes"`
	CreatedByID         uint                `gorm:"index;not null" json:"-"`
	CreatedBy           User                `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator             *CreatedByRef       `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt           time.Time           `json:"createdAt"`
	UpdatedAt           time.Time           `json:"updatedAt"`

	Details []ReceiptDetail `gorm:"foreignKey:ReceiptID;constraint:OnDelete:NO ACTION;" json:"details,omitempty"`
}

func (m *Receipt) BeforeCreate(*gorm.DB) error {
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

func (m *Receipt) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *Receipt) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Vessel{}, m.VesselID)
	markHasDataPtr(tx, &Supplier{}, m.SupplierID)
	MarkHasData(tx, &Product{}, m.ProductID)
	return nil
}

// ReceiptDetail is one customer parcel on a vessel.
type ReceiptDetail struct {
	ID               uint                   `gorm:"primaryKey" json:"-"`
	UID              string                 `gorm:"type:varchar(26);uniqueIndex:idx_uniqueReceiptDetailUID;not null" json:"id"`
	ContentType      types.ContentType      `gorm:"default:52;not null;check:ContentType=52" json:"-"`
	ReceiptID        uint                   `gorm:"index:idx_receiptDetailHdr;not null" json:"-"`
	Receipt          Receipt                `gorm:"foreignKey:ReceiptID;constraint:OnDelete:NO ACTION;" json:"-"`
	CustomerID       uint                   `gorm:"index;not null" json:"-"`
	Customer         Customer               `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer"`
	StockStatusID    *uint                  `gorm:"index" json:"-"`
	StockStatus      *StockStatus           `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus,omitempty"`
	DepotID          *uint                  `gorm:"index" json:"-"`
	Depot            *Depot                 `gorm:"foreignKey:DepotID;constraint:OnDelete:NO ACTION;" json:"depot,omitempty"`
	CollectionMethod types.CollectionMethod `gorm:"type:varchar(20);not null;default:''" json:"collectionMethod"`
	ContractTypeCode types.ContractCode     `gorm:"type:varchar(20)" json:"contractTypeCode"`
	PricingNature    types.PricingNature    `gorm:"type:varchar(30)" json:"pricingNature"`
	// NextBillingDays is the first-cycle length from BillingCycle (not collection).
	NextBillingDays     int             `gorm:"not null;default:15" json:"nextBillingDays"`
	FinancialHold       bool            `gorm:"default:0;not null" json:"financialHold"`
	HoldQuantity        decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"holdQuantity"`
	Density             decimal.Decimal `gorm:"type:decimal(12,6);not null;default:0" json:"density"`
	Quantity            decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter          decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne         decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	LineLoss            decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"lineLoss"`
	LineLossCubicMeter  decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossCubicMeter"`
	LineLossMetricTonne decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossMetricTonne"`
	IsProvision         bool            `gorm:"default:0;not null" json:"isProvision"`
	IsArchived          bool            `gorm:"default:0;not null" json:"isArchived"`
	// OriginDetailID marks an ITT billing child. The sender's original receipt
	// line is never reduced — history stays as received. EffectiveFrom is the
	// approval instant; stock movements post on that date so reports before it
	// still show the sender's full volume.
	OriginDetailID *uint      `gorm:"index" json:"originId,omitempty"`
	EffectiveFrom  *time.Time `gorm:"index" json:"effectiveFrom,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (m *ReceiptDetail) BeforeCreate(*gorm.DB) error {
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

func (m *ReceiptDetail) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	markHasDataPtr(tx, &StockStatus{}, m.StockStatusID)
	markHasDataPtr(tx, &Depot{}, m.DepotID)
	return nil
}

// ReceivedQuantity is outturn + allocated line loss (tank figure for the parcel).
func (d ReceiptDetail) ReceivedQuantity() decimal.Decimal {
	return d.Quantity.Add(d.LineLoss)
}

func (d ReceiptDetail) ReceivedCubicMeter() decimal.Decimal {
	return d.CubicMeter.Add(d.LineLossCubicMeter)
}

func (d ReceiptDetail) ReceivedMetricTonne() decimal.Decimal {
	return d.MetricTonne.Add(d.LineLossMetricTonne)
}

// StatusID is the stock-status PK, or 0 when the line has none (external receipts).
func (d ReceiptDetail) StatusID() uint {
	if d.StockStatusID == nil {
		return 0
	}
	return *d.StockStatusID
}

// ReceptionFact is one approved receipt line, denormalized for reception reports.
// Internal and external rows are written on approval so SBM/KOJ share and
// market-share queries do not join live Receipt + ReceiptDetail.
type ReceptionFact struct {
	ID                  uint                 `gorm:"primaryKey" json:"-"`
	ContentType         types.ContentType    `gorm:"default:105;not null;check:ContentType=105" json:"-"`
	ReceiptID           uint                 `gorm:"index;not null" json:"-"`
	ReceiptDetailID     uint                 `gorm:"uniqueIndex:idx_uniqueReceptionFactDetail;not null" json:"-"`
	DocumentNumber      string               `gorm:"type:varchar(40);not null;index" json:"documentNumber"`
	Date                time.Time            `gorm:"type:date;not null;index:idx_receptionFactPeriod,priority:1;index:idx_receptionFactRoute,priority:2" json:"date"`
	VesselDate          time.Time            `gorm:"type:date;not null;index:idx_receptionFactVessel" json:"vesselDate"`
	RouteCode           types.DischargeRoute `gorm:"type:varchar(20);not null;index:idx_receptionFactPeriod,priority:2;index:idx_receptionFactRoute,priority:1" json:"routeCode"`
	ReceiptType         types.ReceiptType    `gorm:"type:varchar(20);not null;index:idx_receptionFactPeriod,priority:3" json:"receiptType"`
	VesselID            uint                 `gorm:"not null;index:idx_receptionFactVessel" json:"-"`
	VesselCode          string               `gorm:"type:varchar(40);not null" json:"vesselCode"`
	VesselName          string               `gorm:"type:nvarchar(120);not null" json:"vesselName"`
	ProductID           uint                 `gorm:"not null;index" json:"-"`
	ProductCode         string               `gorm:"type:varchar(40);not null" json:"productCode"`
	ProductName         string               `gorm:"type:nvarchar(120);not null" json:"productName"`
	CustomerID          uint                 `gorm:"not null;index" json:"-"`
	CustomerCode        string               `gorm:"type:varchar(40);not null" json:"customerCode"`
	CustomerName        string               `gorm:"type:nvarchar(200);not null" json:"customerName"`
	DepotID             *uint                `gorm:"index" json:"-"`
	DepotCode           string               `gorm:"type:varchar(40)" json:"depotCode"`
	DepotName           string               `gorm:"type:nvarchar(120);not null" json:"depotName"`
	UsesTiperPipeline   bool                 `gorm:"default:0;not null" json:"usesTiperPipeline"`
	FinancialHold       bool                 `gorm:"default:0;not null" json:"financialHold"`
	TenderCode          types.TenderCode     `gorm:"type:varchar(20)" json:"tenderCode"`
	Quantity            decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"quantity"`
	CubicMeter          decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne         decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	LineLoss            decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"lineLoss"`
	LineLossCubicMeter  decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossCubicMeter"`
	LineLossMetricTonne decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"lineLossMetricTonne"`
	CreatedAt           time.Time            `json:"createdAt"`
	UpdatedAt           time.Time            `json:"updatedAt"`
}

// StockMovement is the append-only book ledger.
type StockMovement struct {
	ID              uint              `gorm:"primaryKey" json:"-"`
	UID             string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueMovementUID;not null" json:"id"`
	ContentType     types.ContentType `gorm:"default:54;not null;check:ContentType=54" json:"-"`
	TransactionDate time.Time         `gorm:"not null;index:idx_movementDate;index:idx_movementReport,priority:2;index:idx_movementTypeDate,priority:2" json:"transactionDate"`
	TransactionType types.TxnType     `gorm:"type:varchar(30);not null;index:idx_movementType;index:idx_movementTypeDate,priority:1" json:"transactionType"`
	CustomerID      uint              `gorm:"index:idx_movementBalance,priority:1;index:idx_movementReport,priority:3;not null" json:"-"`
	ProductID       uint              `gorm:"index:idx_movementBalance,priority:2;index:idx_movementReport,priority:4;not null" json:"-"`
	VesselID        uint              `gorm:"index:idx_movementBalance,priority:3;not null" json:"-"`
	VesselDate      time.Time         `gorm:"index:idx_movementBalance,priority:4;not null" json:"vesselDate"`
	StockStatusID   uint              `gorm:"index:idx_movementBalance,priority:5;not null" json:"-"`
	DepotID         *uint             `gorm:"index" json:"-"`
	Quantity        decimal.Decimal   `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter      decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	IsProvision     bool              `gorm:"default:0;not null;index;index:idx_movementReport,priority:1;index:idx_movementTypeDate,priority:3" json:"isProvision"`
	FinancialHold   bool              `gorm:"default:0;not null" json:"financialHold"`
	ReferenceType   string            `gorm:"type:varchar(40);index:idx_movementRef" json:"referenceType"`
	ReferenceID     uint              `gorm:"index:idx_movementRef" json:"-"`
	CreatedAt       time.Time         `json:"createdAt"`
}

func (m *StockMovement) BeforeCreate(*gorm.DB) error {
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

// StockBalance is the live on-hand grain (customer × product × vessel × status × hold).
// Updated in the same transaction as each StockMovement so reports do not scan the ledger.
type StockBalance struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	ContentType   types.ContentType `gorm:"default:100;not null;check:ContentType=100" json:"-"`
	CustomerID    uint              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:1;index:idx_stockBalCust;not null" json:"-"`
	ProductID     uint              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:2;index:idx_stockBalProd;not null" json:"-"`
	VesselID      uint              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:3;not null" json:"-"`
	VesselDate    time.Time         `gorm:"type:date;uniqueIndex:idx_uniqueStockBal,priority:4;not null" json:"vesselDate"`
	StockStatusID uint              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:5;not null" json:"-"`
	FinancialHold bool              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:6;not null;index:idx_stockBalHold" json:"financialHold"`
	IsProvision   bool              `gorm:"uniqueIndex:idx_uniqueStockBal,priority:7;not null" json:"isProvision"`
	DepotID       *uint             `gorm:"index" json:"-"`
	Quantity      decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"quantity"`
	CubicMeter    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne   decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	UpdatedAt     time.Time         `json:"updatedAt"`

	Customer    Customer    `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"-"`
	Product     Product     `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"-"`
	Vessel      Vessel      `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"-"`
	StockStatus StockStatus `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"-"`
}

// StockDailyPosition is end-of-day book by customer × product × status.
// Period reports read this instead of aggregating StockMovement.
type StockDailyPosition struct {
	ID             uint              `gorm:"primaryKey" json:"-"`
	ContentType    types.ContentType `gorm:"default:101;not null;check:ContentType=101" json:"-"`
	PositionDate   time.Time         `gorm:"type:date;uniqueIndex:idx_uniqueDailyPos,priority:1;index:idx_dailyPosDate;index:idx_dailyPosAsOf,priority:3;not null" json:"positionDate"`
	CustomerID     uint              `gorm:"uniqueIndex:idx_uniqueDailyPos,priority:2;index:idx_dailyPosCust;index:idx_dailyPosAsOf,priority:1;not null" json:"-"`
	ProductID      uint              `gorm:"uniqueIndex:idx_uniqueDailyPos,priority:3;index:idx_dailyPosAsOf,priority:2;not null" json:"-"`
	StockStatusID  uint              `gorm:"uniqueIndex:idx_uniqueDailyPos,priority:4;not null" json:"-"`
	ClosingQty     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"closingQty"`
	ReceivedQty    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_recv,[ReceivedQty] >= 0" json:"receivedQty"`
	OutflowQty     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_out,[OutflowQty] >= 0" json:"outflowQty"`
	LoadingQty     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_load,[LoadingQty] >= 0" json:"loadingQty"`
	PumpOverQty    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_pump,[PumpOverQty] >= 0" json:"pumpOverQty"`
	ITTQty         decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_itt,[ITTQty] >= 0" json:"ittQty"`
	AdjustmentQty  decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"adjustmentQty"`
	HoldQty        decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"holdQty"`
	SRTReceivedQty decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_daily_srt,[SRTReceivedQty] >= 0" json:"srtReceivedQty"`
	UpdatedAt      time.Time         `json:"updatedAt"`

	Customer    Customer    `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"-"`
	Product     Product     `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"-"`
	StockStatus StockStatus `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"-"`
}

// ProductDailyBalance is end-of-day book vs physical (dips + line content) by product.
type ProductDailyBalance struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	ContentType types.ContentType `gorm:"default:102;not null;check:ContentType=102" json:"-"`
	BalanceDate time.Time         `gorm:"type:date;uniqueIndex:idx_uniqueProdDay,priority:1;index:idx_prodDayDate;not null" json:"balanceDate"`
	ProductID   uint              `gorm:"uniqueIndex:idx_uniqueProdDay,priority:2;not null" json:"-"`
	BookQty     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"bookQty"`
	PhysicalQty decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"physicalQty"`
	LineQty     decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"lineQty"`
	GainLoss    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"gainLoss"`
	UpdatedAt   time.Time         `json:"updatedAt"`

	Product Product `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"-"`
}

// IttTransfer moves ownership between customers.
type IttTransfer struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueITTUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:55;not null;check:ContentType=55" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueITTNo" json:"documentNumber"`
	TransferDate   time.Time            `gorm:"not null;index;index:idx_ittList,priority:2" json:"transferDate"`
	FromCustomerID uint                 `gorm:"index;index:idx_ittCommit,priority:1;not null" json:"-"`
	FromCustomer   Customer             `gorm:"foreignKey:FromCustomerID;constraint:OnDelete:NO ACTION;" json:"fromCustomer,omitempty"`
	ToCustomerID   uint                 `gorm:"index;not null" json:"-"`
	ToCustomer     Customer             `gorm:"foreignKey:ToCustomerID;constraint:OnDelete:NO ACTION;" json:"toCustomer,omitempty"`
	ProductID      uint                 `gorm:"index;index:idx_ittCommit,priority:2;not null" json:"-"`
	Product        Product              `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	VesselID       uint                 `gorm:"index;not null" json:"-"`
	Vessel         Vessel               `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel,omitempty"`
	VesselDate     time.Time            `gorm:"not null" json:"vesselDate"`
	StockStatusID  uint                 `gorm:"index;not null" json:"-"`
	DepotID        *uint                `gorm:"index" json:"-"`
	Quantity       decimal.Decimal      `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter     decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne    decimal.Decimal      `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	FinancialHold  bool                 `gorm:"default:0;not null" json:"financialHold"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_ittStatus;index:idx_ittList,priority:1;index:idx_ittCommit,priority:3" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
	DjangoID       uint                 `gorm:"index" json:"-"`
	ApprovalTrail  ApprovalTrail        `gorm:"type:nvarchar(max)" json:"-"`
}

func (m *IttTransfer) BeforeCreate(*gorm.DB) error {
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

func (m *IttTransfer) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *IttTransfer) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.FromCustomerID)
	MarkHasData(tx, &Customer{}, m.ToCustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	MarkHasData(tx, &Vessel{}, m.VesselID)
	markHasDataPtr(tx, &Depot{}, m.DepotID)
	return nil
}

// ZerolizationTransfer consolidates leftover volume onto another vessel/parcel.
type ZerolizationTransfer struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueZerolUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:56;not null;check:ContentType=56" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueZerolNo" json:"documentNumber"`
	TransferDate   time.Time            `gorm:"not null;index:idx_zerolTransferDate;index:idx_zerolList,priority:2" json:"transferDate"`
	CustomerID     uint                 `gorm:"index;not null" json:"-"`
	Customer       Customer             `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer,omitempty"`
	ProductID      uint                 `gorm:"index;not null" json:"-"`
	Product        Product              `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	StockStatusID  uint                 `gorm:"index;not null" json:"-"`
	FromVesselID   uint                 `gorm:"index;not null" json:"-"`
	FromVessel     Vessel               `gorm:"foreignKey:FromVesselID;constraint:OnDelete:NO ACTION;" json:"fromVessel,omitempty"`
	FromVesselDate time.Time            `gorm:"not null" json:"fromVesselDate"`
	ToVesselID     uint                 `gorm:"index;not null" json:"-"`
	ToVessel       Vessel               `gorm:"foreignKey:ToVesselID;constraint:OnDelete:NO ACTION;" json:"toVessel,omitempty"`
	ToVesselDate   time.Time            `gorm:"not null" json:"toVesselDate"`
	Quantity       decimal.Decimal      `gorm:"type:decimal(18,3);not null" json:"quantity"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_zerolStatus;index:idx_zerolList,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

func (m *ZerolizationTransfer) BeforeCreate(*gorm.DB) error {
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

func (m *ZerolizationTransfer) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *ZerolizationTransfer) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	MarkHasData(tx, &Vessel{}, m.FromVesselID)
	MarkHasData(tx, &Vessel{}, m.ToVesselID)
	return nil
}

// FinancialHoldRelease records payment against parcels that were received on hold.
// Approval moves the released quantity from financial-hold stock onto free stock.
type FinancialHoldRelease struct {
	ID             uint                 `gorm:"primaryKey" json:"-"`
	UID            string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueHoldRelUID;not null" json:"id"`
	ContentType    types.ContentType    `gorm:"default:53;not null;check:ContentType=53" json:"-"`
	DocumentNumber string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueHoldRelNo" json:"documentNumber"`
	ReleaseDate    time.Time            `gorm:"not null;index:idx_holdRelDate;index:idx_holdRelList,priority:2" json:"releaseDate"`
	Description    string               `gorm:"type:nvarchar(250)" json:"description"`
	Notes          string               `gorm:"type:nvarchar(500)" json:"notes"`
	Status         types.DocumentStatus `gorm:"type:varchar(20);not null;default:draft;index:idx_holdRelStatus;index:idx_holdRelList,priority:1" json:"status"`
	CreatedByID    uint                 `gorm:"index;not null" json:"-"`
	CreatedBy      User                 `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator        *CreatedByRef        `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
	ApprovalTrail  ApprovalTrail        `gorm:"type:nvarchar(max)" json:"-"`

	Lines []FinancialHoldReleaseLine `gorm:"foreignKey:ReleaseID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
}

func (m *FinancialHoldRelease) BeforeCreate(*gorm.DB) error {
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

func (m *FinancialHoldRelease) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

// FinancialHoldReleaseLine is one paid parcel quantity on a hold-release document.
type FinancialHoldReleaseLine struct {
	ID            uint            `gorm:"primaryKey" json:"-"`
	UID           string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueHoldRelLineUID;not null" json:"id"`
	ReleaseID     uint            `gorm:"uniqueIndex:idx_uniqueHoldRelParcel,priority:1;index:idx_holdRelLineHdr;not null" json:"-"`
	CustomerID    uint            `gorm:"uniqueIndex:idx_uniqueHoldRelParcel,priority:2;index;not null" json:"-"`
	Customer      Customer        `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer"`
	ProductID     uint            `gorm:"uniqueIndex:idx_uniqueHoldRelParcel,priority:3;index;not null" json:"-"`
	Product       Product         `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	VesselID      uint            `gorm:"uniqueIndex:idx_uniqueHoldRelParcel,priority:4;index;not null" json:"-"`
	Vessel        Vessel          `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel"`
	VesselDate    time.Time       `gorm:"type:date;uniqueIndex:idx_uniqueHoldRelParcel,priority:5;not null" json:"vesselDate"`
	StockStatusID uint            `gorm:"uniqueIndex:idx_uniqueHoldRelParcel,priority:6;index;not null" json:"-"`
	StockStatus   StockStatus     `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus"`
	Quantity      decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter    decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne   decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (m *FinancialHoldReleaseLine) BeforeCreate(*gorm.DB) error {
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

func (m *FinancialHoldReleaseLine) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	MarkHasData(tx, &Vessel{}, m.VesselID)
	MarkHasData(tx, &StockStatus{}, m.StockStatusID)
	return nil
}

// InventoryEventLog is an inbound gantry / pump-over / ITT message.
type InventoryEventLog struct {
	ID            uint                     `gorm:"primaryKey" json:"-"`
	UID           string                   `gorm:"type:varchar(26);uniqueIndex:idx_uniqueEventUID;not null" json:"id"`
	ContentType   types.ContentType        `gorm:"default:57;not null;check:ContentType=57" json:"-"`
	MessageID     string                   `gorm:"type:varchar(80);uniqueIndex:idx_uniqueEventMsg" json:"messageId"`
	EventType     types.InventoryEventType `gorm:"type:varchar(20);not null;index" json:"eventType"`
	OccurredAt    time.Time                `gorm:"not null;index" json:"occurredAt"`
	CustomerCode  string                   `gorm:"type:varchar(20);index" json:"customerCode"`
	ProductCode   string                   `gorm:"type:varchar(20)" json:"productCode"`
	VesselCode    string                   `gorm:"type:varchar(20)" json:"vesselCode"`
	VesselDate    *time.Time               `json:"vesselDate,omitempty"`
	Quantity      decimal.Decimal          `gorm:"type:decimal(18,3);not null" json:"quantity"`
	StatusCode    string                   `gorm:"type:varchar(20)" json:"statusCode"`
	FinancialHold bool                     `gorm:"default:0;not null" json:"financialHold"`
	OrderNumber   string                   `gorm:"type:varchar(40);index" json:"orderNumber"`
	Payload       JSONMap                  `gorm:"type:nvarchar(max)" json:"payload,omitempty"`
	Posted        bool                     `gorm:"default:0;not null;index" json:"posted"`
	PostedAt      *time.Time               `json:"postedAt,omitempty"`
	Error         string                   `gorm:"type:nvarchar(500)" json:"error,omitempty"`
	CreatedAt     time.Time                `json:"createdAt"`
}

func (m *InventoryEventLog) BeforeCreate(*gorm.DB) error {
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

// StockReservation is a soft hold taken when an external order starts.
type StockReservation struct {
	ID            uint                    `gorm:"primaryKey" json:"-"`
	UID           string                  `gorm:"type:varchar(26);uniqueIndex:idx_uniqueReserveUID;not null" json:"id"`
	ContentType   types.ContentType       `gorm:"default:72;not null;check:ContentType=72" json:"-"`
	CustomerID    uint                    `gorm:"index;index:idx_reserveOpenCust,priority:2;not null" json:"-"`
	ProductID     uint                    `gorm:"index;index:idx_reserveOpenCust,priority:3;not null" json:"-"`
	VesselID      *uint                   `gorm:"index" json:"-"`
	VesselDate    *time.Time              `json:"vesselDate,omitempty"`
	StockStatusID *uint                   `gorm:"index" json:"-"`
	Quantity      decimal.Decimal         `gorm:"type:decimal(18,3);not null" json:"quantity"`
	OrderNumber   string                  `gorm:"type:varchar(40);index" json:"orderNumber"`
	Status        types.ReservationStatus `gorm:"type:varchar(20);not null;default:open;index;index:idx_reserveList,priority:1;index:idx_reserveOpenCust,priority:1" json:"status"`
	ExpiresAt     *time.Time              `json:"expiresAt,omitempty"`
	CreatedAt     time.Time               `gorm:"index:idx_reserveCreatedAt;index:idx_reserveList,priority:2" json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
}

func (m *StockReservation) BeforeCreate(*gorm.DB) error {
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

// PhysicalDip is a daily tank measurement.
type PhysicalDip struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueDipUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:69;not null;check:ContentType=69" json:"-"`
	DipDate     time.Time         `gorm:"uniqueIndex:idx_uniqueDip,priority:1;not null" json:"dipDate"`
	TankID      uint              `gorm:"uniqueIndex:idx_uniqueDip,priority:2;not null" json:"-"`
	Tank        Tank              `gorm:"foreignKey:TankID;constraint:OnDelete:NO ACTION;" json:"tank"`
	Observed    decimal.Decimal   `gorm:"type:decimal(18,3);not null" json:"observed"`
	At20        decimal.Decimal   `gorm:"type:decimal(18,3);not null" json:"at20"`
	CreatedByID uint              `gorm:"index;not null" json:"-"`
	CreatedAt   time.Time         `json:"createdAt"`
}

func (m *PhysicalDip) BeforeCreate(*gorm.DB) error {
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

func (m *PhysicalDip) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Tank{}, m.TankID)
	return nil
}

// LineContent is daily pipeline volume (internal / external).
type LineContent struct {
	ID             uint              `gorm:"primaryKey" json:"-"`
	UID            string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueLineUID;not null" json:"id"`
	ContentType    types.ContentType `gorm:"default:79;not null;check:ContentType=79" json:"-"`
	ContentDate    time.Time         `gorm:"uniqueIndex:idx_uniqueLine,priority:1;not null" json:"contentDate"`
	ProductID      uint              `gorm:"uniqueIndex:idx_uniqueLine,priority:2;not null" json:"-"`
	InternalVolume decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"internalVolume"`
	ExternalVolume decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"externalVolume"`
	CreatedByID    uint              `gorm:"index;not null" json:"-"`
	CreatedAt      time.Time         `json:"createdAt"`
}

func (m *LineContent) BeforeCreate(*gorm.DB) error {
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
