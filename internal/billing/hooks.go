package billing

import (
	"context"

	"dfms/apps/models"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// StatusHook marks a priced document approved or rejected when its workflow ends.
type StatusHook struct {
	apply func(tx *gorm.DB, objectID uint, status types.DocumentStatus) error
}

func NewStatusHook(apply func(tx *gorm.DB, objectID uint, status types.DocumentStatus) error) *StatusHook {
	return &StatusHook{apply: apply}
}

func (h *StatusHook) OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error {
	if h == nil || h.apply == nil || instance == nil {
		return nil
	}
	return h.apply(tx, instance.ObjectID, types.DocApproved)
}

func (h *StatusHook) OnReject(tx *gorm.DB, instance *models.ProcessInstance, total bool) error {
	if h == nil || h.apply == nil || instance == nil {
		return nil
	}
	status := types.DocReturned
	if total {
		status = types.DocRejected
	}
	return h.apply(tx, instance.ObjectID, status)
}

func (h *StatusHook) OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error {
	if h == nil || h.apply == nil || instance == nil {
		return nil
	}
	return h.apply(tx, instance.ObjectID, types.DocSubmitted)
}

// RegisterHooks wires price-batch and billing-run status updates.
func RegisterHooks(engine *wfengine.Engine) {
	if engine == nil {
		return
	}
	engine.RegisterHook(types.BillingRunContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.BillingRun{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.VariableFeeBatchContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.VariableFeeBatch{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.KojFeeBatchContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.KojFeeBatch{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.TbsFeeBatchContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.TbsFeeBatch{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.MiLossBatchContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.MiLossBatch{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.BillingProfileContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.FcfFeeBatch{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.ExchangeRateContent, NewStatusHook(func(tx *gorm.DB, id uint, status types.DocumentStatus) error {
		return tx.Model(&models.ExchangeRate{}).Where("ID = ?", id).Update("Status", status).Error
	}))
	engine.RegisterHook(types.ChangeOfServiceContent, NewStatusHook(ApplyChangeOfService))
}

func (s *Service) DocumentWorkflow(ctx context.Context, ct types.ContentType, objectID uint) (*wfengine.InstanceView, error) {
	if s == nil || s.engine == nil {
		return nil, wfengine.ErrInstanceNotFound
	}
	inst, err := s.engine.InstanceForDocument(ctx, ct, objectID)
	if err != nil {
		return nil, err
	}
	return s.engine.GetInstanceView(ctx, inst.UID)
}

func (s *Service) Initiate(doc types.ContentType, objectID uint, user *models.User, summary, no string) error {
	if s == nil || s.engine == nil || user == nil {
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

func systemUserID(db *gorm.DB) uint {
	var u models.User
	if err := db.Where("IsSuperUser = ?", true).Order("ID ASC").First(&u).Error; err != nil {
		return 1
	}
	return u.ID
}
