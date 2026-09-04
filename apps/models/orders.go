package models

import (
	"strings"
	"time"

	"dfms/pkg/types"
	"dfms/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transporter is a haulier that collects product at the gantry.
type Transporter struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	UID           string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTransporterUID;not null" json:"id"`
	ContentType   types.ContentType `gorm:"default:84;not null;check:ContentType=84" json:"-"`
	Name          string            `gorm:"type:nvarchar(180);not null;uniqueIndex:idx_uniqueTransporterName;index:idx_transporterActiveName,priority:2" json:"name"`
	Phone         string            `gorm:"type:varchar(24);index:idx_transporterPhone" json:"phone"`
	Email         string            `gorm:"type:varchar(160);index:idx_transporterEmail" json:"email"`
	ContactPerson string            `gorm:"type:nvarchar(120)" json:"contactPerson"`
	TinNumber     string            `gorm:"type:varchar(24);index:idx_transporterTin" json:"tinNumber"`
	VrnNumber     string            `gorm:"type:varchar(24)" json:"vrnNumber"`
	License       string            `gorm:"type:varchar(40)" json:"license"`
	CountryCode   *string           `gorm:"type:varchar(2);index:idx_transporterCountry" json:"countryCode,omitempty"`
	Country       *Country          `gorm:"foreignKey:CountryCode;constraint:OnDelete:NO ACTION;" json:"country,omitempty"`
	Address       string            `gorm:"type:nvarchar(120)" json:"address"`
	Address2      string            `gorm:"type:nvarchar(120)" json:"address2"`
	AeoEndDate    *time.Time        `json:"aeoEndDate,omitempty"`
	IsActive      bool              `gorm:"default:1;not null;index:idx_transporterActiveName,priority:1" json:"isActive"`
	HasData       bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	DjangoID      uint              `gorm:"index" json:"-"`
}

func (m *Transporter) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// Driver is assigned to an ILO truck line.
type Driver struct {
	ID             uint              `gorm:"primaryKey" json:"-"`
	UID            string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueDriverUID;not null" json:"id"`
	ContentType    types.ContentType `gorm:"default:85;not null;check:ContentType=85" json:"-"`
	Name           string            `gorm:"type:nvarchar(160);not null;index:idx_driverActiveName,priority:2" json:"name"`
	LicenseNumber  string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueDriverLicense" json:"licenseNumber"`
	LicenseExpires *time.Time        `gorm:"index:idx_driverLicenseExpires" json:"licenseExpires,omitempty"`
	Phone          string            `gorm:"type:varchar(24);index:idx_driverPhone" json:"phone"`
	Email          string            `gorm:"type:varchar(160)" json:"email"`
	IsActive       bool              `gorm:"default:1;not null;index:idx_driverActiveName,priority:1" json:"isActive"`
	HasData        bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	DjangoID       uint              `gorm:"index" json:"-"`
}

func (m *Driver) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// Truck is a horse/trailer used at the one-stop gantry.
type Truck struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	UID          string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTruckUID;not null" json:"id"`
	ContentType  types.ContentType `gorm:"default:86;not null;check:ContentType=86" json:"-"`
	PlateNumber  string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueTruckPlate;index:idx_truckActivePlate,priority:2" json:"plateNumber"`
	Trailer      string            `gorm:"type:varchar(20);index:idx_truckTrailer" json:"trailer"`
	TrailerTwo   string            `gorm:"type:varchar(20)" json:"trailerTwo"`
	DisplayPlate string            `gorm:"-" json:"displayPlate"`
	VehicleType  types.VehicleType `gorm:"type:varchar(16);not null;default:pending" json:"vehicleType"`
	LoadingType  types.LoadingType `gorm:"type:varchar(16);not null;default:top" json:"loadingType"`
	LngCng       bool              `gorm:"default:0;not null" json:"lngCng"`
	Mplw         decimal.Decimal   `gorm:"type:decimal(18,0);not null;default:0" json:"mplw"`
	Gcwr         decimal.Decimal   `gorm:"type:decimal(18,0);not null;default:0" json:"gcwr"`
	TareWeight   decimal.Decimal   `gorm:"type:decimal(18,0);not null;default:0" json:"tareWeight"`
	IsActive     bool              `gorm:"default:1;not null;index:idx_truckActivePlate,priority:1" json:"isActive"`
	HasData      bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	DjangoID     uint              `gorm:"index" json:"-"`
	Tanks        []TruckTank       `gorm:"foreignKey:TruckID;constraint:OnDelete:NO ACTION;" json:"tanks,omitempty"`
}

