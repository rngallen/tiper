package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	sealedPrefix = "enc:v1:"
	aadText      = "tiper-dfms:integration:v1"
)

var aad = []byte(aadText)

// IsSealed reports whether s is an AES-GCM sealed secret produced by Seal.
func IsSealed(s string) bool {
	return strings.HasPrefix(s, sealedPrefix)
}

// Seal encrypts plaintext with AES-GCM and returns enc:v1:+base64(nonce|ciphertext).
func Seal(plaintext, keyMaterial string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := deriveKey(keyMaterial)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("aes gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), aad)
	payload := append(nonce, sealed...)
	return sealedPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

// Open decrypts a value sealed by Seal.
func Open(sealed, keyMaterial string) (string, error) {
	if sealed == "" {
		return "", nil
	}
	if !IsSealed(sealed) {
		return sealed, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, sealedPrefix))
	if err != nil {
		return "", fmt.Errorf("decode sealed secret: %w", err)
	}
	key, err := deriveKey(keyMaterial)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("aes gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("sealed secret too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}

func deriveKey(keyMaterial string) ([]byte, error) {
	keyMaterial = strings.TrimSpace(keyMaterial)
	if keyMaterial == "" {
		return nil, errors.New("empty key material")
	}
	if len(keyMaterial) == 64 {
		if key, err := hex.DecodeString(keyMaterial); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	if len(keyMaterial) == 32 {
		return []byte(keyMaterial), nil
	}
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:], nil
}
