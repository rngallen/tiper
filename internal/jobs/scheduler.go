package jobs

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"dfms/pkg/config"

	"github.com/robfig/cron/v3"
)

const (
	LogRotation      = "logs.rotation"
	EwuraLicenseSync = "ewura.licenses"
	EwuraNpgis       = "ewura.npgis"
	BillingNth       = "billing.nth"
	BillingTBS       = "billing.tbs"
	BillingVCF       = "billing.vcf"
	IloExpire        = "orders.expire"
	NotifyOutbox     = "notify.outbox"
)

var DefaultSpecs = config.SchedulesConfig{
	LogRotation:   "0 0 0 * * *",
	EwuraLicenses: "0 0 2 * * *",
	BillingNth:    "0 30 2 * * *",
	BillingTBS:    "0 15 0 * * *",
	BillingVCF:    "0 30 3 * * *",
	EwuraNpgis:    "0 */1 * * * *",
	IloExpire:     "0 5 0 * * *",
	NotifyOutbox:  "0 */1 * * * *",
}

var cronParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func ValidateSpecCharset(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return fmt.Errorf("cron spec is required")
	}
	for _, r := range spec {
		switch {
		case unicode.IsDigit(r), r == ' ', r == '*', r == '/', r == '-', r == ',':
			continue
		default:
			return fmt.Errorf("invalid character %q (only digits, spaces, *, /, -, , allowed)", r)
		}
	}
	return nil
}

func ValidateSpec(spec string) error {
	spec = strings.TrimSpace(spec)
	if err := ValidateSpecCharset(spec); err != nil {
		return err
	}
	if _, err := cronParser.Parse(spec); err != nil {
		return fmt.Errorf("invalid cron spec %q: %w", spec, err)
	}
	return nil
}

func Normalize(in config.SchedulesConfig) (config.SchedulesConfig, error) {
	out := in
	set := func(dst *string, def string) {
		*dst = strings.TrimSpace(*dst)
		if *dst == "" {
			*dst = def
		}
	}
	set(&out.LogRotation, DefaultSpecs.LogRotation)
	set(&out.EwuraLicenses, DefaultSpecs.EwuraLicenses)
	set(&out.BillingNth, DefaultSpecs.BillingNth)
	set(&out.BillingTBS, DefaultSpecs.BillingTBS)
	set(&out.BillingVCF, DefaultSpecs.BillingVCF)
	set(&out.EwuraNpgis, DefaultSpecs.EwuraNpgis)
	set(&out.IloExpire, DefaultSpecs.IloExpire)
	set(&out.NotifyOutbox, DefaultSpecs.NotifyOutbox)
	for name, spec := range map[string]string{
		LogRotation:      out.LogRotation,
		EwuraLicenseSync: out.EwuraLicenses,
		BillingNth:       out.BillingNth,
		BillingTBS:       out.BillingTBS,
		BillingVCF:       out.BillingVCF,
		EwuraNpgis:       out.EwuraNpgis,
		IloExpire:        out.IloExpire,
		NotifyOutbox:     out.NotifyOutbox,
	} {
		if err := ValidateSpec(spec); err != nil {
			return config.SchedulesConfig{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	return out, nil
}

type Manager struct {
	mu      sync.Mutex
	cron    *cron.Cron
	runners map[string]func()
	entries map[string]cron.EntryID
}

var Default *Manager

func Bind(c *cron.Cron) *Manager {
	m := &Manager{
		cron:    c,
		runners: make(map[string]func()),
		entries: make(map[string]cron.EntryID),
	}
	Default = m
	return m
}

func (m *Manager) Register(name string, run func()) {
	if m == nil || run == nil || name == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runners[name] = run
}

func (m *Manager) Apply(specs config.SchedulesConfig) error {
	if m == nil || m.cron == nil {
		return fmt.Errorf("jobs manager not bound")
	}
	norm, err := Normalize(specs)
	if err != nil {
		return err
	}
	byName := map[string]string{
		LogRotation:      norm.LogRotation,
		EwuraLicenseSync: norm.EwuraLicenses,
		BillingNth:       norm.BillingNth,
		BillingTBS:       norm.BillingTBS,
		BillingVCF:       norm.BillingVCF,
		EwuraNpgis:       norm.EwuraNpgis,
		IloExpire:        norm.IloExpire,
		NotifyOutbox:     norm.NotifyOutbox,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, run := range m.runners {
		spec, ok := byName[name]
		if !ok {
			continue
		}
		if id, exists := m.entries[name]; exists {
			m.cron.Remove(id)
			delete(m.entries, name)
		}
		id, err := m.cron.AddFunc(spec, run)
		if err != nil {
			return fmt.Errorf("schedule %s (%q): %w", name, spec, err)
		}
		m.entries[name] = id
	}
	return nil
}
