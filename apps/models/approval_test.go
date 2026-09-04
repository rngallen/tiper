package models

import (
	"testing"
	"time"
)

func TestApprovalTrailValueAndScan(t *testing.T) {
	at := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	in := ApprovalTrail{
		{ActedAt: at, ActType: "initiate", ActName: "Initiated", UserName: "Jane Doe", Title: "Stock Accountant"},
		{ActedAt: at.Add(time.Hour), ActType: "approve", ActName: "Approved", UserName: "John Smith", Title: "CFO", Comment: "OK"},
	}
	v, err := in.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok || s == "" || s == "[]" {
		t.Fatalf("Value() want JSON array, got %#v", v)
	}

	var out ApprovalTrail
	if err := out.Scan(s); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[1].Comment != "OK" || out[0].UserName != "Jane Doe" {
		t.Fatalf("scan: %+v", out)
	}

	rows := out.AsILR()
	if len(rows) != 2 || rows[0].ApprovedBy != "Jane Doe" || rows[1].Title != "CFO" {
		t.Fatalf("AsILR: %+v", rows)
	}

	var empty ApprovalTrail
	if err := empty.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if empty == nil {
		t.Fatal("Scan(nil) should yield empty slice, not nil")
	}
}

func TestApprovalTrailNilValue(t *testing.T) {
	var tnil ApprovalTrail
	v, err := tnil.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != "[]" {
		t.Fatalf("nil Value() = %#v", v)
	}
}
