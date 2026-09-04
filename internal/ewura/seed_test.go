package ewura

import (
	"encoding/json"
	"testing"
)

func TestLicenseDTO(t *testing.T) {
	raw := []byte(`[{
		"licenseNumber": "PRL-2023-288",
		"licensee": "Kinjekitile Filling Station",
		"licenseClass": "Petroleum Retail",
		"licenseType": "Petroleum Retail",
		"sector": "Petroleum",
		"regionName": "MTWARA",
		"issueDate": "2023-09-27",
		"expiryDate": "2028-09-26",
		"tinNumber": "121906104"
	}]`)
	var rows []licenseDTO
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	row, ok := rows[0].toRow()
	if !ok {
		t.Fatal("expected a license row")
	}
	if row.LicenseNumber != "PRL-2023-288" {
		t.Fatalf("number: %s", row.LicenseNumber)
	}
	if row.Licensee != "Kinjekitile Filling Station" {
		t.Fatalf("licensee: %s", row.Licensee)
	}
	if row.LicenseClass != "Petroleum Retail" {
		t.Fatalf("class: %s", row.LicenseClass)
	}
	if row.IssueDate == nil || row.ExpiryDate == nil {
		t.Fatal("expected parsed dates")
	}
	if !row.IsActive {
		t.Fatal("future expiry should stay active")
	}
}

func TestLicenseDTOExpiredIsInactive(t *testing.T) {
	row, ok := licenseDTO{
		LicenseNumber: "PRL-2010-001",
		Licensee:      "Expired OMC",
		ExpiryDate:    "2010-01-01",
	}.toRow()
	if !ok {
		t.Fatal("expected a license row")
	}
	if row.IsActive {
		t.Fatal("expired license must be inactive")
	}
}

func TestFormatHTTPError(t *testing.T) {
	t.Parallel()
	html := []byte("<html><body><div class=\"title\">Be right back.</div></body></html>")
	err := formatHTTPError(500, html)
	if err == nil || err.Error() != "ewura register unavailable (http 500)" {
		t.Fatalf("html 500: %v", err)
	}
	if err = formatHTTPError(404, nil); err == nil || err.Error() != "ewura http 404" {
		t.Fatalf("empty 404: %v", err)
	}
	if err = formatHTTPError(403, []byte("license register disabled")); err == nil ||
		err.Error() != "ewura http 403: license register disabled" {
		t.Fatalf("plain 403: %v", err)
	}
}