func (m *Truck) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *Truck) BeforeSave(*gorm.DB) error {
	if strings.EqualFold(string(m.VehicleType), "horse") {
		m.VehicleType = types.VehiclePulling
	}
	return nil
}

func (m *Truck) AfterFind(*gorm.DB) error {
	if strings.EqualFold(string(m.VehicleType), "horse") {
		m.VehicleType = types.VehiclePulling
	}
	m.DisplayPlate = TruckComboPlate(m.PlateNumber, m.Trailer, m.TrailerTwo)
	return nil
}

// TruckComboPlate is horse / tank one / tank two, stopping at the last tank that exists.
func TruckComboPlate(horse, tankOne, tankTwo string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{horse, tankOne, tankTwo} {
		if s := strings.TrimSpace(p); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "/")
}

// GantryLoadingRequest is an ILR (Internal Loading Request). Content type 80 is stable.
type GantryLoadingRequest struct {
	ID                    uint                       `gorm:"primaryKey" json:"-"`
	UID                   string                     `gorm:"type:varchar(26);uniqueIndex:idx_uniqueGLRUID;not null" json:"id"`
	ContentType           types.ContentType          `gorm:"default:80;not null;check:ContentType=80" json:"-"`
	DocumentNumber        string                     `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueGLRNo" json:"documentNumber"`
	OrderDate             time.Time                  `gorm:"not null;index;index:idx_glrList,priority:2" json:"orderDate"`
	Description           string                     `gorm:"type:nvarchar(120);not null" json:"description"`
	CustomerID            uint                       `gorm:"index;not null" json:"-"`
	Customer              Customer                   `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer"`
	ProductID             uint                       `gorm:"index;not null" json:"-"`
	Product               Product                    `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	ByProductID           *uint                      `gorm:"index" json:"-"`
	ByProduct             *Product                   `gorm:"foreignKey:ByProductID;constraint:OnDelete:NO ACTION;" json:"byProduct,omitempty"`
	ByProductQuantity     decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"byProductQuantity"`
	StockStatusID         uint                       `gorm:"index;not null" json:"-"`
	StockStatus           StockStatus                `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus"`
	Quantity              decimal.Decimal            `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter            decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne           decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	BatchNumber           string                     `gorm:"type:varchar(30);not null;default:'';index:idx_glrBatch" json:"batchNumber"`
	CustomerOrderNumber   string                     `gorm:"type:varchar(40)" json:"customerOrderNumber"`
	LoadingOrderAvailable bool                       `gorm:"default:0;not null" json:"loadingOrderAvailable"`
	ValidContract         bool                       `gorm:"default:0;not null" json:"validContract"`
	SnapshotFinal         decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"snapshotFinal"`
	SnapshotProvision     decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"snapshotProvision"`
	SnapshotHold          decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"snapshotHold"`
	SnapshotFree          decimal.Decimal            `gorm:"type:decimal(18,3);not null;default:0" json:"snapshotFree"`
	Status                types.OrderStatus          `gorm:"type:varchar(20);not null;default:draft;index;index:idx_glrList,priority:1" json:"status"`
	Notes                 string                     `gorm:"type:nvarchar(400)" json:"notes"`
	CreatedByID           uint                       `gorm:"index;not null" json:"-"`
	CreatedBy             User                       `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator               *CreatedByRef              `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt             time.Time                  `json:"createdAt"`
	UpdatedAt             time.Time                  `json:"updatedAt"`
	DjangoID              uint                       `gorm:"index" json:"-"`
	ApprovalTrail         ApprovalTrail              `gorm:"type:nvarchar(max)" json:"-"`
	Lines                 []GantryLoadingLine        `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
	Vessels               []GantryRequestVessel      `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"vessels,omitempty"`
	StockPositions        []GantryStockPosition      `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"stockPositions,omitempty"`
	Outstanding           *GantryCustomerOutstanding `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"outstanding,omitempty"`
	Charges               []GantryOutstandingCharge  `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"charges,omitempty"`
	IloExpiryDays         int                        `gorm:"-" json:"iloExpiryDays,omitempty"`
	Approvals             []ILRApproval              `gorm:"-" json:"approvals,omitempty"`
}

// ILRApproval is a workflow trail row printed on the ILR at any stage.
type ILRApproval struct {
	ApprovedOn string `json:"approvedOn"`
	ApprovedBy string `json:"approvedBy"`
	Title      string `json:"title"`
	Comment    string `json:"comment"`
	ActType    string `json:"actType"`
	ActName    string `json:"actName"`
}

func (m *GantryLoadingRequest) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *GantryLoadingRequest) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *GantryLoadingRequest) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	if m.ByProductID != nil {
		MarkHasData(tx, &Product{}, *m.ByProductID)
	}
	MarkHasData(tx, &StockStatus{}, m.StockStatusID)
	return nil
}

// GantryRequestVessel is a parcel allocation on an ILR (Django ILRVessel).
type GantryRequestVessel struct {
	ID            uint            `gorm:"primaryKey" json:"-"`
	UID           string          `gorm:"type:varchar(26);uniqueIndex:idx_uniqueILRVesselUID;not null" json:"id"`
	RequestID     uint            `gorm:"uniqueIndex:idx_uniqueILRVessel,priority:1;index;not null" json:"-"`
	VesselID      uint            `gorm:"uniqueIndex:idx_uniqueILRVessel,priority:2;index;not null" json:"-"`
	Vessel        Vessel          `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel"`
	VesselDate    time.Time       `gorm:"uniqueIndex:idx_uniqueILRVessel,priority:3;not null" json:"vesselDate"`
	ProductID     uint            `gorm:"uniqueIndex:idx_uniqueILRVessel,priority:4;index;not null" json:"-"`
	Product       Product         `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	StockStatusID uint            `gorm:"uniqueIndex:idx_uniqueILRVessel,priority:5;index;not null" json:"-"`
	StockStatus   StockStatus     `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus"`
	Quantity      decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CubicMeter    decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne   decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	LoadedQty     decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"loadedQty"`
	FinancialHold bool            `gorm:"default:0;not null" json:"financialHold"`
	DjangoID      uint            `gorm:"index" json:"-"`
}

func (m *GantryRequestVessel) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *GantryRequestVessel) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Vessel{}, m.VesselID)
	MarkHasData(tx, &Product{}, m.ProductID)
	return nil
}

// GantryStockPosition is the customer/product book snapshot frozen on the ILR.
type GantryStockPosition struct {
	ID           uint            `gorm:"primaryKey" json:"-"`
	RequestID    uint            `gorm:"uniqueIndex:idx_uniqueILRPosition,priority:1;not null" json:"-"`
	ProductID    uint            `gorm:"uniqueIndex:idx_uniqueILRPosition,priority:2;not null" json:"-"`
	Product      Product         `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	TotalBalance decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"totalBalance"`
	HoldQty      decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"holdQty"`
	FreeQty      decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"freeQty"`
	FinalQty     decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"finalQty"`
	Price        decimal.Decimal `gorm:"type:decimal(18,3);not null;default:0" json:"price"`
	DjangoID     uint            `gorm:"index" json:"-"`
}

