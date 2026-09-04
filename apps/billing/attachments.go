package billing

import (
	"dfms/apps/models"
	"dfms/apps/reports"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
)

func (h handler) attachFcf(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadFcf(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) attachVar(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadVar(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) attachKoj(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadKoj(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) attachTbs(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadTbs(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) attachFX(c fiber.Ctx) (attachment.Entity, error) {
	var row models.ExchangeRate
	if err := firstUID(h.db, c.Params("uid"), &row); err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{
		ID: row.ID, UID: row.UID,
		DocumentNumber: row.FromCurrency + "-" + row.ToCurrency + "-" + row.EffectiveFrom.Format("20060102"),
		CanMutate:      editable(row.Status),
	}, nil
}

func (h handler) attachMi(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) attachRun(c fiber.Ctx) (attachment.Entity, error) {
	var row models.BillingRun
	if err := firstUID(h.db, c.Params("uid"), &row); err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) getMi(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) loadMi(uid string) (models.MiLossBatch, error) {
	var row models.MiLossBatch
	err := models.PreloadCreatedBy(h.db).
		Preload("Products.Product").Preload("Products.Rates.Product").
		Where("UID = ?", uid).First(&row).Error
	return row, err
}

func (h handler) printMi(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	return reports.WriteMiLossPDF(c, row)
}

func (h handler) miWorkflow(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	return h.docWorkflow(c, types.MiLossBatchContent, row.ID)
}

func (h handler) runWorkflow(c fiber.Ctx) error {
	var row models.BillingRun
	if err := firstUID(h.db, c.Params("uid"), &row); err != nil {
		return notFound(c, err, "billing run not found")
	}
	return h.docWorkflow(c, types.BillingRunContent, row.ID)
}
