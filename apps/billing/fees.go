package billing

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	billsvc "dfms/internal/billing"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func (h handler) listFcf(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.FcfFeeBatch{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description", "CurrencyCode")
	return response.ServeList(c, response.ListOpts[models.FcfFeeBatch]{
		Query: q, Search: search,
		DateColumn:  "Date",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"date":           "Date",
			"effectiveFrom":  "EffectiveFrom",
			"status":         "Status",
		},
		Sheet: "Fixed storage fees", File: "fsf_fees",
		Headers: []any{"Document", "Date", "Effective", "Description", "Created by", "Status"},
		MapRow: func(r models.FcfFeeBatch) []any {
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.EffectiveFrom.Format("2006-01-02"), r.Description, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) getFcf(c fiber.Ctx) error {
	row, err := h.loadFcf(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "fixed storage fee batch not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createFcf(c fiber.Ctx) error {
	var in fcfBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cats, err := h.catalogSet()
	if err != nil {
		return writeErr(c, err, "could not load catalogs")
	}
	if err := checkFcfCatalogs(cats, in); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	n, err := models.AssignDocumentNumber(h.db, "fsffee", "FSFFEE")
	if err != nil {
		return err
	}
	home := companyHomeCurrency(h.db)
	row := models.FcfFeeBatch{
		Date:           parseDate(in.Date),
		EffectiveFrom:  parseDate(in.EffectiveFrom),
		Description:    in.Description,
		CurrencyCode:   home,
		ExchangeRate:   parseDec(in.ExchangeRate),
		FxManual:       in.FxManual,
		DocumentNumber: n,
		CreatedByID:    middleware.GetUserIDFromContext(c),
		Status:         types.DocDraft,
	}
	lines := h.fcfLinesFrom(in, home)
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return insertFcfLines(tx, row.ID, lines)
	}); err != nil {
		return writeErr(c, err, "could not create fixed storage fee batch")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.BillingProfileContent, "fixed storage fee "+row.DocumentNumber+" created", nil, row)
	return h.reloadFcf(c, row.UID)
}

func (h handler) updateFcf(c fiber.Ctx) error {
	row, err := h.loadFcf(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "fixed storage fee batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	var in fcfBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cats, err := h.catalogSet()
	if err != nil {
		return writeErr(c, err, "could not load catalogs")
	}
	if err := checkFcfCatalogs(cats, in); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	before := row
	row.Date = parseDate(in.Date)
	row.EffectiveFrom = parseDate(in.EffectiveFrom)
	row.Description = in.Description
	row.CurrencyCode = companyHomeCurrency(h.db)
	row.ExchangeRate = parseDec(in.ExchangeRate)
	row.FxManual = in.FxManual
	lines := h.fcfLinesFrom(in, row.CurrencyCode)
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := replaceFcfLines(tx, row.ID, lines); err != nil {
			return err
		}
		return tx.Save(&row).Error
	}); err != nil {
		return writeErr(c, err, "could not update fixed storage fee batch")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.BillingProfileContent, "fixed storage fee "+row.DocumentNumber+" updated", before, row)
	return h.reloadFcf(c, row.UID, before)
}

func (h handler) fcfLinesFrom(in fcfBatchSchema, home string) []models.FcfFee {
	fx := parseDec(in.ExchangeRate)
	out := make([]models.FcfFee, 0, len(in.Lines))
	for _, line := range in.Lines {
		party := types.ChargeToCustomer
		if types.ChargeTo(line.FirstChargeTo).Valid() {
			party = types.ChargeTo(line.FirstChargeTo)
		}
		firstSrc := parseDec(line.FirstSourcePrice)
		nthSrc := parseDec(line.NthSourcePrice)
		fee := models.FcfFee{
			ClassOfTrade:            line.ClassOfTrade,
			ProcurementMethod:       types.ProcurementCode(line.ProcurementMethod),
			DischargeRoute:          types.DischargeRoute(line.DischargeRoute),
			CollectionMethod:        types.CollectionMethod(line.CollectionMethod),
			IsPromotional:           line.IsPromotional,
			FirstDays:               line.FirstDays,
			FirstChargeTo:           party,
			FirstRateKind:           types.NormalizeRateKind(line.FirstRateKind),
			FirstUnit:               line.FirstUnit,
			FirstSourceCurrencyCode: line.FirstSourceCurrencyCode,
			FirstSourcePrice:        firstSrc,
			FirstHomePrice:          billsvc.HomePrice(firstSrc, line.FirstSourceCurrencyCode, home, fx),
			NthDays:                 line.NthDays,
			NthRateKind:             types.NormalizeRateKind(line.NthRateKind),
			NthUnit:                 line.NthUnit,
			NthSourceCurrencyCode:   line.NthSourceCurrencyCode,
			NthSourcePrice:          nthSrc,
			NthHomePrice:            billsvc.HomePrice(nthSrc, line.NthSourceCurrencyCode, home, fx),
		}
		for _, t := range line.Tiers {
			src := parseDec(t.SourcePrice)
			ccy := line.FirstSourceCurrencyCode
			if t.Phase == string(types.PhaseNth) {
				ccy = line.NthSourceCurrencyCode
			}
			tier := models.FcfFeeTier{
				Phase:       t.Phase,
				FromQty:     parseDec(t.FromQty),
				SourcePrice: src,
				HomePrice:   billsvc.HomePrice(src, ccy, home, fx),
			}
			if t.ToQty != "" {
				d := parseDec(t.ToQty)
				tier.ToQty = &d
			}
			fee.Tiers = append(fee.Tiers, tier)
		}
		out = append(out, fee)
	}
	return out
}

func replaceFcfLines(tx *gorm.DB, batchID uint, lines []models.FcfFee) error {
	var ids []uint
	if err := tx.Model(&models.FcfFee{}).Where("BatchID = ?", batchID).Pluck("ID", &ids).Error; err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := tx.Where("FeeID IN ?", ids).Delete(&models.FcfFeeTier{}).Error; err != nil {
			return err
		}
		if err := tx.Where("BatchID = ?", batchID).Delete(&models.FcfFee{}).Error; err != nil {
			return err
		}
	}
	return insertFcfLines(tx, batchID, lines)
}

