package auth

import (
	"context"
	"errors"
	"regexp"
	"time"

	"dfms/pkg/constants"
	"dfms/pkg/types"

	"github.com/jellydator/validation"
	"github.com/jellydator/validation/is"
)

// loginPendingResponse is returned when a login requires a second factor.
// Token is an MFA bearer whose JTI is the OTP challenge id — verify and resend
// both look up the challenge by that JTI, so each resend returns a new Token.
type loginPendingResponse struct {
	Message           string    `json:"message"`
	Token             string    `json:"token"`
	IssuedAt          time.Time `json:"issuedAt"`
	ExpiredAt         time.Time `json:"expiredAt"`         // when the 6-digit code expires
	ResendAvailableAt time.Time `json:"resendAvailableAt"` // earliest /mfa/resend
}

// tokenResponse is returned after a successful login or refresh.
type tokenResponse struct {
	Email              string    `json:"email"`
	Name               string    `json:"name"`
	IssuedAt           time.Time `json:"issuedAt"`
	AccessToken        string    `json:"token"`
	Expiration         time.Time `json:"tokenExpiresAt"`
	RefreshToken       string    `json:"refreshToken"`
	RefreshExpiration  time.Time `json:"refreshTokenExpiresAt"`
	MustChangePassword bool      `json:"mustChangePassword"`
}

// ──────────────────────────────────────────────────────────────
//
//	Shared Validation Rules
//
// ──────────────────────────────────────────────────────────────
// nameRule is reused across firstName, lastName, role names, etc.
var nameRule = []validation.Rule{
	validation.Required,
	validation.Length(2, 60).Error(constants.Max60Chars),
}

// phoneRegex enforces strict international format WITHOUT + sign,
// no spaces, no separators, and must NOT start with zero.
// Examples of valid: 255711223344, 254111888111, 256700123456
// Total length: 7 to 15 digits (E.164 without the +)
var phoneRegex = regexp.MustCompile(`^[1-9][0-9]{6,14}$`)

// phoneNumberRule validates international phone numbers in the format you want.
// - Empty value is allowed (optional field)
// - Non-empty values must be digits only, start with 1-9, no +, no spaces, no separators
// - Recommended length: 7–15 digits
var phoneNumberRule = []validation.Rule{
	validation.Length(0, 15).Error("phone number must be 0-15 characters"),
	validation.By(func(value any) error {
		s, _ := value.(string)
		if s == "" {
			return nil
		}
		if !phoneRegex.MatchString(s) {
			return errors.New("must be a valid international phone number without + or spaces (e.g. 255711223344)")
		}
		return nil
	}),
}

// ──────────────────────────────────────────────────────────────
//
//	User Profile & Account Management
//
// ──────────────────────────────────────────────────────────────
type appearanceSettingSchema struct {
	Theme        string `json:"theme"`
	CompactMode  bool   `json:"compactMode"`
	LargeText    bool   `json:"largeText"`
	SidebarState bool   `json:"sidebarState"`
}

// Validate restricts Theme to light/dark/system; the boolean toggles are
// accepted as-is.
func (a appearanceSettingSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &a,
		validation.Field(&a.Theme, validation.Required, validation.In("light", "dark", "system")),
		validation.Field(&a.CompactMode),
		validation.Field(&a.LargeText),
		validation.Field(&a.SidebarState))
}

type tablePrefsPayload struct {
	TableID string   `json:"tableId"`
	Order   []string `json:"order"`
	Hidden  []string `json:"hidden"`
}

// Validate requires a table id and caps the column order/hidden lists at 50
// entries each to bound the stored preference blob.
func (t tablePrefsPayload) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &t,
		validation.Field(&t.TableID, validation.Required, validation.Length(1, 60)),
		validation.Field(&t.Order, validation.Length(0, 50)),
		validation.Field(&t.Hidden, validation.Length(0, 50)),
	)
}

type dashboardWidgetSchema struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Source string `json:"source"`
	Span   int    `json:"span"`
}

type dashboardPayload struct {
	Widgets []dashboardWidgetSchema `json:"widgets"`
}

