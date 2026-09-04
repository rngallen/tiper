package jobs

import (
	"testing"

	"dfms/pkg/config"
)

func TestNormalizeFillsDefaults(t *testing.T) {
	norm, err := Normalize(config.SchedulesConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if norm.EwuraLicenses != DefaultSpecs.EwuraLicenses {
		t.Fatalf("ewuraLicenses=%q want %q", norm.EwuraLicenses, DefaultSpecs.EwuraLicenses)
	}
	if norm.BillingTBS != DefaultSpecs.BillingTBS {
		t.Fatalf("billingTbs=%q want %q", norm.BillingTBS, DefaultSpecs.BillingTBS)
	}
	if norm.BillingVCF != DefaultSpecs.BillingVCF {
		t.Fatalf("billingVcf=%q want %q", norm.BillingVCF, DefaultSpecs.BillingVCF)
	}
	if norm.EwuraNpgis != DefaultSpecs.EwuraNpgis {
		t.Fatalf("ewuraNpgis=%q want %q", norm.EwuraNpgis, DefaultSpecs.EwuraNpgis)
	}
	if norm.IloExpire != DefaultSpecs.IloExpire {
		t.Fatalf("iloExpire=%q want %q", norm.IloExpire, DefaultSpecs.IloExpire)
	}
	if norm.NotifyOutbox != DefaultSpecs.NotifyOutbox {
		t.Fatalf("notifyOutbox=%q want %q", norm.NotifyOutbox, DefaultSpecs.NotifyOutbox)
	}
}

func TestNormalizeRejectsBadSpec(t *testing.T) {
	_, err := Normalize(config.SchedulesConfig{EwuraLicenses: "not-a-cron"})
	if err == nil {
		t.Fatal("expected error for invalid cron")
	}
}

func TestValidateSpecCharset(t *testing.T) {
	if err := ValidateSpecCharset("0 */10 * * * *"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpecCharset("0 0 9 * * MON"); err == nil {
		t.Fatal("expected reject letters")
	}
	if err := ValidateSpecCharset("0 0 0 * * * #comment"); err == nil {
		t.Fatal("expected reject #")
	}
}

func TestValidateSpec(t *testing.T) {
	if err := ValidateSpec(DefaultSpecs.LogRotation); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpec(""); err == nil {
		t.Fatal("expected empty error")
	}
}
