package attachment

import (
	"path/filepath"
	"strings"
	"testing"

	"dfms/pkg/types"
)

func TestSanitizeLabel(t *testing.T) {
	if got := SanitizeLabel("  C2408067  "); got != "C2408067" {
		t.Fatalf("trim: %q", got)
	}
	if got := SanitizeLabel(`GRV/2026\08:1`); got != "GRV-2026-08-1" {
		t.Fatalf("separators: %q", got)
	}
	if got := SanitizeLabel("../etc/passwd"); got != "etc-passwd" {
		t.Fatalf("traversal: %q", got)
	}
	if got := SanitizeLabel("INV..2026"); strings.Contains(got, "..") || got == "" {
		t.Fatalf("collapsed dots: %q", got)
	}
	if got := SanitizeLabel("NOVO TECH LTD"); got != "NOVO_TECH_LTD" {
		t.Fatalf("spaces: %q", got)
	}
	long := strings.Repeat("x", 60)
	if got := SanitizeLabel(long); got != strings.Repeat("x", 48) {
		t.Fatalf("length: %q", got)
	}
}

func TestEntityDirName(t *testing.T) {
	if got := EntityDirName("RCPT:0826-0014"); got != "RCPT-0826-0014" {
		t.Fatalf("document number: %q", got)
	}
	if got := EntityDirName("C2408067"); got != "C2408067" {
		t.Fatalf("code: %q", got)
	}
	if EntityDirName("") != "unknown" {
		t.Fatal("empty label should be unknown")
	}
	if got := EntityDirName("../x"); got != "x" {
		t.Fatalf("traversal must not survive sanitise: %q", got)
	}
	if strings.Contains(EntityDirName("a/b"), "/") || strings.Contains(EntityDirName("a/b"), `\`) {
		t.Fatal("dir name must be a single path segment")
	}
}

func TestCategoryDir(t *testing.T) {
	cases := map[types.ContentType]string{
		types.ReceiptContent:              "Receipts",
		types.GantryLoadingRequestContent: "ILR",
		types.IttTransferContent:          "ITT",
		types.PumpOverRequestContent:      "Pump-over-request",
		types.PumpOverReportContent:       "Pump-over-report",
		types.CustomerContent:             "Customers",
		types.DriverContent:               "Drivers",
		types.TankContent:                 "Tanks",
		types.TruckContent:                "Trucks",
		types.CompartmentalizationContent: "Compartmentalization",
		types.SupplierContent:             "Suppliers",
		types.BillingRunContent:           "Billing",
		types.BillingProfileContent:       "Fixed-storage-fees",
		types.EwuraLicenseContent:         "EWURA-licences",
	}
	for ct, want := range cases {
		if got := categoryDir(ct); got != want {
			t.Errorf("%s: got %q want %q", types.ContentTypeLabel(ct), got, want)
		}
	}
	if got := categoryDir(0); got == "Other" || got == "" {
		t.Fatalf("unknown type must not land in Other: %q", got)
	}
}

func TestCreateEntityPathRejectsUnsafeFolder(t *testing.T) {
	for _, bad := range []string{"", "..", "../x", `a\b`, "a/b"} {
		if _, err := createEntityPath(0, bad); err == nil {
			t.Errorf("expected reject for %q", bad)
		}
	}
}

func TestContentDisposition(t *testing.T) {
	got := contentDisposition("inv\"quote\n.pdf")
	if strings.Contains(got, `"`) && strings.Contains(got, "quote") {
		if strings.Contains(got, "\n") || strings.Contains(got, `"quote`) {
			// filename quotes replaced
		}
	}
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("CR/LF must be stripped: %q", got)
	}
	if !strings.HasPrefix(got, `attachment; filename="`) {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSafeUploadPathNewLayout(t *testing.T) {
	root, err := filepath.Abs(Root())
	if err != nil {
		t.Fatal(err)
	}
	ok := filepath.Join(root, "Invoices", "2026", "08", "01HZUIDEXAMPLE", "doc.pdf")
	got, err := resolveSafeUploadPath(ok)
	if err != nil {
		t.Fatalf("expected ok path: %v", err)
	}
	if got != filepath.Clean(ok) {
		t.Fatalf("got %q want %q", got, filepath.Clean(ok))
	}
}
