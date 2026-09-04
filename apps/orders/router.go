package orders

import (
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/attach"
	ordersvc "dfms/internal/orders"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func Router(app *fiber.App, svc *ordersvc.Service) {
	h := handler{db: db.Db, svc: svc}
	g := app.Group("/api/v1/orders", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	need := func(code string) fiber.Handler {
		return middleware.PermissionMiddleware(code)
	}
	ilrRead, ilrCreate, ilrUpdate := need(permissions.ILRRead), need(permissions.ILRCreate), need(permissions.ILRUpdate)
	ilrReview := middleware.PermissionMiddleware(permissions.ILRRead, permissions.WorkflowTasks)
	ilrWrite := middleware.PermissionMiddleware(permissions.ILRCreate, permissions.ILRUpdate)
	pdoRead, pdoCreate := need(permissions.PumpOverRead), need(permissions.PumpOverCreate)
	pdoReview := middleware.PermissionMiddleware(permissions.PumpOverRead, permissions.WorkflowTasks)
	pdoWrite := middleware.PermissionMiddleware(permissions.PumpOverCreate, permissions.PumpOverUpdate)
	repRead, repCreate := need(permissions.PumpReportRead), need(permissions.PumpReportCreate)
	repReview := middleware.PermissionMiddleware(permissions.PumpReportRead, permissions.WorkflowTasks)
	repWrite := middleware.PermissionMiddleware(permissions.PumpReportCreate, permissions.PumpReportUpdate)
	compRead, compCreate := need(permissions.CompRead), need(permissions.CompCreate)
	compReview := middleware.PermissionMiddleware(permissions.CompRead, permissions.WorkflowTasks)
	compWrite := middleware.PermissionMiddleware(permissions.CompCreate, permissions.CompUpdate)
	amdRead, amdCreate := need(permissions.AmendmentRead), need(permissions.AmendmentCreate)
	amdReview := middleware.PermissionMiddleware(permissions.AmendmentRead, permissions.WorkflowTasks)
	amdWrite := middleware.PermissionMiddleware(permissions.AmendmentCreate, permissions.AmendmentUpdate)

	g.Get("/loading", ilrRead, h.listGLR)
	g.Get("/loading-products", ilrRead, h.listILRProducts)
	g.Post("/loading", ilrCreate, h.createGLR)
	g.Get("/loading-lines", ilrRead, h.listLines)
	g.Get("/loading/:uid", ilrReview, h.getGLR)
	g.Put("/loading/:uid", ilrUpdate, h.updateGLR)
	g.Post("/loading/:uid/refresh-stock", ilrUpdate, h.refreshGLRStock)
	g.Post("/loading/:uid/vessels", ilrWrite, h.addGLRVessel)
	g.Put("/loading/:uid/vessels/:vid", ilrUpdate, h.updateGLRVessel)
	g.Delete("/loading/:uid/vessels/:vid", ilrUpdate, h.deleteGLRVessel)
	g.Post("/loading/:uid/lines", ilrWrite, h.addGLRLine)
	g.Put("/loading/:uid/lines/:lid", ilrUpdate, h.updateGLRLine)
	g.Delete("/loading/:uid/lines/:lid", ilrUpdate, h.deleteGLRLine)
	attach.Register(g, "/loading/:uid", ilrReview, ilrWrite, h.db, types.GantryLoadingRequestContent, h.attachGLR)
	g.Post("/loading/:uid/submit", need(permissions.ILRSubmit), h.submitGLRValidated)
	g.Post("/loading-lines/:uid/complete", need(permissions.ILRComplete), h.completeLine)

	g.Get("/pump-over", pdoRead, h.listPDO)
	g.Post("/pump-over", pdoCreate, h.createPDO)
	g.Get("/pump-over/:uid", pdoReview, h.getPDO)
	g.Post("/pump-over/:uid/submit", need(permissions.PumpOverSubmit), h.submitPDO)
	attach.Register(g, "/pump-over/:uid", pdoReview, pdoWrite, h.db, types.PumpOverRequestContent, h.attachPDO)
	g.Post("/pump-over-reports", repCreate, h.createReport)
	g.Get("/pump-over-reports", repRead, h.listReports)
	g.Get("/pump-over-reports/:uid", repReview, h.getReport)
	g.Post("/pump-over-reports/:uid/submit", need(permissions.PumpReportSubmit), h.submitReport)
	attach.Register(g, "/pump-over-reports/:uid", repReview, repWrite, h.db, types.PumpOverReportContent, h.attachReport)

	g.Get("/compartmentalizations", compRead, h.listComps)
	g.Post("/compartmentalizations", compCreate, h.createComp)
	g.Get("/compartmentalizations/:uid", compReview, h.getComp)
	g.Post("/compartmentalizations/:uid/lines", compWrite, h.saveCompLines)
	g.Post("/compartmentalizations/:uid/submit", need(permissions.CompSubmit), h.submitComp)
	attach.Register(g, "/compartmentalizations/:uid", compReview, compWrite, h.db, types.CompartmentalizationContent, h.attachComp)

	g.Get("/amendments", amdRead, h.listAmendments)
	g.Post("/amendments", amdCreate, h.createAmendment)
	g.Get("/amendments/:uid", amdReview, h.getAmendment)
	attach.Register(g, "/amendments/:uid", amdReview, amdWrite, h.db, types.OrderAmendmentContent, h.attachAmend)

	g.Get("/badges", compRead, h.listBadges)
	g.Post("/badges", compCreate, h.createBadge)
}

type handler struct {
	db  *gorm.DB
	svc *ordersvc.Service
}

func (h handler) listLines(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := h.db.WithContext(c.Context()).Model(&models.GantryLoadingLine{}).
		Select("[GantryLoadingLine].*").
		Joins("LEFT JOIN [GantryLoadingRequest] ON [GantryLoadingRequest].ID = [GantryLoadingLine].RequestID").
		Joins("LEFT JOIN [Customer] ON [Customer].ID = [GantryLoadingRequest].CustomerID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [GantryLoadingRequest].ProductID").
		Preload("Request.Customer").Preload("Request.Product")
	q, err = filterOrderStatus(c, q, "[GantryLoadingLine].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[GantryLoadingLine].DocumentNumber", "[GantryLoadingLine].TruckPlate",
		"[GantryLoadingLine].DriverName", "[GantryLoadingLine].Destination",
		"[GantryLoadingLine].TransporterName", "[GantryLoadingRequest].DocumentNumber",
		"[Customer].Name", "[Customer].Code", "[Product].Name", "[Product].Code",
	)
	return response.ServeList(c, response.ListOpts[models.GantryLoadingLine]{
		Query: q, Search: search,
		DateColumn:  "[GantryLoadingLine].CreatedAt",
		DefaultSort: "[GantryLoadingLine].CreatedAt",
		TieBreak:    "[GantryLoadingLine].ID",
		Sort: map[string]string{
			"documentNumber": "[GantryLoadingLine].DocumentNumber",
			"truckPlate":     "[GantryLoadingLine].TruckPlate",
			"requestedQty":   "[GantryLoadingLine].RequestedQty",
			"status":         "[GantryLoadingLine].Status",
			"createdAt":      "[GantryLoadingLine].CreatedAt",
		},
		Sheet: "ILO", File: "loading_lines",
		Headers: []any{"ILO", "ILR", "Customer", "Product", "Truck", "Driver", "Destination", "Requested", "Loaded", "Status"},
		MapRow: func(r models.GantryLoadingLine) []any {
			ilr, cust, prod := "", "", ""
			if r.Request != nil {
				ilr = r.Request.DocumentNumber
				cust = r.Request.Customer.Code
				prod = r.Request.Product.Code
			}
			return []any{r.DocumentNumber, ilr, cust, prod, r.TruckPlate, r.DriverName, r.Destination, r.RequestedQty.String(), r.LoadedQty.String(), string(r.Status)}
		},
	})
}

func (h handler) completeLine(c fiber.Ctx) error {
	var in struct {
		Quantity string `json:"quantity"`
	}
	if err := bindBody(c, &in); err != nil {
		return err
	}
	qty, err := decimal.NewFromString(in.Quantity)
	if err != nil {
		return response.BadRequest(c, "quantity is required")
	}
	if err := h.svc.CompleteLoadingLine(c.Params("uid"), qty); err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkMessage(c, "Loading posted to the stock ledger")
}

type pdoInput struct {
	OrderDate           time.Time `json:"orderDate"`
	CustomerID          string    `json:"customerId"`
	ProductID           string    `json:"productId"`
	StockStatusID       string    `json:"stockStatusId"`
	DepotID             string    `json:"depotId"`
	Quantity            string    `json:"quantity"`
	CustomerOrderNumber string    `json:"customerOrderNumber"`
	Notes               string    `json:"notes"`
	Vessels             []struct {
		VesselID      string `json:"vesselId"`
		VesselDate    string `json:"vesselDate"`
		StockStatusID string `json:"stockStatusId"`
		Quantity      string `json:"quantity"`
	} `json:"vessels"`
}

func (h handler) listPDO(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.PumpOverRequest{})).
		Select("[PumpOverRequest].*").
		Joins("LEFT JOIN [Customer] ON [Customer].ID = [PumpOverRequest].CustomerID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [PumpOverRequest].ProductID").
		Joins("LEFT JOIN [Depot] ON [Depot].ID = [PumpOverRequest].DepotID").
		Preload("Customer").Preload("Product").Preload("Depot").Preload("StockStatus").
		Preload("Vessels.Vessel").Preload("Vessels.StockStatus")
	q, err = filterOrderStatus(c, q, "[PumpOverRequest].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[PumpOverRequest].DocumentNumber", "[PumpOverRequest].CustomerOrderNumber",
		"[Customer].Name", "[Customer].Code", "[Product].Name", "[Product].Code", "[Depot].Code",
	)
	return response.ServeList(c, response.ListOpts[models.PumpOverRequest]{
		Query: q, Search: search,
		DateColumn:  "[PumpOverRequest].OrderDate",
		DefaultSort: "[PumpOverRequest].OrderDate",
		TieBreak:    "[PumpOverRequest].ID",
		Sort: map[string]string{
			"documentNumber": "[PumpOverRequest].DocumentNumber",
			"orderDate":      "[PumpOverRequest].OrderDate",
			"quantity":       "[PumpOverRequest].Quantity",
			"status":         "[PumpOverRequest].Status",
		},
		Sheet: "Pump-over", File: "pump_over",
		Headers: []any{"Document", "Date", "Customer", "Product", "Depot", "Quantity", "Status"},
		MapRow: func(r models.PumpOverRequest) []any {
			return []any{r.DocumentNumber, r.OrderDate.Format("2006-01-02"), r.Customer.Code, r.Product.Code, r.Depot.Code, r.Quantity.String(), string(r.Status)}
		},
	})
}

