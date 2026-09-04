package settings

import (
	"errors"
	"regexp"
	"strings"

	"dfms/internal/integrations"
	"dfms/pkg/audit"
	"dfms/pkg/config"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/mail"
	"dfms/pkg/precision"
	"dfms/pkg/response"
	"dfms/pkg/sms"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

// integrationSaveError maps store save failures: client validation → 400 with
// message; anything else is logged and returned as a generic 500.
func integrationSaveError(c fiber.Ctx, op string, err error) error {
	if ce, ok := errors.AsType[*integrations.ClientError](err); ok {
		return response.BadRequest(c, ce.Error())
	}
	logs.Errorf("%s: %v", op, err)
	return response.InternalServerError(c)
}

// ── Public views (secrets never returned) ───────────────────────────

func mailPublic(cfg config.MailConfig) fiber.Map {
	return fiber.Map{
		"enabled":     cfg.Enabled,
		"host":        cfg.Host,
		"port":        cfg.Port,
		"user":        cfg.User,
		"fromName":    cfg.FromName,
		"fromEmail":   cfg.FromEmail,
		"useTLS":      cfg.UseTLS,
		"useSSL":      cfg.UseSSL,
		"hasPassword": strings.TrimSpace(cfg.Password) != "",
	}
}

func smsPublic(cfg config.SMSConfig) fiber.Map {
	return fiber.Map{
		"enabled":   cfg.Enabled,
		"apiUrl":    cfg.APIURL,
		"senderId":  cfg.SenderID,
		"hasApiKey": strings.TrimSpace(cfg.APIKey) != "",
	}
}

func storeOrErr(c fiber.Ctx) (*integrations.Store, error) {
	if integrations.Default == nil {
		return nil, response.InternalServerError(c)
	}
	return integrations.Default, nil
}

// GetMail returns SMTP settings (password never included).
func (h *Handler) GetMail(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, mailPublic(s.Mail()))
}

// GetSMS returns SMS gateway settings (API key never included).
func (h *Handler) GetSMS(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, smsPublic(s.SMS()))
}

// UpdateMail saves SMTP settings and reloads the in-memory store.
func (h *Handler) UpdateMail(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body mailUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Mail()
	if body.Enabled != nil {
		cfg.Enabled = *body.Enabled
	}
	cfg.Host = strings.TrimSpace(body.Host)
	if body.Port != nil {
		cfg.Port = *body.Port
	}
	cfg.User = strings.TrimSpace(body.User)
	cfg.FromName = strings.TrimSpace(body.FromName)
	cfg.FromEmail = strings.TrimSpace(body.FromEmail)
	if body.UseTLS != nil {
		cfg.UseTLS = *body.UseTLS
	}
	if body.UseSSL != nil {
		cfg.UseSSL = *body.UseSSL
	}
	secrets := integrations.SecretPatch{}
	if body.ClearPassword {
		empty := ""
		secrets.Password = &empty
	} else if body.Password != nil {
		secrets.Password = body.Password
	}
	beforeCfg := s.Mail()
	before := mailPublic(beforeCfg)
	if err := s.SaveMail(cfg, secrets); err != nil {
		return integrationSaveError(c, "save mail settings", err)
	}
	afterCfg := s.Mail()
	h.auditIntegration(c, "mail", before, mailPublic(afterCfg),
		secretAudit{Field: "password", Touched: secrets.Password != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.Password) != "",
			AfterSet:  strings.TrimSpace(afterCfg.Password) != ""},
	)
	return integrationUpdate(c, before, mailPublic(afterCfg),
		secretAudit{Field: "password", Touched: secrets.Password != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.Password) != "",
			AfterSet:  strings.TrimSpace(afterCfg.Password) != ""})
}

