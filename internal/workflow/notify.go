package workflow

import (
	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// NotificationSettings is the process-level FYI watcher lists.
type NotificationSettings struct {
	NotifyOnSubmit   []string
	NotifyOnComplete []string
	NotifyOnReject   []string
}

func notifyUsers(tx *gorm.DB, processID uint, event types.NotifyEvent) ([]models.User, error) {
	var users []models.User
	err := tx.Raw(`
		SELECT u.* FROM [User] u
		JOIN [WorkflowNotifyUsers] j ON j.UserID = u.ID
		WHERE j.ProcessID = ? AND j.Event = ? AND u.IsActive = 1`, processID, event).
		Scan(&users).Error
	return users, err
}

func (e *Engine) notifyMembers(ctxTx *gorm.DB, processID uint, event types.NotifyEvent) ([]PoolMember, error) {
	users, err := notifyUsers(ctxTx, processID, event)
	if err != nil {
		return nil, err
	}
	out := make([]PoolMember, 0, len(users))
	for _, u := range users {
		out = append(out, PoolMember{
			ID:        u.UID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		})
	}
	return out, nil
}

// outcomeRecipients returns FYI users plus, for complete/reject, the
// creator (creator mode) or the initiator pool (pool mode). exclude drops
// people who already get a task email (first-step operators on submit).
func (e *Engine) outcomeRecipients(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, event types.NotifyEvent, exclude []models.User) ([]models.User, error) {
	fyi, err := notifyUsers(tx, proc.ID, event)
	if err != nil {
		return nil, err
	}
	var creator *models.User
	if inst != nil && inst.CreatedByID != nil {
		var u models.User
		if err := tx.Where("id = ? AND IsActive = 1", *inst.CreatedByID).First(&u).Error; err == nil {
			creator = &u
		}
	}
	var pool []models.User
	if proc.AmendmentMode == types.AmendPool {
		pool, err = initiatorPoolUsers(tx, proc.ID)
		if err != nil {
			return nil, err
		}
	}
	return mergeOutcomeRecipients(event, proc.AmendmentMode, creator, pool, fyi, exclude), nil
}

func mergeOutcomeRecipients(
	event types.NotifyEvent,
	mode types.AmendmentMode,
	creator *models.User,
	pool, fyi, exclude []models.User,
) []models.User {
	excluded := make(map[uint]struct{}, len(exclude))
	for _, u := range exclude {
		if u.ID != 0 {
			excluded[u.ID] = struct{}{}
		}
	}
	seen := make(map[uint]struct{})
	var users []models.User
	add := func(u models.User) {
		if u.ID == 0 {
			return
		}
		if _, skip := excluded[u.ID]; skip {
			return
		}
		if _, dup := seen[u.ID]; dup {
			return
		}
		seen[u.ID] = struct{}{}
		users = append(users, u)
	}

	switch event {
	case types.NotifyComplete, types.NotifyReject:
		if mode == types.AmendPool {
			for _, u := range pool {
				add(u)
			}
			if creator != nil {
				add(*creator)
			}
		} else if creator != nil {
			add(*creator)
		}
	}
	for _, u := range fyi {
		add(u)
	}
	return users
}

func (e *Engine) queueInfoNotify(notes *[]notifyCall, tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, node *models.Node, event types.NotifyEvent, exclude []models.User) error {
	if notes == nil || inst == nil || node == nil {
		return nil
	}
	users, err := e.outcomeRecipients(tx, proc, inst, event, exclude)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	*notes = append(*notes, notifyCall{
		inst:      *inst,
		node:      *node,
		users:     users,
		infoEvent: event,
	})
	return nil
}

func (e *Engine) fireInfoNotifications(proc *models.Process, inst *models.ProcessInstance, node *models.Node, event types.NotifyEvent, exclude []models.User) {
	if e == nil || e.notifier == nil || inst == nil || node == nil {
		return
	}
	tx := e.db
	if proc == nil {
		loaded, err := e.loadProcessForContent(tx, inst.DocContentType)
		if err != nil {
			logs.Errorf("workflow info notify: load process: %v", err)
			return
		}
		proc = loaded
	}
	users, err := e.outcomeRecipients(tx, proc, inst, event, exclude)
	if err != nil {
		logs.Errorf("workflow info notify: recipients: %v", err)
		return
	}
	if len(users) == 0 {
		return
	}
	e.notifier.NotifyInfo(inst, node, users, event)
}

func excludeFromTaskMail(operators []models.User, covers []substituteCover) []models.User {
	out := make([]models.User, 0, len(operators)+len(covers))
	out = append(out, operators...)
	for _, c := range covers {
		out = append(out, c.Delegate)
	}
	return out
}
