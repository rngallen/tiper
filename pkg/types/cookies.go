package types

// Browser cookie names and path for SPA auth (HttpOnly access/refresh) and CSRF.
// Keep these in one place — auth handlers, PASETO extractors, and CSRF middleware
// must agree. The Next.js client reads CsrfCookieName from document.cookie
// (see web/src/lib/api.ts) and must stay in sync manually.
const (
	AccessCookieName  = "dfms_access"
	RefreshCookieName = "dfms_refresh"
	CsrfCookieName    = "dfms_csrf"
	// CookiePath scopes auth cookies to API routes (same-origin /api via
	// Nginx or the Next.js proxy). Names intentionally omit the __Host-
	// prefix so Path can be /api rather than /.
	CookiePath = "/api"
	// CsrfCookiePath must be "/" so document.cookie on /login can read the
	// token and send X-Csrf-Token. Path=/api cookies are invisible to JS on
	// non-/api pages and caused login to return 403 Forbidden.
	CsrfCookiePath = "/"
)
