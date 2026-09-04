package export

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v3"
)

type ILRDoc struct {
	Number        string
	Status        string
	Date          string
	Description   string
	Customer      string
	ProductStatus string
	Contract      bool
	LoadingOrder  bool
	CompanyName   string
	Address       string
	Address2      string
	City          string
	Postal        string
	Country       string
	Phone         string
	Email         string
	Website       string
	TIN           string
	VRN           string
	ISO           string
	LogoPath      string
	Products      [][2]string
	Charges       [][3]string
	VesselHeads   []string
	Vessels       [][]string
	StockHeads    []string
	Stock         [][]string
	ApprovedQty   string
	TruckHeads    []string
	Trucks        [][]string
	Approvals     [][]string
	VerifyURL     string
}

func ILRPDF(c fiber.Ctx, doc ILRDoc) error {
	pdf, err := RenderILR(doc)
	if err != nil {
		return err
	}
	raw, err := pdfBytes(pdf)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("ilr_%s.pdf", time.Now().Format("02012006_150405"))
	return SendPDF(c, name, raw, false)
}

// RenderILR builds the gantry loading request PDF. Helvetica is WinAnsi: every
// string is mapped through pdfSafe + UnicodeTranslator so separators and
// names do not print as "Â·".
func RenderILR(doc ILRDoc) (*fpdf.Fpdf, error) {
	head := resolveLetterhead(doc.Head())
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Gantry Loading Request "+doc.Number, false)
	pdf.SetAuthor(head.DisplayName(), false)
	pdf.SetMargins(14, 0, 14)
	pdf.SetAutoPageBreak(true, 22)
	tr := newPDFText(pdf)
	attachEnterpriseChrome(pdf, tr, head, time.Now(), nil)
	pdf.AddPage()

	writeCenteredTitle(pdf, tr, "Gantry Loading Request", doc.Status, doc.Number)

	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	contentW := pageW - left - right
	gutter := 3.0
	col := (contentW - gutter) / 2
	y := pdf.GetY()

	leftBottom := writeOrderColumn(pdf, tr, left, y, col, doc)
	rightBottom := writeChargesColumn(pdf, tr, left+col+gutter, y, col, doc)
	pdf.SetY(maxY(leftBottom, rightBottom) + 2)

	if strings.TrimSpace(doc.Description) != "" {
		sectionTitle(pdf, tr, left, pdf.GetY(), contentW, "Description")
		pdf.Ln(6.5)
		writeWrappedBox(pdf, tr, left, contentW, strings.TrimSpace(doc.Description))
		pdf.Ln(2)
	}

	sectionTitle(pdf, tr, left, pdf.GetY(), contentW, "Vessel details")
	pdf.Ln(6.5)
	if len(doc.VesselHeads) == 0 {
		doc.VesselHeads = []string{"Date", "Vessel", "F. Hold", "Quantity (Ltrs)"}
	}
	bandedTable(pdf, tr, doc.VesselHeads, doc.Vessels)

	pdf.Ln(3)
	sectionTitle(pdf, tr, left, pdf.GetY(), contentW, "Customer current stock position (Ltrs)")
	pdf.Ln(6.5)
	if len(doc.StockHeads) == 0 {
		doc.StockHeads = []string{"Product", "Total balance", "Volume under F.Hold", "Free volume", "Free volume after GLR"}
	}
	bandedTable(pdf, tr, doc.StockHeads, doc.Stock)
	if doc.ApprovedQty != "" {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(contentW, 6, tr("Approved quantity (Ltrs): "+doc.ApprovedQty), "", 1, "R", false, 0, "")
	}

	pdf.Ln(2)
	sectionTitle(pdf, tr, left, pdf.GetY(), contentW, "Loading instruction(s)")
	pdf.Ln(6.5)
	if len(doc.TruckHeads) == 0 {
		doc.TruckHeads = []string{"S/N", "Order no", "Truck", "Driver", "Licence", "Qty (Ltrs)"}
	}
	bandedTable(pdf, tr, doc.TruckHeads, doc.Trucks)

	pdf.Ln(3)
	sectionTitle(pdf, tr, left, pdf.GetY(), contentW, "Approvals  |  "+strings.ToUpper(first(doc.Status, "draft")))
	pdf.Ln(6.5)
	if len(doc.Approvals) == 0 {
		doc.Approvals = [][]string{{"-", "Not yet submitted", "", ""}}
	}
	bandedTable(pdf, tr, []string{"Approved on", "Approved by", "Approver's title", "Comment"}, doc.Approvals)

	writeScanToVerify(pdf, tr, doc.VerifyURL)

	if pdf.Err() {
		return nil, pdf.Error()
	}
	return pdf, nil
}

