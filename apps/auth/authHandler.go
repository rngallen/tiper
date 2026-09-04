package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/audit"
	"dfms/pkg/constants"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"gorm.io/gorm"
)

// authHandler manages authentication flows.
type AuthHandler struct {
	db          *gorm.DB
	rateLimiter *RateLimiter
}

var loginRateLimiter *RateLimiter

// NewAuthHandler creates an authentication handler with login brute-force
// protection (5 failures per IP per minute).
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	rl := newRateLimiter(5, time.Minute)
	loginRateLimiter = rl
	return &AuthHandler{db: db, rateLimiter: rl}
}

// StopBackgroundWorkers stops auth background goroutines (rate-limiter janitor).
func StopBackgroundWorkers() {
	if loginRateLimiter != nil {
		loginRateLimiter.Stop()
	}
}

// csrfToken returns the current CSRF token so the SPA can send X-Csrf-Token on
// unsafe requests. Visiting this GET also ensures the CSRF cookie is issued.
func (h *AuthHandler) csrfToken(c fiber.Ctx) error {
	return response.OkDetail(c, fiber.Map{
		"csrfToken": csrf.TokenFromContext(c),
	})
}

// Login authenticates a user and initiates a 2FA challenge.
func (h *AuthHandler) login(c fiber.Ctx) error {
	clientIP := audit.GetClientIP(c)

	// === Brute-force protection ===
	if !h.rateLimiter.Allowed(clientIP) {
		return response.TooManyRequests(c)
	}

	// Server-side timeout.
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	// === Parse and validate input ===
	var payload userLogin
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	type loginOutcome struct {
		mfaPending loginPendingResponse
		mfaCode    string
		failure    error
	}
	var outcome loginOutcome
	var lookedUpUser *models.User // populated even on credential failure for follow-up bookkeeping

	db := h.db.WithContext(ctx)
	userEmail := strings.ToLower(payload.Email)
	err := db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Preload("Profile").Where("Email = ?", userEmail).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				outcome.failure = constants.ErrInvalidCredentials
				return nil
			}
			return fmt.Errorf("lookup user: %w", err)
		}
		lookedUpUser = &user

		// Locked accounts must say so even on a wrong password — the holder
		// needs to ask an administrator to unlock. Unknown emails still get
		// the generic credential error above (record-not-found).
		if user.IsLocked || user.Attempts >= models.MaxFailedAttempts {
			outcome.failure = constants.ErrUserLocked
			return nil
		}

		if err := user.ComparePassword(payload.Password); err != nil {
			outcome.failure = err
			return nil
		}

		if !user.IsActive {
			outcome.failure = constants.ErrUserInactive
			return nil
		}

		if err := user.UpdateLastLogin(tx); err != nil {
			return fmt.Errorf("update last login: %w", err)
		}

		challengeID, code, err := issueOTPChallenge(tx, &user, clientIP)
		if err != nil {
			return fmt.Errorf("issue mfa otp: %w", err)
		}
		token, err := h.createOTPAccessToken(&user, challengeID, OTPDeliveryMessage(&user))
		if err != nil {
			return fmt.Errorf("issue otp access toekn : %w", err)
		}
		outcome.mfaPending = token
		outcome.mfaCode = code
		return nil

	})

	// === Context-aware error handling — branches are mutually exclusive
	// and each one terminates with a single response. ===
	if err != nil {
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return response.GatewayTimeout(c)
		case errors.Is(ctx.Err(), context.Canceled):
			return response.ClientClosedRequest(c)
		default:
			logs.Errorf("login transaction failed for %s: %v", payload.Email, err)
			return response.InternalServerError(c)
		}
	}

	// Credential failure: record the failed-attempt counter on its own
	// connection so a later rollback in the same request can't undo it.
	if outcome.failure != nil {
		msg := outcome.failure.Error()
		if lookedUpUser != nil && errors.Is(outcome.failure, constants.ErrInvalidCredentials) {
			if bErr := lookedUpUser.IncrementFailedAttempts(db); bErr != nil {
				logs.Errorf("increment failed_attempts for user %s: %v", lookedUpUser.UID, bErr)
			} else if lookedUpUser.IsLocked || lookedUpUser.Attempts >= models.MaxFailedAttempts {
				msg = constants.ErrUserLocked.Error()
			}
		}
		h.rateLimiter.RegisterFailure(clientIP)
		return response.Unauthorized(c, msg)
	}

	if lookedUpUser != nil {
		// OTP is already persisted; deliver email/SMS in the background so login
		// is not blocked on SMTP latency. Failures are logged — the user can
		// still complete MFA once the message arrives (or an admin can resend).
		u := *lookedUpUser
		code := outcome.mfaCode
		logs.GoSafe("auth.otp.deliver", func() {
			if err := deliverer.Send(context.Background(), &u, code); err != nil {
				logs.Errorf("dispatch mfa otp for user %s: %v", u.UID, err)
			}
		})
	}

	h.rateLimiter.Reset(clientIP)

	// A new login supersedes any leftover session cookies from a previous
	// visit. Those cookies would otherwise force CSRF on /mfa/verify (cookie
	// session wins over Bearer) while the SPA sends only the MFA token.
	ClearAuthCookies(c)

	entry := audit.EntryFromRequest(
		lookedUpUser.ID, types.ModuleAuth, lookedUpUser.FullName(), types.ActionLogin, clientIP, c.Get("User-Agent"),
		c.RequestID(), lookedUpUser.UID, types.UserContent, "successful login waiting for OTP verification",
	)
	_ = audit.Default.Record(c.Context(), nil, entry)

	return response.OkDetail(c, outcome.mfaPending)
}

