package orders

import (
	"dfms/internal/attach"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

func copyCustomerAttachments(tx *gorm.DB, customerID, requestID, userID uint) error {
	return attach.CopyCustomerFiles(tx, customerID, requestID, types.GantryLoadingRequestContent, userID)
}
