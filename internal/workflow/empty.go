package workflow

import (
	"fmt"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// openOrSkip leaves the instance on node, or walks agree transitions while the
// landing step is running and has no operators (no role, or the role has no
// users). Completes if it reaches Approved. Assign a role under Access →
// Workflows (and users to that role) to start waiting on a step — no code change.
func (e *Engine) openOrSkip(
	tx *gorm.DB,
	proc *models.Process,
	inst *models.ProcessInstance,
	node *models.Node,
	notes *[]notifyCall,
) ([]models.User, *models.Node, error) {
	seen := map[uint]struct{}{}
	for node != nil {
		if _, dup := seen[node.ID]; dup {
			return nil, nil, fmt.Errorf("workflow: empty-step cycle in process %q", proc.Code)
		}
		seen[node.ID] = struct{}{}

		inst.CurNodeID = &node.ID
		if node.Status == types.NodeCompleted {
			if err := e.finishApproved(tx, proc, inst, node, notes); err != nil {
				return nil, nil, err
			}
			return nil, node, nil
		}
		if node.Status != types.NodeRunning {
			if err := e.saveInstance(tx, inst); err != nil {
				return nil, nil, err
			}
			return nil, node, nil
		}

		ops, err := resolveOperators(tx, node.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(ops) > 0 {
			if err := e.saveInstance(tx, inst); err != nil {
				return nil, nil, err
			}
			if err := genTasks(tx, inst.ID, node.ID, ops); err != nil {
				return nil, nil, err
			}
			if err := queueNotify(notes, tx, inst, node, ops); err != nil {
				return nil, nil, err
			}
			return ops, node, nil
		}

		trans := transitionFrom(proc, node.ID, types.ActAgree)
		if trans == nil || trans.OutputNodeID == nil {
			return nil, nil, ErrNoTransition
		}
		next := nodeByID(proc, trans.OutputNodeID)
		if next == nil {
			return nil, nil, ErrNodeNotFound
		}
		if err := tx.Create(&models.Event{
			InstanceID: inst.ID,
			ActType:    types.EventAgree,
			ActName:    "Skipped (no approvers)",
			OldNodeID:  &node.ID,
			NewNodeID:  &next.ID,
		}).Error; err != nil {
			return nil, nil, err
		}
		node = next
	}
	return nil, nil, ErrNodeNotFound
}

func (e *Engine) finishApproved(tx *gorm.DB, proc *models.Process, inst *models.ProcessInstance, node *models.Node, notes *[]notifyCall) error {
	inst.Status = types.NodeCompleted
	inst.EndedAt = now()
	if err := e.saveInstance(tx, inst); err != nil {
		return err
	}
	if err := tx.Create(&models.Event{
		InstanceID: inst.ID,
		ActType:    types.EventComplete,
		ActName:    "Completed",
		NewNodeID:  &node.ID,
	}).Error; err != nil {
		return err
	}
	if h := e.hookFor(inst.DocContentType); h != nil {
		if err := h.OnComplete(tx, inst); err != nil {
			return err
		}
	}
	return e.queueInfoNotify(notes, tx, proc, inst, node, types.NotifyComplete, nil)
}
