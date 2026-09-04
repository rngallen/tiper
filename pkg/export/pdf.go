package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v3"
)

// OfficialDoc is a portrait key/value document plus optional lines and QR.
type OfficialDoc struct {
	Title, Number, Status, FilePrefix string
	Head                              Letterhead
	VerifyURL                         string
	Fields                            [][2]string
	LineHeaders                       []string
	Lines                             [][]string
}

// TablePDF writes a landscape A4 table with company letterhead and page numbers.
func TablePDF(c fiber.Ctx, title, filePrefix string, headers []string, rows [][]string) error {
	pdf, err := RenderTable(title, resolveLetterhead(Letterhead{}), headers, rows)
	if err != nil {
		return err
	}
	raw, err := pdfBytes(pdf)
	if err != nil {
		return err
	}
	if filePrefix == "" {
		filePrefix = "report"
	}
	name := fmt.Sprintf("%s_%s.pdf", filePrefix, time.Now().Format("02012006_150405"))
	return SendPDF(c, name, raw, false)
}

// RenderTable builds a landscape report PDF. The organisation header (from
// Company) and column heads repeat on every page; the footer is Page X of Y.
func RenderTable(title string, head Letterhead, headers []string, rows [][]string) (*fpdf.Fpdf, error) {
	head = resolveLetterhead(head)
	if len(headers) == 0 {
		headers = []string{"Value"}
	}
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor(head.DisplayName(), false)
	pdf.SetMargins(14, 0, 14)
	pdf.SetAutoPageBreak(true, 18)
	tr := newPDFText(pdf)
	printed := time.Now()
	attachEnterpriseChrome(pdf, tr, head, printed, func() {
		writeReportTitle(pdf, tr, title, printed)
		writeTableColHeads(pdf, tr, headers)
	})
	pdf.AddPage()

	left, widths := tableLayout(pdf, headers)
	fill := false
	for _, row := range rows {
		if fill {
			pdf.SetFillColor(fillR, fillG, fillB)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetX(left)
		for i := range headers {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			align := "L"
			if isSerialHead(headers[i]) {
				align = "C"
			} else if looksNumber(v) {
				align = "R"
			}
			writeFitCell(pdf, tr, widths[i], 6, v, align, true, "")
		}
		pdf.Ln(-1)
		fill = !fill
	}
	if pdf.Err() {
		return nil, pdf.Error()
	}
	return pdf, nil
}

func writeReportTitle(pdf *fpdf.Fpdf, tr func(string) string, title string, printed time.Time) {
	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	contentW := pageW - left - right
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(navyR, navyG, navyB)
	pdf.SetX(left)
	pdf.CellFormat(contentW, 6, tr(strings.ToUpper(title)), "", 1, "L", false, 0, "")
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetX(left)
	pdf.CellFormat(contentW, 4, tr(printed.Format("02/01/2006 15:04")), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(1.2)
}

func writeTableColHeads(pdf *fpdf.Fpdf, tr func(string) string, headers []string) {
	left, widths := tableLayout(pdf, headers)
	pdf.SetX(left)
	pdf.SetFillColor(navyR, navyG, navyB)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		align := "L"
		if isSerialHead(h) {
			align = "C"
		}
		writeFitCell(pdf, tr, widths[i], 7, h, align, true, "B")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
}

func tableLayout(pdf *fpdf.Fpdf, headers []string) (float64, []float64) {
	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	return left, tableColWidths(pageW-left-right, headers)
}

func tableColWidths(avail float64, headers []string) []float64 {
	n := len(headers)
	w := make([]float64, n)
	if n == 0 {
		return w
	}
	serialW := 0.0
	serialN := 0
	for i, h := range headers {
		if isSerialHead(h) {
			w[i] = 12
			serialW += 12
			serialN++
		}
	}
	rest := n - serialN
	if rest <= 0 {
		each := avail / float64(n)
		for i := range w {
			w[i] = each
		}
		return w
	}
	each := (avail - serialW) / float64(rest)
	for i := range w {
		if w[i] == 0 {
			w[i] = each
		}
	}
	return w
}

func isSerialHead(h string) bool {
	s := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, " ", "")))
	return s == "s/n" || s == "sn" || s == "#" || s == "serial" || s == "serialnumber" || s == "no."
}

