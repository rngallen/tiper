package masterdata

import (
	"errors"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"github.com/jellydator/validation/is"
	"gorm.io/gorm"
)

func (h handler) listDepots(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Depot{}).Preload("Customer"),
		search, "Name", "Code", "EwuraLicense",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code",
	}, "Name", "Depots", "depots",
		[]any{"Code", "Name", "EWURA license", "Internal", "Active"},
		func(r models.Depot) []any {
			return []any{r.Code, r.Name, r.EwuraLicense, r.IsInternal, r.IsActive}
		},
	)
}

func (h handler) createDepot(c fiber.Ctx) error {
	var in depotRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	if in.EwuraLicense != "" {
		if _, err := h.ewuraLicense(c, in.EwuraLicense); err != nil {
			return err
		}
	}
	row := models.Depot{
		Code: in.Code, Name: in.Name, EwuraLicense: in.EwuraLicense,
		IsInternal: in.IsInternal, CreatedByID: middleware.GetUserIDFromContext(c),
		IsActive: activeOrDefault(in.IsActive),
	}
	if cid, err := h.kojCustomerID(c, in.CustomerID); err != nil {
		return err
	} else {
		row.CustomerID = &cid
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a depot with this code already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.DepotContent, "depot "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateDepot(c fiber.Ctx) error {
	var row models.Depot
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "depot not found")
	}
	var in depotUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	if in.EwuraLicense != "" {
		if _, err := h.ewuraLicense(c, in.EwuraLicense); err != nil {
			return err
		}
	}
	before := row
	row.Name = in.Name
	row.EwuraLicense = in.EwuraLicense
	row.IsInternal = in.IsInternal
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if cid, err := h.kojCustomerID(c, in.CustomerID); err != nil {
		return err
	} else {
		row.CustomerID = &cid
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update depot")
	}
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.DepotContent, "depot "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) kojCustomerID(c fiber.Ctx, uid string) (uint, error) {
	db := h.db.WithContext(c.Context())
	if uid == "" {
		if id := firstKojCustomerID(db); id > 0 {
			return id, nil
		}
		return 0, response.UnprocessableEntity(c, validation.Errors{
			"customerId": validation.NewError("validation_required", "select a customer that has a KOJ fee billing account"),
		})
	}
	var cust models.Customer
	if err := firstUID(db, uid, &cust); err != nil {
		return 0, response.UnprocessableEntity(c, validation.Errors{
			"customerId": validation.NewError("validation_not_found", "customer not found"),
		})
	}
	var n int64
	db.Model(&models.CustomerBillingAccount{}).
		Where("CustomerID = ? AND FeeCode = ? AND IsActive = ?", cust.ID, types.FeeKOJ, true).
		Count(&n)
	if n == 0 {
		return 0, response.UnprocessableEntity(c, validation.Errors{
			"customerId": validation.NewError("validation_koj", "customer must have a KOJ fee billing account"),
		})
	}
	return cust.ID, nil
}

func firstKojCustomerID(db *gorm.DB) uint {
	var id uint
	db.Model(&models.CustomerBillingAccount{}).
		Select("CustomerID").
		Where("FeeCode = ? AND IsActive = ?", types.FeeKOJ, true).
		Order("ID ASC").Limit(1).Scan(&id)
	return id
}

func (h handler) ewuraLicense(c fiber.Ctx, number string) (*models.EwuraPetroleumLicense, error) {
	if number == "" {
		return nil, response.UnprocessableEntity(c, validation.Errors{
			"ewuraLicense": validation.NewError("validation_required", "EWURA license is required"),
		})
	}
	var lic models.EwuraPetroleumLicense
	err := h.db.WithContext(c.Context()).Where("LicenseNumber = ?", number).First(&lic).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.UnprocessableEntity(c, validation.Errors{
			"ewuraLicense": validation.NewError("validation_not_found", "EWURA license not found"),
		})
	}
	if err != nil {
		return nil, notFound(c, err, "EWURA license not found")
	}
	return &lic, nil
}

func fillFromLicense(lic *models.EwuraPetroleumLicense, email, tin, phone *string) {
	if lic == nil {
		return
	}
	if email != nil && *email == "" {
		e := lower(lic.Email)
		if e != "" && validation.Validate(e, is.EmailFormat) == nil {
			*email = e
		}
	}
	if tin != nil && *tin == "" {
		*tin = compact(lic.TinNumber)
	}
	if phone != nil && *phone == "" {
		*phone = compact(lic.Phone)
	}
}
