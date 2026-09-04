package workflow

import (
	"context"
	"errors"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// ActParams describes an approval action taken by a user on a document.
type ActParams struct {
	InstanceUID    string
	UserID         uint
	Action         types.TransitionAction
	Comment        string
	IPAddress      string // client IP captured for the approval audit trail
	TotalRejection bool   // for reject: terminal rejection vs. return-for-correction
	TargetNodeUID  string // for transfer/back: destination node
}

var eventNameByAction = map[types.TransitionAction]string{
	types.ActAgree:    "Approved",
	types.ActReject:   "Rejected",
	types.ActTransfer: "Transferred",
	types.ActBack:     "Sent back",
}

var eventActionByTransition = map[types.TransitionAction]types.EventAction{
	types.ActAgree:    types.EventAgree,
	types.ActReject:   types.EventReject,
	types.ActTransfer: types.EventTransfer,
	types.ActBack:     types.EventBack,
}

type notifyCall struct {
	inst      models.ProcessInstance
	node      models.Node
	users     []models.User
	covers    []substituteCover
	infoEvent types.NotifyEvent
}

func queueNotify(notes *[]notifyCall, tx *gorm.DB, inst *models.ProcessInstance, node *models.Node, users []models.User) error {
	if notes == nil || inst == nil || node == nil || len(users) == 0 {
		return nil
	}
	covers, err := withSubstitutes(tx, inst.ProcessID, node.ID, users)
	if err != nil {
		return err
	}
	*notes = append(*notes, notifyCall{inst: *inst, node: *node, users: users, covers: covers})
	return nil
}

// Act applies an approval action to a running instance and advances/returns the
// workflow accordingly. The whole operation is transactional. Emails fire after
// commit so document lookups see the saved document fields.
func (e *Engine) Act(ctx context.Context, p ActParams) (*models.ProcessInstance, error) {
	inst, notes, err := e.act(ctx, nil, p)
	if err != nil {
		return nil, err
	}
	for i := range notes {
		n := notes[i]
		instN, node := n.inst, n.node
		if n.infoEvent != "" {
			e.notifier.NotifyInfo(&instN, &node, n.users, n.infoEvent)
			continue
		}
		fireTaskNotifications(e.notifier, &instN, &node, n.users, n.covers)
	}
	return inst, nil
}

func (e *Engine) act(ctx context.Context, outer *gorm.DB, p ActParams) (*models.ProcessInstance, []notifyCall, error) {
	var result *models.ProcessInstance
	var notes []notifyCall
	run := func(tx *gorm.DB) error {
		var inst models.ProcessInstance
		if err := tx.Where("uid = ?", p.InstanceUID).First(&inst).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInstanceNotFound
			}
			return err
		}
		if inst.Status != types.NodeRunning || inst.CurNodeID == nil {
			return ErrNotRunning
		}

		proc, err := e.loadProcessForContent(tx, inst.DocContentType)
		if err != nil {
			return err
		}
		current := nodeByID(proc, inst.CurNodeID)
		if current == nil {
			return ErrNodeNotFound
		}

		task, onBehalfOf, err := resolveActableTask(tx, inst.ID, current.ID, p.UserID)
		if err != nil {
			return err
		}

		task.Status = types.TaskCompleted
		task.Comment = p.Comment
		task.ActedAt = now()
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		uid := p.UserID
		ev := models.Event{
			InstanceID:       inst.ID,
			UserID:           &uid,
			OnBehalfOfUserID: onBehalfOf,
			ActType:          eventActionByTransition[p.Action],
			ActName:          eventNameByAction[p.Action],
			OldNodeID:        &current.ID,
			Comment:          p.Comment,
			IPAddress:        p.IPAddress,
			TotalRejection:   p.Action == types.ActReject && p.TotalRejection,
		}

		switch p.Action {
		case types.ActAgree:
			return e.handleAgree(tx, proc, &inst, current, &ev, &result, &notes)
		case types.ActReject:
			return e.handleReject(tx, proc, &inst, current, &ev, p.TotalRejection, &result, &notes)
		case types.ActTransfer, types.ActBack:
			return e.handleMove(tx, proc, &inst, &ev, p.TargetNodeUID, &result, &notes)
		default:
			return ErrNoTransition
		}
	}

	if err := e.runTx(ctx, outer, run); err != nil {
		return nil, nil, err
	}
	return result, notes, nil
}

func (e *Engine) handleAgree(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, current *models.Node, ev *models.Event, result **models.ProcessInstance, notes *[]notifyCall) error {
	// Quorum "all": wait until every assigned operator has acted.
	// Quorum "any" (default): one agree decides the step.
	if current.QuorumMode == types.QuorumAll {
		remaining, err := pendingTaskCount(tx, inst.ID, current.ID)
		if err != nil {
			return err
		}
		if remaining > 0 {
			if err := tx.Create(ev).Error; err != nil {
				return err
			}
			*result = inst
			return nil
		}
	}

	if err := skipPendingAtNode(tx, inst.ID, current.ID); err != nil {
		return err
	}

	var out *models.Node
	// Soft-reject resubmit: return to the rejecting step (ResumeNodeID) when set.
	if current.Status == types.NodeDraft && inst.ResumeNodeID != nil {
		out = nodeByID(proc, inst.ResumeNodeID)
	}
	if out == nil {
		trans := transitionFrom(proc, current.ID, types.ActAgree)
		if trans == nil || trans.OutputNodeID == nil {
			return ErrNoTransition
		}
		out = nodeByID(proc, trans.OutputNodeID)
	}
	if out == nil {
		return ErrNodeNotFound
	}

	if current.Status == types.NodeDraft {
		ev.ActName = "Resubmitted"
	}
	ev.NewNodeID = &out.ID
	if err := tx.Create(ev).Error; err != nil {
		return err
	}

	// Soft-rejected documents sit on the draft/Initiator node. Leaving it via
	// agree means the initiator resubmitted after correction.
	if current.Status == types.NodeDraft {
		if h := e.hookFor(inst.DocContentType); h != nil {
			if err := h.OnResubmit(tx, inst); err != nil {
				return err
			}
		}
		inst.ResumeNodeID = nil
	}

	inst.Version++
	if _, _, err := e.openOrSkip(tx, proc, inst, out, notes); err != nil {
		return err
	}
	*result = inst
	return nil
}

