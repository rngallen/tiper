package middleware

import (
	"strings"

	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

// SkipCSRF reports whether CSRF double-submit should be skipped for this request.
//
// Cookie-authenticated browsers always require CSRF — even if the request also
// carries Authorization: Bearer. Skipping solely because a Bearer header is
// present would let a cross-site caller send credentials:include plus a dummy
// Bearer and bypass the check while the session cookie still authenticates.
//
// Skip when:
//   - the path is MFA verify/resend (short-lived MFA PASETO in Authorization;
//     leftover access/refresh cookies must not block OTP),
//   - the path is the session heartbeat (cookie-authenticated keepalive; the
//     CSRF cookie is often missing on the first activity tick), or
//   - there is no access/refresh cookie and Authorization is Bearer
//     (API clients).
func SkipCSRF(c fiber.Ctx) bool {
	return ShouldSkipCSRF(
		c.Path(),
		c.Cookies(types.AccessCookieName),
		c.Cookies(types.RefreshCookieName),
		c.Get(fiber.HeaderAuthorization),
	)
}

// ShouldSkipCSRF is the pure decision used by SkipCSRF (kept separate for tests).
func ShouldSkipCSRF(path, accessCookie, refreshCookie, authorization string) bool {
	p := strings.TrimSuffix(path, "/")
	if strings.HasPrefix(p, "/api/v1/auth/mfa/") {
		return true
	}
	if strings.HasPrefix(p, "/api/v1/public/") {
		return true
	}
	if p == "/api/v1/auth/session/touch" {
		return true
	}
	if strings.TrimSpace(accessCookie) != "" || strings.TrimSpace(refreshCookie) != "" {
		return false
	}
	authz := strings.ToLower(strings.TrimSpace(authorization))
	return strings.HasPrefix(authz, "bearer ") && len(strings.TrimSpace(authz[len("bearer "):])) > 0
}
