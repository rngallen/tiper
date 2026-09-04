package ewura

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	ewurasync "dfms/internal/ewura"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func Router(app *fiber.App) {
	g := app.Group("/api/v1/ewura", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	read := middleware.PermissionMiddleware(permissions.EwuraRead, permissions.EwuraManage)
	manage := middleware.PermissionMiddleware(permissions.EwuraManage)

	g.Get("/licenses", read, listLicenses)
	g.Get("/licenses.pdf", read, listLicenses)
	g.Get("/licenses/options", read, licenseOptions)
	g.Get("/classes", read, func(c fiber.Ctx) error {
		var classes []string
		if err := db.Db.Model(&models.EwuraPetroleumLicense{}).Distinct("LicenseClass").Pluck("LicenseClass", &classes).Error; err != nil {
			return err
		}
		return response.OkDetail(c, classes)
	})
	g.Post("/licenses/sync", manage, func(c fiber.Ctx) error {
		var in struct {
			URL string `json:"url"`
		}
		_ = c.Bind().JSON(&in)
		if err := ewurasync.Sync(c.Context(), db.Db, in.URL); err != nil {
			return err
		}
		return response.OkMessage(c, "EWURA licenses synced")
	})
}
