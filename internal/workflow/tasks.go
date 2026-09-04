package workflow

import (
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// resolveOperators returns the distinct active users assigned to a node, both
// directly (OperatorUsers) and via OperatorRoles membership.
func resolveOperators(tx *gorm.DB, nodeID uint) ([]models.User, error) {
	var users []models.User
	err := tx.Raw(`
		SELECT DISTINCT u.* FROM [User] u
		JOIN NodeOperatorUser j ON j.UserID = u.ID
		WHERE j.NodeID = ? AND u.IsActive = 1
		UNION
		SELECT DISTINCT u.* FROM [User] u
		JOIN UserRole ur ON ur.UserID = u.ID
		JOIN NodeOperatorRole nr ON nr.RoleID = ur.RoleID
		WHERE nr.NodeID = ? AND u.IsActive = 1`, nodeID, nodeID).
		Scan(&users).Error
	return users, err
}

// initiatorPoolUsers returns the users in a process's initiator pool.
func initiatorPoolUsers(tx *gorm.DB, processID uint) ([]models.User, error) {
	var users []models.User
	err := tx.Raw(`
		SELECT u.* FROM [User] u
		JOIN [WorkflowInitiatorPoolUsers] j ON j.UserID = u.ID
		JOIN [InitiatorPool] p ON p.ID = j.InitiatorPoolID
		WHERE p.ProcessID = ? AND u.IsActive = 1`, processID).
		Scan(&users).Error
	return users, err
}

// genTasks creates pending tasks for the given users at a node, after closing
// any still-pending tasks left over from a previous visit to that node.
func genTasks(tx *gorm.DB, instanceID, nodeID uint, users []models.User) error {
	if err := tx.Model(&models.Task{}).
		Where("InstanceID = ? AND NodeID = ? AND Status = ?", instanceID, nodeID, types.TaskPending).
		Update("Status", types.TaskSkipped).Error; err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	tasks := make([]models.Task, 0, len(users))
	for _, u := range users {
		uid := u.ID
		tasks = append(tasks, models.Task{
			InstanceID: instanceID,
			NodeID:     nodeID,
			UserID:     &uid,
			Status:     types.TaskPending,
		})
	}
	return tx.Create(&tasks).Error
}

// skipPendingAtNode closes leftover pending tasks at a node after the stage
// is decided (quorum any: one agree or reject is enough; quorum all: called
// only after the last required agree). Covers peer operators and the
// principal when a substitute acted — there is one task row per assignee.
func skipPendingAtNode(tx *gorm.DB, instanceID, nodeID uint) error {
	if instanceID == 0 || nodeID == 0 {
		return nil
	}
	return tx.Model(&models.Task{}).
		Where("InstanceID = ? AND NodeID = ? AND Status = ?", instanceID, nodeID, types.TaskPending).
		Updates(map[string]any{
			"Status":  types.TaskSkipped,
			"ActedAt": now(),
		}).Error
}

// pendingTaskCount returns the number of still-pending tasks at a node.
func pendingTaskCount(tx *gorm.DB, instanceID, nodeID uint) (int64, error) {
	var n int64
	err := tx.Model(&models.Task{}).
		Where("InstanceID = ? AND NodeID = ? AND Status = ?", instanceID, nodeID, types.TaskPending).
		Count(&n).Error
	return n, err
}

func now() *time.Time { t := time.Now(); return &t }

// substituteCover is one delegate covering a principal at the current node.
type substituteCover struct {
	Delegate  models.User
	Principal models.User
}

// withSubstitutes returns operators plus active substitutes for this process
// and node. Delegates who are already operators are omitted (they already get
// their own task and assignee email).
func withSubstitutes(tx *gorm.DB, processID, nodeID uint, operators []models.User) ([]substituteCover, error) {
	if len(operators) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(operators))
	byID := make(map[uint]models.User, len(operators))
	opIDs := make(map[uint]struct{}, len(operators))
	for _, u := range operators {
		ids = append(ids, u.ID)
		byID[u.ID] = u
		opIDs[u.ID] = struct{}{}
	}
	now := time.Now().UTC()
	var rows []models.ApprovalSubstitute
	err := tx.Preload("Delegate").
		Where("PrincipalUserID IN ? AND ProcessID = ?", ids, processID).
		Where("StartsAt <= ? AND (EndsAt IS NULL OR EndsAt >= ?)", now, now).
		Where("NodeID IS NULL OR NodeID = ?", nodeID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]substituteCover, 0, len(rows))
	for i := range rows {
		d := rows[i].Delegate
		if d == nil || !d.IsActive || d.ID == 0 {
			continue
		}
		if _, isOp := opIDs[d.ID]; isOp {
			continue
		}
		principal, ok := byID[rows[i].PrincipalUserID]
		if !ok {
			continue
		}
		out = append(out, substituteCover{Delegate: *d, Principal: principal})
	}
	return out, nil
}

// fireTaskNotifications emails/SMS operators, then each substitute covering
// those operators (copy says they are acting for the principal). Safe to call
// after the approval transaction commits.
func fireTaskNotifications(n Notifier, inst *models.ProcessInstance, node *models.Node, operators []models.User, covers []substituteCover) {
	if n == nil || inst == nil || node == nil {
		return
	}
	n.NotifyTasks(inst, node, operators)
	for i := range covers {
		n.NotifySubstituteTasks(inst, node, covers[i].Delegate, covers[i].Principal)
	}
}