func (h handler) getPDO(c fiber.Ctx) error {
	var row models.PumpOverRequest
	if err := models.PreloadCreatedBy(h.db).Preload("Customer").Preload("Product").Preload("Depot").Preload("StockStatus").
		Preload("Vessels.Vessel").Preload("Vessels.StockStatus").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) createPDO(c fiber.Ctx) error {
	var in pdoInput
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cid, err := lookupID[models.Customer](h.db, in.CustomerID)
	if err != nil {
		return response.BadRequest(c, "customer not found")
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.BadRequest(c, "product not found")
	}
	did, err := lookupID[models.Depot](h.db, in.DepotID)
	if err != nil {
		return response.BadRequest(c, "depot not found")
	}
	headerStatus, _ := lookupID[models.StockStatus](h.db, in.StockStatusID)
	var vessels []models.PumpOverVessel
	var vesselQty decimal.Decimal
	for _, v := range in.Vessels {
		vid, err := lookupID[models.Vessel](h.db, v.VesselID)
		if err != nil {
			return response.BadRequest(c, "vessel not found")
		}
		vsid, err := lookupID[models.StockStatus](h.db, v.StockStatusID)
		if err != nil {
			if headerStatus == 0 {
				return response.BadRequest(c, "each vessel needs a stock status")
			}
			vsid = headerStatus
		}
		vd, _ := time.Parse("2006-01-02", v.VesselDate)
		vq, err := decimal.NewFromString(v.Quantity)
		if err != nil || !vq.IsPositive() {
			return response.BadRequest(c, "each vessel needs a quantity")
		}
		vessels = append(vessels, models.PumpOverVessel{
			VesselID: vid, VesselDate: vd, StockStatusID: vsid, Quantity: vq,
		})
		vesselQty = vesselQty.Add(vq)
		if headerStatus == 0 {
			headerStatus = vsid
		}
	}
	if len(vessels) == 0 {
		return response.BadRequest(c, "at least one vessel is required")
	}
	if headerStatus == 0 {
		return response.BadRequest(c, "status not found")
	}
	qty, err := decimal.NewFromString(in.Quantity)
	if err != nil || !qty.IsPositive() {
		qty = vesselQty
	}
	if !qty.IsPositive() {
		return response.BadRequest(c, "quantity is required")
	}
	_, _, _, free, err := h.svc.Snapshot(cid, pid)
	if err != nil {
		return err
	}
	row := models.PumpOverRequest{
		OrderDate:           in.OrderDate,
		CustomerID:          cid,
		ProductID:           pid,
		StockStatusID:       headerStatus,
		DepotID:             did,
		Quantity:            qty,
		CustomerOrderNumber: in.CustomerOrderNumber,
		Notes:               in.Notes,
		SnapshotFree:        free,
		Status:              types.OrderDraft,
		CreatedByID:         middleware.GetUserIDFromContext(c),
		Vessels:             vessels,
	}
	if row.OrderDate.IsZero() {
		row.OrderDate = time.Now()
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "pdo", "DO")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		parcels := row.Vessels
		row.Vessels = nil
		if err := tx.Omit("Customer", "Product", "StockStatus", "Depot", "Vessels").Create(&row).Error; err != nil {
			return err
		}
		for i := range parcels {
			parcels[i].RequestID = row.ID
			if err := tx.Omit("Vessel", "StockStatus").Create(&parcels[i]).Error; err != nil {
				return err
			}
		}
		row.Vessels = parcels
		return attach.CopyCustomerFiles(tx, cid, row.ID, types.PumpOverRequestContent, row.CreatedByID)
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.PumpOverRequestContent,
		"pump-over "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) submitPDO(c fiber.Ctx) error {
	return h.submitDoc(c, &models.PumpOverRequest{}, types.PumpOverRequestContent)
}

