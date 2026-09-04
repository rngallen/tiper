package inventory

import (
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// InboundEvent is a gantry loading, pump-over, or ITT completion recorded in DFMS.
type InboundEvent struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	OccurredAt    string `json:"occurredAt"`
	CustomerCode  string `json:"customerCode"`
	ProductCode   string `json:"productCode"`
	VesselCode    string `json:"vesselCode"`
	VesselDate    string `json:"vesselDate"`
	Quantity      string `json:"quantity"`
	StatusCode    string `json:"statusCode"`
	FinancialHold bool   `json:"financialHold"`
	OrderNumber   string `json:"orderNumber"`
}

// IngestEvent posts a delivery against the parcel ledger. This is the native
// DFMS path — gantry and pump-over live in this system, not a second app.
func (s *Service) IngestEvent(ev InboundEvent) (*models.InventoryEventLog, error) {
	if strings.TrimSpace(ev.Type) == "" {
		return nil, fmt.Errorf("event type is required")
	}
	if strings.TrimSpace(ev.CustomerCode) == "" || strings.TrimSpace(ev.ProductCode) == "" {
		return nil, fmt.Errorf("customerCode and productCode are required")
	}
	log := models.InventoryEventLog{
		MessageID:     ev.ID,
		EventType:     types.InventoryEventType(ev.Type),
		CustomerCode:  ev.CustomerCode,
		ProductCode:   ev.ProductCode,
		VesselCode:    ev.VesselCode,
		StatusCode:    ev.StatusCode,
		FinancialHold: ev.FinancialHold,
		OrderNumber:   ev.OrderNumber,
	}
	if log.MessageID == "" {
		log.MessageID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if t, err := time.Parse(time.RFC3339, ev.OccurredAt); err == nil {
		log.OccurredAt = t
	} else {
		log.OccurredAt = time.Now()
	}
	if ev.VesselDate != "" {
		if t, err := time.Parse("2006-01-02", ev.VesselDate); err == nil {
			log.VesselDate = &t
		}
	}
	if q, err := decimal.NewFromString(strings.TrimSpace(ev.Quantity)); err == nil {
		log.Quantity = q
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(models.InventoryEventLog{MessageID: log.MessageID}).FirstOrCreate(&log).Error; err != nil {
			return err
		}
		if log.Posted {
			return nil
		}
		customer, err := FindCustomerByCode(tx, ev.CustomerCode)
		if err != nil {
			return err
		}
		product, err := FindProductByCode(tx, ev.ProductCode)
		if err != nil {
			return err
		}
		vessel, err := FindVesselByCode(tx, ev.VesselCode)
		if err != nil {
			return err
		}
		status, err := FindStatusByCode(tx, ev.StatusCode)
		if err != nil {
			status, err = FindDefaultStatus(tx)
			if err != nil {
				return err
			}
		}
		vd := log.OccurredAt
		if log.VesselDate != nil {
			vd = *log.VesselDate
		}
		return s.PostDelivery(tx, &log, customer, product, vessel, vd, status)
	})
	if err != nil {
		return nil, err
	}
	return &log, nil
}
