package ewura

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/internal/jobs"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// licenseDTO is the EWURA petroleum-license JSON shape (live API and offline dump).
type licenseDTO struct {
	LicenseNumber string `json:"licenseNumber"`
	Licensee      string `json:"licensee"`
	LicenseClass  string `json:"licenseClass"`
	LicenseType   string `json:"licenseType"`
	Sector        string `json:"sector"`
	ZoneName      string `json:"zoneName"`
	RegionName    string `json:"regionName"`
	DistrictName  string `json:"districtName"`
	TinNumber     string `json:"tinNumber"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	IssueDate     string `json:"issueDate"`
	ExpiryDate    string `json:"expiryDate"`
}

func (r licenseDTO) number() string {
	return strings.ToUpper(strings.TrimSpace(r.LicenseNumber))
}

func (r licenseDTO) toRow() (models.EwuraPetroleumLicense, bool) {
	num := r.number()
	if num == "" {
		return models.EwuraPetroleumLicense{}, false
	}
	row := models.EwuraPetroleumLicense{
		LicenseNumber: num,
		ContentType:   types.EwuraLicenseContent,
		Licensee:      strings.TrimSpace(r.Licensee),
		LicenseClass:  strings.TrimSpace(r.LicenseClass),
		LicenseType:   strings.TrimSpace(r.LicenseType),
		Sector:        strings.TrimSpace(r.Sector),
		ZoneName:      strings.TrimSpace(r.ZoneName),
		RegionName:    strings.TrimSpace(r.RegionName),
		DistrictName:  strings.TrimSpace(r.DistrictName),
		TinNumber:     strings.TrimSpace(r.TinNumber),
		Phone:         strings.TrimSpace(r.Phone),
		Email:         strings.TrimSpace(r.Email),
		IsActive:      true,
	}
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(r.IssueDate)); err == nil {
		row.IssueDate = &t
	}
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(r.ExpiryDate)); err == nil {
		row.ExpiryDate = &t
	}
	if row.ExpiryDate != nil && !row.ExpiryDate.After(time.Now().UTC().Truncate(24*time.Hour)) {
		row.IsActive = false
	}
	return row, true
}

func upsertLicenses(db *gorm.DB, rows []licenseDTO) (int, error) {
	out := make([]models.EwuraPetroleumLicense, 0, len(rows))
	for _, r := range rows {
		row, ok := r.toRow()
		if !ok {
			continue
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		return 0, nil
	}
	// SQL Server allows 2100 parameters per request. MERGE binds one value
	// per column per row (~17 on EwuraPetroleumLicense). Stay well under.
	const mssqlLicenseBatch = 80
	if err := db.Clauses(clause.OnConflict{UpdateAll: true}).
		CreateInBatches(out, mssqlLicenseBatch).Error; err != nil {
		return 0, err
	}
	return len(out), nil
}

// ExpireLicenses marks register rows past expiry as inactive. Runs after sync
// and still runs when the EWURA fetch fails so midnight expiry does not wait
// on a live register.
func ExpireLicenses(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return db.Model(&models.EwuraPetroleumLicense{}).
		Where("ExpiryDate IS NOT NULL AND ExpiryDate < ? AND IsActive = ?", today, true).
		Update("IsActive", false).Error
}

func RegisterJobs(ctx context.Context, m *jobs.Manager, db *gorm.DB) {
	m.Register(jobs.EwuraLicenseSync, func() {
		if ctx.Err() != nil {
			return
		}
		if err := Sync(ctx, db, ""); err != nil {
			logs.Warnf("ewura license sync: %v", err)
		}
		if err := ExpireLicenses(db); err != nil {
			logs.Warnf("ewura license expire: %v", err)
		}
	})
}

func ResolveLicenseURL(explicit string) string {
	if u := strings.TrimSpace(explicit); u != "" {
		return u
	}
	if integrations.Default != nil {
		return strings.TrimSpace(integrations.Default.Npgis().LicenseURL)
	}
	return ""
}

func Sync(ctx context.Context, db *gorm.DB, url string) error {
	url = ResolveLicenseURL(url)
	if url == "" {
		return fmt.Errorf("set the EWURA license register URL under Settings → Operations → EWURA")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return formatHTTPError(resp.StatusCode, body)
	}
	var rows []licenseDTO
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return err
	}
	n, err := upsertLicenses(db, rows)
	if err != nil {
		return err
	}
	if err := ExpireLicenses(db); err != nil {
		return err
	}
	logs.Infof("EWURA licenses upserted: %d", n)
	return nil
}

func formatHTTPError(status int, body []byte) error {
	text := strings.TrimSpace(string(body))
	lower := strings.ToLower(text)
	if text == "" || strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
		if status >= 500 {
			return fmt.Errorf("ewura register unavailable (http %d)", status)
		}
		return fmt.Errorf("ewura http %d", status)
	}
	if len(text) > 200 {
		text = text[:200]
	}
	return fmt.Errorf("ewura http %d: %s", status, text)
}
