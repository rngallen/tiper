package models

import (
	"dfms/pkg/types"
	"dfms/utils"
	"time"

	"gorm.io/gorm"
)

// AuditTrail is an append-only record of a meaningful system action (login,
// CRUD, approval decision, cron run, …). It captures who did
// what, from where, and an optional field-level before/after diff. Rows are
// never updated or deleted.
type AuditTrail struct {
	ID          string            `gorm:"type:varchar(26);primaryKey" json:"id"`
	ContentType types.ContentType `gorm:"default:11;not null;check:ContentType=11" json:"-"`
	UserID      *uint             `gorm:"index" json:"-"` // no FK — append-only; user may later be removed
	UserName    string            `gorm:"type:varchar(160);index:idx_auditUserName" json:"userName"`
	IPAddress   string            `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent   string            `gorm:"type:varchar(255)" json:"userAgent"`
	RequestID   string            `gorm:"type:varchar(60);index" json:"requestId"`
	Module      types.Module      `gorm:"type:varchar(60);index:idx_auditModuleAction,priority:1;index:idx_auditModuleCreated,priority:1;not null" json:"module"`
	Action      types.Action      `gorm:"type:varchar(60);index:idx_auditModuleAction,priority:2;not null" json:"action"`
	RecordType  types.ContentType `gorm:"index;index:idx_auditRecordTypeId,priority:1" json:"recordType"`
	RecordID    string            `gorm:"type:varchar(60);index:idx_auditRecordCreated,priority:1;index:idx_auditRecordTypeId,priority:2" json:"recordId"`
	Description string            `gorm:"type:varchar(500)" json:"description"`
	Changes     map[string]any    `gorm:"serializer:json;type:nvarchar(max)" json:"changes,omitempty"`
	Metadata    map[string]any    `gorm:"serializer:json;type:nvarchar(max)" json:"metadata,omitempty"`
	CreatedAt   time.Time         `gorm:"index;index:idx_auditModuleCreated,priority:2;index:idx_auditRecordCreated,priority:2;index:idx_auditModuleAction,priority:3;index:idx_auditRecordTypeId,priority:3" json:"createdAt"`
}

// BeforeCreate assigns a ULID public identifier.
func (a *AuditTrail) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	a.ID = uid
	return nil
}
