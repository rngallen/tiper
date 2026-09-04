package billing

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/attach"
	billsvc "dfms/internal/billing"
	"dfms/internal/catalogs"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func Router(app *fiber.App, svc *billsvc.Service, _ *wfengine.Engine) {
	h := handler{db: db.Db, svc: svc}
	g := app.Group("/api/v1/billing", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	read := middleware.PermissionMiddleware(permissions.BillingRead, permissions.PricesRead)
	review := middleware.PermissionMiddleware(permissions.BillingRead, permissions.PricesRead, permissions.WorkflowTasks)
	write := middleware.PermissionMiddleware(permissions.BillingCreate, permissions.BillingUpdate, permissions.PricesCreate, permissions.PricesUpdate)
	run := middleware.PermissionMiddleware(permissions.BillingRun)
	submit := middleware.PermissionMiddleware(permissions.BillingSubmit, permissions.PricesSubmit)

	g.Get("/fsf/batches", read, h.listFcf)
	g.Post("/fsf/batches", write, h.createFcf)
	g.Get("/fsf/batches/:uid", review, h.getFcf)
	g.Put("/fsf/batches/:uid", write, h.updateFcf)
	g.Post("/fsf/batches/:uid/submit", submit, h.submitFcf)
	g.Get("/fsf/batches/:uid/workflow", review, h.fcfWorkflow)
	attach.Register(g, "/fsf/batches/:uid", review, write, h.db, types.BillingProfileContent, h.attachFcf)

	g.Get("/runs", read, h.listRuns)
	g.Get("/runs/:uid", review, h.getRun)
	g.Post("/runs/:uid/submit", submit, h.submitRun)
	g.Get("/runs/:uid/workflow", review, h.runWorkflow)
	attach.Register(g, "/runs/:uid", review, write, h.db, types.BillingRunContent, h.attachRun)
	g.Post("/engine/simulate", read, h.simulate)
	g.Post("/engine/nth", run, h.runNth)
	g.Post("/engine/tbs", run, h.runTBS)
	g.Post("/engine/vsf", run, h.runVCF)

	g.Get("/vsf/batches", read, h.listVar)
	g.Post("/vsf/batches", write, h.createVar)
	g.Get("/vsf/batches/:uid", review, h.getVar)
	g.Put("/vsf/batches/:uid", write, h.updateVar)
	g.Post("/vsf/batches/:uid/submit", submit, h.submitVar)
	g.Get("/vsf/batches/:uid/workflow", review, h.varWorkflow)
	attach.Register(g, "/vsf/batches/:uid", review, write, h.db, types.VariableFeeBatchContent, h.attachVar)

	g.Get("/koj/batches", read, h.listKoj)
	g.Post("/koj/batches", write, h.createKoj)
	g.Get("/koj/batches/:uid", review, h.getKoj)
	g.Put("/koj/batches/:uid", write, h.updateKoj)
	g.Post("/koj/batches/:uid/submit", submit, h.submitKoj)
	g.Get("/koj/batches/:uid/workflow", review, h.kojWorkflow)
	attach.Register(g, "/koj/batches/:uid", review, write, h.db, types.KojFeeBatchContent, h.attachKoj)

	g.Get("/tbs/batches", read, h.listTbs)
	g.Post("/tbs/batches", write, h.createTbs)
	g.Get("/tbs/batches/:uid", review, h.getTbs)
	g.Put("/tbs/batches/:uid", write, h.updateTbs)
	g.Post("/tbs/batches/:uid/submit", submit, h.submitTbs)
	g.Get("/tbs/batches/:uid/workflow", review, h.tbsWorkflow)
	attach.Register(g, "/tbs/batches/:uid", review, write, h.db, types.TbsFeeBatchContent, h.attachTbs)

	g.Get("/miloss/batches", read, h.listMi)
	g.Post("/miloss/batches", write, h.createMi)
	g.Get("/miloss/batches/:uid/pdf", review, h.printMi)
	g.Get("/miloss/batches/:uid", review, h.getMi)
	g.Put("/miloss/batches/:uid", write, h.updateMi)
	g.Post("/miloss/batches/:uid/products", write, h.addMiProduct)
	g.Delete("/miloss/batches/:uid/products/:productId", write, h.removeMiProduct)
	g.Post("/miloss/batches/:uid/rates", write, h.addMiRate)
	g.Post("/miloss/batches/:uid/submit", submit, h.submitMi)
	g.Get("/miloss/batches/:uid/workflow", review, h.miWorkflow)
	attach.Register(g, "/miloss/batches/:uid", review, write, h.db, types.MiLossBatchContent, h.attachMi)

	g.Get("/fx/approved", read, h.approvedFX)
	g.Get("/fx", read, h.listFX)
	g.Post("/fx", write, h.createFX)
	g.Get("/fx/:uid", review, h.getFX)
	g.Put("/fx/:uid", write, h.updateFX)
	g.Post("/fx/:uid/submit", submit, h.submitFX)
	g.Get("/fx/:uid/workflow", review, h.fxWorkflow)
	attach.Register(g, "/fx/:uid", review, write, h.db, types.ExchangeRateContent, h.attachFX)

	g.Get("/change-of-service/parcels", read, h.listCOSParcels)
	g.Get("/change-of-service", read, h.listCOS)
	g.Post("/change-of-service", write, h.createCOS)
	g.Get("/change-of-service/:uid", review, h.getCOS)
	g.Put("/change-of-service/:uid", write, h.updateCOS)
	g.Post("/change-of-service/:uid/submit", submit, h.submitCOS)
	g.Get("/change-of-service/:uid/workflow", review, h.cosWorkflow)
	attach.Register(g, "/change-of-service/:uid", review, write, h.db, types.ChangeOfServiceContent, h.attachCOS)
}

type handler struct {
	db  *gorm.DB
	svc *billsvc.Service
}

func (h handler) listRuns(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.BillingRun{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "FeeCode", "CurrencyCode", "Source", "ExceptionReason")
	return response.ServeList(c, response.ListOpts[models.BillingRun]{
		Query: q, Search: search,
		DateColumn:  "CreatedAt",
		DefaultSort: "CreatedAt",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"feeCode":        "FeeCode",
			"amount":         "Amount",
			"status":         "Status",
			"periodStart":    "PeriodStart",
		},
		Sheet: "Billing runs", File: "billing_runs",
		Headers: []any{"Document", "Fee", "Period start", "Period end", "Amount", "Currency", "Created by", "Status"},
		MapRow: func(r models.BillingRun) []any {
			return []any{r.DocumentNumber, string(r.FeeCode), r.PeriodStart.Format("2006-01-02"), r.PeriodEnd.Format("2006-01-02"), r.Amount.String(), r.CurrencyCode, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) simulate(c fiber.Ctx) error {
	var in simulateSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	return response.OkDetail(c, fiber.Map{"amount": h.svc.Simulate(in.FeeCode, parseDec(in.Quantity), parseDec(in.Rate))})
}

func (h handler) runNth(c fiber.Ctx) error {
	if err := h.svc.RunDueNth(time.Now()); err != nil {
		return err
	}
	return response.OkMessage(c, "Nth billing prepared")
}

func (h handler) runTBS(c fiber.Ctx) error {
	if err := h.svc.RunDailyTBS(time.Now().AddDate(0, 0, -1)); err != nil {
		return err
	}
	return response.OkMessage(c, "TBS billing prepared")
}

func (h handler) runVCF(c fiber.Ctx) error {
	if err := h.svc.RunMonthlyVCF(time.Now()); err != nil {
		return err
	}
	return response.OkMessage(c, "VSF billing prepared")
}

func (h handler) getRun(c fiber.Ctx) error {
	var row models.BillingRun
	if err := models.PreloadCreatedBy(h.db).Preload("Lines").Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) submitRun(c fiber.Ctx) error {
	return h.submitDoc(c, &models.BillingRun{}, types.BillingRunContent)
}

func (h handler) submitFcf(c fiber.Ctx) error {
	return h.submitDoc(c, &models.FcfFeeBatch{}, types.BillingProfileContent)
}

func (h handler) submitFX(c fiber.Ctx) error {
	return h.submitDoc(c, &models.ExchangeRate{}, types.ExchangeRateContent)
}

func (h handler) submitVar(c fiber.Ctx) error {
	return h.submitDoc(c, &models.VariableFeeBatch{}, types.VariableFeeBatchContent)
}

func (h handler) submitKoj(c fiber.Ctx) error {
	return h.submitDoc(c, &models.KojFeeBatch{}, types.KojFeeBatchContent)
}

func (h handler) submitTbs(c fiber.Ctx) error {
	return h.submitDoc(c, &models.TbsFeeBatch{}, types.TbsFeeBatchContent)
}

func (h handler) submitMi(c fiber.Ctx) error {
	return h.submitDoc(c, &models.MiLossBatch{}, types.MiLossBatchContent)
}

func (h handler) fcfWorkflow(c fiber.Ctx) error {
	row, err := h.loadFcf(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "fixed storage fee batch not found")
	}
	return h.docWorkflow(c, types.BillingProfileContent, row.ID)
}
func (h handler) varWorkflow(c fiber.Ctx) error {
	row, err := h.loadVar(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "variable storage fee batch not found")
	}
	return h.docWorkflow(c, types.VariableFeeBatchContent, row.ID)
}
func (h handler) kojWorkflow(c fiber.Ctx) error {
	row, err := h.loadKoj(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "KOJ fee batch not found")
	}
	return h.docWorkflow(c, types.KojFeeBatchContent, row.ID)
}
func (h handler) tbsWorkflow(c fiber.Ctx) error {
	row, err := h.loadTbs(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "TBS fee batch not found")
	}
	return h.docWorkflow(c, types.TbsFeeBatchContent, row.ID)
}
func (h handler) fxWorkflow(c fiber.Ctx) error {
	var row models.ExchangeRate
	if err := firstUID(h.db, c.Params("uid"), &row); err != nil {
		return notFound(c, err, "exchange rate not found")
	}
	return h.docWorkflow(c, types.ExchangeRateContent, row.ID)
}