func (h handler) createReport(c fiber.Ctx) error {
	var in struct {
		RequestID       string `json:"requestId"`
		ActualDelivered string `json:"actualDelivered"`
		ActualReceived  string `json:"actualReceived"`
	}
	if err := bindBody(c, &in); err != nil {
		return err
	}
	rid, err := lookupID[models.PumpOverRequest](h.db, in.RequestID)
	if err != nil {
		return response.BadRequest(c, "pump-over request not found")
	}
	var req models.PumpOverRequest
	if err := h.db.First(&req, rid).Error; err != nil {
		return err
	}
	if req.Status != types.OrderApproved {
		return response.BadRequest(c, "pump-over request is not approved")
	}
	del, _ := decimal.NewFromString(in.ActualDelivered)
	recv, _ := decimal.NewFromString(in.ActualReceived)
	row := models.PumpOverReport{
		RequestID:       rid,
		ReportDate:      time.Now(),
		ActualDelivered: del,
		ActualReceived:  recv,
		Variance:        recv.Sub(del),
		Status:          types.OrderDraft,
		CreatedByID:     middleware.GetUserIDFromContext(c),
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "por", "POR")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		return tx.Create(&row).Error
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.PumpOverReportContent,
		"pump-over report "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) listReports(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.PumpOverReport{})).
		Select("[PumpOverReport].*").
		Joins("LEFT JOIN [PumpOverRequest] ON [PumpOverRequest].ID = [PumpOverReport].RequestID").
		Joins("LEFT JOIN [Customer] ON [Customer].ID = [PumpOverRequest].CustomerID").
		Preload("Request.Customer").Preload("Request.Product").Preload("Request.Depot")
	q, err = filterOrderStatus(c, q, "[PumpOverReport].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[PumpOverReport].DocumentNumber", "[PumpOverRequest].DocumentNumber",
		"[Customer].Name", "[Customer].Code",
	)
	return response.ServeList(c, response.ListOpts[models.PumpOverReport]{
		Query: q, Search: search,
		DateColumn:  "[PumpOverReport].ReportDate",
		DefaultSort: "[PumpOverReport].ReportDate",
		TieBreak:    "[PumpOverReport].ID",
		Sort: map[string]string{
			"documentNumber":  "[PumpOverReport].DocumentNumber",
			"reportDate":      "[PumpOverReport].ReportDate",
			"actualDelivered": "[PumpOverReport].ActualDelivered",
			"actualReceived":  "[PumpOverReport].ActualReceived",
			"status":          "[PumpOverReport].Status",
		},
		Sheet: "Pump-over reports", File: "pump_over_reports",
		Headers: []any{"Report", "Date", "Request", "Customer", "Delivered", "Received", "Status"},
		MapRow: func(r models.PumpOverReport) []any {
			return []any{r.DocumentNumber, r.ReportDate.Format("2006-01-02"), r.Request.DocumentNumber, r.Request.Customer.Code, r.ActualDelivered.String(), r.ActualReceived.String(), string(r.Status)}
		},
	})
}

