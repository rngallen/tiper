package models

import (
	"strings"
	"time"
	"unicode"

	"dfms/pkg/types"
	"dfms/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// UnitOfMeasure is a stock UOM (L, M3, MT).
type UnitOfMeasure struct {
	Code        string            `gorm:"primaryKey;type:varchar(10);not null" json:"code"`
	Description string            `gorm:"type:nvarchar(80);not null" json:"description"`
	ContentType types.ContentType `gorm:"default:43;not null;check:ContentType=43" json:"-"`
	IsActive    bool              `gorm:"default:1;not null" json:"isActive"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// StockCategory groups products. Seed keeps a single "Petroleum products" row.
type StockCategory struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueStockCategoryUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:40;not null;check:ContentType=40" json:"-"`
	Name        string            `gorm:"type:nvarchar(80);not null;uniqueIndex:idx_uniqueStockCategoryName;check:chk_stockcat_name,[Name] <> ''" json:"name"`
	IsActive    bool              `gorm:"default:1;not null" json:"isActive"`
	HasData     bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID uint              `gorm:"index;not null" json:"-"`
	CreatedBy   User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func (m *StockCategory) BeforeCreate(*gorm.DB) error {
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

// Product is a stock item (AGO, PMS, IK, HFO, …).
type Product struct {
	ID              uint              `gorm:"primaryKey" json:"-"`
	UID             string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueProductUID;not null" json:"id"`
	ContentType     types.ContentType `gorm:"default:41;not null;check:ContentType=41" json:"-"`
	Code            string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueProductCode;index:idx_productActiveCode,priority:2;check:chk_product_code,[Code] <> ''" json:"code"`
	Name            string            `gorm:"type:nvarchar(120);not null;index:idx_productName;check:chk_product_name,[Name] <> ''" json:"name"`
	Unit            string            `gorm:"type:varchar(10);not null;default:L" json:"unit"`
	StockCategoryID uint              `gorm:"index;not null" json:"-"`
	StockCategory   StockCategory     `gorm:"foreignKey:StockCategoryID;constraint:OnDelete:NO ACTION;" json:"stockCategory"`
	IsActive        bool              `gorm:"default:1;not null;index:idx_productActiveCode,priority:1" json:"isActive"`
	HasData         bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID     uint              `gorm:"index;not null" json:"-"`
	CreatedBy       User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	DjangoID        uint              `gorm:"index" json:"-"`
}

func (m *Product) BeforeCreate(*gorm.DB) error {
	m.Unit = "L"
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

func (m *Product) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &StockCategory{}, m.StockCategoryID)
	return nil
}

// StockStatus is a book-stock bucket. Class is carried by flags so reports
// roll up without hard-coding codes:
//
//	IsTransit  — generic Transit and corridor subclasses (Congo, Rwanda, …)
//	IsLocal    — domestic stock; Mining also sets IsMining
//	IsMining   — subclass of local (implies IsLocal)
//	IsProration
//
// ParentID groups subclasses under Transit or Local in the operator picker.
type StockStatus struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueStockStatusUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:42;not null;check:ContentType=42" json:"-"`
	Code        string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueStockStatusCode" json:"code"`
	Name        string            `gorm:"type:nvarchar(80);not null" json:"name"`
	IsTransit   bool              `gorm:"default:0;not null;index:idx_statusTransit;check:chk_status_class,NOT ([IsTransit] = 1 AND [IsLocal] = 1)" json:"isTransit"`
	IsLocal     bool              `gorm:"default:0;not null;index:idx_statusLocal" json:"isLocal"`
	IsMining    bool              `gorm:"default:0;not null;check:chk_status_mining,[IsMining] = 0 OR [IsLocal] = 1" json:"isMining"`
	IsProration bool              `gorm:"default:0;not null" json:"isProration"`
	ParentID    *uint             `gorm:"index:idx_statusParent" json:"-"`
	Parent      *StockStatus      `gorm:"foreignKey:ParentID;constraint:OnDelete:NO ACTION;" json:"parent,omitempty"`
	IsActive    bool              `gorm:"default:1;not null" json:"isActive"`
	HasData     bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
}

func (m *StockStatus) BeforeCreate(tx *gorm.DB) error {
	if m.UID == "" {
		uid, err := utils.GetULID()
		if err != nil {
			return err
		}
		m.UID = uid
	}
	return nil
}

func (m *StockStatus) AfterCreate(tx *gorm.DB) error {
	if m.Code != "" {
		return nil
	}
	return tx.Model(m).Update("Code", m.ID+10).Error
}

// SlugStatusCode turns "Congo DRC" into CONGODRC (max 20).
func SlugStatusCode(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
		if b.Len() >= 20 {
			break
		}
	}
	return b.String()
}

