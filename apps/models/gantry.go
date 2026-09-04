package models

import (
	"time"

	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Compartmentalization is the gantry dispatch document for one open ILO.
// Django: GantryCompartmentalization. Snapshots are taken at create so later
// master-data edits do not rewrite history.
//
// Django → DFMS (column names changed; the migrator maps the old names):
//
//	transaction_id          → TransactionID   (shared NPGIS sequence)
//	loading_status          → Status          (running on create)
//	ilo / order (ILR line)  → IloID           (ILO collapsed into GantryLoadingLine)
//	request                 → RequestID       (ILR)
//	printed / get_pass_date → Printed / PrintedAt (first print / gate-pass)
//	doc_date                → (dropped; PrintedAt is the print timestamp)
//	sent_at / file_name     → AlmaSentAt / AlmaFileName
//	sent_to_ewura / ewura_sent_at → NpgisSent / NpgisSentAt
//	driver_licence          → DriverLicense
//	horse_plate_number      → HorsePlate
//	quantity / loaded       → RequestedQty / LoadedQty
//
// Lifecycle: running (configured, not sent) → submitted → approved → inprogress
// (SAP3C in the ALMA In folder) → loaded (SAP3R) → closed.
type Compartmentalization struct {
	ID                  uint              `gorm:"primaryKey" json:"-"`
	UID                 string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCompUID;not null" json:"id"`
	ContentType         types.ContentType `gorm:"default:93;not null;check:ContentType=93" json:"-"`
	TransactionID       uint              `gorm:"uniqueIndex:idx_uniqueCompTrans;not null" json:"transactionId"`
	DocumentNumber      string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueCompDoc" json:"documentNumber"`
	CustomerOrderNumber string            `gorm:"type:varchar(40);index:idx_compCustOrder" json:"customerOrderNumber"`
	BatchNumber         string            `gorm:"type:varchar(30)" json:"batchNumber"`

	IloID     uint                 `gorm:"index:idx_compIlo;not null" json:"-"`
	Ilo       GantryLoadingLine    `gorm:"foreignKey:IloID;constraint:OnDelete:NO ACTION;" json:"ilo,omitempty"`
	RequestID uint                 `gorm:"index:idx_compRequest;not null" json:"-"`
	Request   GantryLoadingRequest `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"-"`

	CustomerID   uint     `gorm:"index:idx_compCustomer;not null" json:"-"`
	Customer     Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer,omitempty"`
	CustomerCode string   `gorm:"type:varchar(20)" json:"customerCode"`
	CustomerName string   `gorm:"type:nvarchar(160)" json:"customerName"`

	ProductID         uint            `gorm:"index:idx_compProduct;not null" json:"-"`
	Product           Product         `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ProductCode       string          `gorm:"type:varchar(20)" json:"productCode"`
	ProductName       string          `gorm:"type:nvarchar(120)" json:"productName"`
	RequestedQty      decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"requestedQty"`
	ByProductID       *uint           `gorm:"index" json:"-"`
	ByProduct         *Product        `gorm:"foreignKey:ByProductID;constraint:OnDelete:NO ACTION;" json:"byProduct,omitempty"`
	ByProductCode     string          `gorm:"type:varchar(20)" json:"byProductCode,omitempty"`
	ByProductName     string          `gorm:"type:nvarchar(120)" json:"byProductName,omitempty"`
	ByProductQuantity decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"byProductQuantity"`
	StockStatusID     uint            `gorm:"index" json:"-"`
	StockStatus       StockStatus     `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus,omitempty"`
	StockStatusCode   string          `gorm:"type:varchar(20)" json:"stockStatusCode"`
	StockStatusName   string          `gorm:"type:nvarchar(80)" json:"stockStatusName"`

	TransporterID   *uint        `gorm:"index" json:"-"`
	Transporter     *Transporter `gorm:"foreignKey:TransporterID;constraint:OnDelete:NO ACTION;" json:"transporter,omitempty"`
	TransporterName string       `gorm:"type:nvarchar(180)" json:"transporterName"`
	DriverID        *uint        `gorm:"index" json:"-"`
	Driver          *Driver      `gorm:"foreignKey:DriverID;constraint:OnDelete:NO ACTION;" json:"driver,omitempty"`
	DriverName      string       `gorm:"type:nvarchar(160)" json:"driverName"`
	DriverLicense   string       `gorm:"type:varchar(40)" json:"driverLicense"`
	TruckID         *uint        `gorm:"index" json:"-"`
	Truck           *Truck       `gorm:"foreignKey:TruckID;constraint:OnDelete:NO ACTION;" json:"truck,omitempty"`
	PlateNumber     string       `gorm:"type:varchar(60);index:idx_compPlate" json:"plateNumber"`
	HorsePlate      string       `gorm:"type:varchar(20)" json:"horsePlate"`
	TrailerOneID    *uint        `gorm:"index:idx_compTrailerOne" json:"-"`
	TrailerOne      *TruckTank   `gorm:"foreignKey:TrailerOneID;constraint:OnDelete:NO ACTION;" json:"-"`
	TrailerOnePlate string       `gorm:"type:varchar(20)" json:"trailerOnePlate"`
	TrailerTwoID    *uint        `gorm:"index:idx_compTrailerTwo" json:"-"`
	TrailerTwo      *TruckTank   `gorm:"foreignKey:TrailerTwoID;constraint:OnDelete:NO ACTION;" json:"-"`
	TrailerTwoPlate string       `gorm:"type:varchar(20)" json:"trailerTwoPlate"`

	BadgeID   *uint      `gorm:"index:idx_compBadge" json:"-"`
	Badge     *RfidBadge `gorm:"foreignKey:BadgeID;constraint:OnDelete:NO ACTION;" json:"badge,omitempty"`
	BadgeCode string     `gorm:"type:varchar(40)" json:"badgeCode"`

	OrderDate      time.Time  `gorm:"not null;index:idx_compOrderDate" json:"orderDate"`
	ExpirationDate *time.Time `json:"expirationDate,omitempty"`
	EwuraLicense   string     `gorm:"type:varchar(40)" json:"ewuraLicense"`
	Destination    string     `gorm:"type:nvarchar(160)" json:"destination"`
	District       string     `gorm:"type:nvarchar(80)" json:"district"`

	Printed   bool       `gorm:"default:0;not null" json:"printed"`
	PrintedAt *time.Time `json:"printedAt,omitempty"` // first print / gate-pass

	LoadedQty       decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"loadedQty"`
	ByProductLoaded decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"byProductLoaded"`
	LoadedAt        *time.Time      `json:"loadedAt,omitempty"`
	AlmaFileName    string          `gorm:"type:varchar(80)" json:"almaFileName,omitempty"`
	AlmaSentAt      *time.Time      `json:"almaSentAt,omitempty"`
	NpgisSent       bool            `gorm:"default:0;not null" json:"npgisSent"`
	NpgisSentAt     *time.Time      `json:"npgisSentAt,omitempty"`

	Amended       bool                       `gorm:"default:0;not null;index:idx_compAmended" json:"amended"`
	IsActive      bool                       `gorm:"default:1;not null;index:idx_compActive" json:"isActive"`
	Status        types.OrderStatus          `gorm:"type:varchar(20);not null;default:running;index:idx_compStatus;index:idx_compList,priority:1" json:"status"`
	CreatedByID   uint                       `gorm:"index;not null" json:"-"`
	CreatedBy     User                       `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator       *CreatedByRef              `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt     time.Time                  `gorm:"index:idx_compCreatedAt;index:idx_compList,priority:2" json:"createdAt"`
	UpdatedAt     time.Time                  `json:"updatedAt"`
	DjangoID      uint                       `gorm:"index" json:"-"`
	ApprovalTrail ApprovalTrail              `gorm:"type:nvarchar(max)" json:"-"`
	Lines         []CompartmentalizationLine `gorm:"foreignKey:CompartmentalizationID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
}

func (m *Compartmentalization) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *Compartmentalization) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m Compartmentalization) TableName() string { return "GantryCompartmentalization" }

// CompartmentalizationLine is one tank chamber on a dispatch (Django GantryCompartmentalizationLine).
// Capacity is copied from the calibration chart; quantity cannot exceed it.
// Balance = capacity − quantity. Index is the chamber position on that tank
// (tank 1 and tank 2 may both have index 1).
//
// Seals (top / dip / bottom) must be unique across the whole table — a filtered
// unique index rejects empty strings so unused chambers stay blank.
type CompartmentalizationLine struct {
	ID                     uint              `gorm:"primaryKey" json:"-"`
	UID                    string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCompLineUID;not null" json:"id"`
	ContentType            types.ContentType `gorm:"default:94;not null;check:ContentType=94" json:"-"`
	CompartmentalizationID uint              `gorm:"uniqueIndex:idx_uniqueCompCell,priority:1;index:idx_compLineComp;not null" json:"-"`
	CalibrationID          uint              `gorm:"index:idx_compLineCal;not null" json:"-"`
	TankID                 uint              `gorm:"uniqueIndex:idx_uniqueCompCell,priority:2;not null" json:"-"`
	Tank                   TruckTank         `gorm:"foreignKey:TankID;constraint:OnDelete:NO ACTION;" json:"tank,omitempty"`
	TankPlate              string            `gorm:"type:varchar(20);not null" json:"tankPlate"`
	Index                  int               `gorm:"uniqueIndex:idx_uniqueCompCell,priority:3;not null" json:"index"`
	Capacity               decimal.Decimal   `gorm:"type:decimal(18,0);not null" json:"capacity"`
	ProductID              *uint             `gorm:"index:idx_compLineProduct" json:"-"`
	Product                *Product          `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ProductCode            string            `gorm:"type:varchar(20)" json:"productCode,omitempty"`
	ProductName            string            `gorm:"type:nvarchar(120)" json:"productName,omitempty"`
	Quantity               decimal.Decimal   `gorm:"type:decimal(18,0);not null;default:0" json:"quantity"`
	Balance                decimal.Decimal   `gorm:"type:decimal(18,0);not null;default:0" json:"balance"`
	TopSeal                string            `gorm:"type:varchar(40);index:idx_compTopSeal" json:"topSeal,omitempty"`
	DipSeal                string            `gorm:"type:varchar(40);index:idx_compDipSeal" json:"dipSeal,omitempty"`
	BottomSeal             string            `gorm:"type:varchar(40);index:idx_compBottomSeal" json:"bottomSeal,omitempty"`
	DjangoID               uint              `gorm:"index" json:"-"`
}

func (m *CompartmentalizationLine) BeforeCreate(*gorm.DB) error {
	m.Balance = m.Capacity.Sub(m.Quantity)
	return assignUID(&m.UID)
}

func (m *CompartmentalizationLine) BeforeSave(*gorm.DB) error {
	m.Balance = m.Capacity.Sub(m.Quantity)
	return nil
}

func (m CompartmentalizationLine) TableName() string { return "GantryCompartmentalizationLine" }

// GantryLoading is the ATLAS NEO loaded-truck record (Django GantryLoading).
// One row per compartmentalization. Product measures live on GantryLoadingProduct
// so AGO, PMS, and later grades share the same header instead of ago_* / mogas_* columns.
// Year/Month are denormalised from LoadedAt for month reports (no DATEPART in filters).
type GantryLoading struct {
	ID                     uint                 `gorm:"primaryKey" json:"-"`
	UID                    string               `gorm:"type:varchar(26);uniqueIndex:idx_uniqueLoadUID;not null" json:"id"`
	ContentType            types.ContentType    `gorm:"default:95;not null;check:ContentType=95" json:"-"`
	CompartmentalizationID uint                 `gorm:"uniqueIndex:idx_uniqueLoadComp;not null" json:"-"`
	Compartmentalization   Compartmentalization `gorm:"foreignKey:CompartmentalizationID;constraint:OnDelete:NO ACTION;" json:"-"`
	IloID                  uint                 `gorm:"index:idx_loadIlo;not null" json:"-"`
	Ilo                    GantryLoadingLine    `gorm:"foreignKey:IloID;constraint:OnDelete:NO ACTION;" json:"-"`
	RequestID              uint                 `gorm:"index:idx_loadRequest;not null" json:"-"`
	DocumentNumber         string               `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueLoadDoc" json:"documentNumber"`
	CustomerOrderNumber    string               `gorm:"type:varchar(40)" json:"customerOrderNumber"`
	BatchNumber            string               `gorm:"type:varchar(30)" json:"batchNumber"`
	OrderDate              time.Time            `gorm:"not null" json:"orderDate"`
	LoadedAt               time.Time            `gorm:"not null;index:idx_loadAt" json:"loadedAt"`
	Year                   int                  `gorm:"not null;index:idx_loadYearMonth,priority:1" json:"year"`
	Month                  int                  `gorm:"not null;index:idx_loadYearMonth,priority:2" json:"month"`
	RequestedQty           decimal.Decimal      `gorm:"type:decimal(18,3);not null" json:"requestedQty"`

	BadgeID   *uint  `gorm:"index" json:"-"`
	BadgeCode string `gorm:"type:varchar(40)" json:"badgeCode"`

	CustomerID   uint   `gorm:"index:idx_loadCustomer;not null" json:"-"`
	CustomerCode string `gorm:"type:varchar(20)" json:"customerCode"`
	CustomerName string `gorm:"type:nvarchar(160);index:idx_loadCustName" json:"customerName"`

	StockStatusID   uint   `gorm:"index" json:"-"`
	StockStatusCode string `gorm:"type:varchar(20)" json:"stockStatusCode"`
	StockStatusName string `gorm:"type:nvarchar(80)" json:"stockStatusName"`

	TransporterID   *uint                  `gorm:"index" json:"-"`
	TransporterName string                 `gorm:"type:nvarchar(180)" json:"transporterName"`
	DriverID        *uint                  `gorm:"index" json:"-"`
	DriverName      string                 `gorm:"type:nvarchar(160)" json:"driverName"`
	DriverLicense   string                 `gorm:"type:varchar(40)" json:"driverLicense"`
	TruckID         *uint                  `gorm:"index" json:"-"`
	PlateNumber     string                 `gorm:"type:varchar(60);index:idx_loadPlate" json:"plateNumber"`
	EwuraLicense    string                 `gorm:"type:varchar(40)" json:"ewuraLicense"`
	Destination     string                 `gorm:"type:nvarchar(160)" json:"destination"`
	District        string                 `gorm:"type:nvarchar(80)" json:"district"`
	ExpirationDate  *time.Time             `json:"expirationDate,omitempty"`
	AlmaFileName    string                 `gorm:"type:varchar(80)" json:"almaFileName"`
	CreatedAt       time.Time              `json:"createdAt"`
	DjangoID        uint                   `gorm:"index" json:"-"`
	Products        []GantryLoadingProduct `gorm:"foreignKey:LoadingID;constraint:OnDelete:NO ACTION;" json:"products,omitempty"`
}

func (m *GantryLoading) BeforeCreate(*gorm.DB) error {
	if m.Year == 0 {
		m.Year = m.LoadedAt.Year()
	}
	if m.Month == 0 {
		m.Month = int(m.LoadedAt.Month())
	}
	return assignUID(&m.UID)
}

func (m GantryLoading) TableName() string { return "GantryLoading" }

// GantryLoadingProduct is one grade measured on a loaded truck (AGO, PMS, …).
// WCF = density − 0.0011; Weight = WCF × standard volume (m³).
type GantryLoadingProduct struct {
	ID             uint            `gorm:"primaryKey" json:"-"`
	UID            string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueLoadProductUID;not null" json:"id"`
	LoadingID      uint            `gorm:"uniqueIndex:idx_uniqueLoadProduct,priority:1;not null" json:"-"`
	ProductID      uint            `gorm:"uniqueIndex:idx_uniqueLoadProduct,priority:2;index;not null" json:"-"`
	Product        Product         `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ProductCode    string          `gorm:"type:varchar(20);not null" json:"productCode"`
	ProductName    string          `gorm:"type:nvarchar(120)" json:"productName"`
	ObservedVolume decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"observedVolume"`
	StandardVolume decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"standardVolume"`
	Temperature    decimal.Decimal `gorm:"type:decimal(8,2);not null;default:0" json:"temperature"`
	Density        decimal.Decimal `gorm:"type:decimal(10,4);not null;default:0" json:"density"`
	WCF            decimal.Decimal `gorm:"type:decimal(10,4);not null;default:0" json:"wcf"`
	Weight         decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"weight"`
}

func (m *GantryLoadingProduct) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func LoadingWCF(density decimal.Decimal) decimal.Decimal {
	return density.Sub(decimal.NewFromFloat(0.0011))
}

func LoadingWeight(wcf, standardVolume decimal.Decimal) decimal.Decimal {
	return wcf.Mul(standardVolume)
}

func (m GantryLoadingProduct) TableName() string { return "GantryLoadingProduct" }

// ApplyLoadingToSummary rolls a posted loading into the month×product report table.
// Requested volume is split across grades in proportion to standard volume so a
// dual-grade truck does not double-count the ILO quantity.
func ApplyLoadingToSummary(db *gorm.DB, load *GantryLoading, transit bool) error {
	if load == nil || len(load.Products) == 0 {
		return nil
	}
	var totalStd decimal.Decimal
	for _, p := range load.Products {
		totalStd = totalStd.Add(p.StandardVolume)
	}
	monthName := time.Month(load.Month).String()
	for _, p := range load.Products {
		requested := load.RequestedQty
		if totalStd.GreaterThan(decimal.Zero) && len(load.Products) > 1 {
			requested = load.RequestedQty.Mul(p.StandardVolume).Div(totalStd)
		}
		var row GantryLoadingSummary
		err := db.Where("Year = ? AND Month = ? AND ProductID = ?", load.Year, load.Month, p.ProductID).First(&row).Error
		if err != nil {
			row = GantryLoadingSummary{
				Year: load.Year, Month: load.Month, MonthName: monthName,
				ProductID: p.ProductID, ProductCode: p.ProductCode, ProductName: p.ProductName,
			}
		}
		if transit {
			row.TransitLoaded = row.TransitLoaded.Add(p.StandardVolume)
			row.TransitRequested = row.TransitRequested.Add(requested)
			row.TransitWeight = row.TransitWeight.Add(p.Weight)
		} else {
			row.LocalLoaded = row.LocalLoaded.Add(p.StandardVolume)
			row.LocalRequested = row.LocalRequested.Add(requested)
			row.LocalWeight = row.LocalWeight.Add(p.Weight)
		}
		if row.ID == 0 {
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// GantryLoadingSummary is a month×product rollup updated when a loading is posted.
// Django GantryLoadingSummary rows are not copied — rebuild from GantryLoading
// after migrate so figures match the new child-product design.
type GantryLoadingSummary struct {
	ID               uint              `gorm:"primaryKey" json:"-"`
	ContentType      types.ContentType `gorm:"default:103;not null;check:ContentType=103" json:"-"`
	Year             int               `gorm:"uniqueIndex:idx_uniqueLoadSummary,priority:1;not null" json:"year"`
	Month            int               `gorm:"uniqueIndex:idx_uniqueLoadSummary,priority:2;not null" json:"month"`
	MonthName        string            `gorm:"type:varchar(20)" json:"monthName"`
	ProductID        uint              `gorm:"uniqueIndex:idx_uniqueLoadSummary,priority:3;not null" json:"-"`
	Product          Product           `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ProductCode      string            `gorm:"type:varchar(20)" json:"productCode"`
	ProductName      string            `gorm:"type:nvarchar(120)" json:"productName"`
	LocalLoaded      decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"localLoaded"`
	TransitLoaded    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"transitLoaded"`
	LocalRequested   decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"localRequested"`
	TransitRequested decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"transitRequested"`
	LocalWeight      decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"localWeight"`
	TransitWeight    decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"transitWeight"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

func (m GantryLoadingSummary) TableName() string { return "GantryLoadingSummary" }

// GantryVesselLoading is loaded volume allocated to an ILR vessel parcel
// (Django GantryVesselLoading). Django posted / sequence_number / transaction_number
// are dropped — there is no Sage stock post from this table.
type GantryVesselLoading struct {
	ID                     uint                `gorm:"primaryKey" json:"-"`
	UID                    string              `gorm:"type:varchar(26);uniqueIndex:idx_uniqueVesLoadUID;not null" json:"id"`
	ContentType            types.ContentType   `gorm:"default:104;not null;check:ContentType=104" json:"-"`
	RequestVesselID        uint                `gorm:"index:idx_vesLoadParcel;not null" json:"-"`
	RequestVessel          GantryRequestVessel `gorm:"foreignKey:RequestVesselID;constraint:OnDelete:NO ACTION;" json:"-"`
	VesselID               uint                `gorm:"index:idx_vesLoadVessel;not null" json:"-"`
	Vessel                 Vessel              `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel,omitempty"`
	VesselDate             time.Time           `gorm:"not null" json:"vesselDate"`
	VesselName             string              `gorm:"type:nvarchar(120)" json:"vesselName"`
	ProductID              uint                `gorm:"index;not null" json:"-"`
	Product                Product             `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product,omitempty"`
	ProductCode            string              `gorm:"type:varchar(20)" json:"productCode"`
	ProductName            string              `gorm:"type:nvarchar(120)" json:"productName"`
	Quantity               decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"quantity"`
	Density                decimal.Decimal     `gorm:"type:decimal(10,4);not null;default:0" json:"density"`
	Temperature            decimal.Decimal     `gorm:"type:decimal(8,2);not null;default:0" json:"temperature"`
	StandardVolume         decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"standardVolume"`
	WCF                    decimal.Decimal     `gorm:"type:decimal(10,4);not null;default:0" json:"wcf"`
	Weight                 decimal.Decimal     `gorm:"type:decimal(18,3);not null;default:0" json:"weight"`
	FinancialHold          bool                `gorm:"default:0;not null" json:"financialHold"`
	StockStatusID          uint                `gorm:"index" json:"-"`
	StockStatusCode        string              `gorm:"type:varchar(20)" json:"stockStatusCode"`
	StockStatusName        string              `gorm:"type:nvarchar(80)" json:"stockStatusName"`
	CompartmentalizationID uint                `gorm:"index:idx_vesLoadComp;not null" json:"-"`
	IloID                  uint                `gorm:"index:idx_vesLoadIlo;not null" json:"-"`
	RequestID              uint                `gorm:"index:idx_vesLoadRequest;not null" json:"-"`
	LoadedAt               time.Time           `gorm:"not null;index:idx_vesLoadAt" json:"loadedAt"`
	BadgeID                *uint               `json:"-"`
	BadgeCode              string              `gorm:"type:varchar(40)" json:"badgeCode"`
	DocumentNumber         string              `gorm:"type:varchar(40);index:idx_vesLoadDoc" json:"documentNumber"`
	CustomerOrderNumber    string              `gorm:"type:varchar(40)" json:"customerOrderNumber"`
	BatchNumber            string              `gorm:"type:varchar(30)" json:"batchNumber"`
	CustomerID             uint                `gorm:"index;not null" json:"-"`
	CustomerCode           string              `gorm:"type:varchar(20)" json:"customerCode"`
	CustomerName           string              `gorm:"type:nvarchar(160)" json:"customerName"`
	TransporterID          *uint               `json:"-"`
	TransporterName        string              `gorm:"type:nvarchar(180)" json:"transporterName"`
	DriverID               *uint               `json:"-"`
	DriverName             string              `gorm:"type:nvarchar(160)" json:"driverName"`
	DriverLicense          string              `gorm:"type:varchar(40)" json:"driverLicense"`
	TruckID                *uint               `json:"-"`
	PlateNumber            string              `gorm:"type:varchar(60)" json:"plateNumber"`
	OrderDate              time.Time           `gorm:"not null" json:"orderDate"`
	EwuraLicense           string              `gorm:"type:varchar(40)" json:"ewuraLicense"`
	Destination            string              `gorm:"type:nvarchar(160)" json:"destination"`
	District               string              `gorm:"type:nvarchar(80)" json:"district"`
	ExpirationDate         *time.Time          `json:"expirationDate,omitempty"`
	DjangoID               uint                `gorm:"index" json:"-"`
}

func (m *GantryVesselLoading) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m GantryVesselLoading) TableName() string { return "GantryVesselLoading" }

// OrderAmendment is a single-ILO change (Django GantryAmendment).
// Only open and running ILOs can be amended. The old compartmentalization is
// marked amended=true; a new ILO (and later a new compartmentalization) is issued.
type OrderAmendment struct {
	ID              uint                `gorm:"primaryKey" json:"-"`
	UID             string              `gorm:"type:varchar(26);uniqueIndex:idx_uniqueAmendUID;not null" json:"id"`
	ContentType     types.ContentType   `gorm:"default:96;not null;check:ContentType=96" json:"-"`
	DocumentNumber  string              `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueAmendNo" json:"documentNumber"`
	Kind            types.AmendmentKind `gorm:"type:varchar(24);not null;index:idx_amendKind" json:"kind"`
	IloID           uint                `gorm:"index:idx_amendIlo;not null" json:"-"`
	Ilo             GantryLoadingLine   `gorm:"foreignKey:IloID;constraint:OnDelete:NO ACTION;" json:"ilo,omitempty"`
	RequestedQty    decimal.Decimal     `gorm:"type:decimal(18,3);not null" json:"requestedQty"`
	ProductID       *uint               `json:"-"`
	ExpirationDate  *time.Time          `json:"expirationDate,omitempty"`
	Destination     string              `gorm:"type:nvarchar(160)" json:"destination"`
	District        string              `gorm:"type:nvarchar(80)" json:"district"`
	TruckPlate      string              `gorm:"type:varchar(20)" json:"truckPlate"`
	TransporterName string              `gorm:"type:nvarchar(160)" json:"transporterName"`
	DriverName      string              `gorm:"type:nvarchar(160)" json:"driverName"`
	Notes           string              `gorm:"type:nvarchar(400)" json:"notes"`
	Status          types.OrderStatus   `gorm:"type:varchar(20);not null;default:draft;index:idx_amendStatus;index:idx_amendList,priority:1" json:"status"`
	CreatedByID     uint                `gorm:"index;not null" json:"-"`
	CreatedBy       User                `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator         *CreatedByRef       `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt       time.Time           `gorm:"index:idx_amendCreatedAt;index:idx_amendList,priority:2" json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	DjangoID        uint                `gorm:"index" json:"-"`
}

func (m *OrderAmendment) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *OrderAmendment) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}
