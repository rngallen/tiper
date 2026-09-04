package integrations

import (
	"testing"

	"dfms/apps/models"
	"dfms/pkg/config"
)

func TestSessionFromRowDefaults(t *testing.T) {
	got := sessionFromRow(models.IntegrationSetting{})
	if got.IdleMinutes != 10 || got.WarnSeconds != 120 {
		t.Fatalf("empty row: %+v", got)
	}
}

func TestSessionFromRowWarnMinutes(t *testing.T) {
	row := models.IntegrationSetting{
		Config: models.JSONMap{"idleMinutes": 15, "warnMinutes": 2},
	}
	got := sessionFromRow(row)
	if got.IdleMinutes != 15 || got.WarnSeconds != 120 {
		t.Fatalf("warnMinutes 2: %+v", got)
	}
	if got.WarnMinutes() != 2 {
		t.Fatalf("WarnMinutes: %d", got.WarnMinutes())
	}
}

func TestNpgisLicenseURLRoundTrip(t *testing.T) {
	in := config.NpgisConfig{
		Enabled:    true,
		LicenseURL: "https://example.go.tz/licenses",
		BaseURL:    "https://npgisretailer.ewura.go.tz:2990/api/",
		DepotName:  "TIPER",
	}
	got := npgisFromRow(models.IntegrationSetting{Config: models.JSONMap(npgisToConfig(in))})
	if got.LicenseURL != in.LicenseURL || !got.Enabled {
		t.Fatalf("round-trip: %+v", got)
	}
}

func TestSessionToConfigRoundTrip(t *testing.T) {
	in := config.SessionConfig{IdleMinutes: 30, WarnSeconds: 180}.Clamp()
	got := sessionFromRow(models.IntegrationSetting{Config: models.JSONMap(sessionToConfig(in))})
	if got.IdleMinutes != 30 || got.WarnMinutes() != 3 {
		t.Fatalf("round-trip: %+v", got)
	}
}
