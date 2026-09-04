package auth

import (
	"dfms/apps/models"
	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// permissionHandler exposes the catalogue of permission codes.
type permissionHandler struct{ db *gorm.DB }

// NewPermissionHandler constructs a permissionHandler.
func NewPermissionHandler(db *gorm.DB) *permissionHandler { return &permissionHandler{db: db} }

// list returns a paginated, searchable, sortable catalogue of permissions.
// Supports:
//   - Full-text search on name, description, and module
//   - Sorting by safe columns only (prevents SQL injection)
//   - Pagination via query params (?page=2&limit=20&search=approve&orderBy=module&sort=desc)
//
// Example URL:
//
//	GET /api/v1/auth/permissions?page=1&pageSize=25&search=transaction&orderBy=module&sort=asc
func (h *permissionHandler) list(c fiber.Ctx) error {
	// Parse and validate pagination + search parameters
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Whitelist allowed sort columns to prevent SQL injection via ORDER BY
	allowedSortColumns := map[string]string{
		"module":      "Module",
		"code":        "Code",
		"description": "Description",
	}

	sortColumn, ok := allowedSortColumns[search.OrderBy]
	if !ok {
		sortColumn = "Module" // Default sort column
	}

	query := h.db.WithContext(c.Context()).Model(&models.Permission{})
	if search.HasSearch() {
		query = query.Where(
			"Module LIKE ? OR Code LIKE ? OR Description LIKE ?",
			search.Search, search.Search, search.Search,
		)
	}
	query = query.Order(stableOrder(sortColumn, search.SortDirection))

	if search.Export {
		return export.Query(c, query, "Permissions", "permissions",
			[]any{"ID", "Module", "Code", "Description"},
			func(p models.Permission) []any {
				return []any{p.ID, p.Module, p.Code, p.Description}
			},
		)
	}

	var items []models.Permission
	perms, err := response.NewPaginator(c, query, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, perms)
}
