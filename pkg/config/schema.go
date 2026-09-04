package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/validation"
	"github.com/jellydator/validation/is"
)

// Config is the root bootstrap configuration (.env). Mail, SMS, Sage 200,
// attachment limits, session idle, and job schedules live in IntegrationSetting.
type Config struct {
	App AppConfig `mapstructure:"dfms"`
}

// =============================================================================
// Mail / SMS — runtime types (DB-backed via internal/integrations)
// =============================================================================

// MailConfig holds SMTP settings used by pkg/mail for OTP and workflow
// notification email. When Enabled is false the application substitutes a
// logging mailer instead of sending anything.
type MailConfig struct {
	Enabled   bool
	Host      string
	Port      int
	User      string
	Password  string
	FromName  string
	FromEmail string
	// UseTLS enables STARTTLS after a plain dial (typical for port 587).
	UseTLS bool
	// UseSSL enables implicit TLS/SSL from the first byte (typical for port
	// 465). Matches "SSL=true" in other mail clients. Port 465 implies SSL
	// even when this flag is false.
	UseSSL bool
}

// SMSConfig holds settings for the SMS gateway used to deliver OTP codes when a
// user profile includes a phone number.
type SMSConfig struct {
	Enabled  bool
	APIURL   string
	APIKey   string
	SenderID string
}

// SchedulesConfig holds robfig/cron specs (with seconds) for background jobs.
// Stored in IntegrationSetting key=schedules; edited under Settings → Schedules.
type SchedulesConfig struct {
	LogRotation   string // logs.rotation
	EwuraLicenses string // ewura.licenses
	BillingNth    string // billing.nth
	BillingTBS    string // billing.tbs
	BillingVCF    string // billing.vcf
	EwuraNpgis    string // ewura.npgis
	IloExpire     string // orders.expire — midnight ILO expiry
	NotifyOutbox  string // notify.outbox — retry durable mail/SMS
}

// AlmaConfig is the ATLAS NEO file-share root (Settings key=alma).
type AlmaConfig struct {
	FilePath string
}

// NpgisConfig is EWURA integration (Settings key=npgis): petroleum license
// register URL plus the NPGIS retailer API.
type NpgisConfig struct {
	Enabled     bool
	LicenseURL  string // petroleum license register JSON (ewura.licenses job)
	BaseURL     string
	LicenseNo   string
	APISourceID string
	DepotName   string
}

// =============================================================================
// Application Core Settings (including main database)
// =============================================================================

