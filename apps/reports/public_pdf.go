package reports

import (
	"context"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/docsig"
	"dfms/pkg/export"
	"dfms/pkg/portalurl"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PublicDocument is the unauthenticated scan payload (HMAC already checked).
type PublicDocument struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Number      string `json:"documentNumber"`
	Status      string `json:"status"`
	CompanyName string `json:"companyName"`
	Customer    string `json:"customer,omitempty"`
}

// BuildPublicPDF renders the signed document as PDF bytes plus a summary for the scan page.
func BuildPublicPDF(ctx context.Context, kind, id string) ([]byte, string, PublicDocument, error) {
	kind = docsig.NormalizeKind(kind)
	id = strings.TrimSpace(id)
	info := PublicDocument{Kind: kind, Label: docsig.Label(kind), CompanyName: companyHead().DisplayName()}
	if kind == "" || id == "" {
		return nil, "", info, gorm.ErrRecordNotFound
	}

	var (
		raw      []byte
		filename string
		err      error
	)
	switch kind {
	case docsig.KindILR:
		raw, filename, info, err = publicILR(ctx, id, info)
	case docsig.KindDeliveryNote, docsig.KindGateIn, docsig.KindGateOut:
		raw, filename, info, err = publicCompartment(ctx, kind, id, info)
	case docsig.KindPumpOver:
		raw, filename, info, err = publicPumpOver(ctx, id, info)
	case docsig.KindPumpOverReport:
		raw, filename, info, err = publicPumpOverReport(ctx, id, info)
	case docsig.KindITT:
		raw, filename, info, err = publicITT(ctx, id, info)
	case docsig.KindReceipt:
		raw, filename, info, err = publicReceipt(ctx, id, info)
	case docsig.KindZerolization:
		raw, filename, info, err = publicZerolization(ctx, id, info)
	case docsig.KindHoldRelease:
		raw, filename, info, err = publicHoldRelease(ctx, id, info)
	case docsig.KindMiLoss:
		raw, filename, info, err = publicMiLoss(ctx, id, info)
	default:
		return nil, "", info, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, "", info, err
	}
	return raw, filename, info, nil
}

func officialBytes(ctx context.Context, kind, uid, title, number, status, file string, fields [][2]string, heads []string, lines [][]string) ([]byte, string, error) {
	pdf, err := export.RenderOfficial(export.OfficialDoc{
		Title: title, Number: number, Status: status, FilePrefix: file,
		Head: companyHead(), VerifyURL: portalurl.DocumentVerifyURL(ctx, db.Db, kind, uid),
		Fields: fields, LineHeaders: heads, Lines: lines,
	})
	if err != nil {
		return nil, "", err
	}
	raw, err := export.PDFBytes(pdf)
	if err != nil {
		return nil, "", err
	}
	name := fmt.Sprintf("%s_%s.pdf", file, strings.ReplaceAll(number, ":", "-"))
	return raw, name, nil
}

func publicILR(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var req models.GantryLoadingRequest
	if err := db.Db.Preload("Customer").Preload("Product").Preload("ByProduct").Preload("StockStatus").
		Preload("Lines.Product").Preload("Lines.Driver").Preload("Vessels.Vessel").Preload("Vessels.Product").
		Preload("StockPositions.Product").Preload("Outstanding").Preload("Charges").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&req).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status, info.Customer = req.DocumentNumber, string(req.Status), req.Customer.Name
	doc := ilrExportDoc(req)
	doc.VerifyURL = portalurl.DocumentVerifyURL(ctx, db.Db, docsig.KindILR, req.UID)
	pdf, err := export.RenderILR(doc)
	if err != nil {
		return nil, "", info, err
	}
	raw, err := export.PDFBytes(pdf)
	if err != nil {
		return nil, "", info, err
	}
	return raw, fmt.Sprintf("ilr_%s.pdf", strings.ReplaceAll(req.DocumentNumber, ":", "-")), info, nil
}

func publicCompartment(ctx context.Context, kind, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var comp models.Compartmentalization
	if err := db.Db.Preload("Ilo.Request.Customer").Preload("Ilo.Request.Product").
		Preload("Lines.Product").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&comp).Error; err != nil {
		return nil, "", info, err
	}
	title := docsig.Label(kind)
	cust, prod := "", ""
	if comp.Ilo.Request != nil {
		cust = comp.Ilo.Request.Customer.Name
		prod = comp.Ilo.Request.Product.Code
	}
	info.Number, info.Status, info.Customer = comp.DocumentNumber, string(comp.Status), cust
	fields := [][2]string{
		{"Document", comp.DocumentNumber},
		{"GLO", comp.Ilo.DocumentNumber},
		{"Customer", cust},
		{"Product", prod},
		{"Truck", firstNonEmpty(comp.HorsePlate, comp.Ilo.TruckPlate)},
		{"Status", string(comp.Status)},
	}
	var lines [][]string
	for _, cl := range comp.Lines {
		code := ""
		if cl.Product != nil {
			code = cl.Product.Code
		}
		lines = append(lines, []string{fmt.Sprintf("%d", cl.Index), cl.TankPlate, code, cl.Quantity.String()})
	}
	raw, name, err := officialBytes(ctx, kind, comp.UID, title, comp.DocumentNumber, string(comp.Status), strings.ReplaceAll(kind, "-", "_"),
		fields, []string{"#", "Tank", "Product", "Litres"}, lines)
	return raw, name, info, err
}

