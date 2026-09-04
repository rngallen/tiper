package reports

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestFmtProduct(t *testing.T) {
	if got := fmtProduct("AGO", "Gasoil"); got != "AGO — Gasoil" {
		t.Fatalf("got %q", got)
	}
	if got := fmtProduct("AGO", "AGO"); got != "AGO" {
		t.Fatalf("same code/name: %q", got)
	}
	if got := fmtProduct("", "Gasoil"); got != "Gasoil" {
		t.Fatalf("name only: %q", got)
	}
}

func TestWithSerial(t *testing.T) {
	got := withSerial([][]string{{"A"}, {"B"}})
	if len(got) != 2 || got[0][0] != "1" || got[1][0] != "2" || got[0][1] != "A" {
		t.Fatalf("got %#v", got)
	}
}

func TestFormatCell(t *testing.T) {
	if formatCell(true) != "Yes" || formatCell(false) != "No" {
		t.Fatal("bool")
	}
	d := decimal.RequireFromString("138060.000")
	if got := formatCell(d); got != "138,060.00" {
		t.Fatalf("decimal: %s", got)
	}
	if got := formatCell([]byte("13806000")); got != "13,806,000.00" {
		t.Fatalf("bytes: %s", got)
	}
	day := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if got := formatCell(day); got != "02/09/2026" {
		t.Fatalf("date: %s", got)
	}
	if got := fmtDateOnly(&day); got != "02/09/2026" {
		t.Fatalf("date only: %s", got)
	}
}

func TestTableFromSliceOrder(t *testing.T) {
	type row struct {
		Code     string
		Name     string
		IsActive bool
	}
	headers, data, ok := tableFromSlice([]row{{"1008", "AFROIL", true}})
	if !ok {
		t.Fatal("expected struct slice")
	}
	if len(headers) != 3 || headers[0] != "Code" || headers[1] != "Name" || headers[2] != "IsActive" {
		t.Fatalf("headers %#v", headers)
	}
	if data[0][2] != "Yes" {
		t.Fatalf("active cell %q", data[0][2])
	}
	human := humanizeHeaders(headers)
	if human[2] != "Active" || human[0] != "Code" {
		t.Fatalf("human %#v", human)
	}
}

func TestTitled(t *testing.T) {
	if got := titled("Tanks", "Active"); got != "Tanks — Active" {
		t.Fatalf("got %q", got)
	}
	if got := titled("Tanks", "", ""); got != "Tanks" {
		t.Fatalf("empty notes: %q", got)
	}
}

func TestParseQty(t *testing.T) {
	if got := fmtQtyStr("138060.000"); got != "138,060.00" {
		t.Fatalf("got %s", got)
	}
	if !parseQty("").IsZero() || !parseQty("x").IsZero() {
		t.Fatal("empty/invalid")
	}
}

func TestParseClassList(t *testing.T) {
	out := parseClassList("PWL, PSBL,PWL, ")
	if len(out) != 2 || out[0] != "PWL" || out[1] != "PSBL" {
		t.Fatalf("got %#v", out)
	}
}
