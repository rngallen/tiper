package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// OverwriteRealIP replaces any client-supplied X-Real-IP with Fiber's
// validated client IP (c.IP()).
//
// When the API sits behind Nginx, clients can send X-Real-IP themselves.
// Fiber's TrustProxy + EnableIPValidation already walks X-Forwarded-For
// (see internal/app/run.go). This middleware then deletes X-Real-IP and
// writes a single line from c.IP(), matching the Fiber proxy security
// guidance: Set alone is not enough because a duplicate header would leave
// an attacker-controlled value on the wire.
//
// Nginx must overwrite (not pass through) the forwarding header at the edge:
//
//	proxy_set_header X-Forwarded-For $remote_addr;
//	proxy_set_header X-Forwarded-Proto $scheme;
//	proxy_set_header X-Forwarded-Host $host;
func OverwriteRealIP() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		h := &c.Request().Header
		h.Del("X-Real-IP")
		if ip != "" {
			h.Add("X-Real-IP", ip)
		}
		return c.Next()
	}
}