func insertFcfLines(tx *gorm.DB, batchID uint, lines []models.FcfFee) error {
	for i := range lines {
		lines[i].ID = 0
		lines[i].BatchID = batchID
		for j := range lines[i].Tiers {
			lines[i].Tiers[j].ID = 0
			lines[i].Tiers[j].FeeID = 0
		}
		if err := tx.Create(&lines[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h handler) loadFcf(uid string) (models.FcfFeeBatch, error) {
	var row models.FcfFeeBatch
	err := models.PreloadCreatedBy(h.db).Preload("Lines.Tiers").Where("UID = ?", uid).First(&row).Error
	return row, err
}

func (h handler) reloadFcf(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadFcf(uid)
	if err != nil {
		return err
	}
	return respondSaved(c, row, optionalArg(before))
}

func (h handler) listVar(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.VariableFeeBatch{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description")
	return response.ServeList(c, response.ListOpts[models.VariableFeeBatch]{
		Query: q, Search: search,
		DateColumn:  "Date",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"date":           "Date",
			"effectiveFrom":  "EffectiveFrom",
			"status":         "Status",
		},
		Sheet: "Variable storage fees", File: "vsf_fees",
		Headers: []any{"Document", "Date", "Effective", "Description", "Created by", "Status"},
		MapRow: func(r models.VariableFeeBatch) []any {
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.EffectiveFrom.Format("2006-01-02"), r.Description, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) getVar(c fiber.Ctx) error {
	row, err := h.loadVar(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "variable storage fee batch not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createVar(c fiber.Ctx) error {
	var in variableBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	products, miID, err := h.varProductsFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	n, err := models.AssignDocumentNumber(h.db, "vsffee", "VSFFEE")
	if err != nil {
		return err
	}
	row := models.VariableFeeBatch{
		Date:           parseDate(in.Date),
		EffectiveFrom:  parseDate(in.EffectiveFrom),
		Description:    in.Description,
		CurrencyCode:   companyHomeCurrency(h.db),
		ExchangeRate:   parseDec(in.ExchangeRate),
		FxManual:       in.FxManual,
		MiLossBatchID:  miID,
		DocumentNumber: n,
		CreatedByID:    middleware.GetUserIDFromContext(c),
		Status:         types.DocDraft,
	}
	if err := h.saveVarTree(h.db, &row, products); err != nil {
		return writeErr(c, err, "could not create variable storage fee batch")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.VariableFeeBatchContent, "variable storage fee "+row.DocumentNumber+" created", nil, row)
	return h.reloadVar(c, row.UID)
}

func (h handler) updateVar(c fiber.Ctx) error {
	row, err := h.loadVar(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "variable storage fee batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	var in variableBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	products, miID, err := h.varProductsFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	before := row
	row.Date = parseDate(in.Date)
	row.EffectiveFrom = parseDate(in.EffectiveFrom)
	row.Description = in.Description
	row.CurrencyCode = companyHomeCurrency(h.db)
	row.ExchangeRate = parseDec(in.ExchangeRate)
	row.FxManual = in.FxManual
	row.MiLossBatchID = miID
	if err := h.replaceVarTree(&row, products); err != nil {
		return writeErr(c, err, "could not update variable storage fee batch")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.VariableFeeBatchContent, "variable storage fee "+row.DocumentNumber+" updated", before, row)
	return h.reloadVar(c, row.UID, before)
}

func (h handler) varProductsFrom(in variableBatchSchema) ([]models.ProductConfig, *uint, error) {
	asOf := parseDate(in.EffectiveFrom)
	if asOf.IsZero() {
		asOf = parseDate(in.Date)
	}
	mi, err := h.resolveMiLoss(in.MiLossBatchID, asOf)
	if err != nil {
		return nil, nil, err
	}
	if mi.ID == 0 {
		return nil, nil, fmt.Errorf("approve at least one MI-loss batch before creating a variable storage fee")
	}
	miID := mi.ID
	byProduct := map[uint][]models.MiLoss{}
	for _, line := range mi.Lines {
		byProduct[line.ProductID] = append(byProduct[line.ProductID], line)
	}
	fx := parseDec(in.ExchangeRate)
	out := make([]models.ProductConfig, 0, len(in.Products))
	for _, p := range in.Products {
		var prod models.Product
		if err := h.db.Where("UID = ?", p.ProductID).First(&prod).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, errors.New("unknown product")
			}
			return nil, nil, err
		}
		if _, ok := byProduct[prod.ID]; !ok {
			label := strings.TrimSpace(prod.Code + " — " + prod.Name)
			return nil, nil, fmt.Errorf("%s is not among the products on MI-loss batch %s", label, mi.DocumentNumber)
		}
		cfg := models.ProductConfig{
			ProductID:  prod.ID,
			EwuraPrice: parseDec(p.EwuraPrice),
			Density:    parseDec(p.Density),
		}
		billsvc.FillProductConfig(&cfg, fx)
		for _, line := range byProduct[prod.ID] {
			rate := models.ProductContractRate{
				ProductID:        prod.ID,
				ContractTypeCode: line.ContractTypeCode,
				MiLossValue:      line.Value,
			}
			billsvc.FillContractRate(&cfg, &rate, fx)
			cfg.Contracts = append(cfg.Contracts, rate)
		}
		out = append(out, cfg)
	}
	return out, &miID, nil
}

func (h handler) resolveMiLoss(uid string, asOf time.Time) (models.MiLossBatch, error) {
	uid = strings.TrimSpace(uid)
	q := h.db.Preload("Products.Rates")
	asOf = dateOnly(asOf)
	if uid != "" {
		var row models.MiLossBatch
		if err := q.Where("UID = ?", uid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return row, fmt.Errorf("unknown MI-loss batch")
			}
			return row, err
		}
		if row.Status != types.DocApproved {
			return row, fmt.Errorf("MI-loss batch must be approved")
		}
		if err := miLossEffectiveOnOrBefore(row, asOf); err != nil {
			return row, err
		}
		return row, nil
	}
	var row models.MiLossBatch
	q = q.Where("Status = ?", types.DocApproved)
	if !asOf.IsZero() {
		// Same calendar day counts even if EffectiveFrom has a time component.
		q = q.Where("EffectiveFrom < ?", asOf.AddDate(0, 0, 1))
	}
	err := q.Order("EffectiveFrom DESC, Date DESC").Limit(1).Find(&row).Error
	if err != nil {
		return row, err
	}
	if row.ID == 0 {
		return row, fmt.Errorf("approve at least one MI-loss batch effective on or before the variable storage fee effective date")
	}
	return row, nil
}

func dateOnly(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// miLossEffectiveOnOrBefore allows an approved MI-loss card whose effective
// date is the same day as, or earlier than, the variable storage fee effective date.
// Batches are created before they take effect, so matching uses EffectiveFrom,
// not document Date. Loss rates are kept for the year; EWURA prices change monthly.
func miLossEffectiveOnOrBefore(mi models.MiLossBatch, feeEffective time.Time) error {
	feeEffective = dateOnly(feeEffective)
	if feeEffective.IsZero() {
		return nil
	}
	start := dateOnly(mi.EffectiveFrom)
	if start.IsZero() {
		start = dateOnly(mi.Date)
	}
	if start.After(feeEffective) {
		return fmt.Errorf(
			"MI-loss batch %s is effective %s; it must be the same day as or earlier than the variable storage fee effective date %s",
			mi.DocumentNumber, start.Format("2006-01-02"), feeEffective.Format("2006-01-02"),
		)
	}
	return nil
}

func (h handler) saveVarTree(db *gorm.DB, row *models.VariableFeeBatch, products []models.ProductConfig) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return insertVarProducts(tx, row.ID, products)
	})
}

func (h handler) replaceVarTree(row *models.VariableFeeBatch, products []models.ProductConfig) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VariableFeeBatch{}).Where("ID = ?", row.ID).Updates(map[string]any{
			"Date":           row.Date,
			"EffectiveFrom":  row.EffectiveFrom,
			"Description":    row.Description,
			"CurrencyCode":   row.CurrencyCode,
			"ExchangeRate":   row.ExchangeRate,
			"FxManual":       row.FxManual,
			"MiLossBatchID":  row.MiLossBatchID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [ProductContractRate] WHERE [BatchID] = ?", row.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [ProductConfig] WHERE [BatchID] = ?", row.ID).Error; err != nil {
			return err
		}
		return insertVarProducts(tx, row.ID, products)
	})
}

