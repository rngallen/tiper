package orders

import (
	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

type GLRHook struct{ svc *Service }

func NewGLRHook(svc *Service) *GLRHook { return &GLRHook{svc: svc} }

func (h *GLRHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.GantryLoadingRequestContent {
		return nil
	}
	var req models.GantryLoadingRequest
	if err := tx.Preload("Lines").First(&req, instance.ObjectID).Error; err != nil {
		return err
	}
	return h.svc.OnGLRApproved(tx, &req)
}

func (h *GLRHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.GantryLoadingRequest{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderRejected).Error
}

func (h *GLRHook) OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.GantryLoadingRequest{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderSubmitted).Error
}

type PumpHook struct{ svc *Service }

func NewPumpHook(svc *Service) *PumpHook { return &PumpHook{svc: svc} }

func (h *PumpHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.PumpOverRequestContent {
		return nil
	}
	var req models.PumpOverRequest
	if err := tx.First(&req, instance.ObjectID).Error; err != nil {
		return err
	}
	return h.svc.OnPumpOverApproved(tx, &req)
}

func (h *PumpHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.PumpOverRequest{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderRejected).Error
}

func (h *PumpHook) OnResubmit(*gorm.DB, *models.ProcessInstance) error { return nil }

type PumpReportHook struct{ svc *Service }

func NewPumpReportHook(svc *Service) *PumpReportHook { return &PumpReportHook{svc: svc} }

func (h *PumpReportHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.PumpOverReportContent {
		return nil
	}
	var rep models.PumpOverReport
	if err := tx.First(&rep, instance.ObjectID).Error; err != nil {
		return err
	}
	return h.svc.OnPumpOverReportApproved(tx, &rep)
}

func (h *PumpReportHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.PumpOverReport{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderRejected).Error
}

func (h *PumpReportHook) OnResubmit(*gorm.DB, *models.ProcessInstance) error { return nil }

type CompHook struct{ svc *Service }

func NewCompHook(svc *Service) *CompHook { return &CompHook{svc: svc} }

func (h *CompHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.CompartmentalizationContent {
		return nil
	}
	var row models.Compartmentalization
	if err := tx.First(&row, instance.ObjectID).Error; err != nil {
		return err
	}
	return h.svc.OnCompApproved(tx, &row)
}

func (h *CompHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.Compartmentalization{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderRejected).Error
}

func (h *CompHook) OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil {
		return nil
	}
	return tx.Model(&models.Compartmentalization{}).Where("ID = ?", instance.ObjectID).
		Update("Status", types.OrderSubmitted).Error
}

type AmendHook struct{ svc *Service }

func NewAmendHook(svc *Service) *AmendHook { return &AmendHook{svc: svc} }

func (h *AmendHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if instance == nil || instance.DocContentType != types.OrderAmendmentContent {
		return nil
	}
	var row models.OrderAmendment
	if err := tx.First(&row, instance.ObjectID).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Update("Status", types.OrderApproved).Error; err != nil {
		return err
	}
	return h.svc.ApplyAmendment(tx, &row)
}

func (h *AmendHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, _ bool) error {
	if instance == nil {
		return nil
	}
	var row models.OrderAmendment
	if err := tx.First(&row, instance.ObjectID).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Update("Status", types.OrderRejected).Error; err != nil {
		return err
	}
	return tx.Model(&models.GantryLoadingLine{}).Where("ID = ? AND Status <> ?", row.IloID, types.OrderCancelled).
		Update("Amended", false).Error
}

func (h *AmendHook) OnResubmit(*gorm.DB, *models.ProcessInstance) error { return nil }
