package masterdata

import (
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listTransporters(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Transporter{}).Preload("Country"),
		search, "Name", "License", "Phone", "Email", "TinNumber", "ContactPerson",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "license": "License", "tinNumber": "TinNumber",
	}, "Name",
		"Haulers", "haulers",
		[]any{"Name", "TIN", "Phone", "License", "Active"},
		func(r models.Transporter) []any { return []any{r.Name, r.TinNumber, r.Phone, r.License, r.IsActive} },
	)
}

func (h handler) createTransporter(c fiber.Ctx) error {
	var in transporterRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.Transporter{
		Name:          in.Name,
		Phone:         in.Phone,
		Email:         in.Email,
		ContactPerson: in.ContactPerson,
		TinNumber:     in.TinNumber,
		VrnNumber:     in.VrnNumber,
		License:       in.License,
		Address:       in.Address,
		Address2:      in.Address2,
		AeoEndDate:    parseOptionalDate(in.AeoEndDate),
		CountryCode:   countryCodePtr(in.CountryCode),
		IsActive:      activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a transporter with this name already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.TransporterContent, "transporter "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateTransporter(c fiber.Ctx) error {
	var row models.Transporter
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "transporter not found")
	}
	var in transporterRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.License = in.License
	row.Phone = in.Phone
	row.Email = in.Email
	row.ContactPerson = in.ContactPerson
	row.TinNumber = in.TinNumber
	row.VrnNumber = in.VrnNumber
	row.Address = in.Address
	row.Address2 = in.Address2
	row.AeoEndDate = parseOptionalDate(in.AeoEndDate)
	row.CountryCode = countryCodePtr(in.CountryCode)
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "a transporter with this name already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.TransporterContent, "transporter "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
