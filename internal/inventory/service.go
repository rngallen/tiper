package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	engine *wfengine.Engine
}

func NewService(db *gorm.DB, engine *wfengine.Engine) *Service {
	return &Service{db: db, engine: engine}
}

type Balance struct {
	CustomerID    uint            `json:"-"`
	CustomerCode  string          `json:"customerCode"`
	CustomerName  string          `json:"customerName"`
	ProductID     uint            `json:"-"`
	ProductCode   string          `json:"productCode"`
	ProductName   string          `json:"productName"`
	VesselID      uint            `json:"-"`
	VesselCode    string          `json:"vesselCode"`
	VesselDate    time.Time       `json:"vesselDate"`
	StockStatusID uint            `json:"-"`
	StatusName    string          `json:"statusName"`
	IsTransit     bool            `json:"isTransit"`
	FinalQty      decimal.Decimal `json:"final"`
	ProvisionQty  decimal.Decimal `json:"provision"`
	HoldQty       decimal.Decimal `json:"financialHold"`
	ReservedQty   decimal.Decimal `json:"reserved"`
	FreeQty       decimal.Decimal `json:"freeToOrder"`
}

func (s *Service) Balances(customerID, productID uint) ([]Balance, error) {
	type row struct {
		CustomerID    uint
		CustomerCode  string
		CustomerName  string
		ProductID     uint
		ProductCode   string
		ProductName   string
		VesselID      uint
		VesselCode    string
		VesselDate    time.Time
		StockStatusID uint
		StatusName    string
		IsTransit     bool
		FinalQty      decimal.Decimal
		ProvisionQty  decimal.Decimal
		HoldQty       decimal.Decimal
	}
	q := s.db.Table("StockBalance AS b").
		Select(`
			b.CustomerID, c.Code AS CustomerCode, c.Name AS CustomerName,
			b.ProductID, p.Code AS ProductCode, p.Name AS ProductName,
			b.VesselID, v.Code AS VesselCode, b.VesselDate,
			b.StockStatusID, st.Name AS StatusName, st.IsTransit,
			SUM(CASE WHEN b.IsProvision = 0 THEN b.Quantity ELSE 0 END) AS FinalQty,
			SUM(CASE WHEN b.IsProvision = 1 THEN b.Quantity ELSE 0 END) AS ProvisionQty,
			SUM(CASE WHEN b.FinancialHold = 1 THEN b.Quantity ELSE 0 END) AS HoldQty
		`).
		Joins("JOIN Customer c ON c.ID = b.CustomerID").
		Joins("JOIN Product p ON p.ID = b.ProductID").
		Joins("JOIN Vessel v ON v.ID = b.VesselID").
		Joins("JOIN StockStatus st ON st.ID = b.StockStatusID")
	if customerID > 0 {
		q = q.Where("b.CustomerID = ?", customerID)
	}
	if productID > 0 {
		q = q.Where("b.ProductID = ?", productID)
	}
	var rows []row
	if err := q.Group("b.CustomerID, c.Code, c.Name, b.ProductID, p.Code, p.Name, b.VesselID, v.Code, b.VesselDate, b.StockStatusID, st.Name, st.IsTransit").
		Having("SUM(b.Quantity) <> 0").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	type rkey struct {
		c, p, v uint
		d       string
		s       uint
	}
	reserved := map[rkey]decimal.Decimal{}
	var holds []models.StockReservation
	rq := s.db.Where("Status = ?", types.ReservationOpen)
	if customerID > 0 {
		rq = rq.Where("CustomerID = ?", customerID)
	}
	if productID > 0 {
		rq = rq.Where("ProductID = ?", productID)
	}
	if err := rq.Find(&holds).Error; err != nil {
		return nil, err
	}
	for _, h := range holds {
		k := rkey{c: h.CustomerID, p: h.ProductID, s: 0}
		if h.VesselID != nil {
			k.v = *h.VesselID
		}
		if h.VesselDate != nil {
			k.d = h.VesselDate.Format("2006-01-02")
		}
		if h.StockStatusID != nil {
			k.s = *h.StockStatusID
		}
		reserved[k] = reserved[k].Add(h.Quantity)
	}

	out := make([]Balance, 0, len(rows))
	for _, r := range rows {
		b := Balance{
			CustomerID: r.CustomerID, CustomerCode: r.CustomerCode, CustomerName: r.CustomerName,
			ProductID: r.ProductID, ProductCode: r.ProductCode, ProductName: r.ProductName,
			VesselID: r.VesselID, VesselCode: r.VesselCode, VesselDate: r.VesselDate,
			StockStatusID: r.StockStatusID, StatusName: r.StatusName, IsTransit: r.IsTransit,
			FinalQty: r.FinalQty, ProvisionQty: r.ProvisionQty, HoldQty: r.HoldQty,
		}
		k := rkey{c: r.CustomerID, p: r.ProductID, v: r.VesselID, d: r.VesselDate.Format("2006-01-02"), s: r.StockStatusID}
		b.ReservedQty = reserved[k]
		b.FreeQty = r.FinalQty.Add(r.ProvisionQty).Sub(r.HoldQty).Sub(b.ReservedQty)
		out = append(out, b)
	}
	return out, nil
}

