package response

import (
	"errors"

	"dfms/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

// ErrorHandler is the central Fiber error handler. It maps known sentinel
// errors and *fiber.Error values to consistent JSON envelopes and hides
// internal error details from clients for anything unexpected.
func ErrorHandler(c fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	// Sentinel domain errors → specific status codes.
	switch {
	case errors.Is(err, constants.ErrInvalidCredentials),
		errors.Is(err, constants.ErrUserInactive),
		errors.Is(err, constants.ErrUserLocked):
		return Unauthorized(c, err.Error())
	case errors.Is(err, constants.ErrDuplicateEntry),
		errors.Is(err, constants.ErrUserInUse):
		return Conflict(c, err.Error())
	case errors.Is(err, constants.ErrTooManyRequests):
		return TooManyRequests(c)
	case errors.Is(err, constants.ErrInvalidInput):
		return BadRequest(c, err.Error())
	}

	// Fiber HTTP errors carry their own status code.
	if fe, ok := errors.AsType[*fiber.Error](err); ok {
		return sendJSON(c, fe.Code, envelope{Message: fe.Message})
	}

	// Anything else is an unexpected server error.
	return InternalServerError(c)
}
