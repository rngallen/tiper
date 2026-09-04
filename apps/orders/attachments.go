package orders

import (
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
)

func (h handler) attachGLR(c fiber.Ctx) (attachment.Entity, error) {
	var row models.GantryLoadingRequest
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachPDO(c fiber.Ctx) (attachment.Entity, error) {
	var row models.PumpOverRequest
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachReport(c fiber.Ctx) (attachment.Entity, error) {
	var row models.PumpOverReport
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachComp(c fiber.Ctx) (attachment.Entity, error) {
	var row models.Compartmentalization
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) attachAmend(c fiber.Ctx) (attachment.Entity, error) {
	var row models.OrderAmendment
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: attachment.CanMutateStatus(string(row.Status))}, nil
}

func (h handler) getAmendment(c fiber.Ctx) error {
	var row models.OrderAmendment
	if err := models.PreloadCreatedBy(h.db).Preload("Ilo").Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}