func (h handler) docWorkflow(c fiber.Ctx, ct types.ContentType, objectID uint) error {
	view, err := h.svc.DocumentWorkflow(c.Context(), ct, objectID)
	if err != nil {
		if errors.Is(err, wfengine.ErrInstanceNotFound) {
			return response.OkDetail(c, fiber.Map{"id": "", "history": []any{}, "process": nil})
		}
		return err
	}
	return response.OkDetail(c, view)
}

func (h handler) submitDoc(c fiber.Ctx, dest any, ct types.ContentType) error {
	if err := h.db.Where("UID = ?", c.Params("uid")).First(dest).Error; err != nil {
		return err
	}
	if err := h.requireBatchLines(dest); err != nil {
		return response.BadRequest(c, err.Error())
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	id, no := docIdentity(dest)
	_ = h.db.Model(dest).Update("Status", types.DocSubmitted)
	if err := h.svc.Initiate(ct, id, &user, no, no); err != nil {
		return err
	}
	return response.OkMessage(c, "Submitted for approval")
}

func (h handler) requireBatchLines(dest any) error {
	switch row := dest.(type) {
	case *models.MiLossBatch:
		return h.countChildren(&models.MiLoss{}, row.ID, "add at least one MI-loss rate before submitting for approval")
	case *models.FcfFeeBatch:
		return h.countChildren(&models.FcfFee{}, row.ID, "add at least one FSF pricing line before submitting for approval")
	case *models.VariableFeeBatch:
		return h.countChildren(&models.ProductConfig{}, row.ID, "add at least one product before submitting for approval")
	case *models.KojFeeBatch:
		return h.countChildren(&models.KojFee{}, row.ID, "add at least one KOJ fee line before submitting for approval")
	case *models.TbsFeeBatch:
		return h.countChildren(&models.TbsFee{}, row.ID, "add at least one TBS fee line before submitting for approval")
	case *models.ChangeOfService:
		if row.ReceiptDetailID == 0 || row.ToCollection == "" {
			return errors.New("select a vessel parcel and the new delivery method before submitting")
		}
	}
	return nil
}

func (h handler) countChildren(model any, batchID uint, msg string) error {
	var n int64
	if err := h.db.Model(model).Where("BatchID = ?", batchID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errors.New(msg)
	}
	return nil
}

func docIdentity(v any) (uint, string) {
	switch row := v.(type) {
	case *models.BillingRun:
		return row.ID, row.DocumentNumber
	case *models.FcfFeeBatch:
		return row.ID, row.DocumentNumber
	case *models.ExchangeRate:
		return row.ID, rateQuoteNo(row.FromCurrency, row.ToCurrency, row.EffectiveFrom)
	case *models.VariableFeeBatch:
		return row.ID, row.DocumentNumber
	case *models.KojFeeBatch:
		return row.ID, row.DocumentNumber
	case *models.TbsFeeBatch:
		return row.ID, row.DocumentNumber
	case *models.MiLossBatch:
		return row.ID, row.DocumentNumber
	case *models.ChangeOfService:
		return row.ID, row.DocumentNumber
	default:
		return 0, ""
	}
}

func (h handler) listMi(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.MiLossBatch{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description")
	return response.ServeList(c, response.ListOpts[models.MiLossBatch]{
		Query: q, Search: search,
		DateColumn:  "Date",
		DefaultSort: "EffectiveFrom",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"date":           "Date",
			"effectiveFrom":  "EffectiveFrom",
			"status":         "Status",
		},
		Sheet: "MI loss", File: "miloss",
		Headers: []any{"Document", "Date", "Effective", "Description", "Created by", "Status"},
		MapRow: func(r models.MiLossBatch) []any {
			return []any{r.DocumentNumber, r.Date.Format("2006-01-02"), r.EffectiveFrom.Format("2006-01-02"), r.Description, creatorName(r.Creator), string(r.Status)}
		},
	})
}

func (h handler) createMi(c fiber.Ctx) error {
	var in miLossBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cats, err := h.catalogSet()
	if err != nil {
		return writeErr(c, err, "could not load catalogs")
	}
	if err := checkMiCatalogs(cats, in); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	products, lines, err := h.miTreeFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	n, err := models.AssignDocumentNumber(h.db, "miloss", "MILOSS")
	if err != nil {
		return err
	}
	row := models.MiLossBatch{
		Date:           parseDate(in.Date),
		EffectiveFrom:  parseDate(in.EffectiveFrom),
		Description:    in.Description,
		DocumentNumber: n,
		CreatedByID:    middleware.GetUserIDFromContext(c),
		Status:         types.DocDraft,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := clashMiLossRate(tx, row.ID, row.EffectiveFrom, lines); err != nil {
			return err
		}
		return insertMiTree(tx, row.ID, row.EffectiveFrom, products)
	}); err != nil {
		if strings.Contains(err.Error(), "MI-loss") {
			return badProduct(c, err)
		}
		return writeErr(c, err, "this product, contract, and effective date already have an MI-loss rate")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.MiLossBatchContent, "MI loss "+row.DocumentNumber+" created", nil, row)
	return h.reloadMi(c, row.UID)
}

