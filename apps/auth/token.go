package auth

import (
	"fmt"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/utils"

	"github.com/o1egl/paseto"
	"gorm.io/gorm"
)

// createTokens generates an access + refresh token pair and persists the
// refresh token bound to the client IP. It must run inside a transaction so a
// failure leaves no partial state.
func (h *AuthHandler) createTokens(tx *gorm.DB, user *models.User, clientIP string) (*tokenResponse, error) {
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}
	if clientIP == "" {
		return nil, fmt.Errorf("client IP address is required")
	}

	now := time.Now().UTC()
	accessExpiresAt := now.Add(middleware.AccessTokenDuration)
	refreshExpiresAt := now.Add(middleware.RefreshTokenDuration)

	accessJTI, err := utils.GetULID()
	if err != nil {
		return nil, fmt.Errorf("generate access token id: %w", err)
	}
	refreshJTI, err := utils.GetULID()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}

	accessPayload := middleware.AccessToken{
		JTI:            accessJTI,
		Issuer:         middleware.TokenIssuer,
		Audience:       middleware.TokenAudience,
		ISI:            user.ID,
		Subject:        user.UID,
		Username:       user.FullName(),
		IsAdmin:        user.IsSuperUser,
		SessionVersion: user.SessionVersion,
		IssuedAt:       now,
		NotBefore:      now,
		Expiration:     accessExpiresAt,
	}
	accessToken, err := paseto.NewV2().Encrypt(middleware.SymmetricKey, accessPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}

	refreshPayload := middleware.RefreshToken{
		JTI:            refreshJTI,
		Issuer:         middleware.TokenIssuer,
		Audience:       middleware.TokenAudience,
		Subject:        user.UID,
		IpAddress:      clientIP,
		SessionVersion: user.SessionVersion,
		IssuedAt:       now,
		NotBefore:      now,
		Expiration:     refreshExpiresAt,
	}
	refreshToken, err := paseto.NewV2().Encrypt(middleware.RefreshKey, refreshPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt refresh token: %w", err)
	}

	refreshRecord := models.RefreshToken{
		Subject:   user.UID,
		IpAddress: clientIP,
		ExpiresAt: refreshExpiresAt,
		LastSeen:  now,
	}
	if err := tx.Create(&refreshRecord).Error; err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &tokenResponse{
		Email:              user.Email,
		Name:               user.FullName(),
		AccessToken:        accessToken,
		IssuedAt:           now,
		Expiration:         accessExpiresAt,
		RefreshToken:       refreshToken,
		RefreshExpiration:  refreshExpiresAt,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// createOTPAccessToken generate an otp access token to be used on header before verification
func (h *AuthHandler) createOTPAccessToken(user *models.User, challengeID, message string) (loginPendingResponse, error) {
	if user == nil {
		return loginPendingResponse{}, fmt.Errorf("user cannot be nil")
	}
	if message == "" {
		message = OTPDeliveryMessage(user)
	}

	now := time.Now().UTC()
	// Code TTL drives the UI countdown; bearer lasts longer so /mfa/resend
	// still works after the code expires without forcing a full re-login.
	codeExpiresAt := now.Add(otpTTL)
	tokenExpiresAt := now.Add(middleware.AccessTokenDuration)

	accessPayload := middleware.OTPVerifyToken{
		JTI:        challengeID,
		Issuer:     middleware.TokenIssuer,
		Audience:   middleware.TokenAudience,
		Subject:    user.UID,
		IssuedAt:   now,
		NotBefore:  now,
		Expiration: tokenExpiresAt,
	}

	accessToken, err := paseto.NewV2().Encrypt(middleware.MFAKey, accessPayload, nil)
	if err != nil {
		return loginPendingResponse{}, fmt.Errorf("encrypt otp access token: %w", err)
	}

	return loginPendingResponse{
		Token:             accessToken,
		Message:           message,
		IssuedAt:          now,
		ExpiredAt:         codeExpiresAt,
		ResendAvailableAt: now.Add(otpResendCooldown),
	}, nil
}
