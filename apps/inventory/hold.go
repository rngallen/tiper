package inventory

import (
	"fmt"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func (h handler) listHoldParcels(c fiber.Ctx) error {
	type parcel struct {
		CustomerID    string          `json:"customerId"`
		CustomerCode  string          `json:"customerCode"`
		CustomerName  string          `json:"customerName"`
		ProductID     string          `json:"productId"`
		ProductCode   string          `json:"productCode"`
		ProductName   string          `json:"productName"`
		VesselID      string          `json:"vesselId"`
		VesselCode    string          `json:"vesselCode"`
		VesselName    string          `json:"vesselName"`
		VesselDate    time.Time       `json:"vesselDate"`
		StockStatusID string          `json:"stockStatusId"`
		StatusName    string          `json:"statusName"`
		Quantity      decimal.Decimal `json:"quantity"`
		CubicMeter    decimal.Decimal `json:"cubicMeter"`
		MetricTonne   decimal.Decimal `json:"metricTonne"`
	}
	var rows []parcel
	q := h.db.WithContext(c.Context()).Table("StockBalance AS b").
		Select(`
			c.UID AS CustomerID, c.Code AS CustomerCode, c.Name AS CustomerName,
			p.UID AS ProductID, p.Code AS ProductCode, p.Name AS ProductName,
			v.UID AS VesselID, v.Code AS VesselCode, v.Name AS VesselName,
			b.VesselDate,
			st.UID AS StockStatusID, st.Name AS StatusName,
			b.Quantity, b.CubicMeter, b.MetricTonne
		`).
		Joins("JOIN Customer c ON c.ID = b.CustomerID").
		Joins("JOIN Product p ON p.ID = b.ProductID").
		Joins("JOIN Vessel v ON v.ID = b.VesselID").
		Joins("JOIN StockStatus st ON st.ID = b.StockStatusID").
		Where("b.FinancialHold = 1 AND b.IsProvision = 0 AND b.Quantity > 0")
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("c.Code LIKE ? OR c.Name LIKE ? OR p.Code LIKE ? OR v.Code LIKE ? OR v.Name LIKE ?",
			like, like, like, like, like)
	}
	if err := q.Order("c.Code, p.Code, b.VesselDate").Scan(&rows).Error; err != nil {
		return err
	}
	return response.OkDetail(c, rows)
}

func (h handler) listHoldReleases(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.FinancialHoldRelease{}))
	q, err = filterDocStatus(c, q, "Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search, "DocumentNumber", "Description")
	return response.ServeList(c, response.ListOpts[models.FinancialHoldRelease]{
		Query: q, Search: search,
		DateColumn:  "ReleaseDate",
		DefaultSort: "ReleaseDate",
		TieBreak:    "ID",
		Sort: map[string]string{
			"documentNumber": "DocumentNumber",
			"releaseDate":    "ReleaseDate",
			"status":         "Status",
		},
		Sheet: "Hold releases", File: "hold_releases",
		Headers: []any{"Document", "Date", "Description", "Status"},
		MapRow: func(r models.FinancialHoldRelease) []any {
			return []any{r.DocumentNumber, r.ReleaseDate.Format("2006-01-02"), r.Description, string(r.Status)}
		},
	})
}

func (h handler) getHoldRelease(c fiber.Ctx) error {
	var row models.FinancialHoldRelease
	if err := models.PreloadCreatedBy(h.db).
		Preload("Lines.Customer").Preload("Lines.Product").Preload("Lines.Vessel").Preload("Lines.StockStatus").
		Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	return response.OkDetail(c, row)
}

func (h handler) createHoldRelease(c fiber.Ctx) error {
	var in holdReleaseInput
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row := models.FinancialHoldRelease{
		ReleaseDate: in.ReleaseDate,
		Description: in.Description,
		Notes:       in.Notes,
		Status:      types.DocDraft,
		CreatedByID: middleware.GetUserIDFromContext(c),
	}
	if row.ReleaseDate.IsZero() {
		row.ReleaseDate = time.Now()
	}
	for i, line := range in.Lines {
		built, err := h.buildHoldLine(line)
		if err != nil {
			return response.BadRequest(c, fmt.Sprintf("line %d: %v", i+1, err))
		}
		row.Lines = append(row.Lines, built)
	}
	if len(row.Lines) == 0 {
		return response.BadRequest(c, "add at least one parcel to release")
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "hold", "HOLD")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		return tx.Create(&row).Error
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ActionCreate, row.UID, types.FinancialHoldContent, "hold release "+row.DocumentNumber+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) buildHoldLine(in holdLineInput) (models.FinancialHoldReleaseLine, error) {
	cid, err := lookupID[models.Customer](h.db, in.CustomerID)
	if err != nil {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("customer not found")
	}
	pid, err := lookupID[models.Product](h.db, in.ProductID)
	if err != nil {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("product not found")
	}
	vid, err := lookupID[models.Vessel](h.db, in.VesselID)
	if err != nil {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("vessel not found")
	}
	sid, err := lookupID[models.StockStatus](h.db, in.StockStatusID)
	if err != nil {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("status not found")
	}
	qty, err := decimal.NewFromString(strings.TrimSpace(in.Quantity))
	if err != nil || !qty.IsPositive() {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("quantity must be greater than zero")
	}
	vd := in.VesselDate
	if vd.IsZero() {
		return models.FinancialHoldReleaseLine{}, fmt.Errorf("vessel date is required")
	}
	cm, mt := decimal.Zero, decimal.Zero
	if s := strings.TrimSpace(in.CubicMeter); s != "" {
		cm, _ = decimal.NewFromString(s)
	}
	if s := strings.TrimSpace(in.MetricTonne); s != "" {
		mt, _ = decimal.NewFromString(s)
	}
	return models.FinancialHoldReleaseLine{
		CustomerID: cid, ProductID: pid, VesselID: vid, VesselDate: vd,
		StockStatusID: sid, Quantity: qty, CubicMeter: cm, MetricTonne: mt,
	}, nil
}

func (h handler) submitHoldRelease(c fiber.Ctx) error {
	var row models.FinancialHoldRelease
	if err := h.db.Preload("Lines").Where("UID = ?", c.Params("uid")).First(&row).Error; err != nil {
		return err
	}
	if len(row.Lines) == 0 {
		return response.BadRequest(c, "add at least one parcel before submit")
	}
	if err := h.db.Model(&row).Update("Status", types.DocSubmitted).Error; err != nil {
		return err
	}
	userID := middleware.GetUserIDFromContext(c)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := h.svc.Initiate(types.FinancialHoldContent, row.ID, &user, row.DocumentNumber, row.DocumentNumber); err != nil {
		return err
	}
	return response.OkMessage(c, "Hold release submitted for approval")
}
