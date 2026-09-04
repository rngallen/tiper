package audit

import (
	"github.com/gofiber/fiber/v3"
)

// ──────────────────────────────────────────────────────────────────────
// Client IP extraction
// ──────────────────────────────────────────────────────────────────────

// GetClientIP returns the client IP for the current request.
//
// When DFMS.TRUST_FORWARDED_FOR is true, Fiber is configured with TrustProxy +
// EnableIPValidation (see internal/app/run.go) so c.IP() already reflects the
// real client from X-Forwarded-For after stripping trusted proxy hops. When
// that flag is false, c.IP() is the direct TCP peer.
func GetClientIP(c fiber.Ctx) string {
	return c.IP()
}
