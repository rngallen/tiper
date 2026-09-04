package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/catalogs"
	"dfms/pkg/audit"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// rateQuoteNo is the operator-facing FX identity (pair + effective date).
func rateQuoteNo(from, to string, on time.Time) string {
	pair := strings.ToUpper(strings.TrimSpace(from)) + "/" + strings.ToUpper(strings.TrimSpace(to))
	if pair == "/" {
		return ""
	}
	if on.IsZero() {
		return pair
	}
	return pair + " · " + on.Format("2006-01-02")
}

func parseOps(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseOpsSearchRequest(c)
	if err != nil {
		return search, response.BadRequest(c, err.Error())
	}
	return search, nil
}

func parseCatalogue(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return search, response.BadRequest(c, err.Error())
	}
	return search, nil
}

func filterDocStatus(c fiber.Ctx, q *gorm.DB, column string) (*gorm.DB, error) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		return q, nil
	}
	if !types.DocumentStatus(status).Valid() {
		return q, response.BadRequest(c, "invalid status")
	}
	return q.Where(column+" = ?", status), nil
}

func bindBody[T any](c fiber.Ctx, dest *T) error {
	if err := c.Bind().Body(dest); err != nil {
		return response.BadRequestBind(c, err)
	}
	if s, ok := any(dest).(interface{ Sanitize() }); ok {
		s.Sanitize()
	}
	if v, ok := any(dest).(interface{ Validate(context.Context) error }); ok {
		if err := v.Validate(c.Context()); err != nil {
			return response.UnprocessableEntity(c, err)
		}
	}
	return nil
}

func respondSaved(c fiber.Ctx, details, before any) error {
	if c.Method() == fiber.MethodPost {
		return response.Created(c, details)
	}
	if before != nil {
		return response.Ok(c, audit.UpdateMessage(before, details), details)
	}
	return response.OkDetail(c, details)
}

func optionalArg(args []any) any {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

func okUpdate(c fiber.Ctx, details, before, after any) error {
	return response.Ok(c, audit.UpdateMessage(before, after), details)
}

func recordAudit(c fiber.Ctx, action types.Action, id string, ct types.ContentType, desc string, before, after any) {
	audit.RecordHTTP(c, types.ModuleBilling, action, id, ct, desc, before, after)
}

func firstUID[T any](db *gorm.DB, uid string, dest *T) error {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return gorm.ErrRecordNotFound
	}
	return db.Where("UID = ?", uid).First(dest).Error
}

func productID(db *gorm.DB, uid string) (uint, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return 0, errors.New("product is required")
	}
	var row models.Product
	if err := db.Where("UID = ?", uid).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("unknown product")
		}
		return 0, err
	}
	return row.ID, nil
}

func parseDec(s string) decimal.Decimal {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Truncate(24 * time.Hour)
	}
	return time.Time{}
}

func editable(status types.DocumentStatus) bool {
	return status == types.DocDraft || status == types.DocReturned
}

func notFound(c fiber.Ctx, err error, msg string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NotFound(c, msg)
	}
	logs.Error(err)
	return response.InternalServerError(c)
}

func writeErr(c fiber.Ctx, err error, duplicate string) error {
	if response.IsDuplicate(err) {
		return response.Conflict(c, duplicate)
	}
	logs.Error(err)
	return response.InternalServerError(c)
}

func (h handler) catalogSet() (catalogs.Set, error) {
	return catalogs.Load(h.db)
}

func companyHomeCurrency(db *gorm.DB) string {
	var row models.Company
	if err := db.Select("CurrencyCode").First(&row, 1).Error; err != nil {
		return "TZS"
	}
	if row.CurrencyCode != nil {
		if ccy := strings.ToUpper(strings.TrimSpace(*row.CurrencyCode)); len(ccy) == 3 {
			return ccy
		}
	}
	return "TZS"
}

func creatorName(c *models.CreatedByRef) string {
	if c == nil {
		return ""
	}
	return c.Name
}

