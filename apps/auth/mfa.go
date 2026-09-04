package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	otpLength         = 6
	otpTTL            = 5 * time.Minute
	otpResendCooldown = 60 * time.Second // min wait before /mfa/resend
	otpMaxAttempts    = 5
)

// Deliverer dispatches an OTP code to a user over a channel.
type Deliverer interface {
	Send(ctx context.Context, user *models.User, code string) error
}

// LoggingDeliverer writes the OTP to the application log. Development only.
type LoggingDeliverer struct{}

// Send logs the plaintext OTP at warn level instead of delivering it. It is
// the package default until startup wires the real email/SMS deliverer via
// SetDeliverer, so codes remain retrievable from the log in dev/test setups
// with no mail or SMS configured.
func (LoggingDeliverer) Send(_ context.Context, user *models.User, code string) error {
	if user == nil {
		return fmt.Errorf("otp: user required")
	}
	phone := strings.TrimSpace(user.Profile.PhoneNumber)
	if phone != "" {
		logs.Warnf("[MFA] OTP for %s via email and SMS %s: %s", user.Email, phone, code)
	} else {
		logs.Warnf("[MFA] OTP for %s via email: %s", user.Email, code)
	}
	return nil
}

var deliverer Deliverer = LoggingDeliverer{}

// SetDeliverer overrides the OTP deliverer (called during startup once real
// channels are configured).
func SetDeliverer(d Deliverer) {
	if d != nil {
		deliverer = d
	}
}

// OTPDeliveryMessage describes where the verification code was sent.
func OTPDeliveryMessage(user *models.User) string {
	if user != nil && strings.TrimSpace(user.Profile.PhoneNumber) != "" {
		return "A verification code has been sent to your email and phone."
	}
	return "A verification code has been sent to your email."
}

// issueOTPChallenge creates and persists a hashed OTP challenge, returning the
// challenge id and the plaintext code (for delivery).
func issueOTPChallenge(tx *gorm.DB, user *models.User, clientIp string) (challengeID, code string, err error) {
	code, err = generateOTP(otpLength)
	if err != nil {
		return "", "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash otp: %w", err)
	}
	challengeID, err = utils.GetULID()
	if err != nil {
		return "", "", err
	}

	challenge := models.UserOTPChallenge{
		ID:        challengeID,
		UserID:    user.ID,
		IpAddress: clientIp,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(otpTTL),
	}
	if err := tx.Create(&challenge).Error; err != nil {
		return "", "", fmt.Errorf("persist otp challenge: %w", err)
	}
	return challengeID, code, nil
}

// consumeOTPChallenge marks a challenge as used so its code can no longer verify.
func consumeOTPChallenge(tx *gorm.DB, challengeID, clientIp string) error {
	now := time.Now()
	res := tx.Model(&models.UserOTPChallenge{}).
		Where("ID = ? AND IpAddress = ? AND ConsumedAt IS NULL", challengeID, clientIp).
		Update("ConsumedAt", &now)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// verifyOTP validates a submitted code against a stored challenge and, on
// success, marks it consumed and returns the associated user.
func verifyOTP(tx *gorm.DB, challengeID, code, clientIp string) (*models.User, error) {
	var challenge models.UserOTPChallenge
	if err := tx.Where("ID = ? AND IpAddress = ?", challengeID, clientIp).First(&challenge).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired challenge")
		}
		return nil, err
	}
	if challenge.ConsumedAt != nil {
		return nil, errors.New("challenge already used")
	}
	if time.Now().After(challenge.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}
	if challenge.Attempts >= otpMaxAttempts {
		return nil, errors.New("too many attempts")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(challenge.CodeHash), []byte(code)); err != nil {
		_ = tx.Model(&challenge).Update("Attempts", gorm.Expr("Attempts + 1")).Error
		return nil, errors.New("invalid code")
	}

	now := time.Now()
	if err := tx.Model(&challenge).Update("ConsumedAt", &now).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := tx.First(&user, challenge.UserID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// generateOTP returns a cryptographically random numeric code of n digits.
func generateOTP(n int) (string, error) {
	var b strings.Builder
	for range n {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteString(d.String())
	}
	return b.String(), nil
}