// UpdateSMS saves SMS settings and reloads the in-memory store.
func (h *Handler) UpdateSMS(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body smsUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.SMS()
	if body.Enabled != nil {
		cfg.Enabled = *body.Enabled
	}
	cfg.APIURL = strings.TrimSpace(body.APIURL)
	cfg.SenderID = strings.TrimSpace(body.SenderID)
	secrets := integrations.SecretPatch{}
	if body.ClearAPIKey {
		empty := ""
		secrets.APIKey = &empty
	} else if body.APIKey != nil {
		secrets.APIKey = body.APIKey
	}
	beforeCfg := s.SMS()
	before := smsPublic(beforeCfg)
	if err := s.SaveSMS(cfg, secrets); err != nil {
		return integrationSaveError(c, "save sms settings", err)
	}
	afterCfg := s.SMS()
	h.auditIntegration(c, "sms", before, smsPublic(afterCfg),
		secretAudit{Field: "apiKey", Touched: secrets.APIKey != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.APIKey) != "",
			AfterSet:  strings.TrimSpace(afterCfg.APIKey) != ""},
	)
	return integrationUpdate(c, before, smsPublic(afterCfg),
		secretAudit{Field: "apiKey", Touched: secrets.APIKey != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.APIKey) != "",
			AfterSet:  strings.TrimSpace(afterCfg.APIKey) != ""})
}

func schedulesPublic(cfg config.SchedulesConfig) fiber.Map {
	return fiber.Map{
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

// GetSchedules returns background job cron specs.
func (h *Handler) GetSchedules(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, schedulesPublic(s.Schedules()))
}

// UpdateSchedules saves cron specs and reschedules live jobs without restart.
func (h *Handler) UpdateSchedules(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body schedulesUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := config.SchedulesConfig{
		LogRotation:   body.LogRotation,
		EwuraLicenses: body.EwuraLicenses,
		BillingNth:    body.BillingNth,
		BillingTBS:    body.BillingTBS,
		BillingVCF:    body.BillingVCF,
		EwuraNpgis:    body.EwuraNpgis,
		IloExpire:     body.IloExpire,
		NotifyOutbox:  body.NotifyOutbox,
	}
	before := schedulesPublic(s.Schedules())
	if err := s.SaveSchedules(cfg); err != nil {
		return integrationSaveError(c, "save schedules", err)
	}
	after := schedulesPublic(s.Schedules())
	h.auditIntegration(c, "schedules", before, after)
	return integrationUpdate(c, before, after)
}

// TestMail sends a sample message via the warmed SMTP connection (Settings → Mail).
func (h *Handler) TestMail(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body mailTestRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	to := strings.TrimSpace(body.To)
	cfg := s.Mail()
	if !cfg.Enabled || strings.TrimSpace(cfg.Host) == "" {
		return response.BadRequest(c, "enable SMTP and save host credentials before sending a test")
	}
	subject := "TIPER DFMS — test email"
	html := "<p>This is a test message from TIPER DFMS.</p>" +
		"<p>If you received it, SMTP credentials and the reused connection are working.</p>"
	if err := mail.Send(c.Context(), []string{to}, subject, html); err != nil {
		return response.BadRequest(c, "test email failed: "+err.Error())
	}
	return response.Ok(c, "Test email sent", fiber.Map{"to": to})
}

// testPhoneRegex matches E.164 without +: 7–15 digits, first digit 1–9.
var testPhoneRegex = regexp.MustCompile(`^[1-9][0-9]{6,14}$`)

// TestSMS sends a sample SMS via the configured gateway (Settings → SMS).
func (h *Handler) TestSMS(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body smsTestRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	to := strings.TrimSpace(strings.TrimPrefix(body.To, "+"))
	to = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, to)
	if !testPhoneRegex.MatchString(to) {
		return response.BadRequest(c, "a valid international phone number is required (digits only, e.g. 255711223344)")
	}
	cfg := s.SMS()
	if !cfg.Enabled || strings.TrimSpace(cfg.APIURL) == "" {
		return response.BadRequest(c, "enable SMS and save the API URL and key before sending a test")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return response.BadRequest(c, "save an SMS API key before sending a test")
	}
	fromName := strings.TrimSpace(s.Mail().FromName)
	if fromName == "" || strings.Contains(fromName, "TIPER DFMS") {
		fromName = "TIPER DFMS"
	}
	msg := fromName + " test SMS — if you received this, the SMS gateway credentials are working."
	if err := sms.Send(c.Context(), to, msg); err != nil {
		logs.Errorf("test sms to %s: %v", to, err)
		return response.BadRequest(c, "test SMS failed: "+err.Error())
	}
	return response.Ok(c, "Test SMS sent", fiber.Map{"to": to})
}

