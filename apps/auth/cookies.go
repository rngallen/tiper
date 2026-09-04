package auth

import (
	"dfms/internal/integrations"
	"dfms/pkg/config"
	"dfms/pkg/types"
	"time"

	"github.com/gofiber/fiber/v3"
)

func authCookieBase(name, value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     types.CookiePath,
		HTTPOnly: true,
		Secure:   config.Conf.App.CookieSecure(),
		SameSite: fiber.CookieSameSiteLaxMode,
		Expires:  expires,
	}
}

// SetAuthCookies writes access + refresh HttpOnly cookies from a token pair.
func SetAuthCookies(c fiber.Ctx, tokens *tokenResponse) {
	if tokens == nil {
		return
	}
	c.Cookie(authCookieBase(types.AccessCookieName, tokens.AccessToken, tokens.Expiration))
	c.Cookie(authCookieBase(types.RefreshCookieName, tokens.RefreshToken, tokens.RefreshExpiration))
}

// ClearAuthCookies expires auth cookies (logout / failed session).
func ClearAuthCookies(c fiber.Ctx) {
	past := time.Unix(0, 0).UTC()
	c.Cookie(authCookieBase(types.AccessCookieName, "", past))
	c.Cookie(authCookieBase(types.RefreshCookieName, "", past))
}

// publicTokenResponse is the browser-safe login/refresh payload.
// Refresh stays HttpOnly-cookie only. Access is also returned in JSON so the
// SPA can keep a short-lived in-memory Bearer if the cookie hop is delayed.
type publicTokenResponse struct {
	Email                  string    `json:"email"`
	Name                   string    `json:"name"`
	IssuedAt               time.Time `json:"issuedAt"`
	AccessToken            string    `json:"token,omitempty"`
	Expiration             time.Time `json:"tokenExpiresAt"`
	RefreshExpiration      time.Time `json:"refreshTokenExpiresAt"`
	MustChangePassword     bool      `json:"mustChangePassword"`
	SessionIdleMinutes     int       `json:"sessionIdleMinutes"`
	SessionIdleWarnSeconds int       `json:"sessionIdleWarnSeconds"`
}

// Public converts the full token pair into the browser-safe payload: the
// refresh token is dropped (it travels only in the HttpOnly cookie) while its
// expiry is kept so the SPA knows when the session ends. Safe on a nil
// receiver, returning a zero value.
func (t *tokenResponse) Public() publicTokenResponse {
	if t == nil {
		return publicTokenResponse{}
	}
	sess := integrations.LiveSession()
	return publicTokenResponse{
		Email:                  t.Email,
		Name:                   t.Name,
		IssuedAt:               t.IssuedAt,
		AccessToken:            t.AccessToken,
		Expiration:             t.Expiration,
		RefreshExpiration:      t.RefreshExpiration,
		MustChangePassword:     t.MustChangePassword,
		SessionIdleMinutes:     sess.IdleMinutes,
		SessionIdleWarnSeconds: sess.WarnSeconds,
	}
}
