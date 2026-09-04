package workflow

import (
	"context"
	"errors"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// DocumentFacts is the inquiry strip shown to an approver without requiring
// the source module's read permission.
type DocumentFacts struct {
	ID             string `json:"id,omitempty"`
	DocumentNumber string `json:"documentNumber,omitempty"`
	Status         string `json:"status,omitempty"`
	Description    string `json:"description,omitempty"`
	FromCurrency   string `json:"fromCurrency,omitempty"`
	ToCurrency     string `json:"toCurrency,omitempty"`
	Rate           string `json:"rate,omitempty"`
	EffectiveFrom  string `json:"effectiveFrom,omitempty"`
	CustomerName   string `json:"customerName,omitempty"`
	CustomerCode   string `json:"customerCode,omitempty"`
	Product        string `json:"product,omitempty"`
	Quantity       string `json:"quantity,omitempty"`
	Vessel         string `json:"vessel,omitempty"`
	BatchNumber    string `json:"batchNumber,omitempty"`
	Amount         string `json:"amount,omitempty"`
	CurrencyCode   string `json:"currencyCode,omitempty"`
}

func (e *Engine) fillDocument(ctx context.Context, inst *models.ProcessInstance, view *InstanceView) {
	if inst == nil || view == nil {
		return
	}
	view.DocContentType = inst.DocContentType
	facts := LoadDocumentFacts(e.db.WithContext(ctx), inst.DocContentType, inst.ObjectID)
	view.Document = facts
	if facts != nil {
		view.DocUID = facts.ID
		view.DocumentNumber = facts.DocumentNumber
	}
	var n int64
	_ = e.db.WithContext(ctx).Model(&models.Attachment{}).
		Where("EntityType = ? AND EntityID = ?", inst.DocContentType, inst.ObjectID).
		Count(&n).Error
	view.AttachmentCount = int(n)
}

// LoadDocumentFacts reads operator-facing fields for mail and the review modal.
func LoadDocumentFacts(db *gorm.DB, ct types.ContentType, objectID uint) *DocumentFacts {
	if db == nil || objectID == 0 {
		return nil
	}
	type ident struct {
		UID            string
		DocumentNumber string
		Status         string
		Description    string
		FromCurrency   string
		ToCurrency     string
		Rate           string
		EffectiveFrom  string
		CustomerName   string
		CustomerCode   string
		Product        string
		Quantity       string
		Vessel         string
		BatchNumber    string
		Amount         string
		CurrencyCode   string
	}
	var row ident
	scan := func(q *gorm.DB) {
		_ = q.Scan(&row).Error
	}
	switch ct {
	case types.ReceiptContent:
		scan(db.Raw(`
			SELECT r.UID, r.DocumentNumber, r.Status, v.Name AS Vessel,
				CASE WHEN p.ID IS NULL THEN '' ELSE LTRIM(RTRIM(p.Code + ' — ' + p.Name)) END AS Product,
				CONVERT(varchar(40), r.TankQuantity) AS Quantity
			FROM Receipt r
			LEFT JOIN Vessel v ON v.ID = r.VesselID
			LEFT JOIN Product p ON p.ID = r.ProductID
			WHERE r.ID = ?`, objectID))
	case types.BillingRunContent:
		scan(db.Raw(`
			SELECT br.UID, br.DocumentNumber, br.Status, CONVERT(varchar(40), br.Amount) AS Amount,
				br.CurrencyCode, cust.Name AS CustomerName, cust.Code AS CustomerCode
			FROM BillingRun br LEFT JOIN Customer cust ON cust.ID = br.CustomerID WHERE br.ID = ?`, objectID))
	case types.ChangeOfServiceContent:
		scan(db.Raw(`
			SELECT cos.UID, cos.DocumentNumber, cos.Status, cust.Name AS CustomerName, cust.Code AS CustomerCode
			FROM ChangeOfService cos LEFT JOIN Customer cust ON cust.ID = cos.CustomerID WHERE cos.ID = ?`, objectID))
	case types.ExchangeRateContent:
		scan(db.Raw(`
			SELECT UID, Status, FromCurrency, ToCurrency, CONVERT(varchar(40), Rate) AS Rate,
				CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom,
				FromCurrency + '/' + ToCurrency + ' · ' + CONVERT(varchar(10), EffectiveFrom, 23) AS DocumentNumber
			FROM ExchangeRate WHERE ID = ?`, objectID))
	case types.BillingProfileContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description, CurrencyCode,
			CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom FROM FcfFeeBatch WHERE ID = ?`, objectID))
	case types.VariableFeeBatchContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description, CurrencyCode,
			CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom FROM VariableFeeBatch WHERE ID = ?`, objectID))
	case types.KojFeeBatchContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description, CurrencyCode,
			CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom FROM KojFeeBatch WHERE ID = ?`, objectID))
	case types.TbsFeeBatchContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description, CurrencyCode,
			CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom FROM TbsFeeBatch WHERE ID = ?`, objectID))
	case types.MiLossBatchContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description,
			CONVERT(varchar(10), EffectiveFrom, 23) AS EffectiveFrom FROM MiLossBatch WHERE ID = ?`, objectID))
	case types.GantryLoadingRequestContent:
		scan(db.Raw(`
			SELECT glr.UID, glr.DocumentNumber, glr.Status, glr.Description, glr.BatchNumber,
				CONVERT(varchar(40), glr.Quantity) AS Quantity,
				cust.Name AS CustomerName, cust.Code AS CustomerCode,
				CASE WHEN p.ID IS NULL THEN '' ELSE LTRIM(RTRIM(p.Code + ' — ' + p.Name)) END AS Product
			FROM GantryLoadingRequest glr
			LEFT JOIN Customer cust ON cust.ID = glr.CustomerID
			LEFT JOIN Product p ON p.ID = glr.ProductID
			WHERE glr.ID = ?`, objectID))
	case types.CompartmentalizationContent:
		scan(db.Raw(`
			SELECT UID, DocumentNumber, Status, CustomerName, CustomerCode,
				ProductName AS Product, CONVERT(varchar(40), RequestedQty) AS Quantity
			FROM GantryCompartmentalization WHERE ID = ?`, objectID))
	case types.PumpOverRequestContent:
		scan(db.Raw(`
			SELECT pdo.UID, pdo.DocumentNumber, pdo.Status, pdo.Notes AS Description,
				CONVERT(varchar(40), pdo.Quantity) AS Quantity,
				cust.Name AS CustomerName, cust.Code AS CustomerCode,
				CASE WHEN p.ID IS NULL THEN '' ELSE LTRIM(RTRIM(p.Code + ' — ' + p.Name)) END AS Product
			FROM PumpOverRequest pdo
			LEFT JOIN Customer cust ON cust.ID = pdo.CustomerID
			LEFT JOIN Product p ON p.ID = pdo.ProductID
			WHERE pdo.ID = ?`, objectID))
	case types.PumpOverReportContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status FROM PumpOverReport WHERE ID = ?`, objectID))
	case types.IttTransferContent:
		scan(db.Raw(`
			SELECT itt.UID, itt.DocumentNumber, itt.Status,
				CONVERT(varchar(40), itt.Quantity) AS Quantity,
				cust.Name AS CustomerName, cust.Code AS CustomerCode,
				CASE WHEN p.ID IS NULL THEN '' ELSE LTRIM(RTRIM(p.Code + ' — ' + p.Name)) END AS Product
			FROM IttTransfer itt
			LEFT JOIN Customer cust ON cust.ID = itt.FromCustomerID
			LEFT JOIN Product p ON p.ID = itt.ProductID
			WHERE itt.ID = ?`, objectID))
	case types.ZerolizationContent:
		scan(db.Raw(`
			SELECT z.UID, z.DocumentNumber, z.Status, cust.Name AS CustomerName, cust.Code AS CustomerCode,
				CASE WHEN p.ID IS NULL THEN '' ELSE LTRIM(RTRIM(p.Code + ' — ' + p.Name)) END AS Product
			FROM ZerolizationTransfer z
			LEFT JOIN Customer cust ON cust.ID = z.CustomerID
			LEFT JOIN Product p ON p.ID = z.ProductID
			WHERE z.ID = ?`, objectID))
	case types.FinancialHoldContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Description FROM FinancialHoldRelease WHERE ID = ?`, objectID))
	case types.OrderAmendmentContent:
		scan(db.Raw(`SELECT UID, DocumentNumber, Status, Notes AS Description,
			CONVERT(varchar(40), RequestedQty) AS Quantity FROM OrderAmendment WHERE ID = ?`, objectID))
	default:
		return nil
	}
	if row.UID == "" && row.DocumentNumber == "" {
		return nil
	}
	return &DocumentFacts{
		ID: row.UID, DocumentNumber: row.DocumentNumber, Status: row.Status,
		Description: row.Description, FromCurrency: row.FromCurrency, ToCurrency: row.ToCurrency,
		Rate: row.Rate, EffectiveFrom: row.EffectiveFrom, CustomerName: row.CustomerName,
		CustomerCode: row.CustomerCode, Product: row.Product, Quantity: row.Quantity,
		Vessel: row.Vessel, BatchNumber: row.BatchNumber, Amount: row.Amount, CurrencyCode: row.CurrencyCode,
	}
}

// ListInstanceAttachments returns files on the document behind a workflow instance.
func (e *Engine) ListInstanceAttachments(ctx context.Context, instanceUID string) ([]models.Attachment, error) {
	var inst models.ProcessInstance
	if err := e.db.WithContext(ctx).Select("ID", "ObjectID", "DocContentType").
		Where("UID = ?", instanceUID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}
	var rows []models.Attachment
	if err := e.db.WithContext(ctx).
		Where("EntityType = ? AND EntityID = ?", inst.DocContentType, inst.ObjectID).
		Order("CreatedAt ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []models.Attachment{}
	}
	return rows, nil
}

// GetInstanceAttachment loads one file owned by the instance's document.
func (e *Engine) GetInstanceAttachment(ctx context.Context, instanceUID, attachUID string) (*models.ProcessInstance, *models.Attachment, error) {
	var inst models.ProcessInstance
	if err := e.db.WithContext(ctx).Select("ID", "ObjectID", "DocContentType").
		Where("UID = ?", instanceUID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInstanceNotFound
		}
		return nil, nil, err
	}
	var row models.Attachment
	if err := e.db.WithContext(ctx).
		Where("UID = ? AND EntityType = ? AND EntityID = ?", attachUID, inst.DocContentType, inst.ObjectID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, gorm.ErrRecordNotFound
		}
		return nil, nil, err
	}
	return &inst, &row, nil
}
