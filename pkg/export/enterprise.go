package export

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	navyR, navyG, navyB    = 3, 56, 96
	tealR, tealG, tealB    = 8, 145, 178
	lineR, lineG, lineB    = 203, 213, 225
	mutedR, mutedG, mutedB = 100, 116, 139
	textR, textG, textB    = 15, 23, 42
	fillR, fillG, fillB    = 248, 250, 252
)

func attachEnterpriseChrome(pdf *fpdf.Fpdf, tr func(string) string, head Letterhead, printed time.Time, afterHeader func()) {
	head = resolveLetterhead(head)
	if printed.IsZero() {
		printed = time.Now()
	}
	pdf.AliasNbPages("")
	pdf.SetHeaderFunc(func() {
		writeEnterpriseHeader(pdf, tr, head)
		if afterHeader != nil {
			afterHeader()
		}
	})
	pdf.SetFooterFunc(func() {
		writeEnterpriseFooter(pdf, tr, head, printed)
	})
}

func writeEnterpriseHeader(pdf *fpdf.Fpdf, tr func(string) string, head Letterhead) {
	head = resolveLetterhead(head)
	pageW, _ := pdf.GetPageSize()
	landscape := pageW > 250
	headerH := 32.0
	if landscape {
		headerH = 22.0
	}

	pdf.SetFillColor(navyR, navyG, navyB)
	pdf.Rect(0, 0, pageW, headerH, "F")
	pdf.SetFillColor(tealR, tealG, tealB)
	pdf.Rect(0, headerH, pageW, 1.0, "F")

	left := 14.0
	textX := left
	logoTop := 5.0
	logoBox := 14.0
	if landscape {
		logoTop = 3.5
		logoBox = 12.0
	}
	if logo := resolveLogo(head.LogoPath); logo != "" {
		pdf.SetFillColor(255, 255, 255)
		pdf.RoundedRect(left, logoTop, logoBox, logoBox, 1.5, "1234", "F")
		pdf.ImageOptions(logo, left+1, logoTop+1, logoBox-2, 0, false, fpdf.ImageOptions{ReadDpi: true}, 0, "")
		if !pdf.Err() {
			textX = left + logoBox + 4
		}
	}

	nameW := pageW - textX - 14
	pdf.SetXY(textX, logoTop)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 10)
	if landscape {
		pdf.SetFont("Helvetica", "B", 9)
	}
	pdf.CellFormat(nameW, 4.2, tr(head.DisplayName()), "", 1, "L", false, 0, "")
	pdf.SetTextColor(186, 210, 230)
	pdf.SetFont("Helvetica", "", 6.5)
	pdf.SetX(textX)
	for _, line := range letterheadAddress(head) {
		pdf.CellFormat(nameW, 3.0, tr(line), "", 1, "L", false, 0, "")
		pdf.SetX(textX)
	}
	if contact := joinNonEmpty(prefix("Tel ", head.Phone), prefix("Email ", head.Email), prefix("Web ", head.Website)); contact != "" {
		pdf.CellFormat(nameW, 3.0, tr(contact), "", 1, "L", false, 0, "")
		pdf.SetX(textX)
	}
	if !landscape {
		if tax := joinNonEmpty(prefix("VRN ", head.VRN), prefix("TIN ", head.TIN), head.ISO); tax != "" {
			pdf.CellFormat(nameW, 3.0, tr(tax), "", 1, "L", false, 0, "")
		}
	} else if tax := joinNonEmpty(prefix("VRN ", head.VRN), prefix("TIN ", head.TIN)); tax != "" {
		pdf.CellFormat(nameW, 3.0, tr(tax), "", 1, "L", false, 0, "")
	}

	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(headerH + 4)
}

