package export

import (
	"bytes"
	"strings"
	"testing"
)

func TestPdfSafeMiddleDot(t *testing.T) {
	got := pdfSafe("Plot 1 · Kigamboni · Tel")
	if strings.Contains(got, "·") || strings.Contains(got, "Â") {
		t.Fatalf("middle-dot leaked: %q", got)
	}
	if !strings.Contains(got, "|") {
		t.Fatalf("expected ASCII separator, got %q", got)
	}
}

func TestRenderILR(t *testing.T) {
	pdf, err := RenderILR(ILRDoc{
		Number:        "08-2026/255",
		Status:        "approved",
		Date:          "26/08/2026",
		Description:   "Loading order request for OLASITI 04 truck with a longer note that must wrap across the full page width instead of colliding with the value cell.",
		Customer:      "OLASITI INVESTMENT COMPANY LIMITED",
		ProductStatus: "Local",
		Contract:      true,
		LoadingOrder:  true,
		CompanyName:   "Tanzania International Petroleum Reserves Limited",
		Address:       "Kigamboni Depot Site - Plot 1",
		Address2:      "Kigamboni Industrial Area",
		Postal:        "P.O. Box 2608",
		City:          "Dar es Salaam",
		Country:       "Tanzania",
		Phone:         "+255 (0) 22 5511 500",
		Email:         "info@tiper.co.tz",
		Website:       "https://tiper.co.tz",
		TIN:           "100-103-362",
		VRN:           "10-00115-Z",
		ISO:           "ISO 9001: 2015 and ISO 45001: 2018",
		Products:      [][2]string{{"Automotive Gas Oil", "104,500.00"}},
		Charges: [][3]string{
			{"Storage Debt", "TZS", "0.00"},
			{"Storage Debt", "USD", "0.00"},
		},
		VesselHeads: []string{"Date", "Vessel name", "F. Hold", "Automotive Gas Oil"},
		Vessels:     [][]string{{"20/08/2026", "SANTHIA", "No", "104,500.00"}},
		StockHeads:  []string{"Product", "Total balance", "Volume under F.Hold", "Free volume", "Free volume after GLR"},
		Stock:       [][]string{{"Automotive Gas Oil", "801,550.00", "635,598.00", "165,952.00", "61,452.00"}},
		ApprovedQty: "104,500.00",
		TruckHeads:  []string{"S/N", "Order no", "Truck", "Driver name", "Licence / passport", "Automotive Gas Oil"},
		Trucks:      [][]string{{"1", "GLO-1", "T 123 ABC / T 456 DEF", "Juma Ali", "DL-99", "104,500.00"}},
		Approvals:   [][]string{{"26/08/2026 10:00", "Jane Doe", "Stock Accountant", "Approved"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if !bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
	if bytes.Contains(raw, []byte{0xC2, 0xB7}) {
		t.Fatal("raw UTF-8 middle-dot leaked into the PDF")
	}
	if bytes.Contains(raw, []byte("Â")) {
		t.Fatal("mojibake marker leaked into the PDF")
	}
}

func TestLetterheadLines(t *testing.T) {
	lines := Letterhead{
		CompanyName: "Tanzania International Petroleum Reserves Limited",
		Address:     "Kigamboni Depot Site - Plot 1",
		City:        "Dar es Salaam",
		Country:     "Tanzania",
		Phone:       "+255 (0) 22 5511 500",
		Email:       "info@tiper.co.tz",
		TIN:         "100-103-362",
		VRN:         "10-00115-Z",
		ISO:         "ISO 9001: 2015",
	}.Lines()
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		DefaultCompanyName,
		"Kigamboni Depot Site - Plot 1",
		"Dar es Salaam",
		"Tel +255 (0) 22 5511 500",
		"VRN 10-00115-Z",
		"TIN 100-103-362",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}

func TestRenderTableLetterhead(t *testing.T) {
	pdf, err := RenderTable("Stock position", Letterhead{
		CompanyName: "Tanzania International Petroleum Reserves Limited",
		Address:     "Kigamboni Depot Site - Plot 1",
		City:        "Dar es Salaam",
		Country:     "Tanzania",
		Phone:       "+255 22 5511 500",
		TIN:         "100-103-362",
		VRN:         "10-00115-Z",
	}, []string{"Customer", "Product", "Qty"}, [][]string{{"OMC", "AGO", "1,000.00"}})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
}
