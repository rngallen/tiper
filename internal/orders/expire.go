package orders

import (
	"context"
	"time"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// iloCommittedStatuses still hold stock. Expired is omitted on purpose —
// the midnight job flips status so stock queries never need a date filter.
var iloCommittedStatuses = []types.OrderStatus{
	types.OrderSubmitted, types.OrderApproved, types.OrderOpen,
	types.OrderRunning, types.OrderInProgress,
}

// iloExpirableStatuses are open ILOs that have not reached the gantry.
// inprogress stays — the truck is already in ALMA.
var iloExpirableStatuses = []types.OrderStatus{
	types.OrderSubmitted, types.OrderApproved, types.OrderOpen, types.OrderRunning,
}

// IloExpireBefore is the exclusive upper bound used by the midnight job.
// An ILO whose ExpirationDate falls on calendar day D is expired at 00:00 on D
// (same rule as the compartmentalization gate).
func IloExpireBefore(now time.Time) time.Time {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return today.Add(24 * time.Hour)
}

func (s *Service) RegisterJobs(ctx context.Context, m *jobs.Manager) {
	if s == nil || m == nil {
		return
	}
	m.Register(jobs.IloExpire, func() {
		if ctx.Err() != nil {
			return
		}
		n, err := s.ExpireIlos(time.Now())
		if err != nil {
			logs.Errorf("ilo expire: %v", err)
			return
		}
		if n > 0 {
			logs.Infof("ilo expire: marked %d line(s) expired", n)
		}
	})
}

// ExpireIlos flags past-due ILOs as expired and shrinks the parent ILR
// reservation to remaining committed qty. Stock balance SQL stays on Status.
func (s *Service) ExpireIlos(now time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	cutoff := IloExpireBefore(now)
	var lines []models.GantryLoadingLine
	if err := s.db.Select("ID", "RequestID").
		Where("Amended = 0 AND IsActive = 1").
		Where("ExpirationDate IS NOT NULL AND ExpirationDate < ?", cutoff).
		Where("Status IN ?", iloExpirableStatuses).
		Find(&lines).Error; err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, nil
	}
	ids := make([]uint, 0, len(lines))
	seen := map[uint]struct{}{}
	var requestIDs []uint
	for _, l := range lines {
		ids = append(ids, l.ID)
		if _, ok := seen[l.RequestID]; ok {
			continue
		}
		seen[l.RequestID] = struct{}{}
		requestIDs = append(requestIDs, l.RequestID)
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.GantryLoadingLine{}).Where("ID IN ?", ids).
			Updates(map[string]any{"Status": types.OrderExpired, "IsActive": false}).Error; err != nil {
			return err
		}
		for _, rid := range requestIDs {
			if err := s.syncILRReservation(tx, rid); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (s *Service) syncILRReservation(tx *gorm.DB, requestID uint) error {
	if s.ledger == nil || requestID == 0 {
		return nil
	}
	var req models.GantryLoadingRequest
	if err := tx.Select("ID", "DocumentNumber", "ProductID", "ByProductID").
		First(&req, requestID).Error; err != nil {
		return err
	}
	mainQty, err := remainingCommittedILO(tx, requestID, req.ProductID)
	if err != nil {
		return err
	}
	if err := s.ledger.SetOpenReservationQtyTx(tx, req.DocumentNumber, req.ProductID, mainQty); err != nil {
		return err
	}
	if req.ByProductID == nil || *req.ByProductID == 0 {
		return nil
	}
	byQty, err := remainingCommittedILO(tx, requestID, *req.ByProductID)
	if err != nil {
		return err
	}
	return s.ledger.SetOpenReservationQtyTx(tx, req.DocumentNumber, *req.ByProductID, byQty)
}

func remainingCommittedILO(tx *gorm.DB, requestID, productID uint) (decimal.Decimal, error) {
	var agg struct{ Qty decimal.Decimal }
	err := tx.Table("GantryLoadingLine").
		Select("COALESCE(SUM(RequestedQty - LoadedQty), 0) AS Qty").
		Where("RequestID = ? AND ProductID = ? AND Amended = 0 AND IsActive = 1", requestID, productID).
		Where("Status IN ?", iloCommittedStatuses).
		Scan(&agg).Error
	return agg.Qty, err
}
