package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/catalogs"
	"dfms/internal/jobs"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	engine *wfengine.Engine
}

func NewService(db *gorm.DB, engine *wfengine.Engine, _ any) *Service {
	return &Service{db: db, engine: engine}
}

func (s *Service) RegisterJobs(ctx context.Context, m *jobs.Manager) {
	m.Register(jobs.BillingNth, func() {
		if ctx.Err() != nil {
			return
		}
		if err := s.RunDueNth(time.Now()); err != nil {
			logs.Errorf("nth billing: %v", err)
		}
	})
	m.Register(jobs.BillingTBS, func() {
		if ctx.Err() != nil {
			return
		}
		if err := s.RunDailyTBS(time.Now().AddDate(0, 0, -1)); err != nil {
			logs.Errorf("tbs billing: %v", err)
		}
	})
	m.Register(jobs.BillingVCF, func() {
		if ctx.Err() != nil {
			return
		}
		if err := s.RunMonthlyVCF(time.Now()); err != nil {
			logs.Errorf("vcf billing: %v", err)
		}
	})
}

// ComputeVariableFee implements the A–J spreadsheet:
// C = (A*1000)/B, E = C*D, G = F*C, H = F*E, I = G*B, J = H*B
func ComputeVariableFee(ewuraPerLitre, fx, density, miLossPct decimal.Decimal) (usdCM, usdMT, tzsCM, tzsMT decimal.Decimal) {
	if fx.IsZero() {
		return
	}
	thousand := decimal.NewFromInt(1000)
	wholesaleCM := ewuraPerLitre.Mul(thousand).Div(fx)
	wholesaleMT := wholesaleCM.Mul(density)
	usdCM = miLossPct.Mul(wholesaleCM)
	usdMT = miLossPct.Mul(wholesaleMT)
	tzsCM = usdCM.Mul(fx)
	tzsMT = usdMT.Mul(fx)
	return usdCM.Round(2), usdMT.Round(2), tzsCM.Round(2), tzsMT.Round(2)
}

// FillProductConfig writes wholesale from EWURA × FX × density.
func FillProductConfig(p *models.ProductConfig, fx decimal.Decimal) {
	if p == nil {
		return
	}
	if !fx.IsZero() && !p.Density.IsZero() {
		thousand := decimal.NewFromInt(1000)
		p.WholesaleCM = p.EwuraPrice.Mul(thousand).Div(fx).Round(2)
		p.WholesaleMT = p.WholesaleCM.Mul(p.Density).Round(2)
	}
}

// FillContractRate writes VCF fees for one product × contract MI-loss.
func FillContractRate(p *models.ProductConfig, rate *models.ProductContractRate, fx decimal.Decimal) {
	if p == nil || rate == nil {
		return
	}
	rate.FeeUSDCM, rate.FeeUSDMT, rate.FeeTZSCM, rate.FeeTZSMT = ComputeVariableFee(p.EwuraPrice, fx, p.Density, rate.MiLossValue)
}

// HomePrice converts a source-currency unit price into the batch home currency.
// FX is TZS per 1 USD (the usual TIPER quote).
func HomePrice(source decimal.Decimal, srcCurr, homeCurr string, fx decimal.Decimal) decimal.Decimal {
	srcCurr = strings.ToUpper(strings.TrimSpace(srcCurr))
	homeCurr = strings.ToUpper(strings.TrimSpace(homeCurr))
	if srcCurr == homeCurr || fx.IsZero() {
		return source
	}
	if srcCurr == "USD" && homeCurr == "TZS" {
		return source.Mul(fx).Round(4)
	}
	if srcCurr == "TZS" && homeCurr == "USD" {
		return source.Div(fx).Round(4)
	}
	return source.Mul(fx).Round(4)
}

func priceFor(sourceCurr, runCurr string, source, home decimal.Decimal) decimal.Decimal {
	if strings.EqualFold(strings.TrimSpace(sourceCurr), strings.TrimSpace(runCurr)) {
		return source
	}
	if !home.IsZero() {
		return home
	}
	return source
}

