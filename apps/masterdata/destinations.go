package masterdata

import (
	"strings"

	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
)

func (h handler) listDestinations(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(h.db.WithContext(c.Context()).Model(&models.Destination{}), search, "Name"), search)
	if v := strings.TrimSpace(c.Query("isCountry")); v != "" {
		q = q.Where("IsCountry = ?", v == "true" || v == "1")
	}
	return serveCatalogue(c, q, search, map[string]string{"name": "Name"}, "Name",
		"Destinations", "destinations",
		[]any{"Name", "Country", "Active"},
		func(r models.Destination) []any { return []any{r.Name, r.IsCountry, r.IsActive} },
	)
}

func (h handler) createDestination(c fiber.Ctx) error {
	var in destinationRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.Destination{Name: in.Name, IsCountry: in.IsCountry, IsActive: activeOrDefault(in.IsActive)}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a destination with this name already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.DestinationContent, "destination "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateDestination(c fiber.Ctx) error {
	var row models.Destination
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "destination not found")
	}
	var in destinationRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.IsCountry = in.IsCountry
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "a destination with this name already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.DestinationContent, "destination "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listDistricts(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.District{}).Preload("Destination"),
		search, "Name",
	), search)
	q, err = filterByUID[models.Destination](c, h.db, q, "destinationId", "DestinationID")
	if err != nil {
		return err
	}
	return serveCatalogue(c, q, search, map[string]string{"name": "Name"}, "Name",
		"Districts", "districts",
		[]any{"Name", "Destination", "Active"},
		func(r models.District) []any {
			dest := ""
			if r.Destination.Name != "" {
				dest = r.Destination.Name
			}
			return []any{r.Name, dest, r.IsActive}
		},
	)
}

func (h handler) createDistrict(c fiber.Ctx) error {
	var in districtRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	var dest models.Destination
	if err := firstUID(h.db.WithContext(c.Context()), in.DestinationID, &dest); err != nil {
		return response.UnprocessableEntity(c, validation.Errors{
			"destinationId": validation.NewError("validation_required", "destination is required"),
		})
	}
	row := models.District{Name: in.Name, DestinationID: dest.ID, IsActive: activeOrDefault(in.IsActive)}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a district with this name already exists for that destination")
	}
	row.Destination = dest
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.DistrictContent, "district "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateDistrict(c fiber.Ctx) error {
	var row models.District
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "district not found")
	}
	var in districtUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	if in.DestinationID != "" {
		var dest models.Destination
		if err := firstUID(h.db.WithContext(c.Context()), in.DestinationID, &dest); err != nil {
			return response.UnprocessableEntity(c, validation.Errors{
				"destinationId": validation.NewError("validation_required", "destination is required"),
			})
		}
		row.DestinationID = dest.ID
	}
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "a district with this name already exists for that destination")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.DistrictContent, "district "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
