package orders

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	ordersvc "dfms/internal/orders"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type glrInput struct {
	OrderDate             time.Time `json:"orderDate"`
	Description           string    `json:"description"`
	CustomerID            string    `json:"customerId"`
	ProductID             string    `json:"productId"`
	ByProductID           string    `json:"byProductId"`
	ByProductQuantity     string    `json:"byProductQuantity"`
	StockStatusID         string    `json:"stockStatusId"`
	Quantity              string    `json:"quantity"`
	CustomerOrderNumber   string    `json:"customerOrderNumber"`
	LoadingOrderAvailable bool      `json:"loadingOrderAvailable"`
	ValidContract         bool      `json:"validContract"`
	Notes                 string    `json:"notes"`
}

type glrLineIn struct {
	TransporterID       string `json:"transporterId"`
	DriverID            string `json:"driverId"`
	TruckID             string `json:"truckId"`
	DestinationID       string `json:"destinationId"`
	DistrictID          string `json:"districtId"`
	DestinationName     string `json:"destinationName"`
	DistrictName        string `json:"districtName"`
	ProductID           string `json:"productId"`
	CustomerOrderNumber string `json:"customerOrderNumber"`
	RequestedQty        string `json:"requestedQty"`
	ExpirationDate      string `json:"expirationDate"`
	EwuraLicense        string `json:"ewuraLicense"`
	OrderDate           string `json:"orderDate"`
}

type glrVesselIn struct {
	VesselID      string `json:"vesselId"`
	VesselDate    string `json:"vesselDate"`
	ProductID     string `json:"productId"`
	Quantity      string `json:"quantity"`
	FinancialHold bool   `json:"financialHold"`
}

func (h handler) listGLR(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}

	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.GantryLoadingRequest{})).
		Select("[GantryLoadingRequest].*").
		Joins("LEFT JOIN [Customer] ON [Customer].ID = [GantryLoadingRequest].CustomerID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [GantryLoadingRequest].ProductID").
		Joins("LEFT JOIN [Product] AS [ByProduct] ON [ByProduct].ID = [GantryLoadingRequest].ByProductID").
		Joins("LEFT JOIN [StockStatus] ON [StockStatus].ID = [GantryLoadingRequest].StockStatusID").
		Preload("Customer").Preload("Product").Preload("ByProduct").Preload("StockStatus").
		Preload("Lines", func(db *gorm.DB) *gorm.DB { return db.Select("ID", "RequestID") })
	q, err = filterOrderStatus(c, q, "[GantryLoadingRequest].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[GantryLoadingRequest].DocumentNumber", "[GantryLoadingRequest].Description",
		"[GantryLoadingRequest].BatchNumber", "[GantryLoadingRequest].CustomerOrderNumber",
		"[Customer].Name", "[Customer].Code", "[Product].Name", "[Product].Code",
		"[ByProduct].Name", "[ByProduct].Code", "[StockStatus].Name",
	)
	return response.ServeList(c, response.ListOpts[models.GantryLoadingRequest]{
		Query: q, Search: search,
		DateColumn:  "[GantryLoadingRequest].OrderDate",
		DefaultSort: "[GantryLoadingRequest].OrderDate",
		TieBreak:    "[GantryLoadingRequest].ID",
		Sort: map[string]string{
			"documentNumber": "[GantryLoadingRequest].DocumentNumber",
			"orderDate":      "[GantryLoadingRequest].OrderDate",
			"status":         "[GantryLoadingRequest].Status",
			"quantity":       "[GantryLoadingRequest].Quantity",
			"customer":       "[Customer].Name",
			"productStatus":  "[StockStatus].Name",
		},
		Sheet: "ILR", File: "ilr",
		Headers: []any{"Document", "Date", "Customer", "Product status", "Product", "Quantity", "By-product", "By-product qty", "Trucks", "Status"},
		MapRow: func(r models.GantryLoadingRequest) []any {
			byName, byQty := "", "0"
			if r.ByProduct != nil {
				byName = r.ByProduct.Name
				byQty = r.ByProductQuantity.String()
			}
			return []any{
				r.DocumentNumber, r.OrderDate.Format("2006-01-02"), r.Customer.Name,
				r.StockStatus.Name, r.Product.Name, r.Quantity.String(),
				byName, byQty, len(r.Lines), string(r.Status),
			}
		},
	})
}

