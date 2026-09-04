package masterdata

import (
	"context"
	"errors"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/audit"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const productUnitLitre = "L"

// bindBody unmarshals the request by content-type (JSON, form, …), then
// sanitizes and validates when the payload implements those methods.
// Prefer this over Bind().JSON() so form posts and charset variants still bind,
// and so jellydator rules run the same way as auth/settings.
func bindBody[T any](c fiber.Ctx, dest *T) error {
	if err := c.Bind().Body(dest); err != nil {
		return response.BadRequestBind(c, err)
	}
	if s, ok := any(dest).(interface{ Sanitize() }); ok {
		s.Sanitize()
	}
	if v, ok := any(dest).(interface{ Validate(context.Context) error }); ok {
		if err := v.Validate(c.Context()); err != nil {
			return response.UnprocessableEntity(c, err)
		}
	}
	return nil
}

func activeOrDefault(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

func countryCodePtr(s string) *string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 {
		return nil
	}
	return &s
}

func parseOptionalDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		d := t.UTC().Truncate(24 * time.Hour)
		return &d
	}
	return nil
}

func parseDec(s string) (decimal.Decimal, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

func firstUID[T any](db *gorm.DB, uid string, dest *T) error {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return gorm.ErrRecordNotFound
	}
	return db.Where("UID = ?", uid).First(dest).Error
}

func idByUID[T any](db *gorm.DB, uid string) uint {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return 0
	}
	var row T
	if db.Where("UID = ?", uid).First(&row).Error != nil {
		return 0
	}
	return rowID(&row)
}

func rowID(v any) uint {
	switch m := v.(type) {
	case *models.StockCategory:
		return m.ID
	case *models.Product:
		return m.ID
	case *models.StockStatus:
		return m.ID
	case *models.Tank:
		return m.ID
	case *models.Vessel:
		return m.ID
	case *models.Depot:
		return m.ID
	case *models.Customer:
		return m.ID
	case *models.Supplier:
		return m.ID
	case *models.Transporter:
		return m.ID
	case *models.Driver:
		return m.ID
	case *models.Truck:
		return m.ID
	case *models.Destination:
		return m.ID
	case *models.District:
		return m.ID
	default:
		return 0
	}
}

func notFound(c fiber.Ctx, err error, msg string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NotFound(c, msg)
	}
	logs.Error(err)
	return response.InternalServerError(c)
}

func writeErr(c fiber.Ctx, err error, duplicate string) error {
	if response.IsDuplicate(err) {
		return response.Conflict(c, duplicate)
	}
	logs.Error(err)
	return response.InternalServerError(c)
}

func recordAudit(c fiber.Ctx, module types.Module, action types.Action, id string, ct types.ContentType, desc string, before, after any) {
	audit.RecordHTTP(c, module, action, id, ct, desc, before, after)
}

func okUpdate(c fiber.Ctx, details, before, after any) error {
	return response.Ok(c, audit.UpdateMessage(before, after), details)
}