func (s *Service) RunFirstForReceiptTx(tx *gorm.DB, r *models.Receipt) error {
	if tx == nil {
		tx = s.db
	}
	cats, _ := catalogs.Load(tx)
	if !r.ReceiptType.BillsStorage() {
		return s.runKOJ(tx, r)
	}
	actor := systemUserID(tx)
	for _, d := range r.Details {
		if d.IsArchived || d.OriginDetailID != nil {
			continue
		}
		var st models.StockStatus
		if sid := d.StatusID(); sid == 0 {
			continue
		} else if err := tx.First(&st, sid).Error; err == nil && st.IsProration {
			continue
		}
		var existing int64
		if err := tx.Model(&models.BillingRun{}).
			Where("ReceiptDetailID = ? AND BillingSequence = 1 AND FeeCode = ?", d.ID, types.FeeFSF).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		line := lookupFcfLine(tx, r, d, cats)
		days := cycleDays(d)
		if line != nil && line.FirstDays > 0 && d.NextBillingDays == 0 {
			days = line.FirstDays
		}
		run := models.BillingRun{
			ReceiptDetailID: &d.ID,
			CustomerID:      &d.CustomerID,
			SupplierID:      r.SupplierID,
			FeeCode:         types.FeeFSF,
			BillingSequence: 1,
			PeriodStart:     r.BillingEffectiveDate,
			PeriodEnd:       r.BillingEffectiveDate.AddDate(0, 0, days-1),
			ChargeTo:        firstChargeTo(cats, r, d, line),
			CurrencyCode:    "USD",
			Quantity:        qtyForUnit(d, "M3"),
			Status:          "draft",
			Source:          types.BillSrcFirst,
			CreatedByID:     actor,
		}
		n, err := models.AssignDocumentNumber(tx, "fsf-run", "FSF")
		if err != nil {
			return err
		}
		run.DocumentNumber = n
		if line == nil {
			run.ExceptionReason = "no approved FSF pricing model for this receipt"
		} else if err := applyFcfPhase(tx, r, d, line, string(types.PhaseFirst), &run); err != nil {
			run.ExceptionReason = err.Error()
		}
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
	}
	return nil
}

func cycleDays(d models.ReceiptDetail) int {
	if d.NextBillingDays > 0 {
		return d.NextBillingDays
	}
	return 15
}

func firstChargeTo(cats catalogs.Set, r *models.Receipt, d models.ReceiptDetail, line *models.FcfFee) types.ChargeTo {
	if line != nil && line.FirstChargeTo.Valid() {
		if line.FirstChargeTo == types.ChargeToBoth {
			return types.ChargeToCustomer
		}
		return line.FirstChargeTo
	}
	if cats.SupplierPaysUnlessLoading(string(r.TenderCode)) && !cats.GantryLoading(string(d.CollectionMethod)) {
		return types.ChargeToSupplier
	}
	return types.ChargeToCustomer
}

func qtyForUnit(d models.ReceiptDetail, unit string) decimal.Decimal {
	switch types.NormalizeBillingUnit(unit) {
	case "MT":
		return d.ReceivedMetricTonne()
	case "L":
		if !d.ReceivedQuantity().IsZero() {
			return d.ReceivedQuantity()
		}
		return d.ReceivedCubicMeter().Mul(decimal.NewFromInt(1000))
	default:
		if !d.ReceivedCubicMeter().IsZero() {
			return d.ReceivedCubicMeter()
		}
		return d.ReceivedQuantity()
	}
}

// detailBillsKoj is the KOJ infrastructure fee: external cargo discharged
// at KOJ that used TIPER's 10-inch pipeline. Internal receipts use FCF.
func detailBillsKoj(r *models.Receipt, d models.ReceiptDetail) bool {
	if r == nil || d.IsArchived {
		return false
	}
	if r.ReceiptType != types.ReceiptExternal || !r.UsesTiperPipeline {
		return false
	}
	return r.RouteCode.IsKOJ()
}

