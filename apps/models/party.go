package models

import (
	"fmt"
	"strings"
	"time"

	"dfms/pkg/types"
	"dfms/utils"

	"gorm.io/gorm"
)

// Customer is an OMC that stores product at TIPER.
type Customer struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	UID          string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCustomerUID;not null" json:"id"`
	ContentType  types.ContentType `gorm:"default:47;not null;check:ContentType=47" json:"-"`
	Code         string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueCustomerCode" json:"code"`
	Name         string            `gorm:"type:nvarchar(160);not null;index:idx_customerName;index:idx_customerActiveName,priority:2;check:chk_customer_name,[Name] <> ''" json:"name"`
	Email        string            `gorm:"type:varchar(160);index:idx_customerEmail" json:"email"`
	Phone        string            `gorm:"type:varchar(24);index:idx_customerPhone" json:"phone"`
	TinNumber    string            `gorm:"type:varchar(40);index:idx_customerTin" json:"tinNumber"`
	KycNumber    string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_uniqueCustomerKyc;index:idx_customerKyc" json:"kycNumber"`
	VrnNumber    string            `gorm:"type:varchar(40);index:idx_customerVrn" json:"vrnNumber"`
	EwuraLicense string            `gorm:"type:varchar(40);not null;index:idx_customerEwura" json:"ewuraLicense"`
	IsActive     bool              `gorm:"default:1;not null;index:idx_customerActiveName,priority:1" json:"isActive"`
	HasData      bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID  uint              `gorm:"index;not null" json:"-"`
	CreatedBy    User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	// DjangoID is the retired SageCustomer.id. Child rows (attachments, ILR, …)
	// look up this value and store the new Customer.ID as their FK.
	DjangoID uint `gorm:"index" json:"-"`

	BillingAccounts []CustomerBillingAccount `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"billingAccounts,omitempty"`
}

func (m *Customer) BeforeCreate(*gorm.DB) error {
	if m.UID == "" {
		uid, err := utils.GetULID()
		if err != nil {
			return err
		}
		m.UID = uid
	}
	return nil
}

func (m *Customer) AfterCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.Code) != "" {
		return nil
	}
	code := fmt.Sprintf("%d", m.ID+20000)
	m.Code = code
	return tx.Model(m).Update("Code", code).Error
}