func (h handler) getGLR(c fiber.Ctx) error {
	row, err := h.loadGLR(c.Params("uid"))
	if err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) loadGLR(uid string) (models.GantryLoadingRequest, error) {
	var row models.GantryLoadingRequest
	err := models.PreloadCreatedBy(h.db).Preload("Customer").Preload("Product").Preload("ByProduct").Preload("StockStatus").
		Preload("Lines.Transporter").Preload("Lines.Driver").Preload("Lines.Truck").
		Preload("Lines.Product").Preload("Lines.ByProduct").
		Preload("Lines.ToDestination").Preload("Lines.ToDistrict").
		Preload("Vessels.Vessel").Preload("Vessels.Product").Preload("Vessels.StockStatus").
		Preload("StockPositions.Product").Preload("Outstanding").Preload("Charges").
		Where("UID = ?", uid).First(&row).Error
	if err != nil {
		return row, err
	}
	row.IloExpiryDays = ordersvc.IloExpiryDays()
	row.Approvals = ordersvc.DocumentApprovals(h.db, types.GantryLoadingRequestContent, row.ID, row.ApprovalTrail).AsILR()
	return row, nil
}

func (h handler) createGLR(c fiber.Ctx) error {
	var in glrInput
	if err := bindBody(c, &in); err != nil {
		return err
	}
	desc := strings.TrimSpace(in.Description)
	if desc == "" {
		return response.UnprocessableEntity(c, errors.New("description is required"))
	}
	cid, err := lookupID[models.Customer](h.db, in.CustomerID)
	if err != nil {
		return response.BadRequest(c, "customer not found")
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.BadRequest(c, "product not found")
	}
	sid, err := lookupID[models.StockStatus](h.db, in.StockStatusID)
	if err != nil {
		return response.BadRequest(c, "status not found")
	}
	var status models.StockStatus
	if err := h.db.First(&status, sid).Error; err != nil {
		return response.BadRequest(c, "status not found")
	}
	qty, err := parsePositiveDec(in.Quantity, "quantity")
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	byQty := parseDec(in.ByProductQuantity)
	byID, err := lookupIDOpt[models.Product](h.db, in.ByProductID)
	if err != nil {
		return response.BadRequest(c, "by-product not found")
	}
	if err := validateByProduct(pid, byID, byQty, status.IsTransit); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := validateGantryPair(h.db, pid, byID); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	userID := middleware.GetUserIDFromContext(c)
	row := models.GantryLoadingRequest{
		OrderDate:             in.OrderDate,
		Description:           desc,
		CustomerID:            cid,
		ProductID:             pid,
		ByProductID:           byID,
		ByProductQuantity:     byQty,
		StockStatusID:         sid,
		Quantity:              qty,
		CustomerOrderNumber:   strings.TrimSpace(in.CustomerOrderNumber),
		LoadingOrderAvailable: in.LoadingOrderAvailable,
		ValidContract:         in.ValidContract,
		Notes:                 strings.TrimSpace(in.Notes),
		Status:                types.OrderDraft,
		CreatedByID:           userID,
		Outstanding:           &models.GantryCustomerOutstanding{},
		Charges:               defaultOutstandingCharges(),
	}
	if row.OrderDate.IsZero() {
		row.OrderDate = time.Now()
	}
	if err := h.refreshStockPositions(&row); err != nil {
		return err
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "ilr", "ILR")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		row.BatchNumber = n
		if len(row.BatchNumber) > 8 {
			row.BatchNumber = row.BatchNumber[len(row.BatchNumber)-8:]
		}
		if err := tx.Omit("Customer", "Product", "ByProduct", "StockStatus", "Attachments", "Approvals").Create(&row).Error; err != nil {
			return err
		}
		return copyCustomerAttachments(tx, cid, row.ID, userID)
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.GantryLoadingRequestContent,
		"ILR "+row.DocumentNumber+" created", nil, row)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return response.Created(c, row)
	}
	return response.Created(c, out)
}