// mfaResend issues a new OTP challenge using the current MFA bearer (JTI =
// prior challenge id). The prior challenge is consumed and a new bearer is
// returned whose JTI points at the new challenge — the client must replace its
// stored token or verify will look up the old (consumed) row.
func (h *AuthHandler) mfaResend(c fiber.Ctx) error {
	clientIP := audit.GetClientIP(c)
	if !h.rateLimiter.Allowed(clientIP) {
		return response.TooManyRequests(c)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	claims := middleware.OTPClaimsFromContext(c)
	if claims.JTI == "" || claims.Subject == "" {
		return response.Unauthorized(c, "Unauthorized")
	}

	type resendOutcome struct {
		pending loginPendingResponse
		code    string
		user    *models.User
	}
	var outcome resendOutcome

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var challenge models.UserOTPChallenge
		if err := tx.Where("ID = ? AND IpAddress = ?", claims.JTI, clientIP).First(&challenge).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return constants.ErrInvalidCredentials
			}
			return err
		}
		if challenge.ConsumedAt != nil {
			return errors.New("challenge already used")
		}
		if time.Now().Before(challenge.CreatedAt.Add(otpResendCooldown)) {
			return constants.ErrTooManyRequests
		}

		var user models.User
		if err := tx.Preload("Profile").Where("UID = ?", claims.Subject).First(&user).Error; err != nil {
			return err
		}
		if user.ID != challenge.UserID {
			return constants.ErrInvalidCredentials
		}
		if !user.IsActive {
			return constants.ErrUserInactive
		}
		if user.IsLocked {
			return constants.ErrUserLocked
		}

		// Invalidate the challenge the current bearer JTI points at.
		if err := consumeOTPChallenge(tx, challenge.ID, clientIP); err != nil {
			return fmt.Errorf("consume prior otp: %w", err)
		}

		challengeID, code, err := issueOTPChallenge(tx, &user, clientIP)
		if err != nil {
			return fmt.Errorf("issue mfa otp: %w", err)
		}
		// New bearer with JTI = new challenge id (old bearer must not be reused).
		pending, err := h.createOTPAccessToken(&user, challengeID, OTPDeliveryMessage(&user))
		if err != nil {
			return fmt.Errorf("issue otp access token: %w", err)
		}
		outcome.pending = pending
		outcome.code = code
		outcome.user = &user
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return response.GatewayTimeout(c)
		case errors.Is(ctx.Err(), context.Canceled):
			return response.ClientClosedRequest(c)
		case errors.Is(err, constants.ErrTooManyRequests):
			return response.TooManyRequests(c)
		case errors.Is(err, constants.ErrUserInactive),
			errors.Is(err, constants.ErrUserLocked),
			errors.Is(err, constants.ErrInvalidCredentials):
			return response.Unauthorized(c, err.Error())
		case err.Error() == "challenge already used":
			return response.Unauthorized(c, err.Error())
		default:
			logs.Errorf("mfa resend failed for sub=%s: %v", claims.Subject, err)
			return response.InternalServerError(c)
		}
	}

	u := *outcome.user
	code := outcome.code
	logs.GoSafe("auth.otp.resend", func() {
		if err := deliverer.Send(context.Background(), &u, code); err != nil {
			logs.Errorf("dispatch mfa otp resend for user %s: %v", u.UID, err)
		}
	})

	return response.OkDetail(c, outcome.pending)
}

