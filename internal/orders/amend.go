package orders

import (
	"fmt"
	"time"

	"dfms/apps/models"
	"dfms/internal/alma"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AmendmentInput struct {
	Kind            types.AmendmentKind
	LineUID         string
	RequestedQty    decimal.Decimal
	ProductID       uint
	ExpirationDate  *time.Time
	Destination     string
	District        string
	TruckPlate      string
	TransporterName string
	DriverName      string
	Notes           string
}

func (s *Service) CreateAmendment(in AmendmentInput, user *models.User) (*models.OrderAmendment, error) {
	var line models.GantryLoadingLine
	if err := s.db.Where("UID = ?", in.LineUID).First(&line).Error; err != nil {
		return nil, err
	}
	if line.Amended || !line.IsActive {
		return nil, fmt.Errorf("line %s is already amended or inactive", line.DocumentNumber)
	}
	switch line.Status {
	case types.OrderOpen, types.OrderRunning, types.OrderInProgress:
	default:
		return nil, fmt.Errorf("line %s cannot be amended in status %s", line.DocumentNumber, line.Status)
	}
	if err := validateAmendment(in, line); err != nil {
		return nil, err
	}
	var row models.OrderAmendment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "amend", "AMD")
		if err != nil {
			return err
		}
		row = models.OrderAmendment{
			DocumentNumber:  n,
			Kind:            in.Kind,
			IloID:           line.ID,
			RequestedQty:    in.RequestedQty,
			ExpirationDate:  in.ExpirationDate,
			Destination:     firstNonEmpty(in.Destination, line.Destination),
			District:        firstNonEmpty(in.District, line.District),
			TruckPlate:      firstNonEmpty(in.TruckPlate, line.TruckPlate),
			TransporterName: firstNonEmpty(in.TransporterName, line.TransporterName),
			DriverName:      firstNonEmpty(in.DriverName, line.DriverName),
			Notes:           in.Notes,
			Status:          types.OrderDraft,
			CreatedByID:     user.ID,
		}
		if in.ProductID != 0 {
			row.ProductID = &in.ProductID
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := parkLineForAmendment(tx, &line); err != nil {
			return err
		}
		if types.ImmediateAmendment(in.Kind) {
			row.Status = types.OrderApproved
			if err := tx.Model(&row).Update("Status", types.OrderApproved).Error; err != nil {
				return err
			}
			return s.ApplyAmendment(tx, &row)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !types.ImmediateAmendment(in.Kind) {
		if err := s.Initiate(types.OrderAmendmentContent, row.ID, user, row.DocumentNumber, row.DocumentNumber); err != nil {
			return nil, err
		}
		_ = s.db.Model(&row).Update("Status", types.OrderSubmitted)
	}
	return &row, nil
}

func validateAmendment(in AmendmentInput, line models.GantryLoadingLine) error {
	switch in.Kind {
	case types.AmendNormal, types.AmendQtyDecrease:
		if in.RequestedQty.GreaterThan(line.RequestedQty) {
			return fmt.Errorf("this amendment cannot increase quantity")
		}
	case types.AmendQtyIncrease:
		if in.RequestedQty.LessThan(line.RequestedQty) {
			return fmt.Errorf("quantity increase must be greater than the original")
		}
	case types.AmendExtend:
		if in.ExpirationDate == nil {
			return fmt.Errorf("new expiration date is required")
		}
	case types.AmendCancel, types.AmendBatchCancel, types.AmendProduct:
	default:
		return fmt.Errorf("unknown amendment kind %s", in.Kind)
	}
	return nil
}

func parkLineForAmendment(tx *gorm.DB, line *models.GantryLoadingLine) error {
	if err := tx.Model(line).Updates(map[string]any{"Amended": true}).Error; err != nil {
		return err
	}
	var comps []models.Compartmentalization
	if err := tx.Where("IloID = ? AND IsActive = 1", line.ID).Find(&comps).Error; err != nil {
		return err
	}
	for i := range comps {
		if err := tx.Model(&comps[i]).Updates(map[string]any{"Amended": true, "IsActive": false, "Status": types.OrderCancelled}).Error; err != nil {
			return err
		}
		if comps[i].BadgeID != nil {
			if err := tx.Model(&models.RfidBadge{}).Where("ID = ?", *comps[i].BadgeID).
				Update("IsAvailable", true).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ApplyAmendment(tx *gorm.DB, row *models.OrderAmendment) error {
	var line models.GantryLoadingLine
	if err := tx.First(&line, row.IloID).Error; err != nil {
		return err
	}
	switch row.Kind {
	case types.AmendCancel, types.AmendBatchCancel:
		if line.SentToAlma && s.almaRoot != "" {
			var comp models.Compartmentalization
			if err := tx.Preload("Lines.Product").Where("IloID = ?", line.ID).Order("ID DESC").First(&comp).Error; err == nil {
				if order, err := s.almaOrder(tx, &comp, true); err == nil {
					_, _ = alma.WriteOrder(tx, alma.Paths{Root: s.almaRoot}, order, alma.NewFileName(time.Now()))
				}
			}
		}
		return tx.Model(&line).Updates(map[string]any{
			"Status":   types.OrderCancelled,
			"IsActive": false,
			"Amended":  true,
		}).Error
	case types.AmendExtend:
		return tx.Model(&line).Updates(map[string]any{
			"ExpirationDate": row.ExpirationDate,
			"Amended":        false,
			"Status":         types.OrderOpen,
		}).Error
	default:
		n, err := models.AssignDocumentNumber(tx, "ilo", "ILO")
		if err != nil {
			return err
		}
		next := line
		next.ID = 0
		next.UID = ""
		next.DocumentNumber = n
		next.RequestedQty = row.RequestedQty
		next.LoadedQty = decimal.Zero
		next.LoadedAt = nil
		next.Amended = false
		next.IsActive = true
		next.SentToAlma = false
		next.AlmaFileName = ""
		next.AlmaSentAt = nil
		next.Status = types.OrderOpen
		next.Destination = firstNonEmpty(row.Destination, line.Destination)
		next.District = firstNonEmpty(row.District, line.District)
		next.TruckPlate = firstNonEmpty(row.TruckPlate, line.TruckPlate)
		next.TransporterName = firstNonEmpty(row.TransporterName, line.TransporterName)
		next.DriverName = firstNonEmpty(row.DriverName, line.DriverName)
		if row.ExpirationDate != nil {
			next.ExpirationDate = row.ExpirationDate
		}
		if err := tx.Model(&line).Updates(map[string]any{"IsActive": false, "Amended": true}).Error; err != nil {
			return err
		}
		return tx.Create(&next).Error
	}
}
