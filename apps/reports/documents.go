package reports

import (
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/internal/orders"
	"dfms/pkg/db"
	"dfms/pkg/docsig"
	"dfms/pkg/export"
	"dfms/pkg/portalurl"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

func lookupUID(c fiber.Ctx) string {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		id = strings.TrimSpace(c.Params("id"))
	}
	return id
}

func companyHead() export.Letterhead {
	if db.Db == nil {
		return export.Letterhead{}
	}
	var company models.Company
	_ = db.Db.First(&company, 1).Error
	return export.Letterhead{
		CompanyName: company.Name, Address: company.Address, Address2: company.Address2,
		City: company.City, Postal: company.PostalCode, Country: company.Country,
		Phone: company.Phone, Email: company.Email, Website: company.Website,
		TIN: company.TinNumber, VRN: company.VrnNumber, ISO: company.IsoNumber,
		LogoPath: company.LogoPath,
	}
}

// BindLetterhead prints Company ID 1 on every PDF and Excel export.
func BindLetterhead() {
	export.UseLetterhead(companyHead)
}

func writeOfficial(c fiber.Ctx, kind, uid, title, number, status, file string, fields [][2]string, heads []string, lines [][]string) error {
	return export.DocumentPDFHead(c, title, number, status, file, companyHead(),
		portalurl.DocumentVerifyURL(c.Context(), db.Db, kind, uid), fields, heads, lines)
}

func glrDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var req models.GantryLoadingRequest
	if err := db.Db.Preload("Customer").Preload("Product").Preload("ByProduct").Preload("StockStatus").
		Preload("Lines.Product").Preload("Lines.Driver").Preload("Vessels.Vessel").Preload("Vessels.Product").
		Preload("StockPositions.Product").Preload("Outstanding").Preload("Charges").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&req).Error; err != nil {
		return response.NotFound(c, "internal loading request")
	}
	if !wantsPDF(c) && !wantsExcel(c) {
		return response.OkDetail(c, req)
	}
	if wantsExcel(c) {
		return serveAny(c, "ILR "+req.DocumentNumber, "ilr", req.Lines)
	}
	doc := ilrExportDoc(req)
	doc.VerifyURL = portalurl.DocumentVerifyURL(c.Context(), db.Db, docsig.KindILR, req.UID)
	return export.ILRPDF(c, doc)
}

