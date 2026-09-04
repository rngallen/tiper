// Package constants holds sentinel errors and shared validation messages.
// Compare with errors.Is — do not match on error strings.
package constants

import "errors"

const (
	Max60Chars  = "must not exceed 60 characters"
	Max255Chars = "must not exceed 255 characters"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserInactive        = errors.New("account is inactive")
	ErrUserLocked          = errors.New("account is locked; ask an administrator to restore access")
	ErrUserInUse           = errors.New("account is in use")
	ErrDuplicateEntry      = errors.New("record already exists")
	ErrInvalidInput        = errors.New("invalid input data")
	ErrTooManyRequests     = errors.New("too many requests, please try again later")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrPasswordReused      = errors.New("new password must not match your current or last 5 passwords")
)
