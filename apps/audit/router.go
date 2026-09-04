package audit

import (
	"dfms/apps/auth/middleware"
	"dfms/pkg/db"
	"dfms/pkg/permissions"

	"github.com/gofiber/fiber/v3"
)

// AuditRouter registers /api/v1/audit endpoints.
func AuditRouter(router fiber.Router) {
	h := NewHandler(db.Db)

	audit := router.Group("/api/v1/audit", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())

	read := middleware.PermissionMiddleware(permissions.AuditRead)
	audit.Get("/", read, h.List).Name("audit.list")
}
