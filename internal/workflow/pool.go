package workflow

import (
	"context"
	"errors"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// PoolMember is a compact user row for initiator-pool APIs.
type PoolMember struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// DiagramEdge is a transition expressed with public node ids for UI graphs.
type DiagramEdge struct {
	From    string `json:"from"`           // input node UID
	To      string `json:"to"`             // output node UID
	ActType string `json:"actType"`        // agree | reject | back | transfer
	Name    string `json:"name,omitempty"` // transition label
	Code    string `json:"code,omitempty"`
}

// ApprovalStepView is one running approval step for the configure UI.
type ApprovalStepView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Sequence   int    `json:"sequence"`
	QuorumMode string `json:"quorumMode"`
	RoleID     uint   `json:"roleId"`
	RoleName   string `json:"roleName,omitempty"`
}

// ProcessView is a process definition plus its initiator pool membership.
type ProcessView struct {
	Process             models.Process     `json:"process"`
	Diagram             []DiagramEdge      `json:"diagram"`
	InitiatorPool       []PoolMember       `json:"initiatorPool"`
	PoolMemberIDs       []string           `json:"poolMemberIds"`
	NotifyOnSubmit      []PoolMember       `json:"notifyOnSubmit"`
	NotifyOnComplete    []PoolMember       `json:"notifyOnComplete"`
	NotifyOnReject      []PoolMember       `json:"notifyOnReject"`
	AmendmentModes      []string           `json:"amendmentModes"`
	OperatorRoleOptions []RoleOption       `json:"operatorRoleOptions"`
	ApprovalSteps       []ApprovalStepView `json:"approvalSteps"`
}

// GetProcess returns one process (nodes + transitions) and its initiator pool.
func (e *Engine) GetProcess(ctx context.Context, processUID string) (*ProcessView, error) {
	var proc models.Process
	if err := e.db.WithContext(ctx).
		Preload("Nodes", func(db *gorm.DB) *gorm.DB { return db.Order("Sequence ASC") }).
		Preload("Nodes.OperatorRoles").
		Preload("Transitions").
		Where("UID = ?", processUID).First(&proc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessNotFound
		}
		return nil, err
	}

	members, err := e.poolMembers(ctx, proc.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.ID)
	}
	roles, err := e.ListOperatorRoles(ctx)
	if err != nil {
		return nil, err
	}
	db := e.db.WithContext(ctx)
	onSubmit, err := e.notifyMembers(db, proc.ID, types.NotifySubmit)
	if err != nil {
		return nil, err
	}
	onComplete, err := e.notifyMembers(db, proc.ID, types.NotifyComplete)
	if err != nil {
		return nil, err
	}
	onReject, err := e.notifyMembers(db, proc.ID, types.NotifyReject)
	if err != nil {
		return nil, err
	}
	stampTransitionNodeUIDs(&proc)
	return &ProcessView{
		Process:          proc,
		Diagram:          buildDiagram(proc),
		InitiatorPool:    members,
		PoolMemberIDs:    ids,
		NotifyOnSubmit:   onSubmit,
		NotifyOnComplete: onComplete,
		NotifyOnReject:   onReject,
		AmendmentModes: []string{
			string(types.AmendCreator),
			string(types.AmendPool),
		},
		OperatorRoleOptions: roles,
		ApprovalSteps:       approvalStepsFromProcess(proc),
	}, nil
}

func approvalStepsFromProcess(proc models.Process) []ApprovalStepView {
	out := make([]ApprovalStepView, 0, len(proc.Nodes))
	for _, n := range proc.Nodes {
		if n.Status != types.NodeRunning {
			continue
		}
		step := ApprovalStepView{
			ID:         n.UID,
			Name:       n.Name,
			Sequence:   n.Sequence,
			QuorumMode: string(n.QuorumMode),
		}
		if step.QuorumMode == "" {
			step.QuorumMode = string(types.QuorumAny)
		}
		if len(n.OperatorRoles) > 0 {
			step.RoleID = n.OperatorRoles[0].ID
			step.RoleName = n.OperatorRoles[0].Name
		}
		out = append(out, step)
	}
	return out
}

// buildDiagram maps active transitions onto public node UIDs for Mermaid/UI.
func buildDiagram(proc models.Process) []DiagramEdge {
	idToUID := make(map[uint]string, len(proc.Nodes))
	for _, n := range proc.Nodes {
		idToUID[n.ID] = n.UID
	}
	out := make([]DiagramEdge, 0, len(proc.Transitions))
	for _, t := range proc.Transitions {
		if !t.IsActive || t.InputNodeID == nil || t.OutputNodeID == nil {
			continue
		}
		from, okFrom := idToUID[*t.InputNodeID]
		to, okTo := idToUID[*t.OutputNodeID]
		if !okFrom || !okTo {
			continue
		}
		out = append(out, DiagramEdge{
			From:    from,
			To:      to,
			ActType: string(t.ActType),
			Name:    t.Name,
			Code:    t.Code,
		})
	}
	return out
}

