package middleware

import (
	"dfms/pkg/logs"
	"dfms/pkg/permissions"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"strings"

	"github.com/goccy/go-json"
	pasetoware "github.com/gofiber/contrib/v3/paseto"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
)

// fromBearerPASETO reads Authorization: Bearer <token> without RFC 7235
// token68 validation. Fiber's FromAuthHeader("Bearer") rejects some valid
// PASETO payloads (e.g. unexpected alphabet), which surfaces as
// "missing PASETO token" even when the header is present.
func fromBearerPASETO() extractors.Extractor {
	return extractors.Extractor{
		Extract: func(c fiber.Ctx) (string, error) {
			h := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
			if h == "" {
				return "", extractors.ErrNotFound
			}
			const prefix = "bearer "
			if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
				return "", extractors.ErrNotFound
			}
			token := strings.TrimSpace(h[len(prefix):])
			if token == "" {
				return "", extractors.ErrNotFound
			}
			return token, nil
		},
		Key:        fiber.HeaderAuthorization,
		Source:     extractors.SourceAuthHeader,
		AuthScheme: "Bearer",
	}
}

// accessExtractor prefers the HttpOnly access cookie (browser), then Bearer
// (in-memory SPA token / API clients).
func accessExtractor() extractors.Extractor {
	return extractors.Chain(
		extractors.FromCookie(types.AccessCookieName),
		fromBearerPASETO(),
	)
}

// refreshExtractor prefers the HttpOnly refresh cookie, then Bearer.
func refreshExtractor() extractors.Extractor {
	return extractors.Chain(
		extractors.FromCookie(types.RefreshCookieName),
		fromBearerPASETO(),
	)
}

// PasetoMiddleware authenticates a request using the PASETO access token.
func PasetoMiddleware() fiber.Handler {
	return pasetoware.New(pasetoware.Config{
		SymmetricKey: SymmetricKey,
		Extractor:    accessExtractor(),
		Validate: func(decrypted []byte) (any, error) {
			var token AccessToken
			if err := json.Unmarshal(decrypted, &token); err != nil {
				logs.Error(err)
				return nil, pasetoware.ErrDataUnmarshal
			}

			if err := token.Validate(); err != nil {
				return nil, err
			}

			return token, nil
		},

		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Handle errors in a custom way
			logs.Errorf("%s - : %v", c.IP(), err.Error())
			return response.Unauthorized(c, "Unauthorized")
		},
	})
}

// PasetoRefreshMiddleware authenticates a request using the PASETO refresh token.
func PasetoRefreshMiddleware() fiber.Handler {
	return pasetoware.New(pasetoware.Config{
		SymmetricKey: RefreshKey,
		Extractor:    refreshExtractor(),
		Validate: func(decrypted []byte) (any, error) {
			var token RefreshToken
			if err := json.Unmarshal(decrypted, &token); err != nil {
				logs.Error(err)
				return nil, pasetoware.ErrDataUnmarshal
			}

			if err := token.Validate(); err != nil {
				return nil, err
			}

			return token, nil
		},

		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Handle errors in a custom way
			logs.Errorf("%s - : %v", c.IP(), err)
			return response.Unauthorized(c, "Unauthorized")
		},
	})
}

// OTPVerifyMiddleware authenticates a request using the PASETO MFA token.
func OTPVerifyMiddleware() fiber.Handler {
	return pasetoware.New(pasetoware.Config{
		SymmetricKey: MFAKey,
		Extractor:    fromBearerPASETO(),
		Validate: func(decrypted []byte) (any, error) {
			var token OTPVerifyToken
			if err := json.Unmarshal(decrypted, &token); err != nil {
				logs.Error(err)
				return nil, pasetoware.ErrDataUnmarshal
			}

			if err := token.Validate(); err != nil {
				return nil, err
			}

			return token, nil
		},

		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Handle errors in a custom way
			logs.Errorf("%s - : %v", c.IP(), err)
			return response.Unauthorized(c, "Unauthorized")
		},
	})
}

// SessionVersionMiddleware rejects access tokens whose sv claim no longer
// matches the user's current SessionVersion. The current version is served
// from Ristretto (same pattern as permissions); the DB is hit only on miss.
func SessionVersionMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		claims := GetUserClaimsFromContext(c)
		if claims.ISI == 0 || claims.Subject == "" {
			return response.Unauthorized(c, "Unauthorized")
		}

		sv, found := GetSessionVersion(claims.Subject)
		if !found {
			loaded, err := loadSessionVersionFromDb(c.Context(), claims.ISI)
			if err != nil {
				logs.Errorf("session version lookup user=%d: %v", claims.ISI, err)
				return response.Unauthorized(c, "Unauthorized")
			}
			sv = loaded
			SetSessionVersion(claims.Subject, sv, SessionVersionTTL)
			SessionVersionCache.Wait() // Ensure it passes through buffers
		}

		if claims.SessionVersion != sv {
			return response.Unauthorized(c, "session expired")
		}
		return c.Next()
	}
}

// PermissionMiddleware enforces that the authenticated user holds at least one
// of the required permission codes. Super-users bypass the check.
func PermissionMiddleware(requiredPerms ...string) fiber.Handler {
	return func(c fiber.Ctx) error {

		claims := GetUserClaimsFromContext(c)
		if claims.IsAdmin {
			return c.Next()
		}

		if len(requiredPerms) == 0 {
			return response.Forbidden(c, "insufficient permissions")
		}

		// Try cache first
		userPermissions, found := GetUserPermissions(claims.Subject)
		if !found {
			// Load from database
			userPermissions = loadPermissionsFromDb(c.Context(), claims.ISI)

			// Store in cache
			SetUserPermissions(claims.Subject, userPermissions, PermsTTL)
			PermissionCache.Wait() // Ensure it passes through buffers
		}

		if !permissions.SatisfiesAny(userPermissions, requiredPerms...) {
			logs.Infof("permission denied: subject=%s lacks %v on %s %s",
				claims.Subject, requiredPerms, c.Method(), c.Path())
			return response.Forbidden(c, "insufficient permissions")
		}
		return c.Next()
	}
}
