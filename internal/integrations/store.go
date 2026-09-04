package integrations

import (
	"fmt"
	"strings"
	"sync"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/config"
	"dfms/pkg/logs"
	"dfms/pkg/mail"
	"dfms/pkg/precision"
	"dfms/pkg/sms"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"gorm.io/gorm"
)

// RuntimeHooks wires job reschedule without import cycles.
type RuntimeHooks struct {
	ApplySchedules func(config.SchedulesConfig) error
}

var runtimeHooks RuntimeHooks

// SetApplySchedulesHook registers the schedule applier.
func SetApplySchedulesHook(fn func(config.SchedulesConfig) error) {
	runtimeHooks.ApplySchedules = fn
}

// Store holds integration settings loaded from the database with decrypted
// secrets kept in memory for runtime consumers.
type Store struct {
	mu          sync.RWMutex
	db          *gorm.DB
	keyMaterial string
	mail        config.MailConfig
	sms         config.SMSConfig
	schedules   config.SchedulesConfig
	sage        config.DbConfig
	uploads     config.UploadsConfig
	session     config.SessionConfig
	alma        config.AlmaConfig
	npgis       config.NpgisConfig
	precision   precision.Settings
	orders      config.OrdersConfig
}

// Default is the process-wide integration settings store populated by Bootstrap.
var Default *Store

// Bootstrap creates Default, ensures IntegrationSetting rows exist, reloads
// from the database, and applies runtime mail/sms hooks.
func Bootstrap(db *gorm.DB) error {
	Default = &Store{
		db:          db,
		keyMaterial: config.Conf.App.SymmetricKey,
	}
	if err := ensureRows(db); err != nil {
		return err
	}
	if err := rewriteRetiredBrandNames(db); err != nil {
		return err
	}
	if err := Default.Reload(); err != nil {
		return err
	}
	Default.ApplyRuntime()
	return nil
}

// Reload loads all integration settings from the database and decrypts secrets
// into the in-memory config structs.
func (s *Store) Reload() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("integrations store not initialized")
	}
	var rows []models.IntegrationSetting
	if err := s.db.Find(&rows).Error; err != nil {
		return fmt.Errorf("load integration settings: %w", err)
	}
	byKey := make(map[string]models.IntegrationSetting, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}

	mailCfg, err := mailFromRow(byKey[types.KeyMail], s.keyMaterial)
	if err != nil {
		return err
	}
	mailCfg.FromName = productFromName(mailCfg.FromName)
	smsCfg, err := smsFromRow(byKey[types.KeySMS], s.keyMaterial)
	if err != nil {
		return err
	}
	schedCfg := schedulesFromRow(byKey[types.KeySchedules])
	sageCfg, err := sageFromRow(byKey[types.KeySage], s.keyMaterial)
	if err != nil {
		return err
	}
	uploadsCfg := uploadsFromRow(byKey[types.KeyUploads])
	sessionCfg := sessionFromRow(byKey[types.KeySession])
	almaCfg := almaFromRow(byKey[types.KeyAlma])
	npgisCfg := npgisFromRow(byKey[types.KeyNpgis])
	precCfg := precisionFromRow(byKey[types.KeyPrecision])
	ordersCfg := ordersFromRow(byKey[types.KeyOrders])

	s.mu.Lock()
	s.mail = mailCfg
	s.sms = smsCfg
	s.schedules = schedCfg
	s.sage = sageCfg
	s.uploads = uploadsCfg
	s.session = sessionCfg
	s.alma = almaCfg
	s.npgis = npgisCfg
	s.precision = precCfg
	s.orders = ordersCfg
	s.mu.Unlock()
	return nil
}

// ApplyRuntime reinitializes mail/sms deliverers and attachment limits.
func (s *Store) ApplyRuntime() {
	if s == nil {
		return
	}
	s.mu.RLock()
	mailCfg := s.mail
	smsCfg := s.sms
	uploadsCfg := s.uploads
	s.mu.RUnlock()

	mail.Init(mailCfg)
	sms.Init(smsCfg)
	attachment.ApplyLimits(uploadsCfg)
}

// ApplySchedules pushes the in-memory cron specs to the jobs manager (if hooked).
func (s *Store) ApplySchedules() error {
	if s == nil {
		return nil
	}
	if runtimeHooks.ApplySchedules == nil {
		return nil
	}
	return runtimeHooks.ApplySchedules(s.Schedules())
}

