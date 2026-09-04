package masterdata

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listVessels(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(h.db.WithContext(c.Context()).Model(&models.Vessel{}), search, "Name", "Code", "ImoNumber"), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code", "imoNumber": "ImoNumber",
	}, "Name", "Vessels", "vessels",
		[]any{"Code", "Name", "IMO", "Active"},
		func(r models.Vessel) []any { return []any{r.Code, r.Name, r.ImoNumber, r.IsActive} },
	)
}

func (h handler) createVessel(c fiber.Ctx) error {
	var in vesselRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.Vessel{
		Code: in.Code, Name: in.Name,
		ImoNumber:   in.ImoNumber,
		CreatedByID: middleware.GetUserIDFromContext(c),
		IsActive:    activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a vessel with this code already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.VesselContent, "vessel "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateVessel(c fiber.Ctx) error {
	var row models.Vessel
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "vessel not found")
	}
	var in vesselRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.ImoNumber = in.ImoNumber
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update vessel")
	}
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.VesselContent, "vessel "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