func (e *Engine) handleReject(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, current *models.Node, ev *models.Event, total bool, result **models.ProcessInstance, notes *[]notifyCall) error {
	// One reject decides the step under both quorum modes — remaining
	// operators (and the principal if a substitute acted) must not keep a
	// pending inbox item on this node.
	if err := skipPendingAtNode(tx, inst.ID, current.ID); err != nil {
		return err
	}

	if total {
		rejected := nodeByStatus(proc, types.NodeRejected)
		if rejected != nil {
			ev.NewNodeID = &rejected.ID
			inst.CurNodeID = &rejected.ID
		}
		ev.ActName = "Rejected"
		if err := tx.Create(ev).Error; err != nil {
			return err
		}
		inst.Status = types.NodeRejected
		inst.EndedAt = now()
		inst.Version++
		if err := e.saveInstance(tx, inst); err != nil {
			return err
		}
		if h := e.hookFor(inst.DocContentType); h != nil {
			if err := h.OnReject(tx, inst, true); err != nil {
				return err
			}
		}
		node := current
		if rejected != nil {
			node = rejected
		}
		if err := e.queueInfoNotify(notes, tx, proc, inst, node, types.NotifyReject, nil); err != nil {
			return err
		}
		*result = inst
		return nil
	}

	// Soft rejection: return to correction node; remember rejecting step for resubmit.
	target := nodeByID(proc, proc.RejectReturnNodeID)
	if target == nil {
		target = nodeByStatus(proc, types.NodeDraft)
	}
	if target == nil {
		return ErrNodeNotFound
	}
	ev.ActName = "Returned"
	ev.NewNodeID = &target.ID
	if err := tx.Create(ev).Error; err != nil {
		return err
	}

	resumeID := current.ID
	inst.ResumeNodeID = &resumeID
	inst.CurNodeID = &target.ID
	inst.Version++
	if err := e.saveInstance(tx, inst); err != nil {
		return err
	}

	amenders, err := e.resolveAmenders(tx, proc, inst)
	if err != nil {
		return err
	}
	if err := genTasks(tx, inst.ID, target.ID, amenders); err != nil {
		return err
	}
	if err := queueNotify(notes, tx, inst, target, amenders); err != nil {
		return err
	}
	if h := e.hookFor(inst.DocContentType); h != nil {
		if err := h.OnReject(tx, inst, false); err != nil {
			return err
		}
	}
	*result = inst
	return nil
}

func (e *Engine) handleMove(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, ev *models.Event, targetUID string, result **models.ProcessInstance, notes *[]notifyCall) error {
	target := nodeByUID(proc, targetUID)
	if target == nil {
		return ErrNodeNotFound
	}
	if target.Status == types.NodeCompleted || target.Status == types.NodeRejected {
		return ErrInvalidTarget
	}
	if inst.CurNodeID != nil && target.ID == *inst.CurNodeID {
		return ErrInvalidTarget
	}
	if inst.CurNodeID != nil {
		if err := skipPendingAtNode(tx, inst.ID, *inst.CurNodeID); err != nil {
			return err
		}
	}
	ev.NewNodeID = &target.ID
	if err := tx.Create(ev).Error; err != nil {
		return err
	}

	inst.CurNodeID = &target.ID
	inst.Version++
	if err := e.saveInstance(tx, inst); err != nil {
		return err
	}
	operators, err := resolveOperators(tx, target.ID)
	if err != nil {
		return err
	}
	if err := genTasks(tx, inst.ID, target.ID, operators); err != nil {
		return err
	}
	if err := queueNotify(notes, tx, inst, target, operators); err != nil {
		return err
	}
	*result = inst
	return nil
}

// resolveAmenders returns who may correct a soft-rejected document.
func (e *Engine) resolveAmenders(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance) ([]models.User, error) {
	if proc.AmendmentMode == types.AmendPool {
		users, err := initiatorPoolUsers(tx, proc.ID)
		if err != nil {
			return nil, err
		}
		if len(users) > 0 {
			return users, nil
		}
	}
	// default: original initiator
	if inst.CreatedByID == nil {
		return nil, nil
	}
	var u models.User
	if err := tx.Where("id = ? AND IsActive = 1", *inst.CreatedByID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return []models.User{u}, nil
}

func (e *Engine) saveInstance(tx *gorm.DB, inst *models.ProcessInstance) error {
	return tx.Model(&models.ProcessInstance{}).Where("id = ?", inst.ID).
		Select("CurNodeID", "Status", "Version", "EndedAt", "ResumeNodeID").
		Updates(map[string]any{
			"CurNodeID":    inst.CurNodeID,
			"Status":       inst.Status,
			"Version":      inst.Version,
			"EndedAt":      inst.EndedAt,
			"ResumeNodeID": inst.ResumeNodeID,
		}).Error
}
