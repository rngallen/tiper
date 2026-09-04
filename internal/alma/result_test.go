package alma

import (
	"strings"
	"testing"
)

func TestParseSAP3R_singleAGO(t *testing.T) {
	header := padRight("00        001"+spaces(46)+FileTypeIn, 64)
	detail := make([]byte, 1400)
	for i := range detail {
		detail[i] = ' '
	}
	copy(detail[0:], "01")
	copy(detail[2:], "CUST1")
	copy(detail[63:], "143022082025")
	copy(detail[81:], "1002")
	copy(detail[95:], "  8500.500")
	copy(detail[105:], "  8480.250")
	copy(detail[125:], "0215")
	copy(detail[130:], "0007500")
	copy(detail[410:], "HAULIER")
	copy(detail[435:], "GLO0001234")
	copy(detail[1380:], "JOHN DRIVER")
	footer := padRight("02        001"+spaces(46)+FileTypeIn, 64)
	body := header + "\n" + string(detail) + "\n" + footer + "\n"
	got, err := ParseSAP3R(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderNumber != "GLO0001234" || got.CustomerCode != "CUST1" {
		t.Fatalf("%+v", got)
	}
	if len(got.Products) != 1 || got.Products[0].ProductCode != AgoNumber {
		t.Fatalf("products %+v", got.Products)
	}
	if got.Products[0].LoadedVolumeAt20 != 8480.250 {
		t.Fatalf("std vol %v", got.Products[0].LoadedVolumeAt20)
	}
	if got.Products[0].Temperature != 21.5 {
		t.Fatalf("temp %v", got.Products[0].Temperature)
	}
	if got.Products[0].Density != 0.75 {
		t.Fatalf("density %v", got.Products[0].Density)
	}
}
