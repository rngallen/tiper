package masterdata

import (
	"fmt"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listSuppliers(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Supplier{}).Preload("Country"),
		search, "Name", "Code", "Email", "Phone", "Mobile", "TinNumber", "ContactPerson",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code", "email": "Email",
	}, "Name", "Suppliers", "suppliers",
		[]any{"Code", "Name", "Email", "Phone", "TIN", "Active"},
		func(r models.Supplier) []any {
			return []any{r.Code, r.Name, r.Email, r.Phone, r.TinNumber, r.IsActive}
		},
	)
}

func (h handler) getSupplier(c fiber.Ctx) error {
	var row models.Supplier
	if err := h.db.WithContext(c.Context()).
		Preload("BillingAccounts", orderBillingAccounts).
		Preload("Country").
		Where("UID = ?", c.Params("id")).First(&row).Error; err != nil {
		return notFound(c, err, "supplier not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createSupplier(c fiber.Ctx) error {
	var in supplierRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.Supplier{
		Code: in.Code, Name: in.Name,
		Email: in.Email, Phone: in.Phone, Mobile: in.Mobile,
		ContactPerson: in.ContactPerson, TinNumber: in.TinNumber,
		CountryCode: countryCodePtr(in.CountryCode),
		Address:     in.Address,
		Address2:    in.Address2,
		CreatedByID: middleware.GetUserIDFromContext(c),
		IsActive:    activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a supplier with this code already exists")
	}
	if row.Code == "" {
		code := fmt.Sprintf("%d", row.ID+20000)
		if err := h.db.WithContext(c.Context()).Model(&row).Update("Code", code).Error; err != nil {
			logs.Error(err)
			return response.InternalServerError(c)
		}
		row.Code = code
	}
	recordAudit(c, types.ModuleCustomer, types.ActionCreate, row.UID, types.SupplierContent, "supplier "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateSupplier(c fiber.Ctx) error {
	var row models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "supplier not found")
	}
	var in supplierRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.Email = in.Email
	row.Phone = in.Phone
	row.Mobile = in.Mobile
	row.ContactPerson = in.ContactPerson
	row.TinNumber = in.TinNumber
	row.CountryCode = countryCodePtr(in.CountryCode)
	row.Address = in.Address
	row.Address2 = in.Address2
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update supplier")
	}
	recordAudit(c, types.ModuleCustomer, types.ActionUpdate, row.UID, types.SupplierContent, "supplier "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
