package models

import (
	"fmt"
	"time"

	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Country is an ISO 3166-1 country used on the company profile and master
// data (alpha-2 code + official short name). Seeded from the ISO catalogue.
type Country struct {
	Code        string            `gorm:"primaryKey;type:varchar(2);not null" json:"code"`
	Name        string            `gorm:"type:nvarchar(120);not null;index:idx_countryName;check:chk_country_name,[Name] <> ''" json:"name"`
	Alpha3      string            `gorm:"type:varchar(3);not null;uniqueIndex:idx_uniqueCountryAlpha3" json:"alpha3"`
	Numeric     string            `gorm:"type:varchar(3);not null;uniqueIndex:idx_uniqueCountryNumeric" json:"numeric"`
	IsActive    bool              `gorm:"default:1;not null;index:idx_countryActive" json:"isActive"`
	ContentType types.ContentType `gorm:"default:22;not null;check:ContentType=22" json:"-"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// Currency is an ISO 4217 currency used on billing runs and Sage posting.
// Code is the only identifier (Sage 200 and Sage 300 both key by CurrencyCode).
type Currency struct {
	Code string `gorm:"primaryKey;type:varchar(3);not null" json:"code"`
	Name string `gorm:"type:nvarchar(100);not null;check:chk_currency_name,[Name] <> ''" json:"name"`
	// nvarchar: ₹ and other ISO symbols are outside Latin-1; varchar stores them as ?.
	Symbol         string            `gorm:"type:nvarchar(16)" json:"symbol"`
	MinOutstanding decimal.Decimal   `gorm:"type:decimal(18,2);not null;default:0;check:MinOutstanding >= 0" json:"minOutstanding"`
	IsActive       bool              `gorm:"default:1;not null;index:idx_currencyActive" json:"isActive"`
	ContentType    types.ContentType `gorm:"default:20;not null;check:ContentType=20" json:"-"`
	CreatedByID    uint              `gorm:"index;not null" json:"-"`
	CreatedBy      User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION;" json:"-"`
	CreatedAt      time.Time         `json:"-"`
	UpdatedAt      time.Time         `json:"-"`
}

// Company holds the organisation's letterhead/registration details. A single
// row (ID = 1) is maintained and printed on reports and documents.
type Company struct {
	// Singleton: seed and settings always use ID=1. CHECK rejects a second row.
	ID          uint              `gorm:"primaryKey;check:chk_company_singleton,[ID] = 1" json:"-"`
	ContentType types.ContentType `gorm:"default:7;not null;check:ContentType=7" json:"-"`
	Name        string            `gorm:"type:varchar(190);not null" json:"name,omitempty"`
	TinNumber   string            `gorm:"type:varchar(40)" json:"tinNumber,omitempty"`
	VrnNumber   string            `gorm:"type:varchar(40)" json:"vrnNumber,omitempty"`
	IsoNumber   string            `gorm:"type:varchar(80)" json:"isoNumber,omitempty"`
	Address     string            `gorm:"type:varchar(255)" json:"address,omitempty"`
	Address2    string            `gorm:"type:varchar(255)" json:"address2,omitempty"`
	City        string            `gorm:"type:varchar(80)" json:"city,omitempty"`
	PostalCode  string            `gorm:"type:varchar(30)" json:"postalCode,omitempty"`
	Country     string            `gorm:"type:varchar(80)" json:"country,omitempty"`
	Phone       string            `gorm:"type:varchar(40)" json:"phone,omitempty"`
	Email       string            `gorm:"type:varchar(160)" json:"email,omitempty"`
	Website     string            `gorm:"type:varchar(160)" json:"website,omitempty"`
	// PortalURL is the public frontend base (e.g. https://dfms.tiper.co.tz)
	// used in email CTAs and report/document links behind nginx.
	PortalURL    string    `gorm:"type:varchar(255)" json:"portalUrl,omitempty"`
	CurrencyCode *string   `gorm:"type:varchar(3)" json:"currencyCode,omitempty"`
	Currency     Currency  `gorm:"foreignKey:CurrencyCode;constraint:OnDelete:NO ACTION;" json:"-"`
	LogoPath     string    `gorm:"type:varchar(255)" json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// DocumentNumberCounter ensures unique sequential numbering per document type
// per calendar month (e.g. ILR:0526-0143).
// The counter is incremented atomically in the same transaction that creates the document,
// so concurrent requests do not collide.
//
//	DocType: "ilr", Prefix: "ILR", MonthYear: "0526", Counter: 142
//	→ next document number: ILR:0526-0143
type DocumentNumberCounter struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	ContentType types.ContentType `gorm:"default:21;not null;check:ContentType=21" json:"-"`
	DocType     string            `gorm:"type:varchar(40);not null;uniqueIndex:idx_docCounterTypeMonth,priority:1" json:"docType"`
	MonthYear   string            `gorm:"type:varchar(4);not null;uniqueIndex:idx_docCounterTypeMonth,priority:2" json:"monthYear"`
	Prefix      string            `gorm:"type:varchar(10);not null;check:chk_doc_prefix,[Prefix] <> ''" json:"prefix"`
	Counter     int               `gorm:"not null;default:0;check:Counter >= 0" json:"counter"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// AssignDocumentNumber atomically reserves and returns the next document number
// for docType in the current calendar month, formatted PREFIX:MMYY-NNNN.
//
// MSSQL has no portable upsert+returning, so we attempt an atomic
// `UPDATE … SET Counter = Counter + 1 OUTPUT inserted.Counter`; if no row
// exists yet we insert the seed row and, on a race, retry the update. tx SHOULD
// be the same transaction that persists the owning document.
func AssignDocumentNumber(tx *gorm.DB, docType, prefix string) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("AssignDocumentNumber: nil *gorm.DB")
	}
	now := time.Now()
	monthYear := fmt.Sprintf("%02d%02d", int(now.Month()), now.Year()%100)

	bump := func() (int, bool, error) {
		var counters []int
		err := tx.Raw(
			`UPDATE DocumentNumberCounter SET Counter = Counter + 1, UpdatedAt = GETDATE()
			 OUTPUT inserted.Counter
			 WHERE DocType = ? AND MonthYear = ?`, docType, monthYear).Scan(&counters).Error
		if err != nil {
			return 0, false, err
		}
		if len(counters) == 0 {
			return 0, false, nil
		}
		return counters[0], true, nil
	}

	counter, ok, err := bump()
	if err != nil {
		return "", fmt.Errorf("assign document number (%s %s): %w", docType, monthYear, err)
	}
	if !ok {
		seed := DocumentNumberCounter{DocType: docType, MonthYear: monthYear, Prefix: prefix, Counter: 1}
		if err := tx.Create(&seed).Error; err != nil {
			// Lost an insert race: another caller created the row first; retry.
			counter, ok, err = bump()
			if err != nil || !ok {
				return "", fmt.Errorf("assign document number (%s %s): %w", docType, monthYear, err)
			}
		} else {
			counter = 1
		}
	}
	return fmt.Sprintf("%s:%s-%04d", prefix, monthYear, counter), nil
}