func publicPumpOver(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var req models.PumpOverRequest
	if err := db.Db.Preload("Customer").Preload("Product").Preload("Depot").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&req).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status, info.Customer = req.DocumentNumber, string(req.Status), req.Customer.Name
	fields := [][2]string{
		{"Document", req.DocumentNumber},
		{"Customer", req.Customer.Name},
		{"Product", req.Product.Code},
		{"Depot", req.Depot.Name},
		{"Quantity", req.Quantity.String()},
		{"Status", string(req.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindPumpOver, req.UID, "Pump-over request", req.DocumentNumber, string(req.Status), "pump_over", fields, nil, nil)
	return raw, name, info, err
}

func publicPumpOverReport(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var rep models.PumpOverReport
	if err := db.Db.Preload("Request.Customer").Where("UID = ? OR DocumentNumber = ?", id, id).First(&rep).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status = rep.DocumentNumber, string(rep.Status)
	if rep.Request.Customer.Name != "" {
		info.Customer = rep.Request.Customer.Name
	}
	fields := [][2]string{
		{"Document", rep.DocumentNumber},
		{"Delivered", rep.ActualDelivered.String()},
		{"Received", rep.ActualReceived.String()},
		{"Status", string(rep.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindPumpOverReport, rep.UID, "Pump-over report", rep.DocumentNumber, string(rep.Status), "pump_over_report", fields, nil, nil)
	return raw, name, info, err
}

func publicITT(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var itt models.IttTransfer
	if err := db.Db.Preload("FromCustomer").Preload("ToCustomer").Preload("Product").
		Where("UID = ? OR DocumentNumber = ?", id, id).First(&itt).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status, info.Customer = itt.DocumentNumber, string(itt.Status), itt.FromCustomer.Name+" → "+itt.ToCustomer.Name
	fields := [][2]string{
		{"Document", itt.DocumentNumber},
		{"From", itt.FromCustomer.Name},
		{"To", itt.ToCustomer.Name},
		{"Product", itt.Product.Code},
		{"Quantity", itt.Quantity.String()},
		{"Status", string(itt.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindITT, itt.UID, "In-tank transfer", itt.DocumentNumber, string(itt.Status), "itt", fields, nil, nil)
	return raw, name, info, err
}

func publicReceipt(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var row models.Receipt
	if err := db.Db.Preload("Vessel").Preload("Product").Where("UID = ? OR DocumentNumber = ?", id, id).First(&row).Error; err != nil {
		return nil, "", info, err
	}
	title := "Internal vessel receipt"
	if row.ReceiptType == types.ReceiptExternal {
		title = "External vessel receipt"
	}
	info.Number, info.Status = row.DocumentNumber, string(row.Status)
	fields := [][2]string{
		{"Document", row.DocumentNumber},
		{"Date", row.Date.Format("02/01/2006")},
		{"Vessel", firstNonEmpty(row.Vessel.Name, row.Vessel.Code)},
		{"Product", firstNonEmpty(row.Product.Name, row.Product.Code)},
		{"Status", string(row.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindReceipt, row.UID, title, row.DocumentNumber, string(row.Status), "receipt", fields, nil, nil)
	return raw, name, info, err
}

func publicZerolization(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var z models.ZerolizationTransfer
	if err := db.Db.Preload("Customer").Preload("Product").Where("UID = ? OR DocumentNumber = ?", id, id).First(&z).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status, info.Customer = z.DocumentNumber, string(z.Status), z.Customer.Name
	fields := [][2]string{
		{"Document", z.DocumentNumber},
		{"Customer", z.Customer.Name},
		{"Product", z.Product.Code},
		{"Quantity", z.Quantity.String()},
		{"Status", string(z.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindZerolization, z.UID, "Zerolization", z.DocumentNumber, string(z.Status), "zerolization", fields, nil, nil)
	return raw, name, info, err
}

func publicHoldRelease(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	var row models.FinancialHoldRelease
	if err := db.Db.Where("UID = ? OR DocumentNumber = ?", id, id).First(&row).Error; err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status = row.DocumentNumber, string(row.Status)
	fields := [][2]string{
		{"Document", row.DocumentNumber},
		{"Date", row.ReleaseDate.Format("02/01/2006")},
		{"Status", string(row.Status)},
	}
	raw, name, err := officialBytes(ctx, docsig.KindHoldRelease, row.UID, "Financial hold release", row.DocumentNumber, string(row.Status), "hold_release", fields, nil, nil)
	return raw, name, info, err
}

func publicMiLoss(ctx context.Context, id string, info PublicDocument) ([]byte, string, PublicDocument, error) {
	row, err := loadMiLossBatch(id)
	if err != nil {
		return nil, "", info, err
	}
	info.Number, info.Status = row.DocumentNumber, string(row.Status)
	fields, heads, lines := miLossSheet(row)
	raw, name, err := officialBytes(ctx, docsig.KindMiLoss, row.UID, "MI loss", row.DocumentNumber, string(row.Status), "miloss", fields, heads, lines)
	return raw, name, info, err
}

func ilrExportDoc(req models.GantryLoadingRequest) export.ILRDoc {
	head := companyHead()
	products := [][2]string{{productHead(req.Product), fmtL(req.Quantity)}}
	if req.ByProduct != nil {
		products = append(products, [2]string{productHead(*req.ByProduct), fmtL(req.ByProductQuantity)})
	}
	charges := outstandingRows(req)
	prodCols := ilrProductHeads(req)
	vesselHeads := []string{"Date", "Vessel name", "F. Hold"}
	vesselHeads = append(vesselHeads, prodCols...)
	var vessels [][]string
	totals := map[string]decimal.Decimal{}
	for _, v := range req.Vessels {
		row := []string{v.VesselDate.Format("02/01/2006"), v.Vessel.Name, yesNo(v.FinancialHold)}
		for _, code := range prodCols {
			cell := ""
			if productHead(v.Product) == code {
				cell = fmtL(v.Quantity)
				totals[code] = totals[code].Add(v.Quantity)
			}
			row = append(row, cell)
		}
		vessels = append(vessels, row)
	}
	if len(vessels) > 0 {
		totalRow := []string{"", "Total", ""}
		for _, code := range prodCols {
			totalRow = append(totalRow, fmtL(totals[code]))
		}
		vessels = append(vessels, totalRow)
	}
	var stock [][]string
	for _, p := range req.StockPositions {
		stock = append(stock, []string{productHead(p.Product), fmtL(p.TotalBalance), fmtL(p.HoldQty), fmtL(p.FreeQty), fmtL(p.FinalQty)})
	}
	truckHeads := []string{"S/N", "Order no", "Truck", "Driver name", "Licence / passport"}
	truckHeads = append(truckHeads, prodCols...)
	var trucks [][]string
	lineTot := map[string]decimal.Decimal{}
	sn := 0
	for _, l := range req.Lines {
		if l.Amended {
			continue
		}
		sn++
		plate := models.TruckComboPlate(l.HorsePlate, l.TrailerOnePlate, l.TrailerTwoPlate)
		if plate == "" {
			plate = l.TruckPlate
		}
		row := []string{fmt.Sprintf("%d", sn), l.DocumentNumber, plate, l.DriverName, l.EwuraLicense}
		for _, code := range prodCols {
			cell := ""
			if productHead(l.Product) == code {
				cell = fmtL(l.RequestedQty)
				lineTot[code] = lineTot[code].Add(l.RequestedQty)
			}
			row = append(row, cell)
		}
		trucks = append(trucks, row)
	}
	if len(trucks) > 0 {
		totalRow := []string{"", "Total", "", "", ""}
		for _, code := range prodCols {
			totalRow = append(totalRow, fmtL(lineTot[code]))
		}
		trucks = append(trucks, totalRow)
	}
	return export.ILRDoc{
		Number: req.DocumentNumber, Status: string(req.Status),
		Date: req.OrderDate.Format("02/01/2006"), Description: req.Description,
		Customer: req.Customer.Name, ProductStatus: req.StockStatus.Name,
		Contract: req.ValidContract, LoadingOrder: req.LoadingOrderAvailable,
		CompanyName: head.CompanyName, Address: head.Address, Address2: head.Address2,
		City: head.City, Postal: head.Postal, Country: head.Country, Phone: head.Phone,
		Email: head.Email, Website: head.Website, TIN: head.TIN,
		VRN: head.VRN, ISO: head.ISO, LogoPath: head.LogoPath,
		Products: products, Charges: charges,
		VesselHeads: vesselHeads, Vessels: vessels,
		StockHeads: []string{"Product", "Total balance", "Volume under F.Hold", "Free volume", "Free volume after GLR"},
		Stock:      stock, ApprovedQty: fmtL(req.Quantity.Add(req.ByProductQuantity)),
		TruckHeads: truckHeads, Trucks: trucks, Approvals: ilrApprovalRows(req),
	}
}
