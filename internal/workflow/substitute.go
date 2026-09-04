package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/permissions"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

var (
	ErrSubstituteSelf       = errors.New("cannot delegate to yourself")
	ErrSubstituteNotAllowed = errors.New("only approval-step operators may set a substitute")
	ErrSubstituteDelegate   = errors.New("delegate must be an active user with Approvals access (workflow.tasks)")
	ErrSubstituteProcess    = errors.New("select a workflow process for this substitute")
)

// SubstituteAssignment is one process-scoped delegation row.
type SubstituteAssignment struct {
	ID            string     `json:"id,omitempty"`
	DelegateID    string     `json:"delegateId,omitempty"`
	DelegateEmail string     `json:"delegateEmail,omitempty"`
	DelegateName  string     `json:"delegateName,omitempty"`
	ProcessID     string     `json:"processId,omitempty"`
	ProcessName   string     `json:"processName,omitempty"`
	NodeID        string     `json:"nodeId,omitempty"`
	NodeName      string     `json:"nodeName,omitempty"`
	StartsAt      time.Time  `json:"startsAt"`
	EndsAt        *time.Time `json:"endsAt,omitempty"`
	Effective     bool       `json:"effective"`
}

// SubstituteView is the profile / API shape for the principal's delegations.
type SubstituteView struct {
	CanDelegate      bool                   `json:"canDelegate"`
	DelegatableSteps []DelegatableStep      `json:"delegatableSteps,omitempty"`
	Assignments      []SubstituteAssignment `json:"assignments,omitempty"`
}

// DelegatableStep is a running workflow node where the user is an operator.
type DelegatableStep struct {
	NodeID      string `json:"nodeId"`
	NodeName    string `json:"nodeName"`
	ProcessID   string `json:"processId"`
	ProcessName string `json:"processName"`
}

// GetSubstitute returns the principal's process-scoped substitutes plus whether
// they may configure any.
func (e *Engine) GetSubstitute(ctx context.Context, principalID uint) (*SubstituteView, error) {
	can, steps, err := e.delegatableSteps(ctx, principalID)
	if err != nil {
		return nil, err
	}
	view := &SubstituteView{CanDelegate: can, DelegatableSteps: steps, Assignments: []SubstituteAssignment{}}

	var rows []models.ApprovalSubstitute
	err = e.db.WithContext(ctx).
		Preload("Delegate").
		Preload("Node").
		Preload("Process").
		Where("PrincipalUserID = ?", principalID).
		Order("ID ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for i := range rows {
		view.Assignments = append(view.Assignments, assignmentFromRow(&rows[i], now))
	}
	return view, nil
}

func assignmentFromRow(row *models.ApprovalSubstitute, now time.Time) SubstituteAssignment {
	a := SubstituteAssignment{
		ID:        row.UID,
		StartsAt:  row.StartsAt,
		EndsAt:    row.EndsAt,
		Effective: substituteInWindow(row, now),
	}
	if row.Delegate != nil {
		a.DelegateID = row.Delegate.UID
		a.DelegateEmail = row.Delegate.Email
		a.DelegateName = strings.TrimSpace(row.Delegate.FirstName + " " + row.Delegate.LastName)
	}
	if row.Process != nil {
		a.ProcessID = row.Process.UID
		a.ProcessName = row.Process.Name
	}
	if row.Node != nil {
		a.NodeID = row.Node.UID
		a.NodeName = row.Node.Name
	}
	return a
}

// SetSubstituteParams configures or clears a principal's substitute for one process.
type SetSubstituteParams struct {
	PrincipalUserID uint
	DelegateUID     string // empty + ProcessUID clears that process; both empty clears all
	ProcessUID      string
	NodeUID         string // optional step scope within the process
	StartsAt        *time.Time
	EndsAt          *time.Time
}

