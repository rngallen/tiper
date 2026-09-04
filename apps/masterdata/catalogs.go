package masterdata

import (
	"strconv"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func catalogAudit(c fiber.Ctx, action types.Action, code string, desc string, before, after any) {
	recordAudit(c, types.ModuleSettings, action, code, types.LookupContent, desc, before, after)
}

func (h handler) listTenders(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.ImportTenderType{}, "tenders",
		[]any{"Code", "Name", "SRT", "Supplier unless loading", "Active"},
		func(r models.ImportTenderType) []any {
			return []any{r.Code, r.Name, r.IsSingleReceiving, r.SupplierPaysUnlessLoading, r.IsActive}
		})
}

func (h handler) createTender(c fiber.Ctx) error {
	var in tenderRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.ImportTenderType{
		Code:                      types.TenderCode(in.Code),
		Name:                      in.Name,
		IsSingleReceiving:         in.IsSingleReceiving,
		SupplierPaysUnlessLoading: in.SupplierPaysUnlessLoading,
		IsActive:                  activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a tender with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, string(row.Code), "tender "+string(row.Code)+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateTender(c fiber.Ctx) error {
	var row models.ImportTenderType
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "tender not found")
	}
	before := row
	var in tenderRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	row.IsSingleReceiving = in.IsSingleReceiving
	row.SupplierPaysUnlessLoading = in.SupplierPaysUnlessLoading
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update tender")
	}
	catalogAudit(c, types.ActionUpdate, string(row.Code), "tender "+string(row.Code)+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listRoutes(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.DischargeRoute{}, "discharge-routes",
		[]any{"Code", "Name", "Active"},
		func(r models.DischargeRoute) []any {
			return []any{r.Code, r.Name, r.IsActive}
		})
}

func (h handler) createRoute(c fiber.Ctx) error {
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.DischargeRoute{
		Code: types.DischargeRoute(in.Code), Name: in.Name,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a route with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, in.Code, "catalog "+in.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateRoute(c fiber.Ctx) error {
	var row models.DischargeRoute
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "route not found")
	}
	before := row
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update route")
	}
	catalogAudit(c, types.ActionUpdate, c.Params("code"), "catalog updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listDelivery(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.DeliveryMethod{}, "delivery-methods",
		[]any{"Code", "Name", "Gantry loading", "Active"},
		func(r models.DeliveryMethod) []any {
			return []any{r.Code, r.Name, r.IsGantryLoading, r.IsActive}
		})
}

func (h handler) createDelivery(c fiber.Ctx) error {
	var in deliveryRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.DeliveryMethod{
		Code: types.CollectionMethod(in.Code), Name: in.Name, IsGantryLoading: in.IsGantryLoading,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a delivery method with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, in.Code, "catalog "+in.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateDelivery(c fiber.Ctx) error {
	var row models.DeliveryMethod
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "delivery method not found")
	}
	before := row
	var in deliveryRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name, row.IsGantryLoading = in.Name, in.IsGantryLoading
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update delivery method")
	}
	catalogAudit(c, types.ActionUpdate, c.Params("code"), "catalog updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listProcurement(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.ProcurementMethod{}, "procurement-methods",
		[]any{"Code", "Name", "Active"},
		func(r models.ProcurementMethod) []any {
			return []any{r.Code, r.Name, r.IsActive}
		})
}

func (h handler) createProcurement(c fiber.Ctx) error {
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.ProcurementMethod{
		Code: types.ProcurementCode(in.Code), Name: in.Name,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a procurement method with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, in.Code, "catalog "+in.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateProcurement(c fiber.Ctx) error {
	var row models.ProcurementMethod
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "procurement method not found")
	}
	before := row
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update procurement method")
	}
	catalogAudit(c, types.ActionUpdate, c.Params("code"), "catalog updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listContracts(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.ContractType{}, "contract-types",
		[]any{"Code", "Name", "Active"},
		func(r models.ContractType) []any {
			return []any{r.Code, r.Name, r.IsActive}
		})
}

func (h handler) createContract(c fiber.Ctx) error {
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.ContractType{
		Code: types.ContractCode(in.Code), Name: in.Name,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a contract type with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, in.Code, "catalog "+in.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateContract(c fiber.Ctx) error {
	var row models.ContractType
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "contract type not found")
	}
	before := row
	var in catalogNamedRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update contract type")
	}
	catalogAudit(c, types.ActionUpdate, c.Params("code"), "catalog updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listPricing(c fiber.Ctx) error {
	return listNamedCatalog(c, h.db, &[]models.PricingNature{}, "pricing-natures",
		[]any{"Code", "Name", "Promotional", "Active"},
		func(r models.PricingNature) []any {
			return []any{r.Code, r.Name, r.IsPromotional, r.IsActive}
		})
}

func (h handler) createPricing(c fiber.Ctx) error {
	var in pricingRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.PricingNature{
		Code: types.PricingNature(in.Code), Name: in.Name, IsPromotional: in.IsPromotional,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a pricing nature with this code already exists")
	}
	catalogAudit(c, types.ActionCreate, in.Code, "catalog "+in.Code+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updatePricing(c fiber.Ctx) error {
	var row models.PricingNature
	if err := firstCatalog(h.db.WithContext(c.Context()), c.Params("code"), &row); err != nil {
		return notFound(c, err, "pricing nature not found")
	}
	before := row
	var in pricingRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name, row.IsPromotional = in.Name, in.IsPromotional
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update pricing nature")
	}
	catalogAudit(c, types.ActionUpdate, c.Params("code"), "catalog updated", before, row)
	return okUpdate(c, row, before, row)
}

func (h handler) listCycles(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(h.db.WithContext(c.Context()).Model(&models.BillingCycle{}), search, "Description"), search)
	return serveCatalogue(c, q, search, map[string]string{"days": "Days"},
		"Days", "Billing cycles", "billing-cycles",
		[]any{"Days", "Description", "Active"},
		func(r models.BillingCycle) []any {
			return []any{r.Days, r.Description, r.IsActive}
		}, "Days")
}

func (h handler) createCycle(c fiber.Ctx) error {
	var in cycleRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.BillingCycle{
		Days: in.Days, Description: in.Description,
		IsActive: activeOrDefault(in.IsActive),
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a billing cycle with these days already exists")
	}
	catalogAudit(c, types.ActionCreate, strconv.Itoa(row.Days), "billing cycle "+strconv.Itoa(row.Days)+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateCycle(c fiber.Ctx) error {
	days, err := strconv.Atoi(c.Params("days"))
	if err != nil {
		return response.BadRequest(c, "invalid days")
	}
	var row models.BillingCycle
	if err := h.db.WithContext(c.Context()).First(&row, days).Error; err != nil {
		return notFound(c, err, "billing cycle not found")
	}
	before := row
	var in cycleRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Description = in.Description
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update billing cycle")
	}
	catalogAudit(c, types.ActionUpdate, strconv.Itoa(row.Days), "billing cycle updated", before, row)
	return okUpdate(c, row, before, row)
}

func listNamedCatalog[T any](c fiber.Ctx, db *gorm.DB, dest *[]T, prefix string, headers []any, mapRow func(T) []any) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(db.WithContext(c.Context()).Model(new(T)), search, "Code", "Name"), search)
	return serveCatalogue(c, q, search, map[string]string{"code": "Code", "name": "Name"},
		"Code", prefix, prefix, headers, mapRow, "Code")
}

func firstCatalog(db *gorm.DB, code string, dest any) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return gorm.ErrRecordNotFound
	}
	return db.Where("Code = ?", code).First(dest).Error
}
