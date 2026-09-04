package masterdata

import (
	"strings"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func customerDuplicate(err error) string {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "kyc") {
		return "a customer with this KYC number already exists"
	}
	return "a customer with this code already exists"
}

func (h handler) listCustomers(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Customer{}),
		search, "Name", "Code", "Email", "Phone", "TinNumber", "KycNumber", "VrnNumber", "EwuraLicense",
	), search)
	if fee := strings.ToUpper(strings.TrimSpace(c.Query("feeCode"))); fee != "" {
		q = q.Where("ID IN (?)", h.db.WithContext(c.Context()).Model(&models.CustomerBillingAccount{}).
			Select("CustomerID").Where("FeeCode = ? AND IsActive = ?", fee, true))
	}
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code", "email": "Email", "ewuraLicense": "EwuraLicense",
		"kycNumber": "KycNumber", "vrnNumber": "VrnNumber",
	}, "Name", "Customers", "customers",
		[]any{"Code", "Name", "KYC", "VRN", "Email", "Phone", "TIN", "EWURA license", "Active"},
		func(r models.Customer) []any {
			return []any{r.Code, r.Name, r.KycNumber, r.VrnNumber, r.Email, r.Phone, r.TinNumber, r.EwuraLicense, r.IsActive}
		},
	)
}

func (h handler) getCustomer(c fiber.Ctx) error {
	var row models.Customer
	if err := h.db.WithContext(c.Context()).
		Preload("BillingAccounts", orderBillingAccounts).
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return notFound(c, err, "customer not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createCustomer(c fiber.Ctx) error {
	var in customerRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	lic, err := h.ewuraLicense(c, in.EwuraLicense)
	if err != nil {
		return err
	}
	fillFromLicense(lic, &in.Email, &in.TinNumber, &in.Phone)
	row := models.Customer{
		Code: in.Code, Name: in.Name,
		Email: in.Email, Phone: in.Phone, TinNumber: in.TinNumber,
		KycNumber: in.KycNumber, VrnNumber: in.VrnNumber,
		EwuraLicense: in.EwuraLicense,
		CreatedByID:  middleware.GetUserIDFromContext(c),
		IsActive:     activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, customerDuplicate(err))
	}
	recordAudit(c, types.ModuleCustomer, types.ActionCreate, row.UID, types.CustomerContent, "customer "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateCustomer(c fiber.Ctx) error {
	var row models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "customer not found")
	}
	var in customerRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	lic, err := h.ewuraLicense(c, in.EwuraLicense)
	if err != nil {
		return err
	}
	fillFromLicense(lic, &in.Email, &in.TinNumber, &in.Phone)
	before := row
	row.Name = in.Name
	row.Email = in.Email
	row.Phone = in.Phone
	row.TinNumber = in.TinNumber
	row.KycNumber = in.KycNumber
	row.VrnNumber = in.VrnNumber
	row.EwuraLicense = in.EwuraLicense
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, customerDuplicate(err))
	}
	recordAudit(c, types.ModuleCustomer, types.ActionUpdate, row.UID, types.CustomerContent, "customer "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