// mfaVerify completes a login by validating the OTP and issuing tokens.
func (h *AuthHandler) mfaVerify(c fiber.Ctx) error {
	clientIP := audit.GetClientIP(c)
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var payload mfaVerifySchema
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	claims := middleware.OTPClaimsFromContext(c)
	var tokens *tokenResponse
	var verifiedUser *models.User
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := verifyOTP(tx, claims.JTI, payload.Code, clientIP)
		if err != nil {
			return err
		}

		if !user.IsActive {
			return constants.ErrUserInactive
		}

		if user.IsLocked {
			return constants.ErrUserLocked
		}
		verifiedUser = user
		if err := tx.Where("Subject = ?", user.UID).Delete(&models.RefreshToken{}).Error; err != nil {
			logs.Errorf("failed to delete refrest token for user : %s : %s", user.UID, err.Error())
			return fmt.Errorf("internal server error")
		}
		tokens, err = h.createTokens(tx, user, clientIP)
		return err
	})
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	if verifiedUser != nil && audit.Default != nil {
		entry := audit.EntryFromRequest(
			verifiedUser.ID, types.ModuleAuth, verifiedUser.FullName(), types.ActionLogin, clientIP, c.Get("User-Agent"),
			c.RequestID(), verifiedUser.UID, types.UserContent, "successful login (2FA)",
		)
		_ = audit.Default.Record(c.Context(), nil, entry)
	}
	// Drop any stale permission cache so role grants take effect immediately.
	if verifiedUser != nil {
		middleware.InvalidateUserPermissions(verifiedUser.UID)
	}
	SetAuthCookies(c, tokens)
	return response.OkDetail(c, tokens.Public())
}

// RefreshToken exchanges a valid refresh token for a new token pair.
func (h *AuthHandler) refreshToken(c fiber.Ctx) error {
	claims := middleware.RefreshClaimsFromContext(c)
	clientIP := audit.GetClientIP(c)

	if clientIP != claims.IpAddress {
		// Proxy / VPN / IPv4↔IPv6 often change the seen address. Idle is
		// enforced on LastSeen; a live session must still rotate.
		logs.Warnf("refresh token IP changed: token=%s request=%s sub=%s", claims.IpAddress, clientIP, claims.Subject)
	}

	var tokens *tokenResponse
	var user models.User
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// MSSQL rejects SELECT … FOR UPDATE. Rotate with an atomic delete of the
		// still-valid row so concurrent /auth/refresh calls serialize safely:
		// only one request observes RowsAffected=1.
		now, idleSince := sessionWindow()
		row, findErr := findLiveRefresh(tx, claims.Subject, now, idleSince)
		if findErr != nil {
			_ = tx.Where("Subject = ?", claims.Subject).Delete(&models.RefreshToken{}).Error
			logs.Warnf("refresh token not found, expired, or idle for sub=%s ip=%s: %v", claims.Subject, clientIP, findErr)
			return constants.ErrInvalidRefreshToken
		}
		res := tx.Where("Subject = ? AND IpAddress = ?", row.Subject, row.IpAddress).
			Delete(&models.RefreshToken{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			_ = tx.Where("Subject = ?", claims.Subject).Delete(&models.RefreshToken{}).Error
			return constants.ErrInvalidRefreshToken
		}

		// Load User to check active/locked status before issuing new tokens.
		// This is necessary to prevent issuing new tokens to a user whose account has been deactivated
		// or locked since the refresh token was issued.
		if err := tx.Where("UID = ?", claims.Subject).First(&user).Error; err != nil {
			return err
		}

		// Check user status
		if !user.IsActive || user.IsLocked {
			logs.Warnf("user account locked or inactive (sub=%s)", claims.Subject)
			return constants.ErrUserLocked
		}
		if claims.SessionVersion != user.SessionVersion {
			return constants.ErrInvalidRefreshToken
		}
		var e error
		tokens, e = h.createTokens(tx, &user, clientIP)
		return e
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, constants.ErrInvalidRefreshToken) {
			return response.Unauthorized(c, constants.ErrInvalidRefreshToken.Error())
		}
		if errors.Is(err, constants.ErrUserLocked) {
			return response.Unauthorized(c, constants.ErrUserLocked.Error())
		}
		logs.Errorf("refresh token user=%s ip=%s: %v", claims.Subject, clientIP, err)
		return response.InternalServerError(c)
	}

	if audit.Default != nil {
		entry := audit.EntryFromRequest(
			user.ID, types.ModuleAuth, user.FullName(), types.ActionRefreshToken, clientIP, c.Get("User-Agent"),
			c.RequestID(), user.UID, types.UserContent, "successful token refresh",
		)

		_ = audit.Default.Record(c.Context(), nil, entry)
	}
	SetAuthCookies(c, tokens)
	return response.OkDetail(c, tokens.Public())
}

