package masterdata

import (
	"errors"
	"strings"

	"dfms/apps/models"
	internalsage "dfms/internal/sage"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"gorm.io/gorm"
)

const (
	ownerCustomer = "customer"
	ownerSupplier = "supplier"
)

var errSageAccountTaken = errors.New("sage account already mapped to another party")

func orderBillingAccounts(db *gorm.DB) *gorm.DB {
	return db.Order("FeeCode, CurrencyCode")
}

func (h handler) listCustomerBillingAccounts(c fiber.Ctx) error {
	var cust models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &cust); err != nil {
		return notFound(c, err, "customer not found")
	}
	var rows []models.CustomerBillingAccount
	if err := h.db.WithContext(c.Context()).Where("CustomerID = ?", cust.ID).
		Scopes(orderBillingAccounts).Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}

func (h handler) createCustomerBillingAccount(c fiber.Ctx) error {
	var cust models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &cust); err != nil {
		return notFound(c, err, "customer not found")
	}
	in, client, _, err := h.bindBillingAccount(c, ownerCustomer)
	if err != nil {
		return err
	}
	home := homeCurrency(h.db.WithContext(c.Context()))
	row := models.CustomerBillingAccount{
		CustomerID:   cust.ID,
		FeeCode:      types.FeeCode(in.FeeCode),
		CurrencyCode: client.CurrencyCode,
		SageAccount:  client.Account,
		SageName:     client.Name,
		BillingUnit:  defaultUnit(in.BillingUnit),
		IsForeign:    client.CurrencyCode != home,
		IsActive:     activeOrDefault(in.IsActive),
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := claimSageAccount(tx, client.Account, ownerCustomer, cust.ID); err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return writeBillingErr(c, err, "this customer already has a billing account for that fee and currency")
	}
	recordAudit(c, types.ModuleCustomer, types.ActionCreate, row.UID, types.CustomerBillingAccountContent,
		"billing account "+row.SageAccount+" mapped for "+string(row.FeeCode)+" "+row.CurrencyCode, nil, row)
	return response.Created(c, row)
}

func (h handler) updateCustomerBillingAccount(c fiber.Ctx) error {
	var cust models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &cust); err != nil {
		return notFound(c, err, "customer not found")
	}
	var row models.CustomerBillingAccount
	if err := h.db.WithContext(c.Context()).Where("UID = ? AND CustomerID = ?", c.Params("aid"), cust.ID).First(&row).Error; err != nil {
		return notFound(c, err, "billing account not found")
	}
	in, client, _, err := h.bindBillingAccount(c, ownerCustomer)
	if err != nil {
		return err
	}
	home := homeCurrency(h.db.WithContext(c.Context()))
	before := row
	oldAccount := row.SageAccount
	row.FeeCode = types.FeeCode(in.FeeCode)
	row.CurrencyCode = client.CurrencyCode
	row.SageAccount = client.Account
	row.SageName = client.Name
	row.BillingUnit = defaultUnit(in.BillingUnit)
	row.IsForeign = client.CurrencyCode != home
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := claimSageAccount(tx, client.Account, ownerCustomer, cust.ID); err != nil {
			return err
		}
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if oldAccount != client.Account {
			return releaseSageAccountIfUnused(tx, oldAccount, ownerCustomer, cust.ID)
		}
		return nil
	})
	if err != nil {
		return writeBillingErr(c, err, "this customer already has a billing account for that fee and currency")
	}
	recordAudit(c, types.ModuleCustomer, types.ActionUpdate, row.UID, types.CustomerBillingAccountContent,
		"billing account "+row.SageAccount+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) deleteCustomerBillingAccount(c fiber.Ctx) error {
	var cust models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &cust); err != nil {
		return notFound(c, err, "customer not found")
	}
	var row models.CustomerBillingAccount
	if err := h.db.WithContext(c.Context()).Where("UID = ? AND CustomerID = ?", c.Params("aid"), cust.ID).First(&row).Error; err != nil {
		return notFound(c, err, "billing account not found")
	}
	var n int64
	h.db.WithContext(c.Context()).Model(&models.Depot{}).Where("BillingAccountID = ?", row.ID).Count(&n)
	if n > 0 {
		return response.Conflict(c, "this billing account is used by a depot")
	}
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		return releaseSageAccountIfUnused(tx, row.SageAccount, ownerCustomer, cust.ID)
	})
	if err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	recordAudit(c, types.ModuleCustomer, types.ActionDelete, row.UID, types.CustomerBillingAccountContent,
		"billing account "+row.SageAccount+" removed", row, nil)
	return response.Deleted(c)
}

