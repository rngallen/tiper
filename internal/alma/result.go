package alma

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// Result is one SAP3R completion file (alma-files AlmaOrder).
type Result struct {
	BatchNumber  string
	OrderNumber  string
	LoadingDate  time.Time
	CustomerCode string
	DriverName   string
	Transporter  string
	Products     []ProductLine
}

// ProductLine is one 01 record.
type ProductLine struct {
	ProductCode      string
	LoadedVolume     float64
	LoadedVolumeAt20 float64
	Temperature      float64
	Density          float64
}

// ParseSAP3R reads a completion file. Field positions are the ATLAS NEO contract
// and must not be guessed — they match alma-files/watcher/reader.go.
func ParseSAP3R(r io.Reader) (Result, error) {
	var out Result
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 8*1024), 64*1024)
	for sc.Scan() {
		text := sc.Text()
		if len(text) < 2 {
			return out, fmt.Errorf("line too short")
		}
		switch text[:2] {
		case "00":
			if len(text) < 64 {
				return out, fmt.Errorf("header too short")
			}
			if text[59:64] != FileTypeIn {
				return out, fmt.Errorf("invalid file type %q", text[59:64])
			}
			out.BatchNumber = text[10:13]
		case "01":
			if len(text) < 1400 {
				return out, fmt.Errorf("detail line shorter than 1400 characters")
			}
			cust := strings.TrimSpace(text[2:7])
			if cust == "" {
				return out, fmt.Errorf("customer code missing")
			}
			out.CustomerCode = cust
			loadingDate, err := time.Parse("150402012006", text[63:75])
			if err != nil {
				return out, fmt.Errorf("loading date: %w", err)
			}
			out.LoadingDate = loadingDate.Add(-3 * time.Hour).In(time.Local)
			code := strings.TrimSpace(text[81:85])
			if code != MogasNumber && code != AgoNumber {
				return out, fmt.Errorf("invalid product code %s", code)
			}
			obs, err := strconv.ParseFloat(strings.TrimSpace(text[95:105]), 64)
			if err != nil {
				return out, fmt.Errorf("observed volume: %w", err)
			}
			std, err := strconv.ParseFloat(strings.TrimSpace(text[105:115]), 64)
			if err != nil {
				return out, fmt.Errorf("standard volume: %w", err)
			}
			temp, err := strconv.ParseFloat(strings.TrimSpace(text[125:129]), 64)
			if err != nil {
				return out, fmt.Errorf("temperature: %w", err)
			}
			dens, err := strconv.ParseFloat(strings.TrimSpace(text[130:137]), 64)
			if err != nil {
				return out, fmt.Errorf("density: %w", err)
			}
			out.Transporter = strings.TrimSpace(text[410:430])
			out.OrderNumber = strings.TrimSpace(text[435:445])
			out.DriverName = strings.TrimSpace(text[1380:1400])
			out.Products = append(out.Products, ProductLine{
				ProductCode:      code,
				LoadedVolume:     obs,
				LoadedVolumeAt20: std,
				Temperature:      roundN(temp/10, 2),
				Density:          roundN(dens/10_000, 4),
			})
		case "02":
			if len(text) < 64 {
				return out, fmt.Errorf("footer too short")
			}
			if text[10:13] != out.BatchNumber {
				return out, fmt.Errorf("header batch %s and footer batch %s differ", out.BatchNumber, text[10:13])
			}
			if text[59:64] != FileTypeIn {
				return out, fmt.Errorf("invalid footer type %q", text[59:64])
			}
		default:
			return out, fmt.Errorf("invalid line prefix %q", text[:2])
		}
	}
	if err := sc.Err(); err != nil {
		return out, err
	}
	if len(out.Products) == 0 || len(out.Products) > 2 {
		return out, fmt.Errorf("expected 1 or 2 product lines, got %d", len(out.Products))
	}
	if out.OrderNumber == "" {
		return out, fmt.Errorf("order number missing")
	}
	return out, nil
}

func roundN(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
