package inventory

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/attach"
	"dfms/internal/catalogs"
	"dfms/internal/integrations"
	invsvc "dfms/internal/inventory"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/precision"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func Router(app *fiber.App, svc *invsvc.Service) {
	h := handler{db: db.Db, svc: svc}
	g := app.Group("/api/v1/ic", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	need := func(code string) fiber.Handler {
		return middleware.PermissionMiddleware(code)
	}
	stockRead := middleware.PermissionMiddleware(permissions.InventoryRead, permissions.InventoryBalances)
	stockWrite := need(permissions.InventoryUpdate)
	receiptRead := need(permissions.ReceiptsRead)
	receiptReview := middleware.PermissionMiddleware(permissions.ReceiptsRead, permissions.WorkflowTasks)
	receiptCreate := need(permissions.ReceiptsCreate)
	receiptUpdate := need(permissions.ReceiptsUpdate)
	receiptWrite := middleware.PermissionMiddleware(permissions.ReceiptsCreate, permissions.ReceiptsUpdate)
	ittRead := need(permissions.ITTRead)
	ittReview := middleware.PermissionMiddleware(permissions.ITTRead, permissions.WorkflowTasks)
	ittWrite := middleware.PermissionMiddleware(permissions.ITTCreate, permissions.ITTUpdate)
	zerolRead := need(permissions.ZerolRead)
	zerolReview := middleware.PermissionMiddleware(permissions.ZerolRead, permissions.WorkflowTasks)
	zerolWrite := middleware.PermissionMiddleware(permissions.ZerolCreate, permissions.ZerolUpdate)
	holdRead := need(permissions.HoldRead)
	holdReview := middleware.PermissionMiddleware(permissions.HoldRead, permissions.WorkflowTasks)
	holdWrite := middleware.PermissionMiddleware(permissions.HoldCreate, permissions.HoldUpdate)

	g.Get("/balances", stockRead, h.balances)
	g.Post("/reservations", stockWrite, h.reserve)
	g.Post("/reservations/:order/release", stockWrite, h.releaseReservation)
	g.Get("/reservations", stockRead, h.listReservations)
	g.Get("/receipts", receiptRead, h.listReceipts)
	g.Post("/receipts", receiptCreate, h.createReceipt)
	g.Get("/receipts/:uid", receiptReview, h.getReceipt)
	g.Patch("/receipts/:uid", receiptUpdate, h.patchReceipt)
	g.Post("/receipts/:uid/details", receiptWrite, h.addReceiptDetail)
	g.Put("/receipts/:uid/details/:did", receiptUpdate, h.updateReceiptDetail)
	g.Delete("/receipts/:uid/details/:did", receiptUpdate, h.deleteReceiptDetail)
	g.Post("/receipts/:uid/submit", need(permissions.ReceiptsSubmit), h.submitReceipt)
	attach.Register(g, "/receipts/:uid", receiptReview, receiptWrite, h.db, types.ReceiptContent, h.attachReceipt)
	g.Post("/receipts/:uid/allocate-line-loss", receiptUpdate, h.allocateReceiptLineLoss)
	g.Post("/receipts/:uid/convert-to-final", receiptCreate, h.convertReceipt)
	g.Get("/itt", ittRead, h.listITT)
	g.Post("/itt", need(permissions.ITTCreate), h.createITT)
	g.Get("/itt/:uid", ittReview, h.getITT)
	g.Post("/itt/:uid/submit", need(permissions.ITTSubmit), h.submitITT)
	attach.Register(g, "/itt/:uid", ittReview, ittWrite, h.db, types.IttTransferContent, h.attachITT)
	g.Get("/zerolization", zerolRead, h.listZerol)
	g.Post("/zerolization", need(permissions.ZerolCreate), h.createZerol)
	g.Get("/zerolization/:uid", zerolReview, h.getZerol)
	g.Post("/zerolization/:uid/submit", need(permissions.ZerolSubmit), h.submitZerol)
	attach.Register(g, "/zerolization/:uid", zerolReview, zerolWrite, h.db, types.ZerolizationContent, h.attachZerol)
	g.Get("/hold-parcels", holdRead, h.listHoldParcels)
	g.Get("/hold-releases", holdRead, h.listHoldReleases)
	g.Post("/hold-releases", need(permissions.HoldCreate), h.createHoldRelease)
	g.Get("/hold-releases/:uid", holdReview, h.getHoldRelease)
	g.Post("/hold-releases/:uid/submit", need(permissions.HoldSubmit), h.submitHoldRelease)
	attach.Register(g, "/hold-releases/:uid", holdReview, holdWrite, h.db, types.FinancialHoldContent, h.attachHold)
	g.Post("/dips", stockWrite, h.createDip)
	g.Get("/dips", stockRead, h.listDips)
	g.Post("/line-content", stockWrite, h.createLine)
	g.Get("/events", stockRead, h.listEvents)
	g.Post("/events", stockWrite, h.createEvent)
	g.Get("/movements", stockRead, h.listMovements)
}

type handler struct {
	db  *gorm.DB
	svc *invsvc.Service
}