func (d dashboardPayload) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &d,
		validation.Field(&d.Widgets, validation.Length(0, 16)),
	)
}

func (w dashboardWidgetSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &w,
		validation.Field(&w.ID, validation.Required, validation.Length(1, 40)),
		validation.Field(&w.Type, validation.Required, validation.In("kpi", "bar", "line", "doughnut", "pie", "table")),
		validation.Field(&w.Title, validation.Required, validation.Length(1, 80)),
		validation.Field(&w.Source, validation.Required, validation.Length(1, 40)),
		validation.Field(&w.Span, validation.Min(0), validation.Max(2)),
	)
}

// ── Login & MFA ──
type userLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate requires a well-formed email and a non-empty password; password
// strength is deliberately not checked at login.
func (u userLogin) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &u,
		validation.Field(&u.Email, validation.Required, is.EmailFormat),
		validation.Field(&u.Password, validation.Required),
	)
}

type mfaVerifySchema struct {
	Code string `json:"code"`
}

var otpCodeRegex = regexp.MustCompile(`^\d{6}$`)

// Validate requires the submitted OTP to be exactly six digits, matching the
// codes produced by generateOTP.
func (m mfaVerifySchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &m,
		validation.Field(&m.Code, validation.Required, validation.Match(otpCodeRegex).Error("must be a 6-digit code")),
	)
}

// ── Profile / user management ──

// profileInSchema represents common profile fields (phone number)
type profileInSchema struct {
	PhoneNumber string `json:"phoneNumber"`
	Title       string `json:"title"`
}

// Validate allows both fields to be empty; a non-empty phone number must be in
// E.164 form without the + (digits only, no leading zero) and the title is
// capped at 60 characters.
func (p profileInSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &p,
		validation.Field(&p.PhoneNumber, phoneNumberRule...),
		validation.Field(&p.Title, validation.Length(0, 60).Error(constants.Max60Chars)),
	)
}

// userProfileUpdateSchema used when updating personal details
type userProfileUpdateSchema struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	IsActive        bool   `json:"isActive"`
	profileInSchema `json:"profile"`
}

// Validate enforces required 2-60 character first/last names plus the embedded
// profile rules (optional E.164 phone, title length). IsActive is not
// validated here.
func (u userProfileUpdateSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &u,
		validation.Field(&u.FirstName, nameRule...),
		validation.Field(&u.LastName, nameRule...),
		validation.Field(&u.PhoneNumber, phoneNumberRule...),
		validation.Field(&u.Title, validation.Length(0, 60).Error(constants.Max60Chars)),
	)
}

type userUpdateSchema struct {
	userProfileUpdateSchema
}

// Validate includes all profile rules + boolean flag
func (u userUpdateSchema) Validate(ctx context.Context) error {
	return u.userProfileUpdateSchema.Validate(ctx)
}

// ──────────────────────────────────────────────────────────────
//  User Registration / Creation (Admin)
// ──────────────────────────────────────────────────────────────

// userCreateSchema used when creating a new user (admin only)
type userCreateSchema struct {
	Email string `json:"email"`
	userUpdateSchema
}

// Validate ensures email is valid and all nested rules pass
func (u userCreateSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &u,
		validation.Field(&u.Email, validation.Required, is.EmailFormat, validation.Length(6, 60).Error(constants.Max60Chars)),
	); err != nil {
		return err
	}
	return u.userUpdateSchema.Validate(ctx)
}

// ──────────────────────────────────────────────────────────────
//
//	Role & Permission Management
//
// ──────────────────────────────────────────────────────────────
// replaceRoles replaces all roles for a user (removes old, adds new)
type replaceRoles struct {
	RoleIDs []uint `json:"rolesIds"`
}

// Validate imposes no constraints on RoleIDs — an empty list is legal and
// strips every role from the user; unknown ids are caught by the handler.
func (r replaceRoles) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.RoleIDs, validation.Length(0, 50)),
	)
}