// GantryCustomerOutstanding is storage / W&M / TBS debt frozen on the ILR.
type GantryCustomerOutstanding struct {
	ID               uint            `gorm:"primaryKey" json:"-"`
	RequestID        uint            `gorm:"uniqueIndex:idx_uniqueILROutstanding;not null" json:"-"`
	StorageTZS       decimal.Decimal `gorm:"type:decimal(18,2);not null;default:0" json:"storageTzs"`
	StorageUSD       decimal.Decimal `gorm:"type:decimal(18,2);not null;default:0" json:"storageUsd"`
	WeightMeasureTZS decimal.Decimal `gorm:"type:decimal(18,2);not null;default:0" json:"weightMeasureTzs"`
	TbsTZS           decimal.Decimal `gorm:"type:decimal(18,2);not null;default:0" json:"tbsTzs"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	DjangoID         uint            `gorm:"index" json:"-"`
}

// GantryOutstandingCharge is one billing debt line frozen on the ILR (charge / currency / amount).
type GantryOutstandingCharge struct {
	ID           uint            `gorm:"primaryKey" json:"-"`
	RequestID    uint            `gorm:"index;not null" json:"-"`
	Charge       string          `gorm:"type:nvarchar(80);not null" json:"charge"`
	CurrencyCode string          `gorm:"type:varchar(3);not null;default:TZS" json:"currencyCode"`
	Amount       decimal.Decimal `gorm:"type:decimal(18,2);not null;default:0" json:"amount"`
}

// GantryLoadingLine is one truck ILO created from an approved ILR.
type GantryLoadingLine struct {
	ID                  uint                  `gorm:"primaryKey" json:"-"`
	UID                 string                `gorm:"type:varchar(26);uniqueIndex:idx_uniqueGLOUID;not null" json:"id"`
	ContentType         types.ContentType     `gorm:"default:81;not null;check:ContentType=81" json:"-"`
	RequestID           uint                  `gorm:"index;index:idx_gloCommit,priority:1;not null" json:"-"`
	DocumentNumber      string                `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueGLONo" json:"documentNumber"`
	CustomerOrderNumber string                `gorm:"type:varchar(40)" json:"customerOrderNumber"`
	ProductID           uint                  `gorm:"index;index:idx_gloCommit,priority:2;not null" json:"-"`
	Product             Product               `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	ByProductID         *uint                 `gorm:"index" json:"-"`
	ByProduct           *Product              `gorm:"foreignKey:ByProductID;constraint:OnDelete:NO ACTION;" json:"byProduct,omitempty"`
	ByProductQuantity   decimal.Decimal       `gorm:"type:decimal(18,3);not null;default:0" json:"byProductQuantity"`
	TransporterID       *uint                 `gorm:"index" json:"-"`
	Transporter         *Transporter          `gorm:"foreignKey:TransporterID;constraint:OnDelete:NO ACTION;" json:"transporter,omitempty"`
	TransporterName     string                `gorm:"type:nvarchar(160)" json:"transporterName"`
	DriverID            *uint                 `gorm:"index" json:"-"`
	Driver              *Driver               `gorm:"foreignKey:DriverID;constraint:OnDelete:NO ACTION;" json:"driver,omitempty"`
	DriverName          string                `gorm:"type:nvarchar(160)" json:"driverName"`
	TruckID             *uint                 `gorm:"index" json:"-"`
	Truck               *Truck                `gorm:"foreignKey:TruckID;constraint:OnDelete:NO ACTION;" json:"truck,omitempty"`
	TruckPlate          string                `gorm:"type:varchar(80);index:idx_gloPlate" json:"truckPlate"`
	HorsePlate          string                `gorm:"type:varchar(20)" json:"horsePlate"`
	TrailerOnePlate     string                `gorm:"type:varchar(20)" json:"trailerOnePlate"`
	TrailerTwoPlate     string                `gorm:"type:varchar(20)" json:"trailerTwoPlate"`
	DestinationID       *uint                 `gorm:"index" json:"-"`
	ToDestination       *Destination          `gorm:"foreignKey:DestinationID;constraint:OnDelete:NO ACTION;" json:"toDestination,omitempty"`
	Destination         string                `gorm:"type:nvarchar(160)" json:"destination"`
	DistrictID          *uint                 `gorm:"index" json:"-"`
	ToDistrict          *District             `gorm:"foreignKey:DistrictID;constraint:OnDelete:NO ACTION;" json:"toDistrict,omitempty"`
	District            string                `gorm:"type:nvarchar(80)" json:"district"`
	EwuraLicense        string                `gorm:"type:varchar(40)" json:"ewuraLicense"`
	ExpirationDate      *time.Time            `gorm:"index:idx_gloExpire;index:idx_gloExpireScan,priority:3" json:"expirationDate,omitempty"`
	RequestedQty        decimal.Decimal       `gorm:"type:decimal(18,3);not null" json:"requestedQty"`
	CubicMeter          decimal.Decimal       `gorm:"type:decimal(18,3);not null;default:0" json:"cubicMeter"`
	MetricTonne         decimal.Decimal       `gorm:"type:decimal(18,3);not null;default:0" json:"metricTonne"`
	LoadedQty           decimal.Decimal       `gorm:"type:decimal(18,3);not null;default:0" json:"loadedQty"`
	LoadedAt            *time.Time            `json:"loadedAt,omitempty"`
	Amended             bool                  `gorm:"default:0;not null;index:idx_gloAmended;index:idx_gloCommit,priority:4;index:idx_gloExpireScan,priority:2" json:"amended"`
	IsActive            bool                  `gorm:"default:1;not null;index:idx_gloActive;index:idx_gloCommit,priority:3;index:idx_gloExpireScan,priority:1" json:"isActive"`
	SentToAlma          bool                  `gorm:"default:0;not null" json:"sentToAlma"`
	AlmaFileName        string                `gorm:"type:varchar(80)" json:"almaFileName,omitempty"`
	AlmaSentAt          *time.Time            `json:"almaSentAt,omitempty"`
	Status              types.OrderStatus     `gorm:"type:varchar(20);not null;default:draft;index:idx_gloStatus;index:idx_gloList,priority:1;index:idx_gloCommit,priority:5" json:"status"`
	CreatedAt           time.Time             `gorm:"index:idx_gloCreatedAt;index:idx_gloList,priority:2" json:"createdAt"`
	UpdatedAt           time.Time             `json:"updatedAt"`
	DjangoID            uint                  `gorm:"index" json:"-"`
	Request             *GantryLoadingRequest `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"request,omitempty"`
}

func (m *GantryLoadingLine) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *GantryLoadingLine) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Product{}, m.ProductID)
	markHasDataPtr(tx, &Transporter{}, m.TransporterID)
	markHasDataPtr(tx, &Driver{}, m.DriverID)
	markHasDataPtr(tx, &Truck{}, m.TruckID)
	markHasDataPtr(tx, &Destination{}, m.DestinationID)
	markHasDataPtr(tx, &District{}, m.DistrictID)
	return nil
}

// PumpOverRequest is a pipeline delivery order (Django PDO).
type PumpOverRequest struct {
	ID             uint              `gorm:"primaryKey" json:"-"`
	UID            string            `gorm:"type:varchar(26);uniqueIndex:idx_uniquePDOUID;not null" json:"id"`
	ContentType    types.ContentType `gorm:"default:82;not null;check:ContentType=82" json:"-"`
	DocumentNumber string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniquePDONo" json:"documentNumber"`
	OrderDate      time.Time         `gorm:"not null;index;index:idx_pdoList,priority:2" json:"orderDate"`
	CustomerID     uint              `gorm:"index;index:idx_pdoCommit,priority:1;not null" json:"-"`
	Customer       Customer          `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"customer"`
	ProductID      uint              `gorm:"index;index:idx_pdoCommit,priority:2;not null" json:"-"`
	Product        Product           `gorm:"foreignKey:ProductID;constraint:OnDelete:NO ACTION;" json:"product"`
	// StockStatusID is the first vessel parcel (list/reserve). Distinct
	// statuses live on Vessels — one request can mix Local, Transit, Congo, …
	StockStatusID       uint              `gorm:"index;not null" json:"-"`
	StockStatus         StockStatus       `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus"`
	DepotID             uint              `gorm:"index;not null" json:"-"`
	Depot               Depot             `gorm:"foreignKey:DepotID;constraint:OnDelete:NO ACTION;" json:"depot"`
	Quantity            decimal.Decimal   `gorm:"type:decimal(18,3);not null" json:"quantity"`
	CustomerOrderNumber string            `gorm:"type:varchar(40)" json:"customerOrderNumber"`
	SnapshotFree        decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"snapshotFree"`
	Status              types.OrderStatus `gorm:"type:varchar(20);not null;default:draft;index;index:idx_pdoList,priority:1;index:idx_pdoCommit,priority:3" json:"status"`
	Notes               string            `gorm:"type:nvarchar(400)" json:"notes"`
	CreatedByID         uint              `gorm:"index;not null" json:"-"`
	CreatedBy           User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator             *CreatedByRef     `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
	DjangoID            uint              `gorm:"index" json:"-"`
	ApprovalTrail       ApprovalTrail     `gorm:"type:nvarchar(max)" json:"-"`
	Vessels             []PumpOverVessel  `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"vessels,omitempty"`
}

