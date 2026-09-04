package masterdata

import (
	"time"

	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listDrivers(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Driver{}),
		search, "Name", "LicenseNumber", "Phone", "Email",
	), search)
	return serveCatalogue(c, q, search, map[string]string{"name": "Name", "licenseNumber": "LicenseNumber"}, "Name",
		"Drivers", "drivers",
		[]any{"Name", "License", "Phone", "Active"},
		func(r models.Driver) []any { return []any{r.Name, r.LicenseNumber, r.Phone, r.IsActive} },
	)
}

func (h handler) createDriver(c fiber.Ctx) error {
	var in driverRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	exp := parseOptionalDate(in.LicenseExpires)
	if exp == nil {
		t := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
		exp = &t
	}
	row := models.Driver{
		Name: in.Name, LicenseNumber: in.LicenseNumber, LicenseExpires: exp,
		Phone: in.Phone, Email: in.Email,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a driver with this licence already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.DriverContent, "driver "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateDriver(c fiber.Ctx) error {
	var row models.Driver
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "driver not found")
	}
	var in driverUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.Phone = in.Phone
	row.Email = in.Email
	if exp := parseOptionalDate(in.LicenseExpires); exp != nil {
		row.LicenseExpires = exp
	}
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "a driver with this licence already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.DriverContent, "driver "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
