package workflow

import (
	"errors"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// processSeed is one default process created on migrate up. Seed does not
// rewrite an existing graph — change steps in Access → Workflows, then delete
// leftover operator roles if they are unused.
type processSeed struct {
	Code      string
	Name      string
	Doc       types.ContentType
	Note      string
	DraftName string   // draft node shown in the graph; defaults to Initiator
	Steps     []string // running approval nodes; defaults to Credit Controller → Managing Director
}

// Seed creates the default TIPER approval processes when they are missing.
// Steps are named but have no operator role until Access → Workflows assigns
// one. Empty steps skip on submit (see openOrSkip). Soft reject returns to
// draft. Initiator pools start empty.
func Seed(db *gorm.DB) error {
	for _, spec := range defaultProcesses() {
		if err := seedNamedProcess(db, spec.withDefaults()); err != nil {
			return err
		}
	}
	return nil
}

// SeedProcessCodes is the TIPER process catalogue used to retire leftover graphs.
func SeedProcessCodes() []string {
	specs := defaultProcesses()
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec.Code)
	}
	return out
}

const (
	initiatorNode    = "Initiator"
	completeNodeName = "Complete"
)

// defaultApprovalSteps is the chain after the initiator on every seeded process.
var defaultApprovalSteps = []string{"Credit Controller", "Managing Director"}

func (s processSeed) withDefaults() processSeed {
	if strings.TrimSpace(s.DraftName) == "" {
		s.DraftName = initiatorNode
	}
	if len(s.Steps) == 0 {
		s.Steps = append([]string(nil), defaultApprovalSteps...)
	}
	return s
}

func defaultProcesses() []processSeed {
	return []processSeed{
		{Code: "RCPT", Name: "Internal receipt", Doc: types.ReceiptContent, Note: "Provision or final internal vessel reception."},
		{Code: "MILOSS", Name: "MI loss batch", Doc: types.MiLossBatchContent, Note: "MI-loss rates used by variable storage fees — unique per product and contract."},
		{Code: "VARFEE", Name: "Variable storage fee batch", Doc: types.VariableFeeBatchContent, Note: "EWURA card with MI-loss per product and contract type."},
		{Code: "KOJFEE", Name: "KOJ fee batch", Doc: types.KojFeeBatchContent, Note: "KOJ price list."},
		{Code: "TBSFEE", Name: "TBS fee batch", Doc: types.TbsFeeBatchContent, Note: "TBS truck-loading price list."},
		{Code: "FCFEE", Name: "Fixed storage fee batch", Doc: types.BillingProfileContent, Note: "Fixed storage price list — one batch, many pricing models."},
		{Code: "FXRATE", Name: "Exchange rate", Doc: types.ExchangeRateContent, Note: "Dated TZS per USD quote used by fee batches."},
		{Code: "FCFBILL", Name: "Fixed storage billing run", Doc: types.BillingRunContent, Note: "First and nth FSF invoices."},
		{Code: "ZEROL", Name: "Zerolization", Doc: types.ZerolizationContent, Note: "Same-customer vessel consolidation."},
		{Code: "ILR", Name: "Internal loading request", Doc: types.GantryLoadingRequestContent, Note: "Truck loading orders (ILR header, ILO lines after approval)."},
		{Code: "PUMP", Name: "Pump-over request", Doc: types.PumpOverRequestContent, Note: "Pipeline delivery order."},
		{Code: "PUMPRPT", Name: "Pump-over report", Doc: types.PumpOverReportContent, Note: "Executed pump-over quantities."},
		{Code: "ITT", Name: "In-tank transfer", Doc: types.IttTransferContent, Note: "Ownership transfer. Stock posts two dated movements on approval (sender out, receiver in) so history before that date still shows the sender's full receipt volume. Later FSF/VSF on transferred stock charges the receiver."},
		{Code: "HOLDREL", Name: "Financial hold release", Doc: types.FinancialHoldContent, Note: "Release paid volume from financial hold onto free stock."},
		{Code: "COMP", Name: "Compartmentalization", Doc: types.CompartmentalizationContent, Note: "Gantry dispatch — writes the SAP3C file ALMA reads."},
		{Code: "AMEND", Name: "Loading amendment", Doc: types.OrderAmendmentContent, Note: "Quantity, product, or truck changes on an open ILO. Extend and batch-cancel apply immediately."},
		{Code: "COS", Name: "Change of service", Doc: types.ChangeOfServiceContent, Note: "Customer switches delivery method on one vessel parcel (vessel + vessel date). Fees stay; FSF billing follows the new method."},
	}
}

func seedNamedProcess(db *gorm.DB, spec processSeed) error {
	var proc models.Process
	err := db.Where("Code = ?", spec.Code).First(&proc).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := createLinearApproval(db, spec); err != nil {
			return err
		}
		if err := db.Where("Code = ?", spec.Code).First(&proc).Error; err != nil {
			return err
		}
	}

	return ensurePoolConfig(db, &proc)
}