func stampTransitionNodeUIDs(proc *models.Process) {
	if proc == nil {
		return
	}
	idToUID := make(map[uint]string, len(proc.Nodes))
	for _, n := range proc.Nodes {
		idToUID[n.ID] = n.UID
	}
	for i := range proc.Transitions {
		t := &proc.Transitions[i]
		if t.InputNodeID != nil {
			t.InputNodeUID = idToUID[*t.InputNodeID]
		}
		if t.OutputNodeID != nil {
			t.OutputNodeUID = idToUID[*t.OutputNodeID]
		}
	}
}

// UpdateAmendmentMode sets creator vs pool soft-reject amendment policy.
func (e *Engine) UpdateAmendmentMode(ctx context.Context, processUID string, mode types.AmendmentMode) error {
	switch mode {
	case types.AmendCreator, types.AmendPool:
	default:
		return errors.New("amendmentMode must be creator or pool")
	}
	res := e.db.WithContext(ctx).Model(&models.Process{}).
		Where("UID = ?", processUID).
		Update("AmendmentMode", mode)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrProcessNotFound
	}
	if mode == types.AmendPool {
		var proc models.Process
		if err := e.db.WithContext(ctx).Select("ID").Where("UID = ?", processUID).First(&proc).Error; err != nil {
			return err
		}
		return e.ensurePoolRow(ctx, proc.ID)
	}
	return nil
}

// SetInitiatorPool replaces the process initiator-pool membership with the
// given user UIDs (empty clears the pool).
func (e *Engine) SetInitiatorPool(ctx context.Context, processUID string, userUIDs []string) error {
	var proc models.Process
	if err := e.db.WithContext(ctx).Select("ID").Where("UID = ?", processUID).First(&proc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProcessNotFound
		}
		return err
	}
	if err := e.ensurePoolRow(ctx, proc.ID); err != nil {
		return err
	}

	var pool models.InitiatorPool
	if err := e.db.WithContext(ctx).Where("ProcessID = ?", proc.ID).First(&pool).Error; err != nil {
		return err
	}

	var users []models.User
	if len(userUIDs) > 0 {
		if err := e.db.WithContext(ctx).
			Where("UID IN ? AND IsActive = 1", userUIDs).
			Find(&users).Error; err != nil {
			return err
		}
		if len(users) != len(uniqueStrings(userUIDs)) {
			return errors.New("one or more user ids are invalid or inactive")
		}
	}

	return e.db.WithContext(ctx).Model(&pool).Association("Users").Replace(users)
}

// SetNotifications replaces the process FYI watcher lists.
func (e *Engine) SetNotifications(ctx context.Context, processUID string, settings NotificationSettings) error {
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var proc models.Process
		if err := tx.Select("ID").Where("UID = ?", processUID).First(&proc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProcessNotFound
			}
			return err
		}
		if err := tx.Where("ProcessID = ?", proc.ID).Delete(&models.WorkflowNotifyUser{}).Error; err != nil {
			return err
		}
		groups := []struct {
			event types.NotifyEvent
			uids  []string
		}{
			{types.NotifySubmit, settings.NotifyOnSubmit},
			{types.NotifyComplete, settings.NotifyOnComplete},
			{types.NotifyReject, settings.NotifyOnReject},
		}
		for _, g := range groups {
			users, err := activeUsersByUID(tx, g.uids)
			if err != nil {
				return err
			}
			for _, u := range users {
				row := models.WorkflowNotifyUser{ProcessID: proc.ID, Event: g.event, UserID: u.ID}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func activeUsersByUID(tx *gorm.DB, userUIDs []string) ([]models.User, error) {
	uids := uniqueStrings(userUIDs)
	if len(uids) == 0 {
		return nil, nil
	}
	var users []models.User
	if err := tx.Where("UID IN ? AND IsActive = 1", uids).Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) != len(uids) {
		return nil, errors.New("one or more user ids are invalid or inactive")
	}
	return users, nil
}

func (e *Engine) ensurePoolRow(ctx context.Context, processID uint) error {
	var pool models.InitiatorPool
	return e.db.WithContext(ctx).
		Where(models.InitiatorPool{ProcessID: processID}).
		FirstOrCreate(&pool, models.InitiatorPool{ProcessID: processID}).Error
}

func (e *Engine) poolMembers(ctx context.Context, processID uint) ([]PoolMember, error) {
	users, err := initiatorPoolUsers(e.db.WithContext(ctx), processID)
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

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