func (h handler) updateMi(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	var in miLossBatchSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cats, err := h.catalogSet()
	if err != nil {
		return writeErr(c, err, "could not load catalogs")
	}
	if err := checkMiCatalogs(cats, in); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	products, lines, err := h.miTreeFrom(in)
	if err != nil {
		return badProduct(c, err)
	}
	before := row
	date := dateOnly(parseDate(in.Date))
	eff := dateOnly(parseDate(in.EffectiveFrom))
	row.Date = date
	row.EffectiveFrom = eff
	row.Description = in.Description
	row.Products = nil
	row.Lines = nil
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := clashMiLossRate(tx, row.ID, eff, lines); err != nil {
			return err
		}
		if err := tx.Model(&models.MiLossBatch{}).Where("ID = ?", row.ID).Updates(map[string]any{
			"Date": date, "EffectiveFrom": eff, "Description": in.Description,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MiLoss{}).Where("BatchID = ?", row.ID).
			Update("EffectiveFrom", eff).Error; err != nil {
			return err
		}
		if len(in.Products) == 0 && len(products) == 0 {
			return nil
		}
		if err := tx.Exec("DELETE FROM [MiLoss] WHERE [BatchID] = ?", row.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [MiLossProduct] WHERE [BatchID] = ?", row.ID).Error; err != nil {
			return err
		}
		return insertMiTree(tx, row.ID, eff, products)
	}); err != nil {
		if strings.Contains(err.Error(), "MI-loss") {
			return badProduct(c, err)
		}
		return writeErr(c, err, "this product, contract, and effective date already have an MI-loss rate")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.MiLossBatchContent, "MI loss "+row.DocumentNumber+" updated", before, row)
	return h.reloadMi(c, row.UID, before)
}