func (h handler) getReport(c fiber.Ctx) error {
	var row models.PumpOverReport
	if err := models.PreloadCreatedBy(h.db).Preload("Request.Customer").Preload("Request.Product").Preload("Request.Depot").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) submitReport(c fiber.Ctx) error {
	return h.submitDoc(c, &models.PumpOverReport{}, types.PumpOverReportContent)
}

func (h handler) submitDoc(c fiber.Ctx, dest any, ct types.ContentType) error {
	if err := h.db.Where("UID = ?", c.Params("uid")).First(dest).Error; err != nil {
		return err
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	id, no := orderIdentity(dest)
	_ = h.db.Model(dest).Update("Status", types.OrderSubmitted)
	if err := h.svc.Initiate(ct, id, &user, no, no); err != nil {
		return err
	}
	return response.OkMessage(c, "Submitted for approval")
}

func orderIdentity(v any) (uint, string) {
	switch row := v.(type) {
	case *models.GantryLoadingRequest:
		return row.ID, row.DocumentNumber
	case *models.PumpOverRequest:
		return row.ID, row.DocumentNumber
	case *models.PumpOverReport:
		return row.ID, row.DocumentNumber
	default:
		return 0, ""
	}
}

func lookupID[T any](db *gorm.DB, uid string) (uint, error) {
	var row T
	if err := db.Where("UID = ?", uid).First(&row).Error; err != nil {
		return 0, err
	}
	switch v := any(&row).(type) {
	case *models.Customer:
		return v.ID, nil
	case *models.Product:
		return v.ID, nil
	case *models.StockStatus:
		return v.ID, nil
	case *models.Vessel:
		return v.ID, nil
	case *models.Depot:
		return v.ID, nil
	case *models.PumpOverRequest:
		return v.ID, nil
	case *models.GantryLoadingLine:
		return v.ID, nil
	case *models.Truck:
		return v.ID, nil
	case *models.TruckTank:
		return v.ID, nil
	case *models.Transporter:
		return v.ID, nil
	case *models.Driver:
		return v.ID, nil
	case *models.Destination:
		return v.ID, nil
	case *models.District:
		return v.ID, nil
	default:
		return 0, fiber.ErrBadRequest
	}
}

func (h handler) listComps(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.Compartmentalization{})).
		Select("[GantryCompartmentalization].*").
		Joins("LEFT JOIN [GantryLoadingLine] ON [GantryLoadingLine].ID = [GantryCompartmentalization].IloID").
		Preload("Ilo").Preload("Badge")
	q, err = filterOrderStatus(c, q, "[GantryCompartmentalization].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[GantryCompartmentalization].DocumentNumber", "[GantryCompartmentalization].CustomerName",
		"[GantryCompartmentalization].CustomerOrderNumber", "[GantryCompartmentalization].HorsePlate",
		"[GantryLoadingLine].DocumentNumber",
	)
	return response.ServeList(c, response.ListOpts[models.Compartmentalization]{
		Query: q, Search: search,
		DateColumn:  "[GantryCompartmentalization].CreatedAt",
		DefaultSort: "[GantryCompartmentalization].CreatedAt",
		TieBreak:    "[GantryCompartmentalization].ID",
		Sort: map[string]string{
			"documentNumber": "[GantryCompartmentalization].DocumentNumber",
			"status":         "[GantryCompartmentalization].Status",
			"createdAt":      "[GantryCompartmentalization].CreatedAt",
			"customer":       "[GantryCompartmentalization].CustomerName",
			"horsePlate":     "[GantryCompartmentalization].HorsePlate",
		},
		Sheet: "Compartmentalizations", File: "compartmentalizations",
		Headers: []any{"Document", "ILO", "Customer", "Horse", "Status"},
		MapRow: func(r models.Compartmentalization) []any {
			ilo := ""
			if r.Ilo.DocumentNumber != "" {
				ilo = r.Ilo.DocumentNumber
			}
			return []any{r.DocumentNumber, ilo, r.CustomerName, r.HorsePlate, string(r.Status)}
		},
	})
}

