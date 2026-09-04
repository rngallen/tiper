package models

import (
	"dfms/pkg/types"
	"dfms/utils"
	"time"

	"gorm.io/gorm"
)

// MailOutbox is a durable outbound email or SMS (welcome, password reset,
// workflow). OTP codes are never stored here — they expire in five minutes
// and are sent live only.
type MailOutbox struct {
	ID            string            `gorm:"type:varchar(26);primaryKey" json:"id"`
	ContentType   types.ContentType `gorm:"default:77;not null;check:ContentType=77" json:"-"`
	Kind          string            `gorm:"type:varchar(20);not null;index:idx_outboxKind" json:"kind"`
	Channel       string            `gorm:"type:varchar(10);not null" json:"channel"`
	Recipient     string            `gorm:"type:varchar(200);not null" json:"recipient"`
	Subject       string            `gorm:"type:nvarchar(240)" json:"subject"`
	Body          string            `gorm:"type:nvarchar(max);not null" json:"-"`
	Status        string            `gorm:"type:varchar(12);not null;index:idx_outboxDrain,priority:1" json:"status"`
	Attempts      uint              `gorm:"default:0;not null" json:"attempts"`
	MaxAttempts   uint              `gorm:"default:12;not null" json:"maxAttempts"`
	LastError     string            `gorm:"type:nvarchar(500)" json:"lastError,omitempty"`
	NextAttemptAt time.Time         `gorm:"not null;index:idx_outboxDrain,priority:2" json:"nextAttemptAt"`
	SentAt        *time.Time        `json:"sentAt,omitempty"`
	UserID        *uint             `gorm:"index" json:"-"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// BeforeCreate assigns a ULID public identifier.
func (m *MailOutbox) BeforeCreate(tx *gorm.DB) error {
	if m.ID != "" {
		return nil
	}
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	m.ID = uid
	return nil
}