func (h handler) balances(c fiber.Ctx) error {
	rows, err := h.svc.Balances(0, 0)
	if err != nil {
		return err
	}
	if code := c.Query("customer"); code != "" {
		filtered := rows[:0]
		for _, r := range rows {
			if r.CustomerCode == code {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	return response.OkDetail(c, rows)
}

func (h handler) reserve(c fiber.Ctx) error {
	var in struct {
		CustomerID  string          `json:"customerId"`
		ProductID   string          `json:"productId"`
		Quantity    decimal.Decimal `json:"quantity"`
		OrderNumber string          `json:"orderNumber"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	cid, err := lookupID[models.Customer](h.db, in.CustomerID)
	if err != nil {
		return response.UnprocessableEntity(c, errors.New("unknown customer"))
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.UnprocessableEntity(c, errors.New("unknown product"))
	}
	row, err := h.svc.Reserve(cid, pid, in.Quantity, in.OrderNumber, nil)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Created(c, row)
}

func parseOps(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseOpsSearchRequest(c)
	if err != nil {
		return search, response.BadRequest(c, err.Error())
	}
	return search, nil
}

func filterReceiptStatus(c fiber.Ctx, q *gorm.DB, column string) (*gorm.DB, error) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		return q, nil
	}
	if !types.ReceiptStatus(status).Valid() {
		return q, response.BadRequest(c, "invalid status")
	}
	return q.Where(column+" = ?", status), nil
}

func filterDocStatus(c fiber.Ctx, q *gorm.DB, column string) (*gorm.DB, error) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		return q, nil
	}
	if !types.DocumentStatus(status).Valid() {
		return q, response.BadRequest(c, "invalid status")
	}
	return q.Where(column+" = ?", status), nil
}

func (h handler) listReceipts(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.Receipt{})).
		Select("[Receipt].*").
		Joins("LEFT JOIN [Vessel] ON [Vessel].ID = [Receipt].VesselID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [Receipt].ProductID").
		Preload("Vessel").Preload("Product")
	q, err = filterReceiptStatus(c, q, "[Receipt].Status")
	if err != nil {
		return err
	}
	if t := strings.TrimSpace(c.Query("receiptType")); t != "" {
		if !types.ReceiptType(t).Valid() {
			return response.BadRequest(c, "invalid receipt type")
		}
		q = q.Where("[Receipt].ReceiptType = ?", t)
	}
	q = response.ApplyLike(q, search,
		"[Receipt].DocumentNumber", "[Receipt].ReceiptType", "[Receipt].RouteCode",
		"[Receipt].Notes", "[Vessel].Code", "[Vessel].Name", "[Product].Code", "[Product].Name",
	)
	return response.ServeList(c, response.ListOpts[models.Receipt]{
		Query: q, Search: search,
		DateColumn:  "[Receipt].Date",
		DefaultSort: "[Receipt].Date",
		TieBreak:    "[Receipt].ID",
		Sort: map[string]string{
			"documentNumber": "[Receipt].DocumentNumber",
			"date":           "[Receipt].Date",
			"receiptType":    "[Receipt].ReceiptType",
			"routeCode":      "[Receipt].RouteCode",
			"status":         "[Receipt].Status",
		},
		Sheet: "Receipts", File: "receipts",
		Headers: []any{"Document", "Date", "Vessel", "Product", "Type", "Route", "Provision", "Status"},
		MapRow: func(r models.Receipt) []any {
			prov := "No"
			if r.IsProvision {
				prov = "Yes"
			}
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.Vessel.Code, r.Product.Code, string(r.ReceiptType), string(r.RouteCode), prov, string(r.Status)}
		},
	})
}

func (h handler) getReceipt(c fiber.Ctx) error {
	var row models.Receipt
	if err := models.PreloadCreatedBy(h.db).
		Preload("Details", "OriginDetailID IS NULL").
		Preload("Details.Customer").Preload("Details.StockStatus").Preload("Details.Depot").
		Preload("Vessel").Preload("Product").Preload("Supplier").Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) createReceipt(c fiber.Ctx) error {
	var in receiptInput
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row, err := h.buildReceipt(in, middleware.GetUserIDFromContext(c))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := h.applyReceiptLineLoss(row); err != nil {
		return response.BadRequest(c, err.Error())
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "receipt", "RCPT")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		return tx.Create(row).Error
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, row.UID, types.ReceiptContent, "receipt "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

type receiptInput struct {
	Date                  time.Time     `json:"date"`
	VesselDate            time.Time     `json:"vesselDate"`
	BillingEffectiveDate  time.Time     `json:"billingEffectiveDate"`
	VesselID              string        `json:"vesselId"`
	SupplierID            string        `json:"supplierId"`
	ProductID             string        `json:"productId"`
	RouteCode             string        `json:"routeCode"`
	TenderCode            string        `json:"tenderCode"`
	ProcurementMethodCode string        `json:"procurementMethodCode"`
	ReceiptType           string        `json:"receiptType"`
	Density               string        `json:"density"`
	TankQuantity          string        `json:"tankQuantity"`
	TankCubicMeter        string        `json:"tankCubicMeter"`
	TankMetricTonne       string        `json:"tankMetricTonne"`
	LineLoss              string        `json:"lineLoss"`
	LineLossCubicMeter    string        `json:"lineLossCubicMeter"`
	LineLossMetricTonne   string        `json:"lineLossMetricTonne"`
	IsProvision           bool          `json:"isProvision"`
	IsFinal               bool          `json:"isFinal"`
	UsesTiperPipeline     bool          `json:"usesTiperPipeline"`
	Notes                 string        `json:"notes"`
	Details               []detailInput `json:"details"`
}

type detailInput struct {
	CustomerID       string `json:"customerId"`
	StockStatusID    string `json:"stockStatusId"`
	DepotID          string `json:"depotId"`
	CollectionMethod string `json:"collectionMethod"`
	NextBillingDays  int    `json:"nextBillingDays"`
	ContractTypeCode string `json:"contractTypeCode"`
	PricingNature    string `json:"pricingNature"`
	Quantity         string `json:"quantity"`
	CubicMeter       string `json:"cubicMeter"`
	MetricTonne      string `json:"metricTonne"`
	Density          string `json:"density"`
	FinancialHold    bool   `json:"financialHold"`
}

func (h handler) buildReceipt(in receiptInput, userID uint) (*models.Receipt, error) {
	row := models.Receipt{
		Date:                  in.Date,
		VesselDate:            in.VesselDate,
		BillingEffectiveDate:  in.BillingEffectiveDate,
		RouteCode:             types.DischargeRoute(in.RouteCode),
		TenderCode:            types.TenderCode(in.TenderCode),
		ProcurementMethodCode: types.ProcurementCode(in.ProcurementMethodCode),
		ReceiptType:           types.ReceiptType(in.ReceiptType),
		IsProvision:           in.IsProvision,
		IsFinal:               in.IsFinal,
		Notes:                 in.Notes,
		CreatedByID:           userID,
		Status:                types.ReceiptDraft,
	}
	if d, err := decimal.NewFromString(strings.TrimSpace(in.Density)); err == nil {
		row.Density = precision.Round(d, integrations.LivePrecision().Density)
	}
	row.TankQuantity, _ = decimal.NewFromString(strings.TrimSpace(in.TankQuantity))
	row.TankCubicMeter, _ = decimal.NewFromString(strings.TrimSpace(in.TankCubicMeter))
	row.TankMetricTonne, _ = decimal.NewFromString(strings.TrimSpace(in.TankMetricTonne))
	row.LineLoss, _ = decimal.NewFromString(strings.TrimSpace(in.LineLoss))
	row.LineLossCubicMeter, _ = decimal.NewFromString(strings.TrimSpace(in.LineLossCubicMeter))
	row.LineLossMetricTonne, _ = decimal.NewFromString(strings.TrimSpace(in.LineLossMetricTonne))
	if row.BillingEffectiveDate.IsZero() && !row.Date.IsZero() {
		row.BillingEffectiveDate = row.Date.AddDate(0, 0, 1)
	}
	if row.ReceiptType == "" {
		row.ReceiptType = types.ReceiptInternal
	}
	if !row.ReceiptType.Valid() {
		return nil, fmt.Errorf("receipt type must be internal or external")
	}
	if pid, err := lookupID[models.Product](h.db, in.ProductID); err != nil {
		return nil, err
	} else {
		row.ProductID = pid
	}
	if vid, err := lookupID[models.Vessel](h.db, in.VesselID); err != nil {
		return nil, err
	} else {
		row.VesselID = vid
	}
	cats, err := catalogs.Load(h.db)
	if err != nil {
		return nil, err
	}
	if err := catalogs.RequireActive(cats, "route", string(row.RouteCode)); err != nil {
		return nil, err
	}
	if in.SupplierID != "" {
		sid, err := lookupID[models.Supplier](h.db, in.SupplierID)
		if err != nil {
			return nil, err
		}
		row.SupplierID = &sid
	}
	if row.ReceiptType == types.ReceiptExternal {
		row.IsProvision = false
		row.IsFinal = true
		row.TenderCode = ""
		row.ProcurementMethodCode = ""
		row.TankQuantity = decimal.Zero
		row.TankCubicMeter = decimal.Zero
		row.TankMetricTonne = decimal.Zero
		row.LineLoss = decimal.Zero
		row.LineLossCubicMeter = decimal.Zero
		row.LineLossMetricTonne = decimal.Zero
		row.UsesTiperPipeline = in.UsesTiperPipeline && row.RouteCode.IsKOJ()
		if err := h.appendExternalDetails(&row, in); err != nil {
			return nil, err
		}
	} else {
		if err := catalogs.RequireActive(cats, "tender", string(row.TenderCode)); err != nil {
			return nil, err
		}
		if err := catalogs.RequireActive(cats, "procurement", string(row.ProcurementMethodCode)); err != nil {
			return nil, err
		}
		row.UsesTiperPipeline = false
		if err := h.appendInternalDetails(&row, in, cats); err != nil {
			return nil, err
		}
	}
	return &row, nil
}

func (h handler) appendInternalDetails(row *models.Receipt, in receiptInput, cats catalogs.Set) error {
	for _, d := range in.Details {
		cid, err := lookupID[models.Customer](h.db, d.CustomerID)
		if err != nil {
			return err
		}
		sid, err := lookupID[models.StockStatus](h.db, d.StockStatusID)
		if err != nil {
			return err
		}
		days := d.NextBillingDays
		if days <= 0 {
			days = 15
		}
		cm := types.CollectionMethod(strings.ToUpper(strings.TrimSpace(d.CollectionMethod)))
		if err := catalogs.RequireActive(cats, "delivery", string(cm)); err != nil {
			return err
		}
		if err := catalogs.RequireCycle(cats, days); err != nil {
			return err
		}
		contract := types.ContractCode(strings.ToUpper(strings.TrimSpace(d.ContractTypeCode)))
		if err := catalogs.RequireActive(cats, "contract", string(contract)); err != nil {
			return err
		}
		nature := types.PricingNature(strings.ToUpper(strings.TrimSpace(d.PricingNature)))
		if nature == "" {
			nature = types.PricingTariff
		}
		if nature == types.PricingPromotion {
			nature = types.PricingPromotional
		}
		if err := catalogs.RequireActive(cats, "pricing", string(nature)); err != nil {
			return err
		}
		det := models.ReceiptDetail{
			CustomerID:       cid,
			StockStatusID:    &sid,
			CollectionMethod: cm,
			NextBillingDays:  days,
			ContractTypeCode: contract,
			PricingNature:    nature,
			FinancialHold:    d.FinancialHold,
			IsProvision:      in.IsProvision,
		}
		if d.DepotID != "" {
			did, err := lookupID[models.Depot](h.db, d.DepotID)
			if err != nil {
				return err
			}
			det.DepotID = &did
		}
		if err := applyDetailQty(&det, d, row.Density, integrations.LivePrecision()); err != nil {
			return err
		}
		var st models.StockStatus
		if err := h.db.Select("IsProration").First(&st, sid).Error; err == nil && st.IsProration && !det.Quantity.IsNegative() {
			return fmt.Errorf("proration (PRORATA) parcel quantity must be negative")
		}
		det.Density = row.Density
		row.Details = append(row.Details, det)
	}
	return nil
}

func (h handler) appendExternalDetails(row *models.Receipt, in receiptInput) error {
	for i, d := range in.Details {
		cid, err := lookupID[models.Customer](h.db, d.CustomerID)
		if err != nil {
			return fmt.Errorf("customer is required on depot line %d", i+1)
		}
		did, err := lookupID[models.Depot](h.db, d.DepotID)
		if err != nil {
			return fmt.Errorf("depot is required on line %d", i+1)
		}
		det := models.ReceiptDetail{
			CustomerID:  cid,
			DepotID:     &did,
			IsProvision: false,
		}
		if err := applyDetailQty(&det, d, decimal.Zero, integrations.LivePrecision()); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
		row.Details = append(row.Details, det)
	}
	return nil
}

func applyDetailQty(det *models.ReceiptDetail, d detailInput, density decimal.Decimal, prec precision.Settings) error {
	det.Quantity, _ = decimal.NewFromString(strings.TrimSpace(d.Quantity))
	if det.Quantity.IsZero() {
		return fmt.Errorf("quantity in litres is required")
	}
	det.Quantity = precision.Round(det.Quantity, prec.Quantity)
	det.CubicMeter, det.MetricTonne = invsvc.QtyFromLitresRounded(det.Quantity, density, prec)
	return nil
}

func (h handler) qtyPrec() precision.Settings {
	return integrations.LivePrecision()
}

func (h handler) applyReceiptLineLoss(row *models.Receipt) error {
	if row == nil || row.ReceiptType != types.ReceiptInternal {
		return nil
	}
	flags, err := invsvc.ProrationFlags(h.db, row.Details)
	if err != nil {
		return err
	}
	invsvc.ApplyTankLineLoss(row, flags)
	if row.LineLoss.IsZero() && row.LineLossCubicMeter.IsZero() && row.LineLossMetricTonne.IsZero() {
		return nil
	}
	invsvc.AllocateLineLossRounded(row.Details, flags, row.LineLoss, row.LineLossCubicMeter, row.LineLossMetricTonne, h.qtyPrec())
	return nil
}

func (h handler) allocateReceiptLineLoss(c fiber.Ctx) error {
	var in struct {
		TankQuantity        string `json:"tankQuantity"`
		TankCubicMeter      string `json:"tankCubicMeter"`
		TankMetricTonne     string `json:"tankMetricTonne"`
		LineLoss            string `json:"lineLoss"`
		LineLossCubicMeter  string `json:"lineLossCubicMeter"`
		LineLossMetricTonne string `json:"lineLossMetricTonne"`
	}
	_ = c.Bind().JSON(&in)
	dec := func(s string) decimal.Decimal {
		v, _ := decimal.NewFromString(strings.TrimSpace(s))
		return v
	}
	row, err := h.svc.AllocateReceiptLineLoss(c.Params("uid"),
		dec(in.TankQuantity), dec(in.TankCubicMeter), dec(in.TankMetricTonne),
		dec(in.LineLoss), dec(in.LineLossCubicMeter), dec(in.LineLossMetricTonne))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, row)
}

func lookupID[T any](db *gorm.DB, uid string) (uint, error) {
	var row T
	if err := db.Where("UID = ?", uid).First(&row).Error; err != nil {
		return 0, err
	}
	switch v := any(&row).(type) {
	case *models.Product:
		return v.ID, nil
	case *models.Vessel:
		return v.ID, nil
	case *models.Customer:
		return v.ID, nil
	case *models.StockStatus:
		return v.ID, nil
	case *models.Supplier:
		return v.ID, nil
	case *models.Depot:
		return v.ID, nil
	default:
		return 0, fiber.ErrBadRequest
	}
}

func (h handler) loadDraftReceipt(uid string) (*models.Receipt, error) {
	var row models.Receipt
	if err := h.db.Preload("Details").Where("UID = ?", uid).First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status != types.ReceiptDraft {
		return nil, fmt.Errorf("only a draft receipt can be changed")
	}
	return &row, nil
}

func (h handler) reloadReceipt(uid string) (*models.Receipt, error) {
	var row models.Receipt
	if err := h.db.Preload("Details", "OriginDetailID IS NULL").
		Preload("Details.Customer").Preload("Details.StockStatus").Preload("Details.Depot").
		Preload("Vessel").Preload("Product").Preload("Supplier").Where("UID = ?", uid).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (h handler) patchReceipt(c fiber.Ctx) error {
	row, err := h.loadDraftReceipt(c.Params("uid"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	before := *row
	var in receiptInput
	if err := c.Bind().Body(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	in.Sanitize()
	if in.ReceiptType == "" {
		in.ReceiptType = string(row.ReceiptType)
	}
	if err := in.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	cats, err := catalogs.Load(h.db)
	if err != nil {
		return err
	}
	if err := catalogs.RequireActive(cats, "route", in.RouteCode); err != nil {
		return response.BadRequest(c, err.Error())
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	vid, err := lookupID[models.Vessel](h.db, in.VesselID)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	updates := map[string]any{
		"Date":       in.Date,
		"VesselDate": in.VesselDate,
		"Notes":      in.Notes,
		"ProductID":  pid,
		"VesselID":   vid,
		"RouteCode":  types.DischargeRoute(in.RouteCode),
	}
	if d, err := decimal.NewFromString(strings.TrimSpace(in.Density)); err == nil {
		updates["Density"] = d
		row.Density = d
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.TankQuantity)); err == nil {
		updates["TankQuantity"] = v
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.TankCubicMeter)); err == nil {
		updates["TankCubicMeter"] = v
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.TankMetricTonne)); err == nil {
		updates["TankMetricTonne"] = v
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.LineLoss)); err == nil {
		updates["LineLoss"] = v
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.LineLossCubicMeter)); err == nil {
		updates["LineLossCubicMeter"] = v
	}
	if v, err := decimal.NewFromString(strings.TrimSpace(in.LineLossMetricTonne)); err == nil {
		updates["LineLossMetricTonne"] = v
	}
	if row.ReceiptType == types.ReceiptInternal {
		if err := catalogs.RequireActive(cats, "tender", in.TenderCode); err != nil {
			return response.BadRequest(c, err.Error())
		}
		if err := catalogs.RequireActive(cats, "procurement", in.ProcurementMethodCode); err != nil {
			return response.BadRequest(c, err.Error())
		}
		sid, err := lookupID[models.Supplier](h.db, in.SupplierID)
		if err != nil {
			return response.BadRequest(c, err.Error())
		}
		updates["IsProvision"] = in.IsProvision
		updates["IsFinal"] = !in.IsProvision
		updates["TenderCode"] = types.TenderCode(in.TenderCode)
		updates["ProcurementMethodCode"] = types.ProcurementCode(in.ProcurementMethodCode)
		updates["SupplierID"] = sid
	} else {
		updates["UsesTiperPipeline"] = in.UsesTiperPipeline && types.DischargeRoute(in.RouteCode).IsKOJ()
	}
	if err := h.db.Model(row).Updates(updates).Error; err != nil {
		return err
	}
	if d, ok := updates["Density"].(decimal.Decimal); ok {
		for i := range row.Details {
			det := &row.Details[i]
			cm, mt := invsvc.QtyFromLitresRounded(det.Quantity, d, h.qtyPrec())
			if err := h.db.Model(det).Updates(map[string]any{
				"Density": d, "CubicMeter": cm, "MetricTonne": mt,
			}).Error; err != nil {
				return err
			}
		}
	}
	out, err := h.reloadReceipt(row.UID)
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.ReceiptContent, "receipt "+row.DocumentNumber+" updated", before, out)
	return okUpdate(c, out, before, out)
}

func (h handler) addReceiptDetail(c fiber.Ctx) error {
	row, err := h.loadDraftReceipt(c.Params("uid"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	var in detailInput
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	det, err := h.buildOneDetail(row, in)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	det.ReceiptID = row.ID
	if err := h.db.Create(&det).Error; err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, det.UID, types.ReceiptDetailContent, "receipt line added on "+row.DocumentNumber, nil, det)
	out, err := h.reloadReceipt(row.UID)
	if err != nil {
		return err
	}
	return response.Created(c, out)
}

func (h handler) updateReceiptDetail(c fiber.Ctx) error {
	row, err := h.loadDraftReceipt(c.Params("uid"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	var det models.ReceiptDetail
	if err := h.db.Where("UID = ? AND ReceiptID = ?", c.Params("did"), row.ID).First(&det).Error; err != nil {
		return err
	}
	before := det
	var in detailInput
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	next, err := h.buildOneDetail(row, in)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := h.db.Model(&det).Updates(map[string]any{
		"CustomerID": next.CustomerID, "StockStatusID": next.StockStatusID, "DepotID": next.DepotID,
		"CollectionMethod": next.CollectionMethod, "ContractTypeCode": next.ContractTypeCode,
		"PricingNature": next.PricingNature, "NextBillingDays": next.NextBillingDays,
		"FinancialHold": next.FinancialHold, "Density": next.Density,
		"Quantity": next.Quantity, "CubicMeter": next.CubicMeter, "MetricTonne": next.MetricTonne,
		"IsProvision": next.IsProvision,
	}).Error; err != nil {
		return err
	}
	recordAudit(c, types.ActionUpdate, det.UID, types.ReceiptDetailContent, "receipt line updated on "+row.DocumentNumber, before, next)
	out, err := h.reloadReceipt(row.UID)
	if err != nil {
		return err
	}
	return okUpdate(c, out, before, next)
}

func (h handler) deleteReceiptDetail(c fiber.Ctx) error {
	row, err := h.loadDraftReceipt(c.Params("uid"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	var det models.ReceiptDetail
	if err := h.db.Where("UID = ? AND ReceiptID = ?", c.Params("did"), row.ID).First(&det).Error; err != nil {
		return err
	}
	if err := h.db.Delete(&det).Error; err != nil {
		return err
	}
	recordAudit(c, types.ActionDelete, det.UID, types.ReceiptDetailContent, "receipt line removed from "+row.DocumentNumber, det, nil)
	out, err := h.reloadReceipt(row.UID)
	if err != nil {
		return err
	}
	return response.OkDetail(c, out)
}

func (h handler) buildOneDetail(row *models.Receipt, in detailInput) (models.ReceiptDetail, error) {
	tmp := models.Receipt{Density: row.Density, ReceiptType: row.ReceiptType, RouteCode: row.RouteCode}
	if row.ReceiptType == types.ReceiptExternal {
		if err := h.appendExternalDetails(&tmp, receiptInput{Details: []detailInput{in}}); err != nil {
			return models.ReceiptDetail{}, err
		}
	} else {
		cats, err := catalogs.Load(h.db)
		if err != nil {
			return models.ReceiptDetail{}, err
		}
		if err := h.appendInternalDetails(&tmp, receiptInput{Details: []detailInput{in}, IsProvision: row.IsProvision}, cats); err != nil {
			return models.ReceiptDetail{}, err
		}
	}
	if len(tmp.Details) == 0 {
		return models.ReceiptDetail{}, fmt.Errorf("parcel is incomplete")
	}
	return tmp.Details[0], nil
}

func (h handler) submitReceipt(c fiber.Ctx) error {
	var row models.Receipt
	if err := h.db.Preload("Details").Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	if err := assertReceiptHeader(&row); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if len(row.Details) == 0 {
		return response.BadRequest(c, "add at least one quantity line before submit")
	}
	if err := h.db.Model(&row).Update("Status", types.ReceiptSubmitted).Error; err != nil {
		return err
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	ct := types.ReceiptContent
	if err := h.svc.Initiate(ct, row.ID, &user, row.DocumentNumber, row.DocumentNumber); err != nil {
		return err
	}
	return response.OkMessage(c, "Receipt submitted for approval")
}

func (h handler) convertReceipt(c fiber.Ctx) error {
	var in struct {
		Details []invsvc.QtyOverride `json:"details"`
	}
	_ = c.Bind().JSON(&in)
	row, err := h.svc.CreateFinalFromProvision(c.Params("uid"), middleware.GetUserIDFromContext(c), in.Details)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Created(c, row)
}

func (h handler) listITT(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.IttTransfer{})).
		Select("[IttTransfer].*").
		Joins("LEFT JOIN [Customer] AS [FromCustomer] ON [FromCustomer].ID = [IttTransfer].FromCustomerID").
		Joins("LEFT JOIN [Customer] AS [ToCustomer] ON [ToCustomer].ID = [IttTransfer].ToCustomerID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [IttTransfer].ProductID").
		Preload("FromCustomer").Preload("ToCustomer").Preload("Product")
	q, err = filterDocStatus(c, q, "[IttTransfer].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[IttTransfer].DocumentNumber",
		"[FromCustomer].Name", "[FromCustomer].Code", "[ToCustomer].Name", "[ToCustomer].Code",
		"[Product].Name", "[Product].Code",
	)
	return response.ServeList(c, response.ListOpts[models.IttTransfer]{
		Query: q, Search: search,
		DateColumn:  "[IttTransfer].TransferDate",
		DefaultSort: "[IttTransfer].TransferDate",
		TieBreak:    "[IttTransfer].ID",
		Sort: map[string]string{
			"documentNumber": "[IttTransfer].DocumentNumber",
			"transferDate":   "[IttTransfer].TransferDate",
			"quantity":       "[IttTransfer].Quantity",
			"status":         "[IttTransfer].Status",
		},
		Sheet: "ITT", File: "itt",
		Headers: []any{"Document", "Date", "From", "To", "Product", "Quantity", "Status"},
		MapRow: func(r models.IttTransfer) []any {
			return []any{r.DocumentNumber, r.TransferDate.Format("2006-01-02"), r.FromCustomer.Code, r.ToCustomer.Code, r.Product.Code, r.Quantity.String(), string(r.Status)}
		},
	})
}

func (h handler) getITT(c fiber.Ctx) error {
	var row models.IttTransfer
	if err := models.PreloadCreatedBy(h.db).Preload("FromCustomer").Preload("ToCustomer").Preload("Product").
		Preload("Vessel").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) listZerol(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.ZerolizationTransfer{})).
		Preload("Customer").Preload("Product").Preload("FromVessel").Preload("ToVessel")
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber")
	return response.ServeList(c, response.ListOpts[models.ZerolizationTransfer]{
		Query: q, Search: search,
		DateColumn:  "TransferDate",
		DefaultSort: "TransferDate",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"transferDate":   "TransferDate",
			"quantity":       "Quantity",
			"status":         "Status",
		},
		Sheet: "Zerolization", File: "zerolization",
		Headers: []any{"Document", "Date", "Quantity", "Status"},
		MapRow: func(r models.ZerolizationTransfer) []any {
			return []any{r.DocumentNumber, r.TransferDate.Format("2006-01-02"), r.Quantity.String(), string(r.Status)}
		},
	})
}

func (h handler) createEvent(c fiber.Ctx) error {
	var in invsvc.InboundEvent
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	row, err := h.svc.IngestEvent(in)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	recordAudit(c, types.ActionCreate, row.UID, types.InventoryEventContent, "inventory event "+row.MessageID+" ingested", nil, row)
	return response.Created(c, row)
}

func (h handler) listEvents(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := response.ApplyLike(h.db.WithContext(c.Context()).Model(&models.InventoryEventLog{}), search,
		"MessageID", "CustomerCode", "ProductCode", "VesselCode", "OrderNumber", "EventType",
	)
	return response.ServeList(c, response.ListOpts[models.InventoryEventLog]{
		Query: q, Search: search,
		DateColumn:  "OccurredAt",
		DefaultSort: "OccurredAt",
		Sort: map[string]string{
			"occurredAt":   "OccurredAt",
			"eventType":    "EventType",
			"customerCode": "CustomerCode",
			"orderNumber":  "OrderNumber",
		},
		Sheet: "Events", File: "inventory_events",
		Headers: []any{"When", "Type", "Customer", "Product", "Order", "Quantity", "Posted"},
		MapRow: func(r models.InventoryEventLog) []any {
			return []any{r.OccurredAt.Format("2006-01-02 15:04"), string(r.EventType), r.CustomerCode, r.ProductCode, r.OrderNumber, r.Quantity.String(), r.Posted}
		},
	})
}

func (h handler) listReservations(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := h.db.WithContext(c.Context()).Model(&models.StockReservation{})
	if status := strings.TrimSpace(c.Query("status")); status == "all" {
		// no status filter
	} else if status != "" {
		q = q.Where("Status = ?", status)
	} else {
		q = q.Where("Status = ?", types.ReservationOpen)
	}
	q = response.ApplyLike(q, search, "OrderNumber")
	return response.ServeList(c, response.ListOpts[models.StockReservation]{
		Query: q, Search: search,
		DateColumn:  "CreatedAt",
		DefaultSort: "CreatedAt",
		Sort: map[string]string{
			"orderNumber": "OrderNumber",
			"createdAt":   "CreatedAt",
			"quantity":    "Quantity",
			"status":      "Status",
		},
		Sheet: "Reservations", File: "reservations",
		Headers: []any{"Order", "Quantity", "Status", "Created"},
		MapRow: func(r models.StockReservation) []any {
			return []any{r.OrderNumber, r.Quantity.String(), string(r.Status), r.CreatedAt.Format("2006-01-02")}
		},
	})
}

func (h handler) releaseReservation(c fiber.Ctx) error {
	if err := h.svc.ReleaseReservation(c.Params("order")); err != nil {
		return err
	}
	return response.OkMessage(c, "Reservation released")
}

func (h handler) createITT(c fiber.Ctx) error {
	var in struct {
		TransferDate   time.Time `json:"transferDate"`
		FromCustomerID string    `json:"fromCustomerId"`
		ToCustomerID   string    `json:"toCustomerId"`
		ProductID      string    `json:"productId"`
		VesselID       string    `json:"vesselId"`
		VesselDate     string    `json:"vesselDate"`
		StockStatusID  string    `json:"stockStatusId"`
		DepotID        string    `json:"depotId"`
		Quantity       string    `json:"quantity"`
		FinancialHold  bool      `json:"financialHold"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	fromID, err := lookupID[models.Customer](h.db, in.FromCustomerID)
	if err != nil {
		return response.BadRequest(c, "from customer not found")
	}
	toID, err := lookupID[models.Customer](h.db, in.ToCustomerID)
	if err != nil {
		return response.BadRequest(c, "to customer not found")
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.BadRequest(c, "product not found")
	}
	vid, err := lookupID[models.Vessel](h.db, in.VesselID)
	if err != nil {
		return response.BadRequest(c, "vessel not found")
	}
	sid, err := lookupID[models.StockStatus](h.db, in.StockStatusID)
	if err != nil {
		return response.BadRequest(c, "status not found")
	}
	qty, err := decimal.NewFromString(in.Quantity)
	if err != nil || !qty.IsPositive() {
		return response.BadRequest(c, "quantity is required")
	}
	vd := in.TransferDate
	if in.VesselDate != "" {
		if parsed, err := time.Parse("2006-01-02", in.VesselDate); err == nil {
			vd = parsed
		}
	}
	row := models.IttTransfer{
		TransferDate:   in.TransferDate,
		FromCustomerID: fromID,
		ToCustomerID:   toID,
		ProductID:      pid,
		VesselID:       vid,
		VesselDate:     vd,
		StockStatusID:  sid,
		Quantity:       qty,
		CubicMeter:     qty,
		FinancialHold:  in.FinancialHold,
		Status:         types.DocDraft,
		CreatedByID:    middleware.GetUserIDFromContext(c),
	}
	if row.TransferDate.IsZero() {
		row.TransferDate = time.Now()
	}
	if in.DepotID != "" {
		did, err := lookupID[models.Depot](h.db, in.DepotID)
		if err != nil {
			return response.BadRequest(c, "depot not found")
		}
		row.DepotID = &did
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "itt", "ITT")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return attach.CopyCustomerFiles(tx, fromID, row.ID, types.IttTransferContent, row.CreatedByID)
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, row.UID, types.IttTransferContent, "ITT "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) submitITT(c fiber.Ctx) error {
	var row models.IttTransfer
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	if err := h.db.Model(&row).Update("Status", "submitted").Error; err != nil {
		return err
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := h.svc.Initiate(types.IttTransferContent, row.ID, &user, row.DocumentNumber, row.DocumentNumber); err != nil {
		return err
	}
	return response.OkMessage(c, "ITT submitted for approval")
}

func (h handler) createZerol(c fiber.Ctx) error {
	var in models.ZerolizationTransfer
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	in.CreatedByID = middleware.GetUserIDFromContext(c)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "zerol", "ZEROL")
		if err != nil {
			return err
		}
		in.DocumentNumber = n
		return tx.Omit("Customer", "Product", "FromVessel", "ToVessel").Create(&in).Error
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, in.UID, types.ZerolizationContent, "zerolization "+in.DocumentNumber+" created", nil, in)
	return response.Created(c, in)
}

func (h handler) submitZerol(c fiber.Ctx) error {
	var row models.ZerolizationTransfer
	if err := h.db.Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	if err := h.db.Model(&row).Update("Status", "submitted").Error; err != nil {
		return err
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := h.svc.Initiate(types.ZerolizationContent, row.ID, &user, row.DocumentNumber, row.DocumentNumber); err != nil {
		return err
	}
	return response.OkMessage(c, "Zerolization submitted for approval")
}

func (h handler) createDip(c fiber.Ctx) error {
	var in models.PhysicalDip
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	in.CreatedByID = middleware.GetUserIDFromContext(c)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&in).Error; err != nil {
			return err
		}
		return invsvc.ApplyDip(tx, &in)
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, in.UID, types.PhysicalDipContent, "physical dip recorded", nil, in)
	return response.Created(c, in)
}

func (h handler) listDips(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := h.db.WithContext(c.Context()).Model(&models.PhysicalDip{}).
		Select("[PhysicalDip].*").
		Joins("LEFT JOIN [Tank] ON [Tank].ID = [PhysicalDip].TankID").
		Preload("Tank")
	q = response.ApplyLike(q, search, "[Tank].Code", "[Tank].Name")
	return response.ServeList(c, response.ListOpts[models.PhysicalDip]{
		Query: q, Search: search,
		DateColumn:  "[PhysicalDip].DipDate",
		DefaultSort: "[PhysicalDip].DipDate",
		TieBreak:    "[PhysicalDip].ID",
		Sort: map[string]string{
			"dipDate":  "[PhysicalDip].DipDate",
			"observed": "[PhysicalDip].Observed",
		},
		Sheet: "Dips", File: "dips",
		Headers: []any{"Date", "Tank", "Observed", "At 20"},
		MapRow: func(r models.PhysicalDip) []any {
			return []any{r.DipDate.Format("2006-01-02"), r.Tank.Code, r.Observed.String(), r.At20.String()}
		},
	})
}

func (h handler) createLine(c fiber.Ctx) error {
	var in models.LineContent
	if err := c.Bind().JSON(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	in.CreatedByID = middleware.GetUserIDFromContext(c)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&in).Error; err != nil {
			return err
		}
		return invsvc.ApplyLineContent(tx, &in)
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, in.UID, types.LineContentContent, "line content recorded", nil, in)
	return response.Created(c, in)
}

func (h handler) listMovements(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := h.db.WithContext(c.Context()).Model(&models.StockMovement{})
	if p := strings.TrimSpace(c.Query("provision")); p == "1" || strings.EqualFold(p, "true") {
		q = q.Where("IsProvision = ?", true)
	} else if p == "0" || strings.EqualFold(p, "false") {
		q = q.Where("IsProvision = ?", false)
	}
	q = response.ApplyLike(q, search, "TransactionType", "ReferenceType")
	return response.ServeList(c, response.ListOpts[models.StockMovement]{
		Query: q, Search: search,
		DateColumn:  "TransactionDate",
		DefaultSort: "TransactionDate",
		Sort: map[string]string{
			"transactionDate": "TransactionDate",
			"transactionType": "TransactionType",
			"quantity":        "Quantity",
			"referenceType":   "ReferenceType",
		},
		Sheet: "Movements", File: "movements",
		Headers: []any{"Date", "Type", "Quantity", "Reference", "Provision", "Hold"},
		MapRow: func(r models.StockMovement) []any {
			return []any{r.TransactionDate.Format("2006-01-02"), string(r.TransactionType), r.Quantity.String(), r.ReferenceType, r.IsProvision, r.FinancialHold}
		},
	})
}