func (m *PumpOverRequest) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *PumpOverRequest) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func (m *PumpOverRequest) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Customer{}, m.CustomerID)
	MarkHasData(tx, &Product{}, m.ProductID)
	MarkHasData(tx, &Depot{}, m.DepotID)
	return nil
}

type PumpOverVessel struct {
	ID            uint            `gorm:"primaryKey" json:"-"`
	UID           string          `gorm:"type:varchar(26);uniqueIndex:idx_uniquePDOVesselUID;not null" json:"id"`
	RequestID     uint            `gorm:"uniqueIndex:idx_uniquePDOVessel,priority:1;index;not null" json:"-"`
	VesselID      uint            `gorm:"uniqueIndex:idx_uniquePDOVessel,priority:2;index;not null" json:"-"`
	Vessel        Vessel          `gorm:"foreignKey:VesselID;constraint:OnDelete:NO ACTION;" json:"vessel"`
	VesselDate    time.Time       `gorm:"uniqueIndex:idx_uniquePDOVessel,priority:3;not null" json:"vesselDate"`
	StockStatusID uint            `gorm:"uniqueIndex:idx_uniquePDOVessel,priority:4;index;not null" json:"-"`
	StockStatus   StockStatus     `gorm:"foreignKey:StockStatusID;constraint:OnDelete:NO ACTION;" json:"stockStatus"`
	Quantity      decimal.Decimal `gorm:"type:decimal(18,3);not null" json:"quantity"`
	DjangoID      uint            `gorm:"index" json:"-"`
}

