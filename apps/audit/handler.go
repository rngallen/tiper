// Package audit exposes a read-only API over the append-only audit trail.
package audit

import (
	"strings"

	"dfms/apps/models"
	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// handler serves audit-trail endpoints.
type Handler struct {
	db *gorm.DB
}

// newHandler constructs a handler.
func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// List returns paginated audit entries filtered by module/action/user/date.
func (h *Handler) List(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	search, err := response.ParseOpsSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	allowedFields := map[string]string{
		"createdAt":   "[AuditTrail].CreatedAt",
		"userName":    "[AuditTrail].UserName",
		"ipAddress":   "[AuditTrail].IPAddress",
		"userAgent":   "[AuditTrail].UserAgent",
		"requestId":   "[AuditTrail].RequestID",
		"module":      "[AuditTrail].Module",
		"action":      "[AuditTrail].Action",
		"recordType":  "[AuditTrail].RecordType",
		"description": "[AuditTrail].Description",
	}

	sortColumn, ok := allowedFields[search.OrderBy]
	if !ok {
		sortColumn = "[AuditTrail].CreatedAt"
	}

	q := db.Model(&models.AuditTrail{}).
		Where("CreatedAt BETWEEN ? AND ?", search.FromDate, search.ToDate)
	if search.HasSearch() {
		q = q.Where("UserName LIKE ? OR Description LIKE ? OR Module LIKE ? OR Action LIKE ?",
			search.Search, search.Search, search.Search, search.Search)
	}

	if module := strings.TrimSpace(c.Query("module")); module != "" {
		if len(module) > 60 {
			return response.BadRequest(c, "module is too long")
		}
		q = q.Where("Module = ?", module)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		if len(action) > 60 {
			return response.BadRequest(c, "action is too long")
		}
		q = q.Where("Action = ?", action)
	}
	if recordID := strings.TrimSpace(c.Query("recordId")); recordID != "" {
		if len(recordID) > 60 {
			return response.BadRequest(c, "recordId is too long")
		}
		q = q.Where("RecordID = ?", recordID)
	}

	q = q.Order(sortColumn + " " + search.SortDirection + ", [AuditTrail].ID ASC")

	if search.Export {
		return export.Query(c, q, "Audit", "audit",
			[]any{"When", "Actor", "IP", "Module", "Action", "Description", "Record ID"},
			func(e models.AuditTrail) []any {
				return []any{
					e.CreatedAt.Format("2006-01-02 15:04:05"),
					e.UserName,
					e.IPAddress,
					string(e.Module),
					string(e.Action),
					e.Description,
					e.RecordID,
				}
			},
		)
	}

	resp, err := response.NewPaginator(c, q, search, &[]models.AuditTrail{}).Run()
	if err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, resp)
}