func (h handler) getComp(c fiber.Ctx) error {
	var row models.Compartmentalization
	if err := models.PreloadCreatedBy(h.db).Preload("Ilo.Request.Product").Preload("Lines.Product").Preload("Badge").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) createComp(c fiber.Ctx) error {
	var in createCompSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row, err := h.svc.CreateCompartmentalization(in.IloID, in.BadgeID, middleware.GetUserIDFromContext(c), in.ConfirmExpiry)
	if err != nil {
		return gateErr(c, err)
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.CompartmentalizationContent,
		"compartmentalization "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) saveCompLines(c fiber.Ctx) error {
	var in saveCompLinesSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	patches := make([]ordersvc.CompLinePatch, 0, len(in.Lines))
	for _, ln := range in.Lines {
		qty, _ := decimal.NewFromString(ln.Quantity)
		p := ordersvc.CompLinePatch{UID: ln.ID, Quantity: qty, TopSeal: ln.TopSeal, DipSeal: ln.DipSeal, BottomSeal: ln.BottomSeal}
		if ln.ProductID != "" {
			id, err := lookupID[models.Product](h.db, ln.ProductID)
			if err != nil {
				return response.BadRequest(c, "product not found")
			}
			p.ProductID = id
		}
		patches = append(patches, p)
	}
	if err := h.svc.SaveCompLines(c.Params("uid"), patches); err != nil {
		return response.BadRequest(c, err.Error())
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, c.Params("uid"), types.CompartmentalizationContent,
		"compartmentalization compartments saved", nil, patches)
	return response.OkMessage(c, "Compartments saved")
}

func (h handler) submitComp(c fiber.Ctx) error {
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := h.svc.SubmitCompartmentalization(c.Params("uid"), &user); err != nil {
		return response.BadRequest(c, err.Error())
	}
	recordAudit(c, types.ModuleOrders, types.ActionInitiate, c.Params("uid"), types.CompartmentalizationContent,
		"compartmentalization submitted for gantry approval", nil, nil)
	return response.OkMessage(c, "Submitted for gantry approval")
}

func (h handler) listAmendments(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.OrderAmendment{})).
		Select("[OrderAmendment].*").
		Joins("LEFT JOIN [GantryLoadingLine] ON [GantryLoadingLine].ID = [OrderAmendment].IloID").
		Preload("Ilo")
	q, err = filterOrderStatus(c, q, "[OrderAmendment].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[OrderAmendment].DocumentNumber", "[OrderAmendment].Kind",
		"[OrderAmendment].Destination", "[OrderAmendment].TruckPlate",
		"[OrderAmendment].DriverName", "[OrderAmendment].Notes",
		"[GantryLoadingLine].DocumentNumber",
	)
	return response.ServeList(c, response.ListOpts[models.OrderAmendment]{
		Query: q, Search: search,
		DateColumn:  "[OrderAmendment].CreatedAt",
		DefaultSort: "[OrderAmendment].CreatedAt",
		TieBreak:    "[OrderAmendment].ID",
		Sort: map[string]string{
			"documentNumber": "[OrderAmendment].DocumentNumber",
			"kind":           "[OrderAmendment].Kind",
			"requestedQty":   "[OrderAmendment].RequestedQty",
			"status":         "[OrderAmendment].Status",
		},
		Sheet: "Amendments", File: "loading_amendments",
		Headers: []any{"Document", "Kind", "ILO", "Quantity", "Status"},
		MapRow: func(r models.OrderAmendment) []any {
			return []any{r.DocumentNumber, string(r.Kind), r.Ilo.DocumentNumber, r.RequestedQty.String(), string(r.Status)}
		},
	})
}

