package auth

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/audit"
	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"errors"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// TitleHandler manages selectable user job titles.
type TitleHandler struct{ db *gorm.DB }

// NewTitleHandler constructs a TitleHandler.
func NewTitleHandler(db *gorm.DB) *TitleHandler { return &TitleHandler{db: db} }

// getTitles returns all titles.
func (h *TitleHandler) list(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	search, err := response.ParseSearchRequest(c)
	if err != nil {
		logs.Error(err)
		return response.BadRequestBind(c, err)
	}

	// Secure column whitelisting — prevents SQL injection via ORDER BY
	allowedSort := map[string]string{
		"name": "Name",
	}
	sortColumn, ok := allowedSort[search.OrderBy]
	if !ok {
		sortColumn = "Name"
	}

	query := db.WithContext(c.Context()).Model(&models.Title{})
	if search.HasSearch() {
		query = query.Where("Name LIKE ?", search.Search)
	}
	query = query.Order(stableOrder(sortColumn, search.SortDirection))

	if search.Export {
		return export.Query(c, query, "Titles", "titles",
			[]any{"ID", "Name", "In Use"},
			func(title models.Title) []any {
				return []any{title.ID, title.Name, title.HasData}
			},
		)
	}

	var items []models.Title
	titles, err := response.NewPaginator(c, query, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, titles)
}

// Create adds a new title.
func (h *TitleHandler) create(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	var payload titleCreateSchema
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	title := models.Title{
		Name:        payload.Name,
		CreatedByID: middleware.GetUserIDFromContext(c),
	}

	if err := db.WithContext(c.Context()).Create(&title).Error; err != nil {
		if response.IsDuplicate(err) {
			return response.Conflict(c, "a title with this name already exists")
		}
		logs.Errorf("create title: %v", err)
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleTitle, types.ActionCreate, title.ID,
		types.TitleContent, "title "+title.Name+" created", nil, title)
	return response.Created(c, title)
}

func (h *TitleHandler) update(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	titleId := c.Params("id")
	var payload titleCreateSchema

	// Parse payload
	if err := c.Bind().Body(&payload); err != nil {
		logs.Error(err)
		return response.BadRequestBind(c, err)
	}
	// Validate payload
	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Get title from db
	var title models.Title
	if err := db.Where("ID = ?", titleId).First(&title).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "title not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	before := title

	// Check for existence
	err := db.Where("Name = ? AND ID <> ?", payload.Name, titleId).First(&models.Title{}).Error
	if err == nil {
		return response.Conflict(c, "title name exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	if err := db.Model(&title).Update("Name", payload.Name).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	title.Name = payload.Name

	audit.RecordHTTP(c, types.ModuleTitle, types.ActionUpdate, title.ID,
		types.TitleContent, "title "+payload.Name+" updated", before, title)

	return response.OkMessage(c, audit.UpdateMessage(before, title))
}

// Delete removes a title that is not referenced.
func (h *TitleHandler) delete(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	var title models.Title
	if err := db.First(&title, "ID = ?", c.Params("id")).Error; err != nil {
		return response.NotFound(c, "title not found")
	}
	if title.HasData {
		return response.Conflict(c, "title is in use and cannot be deleted")
	}
	if err := db.Delete(&title).Error; err != nil {
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleTitle, types.ActionDelete, title.ID,
		types.TitleContent, "title "+title.Name+" deleted", title, nil)

	return response.Deleted(c)
}