func (m *PumpOverVessel) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *PumpOverVessel) AfterCreate(tx *gorm.DB) error {
	MarkHasData(tx, &Vessel{}, m.VesselID)
	return nil
}

// PumpOverReport is the executed pump-over (Django DeliveryReport).
type PumpOverReport struct {
	ID              uint              `gorm:"primaryKey" json:"-"`
	UID             string            `gorm:"type:varchar(26);uniqueIndex:idx_uniquePORUID;not null" json:"id"`
	ContentType     types.ContentType `gorm:"default:83;not null;check:ContentType=83" json:"-"`
	DocumentNumber  string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniquePORNo" json:"documentNumber"`
	RequestID       uint              `gorm:"index;not null" json:"-"`
	Request         PumpOverRequest   `gorm:"foreignKey:RequestID;constraint:OnDelete:NO ACTION;" json:"request"`
	ReportDate      time.Time         `gorm:"not null;index:idx_porReportDate;index:idx_porList,priority:2" json:"reportDate"`
	ActualDelivered decimal.Decimal   `gorm:"type:decimal(18,3);not null" json:"actualDelivered"`
	ActualReceived  decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"actualReceived"`
	Variance        decimal.Decimal   `gorm:"type:decimal(18,3);not null;default:0" json:"variance"`
	Status          types.OrderStatus `gorm:"type:varchar(20);not null;default:draft;index;index:idx_porList,priority:1" json:"status"`
	CreatedByID     uint              `gorm:"index;not null" json:"-"`
	CreatedBy       User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	Creator         *CreatedByRef     `gorm:"-" json:"createdBy,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	DjangoID        uint              `gorm:"index" json:"-"`
}

func (m *PumpOverReport) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m *PumpOverReport) AfterFind(*gorm.DB) error {
	m.Creator = StampCreator(&m.CreatedBy)
	return nil
}

func assignUID(uid *string) error {
	if uid == nil || *uid != "" {
		return nil
	}
	v, err := utils.GetULID()
	if err != nil {
		return err
	}
	*uid = v
	return nil
}