// DocumentPDF writes a titled key/value sheet plus an optional line table.
func DocumentPDF(c fiber.Ctx, title, filePrefix string, fields [][2]string, lineHeaders []string, lines [][]string) error {
	return DocumentPDFHead(c, title, "", "", filePrefix, resolveLetterhead(Letterhead{}), "", fields, lineHeaders, lines)
}

// DocumentPDFHead prints an official portrait document with company letterhead.
func DocumentPDFHead(c fiber.Ctx, title, number, status, filePrefix string, head Letterhead, verifyURL string, fields [][2]string, lineHeaders []string, lines [][]string) error {
	pdf, err := RenderOfficial(OfficialDoc{
		Title: title, Number: number, Status: status, FilePrefix: filePrefix,
		Head: head, VerifyURL: verifyURL, Fields: fields, LineHeaders: lineHeaders, Lines: lines,
	})
	if err != nil {
		return err
	}
	raw, err := pdfBytes(pdf)
	if err != nil {
		return err
	}
	if filePrefix == "" {
		filePrefix = "document"
	}
	name := fmt.Sprintf("%s_%s.pdf", filePrefix, time.Now().Format("02012006_150405"))
	return SendPDF(c, name, raw, false)
}

// RenderOfficial builds a portrait ERP document (header, title, fields, lines, QR).
func RenderOfficial(doc OfficialDoc) (*fpdf.Fpdf, error) {
	head := resolveLetterhead(doc.Head)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(doc.Title, false)
	pdf.SetAuthor(head.DisplayName(), false)
	pdf.SetMargins(14, 0, 14)
	pdf.SetAutoPageBreak(true, 22)
	tr := newPDFText(pdf)
	attachEnterpriseChrome(pdf, tr, head, time.Now(), nil)
	pdf.AddPage()

	writeCenteredTitle(pdf, tr, doc.Title, doc.Status, doc.Number)

	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	contentW := pageW - left - right
	colW := contentW / 2
	for i := 0; i < len(doc.Fields); i += 2 {
		x := left
		rowY := pdf.GetY()
		if rowY > 250 {
			pdf.AddPage()
			rowY = pdf.GetY()
		}
		writeField := func(kv [2]string, x float64) {
			pdf.SetXY(x, rowY)
			pdf.SetFont("Helvetica", "", 7)
			pdf.SetTextColor(mutedR, mutedG, mutedB)
			pdf.CellFormat(colW-2, 4, tr(kv[0]), "", 1, "L", false, 0, "")
			pdf.SetX(x)
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(textR, textG, textB)
			pdf.MultiCell(colW-2, 4.5, tr(kv[1]), "", "L", false)
		}
		writeField(doc.Fields[i], x)
		h1 := pdf.GetY() - rowY
		if i+1 < len(doc.Fields) {
			writeField(doc.Fields[i+1], left+colW)
			h2 := pdf.GetY() - rowY
			if h2 > h1 {
				h1 = h2
			}
		}
		pdf.SetY(rowY + h1 + 1.5)
	}

	if len(doc.LineHeaders) > 0 {
		pdf.Ln(2)
		bandedTable(pdf, tr, doc.LineHeaders, doc.Lines)
	}
	writeScanToVerify(pdf, tr, doc.VerifyURL)
	if pdf.Err() {
		return nil, pdf.Error()
	}
	return pdf, nil
}

// SendPDF writes a PDF attachment (or inline preview) to the Fiber response.
func SendPDF(c fiber.Ctx, filename string, raw []byte, inline bool) error {
	disp := "attachment"
	if inline {
		disp = "inline"
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disp, filename))
	c.Set("Cache-Control", "private, max-age=300")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(raw)
}
