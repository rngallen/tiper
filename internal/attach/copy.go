package attach

import (
	"fmt"

	"dfms/apps/models"
	"dfms/pkg/types"
	"dfms/utils"

	"gorm.io/gorm"
)

// CopyCustomerFiles clones active customer files onto a new ILR, pump-over, or ITT.
// Inactive customer files stay on the OMC record for history and are not copied.
func CopyCustomerFiles(tx *gorm.DB, customerID, entityID uint, entityType types.ContentType, userID uint) error {
	if customerID == 0 || entityID == 0 {
		return nil
	}
	var src []models.Attachment
	if err := tx.Where("EntityType = ? AND EntityID = ? AND IsActive = ?", types.CustomerContent, customerID, true).
		Find(&src).Error; err != nil {
		return err
	}
	for i := range src {
		uid, err := utils.GetULID()
		if err != nil {
			return err
		}
		clone := models.Attachment{
			OriginalName: src[i].OriginalName,
			StoredName:   fmt.Sprintf("%s-%s", src[i].StoredName, uid[:8]),
			FilePath:     src[i].FilePath,
			EntityID:     entityID,
			EntityType:   entityType,
			Size:         src[i].Size,
			ByteSize:     src[i].ByteSize,
			Extension:    src[i].Extension,
			Mime:         src[i].Mime,
			CanPreview:   src[i].CanPreview,
			UploadedByID: userID,
			CopiedFromID: &src[i].ID,
			IsActive:     true,
		}
		if err := tx.Create(&clone).Error; err != nil {
			return err
		}
	}
	return nil
}
