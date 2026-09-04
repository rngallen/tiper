package inventory

import (
	"fmt"

	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/pkg/precision"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AllocateLineLoss writes a signed header loss onto parcels.
// Reception + line loss = tank figure. Proration (PRORATA) parcels are
// excluded from the pool and keep zero line-loss.
func AllocateLineLoss(details []models.ReceiptDetail, isProration []bool, lossL, lossCM, lossMT decimal.Decimal) {
	AllocateLineLossRounded(details, isProration, lossL, lossCM, lossMT, precision.Defaults)
}

func AllocateLineLossRounded(details []models.ReceiptDetail, isProration []bool, lossL, lossCM, lossMT decimal.Decimal, prec precision.Settings) {
	pool := lineLossPool(details, isProration)
	if len(pool) == 0 {
		return
	}
	prec = prec.Normalize()
	distributeLoss(details, pool, lossL, prec.Quantity, func(d *models.ReceiptDetail) decimal.Decimal { return d.Quantity },
		func(d *models.ReceiptDetail, v decimal.Decimal) { d.LineLoss = v })
	distributeLoss(details, pool, lossCM, prec.CubicMeter, func(d *models.ReceiptDetail) decimal.Decimal { return d.CubicMeter },
		func(d *models.ReceiptDetail, v decimal.Decimal) { d.LineLossCubicMeter = v })
	distributeLoss(details, pool, lossMT, prec.MetricTonne, func(d *models.ReceiptDetail) decimal.Decimal { return d.MetricTonne },
		func(d *models.ReceiptDetail, v decimal.Decimal) { d.LineLossMetricTonne = v })
}

func lineLossPool(details []models.ReceiptDetail, isProration []bool) []int {
	var pool []int
	for i, d := range details {
		if d.IsArchived {
			continue
		}
		if i < len(isProration) && isProration[i] {
			d.LineLoss = decimal.Zero
			d.LineLossCubicMeter = decimal.Zero
			d.LineLossMetricTonne = decimal.Zero
			details[i] = d
			continue
		}
		pool = append(pool, i)
	}
	return pool
}

func lineLossWeight(d models.ReceiptDetail, unit func(*models.ReceiptDetail) decimal.Decimal) decimal.Decimal {
	w := unit(&d)
	if !w.IsZero() {
		return w.Abs()
	}
	if !d.CubicMeter.IsZero() {
		return d.CubicMeter.Abs()
	}
	if !d.Quantity.IsZero() {
		return d.Quantity.Abs()
	}
	return d.MetricTonne.Abs()
}

func distributeLoss(
	details []models.ReceiptDetail,
	pool []int,
	total decimal.Decimal,
	places int,
	unit func(*models.ReceiptDetail) decimal.Decimal,
	set func(*models.ReceiptDetail, decimal.Decimal),
) {
	for _, i := range pool {
		set(&details[i], decimal.Zero)
	}
	if total.IsZero() || len(pool) == 0 {
		return
	}
	weights := make([]decimal.Decimal, len(pool))
	sumW := decimal.Zero
	for n, i := range pool {
		w := lineLossWeight(details[i], unit)
		if w.IsZero() {
			w = decimal.NewFromInt(1)
		}
		weights[n] = w
		sumW = sumW.Add(w)
	}
	assigned := decimal.Zero
	for n, i := range pool {
		if n == len(pool)-1 {
			set(&details[i], total.Sub(assigned))
			return
		}
		share := precision.Round(total.Mul(weights[n]).Div(sumW), places)
		set(&details[i], share)
		assigned = assigned.Add(share)
	}
}

func ApplyTankLineLoss(r *models.Receipt, isProration []bool) {
	if r == nil {
		return
	}
	var outL, outCM, outMT decimal.Decimal
	for i, d := range r.Details {
		if d.IsArchived {
			continue
		}
		if i < len(isProration) && isProration[i] {
			continue
		}
		outL = outL.Add(d.Quantity)
		outCM = outCM.Add(d.CubicMeter)
		outMT = outMT.Add(d.MetricTonne)
	}
	if !r.TankQuantity.IsZero() {
		r.LineLoss = r.TankQuantity.Sub(outL)
	}
	if !r.TankCubicMeter.IsZero() {
		r.LineLossCubicMeter = r.TankCubicMeter.Sub(outCM)
	}
	if !r.TankMetricTonne.IsZero() {
		r.LineLossMetricTonne = r.TankMetricTonne.Sub(outMT)
	}
}

func ProrationFlags(tx *gorm.DB, details []models.ReceiptDetail) ([]bool, error) {
	flags := make([]bool, len(details))
	seen := map[uint]struct{}{}
	var ids []uint
	for _, d := range details {
		sid := d.StatusID()
		if sid == 0 {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		seen[sid] = struct{}{}
		ids = append(ids, sid)
	}
	if len(ids) == 0 {
		return flags, nil
	}
	var rows []models.StockStatus
	if err := tx.Where("ID IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	pror := map[uint]bool{}
	for _, st := range rows {
		pror[st.ID] = st.IsProration
	}
	for i, d := range details {
		flags[i] = pror[d.StatusID()]
	}
	return flags, nil
}

// AllocateReceiptLineLoss applies header tank/outturn difference onto draft
// internal parcels. External receipts have no TIPER tank figure.
func (s *Service) AllocateReceiptLineLoss(uid string, tankL, tankCM, tankMT, lossL, lossCM, lossMT decimal.Decimal) (*models.Receipt, error) {
	var row models.Receipt
	if err := s.db.Preload("Details").Where("UID = ?", uid).First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status != types.ReceiptDraft {
		return nil, fmt.Errorf("line loss can only be allocated on a draft receipt")
	}
	if row.ReceiptType != types.ReceiptInternal {
		return nil, fmt.Errorf("line loss applies to internal TIPER receipts")
	}
	if !tankL.IsZero() {
		row.TankQuantity = tankL
	}
	if !tankCM.IsZero() {
		row.TankCubicMeter = tankCM
	}
	if !tankMT.IsZero() {
		row.TankMetricTonne = tankMT
	}
	if !lossL.IsZero() && tankL.IsZero() {
		row.LineLoss = lossL
	}
	if !lossCM.IsZero() && tankCM.IsZero() {
		row.LineLossCubicMeter = lossCM
	}
	if !lossMT.IsZero() && tankMT.IsZero() {
		row.LineLossMetricTonne = lossMT
	}
	prec := integrations.LivePrecision()
	flags, err := ProrationFlags(s.db, row.Details)
	if err != nil {
		return nil, err
	}
	ApplyTankLineLoss(&row, flags)
	AllocateLineLossRounded(row.Details, flags, row.LineLoss, row.LineLossCubicMeter, row.LineLossMetricTonne, prec)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Updates(map[string]any{
			"TankQuantity":        row.TankQuantity,
			"TankCubicMeter":      row.TankCubicMeter,
			"TankMetricTonne":     row.TankMetricTonne,
			"LineLoss":            row.LineLoss,
			"LineLossCubicMeter":  row.LineLossCubicMeter,
			"LineLossMetricTonne": row.LineLossMetricTonne,
		}).Error; err != nil {
			return err
		}
		for i := range row.Details {
			d := row.Details[i]
			if err := tx.Model(&d).Updates(map[string]any{
				"LineLoss":            d.LineLoss,
				"LineLossCubicMeter":  d.LineLossCubicMeter,
				"LineLossMetricTonne": d.LineLossMetricTonne,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.Preload("Details.Customer").Preload("Details.StockStatus").Preload("Details.Depot").
		Preload("Vessel").Preload("Product").Preload("Supplier").
		Where("UID = ?", uid).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
