package models

import (
	"time"

	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Destination is a delivery country or inland region (Django GantryDestination).
type Destination struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueDestinationUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:87;not null;check:ContentType=87" json:"-"`
	Name        string            `gorm:"type:nvarchar(80);not null;uniqueIndex:idx_uniqueDestinationName;index:idx_destinationActiveName,priority:2" json:"name"`
	IsCountry   bool              `gorm:"default:0;not null;index:idx_destinationCountry" json:"isCountry"`
	IsActive    bool              `gorm:"default:1;not null;index:idx_destinationActiveName,priority:1" json:"isActive"`
	HasData     bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt   time.Time         `json:"createdAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
}

func (m *Destination) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// District belongs to a destination (Django GantryDistrict).
type District struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	UID           string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueDistrictUID;not null" json:"id"`
	ContentType   types.ContentType `gorm:"default:88;not null;check:ContentType=88" json:"-"`
	DestinationID uint              `gorm:"uniqueIndex:idx_uniqueDistrictName,priority:1;not null" json:"-"`
	Destination   Destination       `gorm:"foreignKey:DestinationID;constraint:OnDelete:NO ACTION;" json:"destination,omitempty"`
	Name          string            `gorm:"type:nvarchar(80);uniqueIndex:idx_uniqueDistrictName,priority:2;index:idx_districtActiveName,priority:2;not null" json:"name"`
	IsActive      bool              `gorm:"default:1;not null;index:idx_districtActiveName,priority:1" json:"isActive"`
	HasData       bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedAt     time.Time         `json:"createdAt"`
	DjangoID      uint              `gorm:"index" json:"-"`
}

func (m *District) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// TruckTank is a physical tank on a horse or trailer (Django GantryTank).
// TruckID is optional: a retired tank keeps its plate/calibrations after the
// truck is unlinked.
type TruckTank struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTruckTankUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:89;not null;check:ContentType=89" json:"-"`
	TruckID     *uint             `gorm:"index:idx_truckTankTruck" json:"-"`
	Truck       *Truck            `gorm:"foreignKey:TruckID;constraint:OnDelete:NO ACTION;" json:"truck,omitempty"`
	PlateNumber string            `gorm:"type:varchar(20);not null;index:idx_truckTankPlate" json:"plateNumber"`
	Index       int               `gorm:"not null;default:1" json:"index"`
	IsActive    bool              `gorm:"default:1;not null" json:"isActive"`
	CreatedAt   time.Time         `json:"createdAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
}

func (m *TruckTank) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// TankCalibration is a dated compartment map for a truck tank.
type TankCalibration struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCalUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:90;not null;check:ContentType=90" json:"-"`
	TankID      uint              `gorm:"index:idx_calTank;index:idx_calLookup,priority:1;not null" json:"-"`
	Tank        TruckTank         `gorm:"foreignKey:TankID;constraint:OnDelete:NO ACTION;" json:"tank,omitempty"`
	ValidFrom   time.Time         `gorm:"not null;index:idx_calFrom;index:idx_calLookup,priority:3" json:"validFrom"`
	ValidTo     time.Time         `gorm:"not null;index:idx_calTo" json:"validTo"`
	IsActive    bool              `gorm:"default:1;not null;index:idx_calActive;index:idx_calLookup,priority:2" json:"isActive"`
	CreatedAt   time.Time         `json:"createdAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
	Lines       []TankCompartment `gorm:"foreignKey:CalibrationID;constraint:OnDelete:NO ACTION;" json:"lines,omitempty"`
}

func (m *TankCalibration) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

func (m TankCalibration) Expired(at time.Time) bool {
	day := at.UTC().Truncate(24 * time.Hour)
	from := m.ValidFrom.UTC().Truncate(24 * time.Hour)
	to := m.ValidTo.UTC().Truncate(24 * time.Hour)
	return day.Before(from) || day.After(to)
}

// TankCompartment is one cell on a calibration (index + capacity in litres).
type TankCompartment struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	UID           string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCompCellUID;not null" json:"id"`
	ContentType   types.ContentType `gorm:"default:91;not null;check:ContentType=91" json:"-"`
	CalibrationID uint              `gorm:"uniqueIndex:idx_uniqueCalIndex,priority:1;not null" json:"-"`
	Index         int               `gorm:"uniqueIndex:idx_uniqueCalIndex,priority:2;not null" json:"index"`
	Capacity      decimal.Decimal   `gorm:"type:decimal(18,0);not null" json:"capacity"`
	DjangoID      uint              `gorm:"index" json:"-"`
}

func (m *TankCompartment) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// RfidBadge is an ATLAS NEO gantry badge issued at dispatch.
type RfidBadge struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueRfidUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:92;not null;check:ContentType=92" json:"-"`
	Code        string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueRfidCode" json:"code"`
	IsActive    bool              `gorm:"default:1;not null" json:"isActive"`
	IsAvailable bool              `gorm:"default:1;not null;index:idx_rfidAvail" json:"isAvailable"`
	CreatedAt   time.Time         `json:"createdAt"`
	DjangoID    uint              `gorm:"index" json:"-"`
}

func (m *RfidBadge) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }
