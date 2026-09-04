package billing

import (
	"errors"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type cosParcel struct {
	ID               string          `json:"id"`
	ReceiptNumber    string          `json:"receiptNumber"`
	VesselID         string          `json:"vesselId"`
	VesselName       string          `json:"vesselName"`
	VesselDate       time.Time       `json:"vesselDate"`
	ProductID        string          `json:"productId"`
	ProductCode      string          `json:"productCode"`
	ProductName      string          `json:"productName"`
	CollectionMethod string          `json:"collectionMethod"`
	ContractTypeCode string          `json:"contractTypeCode"`
	Quantity         decimal.Decimal `json:"quantity"`
}

func (h handler) listCOS(c fiber.Ctx) error {
	search, err := parseOps(c)
	if err != nil {
		return err
	}
	q := models.PreloadCreatedBy(h.db.WithContext(c.Context()).Model(&models.ChangeOfService{})).
		Joins("LEFT JOIN [Customer] ON [Customer].ID = [ChangeOfService].CustomerID").
		Joins("LEFT JOIN [Product] ON [Product].ID = [ChangeOfService].ProductID").
		Joins("LEFT JOIN [Vessel] ON [Vessel].ID = [ChangeOfService].VesselID").
		Preload("Customer").Preload("Product").Preload("Vessel")
	q, err = filterDocStatus(c, q, "[ChangeOfService].Status")
	if err != nil {
		return err
	}
	q = response.ApplyLike(q, search,
		"[ChangeOfService].DocumentNumber", "[Customer].Name", "[Customer].Code",
		"[Vessel].Name", "[Vessel].Code", "[Product].Name", "[Product].Code",
	)
	return response.ServeList(c, response.ListOpts[models.ChangeOfService]{
		Query: q, Search: search,
		DateColumn:  "[ChangeOfService].EffectiveDate",
		DefaultSort: "[ChangeOfService].EffectiveDate",
		TieBreak:    "[ChangeOfService].ID",
		Sort: map[string]string{
			"documentNumber": "[ChangeOfService].DocumentNumber",
			"effectiveDate":  "[ChangeOfService].EffectiveDate",
			"status":         "[ChangeOfService].Status",
			"customer":       "[Customer].Name",
			"vessel":         "[Vessel].Name",
			"vesselDate":     "[ChangeOfService].VesselDate",
		},
		Sheet: "Change of service", File: "change_of_service",
		Headers: []any{"Document", "Effective", "Customer", "Vessel", "Vessel date", "Delivery", "Status"},
		MapRow: func(r models.ChangeOfService) []any {
			return []any{
				r.DocumentNumber, r.EffectiveDate.Format("2006-01-02"), r.Customer.Name,
				r.Vessel.Name, r.VesselDate.Format("2006-01-02"),
				string(r.FromCollection) + " → " + string(r.ToCollection), string(r.Status),
			}
		},
	})
}

func (h handler) listCOSParcels(c fiber.Ctx) error {
	cid, err := lookupID[models.Customer](h.db.WithContext(c.Context()), c.Query("customerId"))
	if err != nil {
		return response.BadRequest(c, "customer is required")
	}
	var rows []models.ReceiptDetail
	err = h.db.WithContext(c.Context()).
		Preload("Receipt.Vessel").Preload("Receipt.Product").
		Joins("JOIN [Receipt] ON [Receipt].ID = [ReceiptDetail].ReceiptID").
		Where("[ReceiptDetail].CustomerID = ? AND [ReceiptDetail].IsArchived = 0", cid).
		Where("[Receipt].Status = ?", types.ReceiptApproved).
		Order("[Receipt].VesselDate DESC, [ReceiptDetail].ID").
		Limit(300).
		Find(&rows).Error
	if err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	out := make([]cosParcel, 0, len(rows))
	for _, d := range rows {
		out = append(out, cosParcel{
			ID:               d.UID,
			ReceiptNumber:    d.Receipt.DocumentNumber,
			VesselID:         d.Receipt.Vessel.UID,
			VesselName:       d.Receipt.Vessel.Name,
			VesselDate:       d.Receipt.VesselDate,
			ProductID:        d.Receipt.Product.UID,
			ProductCode:      d.Receipt.Product.Code,
			ProductName:      d.Receipt.Product.Name,
			CollectionMethod: string(d.CollectionMethod),
			ContractTypeCode: string(d.ContractTypeCode),
			Quantity:         d.ReceivedQuantity(),
		})
	}
	return response.OkDetail(c, out)
}

func (h handler) getCOS(c fiber.Ctx) error {
	row, err := h.loadCOS(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "change of service not found")
	}
	return response.OkDetail(c, row)
}

func (h handler) createCOS(c fiber.Ctx) error {
	var in changeOfServiceSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row, err := h.buildCOS(c, in)
	if err != nil {
		return err
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "cos", "COS")
		if err != nil {
			return err
		}
		row.DocumentNumber = n
		return tx.Omit("Customer", "Product", "Vessel", "ReceiptDetail", "CreatedBy").Create(&row).Error
	})
	if err != nil {
		return writeErr(c, err, "could not create change of service")
	}
	recordAudit(c, types.ActionCreate, row.UID, types.ChangeOfServiceContent,
		"change of service "+row.DocumentNumber+" created", nil, row)
	return h.reloadCOS(c, row.UID)
}