func insertVarProducts(tx *gorm.DB, batchID uint, products []models.ProductConfig) error {
	if len(products) == 0 {
		return nil
	}
	for i := range products {
		rates := products[i].Contracts
		products[i].ID = 0
		products[i].BatchID = batchID
		products[i].Contracts = nil
		if err := tx.Create(&products[i]).Error; err != nil {
			return err
		}
		for j := range rates {
			rates[j].ID = 0
			rates[j].ProductConfigID = products[i].ID
			rates[j].BatchID = batchID
			rates[j].ProductID = products[i].ProductID
			if err := tx.Create(&rates[j]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (h handler) loadVar(uid string) (models.VariableFeeBatch, error) {
	var row models.VariableFeeBatch
	err := models.PreloadCreatedBy(h.db).Preload("MiLossBatch").Preload("Products.Product").Preload("Products.Contracts").
		Where("UID = ?", uid).First(&row).Error
	return row, err
}

func (h handler) reloadVar(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadVar(uid)
	if err != nil {
		return err
	}
	return respondSaved(c, row, optionalArg(before))
}

func (h handler) listKoj(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.KojFeeBatch{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description", "CurrencyCode")
	return response.ServeList(c, response.ListOpts[models.KojFeeBatch]{
		Query: q, Search: search,
		DateColumn:  "Date",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"date":           "Date",
			"effectiveFrom":  "EffectiveFrom",
			"status":         "Status",
		},
		Sheet: "KOJ fees", File: "koj_fees",
		Headers: []any{"Document", "Date", "Effective", "Description", "Created by", "Status"},
		MapRow: func(r models.KojFeeBatch) []any {
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.EffectiveFrom.Format("2006-01-02"), r.Description, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) getKoj(c fiber.Ctx) error {
	row, err := h.loadKoj(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "KOJ fee batch not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createKoj(c fiber.Ctx) error {
	return h.createPriceBatch(c, "koj")
}

func (h handler) updateKoj(c fiber.Ctx) error {
	return h.updatePriceBatch(c, "koj")
}

func (h handler) listTbs(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.TbsFeeBatch{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description", "CurrencyCode")
	return response.ServeList(c, response.ListOpts[models.TbsFeeBatch]{
		Query: q, Search: search,
		DateColumn:  "Date",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"date":           "Date",
			"effectiveFrom":  "EffectiveFrom",
			"status":         "Status",
		},
		Sheet: "TBS fees", File: "tbs_fees",
		Headers: []any{"Document", "Date", "Effective", "Description", "Created by", "Status"},
		MapRow: func(r models.TbsFeeBatch) []any {
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.EffectiveFrom.Format("2006-01-02"), r.Description, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) getTbs(c fiber.Ctx) error {
	row, err := h.loadTbs(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "TBS fee batch not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createTbs(c fiber.Ctx) error {
	return h.createPriceBatch(c, "tbs")
}

func (h handler) updateTbs(c fiber.Ctx) error {
	return h.updatePriceBatch(c, "tbs")
}

func (h handler) createPriceBatch(c fiber.Ctx, kind string) error {
	var in priceBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	fees, err := h.priceFeesFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	userID := middleware.GetUserIDFromContext(c)
	date := parseDate(in.Date)
	eff := parseDate(in.EffectiveFrom)
	ccy := companyHomeCurrency(h.db)
	fx := parseDec(in.ExchangeRate)
	if kind == "tbs" {
		n, err := models.AssignDocumentNumber(h.db, "tbsfee", "TBSFEE")
		if err != nil {
			return err
		}
		row := models.TbsFeeBatch{
			Date: date, EffectiveFrom: eff, Description: in.Description,
			CurrencyCode: ccy, ExchangeRate: fx, FxManual: in.FxManual,
			DocumentNumber: n, CreatedByID: userID, Status: types.DocDraft,
		}
		tbs := toTbsFees(fees)
		if err := h.db.Transaction(func(tx *gorm.DB) error {
			if err := clashTbsRate(tx, 0, eff, tbs); err != nil {
				return err
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if len(tbs) == 0 {
				return nil
			}
			for i := range tbs {
				tbs[i].BatchID = row.ID
			}
			return tx.Create(&tbs).Error
		}); err != nil {
			if strings.Contains(err.Error(), "TBS for") {
				return badProduct(c, err)
			}
			return writeErr(c, err, "could not create TBS fee batch")
		}
		recordAudit(c, types.ActionCreate, row.UID, types.TbsFeeBatchContent, "TBS fee "+row.DocumentNumber+" created", nil, row)
		return h.reloadTbs(c, row.UID)
	}
	n, err := models.AssignDocumentNumber(h.db, "kojfee", "KOJFEE")
	if err != nil {
		return err
	}
	row := models.KojFeeBatch{
		Date: date, EffectiveFrom: eff, Description: in.Description,
		CurrencyCode: ccy, ExchangeRate: fx, FxManual: in.FxManual,
		DocumentNumber: n, CreatedByID: userID, Status: types.DocDraft,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := clashKojRate(tx, 0, eff, fees); err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if len(fees) == 0 {
			return nil
		}
		for i := range fees {
			fees[i].BatchID = row.ID
		}
		return tx.Create(&fees).Error
	}); err != nil {
		if strings.Contains(err.Error(), "KOJ for") {
			return badProduct(c, err)
		}
		return writeErr(c, err, "could not create KOJ fee batch")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.KojFeeBatchContent, "KOJ fee "+row.DocumentNumber+" created", nil, row)
	return h.reloadKoj(c, row.UID)
}

func (h handler) updatePriceBatch(c fiber.Ctx, kind string) error {
	var in priceBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	fees, err := h.priceFeesFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	date := parseDate(in.Date)
	eff := parseDate(in.EffectiveFrom)
	ccy := companyHomeCurrency(h.db)
	fx := parseDec(in.ExchangeRate)
	if kind == "tbs" {
		row, err := h.loadTbs(c.Params("uid"))
		if err != nil {
			return notFound(c, err, "TBS fee batch not found")
		}
		if !editable(row.Status) {
			return response.Conflict(c, "only a draft or returned batch can be edited")
		}
		before := row
		row.Date, row.EffectiveFrom, row.Description = date, eff, in.Description
		row.CurrencyCode, row.ExchangeRate, row.FxManual = ccy, fx, in.FxManual
		tbs := toTbsFees(fees)
		if err := h.db.Transaction(func(tx *gorm.DB) error {
			if err := clashTbsRate(tx, row.ID, eff, tbs); err != nil {
				return err
			}
			if err := tx.Model(&models.TbsFeeBatch{}).Where("ID = ?", row.ID).Updates(map[string]any{
				"Date": date, "EffectiveFrom": eff, "Description": in.Description,
				"CurrencyCode": ccy, "ExchangeRate": fx, "FxManual": in.FxManual,
			}).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM [TbsFee] WHERE [BatchID] = ?", row.ID).Error; err != nil {
				return err
			}
			if len(tbs) == 0 {
				return nil
			}
			for i := range tbs {
				tbs[i].ID = 0
				tbs[i].BatchID = row.ID
			}
			return tx.Create(&tbs).Error
		}); err != nil {
			if strings.Contains(err.Error(), "TBS for") {
				return badProduct(c, err)
			}
			return writeErr(c, err, "could not update TBS fee batch")
		}
		recordAudit(c, types.ActionUpdate, row.UID, types.TbsFeeBatchContent, "TBS fee "+row.DocumentNumber+" updated", before, row)
		return h.reloadTbs(c, row.UID, before)
	}
	row, err := h.loadKoj(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "KOJ fee batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	before := row
	row.Date, row.EffectiveFrom, row.Description = date, eff, in.Description
	row.CurrencyCode, row.ExchangeRate, row.FxManual = ccy, fx, in.FxManual
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := clashKojRate(tx, row.ID, eff, fees); err != nil {
			return err
		}
		if err := tx.Model(&models.KojFeeBatch{}).Where("ID = ?", row.ID).Updates(map[string]any{
			"Date": date, "EffectiveFrom": eff, "Description": in.Description,
			"CurrencyCode": ccy, "ExchangeRate": fx, "FxManual": in.FxManual,
		}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [KojFee] WHERE [BatchID] = ?", row.ID).Error; err != nil {
			return err
		}
		if len(fees) == 0 {
			return nil
		}
		for i := range fees {
			fees[i].ID = 0
			fees[i].BatchID = row.ID
		}
		return tx.Create(&fees).Error
	}); err != nil {
		if strings.Contains(err.Error(), "KOJ for") {
			return badProduct(c, err)
		}
		return writeErr(c, err, "could not update KOJ fee batch")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.KojFeeBatchContent, "KOJ fee "+row.DocumentNumber+" updated", before, row)
	return h.reloadKoj(c, row.UID, before)
}

func (h handler) priceFeesFrom(in priceBatchSchema) ([]models.KojFee, error) {
	fx := parseDec(in.ExchangeRate)
	home := companyHomeCurrency(h.db)
	eff := dateOnly(parseDate(in.EffectiveFrom))
	if eff.IsZero() {
		eff = dateOnly(parseDate(in.Date))
	}
	out := make([]models.KojFee, 0, len(in.Fees))
	for _, line := range in.Fees {
		id, err := productID(h.db, line.ProductID)
		if err != nil {
			return nil, err
		}
		src := parseDec(line.SourcePrice)
		out = append(out, models.KojFee{
			ProductID:          id,
			Unit:               line.Unit,
			SourceCurrencyCode: line.SourceCurrencyCode,
			EffectiveFrom:      eff,
			SourcePrice:        src,
			HomePrice:          billsvc.HomePrice(src, line.SourceCurrencyCode, home, fx),
		})
	}
	return out, nil
}

func toTbsFees(fees []models.KojFee) []models.TbsFee {
	out := make([]models.TbsFee, len(fees))
	for i, f := range fees {
		out[i] = models.TbsFee{
			ProductID:          f.ProductID,
			Unit:               f.Unit,
			SourceCurrencyCode: f.SourceCurrencyCode,
			EffectiveFrom:      f.EffectiveFrom,
			SourcePrice:        f.SourcePrice,
			HomePrice:          f.HomePrice,
		}
	}
	return out
}

func (h handler) loadKoj(uid string) (models.KojFeeBatch, error) {
	var row models.KojFeeBatch
	err := models.PreloadCreatedBy(h.db).Preload("Fees.Product").Where("UID = ?", uid).First(&row).Error
	return row, err
}

func (h handler) reloadKoj(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadKoj(uid)
	if err != nil {
		return err
	}
	return respondSaved(c, row, optionalArg(before))
}

func (h handler) loadTbs(uid string) (models.TbsFeeBatch, error) {
	var row models.TbsFeeBatch
	err := models.PreloadCreatedBy(h.db).Preload("Fees.Product").Where("UID = ?", uid).First(&row).Error
	return row, err
}

func (h handler) reloadTbs(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadTbs(uid)
	if err != nil {
		return err
	}
	return respondSaved(c, row, optionalArg(before))
}

func (h handler) listFX(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := response.ApplyLike(models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.ExchangeRate{})), search,
		"FromCurrency", "ToCurrency",
	)
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	return response.ServeList(c, response.ListOpts[models.ExchangeRate]{
		Query: q, Search: search,
		DateColumn:  "EffectiveFrom",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"effectiveFrom": "EffectiveFrom",
			"fromCurrency":  "FromCurrency",
			"toCurrency":    "ToCurrency",
			"rate":          "Rate",
			"status":        "Status",
		},
		Sheet: "Exchange rates", File: "exchange_rates",
		Headers: []any{"Effective", "From", "To", "Rate", "Created by", "Status"},
		MapRow: func(r models.ExchangeRate) []any {
			return []any{r.EffectiveFrom.Format("2006-01-02"), r.FromCurrency, r.ToCurrency, r.Rate.String(), creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) getFX(c fiber.Ctx) error {
	var row models.ExchangeRate
	if err := models.PreloadCreatedBy(h.db).Where("UID = ?", strings.TrimSpace(c.Params("uid"))).First(&row).Error; err != nil {
		return notFound(c, err, "exchange rate not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createFX(c fiber.Ctx) error {
	var in fxSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.ExchangeRate{
		EffectiveFrom: parseDate(in.EffectiveFrom),
		FromCurrency:  in.FromCurrency,
		ToCurrency:    in.ToCurrency,
		Rate:          parseDec(in.Rate),
		Status:        types.DocDraft,
		CreatedByID:   middleware.GetUserIDFromContext(c),
	}
	if err := h.db.Create(&row).Error; err != nil {
		return writeErr(c, err, "could not create exchange rate")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.ExchangeRateContent, fmt.Sprintf("FX %s/%s created", row.FromCurrency, row.ToCurrency), nil, row)
	return response.Created(c, row)
}

func (h handler) approvedFX(c fiber.Ctx) error {
	asOf := parseDate(c.Query("asOf"))
	if asOf.IsZero() {
		asOf = time.Now()
	}
	from := c.Query("from")
	to := c.Query("to")
	row, ok := billsvc.ApprovedFX(h.db.WithContext(c.Context()), asOf, from, to)
	if !ok {
		return response.OkDetail(c, fiber.Map{"found": false})
	}
	return response.OkDetail(c, fiber.Map{
		"found":         true,
		"id":            row.UID,
		"effectiveFrom": row.EffectiveFrom,
		"fromCurrency":  row.FromCurrency,
		"toCurrency":    row.ToCurrency,
		"rate":          row.Rate,
		"status":        row.Status,
	})
}

func (h handler) updateFX(c fiber.Ctx) error {
	var row models.ExchangeRate
	if err := firstUID(h.db, c.Params("uid"), &row); err != nil {
		return notFound(c, err, "exchange rate not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned rate can be edited")
	}
	var in fxSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	before := row
	row.EffectiveFrom = parseDate(in.EffectiveFrom)
	row.FromCurrency = in.FromCurrency
	row.ToCurrency = in.ToCurrency
	row.Rate = parseDec(in.Rate)
	if err := h.db.Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update exchange rate")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.ExchangeRateContent, fmt.Sprintf("FX %s/%s updated", row.FromCurrency, row.ToCurrency), before, row)
	return okUpdate(c, row, before, row)
}
