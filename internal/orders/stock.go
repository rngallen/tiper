package orders

import (
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ILRPosition is the customer/product book used on the ILR stock section.
//
//	TotalBalance  = final receipts (net of loaded/pumped)
//	HoldQty       = financial hold
//	FreeQty       = total − hold − approved-not-loaded (ITT, pump-over, ILO trucks)
//	FinalQty      = free − this order's requested quantity
type ILRPosition struct {
	TotalBalance decimal.Decimal
	HoldQty      decimal.Decimal
	FreeQty      decimal.Decimal
	FinalQty     decimal.Decimal
	CommittedQty decimal.Decimal
}

func (s *Service) ILRPosition(customerID, productID, excludeRequestID uint, thisOrder decimal.Decimal) (ILRPosition, error) {
	var out ILRPosition
	if s == nil || s.ledger == nil {
		return out, nil
	}
	final, _, hold, _, err := s.Snapshot(customerID, productID)
	if err != nil {
		return out, err
	}
	committed, err := committedNotLoaded(s.db, customerID, productID, excludeRequestID)
	if err != nil {
		return out, err
	}
	available := final.Sub(hold)
	if available.IsNegative() {
		available = decimal.Zero
	}
	free := available.Sub(committed)
	if free.IsNegative() {
		free = decimal.Zero
	}
	after := free.Sub(thisOrder)
	if after.IsNegative() {
		after = decimal.Zero
	}
	return ILRPosition{
		TotalBalance: final,
		HoldQty:      hold,
		FreeQty:      free,
		FinalQty:     after,
		CommittedQty: committed,
	}, nil
}

func committedNotLoaded(db *gorm.DB, customerID, productID, excludeRequestID uint) (decimal.Decimal, error) {
	sum := decimal.Zero
	if db == nil || customerID == 0 || productID == 0 {
		return sum, nil
	}

	type agg struct{ Qty decimal.Decimal }

	var ilo agg
	q := db.Table("GantryLoadingLine AS l").
		Select("COALESCE(SUM(l.RequestedQty - l.LoadedQty), 0) AS Qty").
		Joins("JOIN GantryLoadingRequest r ON r.ID = l.RequestID").
		Where("r.CustomerID = ? AND l.ProductID = ? AND l.Amended = 0 AND l.IsActive = 1", customerID, productID).
		Where("l.Status IN ?", iloCommittedStatuses)
	if excludeRequestID > 0 {
		q = q.Where("r.ID <> ?", excludeRequestID)
	}
	if err := q.Scan(&ilo).Error; err != nil {
		return sum, err
	}
	sum = sum.Add(ilo.Qty)

	var pdo agg
	if err := db.Table("PumpOverRequest AS p").
		Select("COALESCE(SUM(p.Quantity), 0) AS Qty").
		Where("p.CustomerID = ? AND p.ProductID = ?", customerID, productID).
		Where("p.Status IN ?", []types.OrderStatus{types.OrderSubmitted, types.OrderApproved, types.OrderOpen}).
		Where(`NOT EXISTS (
			SELECT 1 FROM PumpOverReport r
			WHERE r.RequestID = p.ID AND r.Status IN ?)`,
			[]types.DocumentStatus{types.DocApproved, types.DocPosted}).
		Scan(&pdo).Error; err != nil {
		return sum, err
	}
	sum = sum.Add(pdo.Qty)

	var itt agg
	if err := db.Table("IttTransfer").
		Select("COALESCE(SUM(Quantity), 0) AS Qty").
		Where("FromCustomerID = ? AND ProductID = ?", customerID, productID).
		Where("Status IN ?", []types.DocumentStatus{types.DocSubmitted, types.DocApproved}).
		Scan(&itt).Error; err != nil {
		return sum, err
	}
	sum = sum.Add(itt.Qty)

	return sum, nil
}
