package inventory

import (
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
)

func (h handler) attachReceipt(c fiber.Ctx) (attachment.Entity, error) {
	var row models.Receipt
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachITT(c fiber.Ctx) (attachment.Entity, error) {
	var row models.IttTransfer
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachZerol(c fiber.Ctx) (attachment.Entity, error) {
	var row models.ZerolizationTransfer
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachHold(c fiber.Ctx) (attachment.Entity, error) {
	var row models.FinancialHoldRelease
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) getZerol(c fiber.Ctx) error {
	var row models.ZerolizationTransfer
	if err := models.PreloadCreatedBy(h.db).Preload("Customer").Preload("Product").Preload("FromVessel").Preload("ToVessel").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}