func (h handler) listSupplierBillingAccounts(c fiber.Ctx) error {
	var supp models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &supp); err != nil {
		return notFound(c, err, "supplier not found")
	}
	var rows []models.SupplierBillingAccount
	if err := h.db.WithContext(c.Context()).Where("SupplierID = ?", supp.ID).
		Scopes(orderBillingAccounts).Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}

func (h handler) createSupplierBillingAccount(c fiber.Ctx) error {
	var supp models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &supp); err != nil {
		return notFound(c, err, "supplier not found")
	}
	in, client, _, err := h.bindBillingAccount(c, ownerSupplier)
	if err != nil {
		return err
	}
	row := models.SupplierBillingAccount{
		SupplierID:   supp.ID,
		FeeCode:      types.FeeCode(in.FeeCode),
		CurrencyCode: client.CurrencyCode,
		SageAccount:  client.Account,
		SageName:     client.Name,
		BillingUnit:  defaultUnit(in.BillingUnit),
		IsActive:     activeOrDefault(in.IsActive),
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := claimSageAccount(tx, client.Account, ownerSupplier, supp.ID); err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return writeBillingErr(c, err, "this supplier already has a billing account for that fee and currency")
	}
	recordAudit(c, types.ModuleCustomer, types.ActionCreate, row.UID, types.SupplierBillingAccountContent,
		"billing account "+row.SageAccount+" mapped for "+string(row.FeeCode)+" "+row.CurrencyCode, nil, row)
	return response.Created(c, row)
}

func (h handler) updateSupplierBillingAccount(c fiber.Ctx) error {
	var supp models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &supp); err != nil {
		return notFound(c, err, "supplier not found")
	}
	var row models.SupplierBillingAccount
	if err := h.db.WithContext(c.Context()).Where("UID = ? AND SupplierID = ?", c.Params("aid"), supp.ID).First(&row).Error; err != nil {
		return notFound(c, err, "billing account not found")
	}
	in, client, _, err := h.bindBillingAccount(c, ownerSupplier)
	if err != nil {
		return err
	}
	before := row
	oldAccount := row.SageAccount
	row.FeeCode = types.FeeCode(in.FeeCode)
	row.CurrencyCode = client.CurrencyCode
	row.SageAccount = client.Account
	row.SageName = client.Name
	row.BillingUnit = defaultUnit(in.BillingUnit)
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := claimSageAccount(tx, client.Account, ownerSupplier, supp.ID); err != nil {
			return err
		}
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if oldAccount != client.Account {
			return releaseSageAccountIfUnused(tx, oldAccount, ownerSupplier, supp.ID)
		}
		return nil
	})
	if err != nil {
		return writeBillingErr(c, err, "this supplier already has a billing account for that fee and currency")
	}
	recordAudit(c, types.ModuleCustomer, types.ActionUpdate, row.UID, types.SupplierBillingAccountContent,
		"billing account "+row.SageAccount+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) deleteSupplierBillingAccount(c fiber.Ctx) error {
	var supp models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &supp); err != nil {
		return notFound(c, err, "supplier not found")
	}
	var row models.SupplierBillingAccount
	if err := h.db.WithContext(c.Context()).Where("UID = ? AND SupplierID = ?", c.Params("aid"), supp.ID).First(&row).Error; err != nil {
		return notFound(c, err, "billing account not found")
	}
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		return releaseSageAccountIfUnused(tx, row.SageAccount, ownerSupplier, supp.ID)
	})
	if err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	recordAudit(c, types.ModuleCustomer, types.ActionDelete, row.UID, types.SupplierBillingAccountContent,
		"billing account "+row.SageAccount+" removed", row, nil)
	return response.Deleted(c)
}

