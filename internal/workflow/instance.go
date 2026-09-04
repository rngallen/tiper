package workflow

import (
	"context"
	"errors"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// InstanceView is one running (or completed) approval plus its process graph
// so Approvals can show the same diagram as Access → Workflows.
type InstanceView struct {
	ID               string            `json:"id"`
	No               string            `json:"no"`
	Summary          string            `json:"summary"`
	Status           string            `json:"status"`
	CurrentNodeName  string            `json:"currentNodeName"`
	VisitedNodeNames []string          `json:"visitedNodeNames"`
	DocContentType   types.ContentType `json:"docContentType"`
	DocUID           string            `json:"docId,omitempty"`
	DocumentNumber   string            `json:"documentNumber,omitempty"`
	AttachmentCount  int               `json:"attachmentCount"`
	Document         *DocumentFacts    `json:"document,omitempty"`
	Process          ProcessView       `json:"process"`
	History          []EventView       `json:"history"`
}

// GetInstanceView loads the instance, its process definition, and the trail.
func (e *Engine) GetInstanceView(ctx context.Context, instanceUID string) (*InstanceView, error) {
	var inst models.ProcessInstance
	if err := e.db.WithContext(ctx).
		Preload("Process").
		Where("UID = ?", instanceUID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}
	if inst.Process == nil {
		return nil, ErrProcessNotFound
	}

	proc, err := e.GetProcess(ctx, inst.Process.UID)
	if err != nil {
		return nil, err
	}
	history, err := e.Progress(ctx, instanceUID)
	if err != nil {
		return nil, err
	}

	current := ""
	if inst.CurNodeID != nil {
		var node models.Node
		if err := e.db.WithContext(ctx).Select("Name").First(&node, *inst.CurNodeID).Error; err == nil {
			current = node.Name
		}
	}

	seen := map[string]bool{}
	visited := make([]string, 0, len(history))
	for _, ev := range history {
		for _, name := range []string{ev.OldNode, ev.NewNode} {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			visited = append(visited, name)
		}
	}

	view := &InstanceView{
		ID:               inst.UID,
		No:               inst.No,
		Summary:          inst.Summary,
		Status:           string(inst.Status),
		CurrentNodeName:  current,
		VisitedNodeNames: visited,
		Process:          *proc,
		History:          history,
	}
	e.fillDocument(ctx, &inst, view)
	return view, nil
}