func sagePublic(cfg config.DbConfig) fiber.Map {
	return fiber.Map{
		"host":        cfg.Host,
		"port":        cfg.Port,
		"instance":    cfg.Instance,
		"user":        cfg.User,
		"name":        cfg.Name,
		"encrypt":     cfg.Encrypt,
		"hasPassword": strings.TrimSpace(cfg.Password) != "",
		"connected":   db.Sage() != nil,
	}
}

func sessionPublic(cfg config.SessionConfig) fiber.Map {
	c := cfg.Clamp()
	return fiber.Map{
		"idleMinutes": c.IdleMinutes,
		"warnMinutes": c.WarnMinutes(),
		"warnSeconds": c.WarnSeconds,
	}
}

func npgisPublic(cfg config.NpgisConfig) fiber.Map {
	return fiber.Map{
		"enabled":     cfg.Enabled,
		"licenseUrl":  cfg.LicenseURL,
		"baseUrl":     cfg.BaseURL,
		"licenseNo":   cfg.LicenseNo,
		"apiSourceId": cfg.APISourceID,
		"depotName":   cfg.DepotName,
	}
}

func uploadsPublic(cfg config.UploadsConfig) fiber.Map {
	return fiber.Map{
		"directory":          cfg.Directory,
		"maxFileSizeMB":      cfg.MaxFileSizeMB,
		"maxFilesPerRequest": cfg.MaxFilesPerRequest,
		"processBodyLimitMB": 64,
	}
}

// GetSage returns Sage 200 connection settings (password never included).
func (h *Handler) GetSage(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, sagePublic(s.Sage()))
}

// UpdateSage saves Sage 200 settings and reconnects without restart.
func (h *Handler) UpdateSage(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body sageUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Sage()
	cfg.Host = strings.TrimSpace(body.Host)
	cfg.Port = strings.TrimSpace(body.Port)
	cfg.Instance = strings.TrimSpace(body.Instance)
	cfg.User = strings.TrimSpace(body.User)
	cfg.Name = strings.TrimSpace(body.Name)
	if body.Encrypt != nil {
		cfg.Encrypt = *body.Encrypt
	}
	secrets := integrations.SecretPatch{}
	if body.ClearPassword {
		empty := ""
		secrets.Password = &empty
	} else if body.Password != nil {
		secrets.Password = body.Password
	}
	beforeCfg := s.Sage()
	before := sagePublic(beforeCfg)
	saveErr := s.SaveSage(cfg, secrets)
	afterCfg := s.Sage()
	h.auditIntegration(c, "sage", before, sagePublic(afterCfg),
		secretAudit{Field: "password", Touched: secrets.Password != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.Password) != "",
			AfterSet:  strings.TrimSpace(afterCfg.Password) != ""},
	)
	if saveErr != nil {
		return integrationSaveError(c, "save sage settings", saveErr)
	}
	return integrationUpdate(c, before, sagePublic(afterCfg),
		secretAudit{Field: "password", Touched: secrets.Password != nil,
			BeforeSet: strings.TrimSpace(beforeCfg.Password) != "",
			AfterSet:  strings.TrimSpace(afterCfg.Password) != ""})
}

// TestSage pings Sage 200 with the request overlay (or stored settings).
func (h *Handler) TestSage(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body sageTestRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Sage()
	if host := strings.TrimSpace(body.Host); host != "" {
		cfg.Host = host
	}
	if port := strings.TrimSpace(body.Port); port != "" {
		cfg.Port = port
	}
	if v := strings.TrimSpace(body.Instance); v != "" {
		cfg.Instance = v
	}
	if user := strings.TrimSpace(body.User); user != "" {
		cfg.User = user
	}
	if name := strings.TrimSpace(body.Name); name != "" {
		cfg.Name = name
	}
	if body.Encrypt != nil {
		cfg.Encrypt = *body.Encrypt
	}
	if body.Password != nil {
		cfg.Password = strings.TrimSpace(*body.Password)
	}
	if !db.SageConfigured(cfg) {
		return response.BadRequest(c, "host, user, password, database and port are required — save them or include them in the test")
	}
	if err := db.PingSage(c.Context(), cfg); err != nil {
		return response.BadRequest(c, "Sage 200 test failed: "+err.Error())
	}
	return response.Ok(c, "Sage 200 connection succeeded", nil)
}

