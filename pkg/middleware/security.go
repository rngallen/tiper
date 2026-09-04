// Package middleware contains HTTP middleware shared across the Fiber app.
//
// security.go installs sensible HTTP security headers via helmet, with two
// CSP profiles:
//   - The strict profile is applied to every route by default.
//   - A docs profile is used on /api-docs (and /swagger, debug builds only) so
//     Swagger UI's inline bootstrap script is permitted while assets remain
//     same-origin (the swagger-ui bundle is embedded and served locally).
//
// Helmet middleware is constructed once at startup so the per-request handler
// is cheap and free of allocations.
package middleware

import (
	"dfms/pkg/config"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

// productionCSP is the strict Content-Security-Policy applied to normal HTML
// responses. Newlines and excess whitespace are collapsed because RFC 9110
// header values must not contain CR/LF. upgrade-insecure-requests is appended
// only when cookies require HTTPS (see contentSecurityPolicy).
const productionCSP = `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; frame-src 'self' blob: data:; object-src 'self' blob: data:; frame-ancestors 'self'; base-uri 'self'; form-action 'self';`

// apiDocsCSP relaxes script-src for Swagger UI's inline bootstrap while keeping
// all scripts and styles on the same origin — the swagger-ui-dist bundle is
// embedded in the binary and served from /api-docs/assets/ (no CDN).
// connect-src 'self' allows "Try it out" requests against this API.
const apiDocsCSP = `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; frame-src 'self' blob: data:; object-src 'none'; base-uri 'self'; form-action 'self';`

// SetupSecurityMiddleware registers security middleware on the supplied
// Fiber app. Safe to call once during startup. It is a no-op on a nil app.
func SetupSecurityMiddleware(app *fiber.App) {
	if app == nil {
		return
	}

	hstsMaxAge := 0
	if config.Conf.App.CookieSecure() {
		hstsMaxAge = 31536000
	}
	strictHelmet := helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		XDNSPrefetchControl:   "off",
		XDownloadOptions:      "noopen",
		XPermittedCrossDomain: "none",
		HSTSMaxAge:            hstsMaxAge,

		CrossOriginEmbedderPolicy: "",
		// CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy: "same-origin",
		// cross-origin: the Next.js UI (localhost:3000) reads JSON from this API
		// (localhost:8080). "same-origin" CORP makes the browser report a CORS
		// failure even when Access-Control-Allow-Origin is correct.
		CrossOriginResourcePolicy: "cross-origin",

		ContentSecurityPolicy: contentSecurityPolicy(),
	})

	app.Use(func(c fiber.Ctx) error {
		path := c.Path()
		// Relaxed CSP for Swagger UI (inline bootstrap script).
		if c.Method() == fiber.MethodGet && (path == "/swagger" || strings.HasPrefix(path, "/api-docs")) && config.Conf.App.Debug {
			c.Set("Content-Security-Policy", apiDocsCSP)
			c.Set("X-Frame-Options", "SAMEORIGIN")
			c.Set("X-Content-Type-Options", "nosniff")
			return c.Next()
		}
		return strictHelmet(c)
	})
}

// contentSecurityPolicy is productionCSP, plus upgrade-insecure-requests when
// the process is served on HTTPS. Intranet HTTP (AllowInsecureHTTP / Debug)
// must not tell the browser to rewrite http:// to https://.
func contentSecurityPolicy() string {
	if config.Conf.App.CookieSecure() {
		return productionCSP + " upgrade-insecure-requests;"
	}
	return productionCSP
}