// createLinearApproval inserts the draft (Initiator) node, ordered running
// steps, Complete/Rejected, agree transitions along the chain, and a reject
// transition from each running step.
func createLinearApproval(db *gorm.DB, spec processSeed) error {
	spec = spec.withDefaults()
	if len(spec.Steps) == 0 {
		return fmt.Errorf("workflow seed %q: at least one approval step is required", spec.Code)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		proc := models.Process{
			Code:           spec.Code,
			Name:           spec.Name,
			DocContentType: spec.Doc,
			AmendmentMode:  types.AmendPool,
			Note:           spec.Note,
		}
		if err := tx.Create(&proc).Error; err != nil {
			return err
		}

		draftName := strings.TrimSpace(spec.DraftName)
		if draftName == "" {
			draftName = initiatorNode
		}
		draft := models.Node{ProcessID: proc.ID, Name: draftName, Status: types.NodeDraft, NodeType: types.NodeTypeNode, Sequence: 0}
		running := make([]models.Node, len(spec.Steps))
		for i, name := range spec.Steps {
			running[i] = models.Node{
				ProcessID:  proc.ID,
				Name:       name,
				Status:     types.NodeRunning,
				NodeType:   types.NodeTypeNode,
				QuorumMode: types.QuorumAny,
				Sequence:   i + 1,
			}
		}
		completed := models.Node{ProcessID: proc.ID, Name: completeNodeName, Status: types.NodeCompleted, NodeType: types.NodeTypeNode, Sequence: len(spec.Steps) + 1}
		rejected := models.Node{ProcessID: proc.ID, Name: "Rejected", Status: types.NodeRejected, NodeType: types.NodeTypeNode, Sequence: len(spec.Steps) + 2}

		if err := tx.Create(&draft).Error; err != nil {
			return err
		}
		for i := range running {
			if err := tx.Create(&running[i]).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&completed).Error; err != nil {
			return err
		}
		if err := tx.Create(&rejected).Error; err != nil {
			return err
		}

		transitions := []models.Transition{
			{
				ProcessID:    proc.ID,
				Name:         "Submit",
				Code:         spec.Code + "-submit",
				ActType:      types.ActAgree,
				InputNodeID:  &draft.ID,
				OutputNodeID: &running[0].ID,
				IsActive:     true,
			},
		}
		for i := range running {
			next := &completed.ID
			if i+1 < len(running) {
				next = &running[i+1].ID
			}
			slug := slugCode(running[i].Name)
			transitions = append(transitions,
				models.Transition{
					ProcessID:    proc.ID,
					Name:         running[i].Name + " Approve",
					Code:         spec.Code + "-" + slug + "-approve",
					ActType:      types.ActAgree,
					InputNodeID:  &running[i].ID,
					OutputNodeID: next,
					IsActive:     true,
				},
				models.Transition{
					ProcessID:    proc.ID,
					Name:         running[i].Name + " Reject",
					Code:         spec.Code + "-" + slug + "-reject",
					ActType:      types.ActReject,
					InputNodeID:  &running[i].ID,
					OutputNodeID: &rejected.ID,
					IsActive:     true,
				},
			)
		}
		if err := tx.Create(&transitions).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Process{}).Where("ID = ?", proc.ID).
			Update("RejectReturnNodeID", draft.ID).Error; err != nil {
			return err
		}

		chain := draftName + " → " + strings.Join(spec.Steps, " → ") + " → " + completeNodeName
		logs.Infof("seeded default workflow process %q (%s)", spec.Code, chain)
		return nil
	})
}

// ensurePoolConfig sets pool amendment + reject-return on the draft node and
// creates an empty initiator pool when missing (re-runs of migrate up).
func ensurePoolConfig(db *gorm.DB, proc *models.Process) error {
	updates := map[string]any{}
	if proc.AmendmentMode != types.AmendPool {
		updates["AmendmentMode"] = types.AmendPool
	}

	var draft models.Node
	if err := db.Where("ProcessID = ? AND Status = ?", proc.ID, types.NodeDraft).
		Order("Sequence ASC").First(&draft).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else if proc.RejectReturnNodeID == nil || *proc.RejectReturnNodeID != draft.ID {
		updates["RejectReturnNodeID"] = draft.ID
	}

	if len(updates) > 0 {
		if err := db.Model(&models.Process{}).Where("ID = ?", proc.ID).Updates(updates).Error; err != nil {
			return err
		}
		logs.Infof("workflow: updated process %q amendment/return settings for initiator pool", proc.Code)
	}

	var pool models.InitiatorPool
	err := db.Where("ProcessID = ?", proc.ID).First(&pool).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := db.Create(&models.InitiatorPool{ProcessID: proc.ID}).Error; err != nil {
		return err
	}
	logs.Infof("workflow: created initiator pool for process %q", proc.Code)
	return nil
}
