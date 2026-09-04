package orders

import (
	"context"
	"errors"
	"strings"

	ordersvc "dfms/internal/orders"
	"dfms/pkg/audit"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// bindBody unmarshals by content-type (JSON, form, …), then sanitizes and
// validates when the payload implements those methods. Prefer this over
// Bind().JSON() so charset variants still bind and jellydator rules run.
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

func recordAudit(c fiber.Ctx, module types.Module, action types.Action, id string, ct types.ContentType, desc string, before, after any) {
	audit.RecordHTTP(c, module, action, id, ct, desc, before, after)
}

func okUpdate(c fiber.Ctx, details, before, after any) error {
	return response.Ok(c, audit.UpdateMessage(before, after), details)
}

func parseOps(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseOpsSearchRequest(c)
	if err != nil {
		return search, response.BadRequest(c, err.Error())
	}
	return search, nil
}

func parseCatalogue(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return search, response.BadRequest(c, err.Error())
	}
	return search, nil
}

func filterOrderStatus(c fiber.Ctx, q *gorm.DB, column string) (*gorm.DB, error) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		return q, nil
	}
	if status == "returned" {
		status = string(types.OrderRejected)
	}
	if !types.OrderStatus(status).Valid() {
		return q, response.BadRequest(c, "invalid status")
	}
	return q.Where(column+" = ?", status), nil
}

func gateErr(c fiber.Ctx, err error) error {
	var g *ordersvc.GateError
	if errors.As(err, &g) {
		details := map[string]string{"code": g.Code}
		if g.Code == "nearExpiry" {
			return response.ConflictDetail(c, g.Msg, details)
		}
		return response.FailedCheck(c, g.Msg, details)
	}
	return response.BadRequest(c, err.Error())
}
