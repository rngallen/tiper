package models

import (
	"dfms/pkg/types"
	"dfms/utils"
	"time"

	"gorm.io/gorm"
)

// Attachment is generic file-attachment metadata. The binary lives on disk under
// the directory set in Settings → Attachments (default ./uploads); this row
// records where it is and which entity it belongs to.
// EntityType + EntityID form a polymorphic association (receipt, GLR,
// billing run, …) so the same table serves every attachable resource.
type Attachment struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	UID          string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueAttachmentUID;not null" json:"id"`
	ContentType  types.ContentType `gorm:"default:8;not null;check:ContentType=8" json:"-"`
	OriginalName string            `gorm:"type:varchar(255);not null;check:chk_attachment_name,[OriginalName] <> ''" json:"originalName"`
	StoredName   string            `gorm:"type:varchar(255);not null;uniqueIndex:idx_uniqueAttachmentStoredName" json:"-"`
	FilePath     string            `gorm:"type:varchar(500);not null" json:"-"`
	EntityID     uint              `gorm:"index:idx_attachmentEntity,priority:2;not null" json:"-"`
	EntityType   types.ContentType `gorm:"index:idx_attachmentEntity,priority:1;not null" json:"entityType"`
	Size         int64             `gorm:"not null;check:Size >= 0" json:"size"`
	ByteSize     string            `gorm:"type:varchar(20)" json:"byteSize"`
	Extension    string            `gorm:"type:varchar(20)" json:"extension"`
	Mime         string            `gorm:"type:varchar(120)" json:"mime"`
	CanPreview   bool              `gorm:"default:0;not null" json:"canPreview"`
	UploadedByID uint              `gorm:"index;not null" json:"-"`
	UploadedBy   User              `gorm:"foreignKey:UploadedByID;constraint:OnDelete:NO ACTION" json:"-"`
	CopiedFromID *uint             `gorm:"index" json:"-"`
	IsActive     bool              `gorm:"default:1;not null" json:"isActive"`
	Locked       bool              `gorm:"-" json:"locked"`
	InUse        bool              `gorm:"-" json:"inUse"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"-"`
	DjangoID     uint              `gorm:"index" json:"-"`
}

func (a *Attachment) AfterFind(*gorm.DB) error {
	a.Locked = a.CopiedFromID != nil
	return nil
}

// BeforeCreate assigns a ULID public identifier.
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	a.UID = uid
	return nil
}