func (h handler) addMiProduct(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	var in miLossProductAddSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	pid, err := productID(h.db, in.ProductID)
	if err != nil {
		return badProduct(c, err)
	}
	var n int64
	if err := h.db.Model(&models.MiLossProduct{}).Where("BatchID = ? AND ProductID = ?", row.ID, pid).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return response.Conflict(c, "this product is already on the batch")
	}
	if err := h.db.Create(&models.MiLossProduct{BatchID: row.ID, ProductID: pid}).Error; err != nil {
		return writeErr(c, err, "this product is already on the batch")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.MiLossBatchContent, "MI loss "+row.DocumentNumber+" product added", row, nil)
	return h.reloadMi(c, row.UID)
}

func (h handler) removeMiProduct(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	pid, err := productID(h.db, c.Params("productId"))
	if err != nil {
		return badProduct(c, err)
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("BatchID = ? AND ProductID = ?", row.ID, pid).Delete(&models.MiLoss{}).Error; err != nil {
			return err
		}
		return tx.Where("BatchID = ? AND ProductID = ?", row.ID, pid).Delete(&models.MiLossProduct{}).Error
	}); err != nil {
		return writeErr(c, err, "could not remove this product")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.MiLossBatchContent, "MI loss "+row.DocumentNumber+" product removed", row, nil)
	return h.reloadMi(c, row.UID, row)
}

