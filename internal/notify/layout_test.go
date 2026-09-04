package notify

import (
	"strings"
	"testing"

	"dfms/pkg/types"
)

func TestRenderCorporateEmail_ContainsEnterpriseLayout(t *testing.T) {
	html := renderCorporateEmail(
		EmailBrand{
			Name:      "TIPER Test",
			Address:   "Dar es Salaam, Tanzania",
			LogoURL:   "cid:tiper-logo.png",
			HeroURL:   "cid:tiper-email-hero.jpg",
			PortalURL: "https://dfms.example.com",
		},
		"Action required",
		types.EmailBadgeAction,
		"Hello <strong>Approver</strong>,",
		"A gantry loading request is awaiting your action.",
		"Document summary",
		[]kv{
			{Label: "Customer", Value: "PUMA Energy"},
			{Label: "Document", Value: "GLR:0526-0143", Highlight: true},
			{Label: "Temporary password", Value: "Ab1!xyZ9#q", Secret: true},
		},
		"Open DFMS to review.",
		"If you did not expect this, contact support.",
		"Open DFMS",
		"https://dfms.example.com/approvals",
	)
	for _, want := range []string{
		"ACTION REQUIRED",
		"DOCUMENT SUMMARY",
		"PUMA Energy",
		"GLR:0526-0143",
		"user-select:all",
		"Courier New",
		"Next steps:",
		"Security:",
		"TIPER Test",
		"Dar es Salaam, Tanzania",
		"cid:tiper-logo.png",
		"cid:tiper-email-hero.jpg",
		`height="120"`,
		"Kigamboni bonded warehouse",
		"#033860",
		"#dc295c",
		"TIPER DFMS",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("email HTML missing %q", want)
		}
	}
	if strings.Contains(html, "Or open:") {
		t.Error("CTA should be the button only — no fallback URL line")
	}
}

func TestApplyAssetURLs_UsesInlineLogo(t *testing.T) {
	var b EmailBrand
	applyAssetURLs(&b)
	if findEmailAsset(emailLogoCID) == "" {
		if b.LogoURL != "" {
			t.Fatalf("no logo file: expected empty LogoURL, got %q", b.LogoURL)
		}
	} else if b.LogoURL != "cid:"+emailLogoCID {
		t.Fatalf("LogoURL=%q", b.LogoURL)
	}
	if findEmailAsset(emailHeroCID) == "" {
		if b.HeroURL != "" {
			t.Fatalf("no hero file: expected empty HeroURL, got %q", b.HeroURL)
		}
		return
	}
	if b.HeroURL != "cid:"+emailHeroCID {
		t.Fatalf("HeroURL=%q", b.HeroURL)
	}
}

func TestSMSName_UsesEmailFromName(t *testing.T) {
	if got := (EmailBrand{FromName: "TIPER Operations"}).smsName(); got != "TIPER Operations" {
		t.Fatalf("got %q", got)
	}
	if (EmailBrand{}).smsName() != "TIPER DFMS" {
		t.Fatalf("empty fromName should default")
	}
}

func TestDashEmpty(t *testing.T) {
	if dash("") != "—" {
		t.Fatalf("empty should be em dash")
	}
	if dash("  GLR-1 ") != "GLR-1" {
		t.Fatalf("trim failed: %q", dash("  GLR-1 "))
	}
}