func (h handler) bindBillingAccount(c fiber.Ctx, party string) (billingAccountRequest, internalsage.Client, models.Fee, error) {
	var in billingAccountRequest
	var zero internalsage.Client
	var fee models.Fee
	if err := bindBody(c, &in); err != nil {
		return in, zero, fee, err
	}
	if err := h.db.WithContext(c.Context()).Where("Code = ?", in.FeeCode).First(&fee).Error; err != nil {
		return in, zero, fee, response.UnprocessableEntity(c, validation.Errors{
			"feeCode": validation.NewError("validation_fee", "unknown fee"),
		})
	}
	if !fee.IsActive || !fee.ChargeTo.Allows(party) {
		return in, zero, fee, response.UnprocessableEntity(c, validation.Errors{
			"feeCode": validation.NewError("validation_fee", "this fee cannot be mapped to a "+party),
		})
	}
	sageDB := db.Sage()
	if sageDB == nil {
		return in, zero, fee, response.ServiceUnavailable(c, "Sage 200 is not connected")
	}
	client, err := internalsage.GetClient(c.Context(), sageDB, in.SageAccount)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return in, zero, fee, response.UnprocessableEntity(c, validation.Errors{
				"sageAccount": validation.NewError("validation_sage", "Sage account not found in Client"),
			})
		}
		logs.Error(err)
		return in, zero, fee, response.InternalServerError(c)
	}
	if client.OnHold {
		return in, zero, fee, response.UnprocessableEntity(c, validation.Errors{
			"sageAccount": validation.NewError("validation_hold", "Sage account is on hold"),
		})
	}
	if client.CurrencyCode == "" {
		return in, zero, fee, response.UnprocessableEntity(c, validation.Errors{
			"sageAccount": validation.NewError("validation_currency", "Sage currency id is not mapped"),
		})
	}
	return in, client, fee, nil
}

func claimSageAccount(tx *gorm.DB, account, kind string, ownerID uint) error {
	account = strings.TrimSpace(account)
	var existing models.SageAccountOwner
	err := tx.Where("SageAccount = ?", account).First(&existing).Error
	if err == nil {
		if existing.OwnerKind == kind && existing.OwnerID == ownerID {
			return nil
		}
		return errSageAccountTaken
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	err = tx.Create(&models.SageAccountOwner{
		SageAccount: account, OwnerKind: kind, OwnerID: ownerID,
	}).Error
	if err == nil {
		return nil
	}
	if response.IsDuplicate(err) {
		return errSageAccountTaken
	}
	return err
}

func releaseSageAccountIfUnused(tx *gorm.DB, account, kind string, ownerID uint) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil
	}
	var n int64
	switch kind {
	case ownerCustomer:
		if err := tx.Model(&models.CustomerBillingAccount{}).
			Where("SageAccount = ? AND CustomerID = ?", account, ownerID).Count(&n).Error; err != nil {
			return err
		}
	case ownerSupplier:
		if err := tx.Model(&models.SupplierBillingAccount{}).
			Where("SageAccount = ? AND SupplierID = ?", account, ownerID).Count(&n).Error; err != nil {
			return err
		}
	}
	if n > 0 {
		return nil
	}
	return tx.Where("SageAccount = ? AND OwnerKind = ? AND OwnerID = ?", account, kind, ownerID).
		Delete(&models.SageAccountOwner{}).Error
}

func writeBillingErr(c fiber.Ctx, err error, duplicate string) error {
	if errors.Is(err, errSageAccountTaken) {
		return response.Conflict(c, "this Sage account is already mapped to another customer or supplier")
	}
	return writeErr(c, err, duplicate)
}

func defaultUnit(s string) string {
	u := types.NormalizeBillingUnit(s)
	if u == "" {
		return "M3"
	}
	return u
}

func homeCurrency(db *gorm.DB) string {
	var row models.Company
	if err := db.Select("CurrencyCode").First(&row, 1).Error; err != nil {
		return "TZS"
	}
	if row.CurrencyCode != nil {
		if ccy := strings.ToUpper(strings.TrimSpace(*row.CurrencyCode)); len(ccy) == 3 {
			return ccy
		}
	}
	return "TZS"
}
