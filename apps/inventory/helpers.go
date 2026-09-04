package inventory

import (
	"context"

	"dfms/pkg/audit"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

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

func recordAudit(c fiber.Ctx, action types.Action, id string, ct types.ContentType, desc string, before, after any) {
	audit.RecordHTTP(c, types.ModuleInventory, action, id, ct, desc, before, after)
}

func okUpdate(c fiber.Ctx, details, before, after any) error {
	return response.Ok(c, audit.UpdateMessage(before, after), details)
}