// SetSubstitute upserts one process-scoped substitute, or deletes when
// DelegateUID is empty (one process if ProcessUID is set, otherwise all).
func (e *Engine) SetSubstitute(ctx context.Context, p SetSubstituteParams) (*SubstituteView, error) {
	can, _, err := e.delegatableSteps(ctx, p.PrincipalUserID)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrSubstituteNotAllowed
	}

	db := e.db.WithContext(ctx)

	if strings.TrimSpace(p.DelegateUID) == "" {
		q := db.Where("PrincipalUserID = ?", p.PrincipalUserID)
		if pid := strings.TrimSpace(p.ProcessUID); pid != "" {
			var proc models.Process
			if err := db.Where("UID = ?", pid).First(&proc).Error; err != nil {
				return nil, ErrSubstituteProcess
			}
			q = q.Where("ProcessID = ?", proc.ID)
		}
		if err := q.Delete(&models.ApprovalSubstitute{}).Error; err != nil {
			return nil, err
		}
		return e.GetSubstitute(ctx, p.PrincipalUserID)
	}

	if strings.TrimSpace(p.ProcessUID) == "" {
		return nil, ErrSubstituteProcess
	}

	var proc models.Process
	if err := db.Where("UID = ?", p.ProcessUID).First(&proc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubstituteProcess
		}
		return nil, err
	}

	var delegate models.User
	if err := db.Where("UID = ? AND IsActive = 1 AND IsLocked = 0", p.DelegateUID).First(&delegate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubstituteDelegate
		}
		return nil, err
	}
	if delegate.ID == p.PrincipalUserID {
		return nil, ErrSubstituteSelf
	}
	ok, err := userHasPermission(db, delegate.ID, permissions.WorkflowTasks, permissions.WorkflowManage)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSubstituteDelegate
	}

	var nodeID *uint
	if nid := strings.TrimSpace(p.NodeUID); nid != "" {
		var node models.Node
		if err := db.Where("UID = ? AND ProcessID = ?", nid, proc.ID).First(&node).Error; err != nil {
			return nil, fmt.Errorf("unknown approval step")
		}
		allowed, err := e.userOperatesNode(ctx, p.PrincipalUserID, node.ID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrSubstituteNotAllowed
		}
		nodeID = &node.ID
	} else {
		// Process-wide: principal must operate at least one running step on it.
		operates, err := e.userOperatesProcess(ctx, p.PrincipalUserID, proc.ID)
		if err != nil {
			return nil, err
		}
		if !operates {
			return nil, ErrSubstituteNotAllowed
		}
	}

	starts := time.Now().UTC()
	if p.StartsAt != nil {
		starts = p.StartsAt.UTC()
	}
	var ends *time.Time
	if p.EndsAt != nil {
		t := p.EndsAt.UTC()
		ends = &t
		if !ends.After(starts) {
			return nil, fmt.Errorf("endsAt must be after startsAt")
		}
	}

	if err := db.Where("PrincipalUserID = ? AND ProcessID = ?", p.PrincipalUserID, proc.ID).
		Delete(&models.ApprovalSubstitute{}).Error; err != nil {
		return nil, err
	}
	row := models.ApprovalSubstitute{
		PrincipalUserID: p.PrincipalUserID,
		ProcessID:       proc.ID,
		DelegateUserID:  delegate.ID,
		NodeID:          nodeID,
		StartsAt:        starts,
		EndsAt:          ends,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return e.GetSubstitute(ctx, p.PrincipalUserID)
}

// ClearSubstitute removes one assignment belonging to the principal.
func (e *Engine) ClearSubstitute(ctx context.Context, principalID uint, assignmentUID string) (*SubstituteView, error) {
	can, _, err := e.delegatableSteps(ctx, principalID)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrSubstituteNotAllowed
	}
	uid := strings.TrimSpace(assignmentUID)
	if uid == "" {
		return e.GetSubstitute(ctx, principalID)
	}
	if err := e.db.WithContext(ctx).
		Where("PrincipalUserID = ? AND UID = ?", principalID, uid).
		Delete(&models.ApprovalSubstitute{}).Error; err != nil {
		return nil, err
	}
	return e.GetSubstitute(ctx, principalID)
}

func (e *Engine) delegatableSteps(ctx context.Context, userID uint) (bool, []DelegatableStep, error) {
	type row struct {
		NodeUID     string
		NodeName    string
		ProcessUID  string
		ProcessName string
	}
	var rows []row
	err := e.db.WithContext(ctx).Raw(`
		SELECT DISTINCT n.UID AS NodeUID, n.Name AS NodeName, p.UID AS ProcessUID, p.Name AS ProcessName
		FROM Node n
		JOIN Process p ON p.ID = n.ProcessID
		JOIN NodeOperatorRole nr ON nr.NodeID = n.ID
		JOIN UserRole ur ON ur.RoleID = nr.RoleID
		WHERE ur.UserID = ? AND n.Status = 'running'
		UNION
		SELECT DISTINCT n.UID, n.Name, p.UID, p.Name
		FROM Node n
		JOIN Process p ON p.ID = n.ProcessID
		JOIN NodeOperatorUser nu ON nu.NodeID = n.ID
		WHERE nu.UserID = ? AND n.Status = 'running'
		ORDER BY ProcessName, NodeName`, userID, userID).Scan(&rows).Error
	if err != nil {
		return false, nil, err
	}
	out := make([]DelegatableStep, len(rows))
	for i, r := range rows {
		out[i] = DelegatableStep{
			NodeID:      r.NodeUID,
			NodeName:    r.NodeName,
			ProcessID:   r.ProcessUID,
			ProcessName: r.ProcessName,
		}
	}
	return len(out) > 0, out, nil
}

