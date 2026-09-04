package workflow

import (
	"context"
	"strings"

	engine "dfms/internal/workflow"
	"dfms/pkg/types"

	"github.com/jellydator/validation"
)

type amendmentModeRequest struct {
	AmendmentMode string `json:"amendmentMode"`
}

// Validate requires amendmentMode to be creator or pool.
func (r amendmentModeRequest) Validate(ctx context.Context) error {
	mode := strings.TrimSpace(strings.ToLower(r.AmendmentMode))
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.AmendmentMode, validation.Required, validation.By(func(any) error {
			if mode != string(types.AmendCreator) && mode != string(types.AmendPool) {
				return validation.NewError("validation_invalid", "amendmentMode must be creator or pool")
			}
			return nil
		})),
	)
}

type initiatorPoolRequest struct {
	UserIDs []string `json:"userIds"`
}

type notificationsRequest struct {
	NotifyOnSubmit   []string `json:"notifyOnSubmit"`
	NotifyOnComplete []string `json:"notifyOnComplete"`
	NotifyOnReject   []string `json:"notifyOnReject"`
}

// Validate bounds FYI watcher lists.
func (r notificationsRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.NotifyOnSubmit, validation.Length(0, 50)),
		validation.Field(&r.NotifyOnComplete, validation.Length(0, 50)),
		validation.Field(&r.NotifyOnReject, validation.Length(0, 50)),
	)
}

// Validate bounds the initiator pool; empty clears it.
func (r initiatorPoolRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.UserIDs, validation.Length(0, 50)),
	)
}

type replaceStepsRequest struct {
	Steps []engine.ApprovalStepInput `json:"steps"`
}

// Validate requires a non-empty approval chain.
func (r replaceStepsRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Steps, validation.Required, validation.Length(1, 50)),
	)
}

// actRequest is the body for taking an action on an instance.
type actRequest struct {
	Action         string `json:"action"`         // agree | reject | transfer | back
	Comment        string `json:"comment"`        //
	TotalRejection bool   `json:"totalRejection"` // reject only
	TargetNode     string `json:"targetNode"`     // transfer/back only
}

// Validate requires a known action; transfer/back need targetNode.
// Agree/reject require a non-empty comment for the audit trail.
func (r actRequest) Validate(ctx context.Context) error {
	action := types.TransitionAction(r.Action)
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.Action, validation.Required, validation.By(func(any) error {
			switch action {
			case types.ActAgree, types.ActReject, types.ActTransfer, types.ActBack:
				return nil
			default:
				return validation.NewError("validation_invalid", "action must be one of agree, reject, transfer, back")
			}
		})),
		validation.Field(&r.TargetNode, validation.Length(0, 26), validation.By(func(any) error {
			if (action == types.ActTransfer || action == types.ActBack) && strings.TrimSpace(r.TargetNode) == "" {
				return validation.NewError("validation_required", "targetNode is required for transfer/back")
			}
			return nil
		})),
		validation.Field(&r.Comment, validation.By(func(any) error {
			c := strings.TrimSpace(r.Comment)
			if (action == types.ActAgree || action == types.ActReject) && c == "" {
				return validation.NewError("validation_required", "comment is required when approving or rejecting")
			}
			if len(c) > 2000 {
				return validation.NewError("validation_length", "comment must be at most 2000 characters")
			}
			return nil
		})),
	)
}

// bulkActRequest is the body of POST /workflow/act/bulk.
type bulkActRequest struct {
	InstanceIDs    []string `json:"instanceIds"`
	Action         string   `json:"action"`
	Comment        string   `json:"comment"`
	TotalRejection bool     `json:"totalRejection"`
	TargetNode     string   `json:"targetNode"`
}

// Validate requires instanceIds and a known action.
// Agree/reject require a non-empty comment for the audit trail.
func (r bulkActRequest) Validate(ctx context.Context) error {
	action := types.TransitionAction(r.Action)
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.InstanceIDs, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.Action, validation.Required, validation.By(func(any) error {
			switch action {
			case types.ActAgree, types.ActReject, types.ActTransfer, types.ActBack:
				return nil
			default:
				return validation.NewError("validation_invalid", "action must be one of agree, reject, transfer, back")
			}
		})),
		validation.Field(&r.TargetNode, validation.Length(0, 26), validation.By(func(any) error {
			if (action == types.ActTransfer || action == types.ActBack) && strings.TrimSpace(r.TargetNode) == "" {
				return validation.NewError("validation_required", "targetNode is required for transfer/back")
			}
			return nil
		})),
		validation.Field(&r.Comment, validation.By(func(any) error {
			c := strings.TrimSpace(r.Comment)
			if (action == types.ActAgree || action == types.ActReject) && c == "" {
				return validation.NewError("validation_required", "comment is required when approving or rejecting")
			}
			if len(c) > 2000 {
				return validation.NewError("validation_length", "comment must be at most 2000 characters")
			}
			return nil
		})),
	)
}

type substituteRequest struct {
	DelegateUserID string  `json:"delegateUserId"`
	ProcessID      string  `json:"processId"`
	NodeID         string  `json:"nodeId"`
	StartsAt       *string `json:"startsAt"`
	EndsAt         *string `json:"endsAt"`
}

// Validate bounds substitute UIDs; empty delegate clears (one process or all).
func (r substituteRequest) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &r,
		validation.Field(&r.DelegateUserID, validation.Length(0, 26)),
		validation.Field(&r.ProcessID, validation.Length(0, 26)),
		validation.Field(&r.NodeID, validation.Length(0, 26)),
	)
}
