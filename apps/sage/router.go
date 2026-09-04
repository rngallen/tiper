package sage

import (
	"dfms/apps/auth/middleware"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func Router(app *fiber.App) {
	h := handler{db: db.Db}
	g := app.Group("/api/v1/sage", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	clients := middleware.PermissionMiddleware(
		permissions.SageRead, permissions.SageManage,
		permissions.CustomersRead, permissions.CustomersUpdate,
		permissions.MasterdataRead, permissions.MasterdataUpdate,
	)

	g.Get("/status", clients, func(c fiber.Ctx) error {
		connected := db.Sage() != nil
		return response.OkDetail(c, fiber.Map{"connected": connected})
	})
	g.Get("/clients", clients, h.listClients)
}