// ClassifyStockStatus guesses class from a name when the UI did not send flags.
// Corridor countries are not listed here — create them as Transit in
// Stock → Setups → Stock statuses (or the Django migrator name map).
func ClassifyStockStatus(name string) (transit, local, mining, proration bool) {
	n := strings.ToUpper(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", " ")
	n = strings.Join(strings.Fields(n), " ")
	switch {
	case n == string(types.StockProration) || strings.Contains(n, "PRORAT"):
		return false, false, false, true
	case n == "MINES" || n == string(types.StockMining):
		return false, true, true, false
	case n == string(types.StockLocal) || n == "DOMESTIC":
		return false, true, false, false
	case n == string(types.StockTransit) || strings.Contains(n, "TRANSIT"):
		return true, false, false, false
	default:
		return false, true, false, false
	}
}

// Tank is a physical storage tank at TIPER.
type Tank struct {
	ID              uint              `gorm:"primaryKey" json:"-"`
	UID             string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTankUID;not null" json:"id"`
	ContentType     types.ContentType `gorm:"default:44;not null;check:ContentType=44" json:"-"`
	Code            string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueTankCode;index:idx_tankActiveCode,priority:2" json:"code"`
	Name            string            `gorm:"type:nvarchar(80);not null" json:"name"`
	MaximumCapacity decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0;check:chk_tank_capacity,[MaximumCapacity] > [DeadStock] AND [DeadStock] >= 0" json:"maximumCapacity"`
	DeadStock       decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"deadStock"`
	ProductID       uint              `gorm:"index;not null" json:"-"`
	Product         Product           `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	IsActive        bool              `gorm:"default:1;not null;index:idx_tankActiveCode,priority:1" json:"isActive"`
	HasData         bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID     uint              `gorm:"index;not null" json:"-"`
	CreatedBy       User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	DjangoID        uint              `gorm:"index" json:"-"`
}

func (m *Tank) BeforeCreate(*gorm.DB) error {
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

// Vessel delivers product to the terminal.
type Vessel struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueVesselUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:45;not null;check:ContentType=45" json:"-"`
	Code        string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueVesselCode" json:"code"`
	ImoNumber   string            `gorm:"type:varchar(20);index" json:"imoNumber"`
	Name        string            `gorm:"type:nvarchar(120);not null;index:idx_vesselActiveName,priority:2" json:"name"`
	IsActive    bool              `gorm:"default:1;not null;index:idx_vesselActiveName,priority:1" json:"isActive"`
	HasData     bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID uint              `gorm:"index;not null" json:"-"`
	CreatedBy   User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
}

func (m *Vessel) BeforeCreate(tx *gorm.DB) error {
	if m.UID == "" {
		uid, err := utils.GetULID()
		if err != nil {
			return err
		}
		m.UID = uid
	}
	return nil
}

func (m *Vessel) AfterCreate(tx *gorm.DB) error {
	if m.Code != "" {
		return nil
	}
	return tx.Model(m).Update("Code", m.ID+2000).Error
}

// Depot is TIPER (internal) or a pump-over destination.
type Depot struct {
	ID               uint                    `gorm:"primaryKey" json:"-"`
	UID              string                  `gorm:"type:varchar(26);uniqueIndex:idx_uniqueDepotUID;not null" json:"id"`
	ContentType      types.ContentType       `gorm:"default:46;not null;check:ContentType=46" json:"-"`
	Code             string                  `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueDepotCode" json:"code"`
	Name             string                  `gorm:"type:nvarchar(120);not null;index:idx_depotActiveName,priority:2" json:"name"`
	EwuraLicense     string                  `gorm:"type:varchar(40);index" json:"ewuraLicense"`
	IsInternal       bool                    `gorm:"default:0;not null" json:"isInternal"`
	IsActive         bool                    `gorm:"default:1;not null;index:idx_depotActiveName,priority:1" json:"isActive"`
	HasData          bool                    `gorm:"default:0;not null" json:"hasData"`
	CustomerID       *uint                   `gorm:"index" json:"-"`
	Customer         *Customer               `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer,omitempty"`
	BillingAccountID *uint                   `gorm:"index" json:"-"`
	BillingAccount   *CustomerBillingAccount `gorm:"foreignKey:BillingAccountID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedByID      uint                    `gorm:"index;not null" json:"-"`
	CreatedBy        User                    `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
	DjangoID         uint                    `gorm:"index" json:"-"`
}

func (m *Depot) BeforeCreate(*gorm.DB) error {
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

func (m *Depot) AfterCreate(tx *gorm.DB) error {
	if m.Code != "" {
		return nil
	}
	return tx.Model(m).Update("Code", m.ID+3000).Error
}

// Lookup tables (code PK). Behavior flags drive the engine; adding a row does not need a deploy.
type BillingCycle struct {
	Days        int               `gorm:"primaryKey" json:"days"`
	Description string            `gorm:"type:nvarchar(80);not null" json:"description"`
	IsActive    bool              `gorm:"default:1;not null;index:idx_cycleList" json:"isActive"`
	ContentType types.ContentType `gorm:"default:59;not null;check:ContentType=59" json:"-"`
}

type ImportTenderType struct {
	Code                      types.TenderCode  `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name                      string            `gorm:"type:nvarchar(80);not null" json:"name"`
	IsSingleReceiving         bool              `gorm:"default:0;not null" json:"isSingleReceiving"`
	SupplierPaysUnlessLoading bool              `gorm:"default:0;not null" json:"supplierPaysUnlessLoading"`
	IsActive                  bool              `gorm:"default:1;not null;index:idx_tenderList" json:"isActive"`
	ContentType               types.ContentType `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type DeliveryMethod struct {
	Code            types.CollectionMethod `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name            string                 `gorm:"type:nvarchar(80);not null" json:"name"`
	IsGantryLoading bool                   `gorm:"default:0;not null" json:"isGantryLoading"`
	IsActive        bool                   `gorm:"default:1;not null;index:idx_deliveryList" json:"isActive"`
	ContentType     types.ContentType      `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type ProcurementMethod struct {
	Code        types.ProcurementCode `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name        string                `gorm:"type:nvarchar(80);not null" json:"name"`
	IsActive    bool                  `gorm:"default:1;not null;index:idx_procList" json:"isActive"`
	ContentType types.ContentType     `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type DischargeRoute struct {
	Code        types.DischargeRoute `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name        string               `gorm:"type:nvarchar(80);not null" json:"name"`
	IsActive    bool                 `gorm:"default:1;not null;index:idx_routeList" json:"isActive"`
	ContentType types.ContentType    `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type ContractType struct {
	Code        types.ContractCode `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name        string             `gorm:"type:nvarchar(80);not null" json:"name"`
	IsActive    bool               `gorm:"default:1;not null;index:idx_contractList" json:"isActive"`
	ContentType types.ContentType  `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type PricingNature struct {
	Code          types.PricingNature `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name          string              `gorm:"type:nvarchar(80);not null" json:"name"`
	IsPromotional bool                `gorm:"default:0;not null" json:"isPromotional"`
	IsActive      bool                `gorm:"default:1;not null;index:idx_pricingList" json:"isActive"`
	ContentType   types.ContentType   `gorm:"default:75;not null;check:ContentType=75" json:"-"`
}

type Fee struct {
	Code           types.FeeCode     `gorm:"primaryKey;type:varchar(10)" json:"code"`
	Name           string            `gorm:"type:nvarchar(80);not null" json:"name"`
	RevenueAccount string            `gorm:"type:varchar(40)" json:"revenueAccount"`
	ChargeTo       types.ChargeTo    `gorm:"type:varchar(20);not null;default:customer" json:"chargeTo"`
	IsActive       bool              `gorm:"default:1;not null" json:"isActive"`
	ContentType    types.ContentType `gorm:"default:60;not null;check:ContentType=60" json:"-"`
}
