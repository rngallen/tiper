package masterdata

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/ewura"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listTanks(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Tank{}).Preload("Product"),
		search, "Name", "Code",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code",
	}, "Code", "Tanks", "tanks",
		[]any{"Code", "Name", "Capacity", "Dead stock", "Active"},
		func(r models.Tank) []any {
			return []any{r.Code, r.Name, r.MaximumCapacity.String(), r.DeadStock.String(), r.IsActive}
		},
	)
}

func (h handler) createTank(c fiber.Ctx) error {
	var in tankRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cap, err := parseDec(in.MaximumCapacity)
	if err != nil {
		return response.BadRequest(c, "maximum capacity must be a number")
	}
	dead, err := parseDec(in.DeadStock)
	if err != nil {
		return response.BadRequest(c, "dead stock must be a number")
	}
	pid := idByUID[models.Product](h.db, in.ProductID)
	if pid == 0 {
		return response.BadRequest(c, "product is required")
	}
	row := models.Tank{
		Code: in.Code, Name: in.Name, MaximumCapacity: cap, DeadStock: dead,
		ProductID:   pid,
		CreatedByID: middleware.GetUserIDFromContext(c),
		IsActive:    activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a tank with this code already exists")
	}
	ewura.EnqueueTank(h.db, &row, "create")
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.TankContent, "tank "+row.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateTank(c fiber.Ctx) error {
	var row models.Tank
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "tank not found")
	}
	var in tankUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	cap, err := parseDec(in.MaximumCapacity)
	if err != nil {
		return response.BadRequest(c, "maximum capacity must be a number")
	}
	dead, err := parseDec(in.DeadStock)
	if err != nil {
		return response.BadRequest(c, "dead stock must be a number")
	}
	row.MaximumCapacity = cap
	row.DeadStock = dead
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	pid := idByUID[models.Product](h.db, in.ProductID)
	if pid == 0 {
		return response.BadRequest(c, "product is required")
	}
	row.ProductID = pid
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update tank")
	}
	ewura.EnqueueTank(h.db, &row, "update")
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.TankContent, "tank "+row.Code+" updated", before, row)
	return okUpdate(c, row, before, row)
}