func (h handler) addMiRate(c fiber.Ctx) error {
	row, err := h.loadMi(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "MI loss batch not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned batch can be edited")
	}
	var in miLossRateAddSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	cats, err := h.catalogSet()
	if err != nil {
		return writeErr(c, err, "could not load catalogs")
	}
	if err := catalogs.RequireActive(cats, "contract", in.ContractTypeCode); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	pid, err := productID(h.db, in.ProductID)
	if err != nil {
		return badProduct(c, err)
	}
	rate := models.MiLoss{
		ProductID:        pid,
		ContractTypeCode: types.ContractCode(in.ContractTypeCode),
		Value:            parseDec(in.Value),
		EffectiveFrom:    dateOnly(row.EffectiveFrom),
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := clashMiLossRate(tx, row.ID, row.EffectiveFrom, []models.MiLoss{rate}); err != nil {
			return err
		}
		var prod models.MiLossProduct
		err := tx.Where("BatchID = ? AND ProductID = ?", row.ID, pid).First(&prod).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			prod = models.MiLossProduct{BatchID: row.ID, ProductID: pid}
			if err := tx.Create(&prod).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		var n int64
		if err := tx.Model(&models.MiLoss{}).Where(
			"ProductID = ? AND ContractTypeCode = ? AND EffectiveFrom = ?",
			pid, rate.ContractTypeCode, dateOnly(row.EffectiveFrom),
		).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("MI-loss for this product / %s effective %s already exists", rate.ContractTypeCode, dateOnly(row.EffectiveFrom).Format("2006-01-02"))
		}
		rate.ProductRowID = prod.ID
		rate.BatchID = row.ID
		return tx.Create(&rate).Error
	}); err != nil {
		if strings.Contains(err.Error(), "MI-loss") {
			return badProduct(c, err)
		}
		return writeErr(c, err, "this product, contract, and effective date already have an MI-loss rate")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.MiLossBatchContent, "MI loss "+row.DocumentNumber+" rate added", row, nil)
	return h.reloadMi(c, row.UID)
}

func (h handler) miTreeFrom(in miLossBatchSchema) ([]models.MiLossProduct, []models.MiLoss, error) {
	products := make([]models.MiLossProduct, 0, len(in.Products))
	rates := make([]models.MiLoss, 0)
	for _, p := range in.Products {
		id, err := productID(h.db, p.ProductID)
		if err != nil {
			return nil, nil, err
		}
		row := models.MiLossProduct{ProductID: id}
		for _, rate := range p.Rates {
			r := models.MiLoss{
				ProductID:        id,
				ContractTypeCode: types.ContractCode(rate.ContractTypeCode),
				Value:            parseDec(rate.Value),
			}
			row.Rates = append(row.Rates, r)
			rates = append(rates, r)
		}
		products = append(products, row)
	}
	return products, rates, nil
}

func insertMiTree(tx *gorm.DB, batchID uint, effective time.Time, products []models.MiLossProduct) error {
	effective = dateOnly(effective)
	for i := range products {
		products[i].ID = 0
		products[i].UID = ""
		products[i].BatchID = batchID
		rates := products[i].Rates
		products[i].Rates = nil
		if err := tx.Create(&products[i]).Error; err != nil {
			return err
		}
		for j := range rates {
			rates[j].ID = 0
			rates[j].UID = ""
			rates[j].ProductRowID = products[i].ID
			rates[j].BatchID = batchID
			rates[j].ProductID = products[i].ProductID
			rates[j].EffectiveFrom = effective
			if err := tx.Create(&rates[j]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (h handler) reloadMi(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadMi(uid)
	if err != nil {
		return err
	}
	return respondSaved(c, row, optionalArg(before))
}
