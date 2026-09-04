// Package response provides a unified, consistent JSON API response format
// across the entire application.
//
// All HTTP handlers should use these helper functions to ensure predictable,
// developer-friendly responses with proper status codes and structured error
// details.
//
// Response shape:
//
//	{
//	  "message": "Human readable message",       // Optional
//	  "details": null | object | array | string  // Optional payload or errors
//	}
//
// Examples:
//
//	→ 200 OK:        { "message": "Updated successfully" }
//	→ 201 Created:   { "message": "Created successfully", "details": { ... } }
//	→ 422 Validation: { "message": "Invalid input", "details": [ ... ] }
package response

import (
	"dfms/pkg/constants"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"gorm.io/gorm"
)

// envelope is the standard JSON structure returned by every API endpoint.
// It is internal — handlers compose responses through the helpers below.
type envelope struct {
	Message string `json:"message,omitempty"`
	Details any    `json:"details,omitempty"`
}

// json is a small shim that always status-codes and JSON-encodes the
// envelope in one call, keeping every helper a single line.
func sendJSON(c fiber.Ctx, status int, e envelope) error {
	if e.Message == "" && status >= 200 && status < 300 {
		if msg := defaultWriteMessage(c.Method(), status); msg != "" {
			e.Message = msg
		}
	}
	return c.Status(status).JSON(e)
}

func defaultWriteMessage(method string, status int) string {
	switch strings.ToUpper(method) {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		if status == fiber.StatusCreated {
			return "Created successfully"
		}
		if strings.EqualFold(method, fiber.MethodDelete) {
			return "Deleted successfully"
		}
		return "Updated successfully"
	default:
		return ""
	}
}

// ──────────────────────────────────────────────────────────────
//  Success responses (2xx)
// ──────────────────────────────────────────────────────────────

// OkDetail returns 200 OK with a payload in "details".
// Use for read endpoints that return data without a custom message.
func OkDetail(c fiber.Ctx, details any) error {
	return sendJSON(c, fiber.StatusOK, envelope{Details: details})
}

// OkMessage returns 200 OK with only a message (no payload).
func OkMessage(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusOK, envelope{Message: message})
}

// Ok returns 200 OK with a message and optional payload.
func Ok(c fiber.Ctx, message string, details any) error {
	return sendJSON(c, fiber.StatusOK, envelope{Message: message, Details: details})
}

// Created returns 201 Created with a standard message and the new resource.
func Created(c fiber.Ctx, details any) error {
	return sendJSON(c, fiber.StatusCreated, envelope{
		Message: "Created successfully",
		Details: details,
	})
}

// Updated returns 200 OK with the standard "Updated successfully" message.
func Updated(c fiber.Ctx) error {
	return sendJSON(c, fiber.StatusOK, envelope{Message: "Updated successfully"})
}

// Deleted returns 200 OK with the standard "Deleted successfully" message.
// (204 No Content is also acceptable, but the body lets clients confirm.)
func Deleted(c fiber.Ctx) error {
	return sendJSON(c, fiber.StatusOK, envelope{Message: "Deleted successfully"})
}

// ──────────────────────────────────────────────────────────────
//  Client error responses (4xx)
// ──────────────────────────────────────────────────────────────

// BadRequest returns 400 Bad Request with the given message.
func BadRequest(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusBadRequest, envelope{Message: message})
}

// BadRequestBind returns 400 Bad Request for a request body/binding failure
// with a safe, generic message (the raw bind error is not leaked to clients).
func BadRequestBind(c fiber.Ctx, _ error) error {
	return sendJSON(c, fiber.StatusBadRequest, envelope{Message: "Invalid or malformed request body"})
}

// Unauthorized returns 401 Unauthorized.
func Unauthorized(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusUnauthorized, envelope{Message: message})
}

// Forbidden returns 403 Forbidden.
func Forbidden(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusForbidden, envelope{Message: message})
}

// NotFound returns 404 Not Found.
func NotFound(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusNotFound, envelope{Message: message})
}

// Conflict returns 409 Conflict (e.g. duplicate email/username).
func Conflict(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusConflict, envelope{Message: message})
}

// ConflictDetail returns 409 with a payload (e.g. near-expiry confirmation).
func ConflictDetail(c fiber.Ctx, message string, details any) error {
	return sendJSON(c, fiber.StatusConflict, envelope{Message: message, Details: details})
}

// FailedCheck returns 422 when a business gate failed (expired licence, missing calibration).
func FailedCheck(c fiber.Ctx, message string, details any) error {
	return sendJSON(c, fiber.StatusUnprocessableEntity, envelope{Message: message, Details: details})
}

// invalidField represents a single field-level validation failure.
type invalidField struct {
	Field string `json:"field"` // Field name (e.g. "email", "user.name")
	Error string `json:"error"` // Humanized error message
}

// UnprocessableEntity safely converts a Validate() error into a 422
// response with field-level details. Falls back to 400 when the error is
// not a validation.Errors map.
func UnprocessableEntity(c fiber.Ctx, err error) error {
	if verrs, ok := errors.AsType[validation.Errors](err); ok {
		details := make([]invalidField, 0, len(verrs))
		for field, fieldErr := range verrs {
			details = append(details, invalidField{
				Field: field,
				Error: capitalize(fieldErr.Error()),
			})
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(envelope{
			Message: "Invalid input",
			Details: details,
		})
	}

	// Not a validation error → return generic message
	return c.Status(fiber.StatusBadRequest).JSON(envelope{Message: err.Error()})
}

// capitalize uppercases the first rune of s. It is rune-safe so multi-byte
// characters are not corrupted.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// TooManyRequests returns 429 Too Many Requests using the canonical
// rate-limit error message from the constants package.
func TooManyRequests(c fiber.Ctx) error {
	return sendJSON(c, fiber.StatusTooManyRequests, envelope{
		Message: constants.ErrTooManyRequests.Error(),
	})
}

// ClientClosedRequest returns 499 Client Closed Request (a non-standard but
// widely-used status code, originally from nginx).
func ClientClosedRequest(c fiber.Ctx) error {
	return sendJSON(c, 499, envelope{Message: "Client closed request"})
}

// ──────────────────────────────────────────────────────────────
//  Server error responses (5xx)
// ──────────────────────────────────────────────────────────────

// NotImplemented returns 501 when a catalogued inbound rail has a stored
// token but payload ingest is not live yet.
func NotImplemented(c fiber.Ctx, message string) error {
	return sendJSON(c, fiber.StatusNotImplemented, envelope{Message: message})
}

// InternalServerError returns 500 with a safe generic message.
// Raw error details are intentionally never exposed to clients.
func InternalServerError(c fiber.Ctx) error {
	return sendJSON(c, fiber.StatusInternalServerError, envelope{
		Message: "Internal server error",
	})
}

// ServiceUnavailable returns 503 Service Unavailable. An optional message
// may be provided to override the default "Service temporarily unavailable".
func ServiceUnavailable(c fiber.Ctx, message ...string) error {
	msg := "Service temporarily unavailable"
	if len(message) > 0 && strings.TrimSpace(message[0]) != "" {
		msg = message[0]
	}
	return sendJSON(c, fiber.StatusServiceUnavailable, envelope{Message: msg})
}

// GatewayTimeout returns 504 Gateway Timeout.
func GatewayTimeout(c fiber.Ctx) error {
	return sendJSON(c, fiber.StatusGatewayTimeout, envelope{Message: "Operation timed out"})
}

// IsDuplicate reports whether err is a unique-constraint violation.
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) || errors.Is(err, constants.ErrDuplicateEntry) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique index")
}
