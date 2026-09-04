package billing

import (
	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// ApplyChangeOfService sets document status. On approval, only the selected
// vessel parcel's delivery method is rewritten so later FCF cycles bill that class of trade.
func ApplyChangeOfService(tx *gorm.DB, id uint, status types.DocumentStatus) error {
	if status != types.DocApproved {
		return tx.Model(&models.ChangeOfService{}).Where("ID = ?", id).Update("Status", status).Error
	}
	var row models.ChangeOfService
	if err := tx.First(&row, id).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Update("Status", types.DocApproved).Error; err != nil {
		return err
	}
	if row.ReceiptDetailID == 0 || row.ToCollection == "" {
		return nil
	}
	return tx.Model(&models.ReceiptDetail{}).
		Where("ID = ? AND CustomerID = ? AND IsArchived = 0", row.ReceiptDetailID, row.CustomerID).
		Update("CollectionMethod", row.ToCollection).Error
}
