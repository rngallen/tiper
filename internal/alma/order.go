// Package alma implements the ATLAS NEO gantry file contract used at TIPER.
//
// Outbound orders (this file) are fixed-width SAP3C .dat files dropped in
// {FilePath}/In — the same layout tiper-loadings wrote. Inbound completions
// are SAP3R files in {FilePath}/Alma/Files — the same layout alma-files read.
//
// DFMS writes and reads these files itself. There is no RabbitMQ hop.
package alma

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	FileTypeOut     = "SAP3C"
	FileTypeIn      = "SAP3R"
	CustomerID      = "10"
	LoadingType     = "PRED"
	OrderTypeNew    = "1"
	OrderTypeCancel = "2"
	MogasNumber     = "1001"
	AgoNumber       = "1002"
	WcfFactor       = 0.0011
	DataLineWidth   = 2000
)

// Order is the data needed to write one SAP3C file. No database types here so
// the layout can be tested with golden strings.
type Order struct {
	BatchNumber     string
	BatchDate       time.Time
	CustomerCode    string
	ProductNumber   string
	QuantityLtr     int
	ByProductNumber string
	ByProductLtr    int
	OrderDate       time.Time
	ExpirationDate  time.Time
	DocNumber       string
	TransporterName string
	DriverName      string
	Destination     string
	District        string
	HorsePlate      string
	TrailerOnePlate string
	TrailerTwoPlate string
	Canceled        bool
	Compartments    []Compartment
	ByCompartments  []Compartment
}

// Compartment is one 45-character block on an SAP3C 01 line.
type Compartment struct {
	TankPlate string
	Index     int
	Quantity  int
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padLeftDigits(v int, n int) string {
	s := fmt.Sprintf("%0*d", n, v)
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func spaces(n int) string { return strings.Repeat(" ", n) }

func truckTrailer(o Order) string {
	horse := strings.TrimSpace(o.HorsePlate)
	t1 := strings.TrimSpace(o.TrailerOnePlate)
	t2 := strings.TrimSpace(o.TrailerTwoPlate)
	var code string
	switch {
	case t2 == "":
		if horse == t1 || t1 == "" {
			code = horse
		} else {
			code = horse + " " + t1
		}
	case horse == t1 || t1 == "":
		code = horse + " " + t2
	default:
		code = t1 + " " + t2
	}
	return padRight(code, 20)
}

func appendCompartments(buf string, lines []Compartment) string {
	for _, c := range lines {
		buf += padRight(c.TankPlate, 10) + padLeftDigits(c.Index, 2) + padLeftDigits(c.Quantity, 5) + spaces(28)
	}
	return buf
}

func dataLine(o Order, productNumber string, qty int, lines []Compartment) string {
	orderType := OrderTypeNew
	if o.Canceled {
		orderType = OrderTypeCancel
	}
	cust := strings.TrimSpace(o.CustomerCode)
	validity := o.OrderDate.Format("02012006") + o.ExpirationDate.Format("02012006")
	dest := padRight(strings.TrimSpace(o.Destination)+" "+strings.TrimSpace(o.District), 182)
	orderNo := padRight(o.DocNumber, 10)
	s := "01" + strings.Repeat(cust, 6) + spaces(15) + validity + spaces(18) +
		productNumber + CustomerID + "  " + cust + orderType + padLeftDigits(qty, 10) +
		spaces(32) + padRight(cust, 20) + spaces(9) + truckTrailer(o) + spaces(22) +
		LoadingType + spaces(198) + padRight(o.TransporterName, 20) + spaces(5) +
		orderNo + spaces(42) + dest + spaces(701) + orderNo + padRight(o.DriverName, 20)
	s = appendCompartments(s, lines)
	if len(s) > DataLineWidth {
		s = s[:DataLineWidth]
	}
	return padRight(s, DataLineWidth)
}

// BuildSAP3C returns the full file body (header, 01 lines, footer).
func BuildSAP3C(o Order) string {
	batchDate := o.BatchDate.Format("02012006")
	header := "00" + spaces(8) + o.BatchNumber + batchDate + spaces(38) + FileTypeOut + "\n"
	body := dataLine(o, o.ProductNumber, o.QuantityLtr, o.Compartments) + "\n"
	lines := 1
	total := o.QuantityLtr + o.ByProductLtr
	if o.ByProductNumber != "" && o.ByProductLtr > 0 {
		body += dataLine(o, o.ByProductNumber, o.ByProductLtr, o.ByCompartments) + "\n"
		lines = 2
	}
	footer := "02" + spaces(8) + o.BatchNumber + batchDate +
		fmt.Sprintf("%08d", lines) + padLeftDigits(total, 15) + spaces(15) + FileTypeOut
	return header + body + footer
}

// NewFileName matches tiper-loadings: SAGE{DDMMYYYY}T{HHMMSS}{nanos}Z.dat
func NewFileName(at time.Time) string {
	now := at.Local().Format("02012006T150405.999999999Z")
	now = strings.ReplaceAll(now, ".", "")
	return fmt.Sprintf("SAGE%s.dat", now)
}

// AlmaProductNumber is the Sage/ATLAS item number. Code is the same value
// used at the gantry (1001 PMS, 1002 AGO). Legacy trade names still map.
func AlmaProductNumber(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "PMS", "MOGAS":
		return MogasNumber
	case "AGO":
		return AgoNumber
	default:
		return strings.TrimSpace(code)
	}
}

// Litres converts cubic metres (ledger) to integer litres for ALMA.
func Litres(m3 decimal.Decimal) int {
	return int(m3.Mul(decimal.NewFromInt(1000)).Round(0).IntPart())
}

// CubicMetres converts ALMA litres (or observed volume) back to m³.
func CubicMetres(v float64) decimal.Decimal {
	return decimal.NewFromFloat(v).Div(decimal.NewFromInt(1000))
}

func BatchCode(id uint) string {
	return fmt.Sprintf("%03d", id%1000)
}