func (h handler) createAmendment(c fiber.Ctx) error {
	var in createAmendmentSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	qty, _ := decimal.NewFromString(in.RequestedQty)
	var exp *time.Time
	if in.ExpirationDate != "" {
		if t, err := time.Parse("2006-01-02", in.ExpirationDate); err == nil {
			exp = &t
		}
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	kind := types.AmendmentKind(in.Kind)
	ain := ordersvc.AmendmentInput{
		Kind: kind, LineUID: in.IloID, RequestedQty: qty, ExpirationDate: exp,
		Destination: in.Destination, District: in.District, TruckPlate: in.TruckPlate,
		TransporterName: in.TransporterName, DriverName: in.DriverName, Notes: in.Notes,
	}
	if in.ProductID != "" {
		id, err := lookupID[models.Product](h.db, in.ProductID)
		if err != nil {
			return response.BadRequest(c, "product not found")
		}
		ain.ProductID = id
	}
	row, err := h.svc.CreateAmendment(ain, &user)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.OrderAmendmentContent,
		"amendment "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) listBadges(c fiber.Ctx) error {
	search, err := parseCatalogue(c)
	if err != nil {
		return err
	}
	q := response.ApplyLike(h.db.WithContext(c.Context()).Model(&models.RfidBadge{}), search, "Code")
	return response.ServeList(c, response.ListOpts[models.RfidBadge]{
		Query: q, Search: search, DefaultSort: "Code",
		Sort:         map[string]string{"code": "Code"},
		DumpIfNoPage: true,
	})
}

func (h handler) createBadge(c fiber.Ctx) error {
	var in models.RfidBadge
	if err := bindBody(c, &in); err != nil {
		return err
	}
	in.IsActive = true
	in.IsAvailable = true
	if err := h.db.Create(&in).Error; err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, in.UID, types.RfidBadgeContent,
		"RFID badge "+in.Code+" created", nil, in)
	return response.Created(c, in)
}