// rolesUpdate used when updating role description and family.
type rolesUpdate struct {
	Description string `json:"description"`
	Category    string `json:"category"`
}

func (r *rolesUpdate) Sanitize() {
	r.Category = string(types.NormalizeRoleFamily(r.Category))
}

// Validate requires a description of 2-255 characters; the role name itself is
// immutable after creation and therefore absent here.
func (r rolesUpdate) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Description, validation.Required, validation.Length(2, 255).Error(constants.Max255Chars)),
		validation.Field(&r.Category, validation.Required, validation.In(
			string(types.RoleSystem), string(types.RoleTerminal),
			string(types.RoleGantry), string(types.RoleFinance),
			string(types.RoleBilling),
		)),
	)
}

// rolesCreate used when creating a new role
type rolesCreate struct {
	Name string `json:"name"`
	rolesUpdate
}

func (r *rolesCreate) Sanitize() { r.rolesUpdate.Sanitize() }

// Validate requires an alphanumeric role name of 2-60 characters and applies
// the embedded rolesUpdate description rules.
func (r rolesCreate) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Required, is.Alphanumeric, validation.Length(2, 60).Error(constants.Max60Chars)),
		validation.Field(&r.rolesUpdate),
	)
}

// ──────────────────────────────────────────────────────────────
//  Permission Assignment
// ──────────────────────────────────────────────────────────────

// replacePermissions replaces all permissions on a role (empty clears them).
type replacePermissions struct {
	PermissionIDs []uint `json:"permIds"`
}

// Validate accepts any PermissionIDs value, including an empty list, which
// revokes all permissions from the role.
func (r replacePermissions) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.PermissionIDs, validation.Length(0, 200)),
	)
}

// ── Password management ──
func isValidPassword(val any) error {
	pass, ok := val.(string)
	if !ok {
		return validation.NewError("password_type", "password must be a string")
	}
	if len(pass) < 8 {
		return validation.NewError("password_length", "password must be at least 8 characters")
	}
	if !regexp.MustCompile(`[a-zA-Z]`).MatchString(pass) {
		return validation.NewError("password_letter", "password must contain at least one letter")
	}
	if !regexp.MustCompile(`[0-9]`).MatchString(pass) {
		return validation.NewError("password_number", "password must contain at least one number")
	}
	if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};:'",.<>?]`).MatchString(pass) {
		return validation.NewError("password_special", "password must contain at least one special character")
	}
	return nil
}

func equalPassword(expected string) validation.RuleFunc {
	return func(value any) error {
		s, _ := value.(string)
		if s != expected {
			return validation.NewError("password_mismatch", "passwords do not match")
		}
		return nil
	}
}

func differentFromOld(old string) validation.RuleFunc {
	return func(value any) error {
		s, _ := value.(string)
		if s != "" && s == old {
			return validation.NewError("password_reuse", "new password must be different from the old password")
		}
		return nil
	}
}

// changePasswordSchema used for password update endpoint
type changePasswordSchema struct {
	OldPassword     string `json:"oldPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

// Validate enforces the password policy on the new password (8-150 chars with
// at least one letter, digit and special character), requires it to differ
// from the old password, and requires ConfirmPassword to match it exactly.
func (c changePasswordSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &c,
		validation.Field(&c.OldPassword, validation.Required.Error("old password is required")),
		validation.Field(&c.NewPassword,
			validation.Required.Error("new password is required"),
			validation.Length(8, 150).Error("new password must be between 8 and 150 characters"),
			validation.By(isValidPassword),
			validation.By(differentFromOld(c.OldPassword)),
		),
		validation.Field(&c.ConfirmPassword,
			validation.Required.Error("confirm password is required"),
			validation.By(equalPassword(c.NewPassword)),
		),
	)
}

// ── Titles ──

type titleCreateSchema struct {
	Name string `json:"name"`
}

// Validate requires a title name of 1-60 characters; uniqueness is enforced by
// the database index, not here.
func (t titleCreateSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &t,
		validation.Field(&t.Name, validation.Required, validation.Length(1, 60).Error(constants.Max60Chars)),
	)
}
