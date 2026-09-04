package models

import (
	"strings"
	"time"

	"dfms/pkg/types"
)

// IntegrationSetting stores operator-managed config (mail, sms, sage, uploads, session, schedules, alma, npgis, precision, orders).
// Secrets in Config are AES-GCM sealed (prefix enc:v1:). Non-secrets are plain strings/bools/ints.
type IntegrationSetting struct {
	Key         string            `gorm:"primaryKey;type:varchar(40);not null;check:[Key] IN ('mail','sms','schedules','sage','uploads','session','alma','npgis','precision','orders')" json:"key"`
	ContentType types.ContentType `gorm:"default:9;not null;check:ContentType=9" json:"-"`
	Config      JSONMap           `gorm:"type:nvarchar(max)" json:"-"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	CreatedAt   time.Time         `json:"-"`
}

// ConfigString returns a trimmed string from Config, or "".
func (i *IntegrationSetting) ConfigString(key string) string {
	if i == nil || i.Config == nil {
		return ""
	}
	v, ok := i.Config[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// ConfigBool returns a bool from Config.
func (i *IntegrationSetting) ConfigBool(key string) bool {
	if i == nil || i.Config == nil {
		return false
	}
	v, ok := i.Config[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		s := strings.TrimSpace(strings.ToLower(b))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

// ConfigInt returns an int from Config.
func (i *IntegrationSetting) ConfigInt(key string) int {
	if i == nil || i.Config == nil {
		return 0
	}
	v, ok := i.Config[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
