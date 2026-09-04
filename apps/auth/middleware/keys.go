// Package middleware implements PASETO-based authentication (access + IP-bound
// refresh tokens), two-factor login, and role/permission authorization for the
// TIPER DFMS.
package middleware

import (
	"encoding/hex"
	"fmt"
	"time"

	"dfms/internal/integrations"
	"dfms/pkg/config"
)

// Symmetric keys for PASETO v2 local encryption. Set by InitKeys at startup.
var (
	SymmetricKey []byte // encrypts access tokens
	RefreshKey   []byte // encrypts refresh tokens
	MFAKey       []byte // encrypts OTP codes
)

// Token lifetimes
var (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 1 * time.Hour
	PermsTTL             = 1 * time.Hour
)

// SessionIdleWindow is how long a refresh row may sit unused. Enforced on
// /auth/refresh and /auth/session/touch via RefreshToken.LastSeen (CreatedAt
// if LastSeen was never written). Continue and the SPA heartbeat slide LastSeen.
func SessionIdleWindow() time.Duration {
	return integrations.LiveSession().IdleWindow()
}

// Token issuer/audience guard against a leaked key being used to forge tokens
// for a different deployment.
const (
	TokenIssuer   = "tiper-dfms"
	TokenAudience = "tiper-dfms-api"
)

// InitKeys decodes the hex-encoded symmetric keys from configuration. Each key
// must be exactly 32 bytes (64 hex characters).
func InitKeys() error {
	var err error
	SymmetricKey, err = hex.DecodeString(config.Conf.App.SymmetricKey)
	if err != nil {
		return fmt.Errorf("invalid DFMS.SYMMETRIC_KEY: %w", err)
	}
	RefreshKey, err = hex.DecodeString(config.Conf.App.RefreshKey)
	if err != nil {
		return fmt.Errorf("invalid DFMS.REFRESH_KEY: %w", err)
	}
	MFAKey, err = hex.DecodeString(config.Conf.App.MFAKey)
	if err != nil {
		return fmt.Errorf("invalid DFMS.MFA_KEY: %w", err)
	}
	return nil
}
