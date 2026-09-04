// Package workflow implements a configurable approval engine: processes made of
// nodes and transitions, per-user tasks, quorum rules, soft/total rejection,
// transfer to another node, and an initiator pool for corrections.
package workflow

import (
	"errors"
	"sync"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// Common engine errors.
var (
	ErrProcessNotFound  = errors.New("workflow process not found for content type")
	ErrInstanceNotFound = errors.New("workflow instance not found")
	ErrInstanceActive   = errors.New("an approval is already in progress for this document")
	ErrNoTask           = errors.New("you have no pending task on this document")
	ErrNotRunning       = errors.New("workflow is not in a running state")
	ErrNoTransition     = errors.New("no transition available for this action")
	ErrNodeNotFound     = errors.New("target node not found")
	ErrInvalidTarget    = errors.New("destination must be another in-progress step, not Approved or Rejected")
)

// Hook lets a domain module react to workflow outcomes for a document type
// (e.g. mark the document approved on completion, or returned on soft reject).
type Hook interface {
	OnComplete(tx *gorm.DB, instance *models.ProcessInstance) error
	OnReject(tx *gorm.DB, instance *models.ProcessInstance, total bool) error
	// OnResubmit is called when a soft-rejected document leaves the draft /
	// amendment node again (initiator agrees to send it back into approval).
	OnResubmit(tx *gorm.DB, instance *models.ProcessInstance) error
}

// Notifier is invoked when new tasks are generated so assignees can be alerted.
// Production wires notify.WorkflowNotifier (email/SMS); tests may use LoggingNotifier.
//
// NotifyTasks is for the users who own the pending task (role/user operators).
// NotifySubstituteTasks is for delegates covering those operators — same
// document, but the copy must say they are acting as substitute.
// NotifyInfo is informational mail (submit / complete / reject FYI).
// Recipients do not get an inbox task.
type Notifier interface {
	NotifyTasks(instance *models.ProcessInstance, node *models.Node, users []models.User)
	NotifySubstituteTasks(instance *models.ProcessInstance, node *models.Node, delegate, principal models.User)
	NotifyInfo(instance *models.ProcessInstance, node *models.Node, users []models.User, event types.NotifyEvent)
}

// LoggingNotifier logs task notifications.
type LoggingNotifier struct{}

func (LoggingNotifier) NotifyTasks(instance *models.ProcessInstance, node *models.Node, users []models.User) {
	logs.Infof("[workflow] %d task(s) generated at node %q for instance %s", len(users), node.Name, instance.UID)
}

func (LoggingNotifier) NotifySubstituteTasks(instance *models.ProcessInstance, node *models.Node, delegate, principal models.User) {
	logs.Infof("[workflow] substitute notify %s covering %s at node %q for instance %s",
		delegate.Email, principal.Email, node.Name, instance.UID)
}

func (LoggingNotifier) NotifyInfo(instance *models.ProcessInstance, node *models.Node, users []models.User, event types.NotifyEvent) {
	logs.Infof("[workflow] %s info notify %d user(s) for instance %s at %q",
		event, len(users), instance.UID, node.Name)
}

// Engine orchestrates approval state machines.
type Engine struct {
	db       *gorm.DB
	notifier Notifier

	mu    sync.RWMutex
	hooks map[types.ContentType]Hook
}

// New constructs an Engine.
func New(db *gorm.DB, notifier Notifier) *Engine {
	if notifier == nil {
		notifier = LoggingNotifier{}
	}
	return &Engine{db: db, notifier: notifier, hooks: make(map[types.ContentType]Hook)}
}

// DB is the engine's store (attachment download needs the same connection).
func (e *Engine) DB() *gorm.DB { return e.db }

// RegisterHook registers a domain hook for a document content type.
func (e *Engine) RegisterHook(ct types.ContentType, h Hook) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hooks[ct] = h
}

func (e *Engine) hookFor(ct types.ContentType) Hook {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hooks[ct]
}

// loadProcessForContent loads the process (with nodes and transitions) that
// governs the given document content type.
func (e *Engine) loadProcessForContent(tx *gorm.DB, ct types.ContentType) (*models.Process, error) {
	var p models.Process
	err := tx.Preload("Nodes").Preload("Transitions").
		Where("DocContentType = ?", ct).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessNotFound
		}
		return nil, err
	}
	return &p, nil
}

// nodeByID returns the node with the given id from the process's node set.
func nodeByID(p *models.Process, id *uint) *models.Node {
	if id == nil {
		return nil
	}
	for i := range p.Nodes {
		if p.Nodes[i].ID == *id {
			return &p.Nodes[i]
		}
	}
	return nil
}

// nodeByUID returns the node with the given public id.
func nodeByUID(p *models.Process, uid string) *models.Node {
	for i := range p.Nodes {
		if p.Nodes[i].UID == uid {
			return &p.Nodes[i]
		}
	}
	return nil
}

// nodeByStatus returns the first node with the given status.
func nodeByStatus(p *models.Process, status types.NodeStatus) *models.Node {
	for i := range p.Nodes {
		if p.Nodes[i].Status == status {
			return &p.Nodes[i]
		}
	}
	return nil
}

// transitionFrom returns the active transition leaving inputNode for the action.
func transitionFrom(p *models.Process, inputNodeID uint, act types.TransitionAction) *models.Transition {
	for i := range p.Transitions {
		t := &p.Transitions[i]
		if t.IsActive && t.ActType == act && t.InputNodeID != nil && *t.InputNodeID == inputNodeID {
			return t
		}
	}
	return nil
}