func (h handler) updateCOS(c fiber.Ctx) error {
	row, err := h.loadCOS(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "change of service not found")
	}
	if !editable(row.Status) {
		return response.Conflict(c, "only a draft or returned change of service can be edited")
	}
	var in changeOfServiceSchema
	if err := bindBody(c, &in); err != nil {
		return err
	}
	next, err := h.buildCOS(c, in)
	if err != nil {
		return err
	}
	before := row
	row.EffectiveDate = next.EffectiveDate
	row.CustomerID = next.CustomerID
	row.ReceiptDetailID = next.ReceiptDetailID
	row.VesselID = next.VesselID
	row.VesselDate = next.VesselDate
	row.ProductID = next.ProductID
	row.FromCollection = next.FromCollection
	row.ToCollection = next.ToCollection
	row.Notes = next.Notes
	if err := h.db.WithContext(c.Context()).Omit("Customer", "Product", "Vessel", "ReceiptDetail", "CreatedBy").Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update change of service")
	}
	recordAudit(c, types.ActionUpdate, row.UID, types.ChangeOfServiceContent,
		"change of service "+row.DocumentNumber+" updated", before, row)
	return h.reloadCOS(c, row.UID, before)
}

func (h handler) submitCOS(c fiber.Ctx) error {
	return h.submitDoc(c, &models.ChangeOfService{}, types.ChangeOfServiceContent)
}

func (h handler) cosWorkflow(c fiber.Ctx) error {
	row, err := h.loadCOS(c.Params("uid"))
	if err != nil {
		return notFound(c, err, "change of service not found")
	}
	return h.docWorkflow(c, types.ChangeOfServiceContent, row.ID)
}

func (h handler) attachCOS(c fiber.Ctx) (attachment.Entity, error) {
	row, err := h.loadCOS(c.Params("uid"))
	if err != nil {
		return attachment.Entity{}, err
	}
	return attachment.Entity{ID: row.ID, UID: row.UID, DocumentNumber: row.DocumentNumber, CanMutate: editable(row.Status)}, nil
}

func (h handler) loadCOS(uid string) (models.ChangeOfService, error) {
	var row models.ChangeOfService
	err := models.PreloadCreatedBy(h.db).Preload("Customer").Preload("Product").Preload("Vessel").
		Where("UID = ?", strings.TrimSpace(uid)).First(&row).Error
	if err != nil {
		return row, err
	}
	var d models.ReceiptDetail
	if err := h.db.Select("UID").First(&d, row.ReceiptDetailID).Error; err == nil {
		row.ParcelID = d.UID
	}
	return row, nil
}

func (h handler) reloadCOS(c fiber.Ctx, uid string, before ...any) error {
	row, err := h.loadCOS(uid)
	if err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return respondSaved(c, row, optionalArg(before))
}

func (h handler) buildCOS(c fiber.Ctx, in changeOfServiceSchema) (models.ChangeOfService, error) {
	var zero models.ChangeOfService
	cid, err := lookupID[models.Customer](h.db.WithContext(c.Context()), in.CustomerID)
	if err != nil {
		return zero, response.UnprocessableEntity(c, validation.Errors{
			"customerId": validation.NewError("validation_customer", "customer not found"),
		})
	}
	var d models.ReceiptDetail
	err = h.db.WithContext(c.Context()).Preload("Receipt.Vessel").Preload("Receipt.Product").
		Joins("JOIN [Receipt] ON [Receipt].ID = [ReceiptDetail].ReceiptID").
		Where("[ReceiptDetail].UID = ? AND [ReceiptDetail].CustomerID = ? AND [ReceiptDetail].IsArchived = 0", in.ParcelID, cid).
		Where("[Receipt].Status = ?", types.ReceiptApproved).
		First(&d).Error
	if err != nil {
		return zero, response.UnprocessableEntity(c, validation.Errors{
			"parcelId": validation.NewError("validation_parcel", "approved vessel parcel not found for this customer"),
		})
	}
	to := types.CollectionMethod(in.ToCollection)
	if d.CollectionMethod == to {
		return zero, response.UnprocessableEntity(c, validation.Errors{
			"toCollection": validation.NewError("validation_same", "choose a different delivery method"),
		})
	}
	return models.ChangeOfService{
		EffectiveDate:   parseDate(in.EffectiveDate),
		CustomerID:      cid,
		ReceiptDetailID: d.ID,
		VesselID:        d.Receipt.VesselID,
		VesselDate:      d.Receipt.VesselDate,
		ProductID:       d.Receipt.ProductID,
		FromCollection:  d.CollectionMethod,
		ToCollection:    to,
		Notes:           in.Notes,
		Status:          types.DocDraft,
		CreatedByID:     middleware.GetUserIDFromContext(c),
	}, nil
}

func lookupID[T any](db *gorm.DB, uid string) (uint, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return 0, gorm.ErrRecordNotFound
	}
	var row T
	if err := db.Where("UID = ?", uid).First(&row).Error; err != nil {
		return 0, err
	}
	switch m := any(&row).(type) {
	case *models.Customer:
		return m.ID, nil
	case *models.Product:
		return m.ID, nil
	default:
		return 0, errors.New("unsupported lookup")
	}
}