// Logout drops the refresh token for this client IP only. Other devices stay
// signed in; use password change to revoke every session via SessionVersion.
func (h *AuthHandler) logout(c fiber.Ctx) error {
	claims := middleware.GetUserClaimsFromContext(c)
	clientIP := audit.GetClientIP(c)

	if err := h.db.WithContext(c.Context()).Where("Subject = ? AND IpAddress = ?", claims.Subject, clientIP).Delete(&models.RefreshToken{}).Error; err != nil {
		logs.Errorf("logout delete refresh token user=%s ip=%s: %v", claims.Subject, clientIP, err)
		return response.InternalServerError(c)
	}

	middleware.InvalidateUserPermissions(claims.Subject)

	if audit.Default != nil && claims.ISI != 0 {
		_ = audit.Default.Record(c.Context(), nil, audit.EntryFromRequest(
			claims.ISI, types.ModuleAuth, claims.Username, types.ActionLogout, clientIP, c.Get("User-Agent"),
			c.RequestID(), claims.Subject, types.UserContent, "logout",
		))
	}
	ClearAuthCookies(c)
	return response.OkMessage(c, "logged out successfully")
}

// ChangePassword updates the authenticated user's password, bumps
// SessionVersion (revoking every other session), and returns a fresh token
// pair so the current client stays signed in.
func (h *AuthHandler) changePassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var payload changePasswordSchema
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	userID := middleware.GetUserIDFromContext(c)
	if userID == 0 {
		return response.Unauthorized(c, "unauthorized")
	}
	clientIP := audit.GetClientIP(c)

	var (
		tokens    *tokenResponse
		bumpedUID string
		bumpedSV  uint
	)
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if err := user.ComparePassword(payload.OldPassword); err != nil {
			return constants.ErrInvalidCredentials
		}
		reused, err := models.PasswordReused(tx, user.ID, payload.NewPassword, user.Password)
		if err != nil {
			return err
		}
		if reused {
			return constants.ErrPasswordReused
		}
		oldHash := user.Password
		user.Password = payload.NewPassword
		if err := user.EncryptPassword(); err != nil {
			return err
		}
		if err := models.RecordPasswordHash(tx, user.ID, oldHash); err != nil {
			return err
		}
		if err := tx.Model(&user).
			Select("Password", "MustChangePassword", "SessionVersion").
			Updates(map[string]any{
				"Password":           user.Password,
				"MustChangePassword": false,
				"SessionVersion":     gorm.Expr("SessionVersion + 1"),
			}).Error; err != nil {
			return err
		}
		user.SessionVersion++
		user.MustChangePassword = false
		// Drop every refresh token; only the new pair below remains valid.
		if err := tx.Where("Subject = ?", user.UID).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		bumpedUID = user.UID
		bumpedSV = user.SessionVersion
		var e error
		tokens, e = h.createTokens(tx, &user, clientIP)
		return e
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Unauthorized(c, "user account not found")
		}
		if errors.Is(err, constants.ErrInvalidCredentials) {
			return response.BadRequest(c, "old password is incorrect")
		}
		if errors.Is(err, constants.ErrPasswordReused) {
			return response.BadRequest(c, err.Error())
		}
		logs.Errorf("change password user=%d: %v", userID, err)
		return response.InternalServerError(c)
	}
	// Publish the bumped version so other in-flight requests reject old tokens
	// without another DB round-trip.
	middleware.SetSessionVersion(bumpedUID, bumpedSV, middleware.SessionVersionTTL)
	if audit.Default != nil {
		entry := audit.AuditEntry(c, types.ModuleAuth, types.ActionUpdate, bumpedUID, types.UserContent, "changed password")
		_ = audit.Default.Record(c.Context(), nil, entry)
	}
	SetAuthCookies(c, tokens)
	return response.OkDetail(c, tokens.Public())
}
