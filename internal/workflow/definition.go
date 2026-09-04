package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// Sentinel errors for process-definition updates.
var (
	ErrInvalidDefinition = errors.New("invalid workflow definition")
	ErrNodeInUse         = errors.New("workflow step is in use and cannot be removed")
)

// ApprovalStepInput is one editable running approval step in the chain.
type ApprovalStepInput struct {
	ID         string `json:"id"`                   // existing node UID; empty = create
	Name       string `json:"name"`                 // display name
	RoleID     uint   `json:"roleId"`               // operator role (numeric Role.ID)
	QuorumMode string `json:"quorumMode,omitempty"` // any | all (default any)
}

// RoleOption is a compact role picker row for the steps editor.
type RoleOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// ReplaceApprovalSteps replaces the ordered running approval steps for a
// process and rebuilds agree/reject transitions around the fixed terminal
// nodes (draft / initiator pool, Complete, Rejected).
//
// Steps that still have open instances, pending tasks, or history cannot be
// removed — rename them instead. After a step is removed, its operator role
// can be deleted from Access → Roles if it is unused.
func (e *Engine) ReplaceApprovalSteps(ctx context.Context, processUID string, steps []ApprovalStepInput) (*ProcessView, error) {
	if err := validateApprovalSteps(steps); err != nil {
		return nil, err
	}

	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var proc models.Process
		if err := tx.
			Preload("Nodes", func(db *gorm.DB) *gorm.DB { return db.Order("Sequence ASC") }).
			Preload("Nodes.OperatorRoles").
			Preload("Transitions").
			Where("UID = ?", processUID).First(&proc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProcessNotFound
			}
			return err
		}

		draft, completed, rejected, err := ensureTerminalNodes(tx, &proc)
		if err != nil {
			return err
		}

		roleByID, err := loadRolesByID(tx, steps)
		if err != nil {
			return err
		}

		runningByUID := map[string]*models.Node{}
		for i := range proc.Nodes {
			n := &proc.Nodes[i]
			if n.Status == types.NodeRunning {
				runningByUID[n.UID] = n
			}
		}

		keep := map[string]struct{}{}
		for _, s := range steps {
			if s.ID != "" {
				keep[s.ID] = struct{}{}
			}
		}
		for uid, n := range runningByUID {
			if _, ok := keep[uid]; ok {
				continue
			}
			used, err := nodeReferenced(tx, n.ID)
			if err != nil {
				return err
			}
			if used {
				return fmt.Errorf("%w: %q", ErrNodeInUse, n.Name)
			}
			if err := tx.Model(n).Association("OperatorRoles").Clear(); err != nil {
				return err
			}
			if err := tx.Model(n).Association("OperatorUsers").Clear(); err != nil {
				return err
			}
			if err := tx.Delete(n).Error; err != nil {
				return err
			}
			delete(runningByUID, uid)
		}

		// Park remaining nodes on high sequences so (ProcessID, Sequence)
		// stays unique while we reorder / insert into the middle of the chain.
		if err := parkNodeSequences(tx, proc.ID); err != nil {
			return err
		}
		runningByUID = map[string]*models.Node{}
		var running []models.Node
		if err := tx.Preload("OperatorRoles").
			Where("ProcessID = ? AND Status = ?", proc.ID, types.NodeRunning).
			Find(&running).Error; err != nil {
			return err
		}
		for i := range running {
			runningByUID[running[i].UID] = &running[i]
		}

		// Terminals were parked too — reload so later Saves use current rows.
		if err := tx.Where("ID = ?", draft.ID).First(draft).Error; err != nil {
			return err
		}
		if err := tx.Where("ID = ?", completed.ID).First(completed).Error; err != nil {
			return err
		}
		if err := tx.Where("ID = ?", rejected.ID).First(rejected).Error; err != nil {
			return err
		}

		ordered := make([]*models.Node, 0, len(steps))
		for i, s := range steps {
			seq := i + 1
			qm := types.QuorumMode(s.QuorumMode)
			if qm == "" {
				qm = types.QuorumAny
			}
			roles := []models.Role{}
			if s.RoleID != 0 {
				if role, ok := roleByID[s.RoleID]; ok {
					roles = []models.Role{role}
				}
			}

			var node *models.Node
			if s.ID != "" {
				existing, ok := runningByUID[s.ID]
				if !ok {
					return fmt.Errorf("%w: unknown step id %q", ErrInvalidDefinition, s.ID)
				}
				existing.Name = strings.TrimSpace(s.Name)
				existing.Sequence = seq
				existing.QuorumMode = qm
				if err := tx.Save(existing).Error; err != nil {
					return err
				}
				if err := tx.Model(existing).Association("OperatorRoles").Replace(roles); err != nil {
					return err
				}
				node = existing
			} else {
				created := models.Node{
					ProcessID:  proc.ID,
					Name:       strings.TrimSpace(s.Name),
					Status:     types.NodeRunning,
					NodeType:   types.NodeTypeNode,
					QuorumMode: qm,
					Sequence:   seq,
				}
				if err := tx.Create(&created).Error; err != nil {
					return err
				}
				if err := tx.Model(&created).Association("OperatorRoles").Replace(roles); err != nil {
					return err
				}
				node = &created
				runningByUID[created.UID] = node
			}
			ordered = append(ordered, node)
		}

		// Draft / terminals keep stable sequences around the middle chain.
		draft.Sequence = 0
		completed.Sequence = len(ordered) + 1
		rejected.Sequence = len(ordered) + 2
		for _, n := range []*models.Node{draft, completed, rejected} {
			if err := tx.Save(n).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("ProcessID = ?", proc.ID).Delete(&models.Transition{}).Error; err != nil {
			return err
		}

		chain := make([]*models.Node, 0, len(ordered)+2)
		chain = append(chain, draft)
		chain = append(chain, ordered...)
		chain = append(chain, completed)

		transitions := make([]models.Transition, 0, len(chain)+len(ordered))
		for i := 0; i < len(chain)-1; i++ {
			from, to := chain[i], chain[i+1]
			name := "Approve"
			code := slugCode(from.Name) + "-approve"
			if i == 0 {
				name = "Submit"
				code = "submit"
			} else {
				name = from.Name + " Approve"
			}
			transitions = append(transitions, models.Transition{
				ProcessID:    proc.ID,
				Name:         name,
				Code:         code,
				ActType:      types.ActAgree,
				InputNodeID:  new(from.ID),
				OutputNodeID: new(to.ID),
				IsActive:     true,
			})
		}
		for _, n := range ordered {
			transitions = append(transitions, models.Transition{
				ProcessID:    proc.ID,
				Name:         n.Name + " Reject",
				Code:         slugCode(n.Name) + "-reject",
				ActType:      types.ActReject,
				InputNodeID:  new(n.ID),
				OutputNodeID: new(rejected.ID),
				IsActive:     true,
			})
		}
		for i := range transitions {
			if err := tx.Create(&transitions[i]).Error; err != nil {
				return fmt.Errorf("create transition %q: %w", transitions[i].Code, err)
			}
		}

		return tx.Model(&models.Process{}).Where("ID = ?", proc.ID).
			Updates(map[string]any{
				"RejectReturnNodeID": draft.ID,
				"Note":               buildProcessNote(draft.Name, ordered),
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return e.GetProcess(ctx, processUID)
}

func validateApprovalSteps(steps []ApprovalStepInput) error {
	if len(steps) == 0 {
		return fmt.Errorf("%w: at least one approval step is required", ErrInvalidDefinition)
	}
	if len(steps) > 20 {
		return fmt.Errorf("%w: at most 20 approval steps are allowed", ErrInvalidDefinition)
	}
	seenNames := map[string]struct{}{}
	seenIDs := map[string]struct{}{}
	for i, s := range steps {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return fmt.Errorf("%w: step %d needs a name", ErrInvalidDefinition, i+1)
		}
		key := strings.ToLower(name)
		if _, ok := seenNames[key]; ok {
			return fmt.Errorf("%w: duplicate step name %q", ErrInvalidDefinition, name)
		}
		seenNames[key] = struct{}{}
		qm := types.QuorumMode(s.QuorumMode)
		if qm != "" && qm != types.QuorumAny && qm != types.QuorumAll {
			return fmt.Errorf("%w: quorumMode must be any or all", ErrInvalidDefinition)
		}
		if s.ID != "" {
			if _, ok := seenIDs[s.ID]; ok {
				return fmt.Errorf("%w: duplicate step id", ErrInvalidDefinition)
			}
			seenIDs[s.ID] = struct{}{}
		}
	}
	return nil
}

func loadRolesByID(tx *gorm.DB, steps []ApprovalStepInput) (map[uint]models.Role, error) {
	ids := make([]uint, 0, len(steps))
	seen := map[uint]struct{}{}
	for _, s := range steps {
		if s.RoleID == 0 {
			continue
		}
		if _, ok := seen[s.RoleID]; ok {
			continue
		}
		seen[s.RoleID] = struct{}{}
		ids = append(ids, s.RoleID)
	}
	if len(ids) == 0 {
		return map[uint]models.Role{}, nil
	}
	var roles []models.Role
	if err := tx.Where("ID IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) != len(ids) {
		return nil, fmt.Errorf("%w: one or more role ids are invalid", ErrInvalidDefinition)
	}
	out := make(map[uint]models.Role, len(roles))
	for _, r := range roles {
		out[r.ID] = r
	}
	return out, nil
}

func ensureTerminalNodes(tx *gorm.DB, proc *models.Process) (draft, completed, rejected *models.Node, err error) {
	for i := range proc.Nodes {
		n := &proc.Nodes[i]
		switch n.Status {
		case types.NodeDraft:
			if draft == nil {
				draft = n
			}
		case types.NodeCompleted:
			if completed == nil {
				completed = n
			}
		case types.NodeRejected:
			if rejected == nil {
				rejected = n
			}
		}
	}
	if draft == nil {
		n := models.Node{
			ProcessID: proc.ID,
			Name:      draftNodeName(proc),
			Status:    types.NodeDraft,
			NodeType:  types.NodeTypeNode,
			Sequence:  0,
		}
		if err := tx.Create(&n).Error; err != nil {
			return nil, nil, nil, err
		}
		draft = &n
		proc.Nodes = append(proc.Nodes, n)
	}
	if completed == nil {
		n := models.Node{
			ProcessID: proc.ID,
			Name:      completeNodeName,
			Status:    types.NodeCompleted,
			NodeType:  types.NodeTypeNode,
			Sequence:  99,
		}
		if err := tx.Create(&n).Error; err != nil {
			return nil, nil, nil, err
		}
		completed = &n
		proc.Nodes = append(proc.Nodes, n)
	}
	if rejected == nil {
		n := models.Node{
			ProcessID: proc.ID,
			Name:      "Rejected",
			Status:    types.NodeRejected,
			NodeType:  types.NodeTypeNode,
			Sequence:  100,
		}
		if err := tx.Create(&n).Error; err != nil {
			return nil, nil, nil, err
		}
		rejected = &n
		proc.Nodes = append(proc.Nodes, n)
	}
	return draft, completed, rejected, nil
}

func nodeReferenced(tx *gorm.DB, nodeID uint) (bool, error) {
	var n int64
	if err := tx.Model(&models.ProcessInstance{}).Where("CurNodeID = ?", nodeID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := tx.Model(&models.Task{}).Where("NodeID = ?", nodeID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := tx.Model(&models.Event{}).
		Where("OldNodeID = ? OR NewNodeID = ?", nodeID, nodeID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func buildProcessNote(draftName string, steps []*models.Node) string {
	names := make([]string, 0, len(steps))
	for _, s := range steps {
		names = append(names, s.Name)
	}
	who := strings.TrimSpace(draftName)
	if who == "" {
		who = "The initiator"
	}
	mid := strings.Join(names, ", ")
	if mid == "" {
		return "Soft reject returns to the initiator pool."
	}
	if len(names) == 1 {
		return fmt.Sprintf(
			"%s submits; %s must approve. Soft reject returns to the initiator pool.",
			who, names[0],
		)
	}
	return fmt.Sprintf(
		"%s submits; %s must each approve. Soft reject returns to the initiator pool.",
		who, mid,
	)
}

// draftNodeName is the draft-node label used when a process is missing one.
func draftNodeName(*models.Process) string {
	return initiatorNode
}

func slugCode(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := slugNonAlnum.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		return "step"
	}
	return out
}

// parkNodeSequences moves every node in the process onto a temporary high
// Sequence band so subsequent reorders/inserts cannot violate
// idx_uniqueNodeSequence (ProcessID, Sequence).
func parkNodeSequences(tx *gorm.DB, processID uint) error {
	var nodes []models.Node
	if err := tx.Where("ProcessID = ?", processID).Order("Sequence ASC, ID ASC").Find(&nodes).Error; err != nil {
		return err
	}
	const parkBase = 10_000
	for i := range nodes {
		if err := tx.Model(&nodes[i]).Update("Sequence", parkBase+i).Error; err != nil {
			return fmt.Errorf("park node sequence %s: %w", nodes[i].UID, err)
		}
	}
	return nil
}

// ListOperatorRoles returns roles available for step operator assignment.
func (e *Engine) ListOperatorRoles(ctx context.Context) ([]RoleOption, error) {
	var roles []models.Role
	if err := e.db.WithContext(ctx).Select("ID", "Name").Order("Name ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	out := make([]RoleOption, 0, len(roles))
	for _, r := range roles {
		out = append(out, RoleOption{ID: r.ID, Name: r.Name})
	}
	return out, nil
}
