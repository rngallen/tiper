package settings

import (
	"dfms/apps/auth/middleware"
	"dfms/pkg/db"
	"dfms/pkg/permissions"

	"github.com/gofiber/fiber/v3"
)

// SettingsRouter registers /api/v1/settings endpoints.
func SettingsRouter(app *fiber.App) {
	h := NewHandler(db.Db)

	grp := app.Group("/api/v1/settings", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())

	read := middleware.PermissionMiddleware(permissions.SettingsRead, permissions.SettingsManage)
	grp.Get("/company", read, h.GetCompany).Name("settings.company.get")
	// Active currency catalogue is reference data for billing and reports.
	grp.Get("/currencies",
		middleware.PermissionMiddleware(
			permissions.SettingsRead, permissions.SettingsManage,
			permissions.MasterdataRead, permissions.InventoryRead,
			permissions.BillingRead, permissions.ReportsRead,
		),
		h.Currencies).Name("settings.currencies")
	grp.Post("/currencies",
		middleware.PermissionMiddleware(permissions.SettingsManage),
		h.CreateCurrency).Name("settings.currencies.create")
	// Countries feed company profile and master-data screens.
	grp.Get("/countries",
		middleware.PermissionMiddleware(
			permissions.SettingsRead, permissions.SettingsManage,
			permissions.MasterdataRead, permissions.CustomersRead,
		),
		h.Countries).Name("settings.countries")

	grp.Put("/company",
		middleware.PermissionMiddleware(permissions.SettingsManage),
		h.UpdateCompany).Name("settings.company.update")

	// Integration settings (DB-backed; secrets write-only; reload into memory on save).
	manage := middleware.PermissionMiddleware(permissions.SettingsManage)
	grp.Get("/integrations/mail", read, h.GetMail).Name("settings.integrations.mail.get")
	grp.Put("/integrations/mail", manage, h.UpdateMail).Name("settings.integrations.mail.update")
	grp.Get("/integrations/sms", read, h.GetSMS).Name("settings.integrations.sms.get")
	grp.Put("/integrations/sms", manage, h.UpdateSMS).Name("settings.integrations.sms.update")
	grp.Get("/integrations/sage", read, h.GetSage).Name("settings.integrations.sage.get")
	grp.Put("/integrations/sage", manage, h.UpdateSage).Name("settings.integrations.sage.update")
	grp.Post("/integrations/sage/test", manage, h.TestSage).Name("settings.integrations.sage.test")
	grp.Get("/integrations/uploads", read, h.GetUploads).Name("settings.integrations.uploads.get")
	grp.Put("/integrations/uploads", manage, h.UpdateUploads).Name("settings.integrations.uploads.update")
	grp.Get("/integrations/session", read, h.GetSession).Name("settings.integrations.session.get")
	grp.Put("/integrations/session", manage, h.UpdateSession).Name("settings.integrations.session.update")
	grp.Get("/integrations/npgis", read, h.GetNpgis).Name("settings.integrations.npgis.get")
	grp.Put("/integrations/npgis", manage, h.UpdateNpgis).Name("settings.integrations.npgis.update")

	grp.Get("/precision", h.GetPrecision).Name("settings.precision.get")
	grp.Put("/precision", manage, h.UpdatePrecision).Name("settings.precision.update")

	grp.Get("/schedules",
		middleware.PermissionMiddleware(permissions.SettingsRead, permissions.SettingsManage, permissions.JobsRun),
		h.GetSchedules).Name("settings.schedules.get")
	grp.Put("/schedules", manage, h.UpdateSchedules).Name("settings.schedules.update")
	grp.Post("/integrations/mail/test", manage, h.TestMail).Name("settings.integrations.mail.test")
	grp.Post("/integrations/sms/test", manage, h.TestSMS).Name("settings.integrations.sms.test")
}