type QtyOverride struct {
	ID          string          `json:"id"`
	Quantity    decimal.Decimal `json:"quantity"`
	CubicMeter  decimal.Decimal `json:"cubicMeter"`
	MetricTonne decimal.Decimal `json:"metricTonne"`
}

func (s *Service) CreateFinalFromProvision(provisionUID string, userID uint, overrides []QtyOverride) (*models.Receipt, error) {
	var prov models.Receipt
	if err := s.db.Preload("Details").Where("UID = ?", provisionUID).First(&prov).Error; err != nil {
		return nil, err
	}
	if !prov.IsProvision {
		return nil, fmt.Errorf("receipt %s is not a provision", prov.DocumentNumber)
	}
	if prov.Status != types.ReceiptApproved {
		return nil, fmt.Errorf("provision must be approved before conversion")
	}
	var existing int64
	if err := s.db.Model(&models.Receipt{}).Where("ProvisionReceiptID = ?", prov.ID).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, fmt.Errorf("a final receipt already exists for %s", prov.DocumentNumber)
	}
	byUID := map[string]QtyOverride{}
	for _, o := range overrides {
		byUID[o.ID] = o
	}
	final := models.Receipt{
		Date:                  time.Now(),
		VesselDate:            prov.VesselDate,
		BillingEffectiveDate:  prov.BillingEffectiveDate,
		VesselID:              prov.VesselID,
		SupplierID:            prov.SupplierID,
		ProductID:             prov.ProductID,
		RouteCode:             prov.RouteCode,
		TenderCode:            prov.TenderCode,
		ProcurementMethodCode: prov.ProcurementMethodCode,
		ReceiptType:           prov.ReceiptType,
		UsesTiperPipeline:     prov.UsesTiperPipeline,
		Density:               prov.Density,
		TankQuantity:          prov.TankQuantity,
		TankCubicMeter:        prov.TankCubicMeter,
		TankMetricTonne:       prov.TankMetricTonne,
		LineLoss:              prov.LineLoss,
		LineLossCubicMeter:    prov.LineLossCubicMeter,
		LineLossMetricTonne:   prov.LineLossMetricTonne,
		IsProvision:           false,
		IsFinal:               true,
		ProvisionReceiptID:    &prov.ID,
		Status:                types.ReceiptDraft,
		Notes:                 "Converted from " + prov.DocumentNumber,
		CreatedByID:           userID,
	}
	for _, d := range prov.Details {
		nd := models.ReceiptDetail{
			CustomerID:          d.CustomerID,
			StockStatusID:       d.StockStatusID,
			DepotID:             d.DepotID,
			CollectionMethod:    d.CollectionMethod,
			ContractTypeCode:    d.ContractTypeCode,
			PricingNature:       d.PricingNature,
			NextBillingDays:     d.NextBillingDays,
			FinancialHold:       d.FinancialHold,
			HoldQuantity:        d.HoldQuantity,
			Density:             d.Density,
			Quantity:            d.Quantity,
			CubicMeter:          d.CubicMeter,
			MetricTonne:         d.MetricTonne,
			LineLoss:            d.LineLoss,
			LineLossCubicMeter:  d.LineLossCubicMeter,
			LineLossMetricTonne: d.LineLossMetricTonne,
			IsProvision:         false,
		}
		if o, ok := byUID[d.UID]; ok {
			if !o.Quantity.IsZero() {
				nd.Quantity = o.Quantity
			}
			if !o.CubicMeter.IsZero() {
				nd.CubicMeter = o.CubicMeter
			}
			if !o.MetricTonne.IsZero() {
				nd.MetricTonne = o.MetricTonne
			}
		}
		final.Details = append(final.Details, nd)
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "receipt", "RCPT")
		if err != nil {
			return err
		}
		final.DocumentNumber = n
		return tx.Create(&final).Error
	})
	if err != nil {
		return nil, err
	}
	return &final, nil
}

