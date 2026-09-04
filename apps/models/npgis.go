package models

import (
	"time"

	"dfms/pkg/types"

	"gorm.io/gorm"
)

// TransactionSequence issues the monotonic TransId EWURA NPGIS requires.
type TransactionSequence struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	LastValue uint      `gorm:"not null;default:0" json:"lastValue"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NextTransactionID reserves the next NPGIS TransId inside tx.
func NextTransactionID(tx *gorm.DB) (uint, error) {
	var row TransactionSequence
	if err := tx.FirstOrCreate(&row, TransactionSequence{ID: 1}).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&row).Update("LastValue", gorm.Expr("LastValue + 1")).Error; err != nil {
		return 0, err
	}
	if err := tx.First(&row, 1).Error; err != nil {
		return 0, err
	}
	return row.LastValue, nil
}

// AlmaFileLog records every SAP3C write and SAP3R read (Django AuditAlmaLogs).
type AlmaFileLog struct {
	ID          uint                `gorm:"primaryKey" json:"-"`
	UID         string              `gorm:"type:varchar(26);uniqueIndex:idx_uniqueAlmaLogUID;not null" json:"id"`
	ContentType types.ContentType   `gorm:"default:98;not null;check:ContentType=98" json:"-"`
	Direction   types.AlmaDirection `gorm:"type:varchar(8);not null;index:idx_almaDir" json:"direction"`
	FileName    string              `gorm:"type:varchar(120);not null;index:idx_almaFile" json:"fileName"`
	OrderNumber string              `gorm:"type:varchar(40);index:idx_almaOrder" json:"orderNumber"`
	Message     string              `gorm:"type:nvarchar(500)" json:"message"`
	OK          bool                `gorm:"default:0;not null" json:"ok"`
	CreatedAt   time.Time           `gorm:"index:idx_almaAt" json:"createdAt"`
}

func (m *AlmaFileLog) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }

// NpgisSubmission is the EWURA outbox. A job posts unsent rows; no RabbitMQ.
type NpgisSubmission struct {
	ID             uint              `gorm:"primaryKey" json:"-"`
	UID            string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueNpgisUID;not null" json:"id"`
	ContentType    types.ContentType `gorm:"default:99;not null;check:ContentType=99" json:"-"`
	Kind           types.NpgisKind   `gorm:"type:varchar(20);not null;index:idx_npgisKind" json:"kind"`
	ReferenceType  string            `gorm:"type:varchar(40);not null" json:"referenceType"`
	ReferenceID    uint              `gorm:"index:idx_npgisRef;not null" json:"-"`
	DocumentNumber string            `gorm:"type:varchar(40);index:idx_npgisDoc" json:"documentNumber"`
	TransactionID  uint              `gorm:"index:idx_npgisTrans" json:"transactionId"`
	Payload        JSONMap           `gorm:"type:nvarchar(max)" json:"payload,omitempty"`
	Sent           bool              `gorm:"default:0;not null;index:idx_npgisSent" json:"sent"`
	SentAt         *time.Time        `json:"sentAt,omitempty"`
	Attempts       uint              `gorm:"not null;default:0" json:"attempts"`
	LastError      string            `gorm:"type:nvarchar(500)" json:"lastError,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func (m *NpgisSubmission) BeforeCreate(*gorm.DB) error { return assignUID(&m.UID) }