// AppConfig contains the core application settings: the HTTP listener and
// graceful-shutdown budget, PASETO/MFA signing keys (each at least 32 bytes),
// the main application database, CORS, and cookie security.
// Job cron specs live in IntegrationSetting (Settings → Schedules), not here.
type AppConfig struct {
	// ================== Server & Runtime ==================
	ListenAddress   string        `mapstructure:"listen_address"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	Debug           bool          `mapstructure:"debug"`

	// ================== Security ==================
	SymmetricKey string `mapstructure:"symmetric_key"`
	RefreshKey   string `mapstructure:"refresh_key"`
	MFAKey       string `mapstructure:"mfa_key"`
	// AllowInsecureHTTP sets auth/CSRF cookies without the Secure flag so the
	// application can run on plain HTTP (intranet / lab). Never enable on a host
	// reachable from the public Internet — use HTTPS (docs/windows-deploy-tls.md).
	AllowInsecureHTTP bool `mapstructure:"allow_insecure_http"`
	// ================== Database (Main App DB) ==================
	Db DbConfig `mapstructure:"db"`

	// ================== CORS & File Upload ==================
	Cors              []string `mapstructure:"cors"`
	TrustForwardedFor bool     `mapstructure:"trust_forwarded_for"`
}

// OrdersConfig is depot order policy (Settings key=orders).
type OrdersConfig struct {
	IloExpiryDays int
}

func DefaultOrders() OrdersConfig {
	return OrdersConfig{IloExpiryDays: 14}
}

func (o OrdersConfig) Clamp() OrdersConfig {
	if o.IloExpiryDays < 1 {
		o.IloExpiryDays = 14
	}
	if o.IloExpiryDays > 90 {
		o.IloExpiryDays = 90
	}
	return o
}

// SessionConfig is the live idle policy (Settings → Session).
type SessionConfig struct {
	IdleMinutes int
	WarnSeconds int
}

// ClampIdleMinutes is the server idle window (2–480, default 10).
func ClampIdleMinutes(m int) int {
	if m < 2 {
		return 10
	}
	if m > 480 {
		return 480
	}
	return m
}

// ClampIdleWarnSeconds is how long the UI countdown runs (15–600, default 120),
// always leaving at least 15s of idle before the warning starts.
func ClampIdleWarnSeconds(idleMinutes, warnSeconds int) int {
	idleSec := ClampIdleMinutes(idleMinutes) * 60
	w := warnSeconds
	if w < 15 {
		w = 120
	}
	if w > 600 {
		w = 600
	}
	max := idleSec - 15
	if max < 15 {
		max = 15
	}
	if w > max {
		return max
	}
	return w
}

// DefaultSession is the first-run idle policy written by seed
// (10 minutes idle, 2 minute warning). Operators change it under
// Settings → Session after login — not via .env.
func DefaultSession() SessionConfig {
	return SessionConfig{IdleMinutes: 10, WarnSeconds: 120}.Clamp()
}

// Clamp applies idle/warn bounds.
func (s SessionConfig) Clamp() SessionConfig {
	m := ClampIdleMinutes(s.IdleMinutes)
	return SessionConfig{
		IdleMinutes: m,
		WarnSeconds: ClampIdleWarnSeconds(m, s.WarnSeconds),
	}
}

// WarnMinutes is the operator-facing warning length (seconds rounded to minutes).
func (s SessionConfig) WarnMinutes() int {
	sec := s.Clamp().WarnSeconds
	m := (sec + 30) / 60
	if m < 1 {
		return 1
	}
	return m
}

// IdleWindow is the live refresh unused-age limit.
func (s SessionConfig) IdleWindow() time.Duration {
	return time.Duration(s.Clamp().IdleMinutes) * time.Minute
}

// CookieSecure is true when auth/CSRF cookies must only be sent over HTTPS.
// False when Debug or AllowInsecureHTTP is set (local / intranet HTTP).
func (a AppConfig) CookieSecure() bool {
	if a.Debug || a.AllowInsecureHTTP {
		return false
	}
	return true
}

// UploadsConfig is operator-managed attachment storage (Settings → Attachments).
// Fiber BodyLimit stays at ProcessBodyLimit (64 MiB) so size caps can change live
// without restart, but cannot exceed that process cap.
type UploadsConfig struct {
	Directory          string // disk root for new files (default ./uploads)
	MaxFileSizeMB      int
	MaxFilesPerRequest int
}

// =============================================================================
// Shared Database Config
// =============================================================================

// DbConfig describes a SQL Server connection. Used for the application database
// in .env (AppConfig.Db) and for Sage 200 in Settings (IntegrationSetting key=sage).
// Instance is only needed for named SQL Server instances.
type DbConfig struct {
	Host     string `mapstructure:"host"`
	Instance string `mapstructure:"instance"` // For SQL Server named instances
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	Port     string `mapstructure:"port"`
	Encrypt  bool   `mapstructure:"encrypt"`
}

// =============================================================================
// Validation
// =============================================================================

// validateConfig is the entry point — called with context from InitConfig.
// Each section has its own dedicated validator so error messages stay focused
// on the offending field path.
func validateConfig(ctx context.Context, c *Config) error {
	if c == nil {
		return fmt.Errorf("config: nil pointer")
	}
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.App, validation.By(func(v any) error {
			return validateAppConfig(ctx, v)
		})),
	)
}

func validateAppConfig(ctx context.Context, value any) error {
	app, ok := value.(AppConfig)
	if !ok {
		return fmt.Errorf("expected AppConfig")
	}

	return validation.ValidateStructWithContext(ctx, &app,
		validation.Field(&app.ListenAddress, validation.Required),
		validation.Field(&app.ShutdownTimeout, validation.Required, validation.Min(5*time.Second)),
		validation.Field(&app.SymmetricKey, validation.Required, validation.Length(32, 0)),
		validation.Field(&app.RefreshKey, validation.Required, validation.Length(32, 0)),
		validation.Field(&app.MFAKey, validation.Required, validation.Length(32, 0)),
		validation.Field(&app.Db, validation.Required, validation.By(func(v any) error {
			return validateDbConfig(ctx, v)
		})),
	)
}

func validateDbConfig(ctx context.Context, value any) error {
	db, ok := value.(DbConfig)
	if !ok {
		return fmt.Errorf("expected DbConfig")
	}

	return validation.ValidateStructWithContext(ctx, &db,
		validation.Field(&db.Host, validation.Required, is.Host),
		validation.Field(&db.User, validation.Required),
		validation.Field(&db.Password, validation.Required),
		validation.Field(&db.Name, validation.Required),
		validation.Field(&db.Port, validation.Required, is.Port),
	)
}