func (s *Service) PostReceipt(tx *gorm.DB, r *models.Receipt) error {
	if tx == nil {
		tx = s.db
	}
	if r.Status != types.ReceiptApproved {
		return fmt.Errorf("receipt %s is not approved", r.DocumentNumber)
	}
	if !r.ReceiptType.PostsStock() {
		return nil
	}
	for _, d := range r.Details {
		if d.IsArchived {
			continue
		}
		var posted int64
		if err := tx.Model(&models.StockMovement{}).
			Where("ReferenceType = ? AND ReferenceID = ? AND TransactionType = ?", "ReceiptDetail", d.ID, types.TxnReceipt).
			Count(&posted).Error; err != nil {
			return err
		}
		if posted > 0 {
			continue
		}
		if d.StatusID() == 0 {
			continue
		}
		mv := models.StockMovement{
			TransactionDate: r.Date,
			TransactionType: types.TxnReceipt,
			CustomerID:      d.CustomerID,
			ProductID:       r.ProductID,
			VesselID:        r.VesselID,
			VesselDate:      r.VesselDate,
			StockStatusID:   d.StatusID(),
			DepotID:         d.DepotID,
			Quantity:        d.ReceivedQuantity(),
			CubicMeter:      d.ReceivedCubicMeter(),
			MetricTonne:     d.ReceivedMetricTonne(),
			IsProvision:     r.IsProvision,
			FinancialHold:   d.FinancialHold,
			ReferenceType:   "ReceiptDetail",
			ReferenceID:     d.ID,
		}
		if err := createMovement(tx, &mv); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ConvertProvisionToFinal(tx *gorm.DB, provision, final *models.Receipt) error {
	if tx == nil {
		tx = s.db
	}
	if !provision.IsProvision || !final.IsFinal {
		return fmt.Errorf("conversion requires a provision source and a final target")
	}
	// Reverse provision quantities so they drop out of balance math.
	var moves []models.StockMovement
	if err := tx.Where("ReferenceType = ? AND TransactionType = ? AND IsProvision = 1", "ReceiptDetail", types.TxnReceipt).
		Where("ReferenceID IN (SELECT ID FROM ReceiptDetail WHERE ReceiptID = ?)", provision.ID).
		Find(&moves).Error; err != nil {
		return err
	}
	for _, m := range moves {
		rev := m
		rev.ID = 0
		rev.UID = ""
		rev.TransactionType = types.TxnReversal
		rev.Quantity = m.Quantity.Neg()
		rev.CubicMeter = m.CubicMeter.Neg()
		rev.MetricTonne = m.MetricTonne.Neg()
		rev.TransactionDate = final.Date
		if err := createMovement(tx, &rev); err != nil {
			return err
		}
	}
	return s.PostReceipt(tx, final)
}

func (s *Service) PostDelivery(tx *gorm.DB, ev *models.InventoryEventLog, customer models.Customer, product models.Product, vessel models.Vessel, vesselDate time.Time, status models.StockStatus) error {
	if tx == nil {
		tx = s.db
	}
	txn := types.TxnLoading
	switch strings.ToLower(string(ev.EventType)) {
	case string(types.InvEventPumpOver), string(types.TxnPumpOver), "pumover":
		txn = types.TxnPumpOver
	case string(types.InvEventITT):
		txn = types.TxnITT
	}
	mv := models.StockMovement{
		TransactionDate: ev.OccurredAt,
		TransactionType: txn,
		CustomerID:      customer.ID,
		ProductID:       product.ID,
		VesselID:        vessel.ID,
		VesselDate:      vesselDate,
		StockStatusID:   status.ID,
		Quantity:        ev.Quantity.Neg(),
		IsProvision:     false,
		FinancialHold:   ev.FinancialHold,
		ReferenceType:   "InventoryEventLog",
		ReferenceID:     ev.ID,
	}
	if err := createMovement(tx, &mv); err != nil {
		return err
	}
	now := time.Now()
	return tx.Model(ev).Updates(map[string]any{"Posted": true, "PostedAt": now, "Error": ""}).Error
}

func (s *Service) PostITT(tx *gorm.DB, itt *models.IttTransfer) error {
	if tx == nil {
		tx = s.db
	}
	asOf := time.Now()
	if err := attachIttBillingParcels(tx, itt, asOf); err != nil {
		return err
	}
	density := receiptDensity(tx, itt)
	litres, cm, mt := ittStockQty(itt, density)
	out := models.StockMovement{
		TransactionDate: asOf,
		TransactionType: types.TxnITT,
		CustomerID:      itt.FromCustomerID,
		ProductID:       itt.ProductID,
		VesselID:        itt.VesselID,
		VesselDate:      itt.VesselDate,
		StockStatusID:   itt.StockStatusID,
		DepotID:         itt.DepotID,
		Quantity:        litres.Neg(),
		CubicMeter:      cm.Neg(),
		MetricTonne:     mt.Neg(),
		FinancialHold:   itt.FinancialHold,
		ReferenceType:   "IttTransfer",
		ReferenceID:     itt.ID,
	}
	in := out
	in.UID = ""
	in.CustomerID = itt.ToCustomerID
	in.Quantity = litres
	in.CubicMeter = cm
	in.MetricTonne = mt
	if err := createMovement(tx, &out); err != nil {
		return err
	}
	return createMovement(tx, &in)
}

func (s *Service) PostHoldRelease(tx *gorm.DB, rel *models.FinancialHoldRelease) error {
	if tx == nil {
		tx = s.db
	}
	if rel == nil {
		return fmt.Errorf("hold release is required")
	}
	if len(rel.Lines) == 0 {
		if err := tx.Preload("Lines").First(rel, rel.ID).Error; err != nil {
			return err
		}
	}
	for _, line := range rel.Lines {
		var posted int64
		if err := tx.Model(&models.StockMovement{}).
			Where("ReferenceType = ? AND ReferenceID = ? AND TransactionType = ?", "FinancialHoldReleaseLine", line.ID, types.TxnHoldRelease).
			Count(&posted).Error; err != nil {
			return err
		}
		if posted > 0 {
			continue
		}
		vd := dayUTC(line.VesselDate)
		var hold models.StockBalance
		err := tx.Where(
			"CustomerID = ? AND ProductID = ? AND VesselID = ? AND VesselDate = ? AND StockStatusID = ? AND FinancialHold = 1 AND IsProvision = 0",
			line.CustomerID, line.ProductID, line.VesselID, vd, line.StockStatusID,
		).Take(&hold).Error
		if err != nil {
			return fmt.Errorf("no financial-hold stock for this parcel")
		}
		if hold.Quantity.LessThan(line.Quantity) {
			return fmt.Errorf("hold stock %s is less than release %s", hold.Quantity, line.Quantity)
		}
		cm, mt := line.CubicMeter, line.MetricTonne
		if cm.IsZero() && !hold.Quantity.IsZero() {
			cm = hold.CubicMeter.Mul(line.Quantity).Div(hold.Quantity)
		}
		if mt.IsZero() && !hold.Quantity.IsZero() {
			mt = hold.MetricTonne.Mul(line.Quantity).Div(hold.Quantity)
		}
		base := models.StockMovement{
			TransactionDate: rel.ReleaseDate,
			TransactionType: types.TxnHoldRelease,
			CustomerID:      line.CustomerID,
			ProductID:       line.ProductID,
			VesselID:        line.VesselID,
			VesselDate:      vd,
			StockStatusID:   line.StockStatusID,
			ReferenceType:   "FinancialHoldReleaseLine",
			ReferenceID:     line.ID,
			IsProvision:     false,
		}
		debit := base
		debit.FinancialHold = true
		debit.Quantity = line.Quantity.Neg()
		debit.CubicMeter = cm.Neg()
		debit.MetricTonne = mt.Neg()
		credit := base
		credit.UID = ""
		credit.FinancialHold = false
		credit.Quantity = line.Quantity
		credit.CubicMeter = cm
		credit.MetricTonne = mt
		if err := createMovement(tx, &debit); err != nil {
			return err
		}
		if err := createMovement(tx, &credit); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) PostZerolization(tx *gorm.DB, z *models.ZerolizationTransfer) error {
	if tx == nil {
		tx = s.db
	}
	from := models.StockMovement{
		TransactionDate: z.TransferDate,
		TransactionType: types.TxnZerolization,
		CustomerID:      z.CustomerID,
		ProductID:       z.ProductID,
		VesselID:        z.FromVesselID,
		VesselDate:      z.FromVesselDate,
		StockStatusID:   z.StockStatusID,
		Quantity:        z.Quantity.Neg(),
		ReferenceType:   "ZerolizationTransfer",
		ReferenceID:     z.ID,
	}
	to := from
	to.UID = ""
	to.VesselID = z.ToVesselID
	to.VesselDate = z.ToVesselDate
	to.Quantity = z.Quantity
	if err := createMovement(tx, &from); err != nil {
		return err
	}
	return createMovement(tx, &to)
}

func (s *Service) Reserve(customerID, productID uint, qty decimal.Decimal, orderNumber string, expires *time.Time) (*models.StockReservation, error) {
	return s.ReserveTx(s.db, customerID, productID, qty, orderNumber, expires)
}

func (s *Service) ReserveTx(tx *gorm.DB, customerID, productID uint, qty decimal.Decimal, orderNumber string, expires *time.Time) (*models.StockReservation, error) {
	if tx == nil {
		tx = s.db
	}
	bals, err := s.Balances(customerID, productID)
	if err != nil {
		return nil, err
	}
	var free decimal.Decimal
	for _, b := range bals {
		free = free.Add(b.FreeQty)
	}
	if free.LessThan(qty) {
		return nil, fmt.Errorf("insufficient free stock: have %s need %s", free, qty)
	}
	row := models.StockReservation{
		CustomerID:  customerID,
		ProductID:   productID,
		Quantity:    qty,
		OrderNumber: orderNumber,
		Status:      types.ReservationOpen,
		ExpiresAt:   expires,
	}
	if err := tx.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) ReleaseReservation(orderNumber string) error {
	return s.ReleaseReservationTx(s.db, orderNumber)
}

func (s *Service) ReleaseReservationTx(tx *gorm.DB, orderNumber string) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Model(&models.StockReservation{}).
		Where("OrderNumber = ? AND Status = ?", orderNumber, types.ReservationOpen).
		Update("Status", types.ReservationReleased).Error
}

// SetOpenReservationQtyTx shrinks (or releases) the open hold for one
// order+product. Qty ≤ 0 releases that product only.
func (s *Service) SetOpenReservationQtyTx(tx *gorm.DB, orderNumber string, productID uint, qty decimal.Decimal) error {
	if tx == nil {
		tx = s.db
	}
	q := tx.Model(&models.StockReservation{}).
		Where("OrderNumber = ? AND ProductID = ? AND Status = ?", orderNumber, productID, types.ReservationOpen)
	if qty.LessThanOrEqual(decimal.Zero) {
		return q.Update("Status", types.ReservationReleased).Error
	}
	return q.Update("Quantity", qty).Error
}

func (s *Service) Initiate(doc types.ContentType, objectID uint, user *models.User, summary, no string) error {
	if s.engine == nil || user == nil {
		return nil
	}
	_, err := s.engine.Initiate(context.Background(), wfengine.InitiateParams{
		ContentType: doc,
		ObjectID:    objectID,
		No:          no,
		Summary:     summary,
		CreatedByID: user.ID,
	})
	return err
}

func FindCustomerByCode(db *gorm.DB, code string) (models.Customer, error) {
	var c models.Customer
	err := db.Where("Code = ?", strings.TrimSpace(code)).First(&c).Error
	return c, err
}

func FindProductByCode(db *gorm.DB, code string) (models.Product, error) {
	var p models.Product
	err := db.Where("Code = ?", strings.TrimSpace(code)).First(&p).Error
	return p, err
}

func FindVesselByCode(db *gorm.DB, code string) (models.Vessel, error) {
	var v models.Vessel
	err := db.Where("Code = ?", strings.TrimSpace(code)).First(&v).Error
	return v, err
}

func FindStatusByCode(db *gorm.DB, code string) (models.StockStatus, error) {
	var st models.StockStatus
	err := db.Where("Code = ?", strings.TrimSpace(code)).First(&st).Error
	return st, err
}

func FindDefaultStatus(db *gorm.DB) (models.StockStatus, error) {
	var st models.StockStatus
	err := db.Where("IsActive = ?", true).Order("ID ASC").First(&st).Error
	return st, err
}