// Mail returns a thread-safe copy of the mail config.
func (s *Store) Mail() config.MailConfig {
	if s == nil {
		return config.MailConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mail
}

// SMS returns a thread-safe copy of the SMS config.
func (s *Store) SMS() config.SMSConfig {
	if s == nil {
		return config.SMSConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sms
}

// Schedules returns a thread-safe copy of job cron specs.
func (s *Store) Schedules() config.SchedulesConfig {
	if s == nil {
		return config.SchedulesConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.schedules
}

// Sage returns Sage 200 connection settings (password in memory only).
func (s *Store) Sage() config.DbConfig {
	if s == nil {
		return config.DbConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sage
}

// Uploads returns live attachment directory and size/count caps.
func (s *Store) Uploads() config.UploadsConfig {
	if s == nil {
		return config.UploadsConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.uploads
}

// Session returns the live idle policy (clamped).
func (s *Store) Session() config.SessionConfig {
	if s == nil {
		return config.DefaultSession()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session.IdleMinutes == 0 {
		return config.DefaultSession()
	}
	return s.session.Clamp()
}

func (s *Store) Alma() config.AlmaConfig {
	if s == nil {
		return config.AlmaConfig{FilePath: "./exchange"}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alma
}

func (s *Store) Npgis() config.NpgisConfig {
	if s == nil {
		return config.NpgisConfig{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.npgis
}

// LiveSession is the process-wide idle policy. Falls back to the seeded
// default when the store is not bootstrapped yet (tests, early auth).
func LiveSession() config.SessionConfig {
	if Default == nil {
		return config.DefaultSession()
	}
	return Default.Session()
}

func schedulesFromRow(row models.IntegrationSetting) config.SchedulesConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	raw := config.SchedulesConfig{
		LogRotation:   setting.ConfigString("logRotation"),
		EwuraLicenses: setting.ConfigString("ewuraLicenses"),
		BillingNth:    setting.ConfigString("billingNth"),
		BillingTBS:    setting.ConfigString("billingTbs"),
		BillingVCF:    setting.ConfigString("billingVcf"),
		EwuraNpgis:    setting.ConfigString("ewuraNpgis"),
		IloExpire:     setting.ConfigString("iloExpire"),
		NotifyOutbox:  setting.ConfigString("notifyOutbox"),
	}
	norm, err := jobs.Normalize(raw)
	if err != nil {
		logs.Warnf("schedules config invalid, using defaults: %v", err)
		return jobs.DefaultSpecs
	}
	return norm
}

func schedulesToConfig(cfg config.SchedulesConfig) map[string]any {
	return map[string]any{
		"logRotation":   cfg.LogRotation,
		"ewuraLicenses": cfg.EwuraLicenses,
		"billingNth":    cfg.BillingNth,
		"billingTbs":    cfg.BillingTBS,
		"billingVcf":    cfg.BillingVCF,
		"ewuraNpgis":    cfg.EwuraNpgis,
		"iloExpire":     cfg.IloExpire,
		"notifyOutbox":  cfg.NotifyOutbox,
	}
}

func almaFromRow(row models.IntegrationSetting) config.AlmaConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	path := setting.ConfigString("filePath")
	if path == "" {
		path = "./exchange"
	}
	return config.AlmaConfig{FilePath: path}
}

func npgisFromRow(row models.IntegrationSetting) config.NpgisConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	return config.NpgisConfig{
		Enabled:     setting.ConfigBool("enabled"),
		LicenseURL:  setting.ConfigString("licenseUrl"),
		BaseURL:     setting.ConfigString("baseUrl"),
		LicenseNo:   setting.ConfigString("licenseNo"),
		APISourceID: setting.ConfigString("apiSourceId"),
		DepotName:   firstNonEmpty(setting.ConfigString("depotName"), "TIPER"),
	}
}

func npgisToConfig(cfg config.NpgisConfig) map[string]any {
	depot := strings.TrimSpace(cfg.DepotName)
	if depot == "" {
		depot = "TIPER"
	}
	return map[string]any{
		"enabled":     cfg.Enabled,
		"licenseUrl":  strings.TrimSpace(cfg.LicenseURL),
		"baseUrl":     strings.TrimSpace(cfg.BaseURL),
		"licenseNo":   strings.TrimSpace(cfg.LicenseNo),
		"apiSourceId": strings.TrimSpace(cfg.APISourceID),
		"depotName":   depot,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func mailFromRow(row models.IntegrationSetting, keyMaterial string) (config.MailConfig, error) {
	cfg, err := openConfigSecrets(types.KeyMail, map[string]any(row.Config), keyMaterial)
	if err != nil {
		return config.MailConfig{}, err
	}
	setting := &models.IntegrationSetting{Config: models.JSONMap(cfg)}
	return config.MailConfig{
		Enabled:   setting.ConfigBool("enabled"),
		Host:      setting.ConfigString("host"),
		Port:      setting.ConfigInt("port"),
		User:      setting.ConfigString("user"),
		Password:  setting.ConfigString("password"),
		FromName:  setting.ConfigString("fromName"),
		FromEmail: setting.ConfigString("fromEmail"),
		UseTLS:    setting.ConfigBool("useTLS"),
		UseSSL:    setting.ConfigBool("useSSL"),
	}, nil
}

func smsFromRow(row models.IntegrationSetting, keyMaterial string) (config.SMSConfig, error) {
	cfg, err := openConfigSecrets(types.KeySMS, map[string]any(row.Config), keyMaterial)
	if err != nil {
		return config.SMSConfig{}, err
	}
	setting := &models.IntegrationSetting{Config: models.JSONMap(cfg)}
	return config.SMSConfig{
		Enabled:  setting.ConfigBool("enabled"),
		APIURL:   setting.ConfigString("apiUrl"),
		APIKey:   setting.ConfigString("apiKey"),
		SenderID: setting.ConfigString("senderId"),
	}, nil
}

func mailToConfig(cfg config.MailConfig) map[string]any {
	return map[string]any{
		"enabled":   cfg.Enabled,
		"host":      cfg.Host,
		"port":      cfg.Port,
		"user":      cfg.User,
		"password":  cfg.Password,
		"fromName":  cfg.FromName,
		"fromEmail": cfg.FromEmail,
		"useTLS":    cfg.UseTLS,
		"useSSL":    cfg.UseSSL,
	}
}

func smsToConfig(cfg config.SMSConfig) map[string]any {
	return map[string]any{
		"enabled":  cfg.Enabled,
		"apiUrl":   cfg.APIURL,
		"apiKey":   cfg.APIKey,
		"senderId": cfg.SenderID,
	}
}

func sageFromRow(row models.IntegrationSetting, keyMaterial string) (config.DbConfig, error) {
	cfg, err := openConfigSecrets(types.KeySage, map[string]any(row.Config), keyMaterial)
	if err != nil {
		return config.DbConfig{}, err
	}
	setting := &models.IntegrationSetting{Config: models.JSONMap(cfg)}
	return config.DbConfig{
		Host:     setting.ConfigString("host"),
		Instance: setting.ConfigString("instance"),
		User:     setting.ConfigString("user"),
		Password: setting.ConfigString("password"),
		Name:     setting.ConfigString("name"),
		Port:     setting.ConfigString("port"),
		Encrypt:  setting.ConfigBool("encrypt"),
	}, nil
}

func sageToConfig(cfg config.DbConfig) map[string]any {
	return map[string]any{
		"host":     cfg.Host,
		"instance": cfg.Instance,
		"user":     cfg.User,
		"password": cfg.Password,
		"name":     cfg.Name,
		"port":     cfg.Port,
		"encrypt":  cfg.Encrypt,
	}
}

func uploadsFromRow(row models.IntegrationSetting) config.UploadsConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	return attachment.ClampUploads(config.UploadsConfig{
		Directory:          setting.ConfigString("directory"),
		MaxFileSizeMB:      setting.ConfigInt("maxFileSizeMB"),
		MaxFilesPerRequest: setting.ConfigInt("maxFilesPerRequest"),
	})
}

func uploadsToConfig(cfg config.UploadsConfig) map[string]any {
	cfg = attachment.ClampUploads(cfg)
	return map[string]any{
		"directory":          cfg.Directory,
		"maxFileSizeMB":      cfg.MaxFileSizeMB,
		"maxFilesPerRequest": cfg.MaxFilesPerRequest,
	}
}

func sessionFromRow(row models.IntegrationSetting) config.SessionConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	idle := setting.ConfigInt("idleMinutes")
	warnMin := setting.ConfigInt("warnMinutes")
	warnSec := setting.ConfigInt("warnSeconds")
	if warnMin > 0 {
		warnSec = warnMin * 60
	}
	if idle == 0 && warnMin == 0 && warnSec == 0 {
		return config.DefaultSession()
	}
	return config.SessionConfig{IdleMinutes: idle, WarnSeconds: warnSec}.Clamp()
}

func sessionToConfig(cfg config.SessionConfig) map[string]any {
	c := cfg.Clamp()
	return map[string]any{
		"idleMinutes": c.IdleMinutes,
		"warnMinutes": c.WarnMinutes(),
		"warnSeconds": c.WarnSeconds,
	}
}