func fmtL(d decimal.Decimal) string {
	neg := d.IsNegative()
	if neg {
		d = d.Abs()
	}
	s := d.StringFixed(2)
	parts := strings.SplitN(s, ".", 2)
	intp := parts[0]
	var b strings.Builder
	n := len(intp)
	for i, c := range intp {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String() + "." + parts[1]
	if neg {
		return "-" + out
	}
	return out
}

func productHead(p models.Product) string {
	name := strings.TrimSpace(p.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(p.Code)
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func ilrProductHeads(req models.GantryLoadingRequest) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p models.Product) {
		code := productHead(p)
		if code == "" || seen[code] {
			return
		}
		seen[code] = true
		out = append(out, code)
	}
	add(req.Product)
	if req.ByProduct != nil {
		add(*req.ByProduct)
	}
	for _, v := range req.Vessels {
		add(v.Product)
	}
	for _, l := range req.Lines {
		add(l.Product)
	}
	return out
}

func outstandingRows(req models.GantryLoadingRequest) [][3]string {
	if len(req.Charges) > 0 {
		var out [][3]string
		for _, ch := range req.Charges {
			out = append(out, [3]string{ch.Charge, ch.CurrencyCode, ch.Amount.StringFixed(2)})
		}
		return out
	}
	if req.Outstanding == nil {
		return [][3]string{
			{"Storage Debt", "TZS", "0.00"},
			{"Storage Debt", "USD", "0.00"},
			{"Weight & Measure", "TZS", "0.00"},
			{"TBS Debt", "TZS", "0.00"},
		}
	}
	o := req.Outstanding
	return [][3]string{
		{"Storage Debt", "TZS", o.StorageTZS.StringFixed(2)},
		{"Storage Debt", "USD", o.StorageUSD.StringFixed(2)},
		{"Weight & Measure", "TZS", o.WeightMeasureTZS.StringFixed(2)},
		{"TBS Debt", "TZS", o.TbsTZS.StringFixed(2)},
	}
}

func ilrApprovalRows(req models.GantryLoadingRequest) [][]string {
	trail := orders.DocumentApprovals(db.Db, types.GantryLoadingRequestContent, req.ID, req.ApprovalTrail)
	out := make([][]string, 0, len(trail))
	for _, s := range trail {
		out = append(out, []string{
			s.ActedAt.Format("02/01/2006 15:04"),
			s.UserName,
			s.Title,
			firstNonEmpty(s.Comment, s.ActName),
		})
	}
	return out
}

func appendApprovalFields(fields [][2]string, ct types.ContentType, objectID uint, snapshot models.ApprovalTrail) [][2]string {
	for _, s := range orders.DocumentApprovals(db.Db, ct, objectID, snapshot) {
		who := strings.TrimSpace(s.UserName)
		if t := strings.TrimSpace(s.Title); t != "" {
			if who != "" {
				who += " · "
			}
			who += t
		}
		detail := s.ActedAt.Format("02/01/2006 15:04")
		if c := strings.TrimSpace(s.Comment); c != "" {
			detail += " — " + c
		}
		label := firstNonEmpty(s.ActName, s.ActType, "Approval")
		fields = append(fields, [2]string{label, strings.TrimSpace(who + "  " + detail)})
	}
	return fields
}

func compartmentDoc(c fiber.Ctx, kind string) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var comp models.Compartmentalization
	if err := db.Db.Preload("Ilo.Request.Customer").Preload("Ilo.Request.Product").
		Preload("Lines.Product").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&comp).Error; err != nil {
		return response.NotFound(c, "compartmentalization")
	}
	cust, prod := "", ""
	if comp.Ilo.Request != nil {
		cust = comp.Ilo.Request.Customer.Name
		prod = comp.Ilo.Request.Product.Code
	}
	title := map[string]string{
		"delivery-note": "Delivery note",
		"gate-in":       "Gate-in pass",
		"gate-out":      "Gate-out pass",
	}[kind]
	fields := [][2]string{
		{"Document", comp.DocumentNumber},
		{"GLO", comp.Ilo.DocumentNumber},
		{"Customer", cust},
		{"Product", prod},
		{"Truck", firstNonEmpty(comp.HorsePlate, comp.Ilo.TruckPlate)},
		{"Trailer 1", firstNonEmpty(comp.TrailerOnePlate, comp.Ilo.TrailerOnePlate)},
		{"Trailer 2", firstNonEmpty(comp.TrailerTwoPlate, comp.Ilo.TrailerTwoPlate)},
		{"Destination", comp.Ilo.Destination},
		{"Status", string(comp.Status)},
	}
	fields = appendApprovalFields(fields, types.CompartmentalizationContent, comp.ID, comp.ApprovalTrail)
	var lines [][]string
	for _, cl := range comp.Lines {
		code := ""
		if cl.Product != nil {
			code = cl.Product.Code
		}
		lines = append(lines, []string{
			fmt.Sprintf("%d", cl.Index), cl.TankPlate, code, cl.Quantity.String(),
			cl.TopSeal, cl.DipSeal, cl.BottomSeal,
		})
	}
	if wantsPDF(c) {
		return writeOfficial(c, kind, comp.UID, title, comp.DocumentNumber, string(comp.Status), strings.ReplaceAll(kind, "-", "_"),
			fields, []string{"#", "Tank", "Product", "Litres", "Top seal", "Dip seal", "Bottom seal"}, lines)
	}
	if wantsExcel(c) {
		return serveAny(c, title, kind, comp.Lines)
	}
	return response.OkDetail(c, fiber.Map{"header": comp, "fields": fields})
}

func deliveryNote(c fiber.Ctx) error { return compartmentDoc(c, "delivery-note") }
func gateIn(c fiber.Ctx) error       { return compartmentDoc(c, "gate-in") }
func gateOut(c fiber.Ctx) error      { return compartmentDoc(c, "gate-out") }

func pumpOverDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var req models.PumpOverRequest
	if err := db.Db.Preload("Customer").Preload("Product").Preload("StockStatus").
		Preload("Depot").Preload("Vessels.Vessel").Preload("Vessels.StockStatus").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&req).Error; err != nil {
		return response.NotFound(c, "pump-over request")
	}
	fields := [][2]string{
		{"Document", req.DocumentNumber},
		{"Date", req.OrderDate.Format("02/01/2006")},
		{"Customer", req.Customer.Name},
		{"Product", req.Product.Code},
		{"Depot", req.Depot.Name},
		{"Quantity", req.Quantity.String()},
		{"Status", string(req.Status)},
	}
	fields = appendApprovalFields(fields, types.PumpOverRequestContent, req.ID, req.ApprovalTrail)
	var lines [][]string
	for _, v := range req.Vessels {
		lines = append(lines, []string{
			v.Vessel.Code,
			v.VesselDate.Format("02/01/2006"),
			v.StockStatus.Name,
			v.Quantity.String(),
		})
	}
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindPumpOver, req.UID, "Pump-over request", req.DocumentNumber, string(req.Status), "pump_over",
			fields, []string{"Vessel", "Vessel date", "Stock status", "Quantity"}, lines)
	}
	if wantsExcel(c) {
		return serveAny(c, "Pump-over request", "pump_over", req.Vessels)
	}
	return response.OkDetail(c, req)
}

func pumpOverReportDoc(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var rep models.PumpOverReport
	if err := db.Db.Preload("Request.Customer").Preload("Request.Product").Preload("Request.Depot").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&rep).Error; err != nil {
		return response.NotFound(c, "pump-over report")
	}
	fields := [][2]string{
		{"Document", rep.DocumentNumber},
		{"Request", rep.Request.DocumentNumber},
		{"Date", rep.ReportDate.Format("02/01/2006")},
		{"Customer", rep.Request.Customer.Name},
		{"Product", rep.Request.Product.Code},
		{"Depot", rep.Request.Depot.Name},
		{"Delivered", rep.ActualDelivered.String()},
		{"Received", rep.ActualReceived.String()},
		{"Variance", rep.Variance.String()},
		{"Status", string(rep.Status)},
	}
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindPumpOverReport, rep.UID, "Pump-over report", rep.DocumentNumber, string(rep.Status), "pump_over_report", fields, nil, nil)
	}
	if wantsExcel(c) {
		type row struct {
			DocumentNumber string
			Delivered      decimal.Decimal
			Received       decimal.Decimal
			Variance       decimal.Decimal
			Status         string
		}
		return serveAny(c, "Pump-over report", "pump_over_report", []row{{
			rep.DocumentNumber, rep.ActualDelivered, rep.ActualReceived, rep.Variance, string(rep.Status),
		}})
	}
	return response.OkDetail(c, rep)
}

func ittDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var itt models.IttTransfer
	if err := db.Db.Preload("FromCustomer").Preload("ToCustomer").Preload("Product").Preload("Vessel").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&itt).Error; err != nil {
		return response.NotFound(c, "in-tank transfer")
	}
	fields := [][2]string{
		{"Document", itt.DocumentNumber},
		{"Date", itt.TransferDate.Format("02/01/2006")},
		{"From", itt.FromCustomer.Name},
		{"To", itt.ToCustomer.Name},
		{"Product", itt.Product.Code},
		{"Vessel", itt.Vessel.Code},
		{"Vessel date", itt.VesselDate.Format("02/01/2006")},
		{"Quantity", itt.Quantity.String()},
		{"Status", string(itt.Status)},
	}
	fields = appendApprovalFields(fields, types.IttTransferContent, itt.ID, itt.ApprovalTrail)
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindITT, itt.UID, "In-tank transfer", itt.DocumentNumber, string(itt.Status), "itt", fields, nil, nil)
	}
	if wantsExcel(c) {
		return serveAny(c, "ITT", "itt", []models.IttTransfer{itt})
	}
	return response.OkDetail(c, itt)
}

func receiptDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var row models.Receipt
	if err := models.PreloadCreatedBy(db.Db).Preload("Vessel").Preload("Product").Preload("Supplier").
		Preload("Details.Customer").Preload("Details.StockStatus").Preload("Details.Depot").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&row).Error; err != nil {
		return response.NotFound(c, "receipt")
	}
	kind := "Internal vessel receipt"
	if row.ReceiptType == types.ReceiptExternal {
		kind = "External vessel receipt"
	}
	survey := "Final"
	if row.IsProvision {
		survey = "Provision"
	}
	fields := [][2]string{
		{"Document", row.DocumentNumber},
		{"Date", row.Date.Format("02/01/2006")},
		{"Vessel date", row.VesselDate.Format("02/01/2006")},
		{"Vessel", firstNonEmpty(row.Vessel.Name, row.Vessel.Code)},
		{"Product", firstNonEmpty(row.Product.Name, row.Product.Code)},
		{"Route", string(row.RouteCode)},
		{"Status", string(row.Status)},
	}
	if row.ReceiptType == types.ReceiptInternal {
		sup := ""
		if row.Supplier != nil {
			sup = firstNonEmpty(row.Supplier.Name, row.Supplier.Code)
		}
		fields = append(fields,
			[2]string{"Supplier", sup},
			[2]string{"Survey", survey},
			[2]string{"Tender", string(row.TenderCode)},
			[2]string{"Procurement", string(row.ProcurementMethodCode)},
			[2]string{"Density", row.Density.String()},
		)
	} else {
		fields = append(fields, [2]string{"10-inch pipeline", yesNo(row.UsesTiperPipeline)})
	}
	if row.Creator != nil {
		fields = append(fields, [2]string{"Created by", firstNonEmpty(row.Creator.Name, row.Creator.Email)})
	}
	if strings.TrimSpace(row.Notes) != "" {
		fields = append(fields, [2]string{"Notes", row.Notes})
	}
	fields = appendApprovalFields(fields, types.ReceiptContent, row.ID, nil)
	internal := row.ReceiptType != types.ReceiptExternal
	heads := []string{"Customer"}
	if internal {
		heads = append(heads, "Status", "Delivery", "Hold")
	} else {
		heads = append(heads, "Depot")
	}
	heads = append(heads, "Litres", "m3", "MT")
	var lines [][]string
	for _, d := range row.Details {
		line := []string{firstNonEmpty(d.Customer.Name, d.Customer.Code)}
		if internal {
			st := ""
			if d.StockStatus != nil {
				st = d.StockStatus.Name
			}
			line = append(line, st, string(d.CollectionMethod), yesNo(d.FinancialHold))
		} else {
			depot := ""
			if d.Depot != nil {
				depot = firstNonEmpty(d.Depot.Name, d.Depot.Code)
			}
			line = append(line, depot)
		}
		line = append(line, fmtL(d.Quantity), fmtL(d.CubicMeter), fmtL(d.MetricTonne))
		lines = append(lines, line)
	}
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindReceipt, row.UID, kind, row.DocumentNumber, string(row.Status), "receipt", fields, heads, lines)
	}
	if wantsExcel(c) {
		return serveAny(c, kind, "receipt", row.Details)
	}
	return response.OkDetail(c, row)
}

func zerolizationDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var z models.ZerolizationTransfer
	if err := db.Db.Preload("Customer").Preload("Product").Preload("FromVessel").Preload("ToVessel").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&z).Error; err != nil {
		return response.NotFound(c, "zerolization")
	}
	fields := [][2]string{
		{"Document", z.DocumentNumber},
		{"Date", z.TransferDate.Format("02/01/2006")},
		{"Customer", z.Customer.Name},
		{"Product", z.Product.Code},
		{"From vessel", z.FromVessel.Code},
		{"From vessel date", z.FromVesselDate.Format("02/01/2006")},
		{"To vessel", z.ToVessel.Code},
		{"To vessel date", z.ToVesselDate.Format("02/01/2006")},
		{"Quantity", z.Quantity.String()},
		{"Status", string(z.Status)},
	}
	fields = appendApprovalFields(fields, types.ZerolizationContent, z.ID, nil)
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindZerolization, z.UID, "Zerolization", z.DocumentNumber, string(z.Status), "zerolization", fields, nil, nil)
	}
	return response.OkDetail(c, z)
}

func holdReleaseDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	var row models.FinancialHoldRelease
	if err := models.PreloadCreatedBy(db.Db).
		Preload("Lines.Customer").Preload("Lines.Product").Preload("Lines.Vessel").Preload("Lines.StockStatus").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&row).Error; err != nil {
		return response.NotFound(c, "financial hold release")
	}
	fields := [][2]string{
		{"Document", row.DocumentNumber},
		{"Date", row.ReleaseDate.Format("02/01/2006")},
		{"Description", row.Description},
		{"Status", string(row.Status)},
	}
	if row.Creator != nil {
		fields = append(fields, [2]string{"Created by", firstNonEmpty(row.Creator.Name, row.Creator.Email)})
	}
	if strings.TrimSpace(row.Notes) != "" {
		fields = append(fields, [2]string{"Notes", row.Notes})
	}
	fields = appendApprovalFields(fields, types.FinancialHoldContent, row.ID, row.ApprovalTrail)
	var lines [][]string
	for _, l := range row.Lines {
		lines = append(lines, []string{
			firstNonEmpty(l.Customer.Name, l.Customer.Code),
			firstNonEmpty(l.Product.Name, l.Product.Code),
			firstNonEmpty(l.Vessel.Name, l.Vessel.Code),
			l.VesselDate.Format("02/01/2006"),
			l.StockStatus.Name,
			fmtL(l.Quantity),
			fmtL(l.CubicMeter),
			fmtL(l.MetricTonne),
		})
	}
	heads := []string{"Customer", "Product", "Vessel", "Vessel date", "Status", "Litres", "m3", "MT"}
	if wantsPDF(c) {
		return writeOfficial(c, docsig.KindHoldRelease, row.UID, "Financial hold release", row.DocumentNumber, string(row.Status), "hold_release", fields, heads, lines)
	}
	return response.OkDetail(c, row)
}

func miLossDocument(c fiber.Ctx) error {
	id := lookupUID(c)
	if id == "" {
		return response.BadRequest(c, "id is required")
	}
	row, err := loadMiLossBatch(id)
	if err != nil {
		return response.NotFound(c, "MI loss batch")
	}
	if wantsPDF(c) {
		return WriteMiLossPDF(c, row)
	}
	if wantsExcel(c) {
		type line struct {
			Product  string
			Contract string
			Percent  string
		}
		out := make([]line, 0, len(row.Lines))
		_, _, rows := miLossSheet(row)
		for _, r := range rows {
			if len(r) < 3 {
				continue
			}
			out = append(out, line{Product: r[0], Contract: r[1], Percent: r[2]})
		}
		return serveAny(c, "MI loss", "miloss", out)
	}
	return response.OkDetail(c, row)
}

func loadMiLossBatch(id string) (models.MiLossBatch, error) {
	var row models.MiLossBatch
	err := models.PreloadCreatedBy(db.Db).
		Preload("Products.Product").Preload("Products.Rates.Product").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&row).Error
	return row, err
}

// WriteMiLossPDF prints an MI-loss batch with company letterhead.
func WriteMiLossPDF(c fiber.Ctx, row models.MiLossBatch) error {
	fields, heads, lines := miLossSheet(row)
	return writeOfficial(c, docsig.KindMiLoss, row.UID, "MI loss", row.DocumentNumber, string(row.Status), "miloss", fields, heads, lines)
}

func miLossSheet(row models.MiLossBatch) ([][2]string, []string, [][]string) {
	created := ""
	if row.Creator != nil {
		created = row.Creator.Name
	}
	fields := [][2]string{
		{"Document", row.DocumentNumber},
		{"Date", row.Date.Format("02/01/2006")},
		{"Effective from", row.EffectiveFrom.Format("02/01/2006")},
		{"Description", firstNonEmpty(row.Description, "—")},
		{"Created by", firstNonEmpty(created, "—")},
		{"Rates", fmt.Sprintf("%d", len(row.Lines))},
		{"Status", string(row.Status)},
	}
	heads := []string{"Product", "Contract", "MI-loss %"}
	lines := make([][]string, 0, len(row.Lines))
	for _, l := range row.Lines {
		prod := ""
		if l.Product != nil {
			prod = firstNonEmpty(strings.TrimSpace(l.Product.Code+" — "+l.Product.Name), l.Product.Code, l.Product.Name)
		}
		pct := l.Value.Mul(decimal.NewFromInt(100)).StringFixed(4)
		lines = append(lines, []string{prod, string(l.ContractTypeCode), pct})
	}
	return fields, heads, lines
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
