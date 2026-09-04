package middleware

import (
	"errors"
	"time"
)

const (
	Issuer   = "tiper"
	Audience = "tiper-dfms"
)

// AccessToken represents the claims inside a PASETO access token.
// We use ISI as internal subject identifier
type AccessToken struct {
	// Standard registered claims
	Issuer     string    `json:"iss,omitempty"`
	Subject    string    `json:"sub"` // ← Public User ID (ULID)
	Audience   string    `json:"aud,omitempty"`
	IssuedAt   time.Time `json:"iat"`
	NotBefore  time.Time `json:"nbf,omitzero"`
	Expiration time.Time `json:"exp"`
	JTI        string    `json:"jti,omitempty"`
	// Application-specific claims
	Username string `json:"username,omitempty"`
	IsAdmin  bool   `json:"isAdmin,omitempty"`
	ISI      uint   `json:"isi"`
	// SessionVersion must match User.SessionVersion or the token is rejected.
	SessionVersion uint `json:"sv"`
}

// Validate performs time-based and basic claim validation
func (t AccessToken) Validate() error {
	now := time.Now().UTC()

	if !t.IssuedAt.IsZero() && now.Before(t.IssuedAt) {
		return errors.New("token was issued in the future")
	}

	if !t.NotBefore.IsZero() && now.Before(t.NotBefore) {
		return errors.New("token is not valid yet")
	}

	if !t.Expiration.IsZero() && now.After(t.Expiration) {
		return errors.New("token has expired")
	}

	// Subject
	if t.Subject == "" {
		return errors.New("token missing subject (user identifier)")
	}

	return nil
}

// RefreshToken carries the PASETO refresh-token claims (IP-bound).
type RefreshToken struct {
	JTI            string    `json:"jti"`
	Issuer         string    `json:"iss"`
	Audience       string    `json:"aud"`
	Subject        string    `json:"sub"` // user ULID
	IpAddress      string    `json:"ipAddress"`
	SessionVersion uint      `json:"sv"`
	IssuedAt       time.Time `json:"iat"`
	NotBefore      time.Time `json:"nbf"`
	Expiration     time.Time `json:"exp"`
}

// Validate performs time-based and basic claim validation
func (t RefreshToken) Validate() error {
	now := time.Now().UTC()

	if !t.IssuedAt.IsZero() && now.Before(t.IssuedAt) {
		return errors.New("token was issued in the future")
	}

	if !t.NotBefore.IsZero() && now.Before(t.NotBefore) {
		return errors.New("token is not valid yet")
	}

	if !t.Expiration.IsZero() && now.After(t.Expiration) {
		return errors.New("token has expired")
	}

	// Subject
	if t.Subject == "" {
		return errors.New("token missing subject (user identifier)")
	}

	return nil
}

// OTPVerifyToken carries the PASETO otp-verify-token claims.
type OTPVerifyToken struct {
	JTI        string    `json:"jti"`
	Issuer     string    `json:"iss"`
	Audience   string    `json:"aud"`
	Subject    string    `json:"sub"` // user ULID
	IssuedAt   time.Time `json:"iat"`
	NotBefore  time.Time `json:"nbf"`
	Expiration time.Time `json:"exp"`
}

// Validate performs time-based and basic claim validation
func (t OTPVerifyToken) Validate() error {
	now := time.Now().UTC()

	if !t.IssuedAt.IsZero() && now.Before(t.IssuedAt) {
		return errors.New("token was issued in the future")
	}

	if !t.NotBefore.IsZero() && now.Before(t.NotBefore) {
		return errors.New("token is not valid yet")
	}

	if !t.Expiration.IsZero() && now.After(t.Expiration) {
		return errors.New("token has expired")
	}

	// Subject
	if t.Subject == "" {
		return errors.New("token missing subject (user identifier)")
	}

	return nil
}