// GetSession returns the live idle timeout (Settings → Session).
func (h *Handler) GetSession(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, sessionPublic(s.Session()))
}

// UpdateSession saves idle timeout and applies it without restart.
func (h *Handler) UpdateSession(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body sessionUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Session()
	if body.IdleMinutes != nil {
		cfg.IdleMinutes = *body.IdleMinutes
	}
	if body.WarnMinutes != nil {
		cfg.WarnSeconds = *body.WarnMinutes * 60
	}
	if cfg.WarnSeconds >= cfg.IdleMinutes*60 {
		return response.BadRequest(c, "warning must start before sign-out (shorter than the idle time)")
	}
	before := sessionPublic(s.Session())
	if err := s.SaveSession(cfg); err != nil {
		return integrationSaveError(c, "save session settings", err)
	}
	after := sessionPublic(s.Session())
	h.auditIntegration(c, "session", before, after)
	return integrationUpdate(c, before, after)
}

func precisionPublic(p precision.Settings, o config.OrdersConfig) fiber.Map {
	p = p.Normalize()
	o = o.Clamp()
	return fiber.Map{
		"quantityPrecision":    p.Quantity,
		"cubicMeterPrecision":  p.CubicMeter,
		"metricTonnePrecision": p.MetricTonne,
		"densityPrecision":     p.Density,
		"pricePrecision":       p.Price,
		"miLossPrecision":      p.MiLoss,
		"iloExpiryDays":        o.IloExpiryDays,
	}
}

// GetPrecision returns live rounding plus ILO expiry (merged for the Precision panel).
func (h *Handler) GetPrecision(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, precisionPublic(s.Precision(), s.Orders()))
}

// UpdatePrecision saves rounding (key=precision) and ILO days (key=orders).
func (h *Handler) UpdatePrecision(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body precisionUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	before := precisionPublic(s.Precision(), s.Orders())
	beforePrec := precisionPublic(s.Precision(), config.OrdersConfig{})
	beforeOrders := fiber.Map{"iloExpiryDays": s.Orders().IloExpiryDays}
	prec := precision.Settings{
		Quantity:    body.QuantityPrecision,
		CubicMeter:  body.CubicMeterPrecision,
		MetricTonne: body.MetricTonnePrecision,
		Density:     body.DensityPrecision,
		Price:       body.PricePrecision,
		MiLoss:      body.MiLossPrecision,
	}
	if err := s.SavePrecision(prec); err != nil {
		return integrationSaveError(c, "save precision settings", err)
	}
	if err := s.SaveOrders(config.OrdersConfig{IloExpiryDays: body.IloExpiryDays}); err != nil {
		return integrationSaveError(c, "save order settings", err)
	}
	after := precisionPublic(s.Precision(), s.Orders())
	h.auditIntegration(c, types.KeyPrecision, beforePrec, fiber.Map{
		"quantityPrecision":    after["quantityPrecision"],
		"cubicMeterPrecision":  after["cubicMeterPrecision"],
		"metricTonnePrecision": after["metricTonnePrecision"],
		"densityPrecision":     after["densityPrecision"],
		"pricePrecision":       after["pricePrecision"],
		"miLossPrecision":      after["miLossPrecision"],
	})
	h.auditIntegration(c, types.KeyOrders, beforeOrders, fiber.Map{"iloExpiryDays": after["iloExpiryDays"]})
	return response.Ok(c, audit.UpdateMessage(before, after), after)
}

// GetNpgis returns EWURA license-register and NPGIS retailer settings.
func (h *Handler) GetNpgis(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, npgisPublic(s.Npgis()))
}

