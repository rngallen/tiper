package inventory

import (
	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// ReceiptBiller creates the first-cycle FCF billing run after a receipt posts.
type ReceiptBiller interface {
	RunFirstForReceiptTx(tx *gorm.DB, r *models.Receipt) error
}

// ReceiptHook posts stock when a receipt workflow completes.
type ReceiptHook struct {
	svc    *Service
	biller ReceiptBiller
}

func NewReceiptHook(svc *Service, biller ReceiptBiller) *ReceiptHook {
	return &ReceiptHook{svc: svc, biller: biller}
}

func (h *ReceiptHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.ReceiptContent {
		return nil
	}
	var r models.Receipt
	if err := tx.Preload("Details").First(&r, instance.ObjectID).Error; err != nil {
		return err
	}
	r.Status = types.ReceiptApproved
	if err := tx.Model(&r).Update("Status", types.ReceiptApproved).Error; err != nil {
		return err
	}
	if r.IsFinal && r.ProvisionReceiptID != nil {
		var provision models.Receipt
		if err := tx.Preload("Details").First(&provision, *r.ProvisionReceiptID).Error; err != nil {
			return err
		}
		if err := h.svc.ConvertProvisionToFinal(tx, &provision, &r); err != nil {
			return err
		}
		return h.svc.RecordReceptionFacts(tx, &r)
	}
	if err := h.svc.PostReceipt(tx, &r); err != nil {
		return err
	}
	if err := h.svc.RecordReceptionFacts(tx, &r); err != nil {
		return err
	}
	if h.biller != nil && r.ProvisionReceiptID == nil {
		return h.biller.RunFirstForReceiptTx(tx, &r)
	}
	return nil
}

func (h *ReceiptHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.Receipt{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.ReceiptRejected).Error
}

func (h *ReceiptHook) OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.Receipt{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.ReceiptSubmitted).Error
}

// ZerolHook posts a vessel consolidation after approval.
type ZerolHook struct {
	svc *Service
}

func NewZerolHook(svc *Service) *ZerolHook {
	return &ZerolHook{svc: svc}
}

func (h *ZerolHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.ZerolizationContent {
		return nil
	}
	var z models.ZerolizationTransfer
	if err := tx.First(&z, instance.ObjectID).Error; err != nil {
		return err
	}
	if err := tx.Model(&z).Update("Status", types.DocPosted).Error; err != nil {
		return err
	}
	return h.svc.PostZerolization(tx, &z)
}

func (h *ZerolHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.ZerolizationTransfer{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.DocRejected).Error
}

func (h *ZerolHook) OnResubmit(*gorm.DB, *models.ProcessInstance) error {
	return nil
}

// IttHook posts an ownership transfer after approval.
type IttHook struct {
	svc *Service
}

func NewIttHook(svc *Service) *IttHook {
	return &IttHook{svc: svc}
}

func (h *IttHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.IttTransferContent {
		return nil
	}
	var row models.IttTransfer
	if err := tx.First(&row, instance.ObjectID).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Update("Status", types.DocPosted).Error; err != nil {
		return err
	}
	return h.svc.PostITT(tx, &row)
}

func (h *IttHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.IttTransfer{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.DocRejected).Error
}

func (h *IttHook) OnResubmit(*gorm.DB, *models.ProcessInstance) error {
	return nil
}

// HoldHook posts a financial-hold release after approval.
type HoldHook struct {
	svc *Service
}

func NewHoldHook(svc *Service) *HoldHook {
	return &HoldHook{svc: svc}
}

func (h *HoldHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.FinancialHoldContent {
		return nil
	}
	var row models.FinancialHoldRelease
	if err := tx.Preload("Lines").First(&row, instance.ObjectID).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Update("Status", types.DocPosted).Error; err != nil {
		return err
	}
	return h.svc.PostHoldRelease(tx, &row)
}

func (h *HoldHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.FinancialHoldRelease{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.DocRejected).Error
}

func (h *HoldHook) OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.FinancialHoldRelease{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.DocSubmitted).Error
}
