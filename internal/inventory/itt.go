package inventory

import (
	"errors"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var thousand = decimal.NewFromInt(1000)

func ittCubic(itt *models.IttTransfer) decimal.Decimal {
	if itt == nil {
		return decimal.Zero
	}
	if !itt.CubicMeter.IsZero() {
		return itt.CubicMeter
	}
	return itt.Quantity
}

func receiptDensity(tx *gorm.DB, itt *models.IttTransfer) decimal.Decimal {
	var rec models.Receipt
	err := tx.Where("ProductID = ? AND VesselID = ? AND CONVERT(date, VesselDate) = CONVERT(date, ?)",
		itt.ProductID, itt.VesselID, itt.VesselDate).
		Order("ID DESC").Limit(1).Find(&rec).Error
	if err != nil || rec.ID == 0 {
		return decimal.Zero
	}
	return rec.Density
}

func ittStockQty(itt *models.IttTransfer, density decimal.Decimal) (litres, cm, mt decimal.Decimal) {
	cm = ittCubic(itt)
	litres = cm.Mul(thousand)
	mt = itt.MetricTonne
	if mt.IsZero() && !density.IsZero() {
		mt = cm.Mul(density)
	}
	return litres, cm, mt
}

func parcelAvailable(d models.ReceiptDetail) decimal.Decimal {
	if !d.CubicMeter.IsZero() {
		return d.CubicMeter
	}
	return d.Quantity
}

func originTaken(tx *gorm.DB, originID uint) decimal.Decimal {
	var cm decimal.Decimal
	_ = tx.Model(&models.ReceiptDetail{}).
		Where("OriginDetailID = ? AND IsArchived = 0", originID).
		Select("ISNULL(SUM(CubicMeter), 0)").Scan(&cm).Error
	return cm
}

// scaleMoved copies src with quantities scaled to `take` of `total` (same unit).
func scaleMoved(src models.ReceiptDetail, take, total decimal.Decimal) models.ReceiptDetail {
	out := src
	out.ID = 0
	out.UID = ""
	if total.IsZero() || take.IsZero() {
		out.Quantity = decimal.Zero
		out.CubicMeter = decimal.Zero
		out.MetricTonne = decimal.Zero
		out.LineLoss = decimal.Zero
		out.LineLossCubicMeter = decimal.Zero
		out.LineLossMetricTonne = decimal.Zero
		out.HoldQuantity = decimal.Zero
		return out
	}
	out.Quantity = share(src.Quantity, take, total)
	out.CubicMeter = share(src.CubicMeter, take, total)
	out.MetricTonne = share(src.MetricTonne, take, total)
	out.LineLoss = share(src.LineLoss, take, total)
	out.LineLossCubicMeter = share(src.LineLossCubicMeter, take, total)
	out.LineLossMetricTonne = share(src.LineLossMetricTonne, take, total)
	out.HoldQuantity = share(src.HoldQuantity, take, total)
	return out
}

func share(v, take, total decimal.Decimal) decimal.Decimal {
	return v.Mul(take).Div(total).Round(3)
}

func reduceParcel(d *models.ReceiptDetail, take, total decimal.Decimal) {
	if d == nil {
		return
	}
	moved := scaleMoved(*d, take, total)
	d.Quantity = d.Quantity.Sub(moved.Quantity)
	d.CubicMeter = d.CubicMeter.Sub(moved.CubicMeter)
	d.MetricTonne = d.MetricTonne.Sub(moved.MetricTonne)
	d.LineLoss = d.LineLoss.Sub(moved.LineLoss)
	d.LineLossCubicMeter = d.LineLossCubicMeter.Sub(moved.LineLossCubicMeter)
	d.LineLossMetricTonne = d.LineLossMetricTonne.Sub(moved.LineLossMetricTonne)
	d.HoldQuantity = d.HoldQuantity.Sub(moved.HoldQuantity)
	clampParcel(d)
}

func clampParcel(d *models.ReceiptDetail) {
	if d.Quantity.IsNegative() {
		d.Quantity = decimal.Zero
	}
	if d.CubicMeter.IsNegative() {
		d.CubicMeter = decimal.Zero
	}
	if d.MetricTonne.IsNegative() {
		d.MetricTonne = decimal.Zero
	}
}

// attachIttBillingParcels adds receiver billing children without changing the
// sender's original receipt figures. Stock history is the two dated movements
// posted at approval (sender out, receiver in).
func attachIttBillingParcels(tx *gorm.DB, itt *models.IttTransfer, asOf time.Time) error {
	if tx == nil || itt == nil || !ittCubic(itt).IsPositive() {
		return nil
	}
	need := ittCubic(itt)
	var parcels []models.ReceiptDetail
	q := tx.Joins("JOIN Receipt r ON r.ID = ReceiptDetail.ReceiptID").
		Where("ReceiptDetail.CustomerID = ? AND r.ProductID = ? AND r.VesselID = ?",
			itt.FromCustomerID, itt.ProductID, itt.VesselID).
		Where("CONVERT(date, r.VesselDate) = CONVERT(date, ?)", itt.VesselDate).
		Where("ReceiptDetail.StockStatusID = ? AND ReceiptDetail.FinancialHold = ?", itt.StockStatusID, itt.FinancialHold).
		Where("ReceiptDetail.OriginDetailID IS NULL").
		Where("ReceiptDetail.IsArchived = 0 AND ReceiptDetail.IsProvision = 0").
		Where("r.IsProvision = 0 AND r.Status = ?", types.ReceiptApproved).
		Order("r.Date, ReceiptDetail.ID")
	if err := q.Find(&parcels).Error; err != nil {
		return err
	}
	for i := range parcels {
		if !need.IsPositive() {
			break
		}
		src := parcels[i]
		avail := parcelAvailable(src).Sub(originTaken(tx, src.ID))
		if !avail.IsPositive() {
			continue
		}
		take := avail
		if need.LessThan(avail) {
			take = need
		}
		moved := scaleMoved(src, take, parcelAvailable(src))
		if err := addReceiverParcel(tx, src.ID, itt.ToCustomerID, moved, asOf); err != nil {
			return err
		}
		need = need.Sub(take)
	}
	return nil
}

func addReceiverParcel(tx *gorm.DB, originID, toCustomer uint, moved models.ReceiptDetail, asOf time.Time) error {
	origin := originID
	moved.CustomerID = toCustomer
	moved.OriginDetailID = &origin
	moved.EffectiveFrom = &asOf
	var existing models.ReceiptDetail
	err := tx.Where("OriginDetailID = ? AND CustomerID = ? AND IsArchived = 0", originID, toCustomer).
		First(&existing).Error
	if err == nil {
		existing.Quantity = existing.Quantity.Add(moved.Quantity)
		existing.CubicMeter = existing.CubicMeter.Add(moved.CubicMeter)
		existing.MetricTonne = existing.MetricTonne.Add(moved.MetricTonne)
		existing.LineLoss = existing.LineLoss.Add(moved.LineLoss)
		existing.LineLossCubicMeter = existing.LineLossCubicMeter.Add(moved.LineLossCubicMeter)
		existing.LineLossMetricTonne = existing.LineLossMetricTonne.Add(moved.LineLossMetricTonne)
		existing.HoldQuantity = existing.HoldQuantity.Add(moved.HoldQuantity)
		existing.EffectiveFrom = &asOf
		return tx.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Omit("Customer", "StockStatus", "Depot", "Receipt").Create(&moved).Error
}