func (s *Service) runKOJ(tx *gorm.DB, r *models.Receipt) error {
	actor := systemUserID(tx)
	for _, d := range r.Details {
		if d.OriginDetailID != nil || !detailBillsKoj(r, d) {
			continue
		}
		var existing int64
		if err := tx.Model(&models.BillingRun{}).
			Where("ReceiptDetailID = ? AND FeeCode = ? AND BillingSequence = 1", d.ID, types.FeeKOJ).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		run := models.BillingRun{
			ReceiptDetailID: &d.ID,
			CustomerID:      &d.CustomerID,
			FeeCode:         types.FeeKOJ,
			BillingSequence: 1,
			PeriodStart:     r.BillingEffectiveDate,
			PeriodEnd:       r.BillingEffectiveDate,
			ChargeTo:        types.ChargeToCustomer,
			CurrencyCode:    "USD",
			Quantity:        d.ReceivedCubicMeter(),
			Status:          "draft",
			Source:          types.BillSrcKOJ,
			CreatedByID:     actor,
		}
		n, err := models.AssignDocumentNumber(tx, "koj-run", "KOJ")
		if err != nil {
			return err
		}
		run.DocumentNumber = n
		fee, ok := latestKojFee(tx, r.ProductID, r.BillingEffectiveDate)
		if !ok {
			run.ExceptionReason = "no approved KOJ fee for product"
		} else {
			if strings.EqualFold(fee.Unit, "MT") {
				run.Quantity = d.ReceivedMetricTonne()
			}
			run.Rate = priceFor(fee.SourceCurrencyCode, run.CurrencyCode, fee.SourcePrice, fee.HomePrice)
			run.Amount = run.Quantity.Mul(run.Rate).Round(2)
		}
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RunDueNth(asOf time.Time) error {
	var due []models.BillingRun
	if err := s.db.Where("FeeCode = ? AND Status IN ? AND PeriodEnd <= ?", types.FeeFSF, []types.DocumentStatus{types.DocApproved, types.DocPosted}, asOf).
		Find(&due).Error; err != nil {
		return err
	}
	actor := systemUserID(s.db)
	for _, prev := range due {
		if prev.ReceiptDetailID == nil {
			continue
		}
		var src models.ReceiptDetail
		if err := s.db.First(&src, *prev.ReceiptDetailID).Error; err != nil {
			continue
		}
		var rec models.Receipt
		if err := s.db.First(&rec, src.ReceiptID).Error; err != nil {
			continue
		}
		if rec.Details == nil {
			var all []models.ReceiptDetail
			_ = s.db.Where("ReceiptID = ?", rec.ID).Find(&all).Error
			rec.Details = all
		}
		parcels := []models.ReceiptDetail{src}
		var kids []models.ReceiptDetail
		if err := s.db.Where("OriginDetailID = ? AND IsArchived = 0", src.ID).Find(&kids).Error; err != nil {
			return err
		}
		parcels = append(parcels, kids...)
		cats, _ := catalogs.Load(s.db)
		for _, d := range parcels {
			if err := s.spawnNth(prev, rec, d, cats, actor); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) spawnNth(prev models.BillingRun, rec models.Receipt, d models.ReceiptDetail, cats catalogs.Set, actor uint) error {
	var exists int64
	if err := s.db.Model(&models.BillingRun{}).
		Where("ReceiptDetailID = ? AND FeeCode = ? AND BillingSequence = ?", d.ID, types.FeeFSF, prev.BillingSequence+1).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	if remainingQty(s.db, rec, d).LessThanOrEqual(decimal.Zero) {
		return nil
	}
	cid := d.CustomerID
	did := d.ID
	next := prev
	next.ID = 0
	next.UID = ""
	next.ReceiptDetailID = &did
	next.CustomerID = &cid
	next.SupplierID = rec.SupplierID
	next.BillingSequence = prev.BillingSequence + 1
	next.PeriodStart = prev.PeriodEnd.AddDate(0, 0, 1)
	if d.EffectiveFrom != nil {
		from := time.Date(d.EffectiveFrom.Year(), d.EffectiveFrom.Month(), d.EffectiveFrom.Day(), 0, 0, 0, 0, next.PeriodStart.Location())
		if next.PeriodStart.Before(from) {
			next.PeriodStart = from
		}
	}
	next.ChargeTo = types.ChargeToCustomer
	next.Status = "draft"
	next.Source = types.BillSrcNth
	next.ExceptionReason = ""
	next.CreatedByID = actor
	next.Lines = nil
	n, err := models.AssignDocumentNumber(s.db, "fsf-run", "FSF")
	if err != nil {
		return err
	}
	next.DocumentNumber = n
	line := lookupFcfLine(s.db, &rec, d, cats)
	nthDays := cycleDays(d)
	if line != nil && line.NthDays > 0 {
		nthDays = line.NthDays
	}
	next.PeriodEnd = next.PeriodStart.AddDate(0, 0, nthDays-1)
	if line == nil {
		next.ExceptionReason = "no approved FSF second-billing rate for this receipt"
	} else if err := applyFcfPhase(s.db, &rec, d, line, string(types.PhaseNth), &next); err != nil {
		next.ExceptionReason = err.Error()
	}
	return s.db.Create(&next).Error
}

func remainingStock(db *gorm.DB, r models.Receipt, d models.ReceiptDetail) (qty, cm, mt decimal.Decimal) {
	if db == nil {
		return
	}
	var row struct {
		Q  decimal.Decimal
		CM decimal.Decimal
		MT decimal.Decimal
	}
	_ = db.Raw(`
		SELECT ISNULL(SUM(Quantity), 0) AS Q, ISNULL(SUM(CubicMeter), 0) AS CM, ISNULL(SUM(MetricTonne), 0) AS MT
		FROM StockMovement
		WHERE CustomerID = ? AND ProductID = ? AND VesselID = ?
			AND CONVERT(date, VesselDate) = CONVERT(date, ?)
			AND StockStatusID = ? AND FinancialHold = ? AND IsProvision = 0`,
		d.CustomerID, r.ProductID, r.VesselID, r.VesselDate, d.StatusID(), d.FinancialHold,
	).Scan(&row).Error
	return row.Q, row.CM, row.MT
}

func remainingQty(db *gorm.DB, r models.Receipt, d models.ReceiptDetail) decimal.Decimal {
	qty, cm, mt := remainingStock(db, r, d)
	if cm.IsPositive() {
		return cm
	}
	if qty.IsPositive() {
		return qty
	}
	return mt
}

func remainingUnit(db *gorm.DB, r models.Receipt, d models.ReceiptDetail, unit string) decimal.Decimal {
	qty, cm, mt := remainingStock(db, r, d)
	switch types.NormalizeBillingUnit(unit) {
	case "MT":
		return mt
	case "L":
		return qty
	default:
		if !cm.IsZero() {
			return cm
		}
		return qty
	}
}

func (s *Service) RunDailyTBS(day time.Time) error {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)
	var events []models.InventoryEventLog
	if err := s.db.Where("EventType IN ? AND OccurredAt >= ? AND OccurredAt < ? AND Posted = 1",
		[]string{"loading", "gantry"}, start, end).Find(&events).Error; err != nil {
		return err
	}
	byCustomer := map[string][]models.InventoryEventLog{}
	for _, e := range events {
		byCustomer[e.CustomerCode] = append(byCustomer[e.CustomerCode], e)
	}
	actor := systemUserID(s.db)
	return s.db.Transaction(func(tx *gorm.DB) error {
		batch, hasBatch := latestTbsBatch(tx, start)
		for code, evs := range byCustomer {
			var cust models.Customer
			if err := tx.Where("Code = ?", code).First(&cust).Error; err != nil {
				continue
			}
			var exists int64
			if err := tx.Model(&models.BillingRun{}).
				Where("CustomerID = ? AND FeeCode = ? AND PeriodStart = ? AND Source = ?", cust.ID, types.FeeTBS, start, types.BillSrcTBSDaily).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				continue
			}
			run := models.BillingRun{
				CustomerID:      &cust.ID,
				FeeCode:         types.FeeTBS,
				BillingSequence: 1,
				PeriodStart:     start,
				PeriodEnd:       end.Add(-time.Second),
				ChargeTo:        types.ChargeToCustomer,
				CurrencyCode:    "TZS",
				Status:          "draft",
				Source:          types.BillSrcTBSDaily,
				CreatedByID:     actor,
			}
			n, err := models.AssignDocumentNumber(tx, "tbs-run", "TBS")
			if err != nil {
				return err
			}
			run.DocumentNumber = n
			var qty decimal.Decimal
			var amount decimal.Decimal
			var missing int
			rates := map[string]decimal.Decimal{}
			for _, e := range evs {
				qty = qty.Add(e.Quantity)
				if hasBatch {
					if fee, ok := tbsFeeForProduct(tx, batch, e.ProductCode); ok {
						rate := priceFor(fee.SourceCurrencyCode, "TZS", fee.SourcePrice, fee.HomePrice)
						amount = amount.Add(e.Quantity.Mul(rate).Round(2))
						rates[rate.String()] = rate
					} else {
						missing++
					}
				} else {
					missing++
				}
			}
			run.Quantity = qty
			run.Amount = amount
			if len(rates) == 1 {
				for _, r := range rates {
					run.Rate = r
				}
			}
			if !hasBatch {
				run.ExceptionReason = "no approved TBS fee batch"
			} else if missing > 0 {
				run.ExceptionReason = "missing TBS fee for one or more products"
			}
			if err := tx.Create(&run).Error; err != nil {
				return err
			}
			for _, e := range evs {
				id := e.ID
				line := models.ChargeLine{
					BillingRunID:        run.ID,
					FeeCode:             types.FeeTBS,
					InventoryEventLogID: &id,
					Quantity:            e.Quantity,
					CurrencyCode:        "TZS",
					TruckReference:      e.OrderNumber,
				}
				if hasBatch {
					if fee, ok := tbsFeeForProduct(tx, batch, e.ProductCode); ok {
						line.Rate = priceFor(fee.SourceCurrencyCode, "TZS", fee.SourcePrice, fee.HomePrice)
						line.Amount = e.Quantity.Mul(line.Rate).Round(2)
					}
				}
				if err := tx.Create(&line).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Service) RunMonthlyVCF(asOf time.Time) error {
	var batch models.VariableFeeBatch
	if err := s.db.Preload("Products.Contracts").
		Where("Status = ? AND EffectiveFrom <= ?", types.DocApproved, asOf).
		Order("EffectiveFrom DESC").Limit(1).Find(&batch).Error; err != nil || batch.ID == 0 {
		return fmt.Errorf("no approved variable storage fee batch")
	}
	start := time.Date(asOf.Year(), asOf.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	var details []models.ReceiptDetail
	if err := s.db.Joins("JOIN Receipt r ON r.ID = ReceiptDetail.ReceiptID").
		Where("r.Status = ? AND r.IsProvision = 0 AND ReceiptDetail.IsArchived = 0", types.ReceiptApproved).
		Find(&details).Error; err != nil {
		return err
	}
	type rateKey struct {
		product  uint
		contract string
	}
	byRate := map[rateKey]decimal.Decimal{}
	for _, p := range batch.Products {
		for _, c := range p.Contracts {
			byRate[rateKey{p.ProductID, string(c.ContractTypeCode)}] = c.FeeUSDCM
		}
	}
	actor := systemUserID(s.db)
	cats, _ := catalogs.Load(s.db)
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, d := range details {
			var rec models.Receipt
			if err := tx.First(&rec, d.ReceiptID).Error; err != nil {
				continue
			}
			if d.EffectiveFrom != nil && d.EffectiveFrom.After(end) {
				continue
			}
			var st models.StockStatus
			if sid := d.StatusID(); sid == 0 {
				continue
			} else if err := tx.First(&st, sid).Error; err == nil && st.IsProration {
				continue
			}
			qty := remainingQty(tx, rec, d)
			if qty.LessThanOrEqual(decimal.Zero) {
				continue
			}
			rate, ok := byRate[rateKey{rec.ProductID, string(d.ContractTypeCode)}]
			if !ok || rate.IsZero() {
				continue
			}
			var exists int64
			if err := tx.Model(&models.BillingRun{}).
				Where("ReceiptDetailID = ? AND FeeCode = ? AND PeriodStart = ?", d.ID, types.FeeVSF, start).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				continue
			}
			cid := d.CustomerID
			run := models.BillingRun{
				ReceiptDetailID: &d.ID,
				CustomerID:      &cid,
				SupplierID:      rec.SupplierID,
				FeeCode:         types.FeeVSF,
				BillingSequence: 1,
				PeriodStart:     start,
				PeriodEnd:       end,
				ChargeTo:        firstChargeTo(cats, &rec, d, nil),
				CurrencyCode:    "USD",
				Quantity:        qty,
				Rate:            rate,
				Amount:          qty.Mul(rate).Round(2),
				Status:          "draft",
				Source:          types.BillSrcVCFMonthly,
				CreatedByID:     actor,
			}
			n, err := models.AssignDocumentNumber(tx, "vsf-run", "VSF")
			if err != nil {
				return err
			}
			run.DocumentNumber = n
			if err := tx.Create(&run).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func fcfLineFits(cats catalogs.Set, line models.FcfFee, r *models.Receipt, d models.ReceiptDetail) bool {
	if line.ClassOfTrade != string(r.TenderCode) {
		return false
	}
	if line.DischargeRoute != r.RouteCode {
		return false
	}
	if line.ProcurementMethod != "" && line.ProcurementMethod != r.ProcurementMethodCode {
		return false
	}
	if line.IsPromotional != cats.Promotional(string(d.PricingNature)) {
		return false
	}
	if line.CollectionMethod != "" && line.CollectionMethod != d.CollectionMethod {
		return false
	}
	return true
}

func matchFcfLine(cats catalogs.Set, lines []models.FcfFee, r *models.Receipt, d models.ReceiptDetail) *models.FcfFee {
	var best *models.FcfFee
	bestScore := -1
	for i := range lines {
		line := &lines[i]
		if !fcfLineFits(cats, *line, r, d) {
			continue
		}
		score := 0
		if line.CollectionMethod != "" && line.CollectionMethod == d.CollectionMethod {
			score += 2
		}
		if d.NextBillingDays > 0 && line.FirstDays == d.NextBillingDays {
			score += 3
		}
		if score > bestScore {
			bestScore = score
			best = line
		}
	}
	return best
}

func lookupFcfLine(tx *gorm.DB, r *models.Receipt, d models.ReceiptDetail, cats catalogs.Set) *models.FcfFee {
	var batch models.FcfFeeBatch
	err := tx.Preload("Lines.Tiers").
		Where("Status = ? AND EffectiveFrom <= ?", types.DocApproved, r.BillingEffectiveDate).
		Order("EffectiveFrom DESC, Date DESC").Limit(1).Find(&batch).Error
	if err != nil || batch.ID == 0 {
		return nil
	}
	return matchFcfLine(cats, batch.Lines, r, d)
}

func pickTier(tiers []models.FcfFeeTier, phase string, qty decimal.Decimal) *models.FcfFeeTier {
	var best *models.FcfFeeTier
	for i := range tiers {
		t := &tiers[i]
		if !strings.EqualFold(t.Phase, phase) {
			continue
		}
		if qty.LessThan(t.FromQty) {
			continue
		}
		if t.ToQty != nil && qty.GreaterThan(*t.ToQty) {
			continue
		}
		if best == nil || t.FromQty.GreaterThan(best.FromQty) {
			best = t
		}
	}
	return best
}

func applyFcfPhase(tx *gorm.DB, r *models.Receipt, d models.ReceiptDetail, line *models.FcfFee, phase string, run *models.BillingRun) error {
	if line == nil {
		return errors.New("no approved FSF pricing model for this receipt")
	}
	kind := line.FirstRateKind
	unit := line.FirstUnit
	srcCurr := line.FirstSourceCurrencyCode
	source, home := line.FirstSourcePrice, line.FirstHomePrice
	if phase == string(types.PhaseNth) {
		kind = line.NthRateKind
		unit = line.NthUnit
		srcCurr = line.NthSourceCurrencyCode
		source, home = line.NthSourcePrice, line.NthHomePrice
	}
	unit = types.NormalizeBillingUnit(unit)
	qty := qtyForUnit(d, unit)
	if phase == string(types.PhaseNth) && tx != nil && r != nil {
		if rem := remainingUnit(tx, *r, d, unit); rem.IsPositive() {
			qty = rem
		}
	}
	if kind == types.RateTier {
		t := pickTier(line.Tiers, phase, qty)
		if t == nil {
			return fmt.Errorf("no FSF volume tier for %s quantity %s", unit, qty.String())
		}
		source, home = t.SourcePrice, t.HomePrice
	}
	homeCurr := srcCurr
	if tx != nil && line.BatchID != 0 {
		var batch models.FcfFeeBatch
		_ = tx.Select("CurrencyCode").First(&batch, line.BatchID).Error
		if batch.CurrencyCode != "" {
			homeCurr = batch.CurrencyCode
		}
	}
	if srcCurr == "" {
		srcCurr = "USD"
	}
	run.Quantity = qty
	run.CurrencyCode, run.Rate = resolveRunPrice(source, home, srcCurr, homeCurr)
	run.Amount = run.Quantity.Mul(run.Rate).Round(2)
	return nil
}

func resolveRunPrice(source, home decimal.Decimal, srcCurr, homeCurr string) (string, decimal.Decimal) {
	srcCurr = strings.ToUpper(strings.TrimSpace(srcCurr))
	homeCurr = strings.ToUpper(strings.TrimSpace(homeCurr))
	if homeCurr == "" || homeCurr == srcCurr {
		return srcCurr, source
	}
	if !home.IsZero() {
		return homeCurr, home
	}
	return srcCurr, source
}

// ApprovedFX returns the latest approved TZS-per-USD quote on or before asOf.
func ApprovedFX(db *gorm.DB, asOf time.Time, from, to string) (models.ExchangeRate, bool) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" {
		from = "USD"
	}
	if to == "" {
		to = "TZS"
	}
	var row models.ExchangeRate
	err := db.Where("Status = ? AND EffectiveFrom <= ? AND FromCurrency = ? AND ToCurrency = ?",
		types.DocApproved, asOf, from, to).
		Order("EffectiveFrom DESC").Limit(1).Find(&row).Error
	if err != nil || row.ID == 0 {
		return models.ExchangeRate{}, false
	}
	return row, true
}

func latestKojFee(tx *gorm.DB, productID uint, asOf time.Time) (models.KojFee, bool) {
	var batch models.KojFeeBatch
	if err := tx.Preload("Fees").Where("Status = ? AND EffectiveFrom <= ?", types.DocApproved, asOf).
		Order("EffectiveFrom DESC, Date DESC").Limit(1).Find(&batch).Error; err != nil || batch.ID == 0 {
		return models.KojFee{}, false
	}
	for _, f := range batch.Fees {
		if f.ProductID == productID {
			return f, true
		}
	}
	return models.KojFee{}, false
}

func latestTbsBatch(tx *gorm.DB, asOf time.Time) (models.TbsFeeBatch, bool) {
	var batch models.TbsFeeBatch
	if err := tx.Preload("Fees").Where("Status = ? AND EffectiveFrom <= ?", types.DocApproved, asOf).
		Order("EffectiveFrom DESC, Date DESC").Limit(1).Find(&batch).Error; err != nil || batch.ID == 0 {
		return models.TbsFeeBatch{}, false
	}
	return batch, true
}

func tbsFeeForProduct(tx *gorm.DB, batch models.TbsFeeBatch, productCode string) (models.TbsFee, bool) {
	productCode = strings.TrimSpace(productCode)
	if productCode == "" {
		return models.TbsFee{}, false
	}
	var pid uint
	for _, f := range batch.Fees {
		if f.Product != nil && strings.EqualFold(f.Product.Code, productCode) {
			return f, true
		}
	}
	var p models.Product
	if err := tx.Where("Code = ?", productCode).First(&p).Error; err != nil {
		return models.TbsFee{}, false
	}
	pid = p.ID
	for _, f := range batch.Fees {
		if f.ProductID == pid {
			return f, true
		}
	}
	return models.TbsFee{}, false
}

func (s *Service) Simulate(feeCode string, qty, rate decimal.Decimal) decimal.Decimal {
	return qty.Mul(rate).Round(2)
}
