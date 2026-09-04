package middleware

import (
	pasetoware "github.com/gofiber/contrib/v3/paseto"
	"github.com/gofiber/fiber/v3"
)

// OTPClaimsFromContext is the single point of truth for extracting OTP claims from the context.
// It returns the zero OTPClaims if the value is missing.
func OTPClaimsFromContext(c fiber.Ctx) OTPVerifyToken {
	v := pasetoware.FromContext(c)
	if claims, ok := v.(OTPVerifyToken); ok {
		return claims
	}
	return OTPVerifyToken{}
}

// claimsFromContext is the single point of truth for extracting access-token claims.
// It returns the zero AccessToken if the value is missing
func claimsFromContext(c fiber.Ctx) AccessToken {
	v := pasetoware.FromContext(c)
	if claims, ok := v.(AccessToken); ok {
		return claims
	}
	return AccessToken{}
}

// RefreshClaimsFromContext is the single point of truth for extracting refresh-token claims.
// It returns the zero RefreshToken if the value is missing
func RefreshClaimsFromContext(c fiber.Ctx) RefreshToken {
	v := pasetoware.FromContext(c)
	if claims, ok := v.(RefreshToken); ok {
		return claims
	}
	return RefreshToken{}
}

// GetUserIDFromContext returns the numeric user ID stored in the access
// token. Returns 0 if no claims are present (which downstream handlers
// must treat as "not authenticated").
func GetUserIDFromContext(c fiber.Ctx) uint {
	return claimsFromContext(c).ISI
}

// GetUserUIDFromContext returns the ULID (public subject identifier) from
// the access token. Returns "" if no claims are present.
func GetUserUIDFromContext(c fiber.Ctx) string {
	return claimsFromContext(c).Subject
}

// GetUserClaimsFromContext returns the full claim set. Returns the zero
// value if no claims are present.
func GetUserClaimsFromContext(c fiber.Ctx) AccessToken {
	return claimsFromContext(c)
}
