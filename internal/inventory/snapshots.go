package inventory

import (
	"errors"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// eastAfrica is the TIPER depot calendar (Africa/Dar_es_Salaam, UTC+3).
var eastAfrica = time.FixedZone("EAT", 3*3600)

func dayUTC(t time.Time) time.Time {
	y, m, d := t.In(eastAfrica).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func createMovement(tx *gorm.DB, mv *models.StockMovement) error {
	if tx == nil {
		return gorm.ErrInvalidDB
	}
	if err := tx.Create(mv).Error; err != nil {
		return err
	}
	return ApplyMovement(tx, mv)
}

// ApplyMovement updates live and daily snapshot tables in the same transaction
// as the ledger row so reports stay current without scanning StockMovement.
func ApplyMovement(tx *gorm.DB, m *models.StockMovement) error {
	if tx == nil || m == nil {
		return nil
	}
	if err := applyBalance(tx, m); err != nil {
		return err
	}
	if err := applyDaily(tx, m); err != nil {
		return err
	}
	return applyProductDayBook(tx, m)
}

func applyBalance(tx *gorm.DB, m *models.StockMovement) error {
	vd := dayUTC(m.VesselDate)
	var row models.StockBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("CustomerID = ? AND ProductID = ? AND VesselID = ? AND VesselDate = ? AND StockStatusID = ? AND FinancialHold = ? AND IsProvision = ?",
			m.CustomerID, m.ProductID, m.VesselID, vd, m.StockStatusID, m.FinancialHold, m.IsProvision).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.StockBalance{
			CustomerID: m.CustomerID, ProductID: m.ProductID, VesselID: m.VesselID,
			VesselDate: vd, StockStatusID: m.StockStatusID, FinancialHold: m.FinancialHold,
			IsProvision: m.IsProvision, DepotID: m.DepotID,
			Quantity: m.Quantity, CubicMeter: m.CubicMeter, MetricTonne: m.MetricTonne,
		}
		return tx.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&row).Updates(map[string]any{
		"Quantity":    row.Quantity.Add(m.Quantity),
		"CubicMeter":  row.CubicMeter.Add(m.CubicMeter),
		"MetricTonne": row.MetricTonne.Add(m.MetricTonne),
		"DepotID":     m.DepotID,
	}).Error
}

func applyDaily(tx *gorm.DB, m *models.StockMovement) error {
	if m.IsProvision {
		return nil
	}
	day := dayUTC(m.TransactionDate)
	received, outflow, hold, srt := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	loading, pump, itt, adj := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	if m.TransactionType == types.TxnReceipt && m.Quantity.IsPositive() {
		received = m.Quantity
		if isSRTReceipt(tx, m) {
			srt = m.Quantity
		}
	}
	if m.Quantity.IsNegative() && m.TransactionType != types.TxnHold && m.TransactionType != types.TxnHoldRelease {
		outflow = m.Quantity.Abs()
	}
	switch m.TransactionType {
	case types.TxnLoading:
		loading = m.Quantity.Abs()
	case types.TxnPumpOver:
		pump = m.Quantity.Abs()
	case types.TxnITT:
		itt = m.Quantity.Abs()
	case types.TxnAdjustment, types.TxnReversal, types.TxnZerolization:
		adj = m.Quantity
	}
	if m.FinancialHold {
		hold = m.Quantity
	}

	var row models.StockDailyPosition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("PositionDate = ? AND CustomerID = ? AND ProductID = ? AND StockStatusID = ?",
			day, m.CustomerID, m.ProductID, m.StockStatusID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.StockDailyPosition{
			PositionDate: day, CustomerID: m.CustomerID, ProductID: m.ProductID, StockStatusID: m.StockStatusID,
			ClosingQty: m.Quantity, ReceivedQty: received, OutflowQty: outflow,
			LoadingQty: loading, PumpOverQty: pump, ITTQty: itt, AdjustmentQty: adj,
			HoldQty: hold, SRTReceivedQty: srt,
		}
		return tx.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&row).Updates(map[string]any{
		"ClosingQty":     row.ClosingQty.Add(m.Quantity),
		"ReceivedQty":    row.ReceivedQty.Add(received),
		"OutflowQty":     row.OutflowQty.Add(outflow),
		"LoadingQty":     row.LoadingQty.Add(loading),
		"PumpOverQty":    row.PumpOverQty.Add(pump),
		"ITTQty":         row.ITTQty.Add(itt),
		"AdjustmentQty":  row.AdjustmentQty.Add(adj),
		"HoldQty":        row.HoldQty.Add(hold),
		"SRTReceivedQty": row.SRTReceivedQty.Add(srt),
	}).Error
}

func isSRTReceipt(tx *gorm.DB, m *models.StockMovement) bool {
	if m.ReferenceType != "ReceiptDetail" || m.ReferenceID == 0 {
		return false
	}
	var flag int
	_ = tx.Raw(`
		SELECT CASE WHEN t.IsSingleReceiving = 1 THEN 1 ELSE 0 END
		FROM Receipt r
		JOIN ReceiptDetail d ON d.ReceiptID = r.ID
		JOIN ImportTenderType t ON t.Code = r.TenderCode
		WHERE d.ID = ?`, m.ReferenceID).Scan(&flag).Error
	return flag == 1
}

func applyProductDayBook(tx *gorm.DB, m *models.StockMovement) error {
	if m.IsProvision {
		return nil
	}
	day := dayUTC(m.TransactionDate)
	var row models.ProductDailyBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("BalanceDate = ? AND ProductID = ?", day, m.ProductID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.ProductDailyBalance{BalanceDate: day, ProductID: m.ProductID, BookQty: m.Quantity}
		row.GainLoss = row.PhysicalQty.Add(row.LineQty).Sub(row.BookQty)
		return tx.Create(&row).Error
	}
	if err != nil {
		return err
	}
	book := row.BookQty.Add(m.Quantity)
	return tx.Model(&row).Updates(map[string]any{
		"BookQty":  book,
		"GainLoss": row.PhysicalQty.Add(row.LineQty).Sub(book),
	}).Error
}

// ApplyDip writes physical tank volume onto ProductDailyBalance for that day.
func ApplyDip(tx *gorm.DB, d *models.PhysicalDip) error {
	if tx == nil || d == nil {
		return nil
	}
	var tank models.Tank
	if err := tx.First(&tank, d.TankID).Error; err != nil {
		return err
	}
	if tank.ProductID == 0 {
		return nil
	}
	day := dayUTC(d.DipDate)
	var physical decimal.Decimal
	if err := tx.Raw(`
		SELECT ISNULL(SUM(x.At20), 0) FROM PhysicalDip x
		JOIN Tank t ON t.ID = x.TankID
		WHERE t.ProductID = ? AND CAST(x.DipDate AS date) = ?`, tank.ProductID, day).
		Scan(&physical).Error; err != nil {
		return err
	}
	return upsertProductPhysical(tx, day, tank.ProductID, &physical, nil)
}

// ApplyLineContent writes pipeline volume onto ProductDailyBalance for that day.
func ApplyLineContent(tx *gorm.DB, l *models.LineContent) error {
	if tx == nil || l == nil {
		return nil
	}
	day := dayUTC(l.ContentDate)
	line := l.InternalVolume.Add(l.ExternalVolume)
	return upsertProductPhysical(tx, day, l.ProductID, nil, &line)
}

func upsertProductPhysical(tx *gorm.DB, day time.Time, productID uint, physical, line *decimal.Decimal) error {
	var row models.ProductDailyBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("BalanceDate = ? AND ProductID = ?", day, productID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.ProductDailyBalance{BalanceDate: day, ProductID: productID}
		if physical != nil {
			row.PhysicalQty = *physical
		}
		if line != nil {
			row.LineQty = *line
		}
		row.GainLoss = row.PhysicalQty.Add(row.LineQty).Sub(row.BookQty)
		return tx.Create(&row).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{}
	phys, ln := row.PhysicalQty, row.LineQty
	if physical != nil {
		phys = *physical
		updates["PhysicalQty"] = phys
	}
	if line != nil {
		ln = *line
		updates["LineQty"] = ln
	}
	updates["GainLoss"] = phys.Add(ln).Sub(row.BookQty)
	return tx.Model(&row).Updates(updates).Error
}

// RebuildSnapshots rebuilds snapshot tables from the ledger. Safe to run after
// a Django import or if snapshots were skipped.
func RebuildSnapshots(db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&models.StockBalance{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&models.StockDailyPosition{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&models.ProductDailyBalance{}).Error; err != nil {
			return err
		}
		var moves []models.StockMovement
		if err := tx.Order("ID ASC").Find(&moves).Error; err != nil {
			return err
		}
		for i := range moves {
			if err := ApplyMovement(tx, &moves[i]); err != nil {
				return err
			}
		}
		var dips []models.PhysicalDip
		if err := tx.Find(&dips).Error; err != nil {
			return err
		}
		for i := range dips {
			if err := ApplyDip(tx, &dips[i]); err != nil {
				return err
			}
		}
		var lines []models.LineContent
		if err := tx.Find(&lines).Error; err != nil {
			return err
		}
		for i := range lines {
			if err := ApplyLineContent(tx, &lines[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
