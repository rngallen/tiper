package alma

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSAP3C_singleProduct(t *testing.T) {
	o := Order{
		BatchNumber:     "001",
		BatchDate:       time.Date(2025, 11, 22, 0, 0, 0, 0, time.UTC),
		CustomerCode:    "CUST1",
		ProductNumber:   "1002",
		QuantityLtr:     4000,
		OrderDate:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DocNumber:       "GLO-00001",
		TransporterName: "Haulier Ltd",
		DriverName:      "John Driver",
		Destination:     "Dar es Salaam",
		District:        "Ilala",
		HorsePlate:      "T123ABC",
		TrailerOnePlate: "T123ABC",
		Compartments:    []Compartment{{TankPlate: "T123ABC", Index: 1, Quantity: 2500}, {TankPlate: "T123ABC", Index: 2, Quantity: 1500}},
	}
	body := BuildSAP3C(o)
	lines := strings.Split(body, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + data + footer, got %d lines", len(lines))
	}
	if !strings.HasPrefix(lines[0], "00") || !strings.HasSuffix(lines[0], FileTypeOut) {
		t.Fatalf("header: %q", lines[0])
	}
	if len(lines[1]) != DataLineWidth {
		t.Fatalf("01 line width %d want %d", len(lines[1]), DataLineWidth)
	}
	if !strings.Contains(lines[1], "1002") || !strings.Contains(lines[1], "0000004000") {
		t.Fatalf("product/qty missing: %s", lines[1][:120])
	}
	if !strings.HasPrefix(lines[2], "02") || !strings.Contains(lines[2], FileTypeOut) {
		t.Fatalf("footer: %q", lines[2])
	}
}

func TestAlmaProductNumber(t *testing.T) {
	if AlmaProductNumber("AGO") != AgoNumber {
		t.Fatal("AGO")
	}
	if AlmaProductNumber("PMS") != MogasNumber {
		t.Fatal("PMS")
	}
	if AlmaProductNumber("1002") != "1002" {
		t.Fatal("item number")
	}
}
