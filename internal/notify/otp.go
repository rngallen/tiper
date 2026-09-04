package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/pkg/logs"
	"dfms/pkg/sms"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// OTPDeliverer sends login verification codes. Email is always used; when the
// user profile has a phone number an SMS is sent as well.
//
// When Mail/SMS integrations are empty or disabled, pkg/mail and pkg/sms use
// logging backends that do not deliver. In that case the plaintext OTP is
// written to the application log so bootstrap / first login can still complete
// (same idea as auth.LoggingDeliverer).
type OTPDeliverer struct {
	db *gorm.DB
}

// NewOTPDeliverer constructs an OTPDeliverer. db may be nil (letterhead falls back).
func NewOTPDeliverer(db *gorm.DB) *OTPDeliverer { return &OTPDeliverer{db: db} }

// Send implements auth.Deliverer. Email and SMS are independent — a failure on
// one channel does not skip the other. Returns a joined error if either fails.
func (d *OTPDeliverer) Send(ctx context.Context, user *models.User, code string) error {
	if user == nil {
		return fmt.Errorf("otp: user required")
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		return fmt.Errorf("otp: user email required")
	}

	var errs []error
	if err := d.sendEmail(ctx, user, code); err != nil {
		errs = append(errs, fmt.Errorf("otp email: %w", err))
	}

	phone := strings.TrimSpace(user.Profile.PhoneNumber)
	if phone != "" {
		if err := sms.Send(ctx, phone, otpSMSBody(loadBrand(ctx, d.db), code)); err != nil {
			errs = append(errs, fmt.Errorf("otp sms: %w", err))
		}
	}

	if !otpDeliveryConfigured(phone != "") {
		logs.Warnf("[MFA] OTP for %s: %s (mail/SMS not configured — code logged for operators)", email, code)
	}
	return errors.Join(errs...)
}

// otpDeliveryConfigured reports whether a real outbound channel will carry the
// code: SMTP enabled, or SMS enabled with a phone on the profile.
func otpDeliveryConfigured(hasPhone bool) bool {
	if integrations.Default == nil {
		return false
	}
	if integrations.Default.Mail().Enabled {
		return true
	}
	return hasPhone && integrations.Default.SMS().Enabled
}

func (d *OTPDeliverer) sendEmail(ctx context.Context, user *models.User, code string) error {
	email := strings.TrimSpace(user.Email)
	name := strings.TrimSpace(user.FullName())
	if name == "" {
		name = email
	}
	brand := loadBrand(ctx, d.db)
	subject := "Your TIPER DFMS verification code"
	body := renderCorporateEmail(brand, "Verification code", types.EmailBadgeInfo,
		fmt.Sprintf("Hello <strong>%s</strong>,", htmlEscape(name)),
		"Use the following one-time code to complete your sign in. It expires in 5 minutes.",
		"Verification",
		[]kv{{Label: "One-time code", Value: code, Highlight: true, Secret: true}},
		"",
		"If you did not request this code, ignore this email and contact your system administrator.",
		"Continue sign-in", brand.link("/login"),
	)
	return sendBranded(ctx, []string{email}, subject, body)
}

func otpSMSBody(brand EmailBrand, code string) string {
	return fmt.Sprintf("%s: Your verification code is %s. It expires in 5 minutes. Please do not share your OTP ", brand.smsName(), code)
}
