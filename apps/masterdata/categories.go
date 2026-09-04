package masterdata

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

func (h handler) listCategories(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(h.db.WithContext(c.Context()).Model(&models.StockCategory{}), search, "Name"), search)
	return serveCatalogue(c, q, search, map[string]string{"name": "Name"}, "Name",
		"Categories", "categories",
		[]any{"Name", "Active"},
		func(r models.StockCategory) []any { return []any{r.Name, r.IsActive} },
	)
}

func (h handler) createCategory(c fiber.Ctx) error {
	var in categoryRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.StockCategory{
		Name: in.Name, CreatedByID: middleware.GetUserIDFromContext(c), IsActive: true,
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a category with this name already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.StockCategoryContent, "category "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateCategory(c fiber.Ctx) error {
	var row models.StockCategory
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "category not found")
	}
	before := row
	var in categoryUpdateRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "a category with this name already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.StockCategoryContent, "category "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}