func (h handler) updateGLR(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	before := row
	var in glrInput
	if err := bindBody(c, &in); err != nil {
		return err
	}
	desc := strings.TrimSpace(in.Description)
	if desc == "" {
		return response.UnprocessableEntity(c, errors.New("description is required"))
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return response.BadRequest(c, "product not found")
	}
	sid, err := lookupID[models.StockStatus](h.db, in.StockStatusID)
	if err != nil {
		return response.BadRequest(c, "status not found")
	}
	var status models.StockStatus
	if err := h.db.First(&status, sid).Error; err != nil {
		return response.BadRequest(c, "status not found")
	}
	qty, err := parsePositiveDec(in.Quantity, "quantity")
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	byQty := parseDec(in.ByProductQuantity)
	byID, err := lookupIDOpt[models.Product](h.db, in.ByProductID)
	if err != nil {
		return response.BadRequest(c, "by-product not found")
	}
	if err := validateByProduct(pid, byID, byQty, status.IsTransit); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := validateGantryPair(h.db, pid, byID); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if !in.OrderDate.IsZero() {
		row.OrderDate = in.OrderDate
	}
	row.Description = desc
	row.ProductID = pid
	row.ByProductID = byID
	row.ByProductQuantity = byQty
	row.StockStatusID = sid
	row.Quantity = qty
	row.LoadingOrderAvailable = in.LoadingOrderAvailable
	row.ValidContract = in.ValidContract
	row.Notes = strings.TrimSpace(in.Notes)
	if err := h.refreshStockPositions(&row); err != nil {
		return err
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Customer", "Product", "ByProduct", "StockStatus", "Lines", "Vessels",
			"StockPositions", "Outstanding", "Charges", "Attachments", "Approvals").Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("RequestID = ?", row.ID).Delete(&models.GantryStockPosition{}).Error; err != nil {
			return err
		}
		for i := range row.StockPositions {
			row.StockPositions[i].RequestID = row.ID
			row.StockPositions[i].ID = 0
			if err := tx.Create(&row.StockPositions[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.GantryLoadingRequestContent,
		"ILR "+row.DocumentNumber+" updated", before, row)
	out, loadErr := h.loadGLR(row.UID)
	if loadErr != nil {
		return okUpdate(c, row, before, row)
	}
	return okUpdate(c, out, before, row)
}

func (h handler) loadDraft(uid string) (models.GantryLoadingRequest, error) {
	var row models.GantryLoadingRequest
	if err := h.db.Preload("Vessels").Preload("Lines").Preload("StockStatus").
		Where("UID = ?", uid).First(&row).Error; err != nil {
		return row, err
	}
	if row.Status != types.OrderDraft {
		return row, errNotDraft
	}
	return row, nil
}

func (h handler) refreshStockPositions(row *models.GantryLoadingRequest) error {
	pos, err := h.stockPosition(row.CustomerID, row.ProductID, row.ID, row.Quantity)
	if err != nil {
		return err
	}
	row.SnapshotFinal = pos.TotalBalance
	row.SnapshotHold = pos.HoldQty
	row.SnapshotFree = pos.FreeQty
	row.StockPositions = []models.GantryStockPosition{pos}
	if row.ByProductID != nil {
		bpos, err := h.stockPosition(row.CustomerID, *row.ByProductID, row.ID, row.ByProductQuantity)
		if err != nil {
			return err
		}
		row.StockPositions = append(row.StockPositions, bpos)
	}
	return nil
}

func (h handler) refreshGLRStock(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		if errors.Is(err, errNotDraft) {
			return h.getGLR(c)
		}
		return draftErr(c, err)
	}
	if err := h.refreshStockPositions(&row); err != nil {
		return err
	}
	_ = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Updates(map[string]any{
			"SnapshotFinal": row.SnapshotFinal,
			"SnapshotHold":  row.SnapshotHold,
			"SnapshotFree":  row.SnapshotFree,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("RequestID = ?", row.ID).Delete(&models.GantryStockPosition{}).Error; err != nil {
			return err
		}
		for i := range row.StockPositions {
			row.StockPositions[i].RequestID = row.ID
			row.StockPositions[i].ID = 0
			if err := tx.Create(&row.StockPositions[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return response.OkDetail(c, out)
}

func (h handler) buildILRLine(ln glrLineIn, req models.GantryLoadingRequest, customer models.Customer) (models.GantryLoadingLine, error) {
	var out models.GantryLoadingLine
	tid, err := lookupID[models.Transporter](h.db, ln.TransporterID)
	if err != nil {
		return out, errors.New("transporter is required")
	}
	did, err := lookupID[models.Driver](h.db, ln.DriverID)
	if err != nil {
		return out, errors.New("driver is required")
	}
	truckID, err := lookupID[models.Truck](h.db, ln.TruckID)
	if err != nil {
		return out, errors.New("truck is required")
	}
	qty, err := parsePositiveDec(ln.RequestedQty, "requested quantity")
	if err != nil {
		return out, err
	}
	var transporter models.Transporter
	var driver models.Driver
	var truck models.Truck
	if err := h.db.First(&transporter, tid).Error; err != nil {
		return out, errors.New("transporter is required")
	}
	if err := h.db.First(&driver, did).Error; err != nil {
		return out, errors.New("driver is required")
	}
	if err := h.db.First(&truck, truckID).Error; err != nil {
		return out, errors.New("truck is required")
	}
	lineProduct := req.ProductID
	if strings.TrimSpace(ln.ProductID) != "" {
		lineProduct, err = lookupID[models.Product](h.db, ln.ProductID)
		if err != nil {
			return out, errors.New("product not found")
		}
	}
	if err := matchingOrderProduct(req, lineProduct); err != nil {
		return out, err
	}
	days := ordersvc.IloExpiryDays()
	exp, err := parseDate(ln.ExpirationDate)
	if err != nil || exp == nil {
		d := req.OrderDate.Add(time.Duration(days) * 24 * time.Hour)
		exp = &d
	}
	if exp.Before(req.OrderDate.Truncate(24 * time.Hour)) {
		return out, errors.New("expiration must be on or after the request date")
	}
	lic := strings.TrimSpace(ln.EwuraLicense)
	if lic == "" {
		lic = customer.EwuraLicense
	}
	destID, distID, destName, distName, err := h.resolveLinePlace(req, ln, lic)
	if err != nil {
		return out, err
	}
	plate := models.TruckComboPlate(truck.PlateNumber, truck.Trailer, truck.TrailerTwo)
	out = models.GantryLoadingLine{
		CustomerOrderNumber: strings.TrimSpace(firstNonEmpty(ln.CustomerOrderNumber, req.CustomerOrderNumber)),
		ProductID:           lineProduct,
		TransporterID:       &tid,
		TransporterName:     transporter.Name,
		DriverID:            &did,
		DriverName:          driver.Name,
		TruckID:             &truckID,
		TruckPlate:          plate,
		HorsePlate:          truck.PlateNumber,
		TrailerOnePlate:     truck.Trailer,
		TrailerTwoPlate:     truck.TrailerTwo,
		DestinationID:       destID,
		Destination:         destName,
		DistrictID:          distID,
		District:            distName,
		EwuraLicense:        lic,
		ExpirationDate:      exp,
		RequestedQty:        qty,
		IsActive:            true,
		Status:              types.OrderDraft,
	}
	return out, nil
}

func (h handler) resolveLinePlace(req models.GantryLoadingRequest, ln glrLineIn, license string) (*uint, *uint, string, string, error) {
	if req.StockStatus.IsTransit {
		destUID := strings.TrimSpace(ln.DestinationID)
		if destUID == "" {
			return nil, nil, "", "", errors.New("destination is required")
		}
		destID, err := lookupID[models.Destination](h.db, destUID)
		if err != nil {
			return nil, nil, "", "", errors.New("destination is required")
		}
		var dest models.Destination
		if err := h.db.First(&dest, destID).Error; err != nil {
			return nil, nil, "", "", errors.New("destination is required")
		}
		if !dest.IsCountry {
			return nil, nil, "", "", errors.New("transit destination must be a country")
		}
		return &destID, nil, dest.Name, "N/A", nil
	}
	id1, id2, n1, n2, err := h.placeFromLicenseNumber(license)
	if err != nil {
		return nil, nil, "", "", err
	}
	if n1 == "" {
		n1 = strings.TrimSpace(ln.DestinationName)
	}
	if n2 == "" {
		n2 = strings.TrimSpace(ln.DistrictName)
	}
	return id1, id2, n1, n2, nil
}

func (h handler) placeFromLicenseNumber(license string) (*uint, *uint, string, string, error) {
	license = strings.TrimSpace(license)
	if license == "" {
		return nil, nil, "", "", errors.New("EWURA license is required")
	}
	var lic models.EwuraPetroleumLicense
	if err := h.db.Where("LicenseNumber = ?", license).First(&lic).Error; err != nil {
		return nil, nil, "", "", errors.New("EWURA license not found")
	}
	var dist models.District
	if lic.DistrictName != "" {
		if h.db.Where("Name = ?", lic.DistrictName).First(&dist).Error == nil {
			var dest models.Destination
			if h.db.First(&dest, dist.DestinationID).Error == nil {
				return &dest.ID, &dist.ID, dest.Name, dist.Name, nil
			}
		}
	}
	if lic.RegionName != "" {
		var dest models.Destination
		if h.db.Where("Name = ?", lic.RegionName).First(&dest).Error == nil {
			return &dest.ID, nil, dest.Name, lic.DistrictName, nil
		}
	}
	return nil, nil, lic.RegionName, lic.DistrictName, nil
}

func (h handler) buildILRVessel(v glrVesselIn, req models.GantryLoadingRequest) (models.GantryRequestVessel, error) {
	var out models.GantryRequestVessel
	vid, err := lookupID[models.Vessel](h.db, v.VesselID)
	if err != nil {
		return out, errors.New("vessel not found")
	}
	qty, err := parsePositiveDec(v.Quantity, "vessel quantity")
	if err != nil {
		return out, err
	}
	vd, err := parseDate(v.VesselDate)
	if err != nil || vd == nil {
		return out, errors.New("vessel date is required")
	}
	if vd.After(req.OrderDate.Truncate(24 * time.Hour)) {
		return out, errors.New("vessel date cannot be after the ILR date")
	}
	productID := req.ProductID
	if strings.TrimSpace(v.ProductID) != "" {
		productID, err = lookupID[models.Product](h.db, v.ProductID)
		if err != nil {
			return out, errors.New("vessel product not found")
		}
	}
	if err := matchingOrderProduct(req, productID); err != nil {
		return out, err
	}
	cap := orderedQty(req, productID)
	if qty.GreaterThan(cap) {
		return out, fmt.Errorf("vessel quantity cannot exceed the ordered quantity for this product")
	}
	return models.GantryRequestVessel{
		VesselID:      vid,
		VesselDate:    *vd,
		ProductID:     productID,
		StockStatusID: req.StockStatusID,
		Quantity:      qty,
		FinancialHold: v.FinancialHold,
	}, nil
}

func (h handler) stockPosition(customerID, productID, requestID uint, ordered decimal.Decimal) (models.GantryStockPosition, error) {
	pos, err := h.svc.ILRPosition(customerID, productID, requestID, ordered)
	if err != nil {
		return models.GantryStockPosition{}, err
	}
	return models.GantryStockPosition{
		ProductID:    productID,
		TotalBalance: pos.TotalBalance,
		HoldQty:      pos.HoldQty,
		FreeQty:      pos.FreeQty,
		FinalQty:     pos.FinalQty,
		Price:        decimal.Zero,
	}, nil
}

func validateByProduct(productID uint, byID *uint, byQty decimal.Decimal, transit bool) error {
	if byID != nil && *byID == productID {
		return errors.New("by-product must differ from the main product")
	}
	if byID == nil && byQty.IsPositive() {
		return errors.New("by-product is required when by-product quantity is greater than zero")
	}
	if byID != nil && byQty.LessThan(decimal.NewFromInt(1)) {
		return errors.New("by-product quantity must be at least 1")
	}
	if transit && byID != nil {
		return errors.New("transit orders do not allow a by-product")
	}
	return nil
}

func matchingOrderProduct(req models.GantryLoadingRequest, productID uint) error {
	if productID == req.ProductID {
		return nil
	}
	if req.ByProductID != nil && productID == *req.ByProductID {
		return nil
	}
	return errors.New("product must match the ordered product or by-product")
}

func orderedQty(req models.GantryLoadingRequest, productID uint) decimal.Decimal {
	if productID == req.ProductID {
		return req.Quantity
	}
	if req.ByProductID != nil && productID == *req.ByProductID {
		return req.ByProductQuantity
	}
	return decimal.Zero
}

func lookupIDOpt[T any](db *gorm.DB, uid string) (*uint, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, nil
	}
	id, err := lookupID[T](db, uid)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parsePositiveDec(s, field string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(strings.ReplaceAll(s, ",", "")))
	if err != nil || !d.IsPositive() {
		return decimal.Zero, fmt.Errorf("%s is required", field)
	}
	return d, nil
}

func parseDec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(strings.ReplaceAll(s, ",", "")))
	if err != nil {
		return decimal.Zero
	}
	return d
}

func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		d := t.UTC().Truncate(24 * time.Hour)
		return &d, nil
	}
	return nil, fmt.Errorf("invalid date")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func defaultOutstandingCharges() []models.GantryOutstandingCharge {
	return []models.GantryOutstandingCharge{
		{Charge: "Storage Debt", CurrencyCode: "TZS"},
		{Charge: "Storage Debt", CurrencyCode: "USD"},
		{Charge: "Weight & Measure", CurrencyCode: "TZS"},
		{Charge: "TBS Debt", CurrencyCode: "TZS"},
	}
}
