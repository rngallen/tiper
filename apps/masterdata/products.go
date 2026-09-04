package masterdata

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listProducts(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Product{}).Preload("StockCategory"),
		search, "Name", "Code", "Unit",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code", "unit": "Unit",
	}, "Name", "Products", "products",
		[]any{"Code", "Name", "Unit", "Active"},
		func(r models.Product) []any { return []any{r.Code, r.Name, r.Unit, r.IsActive} },
	)
}

func (h handler) createProduct(c fiber.Ctx) error {
	var in productRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.Product{
		Code: in.Code, Name: in.Name, Unit: productUnitLitre,
		CreatedByID: middleware.GetUserIDFromContext(c),
		IsActive:    activeOrDefault(in.IsActive),
	}
	cid := idByUID[models.StockCategory](h.db, in.StockCategoryID)
	if cid == 0 {
		return response.BadRequest(c, "category is required")
	}
	row.StockCategoryID = cid
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a product with this code already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.ProductContent, "product "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateProduct(c fiber.Ctx) error {
	var row models.Product
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "product not found")
	}
	var in productUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.Name = in.Name
	row.Unit = productUnitLitre
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	cid := idByUID[models.StockCategory](h.db, in.StockCategoryID)
	if cid == 0 {
		return response.BadRequest(c, "category is required")
	}
	row.StockCategoryID = cid
	models.MarkHasData(h.db, &models.StockCategory{}, cid)
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update product")
	}
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.ProductContent, "product "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
