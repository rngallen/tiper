package notify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/pkg/mail"
	"dfms/pkg/portalurl"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// EmailBrand is letterhead used on outbound HTML mail. Colours follow the
// TIPER DFMS UI. Optional logo/hero files are inlined as CID parts so clients
// never fetch the public site (localhost and VPN-only hosts fail on phones).
type EmailBrand struct {
	Name      string
	FromName  string // SMTP From display name; also the SMS message prefix
	Address   string
	LogoURL   string
	HeroURL   string
	PortalURL string
}

const (
	defaultOrgName  = "Tanzania International Petroleum Reserves Limited"
	defaultFromName = "TIPER DFMS"
	colorNavy       = "#033860"
	colorAccent     = "#dc295c"
	colorCanvas     = "#f0f4f8"
	colorText       = "#0f172a"
	colorMuted      = "#64748b"
	colorSuccess    = "#059669"
	emailLogoCID    = "tiper-logo.png"
	emailHeroCID    = "tiper-email-hero.jpg"
)

// loadBrand reads Company letterhead for email footers and hosted brand assets.
func loadBrand(ctx context.Context, db *gorm.DB) EmailBrand {
	b := EmailBrand{Name: defaultOrgName, FromName: defaultFromName}
	if integrations.Default != nil {
		if fn := strings.TrimSpace(integrations.Default.Mail().FromName); fn != "" {
			b.FromName = productFromName(fn)
		}
	}
	if db == nil {
		return b
	}
	var c models.Company
	if err := db.WithContext(ctx).First(&c).Error; err != nil {
		b.PortalURL = portalurl.Base(ctx, db)
		applyAssetURLs(&b)
		return b
	}
	if name := strings.TrimSpace(c.Name); name != "" {
		b.Name = name
	}
	b.Address = companyAddress(c)
	base := strings.TrimRight(strings.TrimSpace(c.PortalURL), "/")
	if base == "" {
		base = portalurl.Base(ctx, db)
	}
	b.PortalURL = base
	applyAssetURLs(&b)
	return b
}

func applyAssetURLs(b *EmailBrand) {
	// Assets travel inside the MIME message (cid:). Remote public URLs
	// (localhost or VPN-only) never load on phones or home networks.
	if findEmailAsset(emailLogoCID) != "" {
		b.LogoURL = "cid:" + emailLogoCID
	}
	if findEmailAsset(emailHeroCID) != "" {
		b.HeroURL = "cid:" + emailHeroCID
	}
}

func findEmailAsset(name string) string {
	candidates := []string{
		filepath.Join("web", "public", name),
		filepath.Join("public", name),
		name,
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "web", "public", name),
			filepath.Join(dir, name),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func sendBranded(ctx context.Context, to []string, subject, html string) error {
	msg := mail.Message{To: to, Subject: subject, HTMLBody: html}
	for _, file := range []struct {
		cid, ctype string
	}{
		{emailLogoCID, "image/png"},
		{emailHeroCID, "image/jpeg"},
	} {
		p := findEmailAsset(file.cid)
		if p == "" {
			continue
		}
		raw, err := os.ReadFile(p)
		if err != nil || len(raw) == 0 {
			continue
		}
		msg.Embeds = append(msg.Embeds, mail.Attachment{
			Filename:    file.cid,
			ContentType: file.ctype,
			Content:     raw,
		})
	}
	return mail.Default.SendMessage(ctx, msg)
}

// link joins path onto the configured public frontend base URL.
func (b EmailBrand) link(path string) string {
	return portalurl.Join(b.PortalURL, path)
}

// smsName is the product label used as the SMS prefix (same as email From name).
func (b EmailBrand) smsName() string {
	if s := strings.TrimSpace(b.FromName); s != "" {
		return productFromName(s)
	}
	return defaultFromName
}

func productFromName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultFromName
	}
	return s
}

func companyAddress(c models.Company) string {
	parts := make([]string, 0, 6)
	for _, s := range []string{c.Address, c.Address2} {
		if t := strings.TrimSpace(s); t != "" {
			parts = append(parts, t)
		}
	}
	cityLine := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(c.City), strings.TrimSpace(c.PostalCode),
	}, " "))
	if cityLine != "" {
		parts = append(parts, cityLine)
	}
	if t := strings.TrimSpace(c.Country); t != "" {
		parts = append(parts, t)
	}
	contact := strings.TrimSpace(strings.Join(filterNonEmpty(c.Phone, c.Email), " · "))
	if contact != "" {
		parts = append(parts, contact)
	}
	return strings.Join(parts, "<br>")
}

func filterNonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// kv is one label/value row in a summary table.
type kv struct {
	Label     string
	Value     string
	Highlight bool
	// Secret renders as a selectable monospace box (passwords, OTP codes).
	Secret bool
}

func badgeColors(kind types.EmailBadge) (bg, fg string) {
	switch kind {
	case types.EmailBadgeSuccess:
		return "#ecfdf5", colorSuccess
	case types.EmailBadgeWarning:
		return "#fff7ed", "#c2410c"
	case types.EmailBadgeInfo:
		return "#e8f1f8", colorNavy
	default:
		return "#fde8ee", colorAccent
	}
}

// renderCorporateEmail matches the TIPER DFMS chrome: navy + accent,
// white card on the app canvas, summary table, callouts, CTA.
func renderCorporateEmail(brand EmailBrand, badge string, kind types.EmailBadge, greeting, intro, sectionTitle string, rows []kv, nextSteps, security, ctaLabel, ctaURL string) string {
	bg, fg := badgeColors(kind)
	name := htmlEscape(brand.Name)
	if name == "" {
		name = defaultOrgName
	}

	logoHTML := fmt.Sprintf(
		`<div style="font-size:18px;font-weight:700;color:%s;letter-spacing:.04em">TIPER</div>
		 <div style="font-size:11px;color:%s;margin-top:2px;letter-spacing:.12em;text-transform:uppercase">DFMS</div>`,
		colorNavy, colorMuted,
	)
	if brand.LogoURL != "" {
		logoHTML = fmt.Sprintf(
			`<img src="%s" alt="%s" height="44" style="height:44px;width:auto;border:0;display:block">`,
			htmlEscape(brand.LogoURL), name,
		)
	}

	heroHTML := ""
	if brand.HeroURL != "" {
		heroHTML = fmt.Sprintf(
			`<tr><td style="padding:0;line-height:0;font-size:0;background:%s">
				<img src="%s" alt="TIPER Kigamboni — storage tanks and gantry loading" width="600" height="120" style="display:block;width:100%%;max-width:600px;height:120px;object-fit:cover;border:0">
			</td></tr>
			<tr><td style="padding:10px 28px;background:%s">
				<div style="font-size:11px;font-weight:700;letter-spacing:.14em;text-transform:uppercase;color:#ffffff">TIPER DFMS</div>
				<div style="font-size:12px;color:#c5d5e4;margin-top:3px">Kigamboni bonded warehouse · Dar es Salaam</div>
			</td></tr>
			<tr><td style="height:3px;background:%s;font-size:0;line-height:0">&nbsp;</td></tr>`,
			colorNavy, htmlEscape(brand.HeroURL), colorNavy, colorAccent,
		)
	} else {
		heroHTML = fmt.Sprintf(
			`<tr><td style="height:6px;background:%s;font-size:0;line-height:0">&nbsp;</td></tr>
			<tr><td style="height:3px;background:%s;font-size:0;line-height:0">&nbsp;</td></tr>`,
			colorNavy, colorAccent,
		)
	}

	badgeHTML := ""
	if strings.TrimSpace(badge) != "" {
		badgeHTML = fmt.Sprintf(
			`<span style="display:inline-block;background:%s;color:%s;font-size:11px;font-weight:700;letter-spacing:.06em;padding:6px 12px;border-radius:999px;white-space:nowrap">%s</span>`,
			bg, fg, htmlEscape(strings.ToUpper(badge)),
		)
	}

	var rowsHTML strings.Builder
	for i, r := range rows {
		border := ""
		if i > 0 {
			border = "border-top:1px solid #e2e8f0;"
		}
		if r.Secret {
			codeSize := "16px"
			codeTrack := ".04em"
			if n := len(strings.TrimSpace(r.Value)); n > 0 && n <= 8 {
				codeSize = "28px"
				codeTrack = ".18em"
			}
			fmt.Fprintf(&rowsHTML, `<tr>
				<td colspan="2" style="padding:8px 0;%s">
					<div style="color:%s;font-size:11px;font-weight:700;letter-spacing:.1em;text-transform:uppercase;text-align:center;margin-bottom:10px">%s</div>
					<div style="font-family:Consolas,'Courier New',monospace;font-size:%s;font-weight:700;letter-spacing:%s;color:%s;background:#ffffff;border:1px solid #c5d5e4;border-radius:8px;padding:14px 16px;text-align:center;-webkit-user-select:all;user-select:all;word-break:break-all">%s</div>
				</td>
			</tr>`,
				border, colorMuted, htmlEscape(r.Label), codeSize, codeTrack, colorNavy, htmlEscape(r.Value))
			continue
		}
		valColor := colorText
		if r.Highlight {
			valColor = colorNavy
		}
		fmt.Fprintf(&rowsHTML, `<tr>
				<td style="padding:10px 0;color:%s;font-size:13px;width:42%%;%s">%s</td>
				<td style="padding:10px 0;color:%s;font-size:13px;font-weight:700;text-align:right;%s">%s</td>
			</tr>`,
			colorMuted, border, htmlEscape(r.Label),
			valColor, border, htmlEscape(r.Value))
	}

	section := ""
	if sectionTitle != "" && rowsHTML.Len() > 0 {
		section = fmt.Sprintf(`
			<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f8fafc;border:1px solid #d7e3ee;border-radius:8px;margin:22px 0 4px">
				<tr><td style="padding:20px 22px">
					<div style="font-size:11px;font-weight:700;letter-spacing:.1em;color:%s;margin-bottom:10px;text-align:center">%s</div>
					<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">%s</table>
				</td></tr>
			</table>`, colorNavy, htmlEscape(strings.ToUpper(sectionTitle)), rowsHTML.String())
	}

	callouts := ""
	if strings.TrimSpace(nextSteps) != "" {
		callouts += fmt.Sprintf(`
			<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin-top:16px">
				<tr><td style="background:#e8f1f8;border-left:4px solid %s;padding:12px 16px;border-radius:4px;font-size:13px;color:%s;line-height:1.5">
					<strong>Next steps:</strong> %s
				</td></tr>
			</table>`, colorNavy, colorNavy, nextSteps)
	}
	if strings.TrimSpace(security) != "" {
		callouts += fmt.Sprintf(`
			<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin-top:10px">
				<tr><td style="background:#fff7ed;border-left:4px solid %s;padding:12px 16px;border-radius:4px;font-size:13px;color:#9a3412;line-height:1.5">
					<strong>Security:</strong> %s
				</td></tr>
			</table>`, colorAccent, security)
	}

	cta := ""
	if ctaURL != "" {
		label := ctaLabel
		if label == "" {
			label = "Open DFMS"
		}
		safe := htmlEscape(ctaURL)
		cta = fmt.Sprintf(`
			<table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 8px">
				<tr><td style="background:%s;border-radius:8px">
					<a href="%s" style="display:inline-block;color:#ffffff;text-decoration:none;padding:12px 22px;font-weight:700;font-size:14px">%s</a>
				</td></tr>
			</table>`,
			colorNavy, safe, htmlEscape(label))
	} else {
		cta = fmt.Sprintf(`<p style="color:%s;font-size:12px;margin:20px 0 0">Sign in to TIPER DFMS (ask your administrator for the address if you do not have it).</p>`, colorMuted)
	}

	footerAddr := brand.Address
	if footerAddr == "" {
		footerAddr = "Tanzania"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"></head>
<body style="margin:0;padding:0;background:%s">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;padding:24px 0">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background:#ffffff;border:1px solid #d7e3ee;font-family:'Source Sans 3',Arial,Helvetica,sans-serif;color:%s">
	%s
	<tr><td style="padding:22px 32px 8px">
		<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
			<tr>
				<td valign="middle">%s</td>
				<td valign="middle" align="right">%s</td>
			</tr>
		</table>
	</td></tr>
	<tr><td style="padding:8px 32px 28px">
		<p style="font-size:15px;line-height:1.5;margin:16px 0 8px;color:%s">%s</p>
		<p style="font-size:14px;line-height:1.6;color:%s;margin:0 0 8px">%s</p>
		%s
		%s
		%s
	</td></tr>
	<tr><td style="height:3px;background:%s;font-size:0;line-height:0">&nbsp;</td></tr>
	<tr><td style="padding:20px 32px 22px;background:%s">
		<div style="font-size:13px;font-weight:700;color:#ffffff">%s</div>
		<div style="font-size:12px;color:#c5d5e4;line-height:1.6;margin-top:6px">%s</div>
	</td></tr>
</table>
</td></tr>
</table>
</body></html>`,
		colorCanvas, colorCanvas, colorText,
		heroHTML,
		logoHTML, badgeHTML,
		colorText, greeting, colorMuted, intro,
		section, callouts, cta,
		colorAccent, colorNavy, name, footerAddr,
	)
}