func checkFcfCatalogs(cats catalogs.Set, in fcfBatchSchema) error {
	for _, line := range in.Lines {
		if err := catalogs.RequireActive(cats, "tender", line.ClassOfTrade); err != nil {
			return err
		}
		if err := catalogs.RequireActive(cats, "route", line.DischargeRoute); err != nil {
			return err
		}
		if err := catalogs.RequireActive(cats, "procurement", line.ProcurementMethod); err != nil {
			return err
		}
		if line.CollectionMethod != "" {
			if err := catalogs.RequireActive(cats, "delivery", line.CollectionMethod); err != nil {
				return err
			}
		}
		if err := catalogs.RequireCycle(cats, line.FirstDays); err != nil {
			return err
		}
		if err := catalogs.RequireCycle(cats, line.NthDays); err != nil {
			return err
		}
	}
	return nil
}

func checkMiCatalogs(cats catalogs.Set, in miLossBatchSchema) error {
	for _, p := range in.Products {
		for _, rate := range p.Rates {
			if err := catalogs.RequireActive(cats, "contract", rate.ContractTypeCode); err != nil {
				return err
			}
		}
	}
	return nil
}

// clashMiLossRate rejects a product × contract that already has a rate on
// another batch for the same effective date.
func clashMiLossRate(tx *gorm.DB, exceptBatch uint, asOf time.Time, lines []models.MiLoss) error {
	asOf = dateOnly(asOf)
	if asOf.IsZero() {
		return nil
	}
	for _, line := range lines {
		q := tx.Model(&models.MiLoss{}).Where(
			"ProductID = ? AND ContractTypeCode = ? AND EffectiveFrom = ?",
			line.ProductID, line.ContractTypeCode, asOf,
		)
		if exceptBatch > 0 {
			q = q.Where("BatchID <> ?", exceptBatch)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		label := "this product"
		var prod models.Product
		if err := tx.Select("Code", "Name").First(&prod, line.ProductID).Error; err == nil {
			name := strings.TrimSpace(prod.Code + " — " + prod.Name)
			if name != "" && name != "—" {
				label = name
			}
		}
		return fmt.Errorf(
			"MI-loss for %s / %s effective %s already exists on another batch",
			label, line.ContractTypeCode, asOf.Format("2006-01-02"),
		)
	}
	return nil
}

func productLabel(tx *gorm.DB, id uint) string {
	var prod models.Product
	if err := tx.Select("Code", "Name").First(&prod, id).Error; err != nil {
		return "this product"
	}
	name := strings.TrimSpace(prod.Code + " — " + prod.Name)
	if name == "" || name == "—" {
		return "this product"
	}
	return name
}

func clashKojRate(tx *gorm.DB, exceptBatch uint, asOf time.Time, fees []models.KojFee) error {
	asOf = dateOnly(asOf)
	if asOf.IsZero() {
		return nil
	}
	for _, fee := range fees {
		q := tx.Model(&models.KojFee{}).Where(
			"ProductID = ? AND Unit = ? AND SourceCurrencyCode = ? AND EffectiveFrom = ?",
			fee.ProductID, fee.Unit, fee.SourceCurrencyCode, asOf,
		)
		if exceptBatch > 0 {
			q = q.Where("BatchID <> ?", exceptBatch)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		return fmt.Errorf(
			"KOJ for %s / %s / %s effective %s already exists on another batch",
			productLabel(tx, fee.ProductID), fee.Unit, fee.SourceCurrencyCode, asOf.Format("2006-01-02"),
		)
	}
	return nil
}

func clashTbsRate(tx *gorm.DB, exceptBatch uint, asOf time.Time, fees []models.TbsFee) error {
	asOf = dateOnly(asOf)
	if asOf.IsZero() {
		return nil
	}
	for _, fee := range fees {
		q := tx.Model(&models.TbsFee{}).Where(
			"ProductID = ? AND Unit = ? AND SourceCurrencyCode = ? AND EffectiveFrom = ?",
			fee.ProductID, fee.Unit, fee.SourceCurrencyCode, asOf,
		)
		if exceptBatch > 0 {
			q = q.Where("BatchID <> ?", exceptBatch)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		return fmt.Errorf(
			"TBS for %s / %s / %s effective %s already exists on another batch",
			productLabel(tx, fee.ProductID), fee.Unit, fee.SourceCurrencyCode, asOf.Format("2006-01-02"),
		)
	}
	return nil
}

func badProduct(c fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if msg == "product is required" || msg == "unknown product" ||
		strings.Contains(msg, "unknown contract") || strings.Contains(msg, "invalid ") ||
		strings.Contains(msg, "MI-loss") {
		return response.UnprocessableEntity(c, err)
	}
	logs.Error(err)
	return response.InternalServerError(c)
}
