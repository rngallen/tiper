package workflow

import (
	"context"
	"errors"
	"fmt"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// InitiateParams describes a new workflow instance for a document.
type InitiateParams struct {
	ContentType types.ContentType // document content type (must match a process)
	ObjectID    uint              // primary key of the document
	No          string            // human reference (e.g. ILR or receipt number)
	Summary     string
	CreatedByID uint
}

type startedInstance struct {
	Inst      *models.ProcessInstance
	Node      *models.Node
	Operators []models.User
	Covers    []substituteCover
	Proc      *models.Process
	Notes     []notifyCall
}

// Initiate starts an approval workflow for a document. It creates an instance,
// advances from draft to the first running node, and generates tasks when that
// step has operators. Empty steps (no role, or role with no users) are skipped
// through to the next occupied step or Approved. Notifications fire after commit.
func (e *Engine) Initiate(ctx context.Context, p InitiateParams) (*models.ProcessInstance, error) {
	started, err := e.initiate(ctx, nil, p)
	if err != nil {
		return nil, err
	}
	e.fireAfterInitiate(started)
	return started.Inst, nil
}

// InitiateInTx starts a workflow on an existing transaction (same connection as
// the caller's writes). Notifications are not sent — call the returned func
// after the outer transaction commits so assignees are not emailed for work
// that later rolls back.
func (e *Engine) InitiateInTx(tx *gorm.DB, p InitiateParams) (*models.ProcessInstance, func(), error) {
	if tx == nil {
		return nil, nil, errors.New("workflow: InitiateInTx requires a transaction")
	}
	started, err := e.initiate(txContext(tx), tx, p)
	if err != nil {
		return nil, nil, err
	}
	notify := func() {
		e.fireAfterInitiate(started)
	}
	return started.Inst, notify, nil
}

func (e *Engine) initiate(ctx context.Context, outer *gorm.DB, p InitiateParams) (*startedInstance, error) {
	var started *startedInstance
	run := func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&models.ProcessInstance{}).
			Where("DocContentType = ? AND ObjectID = ? AND Status IN ?",
				p.ContentType, p.ObjectID, []types.NodeStatus{types.NodeRunning, types.NodeDraft}).
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrInstanceActive
		}

		proc, err := e.loadProcessForContent(tx, p.ContentType)
		if err != nil {
			return err
		}

		draft := nodeByStatus(proc, types.NodeDraft)
		if draft == nil {
			return fmt.Errorf("process %q has no draft node", proc.Code)
		}
		agree := transitionFrom(proc, draft.ID, types.ActAgree)
		if agree == nil || agree.OutputNodeID == nil {
			return fmt.Errorf("process %q has no agree transition from draft", proc.Code)
		}
		firstNode := nodeByID(proc, agree.OutputNodeID)
		if firstNode == nil {
			return ErrNodeNotFound
		}

		creator := p.CreatedByID
		inst := models.ProcessInstance{
			No:             p.No,
			ProcessID:      proc.ID,
			DocContentType: p.ContentType,
			ObjectID:       p.ObjectID,
			CreatedByID:    &creator,
			CurNodeID:      &firstNode.ID,
			Status:         types.NodeRunning,
			Summary:        p.Summary,
			SubmittedAt:    now(),
		}
		if err := tx.Create(&inst).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.Event{
			InstanceID: inst.ID,
			UserID:     &creator,
			ActType:    types.EventInitiate,
			ActName:    "Initiated",
			NewNodeID:  &firstNode.ID,
		}).Error; err != nil {
			return err
		}

		var notes []notifyCall
		operators, landed, err := e.openOrSkip(tx, proc, &inst, firstNode, &notes)
		if err != nil {
			return err
		}
		if landed == nil {
			return ErrNodeNotFound
		}
		covers, err := withSubstitutes(tx, proc.ID, landed.ID, operators)
		if err != nil {
			return err
		}

		started = &startedInstance{
			Inst: &inst, Node: landed, Operators: operators, Covers: covers, Proc: proc, Notes: notes,
		}
		return nil
	}

	if err := e.runTx(ctx, outer, run); err != nil {
		return nil, err
	}
	return started, nil
}

func (e *Engine) fireAfterInitiate(started *startedInstance) {
	if started == nil {
		return
	}
	fireTaskNotifications(e.notifier, started.Inst, started.Node, started.Operators, started.Covers)
	e.fireInfoNotifications(started.Proc, started.Inst, started.Node, types.NotifySubmit, excludeFromTaskMail(started.Operators, started.Covers))
	for i := range started.Notes {
		n := started.Notes[i]
		instN, node := n.inst, n.node
		if n.infoEvent != "" {
			e.notifier.NotifyInfo(&instN, &node, n.users, n.infoEvent)
		}
	}
}

// runTx runs fn on outer when provided (caller's transaction); otherwise it
// opens a new transaction on the engine DB.
func (e *Engine) runTx(ctx context.Context, outer *gorm.DB, fn func(tx *gorm.DB) error) error {
	if outer != nil {
		return fn(outer)
	}
	return e.db.WithContext(ctx).Transaction(fn)
}

func txContext(tx *gorm.DB) context.Context {
	if tx != nil && tx.Statement != nil && tx.Statement.Context != nil {
		return tx.Statement.Context
	}
	return context.Background()
}

// InstanceForDocument returns the latest instance for a document, if any.
// Uses Take (not First+Order): GORM First always appends ORDER BY PK, and MSSQL
// rejects duplicate columns in the ORDER BY list.
func (e *Engine) InstanceForDocument(ctx context.Context, ct types.ContentType, objectID uint) (*models.ProcessInstance, error) {
	var inst models.ProcessInstance
	err := e.db.WithContext(ctx).
		Where("DocContentType = ? AND ObjectID = ?", ct, objectID).
		Order("ID DESC").
		Take(&inst).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}
	return &inst, nil
}
