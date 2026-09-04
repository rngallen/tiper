package settings

import (
	"context"
	"regexp"
	"strings"

	"github.com/jellydator/validation"
	"github.com/jellydator/validation/is"
)

// companyRequest is the editable subset of the company profile.
type companyRequest struct {
	Name         string `json:"name"`
	TinNumber    string `json:"tinNumber"`
	VrnNumber    string `json:"vrnNumber"`
	IsoNumber    string `json:"isoNumber"`
	Address      string `json:"address"`
	Address2     string `json:"address2"`
	City         string `json:"city"`
	Country      string `json:"country"`
	PostalCode   string `json:"postalCode"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Website      string `json:"website"`
	PortalURL    string `json:"portalUrl"`
	CurrencyCode string `json:"currencyCode"`
}

// Validate bounds company profile fields.
func (r companyRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Name, validation.Length(0, 200)),
		validation.Field(&r.TinNumber, validation.Length(0, 40)),
		validation.Field(&r.VrnNumber, validation.Length(0, 40)),
		validation.Field(&r.IsoNumber, validation.Length(0, 80)),
		validation.Field(&r.Address, validation.Length(0, 255)),
		validation.Field(&r.Address2, validation.Length(0, 255)),
		validation.Field(&r.City, validation.Length(0, 80)),
		validation.Field(&r.Country, validation.Length(0, 80)),
		validation.Field(&r.PostalCode, validation.Length(0, 40)),
		validation.Field(&r.Phone, validation.Length(0, 40)),
		validation.Field(&r.Email, validation.When(r.Email != "", is.Email)),
		validation.Field(&r.Website, validation.Length(0, 200)),
		validation.Field(&r.PortalURL, validation.Length(0, 255), validation.When(
			strings.TrimSpace(r.PortalURL) != "",
			is.URL,
		)),
		validation.Field(&r.CurrencyCode, validation.Length(0, 3)),
	)
}

type mailUpdateRequest struct {
	Enabled       *bool   `json:"enabled"`
	Host          string  `json:"host"`
	Port          *int    `json:"port"`
	User          string  `json:"user"`
	FromName      string  `json:"fromName"`
	FromEmail     string  `json:"fromEmail"`
	UseTLS        *bool   `json:"useTLS"`
	UseSSL        *bool   `json:"useSSL"`
	Password      *string `json:"password"`
	ClearPassword bool    `json:"clearPassword"`
}

// Validate bounds SMTP settings fields.
func (r mailUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Host, validation.Length(0, 255)),
		validation.Field(&r.User, validation.Length(0, 255)),
		validation.Field(&r.FromName, validation.Length(0, 120)),
		validation.Field(&r.FromEmail, validation.When(strings.TrimSpace(r.FromEmail) != "", is.Email)),
		validation.Field(&r.Port, validation.By(func(v any) error {
			p, _ := v.(*int)
			if p == nil {
				return nil
			}
			if *p < 1 || *p > 65535 {
				return validation.NewError("validation_range", "port must be between 1 and 65535")
			}
			return nil
		})),
	)
}

type smsUpdateRequest struct {
	Enabled     *bool   `json:"enabled"`
	APIURL      string  `json:"apiUrl"`
	SenderID    string  `json:"senderId"`
	APIKey      *string `json:"apiKey"`
	ClearAPIKey bool    `json:"clearApiKey"`
}

// Validate bounds SMS gateway fields.
func (r smsUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.APIURL, validation.Length(0, 500)),
		validation.Field(&r.SenderID, validation.Length(0, 40)),
	)
}

type schedulesUpdateRequest struct {
	LogRotation   string `json:"logRotation"`
	EwuraLicenses string `json:"ewuraLicenses"`
	BillingNth    string `json:"billingNth"`
	BillingTBS    string `json:"billingTbs"`
	BillingVCF    string `json:"billingVcf"`
	EwuraNpgis    string `json:"ewuraNpgis"`
	IloExpire     string `json:"iloExpire"`
	NotifyOutbox  string `json:"notifyOutbox"`
}

// Validate requires every cron field to be present (empty clears are rejected by store).
func (r schedulesUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.LogRotation, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.EwuraLicenses, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.BillingNth, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.BillingTBS, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.BillingVCF, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.EwuraNpgis, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.IloExpire, validation.Required, validation.Length(1, 120)),
		validation.Field(&r.NotifyOutbox, validation.Required, validation.Length(1, 120)),
	)
}

type mailTestRequest struct {
	To string `json:"to"`
}

// Validate requires a plausible recipient email.
func (r mailTestRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.To, validation.Required, is.Email),
	)
}

type sageUpdateRequest struct {
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	Instance      string  `json:"instance"`
	User          string  `json:"user"`
	Name          string  `json:"name"`
	Encrypt       *bool   `json:"encrypt"`
	Password      *string `json:"password"`
	ClearPassword bool    `json:"clearPassword"`
}

// Validate bounds Sage 200 connection fields.
func (r sageUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Host, validation.Length(0, 255), validation.When(strings.TrimSpace(r.Host) != "", is.Host)),
		validation.Field(&r.Port, validation.Length(0, 8), validation.When(strings.TrimSpace(r.Port) != "", is.Port)),
		validation.Field(&r.Instance, validation.Length(0, 128)),
		validation.Field(&r.User, validation.Length(0, 128)),
		validation.Field(&r.Name, validation.Length(0, 128)),
	)
}

type sageTestRequest struct {
	Host     string  `json:"host"`
	Port     string  `json:"port"`
	Instance string  `json:"instance"`
	User     string  `json:"user"`
	Name     string  `json:"name"`
	Encrypt  *bool   `json:"encrypt"`
	Password *string `json:"password"`
}

// Validate bounds optional overlay fields for a Sage ping.
func (r sageTestRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Host, validation.Length(0, 255), validation.When(strings.TrimSpace(r.Host) != "", is.Host)),
		validation.Field(&r.Port, validation.Length(0, 8), validation.When(strings.TrimSpace(r.Port) != "", is.Port)),
		validation.Field(&r.Instance, validation.Length(0, 128)),
		validation.Field(&r.User, validation.Length(0, 128)),
		validation.Field(&r.Name, validation.Length(0, 128)),
	)
}

type npgisUpdateRequest struct {
	Enabled     *bool  `json:"enabled"`
	LicenseURL  string `json:"licenseUrl"`
	BaseURL     string `json:"baseUrl"`
	LicenseNo   string `json:"licenseNo"`
	APISourceID string `json:"apiSourceId"`
	DepotName   string `json:"depotName"`
}

// Validate bounds EWURA license-register and NPGIS retailer fields.
func (r npgisUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.LicenseURL, validation.Length(0, 500), validation.When(
			strings.TrimSpace(r.LicenseURL) != "",
			is.URL,
		)),
		validation.Field(&r.BaseURL, validation.Length(0, 500), validation.When(
			strings.TrimSpace(r.BaseURL) != "",
			is.URL,
		)),
		validation.Field(&r.LicenseNo, validation.Length(0, 80)),
		validation.Field(&r.APISourceID, validation.Length(0, 80)),
		validation.Field(&r.DepotName, validation.Length(0, 80)),
	)
}

type sessionUpdateRequest struct {
	IdleMinutes *int `json:"idleMinutes"`
	WarnMinutes *int `json:"warnMinutes"`
}

// Validate keeps idle 2–480 minutes and warning 1–10 minutes, shorter than idle.
func (r sessionUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.IdleMinutes, validation.By(func(v any) error {
			p, _ := v.(*int)
			if p == nil {
				return nil
			}
			if *p < 2 || *p > 480 {
				return validation.NewError("validation_range", "sign-out after inactivity must be between 2 and 480 minutes")
			}
			return nil
		})),
		validation.Field(&r.WarnMinutes, validation.By(func(v any) error {
			p, _ := v.(*int)
			if p == nil {
				return nil
			}
			if *p < 1 || *p > 10 {
				return validation.NewError("validation_range", "warning must be between 1 and 10 minutes")
			}
			return nil
		})),
	)
}

type uploadsUpdateRequest struct {
	Directory          *string `json:"directory"`
	MaxFileSizeMB      *int    `json:"maxFileSizeMB"`
	MaxFilesPerRequest *int    `json:"maxFilesPerRequest"`
}

func (r *uploadsUpdateRequest) Sanitize() {
	if r.Directory != nil {
		s := strings.TrimSpace(*r.Directory)
		r.Directory = &s
	}
}

// Validate keeps attachment caps inside the product envelope (1–25 MB, 1–10 files).
func (r uploadsUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Directory, validation.By(func(v any) error {
			p, _ := v.(*string)
			if p == nil || strings.TrimSpace(*p) == "" {
				return nil
			}
			s := strings.TrimSpace(*p)
			if strings.Contains(s, "..") || strings.ContainsRune(s, 0) {
				return validation.NewError("validation_path", "directory must not contain ..")
			}
			if len(s) > 260 {
				return validation.NewError("validation_length", "directory must be at most 260 characters")
			}
			return nil
		})),
		validation.Field(&r.MaxFileSizeMB, validation.By(func(v any) error {
			p, _ := v.(*int)
			if p == nil {
				return nil
			}
			if *p < 1 || *p > 25 {
				return validation.NewError("validation_range", "max file size must be between 1 and 25 MB")
			}
			return nil
		})),
		validation.Field(&r.MaxFilesPerRequest, validation.By(func(v any) error {
			p, _ := v.(*int)
			if p == nil {
				return nil
			}
			if *p < 1 || *p > 10 {
				return validation.NewError("validation_range", "max files per request must be between 1 and 10")
			}
			return nil
		})),
	)
}

type smsTestRequest struct {
	To string `json:"to"`
}

// Validate requires a recipient; digit-shape is checked after handler normalization.
func (r smsTestRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.To, validation.Required, validation.Length(7, 20)),
	)
}

type currencyCreateRequest struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type precisionUpdateRequest struct {
	QuantityPrecision    int `json:"quantityPrecision"`
	CubicMeterPrecision  int `json:"cubicMeterPrecision"`
	MetricTonnePrecision int `json:"metricTonnePrecision"`
	DensityPrecision     int `json:"densityPrecision"`
	PricePrecision       int `json:"pricePrecision"`
	MiLossPrecision      int `json:"miLossPrecision"`
	IloExpiryDays        int `json:"iloExpiryDays"`
}

func (r precisionUpdateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.QuantityPrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.CubicMeterPrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.MetricTonnePrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.DensityPrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.PricePrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.MiLossPrecision, validation.Min(0), validation.Max(6)),
		validation.Field(&r.IloExpiryDays, validation.Required, validation.Min(1), validation.Max(90)),
	)
}

func (r currencyCreateRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Code, validation.Required, validation.Length(3, 3), validation.Match(regexp.MustCompile(`^[A-Za-z]{3}$`))),
		validation.Field(&r.Symbol, validation.Required, validation.Length(1, 10)),
		validation.Field(&r.Name, validation.Length(0, 100)),
	)
}
