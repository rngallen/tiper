package integrations

import (
	"errors"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/config"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

func defaultSessionConfig() map[string]any {
	return sessionToConfig(config.DefaultSession())
}

func defaultSchedulesConfig() map[string]any {
	d := jobs.DefaultSpecs
	return map[string]any{
		"logRotation":   d.LogRotation,
		"ewuraLicenses": d.EwuraLicenses,
		"billingNth":    d.BillingNth,
		"billingTbs":    d.BillingTBS,
		"billingVcf":    d.BillingVCF,
		"ewuraNpgis":    d.EwuraNpgis,
		"iloExpire":     d.IloExpire,
		"notifyOutbox":  d.NotifyOutbox,
	}
}

const (
	defaultProductFromName = "TIPER DFMS"
	defaultMailHost        = "smtp.gmail.com"
	defaultMailPort        = 465
	defaultMailUser        = "tipergantry@gmail.com"
	defaultSMSAPIURL       = "https://api.notify.africa/api/v1/api/messages/send"
	defaultSMSSenderID     = "563"
	defaultSageHost        = "173.249.63.248"
	defaultSagePort        = "1343"
	defaultSageUser        = "sa"
	defaultSageName        = "TIPER LIMITED"
	defaultNpgisLicenseURL = "https://www.ewura.go.tz/licensees/fetch-licensees/Petroleum"
	defaultNpgisLicenseNo  = "PSBL-2018-003"
	defaultNpgisAPISource  = "TIPER_DEPOT"
)

// ensureRows creates IntegrationSetting rows for any missing keys so Settings
// UI always has a row to edit.
func ensureRows(db *gorm.DB) error {
	seeds := []struct {
		key    string
		config map[string]any
	}{
		{types.KeyMail, map[string]any{
			"fromName":  defaultProductFromName,
			"host":      defaultMailHost,
			"port":      defaultMailPort,
			"user":      defaultMailUser,
			"fromEmail": defaultMailUser,
			"useSSL":    true,
			"useTLS":    false,
		}},
		{types.KeySMS, map[string]any{
			"apiUrl":   defaultSMSAPIURL,
			"senderId": defaultSMSSenderID,
		}},
		{types.KeySchedules, defaultSchedulesConfig()},
		{types.KeySage, map[string]any{
			"host": defaultSageHost,
			"port": defaultSagePort,
			"user": defaultSageUser,
			"name": defaultSageName,
		}},
		{types.KeyUploads, map[string]any{"directory": "./uploads", "maxFileSizeMB": 10, "maxFilesPerRequest": 5}},
		{types.KeySession, defaultSessionConfig()},
		{types.KeyAlma, map[string]any{"filePath": "./exchange"}},
		{types.KeyNpgis, map[string]any{
			"enabled":     false,
			"baseUrl":     "https://npgisretailer.ewura.go.tz:2990/api/",
			"licenseUrl":  defaultNpgisLicenseURL,
			"licenseNo":   defaultNpgisLicenseNo,
			"apiSourceId": defaultNpgisAPISource,
			"depotName":   "TIPER",
		}},
		{types.KeyPrecision, defaultPrecisionConfig()},
		{types.KeyOrders, defaultOrdersConfig()},
	}
	for _, seed := range seeds {
		var count int64
		if err := db.Model(&models.IntegrationSetting{}).
			Where("[Key] = ?", seed.key).
			Count(&count).Error; err != nil {
			return fmt.Errorf("count integration setting %s: %w", seed.key, err)
		}
		if count > 0 {
			continue
		}
		cfg := seed.config
		if cfg == nil {
			cfg = map[string]any{}
		}
		row := models.IntegrationSetting{
			Key:         seed.key,
			ContentType: types.IntegrationSettingContent,
			Config:      models.JSONMap(cfg),
		}
		if err := db.Create(&row).Error; err != nil {
			return fmt.Errorf("create integration setting %s: %w", seed.key, err)
		}
	}
	return nil
}

// productFromName is the email From display name (and SMS body prefix).
func productFromName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultProductFromName
	}
	return s
}

// rewriteRetiredBrandNames updates stored mail fromName that still uses a previous product name.
func rewriteRetiredBrandNames(db *gorm.DB) error {
	return rewriteConfigString(db, types.KeyMail, "fromName", productFromName)
}

func rewriteConfigString(db *gorm.DB, key, field string, next func(string) string) error {
	var row models.IntegrationSetting
	err := db.Where("[Key] = ?", key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load %s for brand rewrite: %w", key, err)
	}
	if row.Config == nil {
		row.Config = models.JSONMap{}
	}
	old := row.ConfigString(field)
	want := next(old)
	if old == want {
		return nil
	}
	row.Config[field] = want
	if err := db.Model(&row).Update("Config", row.Config).Error; err != nil {
		return fmt.Errorf("rewrite %s.%s: %w", key, field, err)
	}
	return nil
}