func writeEnterpriseFooter(pdf *fpdf.Fpdf, tr func(string) string, head Letterhead, printed time.Time) {
	head = resolveLetterhead(head)
	pageW, pageH := pdf.GetPageSize()
	barH := 10.0
	barY := pageH - barH
	left := 14.0
	contentW := pageW - 28

	pdf.SetY(barY - 6)
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	pdf.SetFont("Helvetica", "", 5.5)
	pdf.SetX(left)
	pdf.CellFormat(contentW, 3, tr("Computer-generated document. Valid only when matched against the official system record."), "", 0, "L", false, 0, "")

	pdf.SetFillColor(navyR, navyG, navyB)
	pdf.Rect(0, barY, pageW, barH, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 6.5)
	pdf.SetXY(left, barY+3)
	pdf.CellFormat(contentW*0.42, 4, tr(fmt.Sprintf("%s  |  Generated %s", head.DisplayName(), printed.Format("02/01/2006 15:04"))), "", 0, "L", false, 0, "")
	pdf.SetXY(left+contentW*0.42, barY+3)
	pdf.CellFormat(contentW*0.20, 4, tr(fmt.Sprintf("Page %d of {nb}", pdf.PageNo())), "", 0, "C", false, 0, "")
	pdf.SetXY(left+contentW*0.62, barY+3)
	pdf.CellFormat(contentW*0.38, 4, tr("CONFIDENTIAL"), "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func writeCenteredTitle(pdf *fpdf.Fpdf, tr func(string) string, title, status, number string) {
	pageW, _ := pdf.GetPageSize()
	left := 14.0
	contentW := pageW - 28
	pdf.SetTextColor(navyR, navyG, navyB)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetX(left)
	pdf.CellFormat(contentW, 6, tr(strings.ToUpper(title)), "", 1, "C", false, 0, "")
	uy := pdf.GetY() + 0.2
	pdf.SetDrawColor(tealR, tealG, tealB)
	pdf.SetLineWidth(0.9)
	mid := left + contentW/2
	pdf.Line(mid-36, uy, mid+36, uy)
	pdf.SetLineWidth(0.2)
	pdf.SetY(uy + 2.5)

	if number != "" || status != "" {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(navyR, navyG, navyB)
		pdf.SetX(left)
		pdf.CellFormat(contentW*0.62, 5, tr(number), "", 0, "L", false, 0, "")
		if status != "" {
			st := strings.ToUpper(strings.ReplaceAll(status, "_", " "))
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetFillColor(navyR, navyG, navyB)
			pdf.SetTextColor(255, 255, 255)
			badgeW := pdf.GetStringWidth(tr(st)) + 8
			if badgeW < 22 {
				badgeW = 22
			}
			pdf.SetXY(left+contentW-badgeW, pdf.GetY())
			pdf.CellFormat(badgeW, 5.5, tr(st), "", 0, "C", true, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(8)
	}
}

func writeScanToVerify(pdf *fpdf.Fpdf, tr func(string) string, verifyURL string) {
	verifyURL = strings.TrimSpace(verifyURL)
	if verifyURL == "" {
		return
	}
	png, err := qrcode.Encode(verifyURL, qrcode.Medium, 128)
	if err != nil || len(png) == 0 {
		return
	}
	pageW, pageH := pdf.GetPageSize()
	left := 14.0
	contentW := pageW - 28
	qrW, qrH := 36.0, 38.0
	if pdf.GetY()+qrH+18 > pageH {
		pdf.AddPage()
	}
	y := pdf.GetY() + 2
	remarksW := contentW - qrW - 4

	pdf.SetFillColor(240, 249, 255)
	pdf.SetDrawColor(tealR, tealG, tealB)
	pdf.RoundedRect(left, y, remarksW, qrH, 1.2, "1234", "FD")
	pdf.SetXY(left+3, y+4)
	pdf.SetTextColor(navyR, navyG, navyB)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(remarksW-6, 5, tr("Scan to confirm"), "", 1, "L", false, 0, "")
	pdf.SetX(left + 3)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(textR, textG, textB)
	pdf.MultiCell(remarksW-6, 3.8, tr("Scan the QR code on a mobile device to open this document from the official record. No login is required. The link is signed and cannot be reused for another document."), "", "L", false)

	name := fmt.Sprintf("verify-qr-%d", pdf.PageNo())
	opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(png))
	qx := left + contentW - qrW
	pdf.SetDrawColor(lineR, lineG, lineB)
	pdf.SetFillColor(255, 255, 255)
	pdf.RoundedRect(qx, y, qrW, qrH, 1.2, "1234", "FD")
	pdf.ImageOptions(name, qx+6, y+2.5, 24, 0, false, opt, 0, "")
	pdf.SetXY(qx, y+27.5)
	pdf.SetTextColor(navyR, navyG, navyB)
	pdf.SetFont("Helvetica", "B", 6)
	pdf.CellFormat(qrW, 3, tr("SCAN TO CONFIRM"), "", 1, "C", false, 0, "")
	pdf.SetX(qx)
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	pdf.SetFont("Helvetica", "", 5)
	pdf.CellFormat(qrW, 2.8, tr("No login required"), "", 0, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(y + qrH + 2)
}

func PDFBytes(pdf *fpdf.Fpdf) ([]byte, error) {
	return pdfBytes(pdf)
}

func pdfBytes(pdf *fpdf.Fpdf) ([]byte, error) {
	if pdf.Err() {
		return nil, pdf.Error()
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
