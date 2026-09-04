// Package utils holds small helpers: secure passwords, ULIDs, JSON, dates, MIME.
package utils

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// defaultPasswordLength is the fallback when length <= 0 (~78 bits with charset below).
const defaultPasswordLength = 12

// passwordCharset: uppercase, lowercase, digits, common symbols (91 chars).
const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?/~`"

// GenerateSecurePassword creates a cryptographically secure random password
// of the requested length using rejection sampling to eliminate modulo bias.
func GenerateSecurePassword(length int) (string, error) {
	if length <= 0 {
		length = defaultPasswordLength
	}

	charsetLen := len(passwordCharset)
	if charsetLen == 0 {
		return "", fmt.Errorf("charset is empty")
	}

	// Largest multiple of charsetLen that fits in a byte (< 256). Drawn bytes
	// at or above this threshold are rejected to avoid modulo bias. With a
	// 91-character charset the acceptance rate is 182/256 ≈ 71%.
	maxValidValue := (256 / charsetLen) * charsetLen

	password := make([]byte, length)
	buf := make([]byte, length*2)

	i := 0
	for i < length {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to read random bytes: %w", err)
		}
		for _, b := range buf {
			if int(b) >= maxValidValue {
				continue
			}
			password[i] = passwordCharset[int(b)%charsetLen]
			i++
			if i == length {
				break
			}
		}
	}

	return string(password), nil
}

// ulidEntropy is process-global, crypto-random, and monotonic under lock so
// concurrent callers never collide in the same millisecond.
var ulidEntropy = &ulid.LockedMonotonicReader{
	MonotonicReader: ulid.Monotonic(rand.Reader, 0),
}

// GetULID returns a new public identifier (26-char ULID).
func GetULID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), ulidEntropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return id.String(), nil
}