// CustomerBillingAccount maps a customer+fee+currency to a Sage AR account.
// Unique on (Customer, Fee, Currency). The same SageAccount may be reused
// for several fees on this customer; SageAccountOwner forbids sharing it
// with any other customer or supplier.
type CustomerBillingAccount struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	UID          string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueCBAUID;not null" json:"id"`
	ContentType  types.ContentType `gorm:"default:48;not null;check:ContentType=48" json:"-"`
	CustomerID   uint              `gorm:"uniqueIndex:idx_uniqueCBA,priority:1;not null" json:"-"`
	Customer     Customer          `gorm:"foreignKey:CustomerID;constraint:OnDelete:NO ACTION;" json:"-"`
	FeeCode      types.FeeCode     `gorm:"type:varchar(10);uniqueIndex:idx_uniqueCBA,priority:2;not null" json:"feeCode"`
	Fee          Fee               `gorm:"foreignKey:FeeCode;constraint:OnDelete:NO ACTION;" json:"-"`
	CurrencyCode string            `gorm:"type:varchar(3);uniqueIndex:idx_uniqueCBA,priority:3;not null" json:"currencyCode"`
	Currency     Currency          `gorm:"foreignKey:CurrencyCode;constraint:OnDelete:NO ACTION;" json:"-"`
	SageAccount  string            `gorm:"type:varchar(40);not null;index:idx_cbaSageAccount" json:"sageAccount"`
	SageName     string            `gorm:"type:nvarchar(160)" json:"sageName"`
	BillingUnit  string            `gorm:"type:varchar(10);not null;default:M3" json:"billingUnit"`
	IsForeign    bool              `gorm:"default:0;not null" json:"isForeign"`
	IsActive     bool              `gorm:"default:1;not null" json:"isActive"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

func (m *CustomerBillingAccount) BeforeCreate(*gorm.DB) error {
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

// Supplier is billed for SRT first-cycle storage.
type Supplier struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	UID           string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueSupplierUID;not null" json:"id"`
	ContentType   types.ContentType `gorm:"default:49;not null;check:ContentType=49" json:"-"`
	Code          string            `gorm:"type:varchar(20);not null;uniqueIndex:idx_uniqueSupplierCode" json:"code"`
	Name          string            `gorm:"type:nvarchar(160);not null;index:idx_supplierName;index:idx_supplierActiveName,priority:2" json:"name"`
	Email         string            `gorm:"type:varchar(160);index:idx_supplierEmail" json:"email"`
	Phone         string            `gorm:"type:varchar(24);index:idx_supplierPhone" json:"phone"`
	Mobile        string            `gorm:"type:varchar(24)" json:"mobile"`
	ContactPerson string            `gorm:"type:nvarchar(120)" json:"contactPerson"`
	TinNumber     string            `gorm:"type:varchar(40);index:idx_supplierTin" json:"tinNumber"`
	CountryCode   *string           `gorm:"type:varchar(2);index:idx_supplierCountry" json:"countryCode,omitempty"`
	Country       *Country          `gorm:"foreignKey:CountryCode;constraint:OnDelete:NO ACTION;" json:"country,omitempty"`
	Address       string            `gorm:"type:nvarchar(120)" json:"address"`
	Address2      string            `gorm:"type:nvarchar(120)" json:"address2"`
	IsActive      bool              `gorm:"default:1;not null;index:idx_supplierActiveName,priority:1" json:"isActive"`
	HasData       bool              `gorm:"default:0;not null" json:"hasData"`
	CreatedByID   uint              `gorm:"index;not null" json:"-"`
	CreatedBy     User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`

	BillingAccounts []SupplierBillingAccount `gorm:"foreignKey:SupplierID;constraint:OnDelete:NO ACTION;" json:"billingAccounts,omitempty"`
}

func (m *Supplier) BeforeCreate(*gorm.DB) error {
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

type SupplierBillingAccount struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	UID          string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueSBAUID;not null" json:"id"`
	ContentType  types.ContentType `gorm:"default:50;not null;check:ContentType=50" json:"-"`
	SupplierID   uint              `gorm:"uniqueIndex:idx_uniqueSBA,priority:1;not null" json:"-"`
	Supplier     Supplier          `gorm:"foreignKey:SupplierID;constraint:OnDelete:NO ACTION;" json:"-"`
	FeeCode      types.FeeCode     `gorm:"type:varchar(10);uniqueIndex:idx_uniqueSBA,priority:2;not null" json:"feeCode"`
	Fee          Fee               `gorm:"foreignKey:FeeCode;constraint:OnDelete:NO ACTION;" json:"-"`
	CurrencyCode string            `gorm:"type:varchar(3);uniqueIndex:idx_uniqueSBA,priority:3;not null" json:"currencyCode"`
	Currency     Currency          `gorm:"foreignKey:CurrencyCode;constraint:OnDelete:NO ACTION;" json:"-"`
	SageAccount  string            `gorm:"type:varchar(40);not null;index:idx_sbaSageAccount" json:"sageAccount"`
	SageName     string            `gorm:"type:nvarchar(160)" json:"sageName"`
	BillingUnit  string            `gorm:"type:varchar(10);not null;default:M3" json:"billingUnit"`
	IsActive     bool              `gorm:"default:1;not null" json:"isActive"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// SageAccountOwner is the exclusive claim of a Sage Client.Account.
// One AR account can map to several fees for the same party, but cannot
// be shared across customers, or between a customer and a supplier.
type SageAccountOwner struct {
	SageAccount string `gorm:"primaryKey;type:varchar(40)" json:"sageAccount"`
	OwnerKind   string `gorm:"type:varchar(12);not null;index:idx_sageOwner,priority:1" json:"ownerKind"`
	OwnerID     uint   `gorm:"not null;index:idx_sageOwner,priority:2" json:"-"`
}

func (m *SupplierBillingAccount) BeforeCreate(*gorm.DB) error {
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

// EwuraPetroleumLicense is synced from the EWURA portal.
type EwuraPetroleumLicense struct {
	LicenseNumber string            `gorm:"primaryKey;type:varchar(40)" json:"licenseNumber"`
	ContentType   types.ContentType `gorm:"default:68;not null;check:ContentType=68" json:"-"`
	Licensee      string            `gorm:"type:nvarchar(200);not null;index" json:"licensee"`
	LicenseClass  string            `gorm:"type:varchar(80);index" json:"licenseClass"`
	LicenseType   string            `gorm:"type:varchar(80)" json:"licenseType"`
	Sector        string            `gorm:"type:varchar(80)" json:"sector"`
	ZoneName      string            `gorm:"type:varchar(80)" json:"zoneName"`
	RegionName    string            `gorm:"type:varchar(80)" json:"regionName"`
	DistrictName  string            `gorm:"type:varchar(80)" json:"districtName"`
	TinNumber     string            `gorm:"type:varchar(40);index" json:"tinNumber"`
	Phone         string            `gorm:"type:varchar(40)" json:"phone"`
	Email         string            `gorm:"type:varchar(160);index:idx_ewuraEmail" json:"email"`
	IssueDate     *time.Time        `json:"issueDate,omitempty"`
	ExpiryDate    *time.Time        `json:"expiryDate,omitempty"`
	IsActive      bool              `gorm:"default:1;not null" json:"isActive"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	CreatedAt     time.Time         `json:"createdAt"`
}