func (e *Engine) userOperatesNode(ctx context.Context, userID, nodeID uint) (bool, error) {
	var n int64
	err := e.db.WithContext(ctx).Raw(`
		SELECT COUNT(1) FROM (
			SELECT 1 AS n FROM NodeOperatorRole nr
			JOIN UserRole ur ON ur.RoleID = nr.RoleID
			WHERE nr.NodeID = ? AND ur.UserID = ?
			UNION ALL
			SELECT 1 AS n FROM NodeOperatorUser nu
			WHERE nu.NodeID = ? AND nu.UserID = ?
		) x`, nodeID, userID, nodeID, userID).Scan(&n).Error
	return n > 0, err
}

func (e *Engine) userOperatesProcess(ctx context.Context, userID, processID uint) (bool, error) {
	var n int64
	err := e.db.WithContext(ctx).Raw(`
		SELECT COUNT(1) FROM (
			SELECT 1 AS n
			FROM Node n
			JOIN NodeOperatorRole nr ON nr.NodeID = n.ID
			JOIN UserRole ur ON ur.RoleID = nr.RoleID
			WHERE n.ProcessID = ? AND ur.UserID = ? AND n.Status = 'running'
			UNION ALL
			SELECT 1 AS n
			FROM Node n
			JOIN NodeOperatorUser nu ON nu.NodeID = n.ID
			WHERE n.ProcessID = ? AND nu.UserID = ? AND n.Status = 'running'
		) x`, processID, userID, processID, userID).Scan(&n).Error
	return n > 0, err
}

func substituteInWindow(s *models.ApprovalSubstitute, now time.Time) bool {
	if s == nil {
		return false
	}
	if now.Before(s.StartsAt) {
		return false
	}
	if s.EndsAt != nil && now.After(*s.EndsAt) {
		return false
	}
	return true
}

// resolveActableTask finds a pending task the actor may complete: either their
// own, or a principal's task they are currently substituting for.
func resolveActableTask(tx *gorm.DB, instanceID, nodeID, actorID uint) (models.Task, *uint, error) {
	var task models.Task
	err := tx.Where("InstanceID = ? AND NodeID = ? AND UserID = ? AND Status = ?",
		instanceID, nodeID, actorID, types.TaskPending).First(&task).Error
	if err == nil {
		return task, nil, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return task, nil, err
	}

	now := time.Now().UTC()
	err = tx.Raw(`
		SELECT t.* FROM Task t
		JOIN ProcessInstance i ON i.ID = t.InstanceID
		JOIN ApprovalSubstitute s ON s.PrincipalUserID = t.UserID AND s.ProcessID = i.ProcessID
		WHERE t.InstanceID = ? AND t.NodeID = ? AND t.Status = ?
			AND s.DelegateUserID = ?
			AND s.StartsAt <= ? AND (s.EndsAt IS NULL OR s.EndsAt >= ?)
			AND (s.NodeID IS NULL OR s.NodeID = t.NodeID)`,
		instanceID, nodeID, types.TaskPending, actorID, now, now,
	).Take(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return task, nil, ErrNoTask
		}
		return task, nil, err
	}
	if task.UserID == nil {
		return task, nil, ErrNoTask
	}
	principal := *task.UserID
	return task, &principal, nil
}

func userHasPermission(db *gorm.DB, userID uint, codes ...string) (bool, error) {
	if len(codes) == 0 {
		return false, nil
	}
	var held int64
	if err := db.Raw(`
		SELECT COUNT(1)
		FROM UserRole ur
		JOIN RolePermission rp ON rp.RoleID = ur.RoleID
		JOIN Permission p ON p.ID = rp.PermissionID
		WHERE ur.UserID = ? AND p.Code IN ?`, userID, codes).Scan(&held).Error; err != nil {
		return false, err
	}
	if held > 0 {
		return true, nil
	}
	var super int64
	if err := db.Model(&models.User{}).Where("ID = ? AND IsSuperUser = 1", userID).Count(&super).Error; err != nil {
		return false, err
	}
	return super > 0, nil
}
