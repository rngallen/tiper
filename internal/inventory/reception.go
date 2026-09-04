package inventory

import (
	"errors"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// factDepotLabel is the report depot for a receipt line. Named depots win;
// otherwise internal cargo is TIPER and external cargo is OTHERS.
func factDepotLabel(receiptType types.ReceiptType, depot *models.Depot) (code, name string) {
	if depot != nil && (depot.Code != "" || depot.Name != "") {
		return depot.Code, depot.Name
	}
	if receiptType == types.ReceiptInternal {
		return "", "TIPER"
	}
	return "", "OTHERS"
}

// RecordReceptionFacts upserts one ReceptionFact per approved detail so
// reception reports stay on a narrow indexed table.
func (s *Service) RecordReceptionFacts(tx *gorm.DB, r *models.Receipt) error {
	if tx == nil {
		tx = s.db
	}
	if r == nil || r.Status != types.ReceiptApproved {
		return nil
	}
	if err := loadReceiptFactSources(tx, r); err != nil {
		return err
	}
	for _, d := range r.Details {
		if d.IsArchived || d.OriginDetailID != nil {
			continue
		}
		depot := d.Depot
		if depot == nil && d.DepotID != nil {
			var row models.Depot
			if err := tx.First(&row, *d.DepotID).Error; err == nil {
				depot = &row
			}
		}
		code, name := factDepotLabel(r.ReceiptType, depot)
		fact := models.ReceptionFact{
			ReceiptID:           r.ID,
			ReceiptDetailID:     d.ID,
			DocumentNumber:      r.DocumentNumber,
			Date:                dayUTC(r.Date),
			VesselDate:          dayUTC(r.VesselDate),
			RouteCode:           r.RouteCode,
			ReceiptType:         r.ReceiptType,
			VesselID:            r.VesselID,
			VesselCode:          r.Vessel.Code,
			VesselName:          r.Vessel.Name,
			ProductID:           r.ProductID,
			ProductCode:         r.Product.Code,
			ProductName:         r.Product.Name,
			CustomerID:          d.CustomerID,
			CustomerCode:        d.Customer.Code,
			CustomerName:        d.Customer.Name,
			DepotID:             d.DepotID,
			DepotCode:           code,
			DepotName:           name,
			UsesTiperPipeline:   r.UsesTiperPipeline,
			FinancialHold:       d.FinancialHold,
			TenderCode:          r.TenderCode,
			Quantity:            d.ReceivedQuantity(),
			CubicMeter:          d.ReceivedCubicMeter(),
			MetricTonne:         d.ReceivedMetricTonne(),
			LineLoss:            d.LineLoss,
			LineLossCubicMeter:  d.LineLossCubicMeter,
			LineLossMetricTonne: d.LineLossMetricTonne,
		}
		var existing models.ReceptionFact
		err := tx.Where("ReceiptDetailID = ?", d.ID).Take(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&fact).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&existing).Omit(clause.Associations).Updates(map[string]any{
			"DocumentNumber":      fact.DocumentNumber,
			"Date":                fact.Date,
			"VesselDate":          fact.VesselDate,
			"RouteCode":           fact.RouteCode,
			"ReceiptType":         fact.ReceiptType,
			"VesselID":            fact.VesselID,
			"VesselCode":          fact.VesselCode,
			"VesselName":          fact.VesselName,
			"ProductID":           fact.ProductID,
			"ProductCode":         fact.ProductCode,
			"ProductName":         fact.ProductName,
			"CustomerID":          fact.CustomerID,
			"CustomerCode":        fact.CustomerCode,
			"CustomerName":        fact.CustomerName,
			"DepotID":             fact.DepotID,
			"DepotCode":           fact.DepotCode,
			"DepotName":           fact.DepotName,
			"UsesTiperPipeline":   fact.UsesTiperPipeline,
			"FinancialHold":       fact.FinancialHold,
			"TenderCode":          fact.TenderCode,
			"Quantity":            fact.Quantity,
			"CubicMeter":          fact.CubicMeter,
			"MetricTonne":         fact.MetricTonne,
			"LineLoss":            fact.LineLoss,
			"LineLossCubicMeter":  fact.LineLossCubicMeter,
			"LineLossMetricTonne": fact.LineLossMetricTonne,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadReceiptFactSources(tx *gorm.DB, r *models.Receipt) error {
	return tx.Preload("Vessel").Preload("Product").
		Preload("Details.Customer").Preload("Details.Depot").
		First(r, r.ID).Error
}