func writeOrderColumn(pdf *fpdf.Fpdf, tr func(string) string, x, y, w float64, doc ILRDoc) float64 {
	sectionTitle(pdf, tr, x, y, w, "Order details")
	pdf.SetY(y + 6.5)
	kv(pdf, tr, x, w, [][2]string{
		{"Order date", doc.Date},
		{"Customer", doc.Customer},
		{"Product status", doc.ProductStatus},
		{"Valid contract", yesNo(doc.Contract)},
		{"Loading order", yesNo(doc.LoadingOrder)},
	})
	pdf.SetX(x)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(3, 56, 96)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(w*0.62, 5.5, tr("Product"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(w*0.38, 5.5, tr("Quantity (Ltrs)"), "1", 1, "R", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)
	for _, p := range doc.Products {
		pdf.SetX(x)
		pdf.CellFormat(w*0.62, 5.5, tr(p[0]), "1", 0, "L", false, 0, "")
		pdf.CellFormat(w*0.38, 5.5, tr(p[1]), "1", 1, "R", false, 0, "")
	}
	return pdf.GetY()
}

func writeChargesColumn(pdf *fpdf.Fpdf, tr func(string) string, x, y, w float64, doc ILRDoc) float64 {
	sectionTitle(pdf, tr, x, y, w, "Customer outstandings")
	pdf.SetXY(x, y+6.5)
	heads := []string{"Charge", "Currency", "Amount"}
	charges := doc.Charges
	if len(charges) == 0 {
		charges = [][3]string{{"-", "", "0.00"}}
	}
	tableAt(pdf, tr, x, w, heads, triples(charges), []float64{0.50, 0.20, 0.30})
	return pdf.GetY()
}

func sectionTitle(pdf *fpdf.Fpdf, tr func(string) string, x, y, w float64, title string) {
	pdf.SetXY(x, y)
	pdf.SetFillColor(3, 56, 96)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(w, 6, tr("  "+title), "0", 0, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func kv(pdf *fpdf.Fpdf, tr func(string) string, x, w float64, rows [][2]string) {
	labelW := w * 0.38
	valueW := w - labelW
	const lineH = 4.2
	for _, r := range rows {
		label := tr(r[0])
		value := tr(r[1])
		pdf.SetFont("Helvetica", "", 8)
		lines := pdf.SplitText(value, valueW-2)
		if len(lines) == 0 {
			lines = []string{""}
		}
		h := float64(len(lines))*lineH + 1.4
		if h < 5.8 {
			h = 5.8
		}
		y := pdf.GetY()
		pdf.SetXY(x, y)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(labelW, h, label, "1", 0, "LM", false, 0, "")
		pdf.Rect(x+labelW, y, valueW, h, "D")
		pdf.SetXY(x+labelW+1, y+0.7)
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(valueW-2, lineH, value, "", "L", false)
		pdf.SetXY(x, y+h)
	}
}

func writeWrappedBox(pdf *fpdf.Fpdf, tr func(string) string, x, w float64, text string) {
	value := tr(text)
	pdf.SetFont("Helvetica", "", 8)
	const lineH = 4.2
	lines := pdf.SplitText(value, w-3)
	if len(lines) == 0 {
		lines = []string{""}
	}
	h := float64(len(lines))*lineH + 2
	if h < 7 {
		h = 7
	}
	y := pdf.GetY()
	pdf.Rect(x, y, w, h, "D")
	pdf.SetXY(x+1.5, y+1)
	pdf.MultiCell(w-3, lineH, value, "", "L", false)
	pdf.SetXY(x, y+h)
}

func bandedTable(pdf *fpdf.Fpdf, tr func(string) string, heads []string, rows [][]string) {
	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	w := pageW - left - right
	weights := tableWeights(heads)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(3, 56, 96)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range heads {
		pdf.CellFormat(w*weights[i], 6, tr(h), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
	fill := false
	for _, row := range rows {
		if pdf.GetY() > 270 {
			pdf.AddPage()
		}
		if fill {
			pdf.SetFillColor(241, 245, 249)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for i := range heads {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			align := "L"
			if i > 0 && looksNumber(v) {
				align = "R"
			}
			fitCell(pdf, tr, w*weights[i], 5.5, v, align, true)
		}
		pdf.Ln(-1)
		fill = !fill
	}
}

func tableAt(pdf *fpdf.Fpdf, tr func(string) string, x, w float64, heads []string, rows [][]string, weights []float64) {
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(3, 56, 96)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetX(x)
	for i, h := range heads {
		pdf.CellFormat(w*weights[i], 6, tr(h), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
	for _, row := range rows {
		pdf.SetX(x)
		for i := range heads {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			align := "L"
			if i == len(heads)-1 {
				align = "R"
			}
			fitCell(pdf, tr, w*weights[i], 5.5, v, align, false)
		}
		pdf.Ln(-1)
	}
}

func fitCell(pdf *fpdf.Fpdf, tr func(string) string, w, h float64, s, align string, fill bool) {
	writeFitCell(pdf, tr, w, h, s, align, fill, "")
}

func writeFitCell(pdf *fpdf.Fpdf, tr func(string) string, w, h float64, s, align string, fill bool, style string) {
	s = tr(s)
	pdf.SetFont("Helvetica", style, 7)
	if pdf.GetStringWidth(s) > w-1.6 {
		pdf.SetFont("Helvetica", style, 6)
	}
	if pdf.GetStringWidth(s) > w-1.6 {
		ellipsis := "..."
		for len(s) > 3 && pdf.GetStringWidth(s+ellipsis) > w-1.6 {
			s = strings.TrimRight(s[:len(s)-1], " ")
		}
		s += ellipsis
	}
	pdf.CellFormat(w, h, s, "1", 0, align, fill, 0, "")
}

func tableWeights(heads []string) []float64 {
	n := len(heads)
	w := make([]float64, n)
	if n == 0 {
		return w
	}
	sum := 0.0
	for i, h := range heads {
		switch {
		case i == 0 && n > 3:
			w[i] = 1.4
		case looksQtyHead(h):
			w[i] = 1.3
		default:
			w[i] = 1
		}
		sum += w[i]
	}
	for i := range w {
		w[i] /= sum
	}
	return w
}

func looksQtyHead(h string) bool {
	s := strings.ToLower(h)
	return strings.Contains(s, "qty") || strings.Contains(s, "quantity") ||
		strings.Contains(s, "ltrs") || strings.Contains(s, "balance") ||
		strings.Contains(s, "volume")
}

func triples(in [][3]string) [][]string {
	out := make([][]string, 0, len(in))
	for _, r := range in {
		out = append(out, []string{r[0], r[1], r[2]})
	}
	return out
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func first(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func prefix(p, v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return p + v
}

func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, "  |  ")
}

func letterheadAddress(head Letterhead) []string {
	var lines []string
	if s := joinPlain(head.Address, head.Address2); s != "" {
		lines = append(lines, s)
	}
	if s := joinPlain(joinPlain(head.Postal, head.City), head.Country); s != "" {
		lines = append(lines, s)
	}
	return lines
}

func joinPlain(parts ...string) string {
	var out []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, ", ")
}

func looksNumber(s string) bool {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return false
	}
	dot := 0
	for _, r := range s {
		if r == '.' {
			dot++
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return dot <= 1
}

func maxY(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func resolveLogo(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, _, err := image.DecodeConfig(f); err != nil {
		return ""
	}
	probe := fpdf.New("P", "mm", "A4", "")
	if probe.RegisterImageOptions(path, fpdf.ImageOptions{ReadDpi: true}) == nil || probe.Err() {
		return ""
	}
	return path
}

func newPDFText(pdf *fpdf.Fpdf) func(string) string {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	return func(s string) string {
		return tr(pdfSafe(s))
	}
}

// pdfSafe maps punctuation Helvetica cannot take as raw UTF-8. Middle-dot
// bytes otherwise render as "Â·".
func pdfSafe(s string) string {
	r := strings.NewReplacer(
		"·", "|",
		"•", "-",
		"–", "-",
		"—", "-",
		"’", "'",
		"‘", "'",
		"“", `"`,
		"”", `"`,
		"\u00a0", " ",
		"…", "...",
	)
	return r.Replace(s)
}