// UpdateNpgis saves EWURA license-register and NPGIS settings without restart.
func (h *Handler) UpdateNpgis(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body npgisUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Npgis()
	if body.Enabled != nil {
		cfg.Enabled = *body.Enabled
	}
	cfg.LicenseURL = strings.TrimSpace(body.LicenseURL)
	cfg.BaseURL = strings.TrimSpace(body.BaseURL)
	cfg.LicenseNo = strings.TrimSpace(body.LicenseNo)
	cfg.APISourceID = strings.TrimSpace(body.APISourceID)
	cfg.DepotName = strings.TrimSpace(body.DepotName)
	before := npgisPublic(s.Npgis())
	if err := s.SaveNpgis(cfg); err != nil {
		return integrationSaveError(c, "save EWURA settings", err)
	}
	after := npgisPublic(s.Npgis())
	h.auditIntegration(c, "npgis", before, after)
	return integrationUpdate(c, before, after)
}

// GetUploads returns live attachment directory and size/count caps.
func (h *Handler) GetUploads(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	return response.OkDetail(c, uploadsPublic(s.Uploads()))
}

// UpdateUploads saves the attachment directory and caps and applies them without restart.
func (h *Handler) UpdateUploads(c fiber.Ctx) error {
	s, err := storeOrErr(c)
	if err != nil {
		return err
	}
	var body uploadsUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	body.Sanitize()
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cfg := s.Uploads()
	if body.MaxFileSizeMB != nil {
		cfg.MaxFileSizeMB = *body.MaxFileSizeMB
	}
	if body.MaxFilesPerRequest != nil {
		cfg.MaxFilesPerRequest = *body.MaxFilesPerRequest
	}
	if body.Directory != nil {
		cfg.Directory = *body.Directory
	}
	before := uploadsPublic(s.Uploads())
	if err := s.SaveUploads(cfg); err != nil {
		return integrationSaveError(c, "save upload settings", err)
	}
	after := uploadsPublic(s.Uploads())
	h.auditIntegration(c, "uploads", before, after)
	return integrationUpdate(c, before, after)
}

func integrationUpdate(c fiber.Ctx, before, after fiber.Map, secrets ...secretAudit) error {
	changes := audit.DiffValues(before, after)
	for _, k := range []string{"hasPassword", "hasApiKey", "connected"} {
		delete(changes, k)
	}
	for _, sec := range secrets {
		audit.MergeSecretChange(changes, sec.Field, sec.Touched, sec.BeforeSet, sec.AfterSet)
	}
	return response.OkMessage(c, audit.UpdateMessageFromChanges(changes))
}

func (h *Handler) auditIntegration(c fiber.Ctx, key string, before, after fiber.Map, secrets ...secretAudit) {
	if audit.Default == nil {
		return
	}
	// Booleans like hasPassword are replaced by masked secret fields below.
	skip := map[string]bool{
		"hasPassword": true,
		"hasApiKey":   true,
		"connected":   true,
	}
	changes := map[string]any{}
	for k, av := range after {
		if skip[k] {
			continue
		}
		bv := before[k]
		if equalAuditValue(bv, av) {
			continue
		}
		changes[k] = audit.FieldChange{Before: bv, After: av}
	}
	for _, sec := range secrets {
		if !sec.Touched || sec.Field == "" {
			continue
		}
		changes[sec.Field] = audit.FieldChange{
			Before: maskSecretAudit(sec.BeforeSet),
			After:  maskSecretAudit(sec.AfterSet),
		}
	}
	if len(changes) == 0 {
		return
	}
	entry := audit.AuditEntry(
		c, types.ModuleSettings, types.ActionUpdate, key, types.IntegrationSettingContent,
		audit.EnrichDescription("updated integration "+key, changes),
	)
	entry.Changes = changes
	_ = audit.Default.Record(c.Context(), nil, entry)
}

// secretAudit records a secret field change without ever storing the plaintext.
type secretAudit struct {
	Field     string // password | apiKey
	Touched   bool   // request included a set/clear for this secret
	BeforeSet bool
	AfterSet  bool
}

func maskSecretAudit(set bool) string {
	if set {
		return "***"
	}
	return "(not set)"
}

func equalAuditValue(a, b any) bool {
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as == bs
	}
	return a == b
}
