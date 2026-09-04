package orders

import (
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/alma"
	"dfms/internal/ewura"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GateError is a create-time check the operator must fix (or confirm).
type GateError struct {
	Code string
	Msg  string
}

func (e *GateError) Error() string { return e.Msg }

func gate(code, msg string) error { return &GateError{Code: code, Msg: msg} }

// CreateCompartmentalization builds tank cells from the truck's active calibration.
// confirmExpiry is required when the ILO expires within 3 days.
func (s *Service) CreateCompartmentalization(lineUID, badgeUID string, userID uint, confirmExpiry bool) (*models.Compartmentalization, error) {
	var line models.GantryLoadingLine
	err := s.db.Preload("Request.Customer").Preload("Request.Product").Preload("Request.StockStatus").
		Preload("Request.ByProduct").Preload("Product").Preload("ByProduct").
		Preload("Driver").Preload("Truck").Preload("Transporter").
		Where("UID = ?", lineUID).First(&line).Error
	if err != nil {
		return nil, err
	}
	if !line.IsActive || line.Amended {
		return nil, fmt.Errorf("line %s is not available for compartmentalization", line.DocumentNumber)
	}
	if line.Status != types.OrderOpen {
		return nil, fmt.Errorf("line %s must be open (not yet loaded)", line.DocumentNumber)
	}
	var existing int64
	if err := s.db.Model(&models.Compartmentalization{}).
		Where("IloID = ? AND Amended = 0 AND IsActive = 1", line.ID).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, fmt.Errorf("ILO %s already has an active compartmentalization", line.DocumentNumber)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if line.ExpirationDate != nil {
		exp := line.ExpirationDate.UTC().Truncate(24 * time.Hour)
		if !exp.After(today) {
			return nil, gate("expired", "ILO "+line.DocumentNumber+" has expired — extend the order before compartmentalization")
		}
		if exp.Sub(today) <= 3*24*time.Hour && !confirmExpiry {
			days := int(exp.Sub(today).Hours() / 24)
			return nil, gate("nearExpiry", fmt.Sprintf("ILO %s expires in %d day(s). Continue to proceed, or extend the order first", line.DocumentNumber, days))
		}
	}
	if line.Driver != nil && line.Driver.LicenseExpires != nil {
		if !line.Driver.LicenseExpires.UTC().Truncate(24 * time.Hour).After(today) {
			return nil, gate("driverLicenceExpired", "Driver licence has expired — update the licence on the driver master before continuing")
		}
	}
	if line.Truck != nil && !types.VehicleTypeConfigured(line.Truck.VehicleType) {
		return nil, gate("truckTypeRequired", "Truck "+line.Truck.PlateNumber+" is not configured (vehicle type) — set type and calibration before compartmentalization")
	}

	var badge *models.RfidBadge
	if badgeUID != "" {
		var b models.RfidBadge
		if err := s.db.Where("UID = ? AND IsActive = 1 AND IsAvailable = 1", badgeUID).First(&b).Error; err != nil {
			return nil, fmt.Errorf("RFID badge is not available")
		}
		badge = &b
	}
	cells, err := s.calibrationCells(line)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "no active truck tanks") || strings.Contains(msg, "calibration first") {
			return nil, gate("calibrationRequired", msg)
		}
		if strings.Contains(msg, "no active calibration") {
			return nil, gate("calibrationExpired", "Truck calibration has expired or is missing — renew the chart before continuing")
		}
		return nil, err
	}
	var row models.Compartmentalization
	err = s.db.Transaction(func(tx *gorm.DB) error {
		transID, err := models.NextTransactionID(tx)
		if err != nil {
			return err
		}
		req := line.Request
		if req == nil {
			return fmt.Errorf("ILO %s has no ILR", line.DocumentNumber)
		}
		productCode := firstNonEmpty(line.Product.Code, req.Product.Code)
		productName := firstNonEmpty(line.Product.Name, req.Product.Name)
		productID := line.ProductID
		if productID == 0 {
			productID = req.ProductID
		}
		row = models.Compartmentalization{
			TransactionID:       transID,
			DocumentNumber:      line.DocumentNumber,
			CustomerOrderNumber: line.CustomerOrderNumber,
			BatchNumber:         req.BatchNumber,
			IloID:               line.ID,
			RequestID:           line.RequestID,
			CustomerID:          req.CustomerID,
			CustomerCode:        req.Customer.Code,
			CustomerName:        req.Customer.Name,
			ProductID:           productID,
			ProductCode:         productCode,
			ProductName:         productName,
			RequestedQty:        line.RequestedQty,
			ByProductQuantity:   line.ByProductQuantity,
			StockStatusID:       req.StockStatusID,
			StockStatusCode:     req.StockStatus.Code,
			StockStatusName:     req.StockStatus.Name,
			TransporterID:       line.TransporterID,
			TransporterName:     line.TransporterName,
			DriverID:            line.DriverID,
			DriverName:          line.DriverName,
			TruckID:             line.TruckID,
			PlateNumber:         firstNonEmpty(line.TruckPlate, line.HorsePlate),
			HorsePlate:          firstNonEmpty(line.HorsePlate, line.TruckPlate),
			TrailerOnePlate:     line.TrailerOnePlate,
			TrailerTwoPlate:     line.TrailerTwoPlate,
			OrderDate:           req.OrderDate,
			ExpirationDate:      line.ExpirationDate,
			EwuraLicense:        line.EwuraLicense,
			Destination:         line.Destination,
			District:            line.District,
			IsActive:            true,
			Status:              types.OrderRunning,
			CreatedByID:         userID,
		}
		if line.ByProductID != nil {
			row.ByProductID = line.ByProductID
			if line.ByProduct != nil {
				row.ByProductCode = line.ByProduct.Code
				row.ByProductName = line.ByProduct.Name
			}
		}
		if line.Driver != nil {
			row.DriverLicense = line.Driver.LicenseNumber
		}
		if badge != nil {
			row.BadgeID = &badge.ID
			row.BadgeCode = badge.Code
			if err := tx.Model(badge).Updates(map[string]any{"IsAvailable": false}).Error; err != nil {
				return err
			}
		}
		pid := productID
		for i := range cells {
			cells[i].ProductID = &pid
			cells[i].ProductCode = productCode
			cells[i].ProductName = productName
			cells[i].Balance = cells[i].Capacity
			if strings.EqualFold(cells[i].TankPlate, row.TrailerOnePlate) {
				row.TrailerOneID = &cells[i].TankID
			}
			if row.TrailerTwoPlate != "" && strings.EqualFold(cells[i].TankPlate, row.TrailerTwoPlate) {
				row.TrailerTwoID = &cells[i].TankID
			}
		}
		row.Lines = cells
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&line).Update("Status", types.OrderRunning).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) calibrationCells(line models.GantryLoadingLine) ([]models.CompartmentalizationLine, error) {
	plates := uniquePlates(line.HorsePlate, line.TrailerOnePlate, line.TrailerTwoPlate, line.TruckPlate)
	var tanks []models.TruckTank
	if err := s.db.Where("PlateNumber IN ? AND IsActive = 1", plates).Find(&tanks).Error; err != nil {
		return nil, err
	}
	if len(tanks) == 0 {
		return nil, fmt.Errorf("no active truck tanks match plates %s — capture the calibration first", strings.Join(plates, ", "))
	}
	tankIDs := make([]uint, len(tanks))
	for i, tank := range tanks {
		tankIDs[i] = tank.ID
	}
	var cals []models.TankCalibration
	if err := s.db.Where("TankID IN ? AND IsActive = 1", tankIDs).
		Order("ValidFrom DESC").Find(&cals).Error; err != nil {
		return nil, err
	}
	latest := make(map[uint]models.TankCalibration, len(tanks))
	for _, cal := range cals {
		if _, ok := latest[cal.TankID]; ok {
			continue
		}
		latest[cal.TankID] = cal
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	calIDs := make([]uint, 0, len(tanks))
	for _, tank := range tanks {
		cal, ok := latest[tank.ID]
		if !ok || cal.ID == 0 || cal.Expired(today) {
			return nil, fmt.Errorf("no active calibration for tank %s", tank.PlateNumber)
		}
		calIDs = append(calIDs, cal.ID)
	}
	var compartments []models.TankCompartment
	if err := s.db.Where("CalibrationID IN ?", calIDs).Order("CalibrationID, [Index]").Find(&compartments).Error; err != nil {
		return nil, err
	}
	byCal := make(map[uint][]models.TankCompartment, len(calIDs))
	for _, c := range compartments {
		byCal[c.CalibrationID] = append(byCal[c.CalibrationID], c)
	}
	var cells []models.CompartmentalizationLine
	for _, tank := range tanks {
		cal := latest[tank.ID]
		for _, c := range byCal[cal.ID] {
			cells = append(cells, models.CompartmentalizationLine{
				CalibrationID: cal.ID,
				TankID:        tank.ID,
				TankPlate:     tank.PlateNumber,
				Index:         c.Index,
				Capacity:      c.Capacity,
			})
		}
	}
	if len(cells) == 0 {
		return nil, fmt.Errorf("calibration has no compartments")
	}
	return cells, nil
}

func (s *Service) SaveCompLines(compUID string, lines []CompLinePatch) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var comp models.Compartmentalization
		if err := tx.Where("UID = ?", compUID).First(&comp).Error; err != nil {
			return err
		}
		if comp.Status != types.OrderRunning && comp.Status != types.OrderDraft && comp.Status != types.OrderRejected {
			return fmt.Errorf("compartmentalization %s is not editable", comp.DocumentNumber)
		}
		for _, p := range lines {
			var cell models.CompartmentalizationLine
			if err := tx.Where("UID = ? AND CompartmentalizationID = ?", p.UID, comp.ID).First(&cell).Error; err != nil {
				return err
			}
			p.TopSeal = strings.ToUpper(strings.TrimSpace(p.TopSeal))
			p.DipSeal = strings.ToUpper(strings.TrimSpace(p.DipSeal))
			p.BottomSeal = strings.ToUpper(strings.TrimSpace(p.BottomSeal))
			qty := p.Quantity
			if qty.GreaterThan(cell.Capacity) {
				return fmt.Errorf("quantity %s exceeds capacity %s on compartment %d", qty, cell.Capacity, cell.Index)
			}
			seals := []string{p.TopSeal, p.DipSeal, p.BottomSeal}
			for i, a := range seals {
				a = strings.TrimSpace(a)
				if a == "" {
					continue
				}
				for j, b := range seals {
					if i != j && strings.EqualFold(a, strings.TrimSpace(b)) {
						return fmt.Errorf("top, dip and bottom seals on a compartment must all be different")
					}
				}
				var n int64
				q := tx.Model(&models.CompartmentalizationLine{}).Where("(TopSeal = ? OR DipSeal = ? OR BottomSeal = ?) AND UID <> ?", a, a, a, cell.UID)
				if err := q.Count(&n).Error; err != nil {
					return err
				}
				if n > 0 {
					return fmt.Errorf("seal %s is already used on another compartment", a)
				}
			}
			upd := map[string]any{
				"Quantity": qty, "Balance": cell.Capacity.Sub(qty),
				"TopSeal": p.TopSeal, "DipSeal": p.DipSeal, "BottomSeal": p.BottomSeal,
			}
			if p.ProductID != 0 {
				var prod models.Product
				if err := tx.First(&prod, p.ProductID).Error; err != nil {
					return fmt.Errorf("product not found")
				}
				upd["ProductID"] = p.ProductID
				upd["ProductCode"] = prod.Code
				upd["ProductName"] = prod.Name
			}
			if err := tx.Model(&cell).Updates(upd).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type CompLinePatch struct {
	UID        string
	ProductID  uint
	Quantity   decimal.Decimal
	TopSeal    string
	DipSeal    string
	BottomSeal string
}

func (s *Service) SubmitCompartmentalization(uid string, user *models.User) error {
	var comp models.Compartmentalization
	if err := s.db.Preload("Lines").Preload("Ilo.Request").Where("UID = ?", uid).First(&comp).Error; err != nil {
		return err
	}
	need := alma.Litres(comp.Ilo.RequestedQty)
	var got int
	for _, ln := range comp.Lines {
		got += int(ln.Quantity.IntPart())
	}
	if got != need {
		return fmt.Errorf("compartment litres %d must equal order litres %d", got, need)
	}
	if err := s.db.Model(&comp).Update("Status", types.OrderSubmitted).Error; err != nil {
		return err
	}
	return s.Initiate(types.CompartmentalizationContent, comp.ID, user, comp.DocumentNumber, comp.DocumentNumber)
}

// OnCompApproved writes the SAP3C file and marks the GLO in progress.
func (s *Service) OnCompApproved(tx *gorm.DB, comp *models.Compartmentalization) error {
	if err := tx.Preload("Lines.Product").Preload("Ilo.Request.Customer").
		Preload("Ilo.Request.Product").First(comp, comp.ID).Error; err != nil {
		return err
	}
	if err := tx.Model(comp).Update("Status", types.OrderInProgress).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.GantryLoadingLine{}).Where("ID = ?", comp.IloID).
		Updates(map[string]any{"Status": types.OrderInProgress, "SentToAlma": true}).Error; err != nil {
		return err
	}
	order, err := s.almaOrder(tx, comp, false)
	if err != nil {
		return err
	}
	fileName := alma.NewFileName(time.Now())
	if s.almaRoot != "" {
		if _, err := alma.WriteOrder(tx, alma.Paths{Root: s.almaRoot}, order, fileName); err != nil {
			return err
		}
	} else {
		logs.Warn("ALMA file path is empty — compartmentalization approved without writing SAP3C")
	}
	now := time.Now()
	return tx.Model(comp).Updates(map[string]any{
		"AlmaFileName": fileName,
		"AlmaSentAt":   now,
	}).Error
}

func (s *Service) almaOrder(tx *gorm.DB, comp *models.Compartmentalization, canceled bool) (alma.Order, error) {
	line := comp.Ilo
	req := line.Request
	if req == nil {
		var r models.GantryLoadingRequest
		if err := tx.Preload("Customer").Preload("Product").First(&r, line.RequestID).Error; err != nil {
			return alma.Order{}, err
		}
		req = &r
	}
	exp := time.Now().Add(7 * 24 * time.Hour)
	if line.ExpirationDate != nil {
		exp = *line.ExpirationDate
	}
	batch := req.BatchNumber
	if batch == "" {
		batch = alma.BatchCode(req.ID)
	}
	o := alma.Order{
		BatchNumber:     batch,
		BatchDate:       req.OrderDate,
		CustomerCode:    req.Customer.Code,
		ProductNumber:   alma.AlmaProductNumber(req.Product.Code),
		QuantityLtr:     alma.Litres(line.RequestedQty),
		OrderDate:       req.OrderDate,
		ExpirationDate:  exp,
		DocNumber:       line.DocumentNumber,
		TransporterName: line.TransporterName,
		DriverName:      line.DriverName,
		Destination:     line.Destination,
		District:        line.District,
		HorsePlate:      firstNonEmpty(comp.HorsePlate, line.HorsePlate, line.TruckPlate),
		TrailerOnePlate: firstNonEmpty(comp.TrailerOnePlate, line.TrailerOnePlate, line.TruckPlate),
		TrailerTwoPlate: firstNonEmpty(comp.TrailerTwoPlate, line.TrailerTwoPlate),
		Canceled:        canceled,
	}
	for _, ln := range comp.Lines {
		if ln.ProductID == nil || ln.Quantity.IsZero() {
			continue
		}
		code := ln.ProductCode
		if ln.Product != nil && ln.Product.Code != "" {
			code = ln.Product.Code
		}
		cell := alma.Compartment{TankPlate: ln.TankPlate, Index: ln.Index, Quantity: int(ln.Quantity.IntPart())}
		if alma.AlmaProductNumber(code) == o.ProductNumber {
			o.Compartments = append(o.Compartments, cell)
		} else {
			o.ByProductNumber = alma.AlmaProductNumber(code)
			o.ByProductLtr += cell.Quantity
			o.ByCompartments = append(o.ByCompartments, cell)
		}
	}
	return o, nil
}

// CompleteFromAlma posts the SAP3R result onto the matching in-progress GLO.
func (s *Service) CompleteFromAlma(res alma.Result, fileName string) error {
	var line models.GantryLoadingLine
	err := s.db.Preload("Request.Customer").Preload("Request.Product").Preload("Request.StockStatus").
		Where("DocumentNumber = ? AND Status = ? AND IsActive = 1", res.OrderNumber, types.OrderInProgress).
		First(&line).Error
	if err != nil {
		return fmt.Errorf("GLO %s is not waiting for ALMA: %w", res.OrderNumber, err)
	}
	if line.Request != nil && line.Request.Customer.Code != "" &&
		!strings.EqualFold(strings.TrimSpace(line.Request.Customer.Code), strings.TrimSpace(res.CustomerCode)) {
		return fmt.Errorf("customer code %s does not match %s", res.CustomerCode, line.Request.Customer.Code)
	}
	std := res.Products[0].LoadedVolumeAt20
	qty := alma.CubicMetres(std)
	if err := s.CompleteLoadingLine(line.UID, qty); err != nil {
		return err
	}
	var comp models.Compartmentalization
	if err := s.db.Where("IloID = ? AND IsActive = 1", line.ID).Order("ID DESC").First(&comp).Error; err == nil {
		now := res.LoadingDate
		_ = s.db.Model(&comp).Updates(map[string]any{
			"Status": types.OrderLoaded, "LoadedAt": now, "AlmaFileName": fileName, "LoadedQty": qty,
		}).Error
		req := line.Request
		load := models.GantryLoading{
			CompartmentalizationID: comp.ID,
			IloID:                  line.ID,
			RequestID:              line.RequestID,
			DocumentNumber:         line.DocumentNumber,
			CustomerOrderNumber:    line.CustomerOrderNumber,
			BatchNumber:            req.BatchNumber,
			OrderDate:              req.OrderDate,
			LoadedAt:               res.LoadingDate,
			Year:                   res.LoadingDate.Year(),
			Month:                  int(res.LoadingDate.Month()),
			RequestedQty:           line.RequestedQty,
			BadgeID:                comp.BadgeID,
			BadgeCode:              comp.BadgeCode,
			CustomerID:             req.CustomerID,
			CustomerCode:           req.Customer.Code,
			CustomerName:           req.Customer.Name,
			StockStatusID:          req.StockStatusID,
			StockStatusCode:        req.StockStatus.Code,
			StockStatusName:        req.StockStatus.Name,
			TransporterID:          line.TransporterID,
			TransporterName:        line.TransporterName,
			DriverID:               line.DriverID,
			DriverName:             line.DriverName,
			DriverLicense:          comp.DriverLicense,
			TruckID:                line.TruckID,
			PlateNumber:            firstNonEmpty(line.TruckPlate, comp.PlateNumber),
			EwuraLicense:           line.EwuraLicense,
			Destination:            line.Destination,
			District:               line.District,
			ExpirationDate:         line.ExpirationDate,
			AlmaFileName:           fileName,
		}
		for _, pl := range res.Products {
			pid := productIDByCode(s.db, pl.ProductCode)
			if pid == 0 {
				pid = req.ProductID
			}
			var prod models.Product
			_ = s.db.First(&prod, pid).Error
			dens := decimal.NewFromFloat(pl.Density)
			stdM3 := alma.CubicMetres(pl.LoadedVolumeAt20)
			wcf := models.LoadingWCF(dens)
			load.Products = append(load.Products, models.GantryLoadingProduct{
				ProductID:      pid,
				ProductCode:    firstNonEmpty(prod.Code, pl.ProductCode),
				ProductName:    prod.Name,
				ObservedVolume: alma.CubicMetres(pl.LoadedVolume),
				StandardVolume: stdM3,
				Temperature:    decimal.NewFromFloat(pl.Temperature),
				Density:        dens,
				WCF:            wcf,
				Weight:         models.LoadingWeight(wcf, stdM3),
			})
		}
		if err := s.db.Create(&load).Error; err != nil {
			logs.Error(err)
		} else {
			transit := req.StockStatus.IsTransit
			_ = models.ApplyLoadingToSummary(s.db, &load, transit)
		}
		ewura.EnqueueLoading(s.db, &line, &comp, qty)
	}
	return nil
}

func productIDByCode(db *gorm.DB, code string) uint {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0
	}
	var p models.Product
	if db.Where("Code = ?", code).First(&p).Error == nil {
		return p.ID
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func uniquePlates(vals ...string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
