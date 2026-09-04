package public

import (
	"errors"
	"strings"

	"dfms/apps/reports"
	"dfms/pkg/docsig"
	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Router registers unauthenticated scan-to-confirm routes.
func Router(app *fiber.App) {
	g := app.Group("/api/v1/public")
	g.Get("/documents/:kind<string>/:uid<string>/:sig<string>", document)
	g.Get("/documents/:kind<string>/:uid<string>/:sig<string>/pdf", documentPDF)
}

func document(c fiber.Ctx) error {
	kind, uid, sig := publicParams(c)
	if !docsig.Valid(kind, uid, sig) {
		return response.NotFound(c, "document not found")
	}
	_, _, info, err := reports.BuildPublicPDF(c.Context(), kind, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "document not found")
		}
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, info)
}

func documentPDF(c fiber.Ctx) error {
	kind, uid, sig := publicParams(c)
	if !docsig.Valid(kind, uid, sig) {
		return response.NotFound(c, "document not found")
	}
	raw, filename, _, err := reports.BuildPublicPDF(c.Context(), kind, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "document not found")
		}
		return response.InternalServerError(c)
	}
	return export.SendPDF(c, filename, raw, true)
}

func publicParams(c fiber.Ctx) (kind, uid, sig string) {
	return strings.TrimSpace(c.Params("kind")), strings.TrimSpace(c.Params("uid")), strings.TrimSpace(c.Params("sig"))
}
