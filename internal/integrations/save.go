package integrations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/config"
	"dfms/pkg/db"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"gorm.io/gorm"
)

// ClientError is a validation / client-fixable failure from an integration save.
// Handlers may surface Error() to the API; other errors must be logged and hidden.
type ClientError struct {
	Msg string
}

func (e *ClientError) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

// SecretPatch controls write-only secret fields on update.
// Nil = keep existing; non-nil empty = clear; non-nil value = set.
type SecretPatch struct {
	Password *string
	APIKey   *string
}

// SaveMail persists mail settings, seals secrets, reloads memory, and applies runtime.
func (s *Store) SaveMail(cfg config.MailConfig, secrets SecretPatch) error {
	existing := s.Mail()
	if secrets.Password != nil {
		cfg.Password = strings.TrimSpace(*secrets.Password)
	} else {
		cfg.Password = existing.Password
	}
	cfg.FromName = productFromName(cfg.FromName)
	return s.persist(types.KeyMail, mailToConfig(cfg))
}

// SaveSMS persists SMS settings.
func (s *Store) SaveSMS(cfg config.SMSConfig, secrets SecretPatch) error {
	existing := s.SMS()
	if secrets.APIKey != nil {
		cfg.APIKey = strings.TrimSpace(*secrets.APIKey)
	} else {
		cfg.APIKey = existing.APIKey
	}
	return s.persist(types.KeySMS, smsToConfig(cfg))
}

// SaveSchedules persists cron specs (non-secret) and reschedules live jobs.
func (s *Store) SaveSchedules(cfg config.SchedulesConfig) error {
	norm, err := jobs.Normalize(cfg)
	if err != nil {
		return &ClientError{Msg: err.Error()}
	}
	if err := s.persist(types.KeySchedules, schedulesToConfig(norm)); err != nil {
		return err
	}
	return s.ApplySchedules()
}

// SaveSage persists Sage 200 connection settings and reconnects without restart.
// Settings are stored even when the ping fails so operators can fix host/password
// from the UI; the live handle is disconnected on failure.
func (s *Store) SaveSage(cfg config.DbConfig, secrets SecretPatch) error {
	existing := s.Sage()
	if secrets.Password != nil {
		cfg.Password = strings.TrimSpace(*secrets.Password)
	} else {
		cfg.Password = existing.Password
	}
	if err := s.persist(types.KeySage, sageToConfig(cfg)); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := db.ApplySageConfig(ctx, s.Sage()); err != nil {
		return &ClientError{Msg: "saved, but could not connect: " + err.Error()}
	}
	return nil
}

// SaveSession persists idle timeout and applies it live (no restart).
func (s *Store) SaveSession(cfg config.SessionConfig) error {
	return s.persist(types.KeySession, sessionToConfig(cfg.Clamp()))
}

// SaveNpgis persists EWURA license-register and NPGIS retailer settings.
func (s *Store) SaveNpgis(cfg config.NpgisConfig) error {
	return s.persist(types.KeyNpgis, npgisToConfig(cfg))
}

// SaveUploads persists attachment directory and caps and applies them live
// (within the 64 MiB process cap).
func (s *Store) SaveUploads(cfg config.UploadsConfig) error {
	cfg = attachment.ClampUploads(cfg)
	if err := attachment.EnsureDir(cfg.Directory); err != nil {
		return fmt.Errorf("attachment directory: %w", err)
	}
	if err := s.persist(types.KeyUploads, uploadsToConfig(cfg)); err != nil {
		return err
	}
	attachment.ApplyLimits(s.Uploads())
	return nil
}

func (s *Store) persist(key string, plain map[string]any) error {
	// Shared by every IntegrationSetting key — Config must be JSONMap
	// (driver.Valuer), never a raw map[string]any, or MSSQL rejects the bind.
	if s == nil || s.db == nil {
		return fmt.Errorf("integrations store not initialized")
	}
	sealed, err := sealConfigSecrets(key, plain, s.keyMaterial)
	if err != nil {
		return err
	}
	cfg := models.JSONMap(sealed)

	var existing models.IntegrationSetting
	err = s.db.Where("[Key] = ?", key).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		row := models.IntegrationSetting{
			Key:         key,
			ContentType: types.IntegrationSettingContent,
			Config:      cfg,
		}
		if err := s.db.Create(&row).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		// Assign JSONMap (driver.Valuer) — never pass map[string]any to Update.
		if err := s.db.Model(&existing).Update("Config", cfg).Error; err != nil {
			return err
		}
	}
	if err := s.Reload(); err != nil {
		return err
	}
	s.ApplyRuntime()
	return nil
}
