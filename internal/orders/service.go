package orders

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/ewura"
	"dfms/internal/integrations"
	"dfms/internal/inventory"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	engine   *wfengine.Engine
	ledger   *inventory.Service
	almaRoot string
}

func NewService(db *gorm.DB, engine *wfengine.Engine, ledger *inventory.Service) *Service {
	return &Service{db: db, engine: engine, ledger: ledger}
}

func (s *Service) SetAlmaRoot(root string) {
	if s != nil {
		s.almaRoot = strings.TrimSpace(root)
	}
}

func (s *Service) Snapshot(customerID, productID uint) (final, provision, hold, free decimal.Decimal, err error) {
	bals, err := s.ledger.Balances(customerID, productID)
	if err != nil {
		return
	}
	for _, b := range bals {
		final = final.Add(b.FinalQty)
		provision = provision.Add(b.ProvisionQty)
		hold = hold.Add(b.HoldQty)
		free = free.Add(b.FreeQty)
	}
	return
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

func IloExpiryDays() int {
	return integrations.LiveOrders().IloExpiryDays
}

func (s *Service) OnGLRApproved(tx *gorm.DB, req *models.GantryLoadingRequest) error {
	if err := tx.Model(req).Update("Status", types.OrderApproved).Error; err != nil {
		return err
	}
	if _, err := s.ledger.ReserveTx(tx, req.CustomerID, req.ProductID, req.Quantity, req.DocumentNumber, nil); err != nil {
		return err
	}
	if req.ByProductID != nil && req.ByProductQuantity.IsPositive() {
		if _, err := s.ledger.ReserveTx(tx, req.CustomerID, *req.ByProductID, req.ByProductQuantity, req.DocumentNumber, nil); err != nil {
			return err
		}
	}
	var lines []models.GantryLoadingLine
	if err := tx.Where("RequestID = ?", req.ID).Find(&lines).Error; err != nil {
		return err
	}
	for i := range lines {
		if lines[i].DocumentNumber == "" {
			n, err := models.AssignDocumentNumber(tx, "ilo", "ILO")
			if err != nil {
				return err
			}
			lines[i].DocumentNumber = n
		}
		if err := tx.Model(&lines[i]).Updates(map[string]any{
			"Status":         types.OrderOpen,
			"DocumentNumber": lines[i].DocumentNumber,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CompleteLoadingLine(uid string, loaded decimal.Decimal) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var line models.GantryLoadingLine
		if err := tx.Where("UID = ?", uid).First(&line).Error; err != nil {
			return err
		}
		switch line.Status {
		case types.OrderOpen, types.OrderApproved, types.OrderRunning, types.OrderInProgress:
		default:
			return fmt.Errorf("line %s is not open for loading", line.DocumentNumber)
		}
		var req models.GantryLoadingRequest
		if err := tx.Preload("Vessels.Vessel").Preload("Customer").Preload("Product").Preload("StockStatus").
			First(&req, line.RequestID).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&line).Updates(map[string]any{
			"LoadedQty": loaded,
			"LoadedAt":  now,
			"Status":    types.OrderLoaded,
		}).Error; err != nil {
			return err
		}
		ev := &models.InventoryEventLog{
			MessageID:    "glo-" + line.UID,
			EventType:    types.InvEventLoading,
			OccurredAt:   now,
			Quantity:     loaded,
			OrderNumber:  line.DocumentNumber,
			CustomerCode: "",
		}
		cust, product, status := req.Customer, req.Product, req.StockStatus
		vessel := models.Vessel{}
		vd := now
		if len(req.Vessels) > 0 {
			vessel = req.Vessels[0].Vessel
			vd = req.Vessels[0].VesselDate
		} else if err := tx.Order("ID ASC").First(&vessel).Error; err != nil {
			return err
		}
		ev.CustomerCode = cust.Code
		ev.ProductCode = product.Code
		ev.VesselCode = vessel.Code
		if err := tx.Create(ev).Error; err != nil {
			return err
		}
		if err := s.ledger.PostDelivery(tx, ev, cust, product, vessel, vd, status); err != nil {
			return err
		}
		var remaining int64
		if err := tx.Model(&models.GantryLoadingLine{}).
			Where("RequestID = ? AND Amended = 0 AND IsActive = 1 AND Status IN ?",
				req.ID, iloCommittedStatuses).
			Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			if err := s.ledger.ReleaseReservationTx(tx, req.DocumentNumber); err != nil {
				return err
			}
			return tx.Model(&req).Update("Status", types.OrderCompleted).Error
		}
		return nil
	})
}

func (s *Service) OnPumpOverApproved(tx *gorm.DB, req *models.PumpOverRequest) error {
	if err := tx.Model(req).Update("Status", types.OrderApproved).Error; err != nil {
		return err
	}
	_, err := s.ledger.ReserveTx(tx, req.CustomerID, req.ProductID, req.Quantity, req.DocumentNumber, nil)
	return err
}

func (s *Service) OnPumpOverReportApproved(tx *gorm.DB, rep *models.PumpOverReport) error {
	if err := tx.Model(rep).Update("Status", types.OrderCompleted).Error; err != nil {
		return err
	}
	var req models.PumpOverRequest
	if err := tx.Preload("Vessels.Vessel").Preload("Customer").Preload("Product").Preload("StockStatus").
		First(&req, rep.RequestID).Error; err != nil {
		return err
	}
	cust, product, status := req.Customer, req.Product, req.StockStatus
	vessel := models.Vessel{}
	vd := time.Now()
	if len(req.Vessels) > 0 {
		vessel = req.Vessels[0].Vessel
		vd = req.Vessels[0].VesselDate
	} else if err := tx.Order("ID ASC").First(&vessel).Error; err != nil {
		return err
	}
	ev := &models.InventoryEventLog{
		MessageID:    "pdo-" + rep.UID,
		EventType:    types.InvEventPumpOver,
		OccurredAt:   time.Now(),
		CustomerCode: cust.Code,
		ProductCode:  product.Code,
		VesselCode:   vessel.Code,
		Quantity:     rep.ActualDelivered,
		OrderNumber:  req.DocumentNumber,
	}
	if err := tx.Create(ev).Error; err != nil {
		return err
	}
	if err := s.ledger.PostDelivery(tx, ev, cust, product, vessel, vd, status); err != nil {
		return err
	}
	if err := s.ledger.ReleaseReservationTx(tx, req.DocumentNumber); err != nil {
		return err
	}
	if err := tx.Model(&req).Update("Status", types.OrderCompleted).Error; err != nil {
		return err
	}
	ewura.EnqueuePumpOver(tx, &req, rep.ActualDelivered)
	return nil
}
