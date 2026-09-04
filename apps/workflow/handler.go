// Package workflow exposes HTTP endpoints over the approval engine: process
// definitions, a user's task inbox, taking actions, and instance history.
package workflow

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	engine "dfms/internal/workflow"
	"dfms/pkg/audit"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

// Handler serves workflow endpoints backed by the engine.
type Handler struct {
	engine *engine.Engine
}

// NewHandler constructs a Handler.
func NewHandler(e *engine.Engine) *Handler { return &Handler{engine: e} }

// Processes lists all workflow process definitions with initiator pools.
func (h *Handler) Processes(c fiber.Ctx) error {
	procs, err := h.engine.ListProcesses(c.Context())
	if err != nil {
		return response.InternalServerError(c)
	}
	out := make([]engine.ProcessView, 0, len(procs))
	for _, p := range procs {
		view, err := h.engine.GetProcess(c.Context(), p.UID)
		if err != nil {
			return response.InternalServerError(c)
		}
		out = append(out, *view)
	}
	return response.OkDetail(c, out)
}

// GetProcess returns one process definition and its initiator pool.
func (h *Handler) GetProcess(c fiber.Ctx) error {
	view, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	return response.OkDetail(c, view)
}

// UpdateAmendmentMode sets creator vs pool soft-reject behaviour.
func (h *Handler) UpdateAmendmentMode(c fiber.Ctx) error {
	var body amendmentModeRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	before, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	if err := h.engine.UpdateAmendmentMode(c.Context(), c.Params("uid"), types.AmendmentMode(body.AmendmentMode)); err != nil {
		if err.Error() == "amendmentMode must be creator or pool" {
			return response.BadRequest(c, err.Error())
		}
		return mapEngineError(c, err)
	}
	prev := workflowSettingsAudit{AmendmentMode: string(before.Process.AmendmentMode)}
	next := workflowSettingsAudit{AmendmentMode: body.AmendmentMode}
	recordWorkflowAudit(c, before.Process.UID, "updated workflow amendment mode", prev, next)
	return response.OkMessage(c, audit.UpdateMessage(prev, next))
}

// ReplaceInitiatorPool sets who may amend soft-rejected documents for a process.
func (h *Handler) ReplaceInitiatorPool(c fiber.Ctx) error {
	var body initiatorPoolRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if body.UserIDs == nil {
		body.UserIDs = []string{}
	}
	before, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	if err := h.engine.SetInitiatorPool(c.Context(), c.Params("uid"), body.UserIDs); err != nil {
		if err.Error() == "one or more user ids are invalid or inactive" {
			return response.BadRequest(c, err.Error())
		}
		return mapEngineError(c, err)
	}
	view, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	prev := workflowSettingsAudit{InitiatorPool: poolAuditLabel(before.InitiatorPool)}
	next := workflowSettingsAudit{InitiatorPool: poolAuditLabel(view.InitiatorPool)}
	recordWorkflowAudit(c, view.Process.UID, "updated workflow initiator pool", prev, next)
	return response.Ok(c, audit.UpdateMessage(prev, next), view)
}

// ReplaceNotifications sets the process FYI watcher lists.
func (h *Handler) ReplaceNotifications(c fiber.Ctx) error {
	var body notificationsRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if body.NotifyOnSubmit == nil {
		body.NotifyOnSubmit = []string{}
	}
	if body.NotifyOnComplete == nil {
		body.NotifyOnComplete = []string{}
	}
	if body.NotifyOnReject == nil {
		body.NotifyOnReject = []string{}
	}
	before, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	if err := h.engine.SetNotifications(c.Context(), c.Params("uid"), engine.NotificationSettings{
		NotifyOnSubmit:   body.NotifyOnSubmit,
		NotifyOnComplete: body.NotifyOnComplete,
		NotifyOnReject:   body.NotifyOnReject,
	}); err != nil {
		if err.Error() == "one or more user ids are invalid or inactive" {
			return response.BadRequest(c, err.Error())
		}
		return mapEngineError(c, err)
	}
	view, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	prev := workflowSettingsAudit{
		NotifyOnSubmit:   poolAuditLabel(before.NotifyOnSubmit),
		NotifyOnComplete: poolAuditLabel(before.NotifyOnComplete),
		NotifyOnReject:   poolAuditLabel(before.NotifyOnReject),
	}
	next := workflowSettingsAudit{
		NotifyOnSubmit:   poolAuditLabel(view.NotifyOnSubmit),
		NotifyOnComplete: poolAuditLabel(view.NotifyOnComplete),
		NotifyOnReject:   poolAuditLabel(view.NotifyOnReject),
	}
	recordWorkflowAudit(c, view.Process.UID, "updated workflow notifications", prev, next)
	return response.Ok(c, audit.UpdateMessage(prev, next), view)
}

// ReplaceApprovalSteps updates the ordered approval chain for a process.
func (h *Handler) ReplaceApprovalSteps(c fiber.Ctx) error {
	var body replaceStepsRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	before, err := h.engine.GetProcess(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	view, err := h.engine.ReplaceApprovalSteps(c.Context(), c.Params("uid"), body.Steps)
	if err != nil {
		return mapEngineError(c, err)
	}
	prev := stepsAuditSnapshot(before)
	next := stepsAuditSnapshot(view)
	recordWorkflowAudit(c, view.Process.UID, "updated workflow approval steps", prev, next)
	return response.Ok(c, audit.UpdateMessage(prev, next), view)
}

// workflowSettingsAudit is a struct so audit.Diff can compare fields (Diff ignores maps).
type workflowSettingsAudit struct {
	AmendmentMode    string `json:"amendmentMode"`
	InitiatorPool    string `json:"initiatorPool"`
	NotifyOnSubmit   string `json:"notifyOnSubmit"`
	NotifyOnComplete string `json:"notifyOnComplete"`
	NotifyOnReject   string `json:"notifyOnReject"`
}

func poolAuditLabel(members []engine.PoolMember) string {
	if len(members) == 0 {
		return "(empty)"
	}
	parts := make([]string, 0, len(members))
	for _, m := range members {
		name := strings.TrimSpace(m.FirstName + " " + m.LastName)
		switch {
		case name != "" && m.Email != "":
			parts = append(parts, name+" <"+m.Email+">")
		case name != "":
			parts = append(parts, name)
		default:
			parts = append(parts, m.Email)
		}
	}
	return strings.Join(parts, "; ")
}

type workflowStepsAudit struct {
	ApprovalChain string `json:"approvalChain"`
}

func stepsAuditSnapshot(view *engine.ProcessView) workflowStepsAudit {
	if view == nil {
		return workflowStepsAudit{}
	}
	parts := make([]string, 0, len(view.ApprovalSteps))
	for _, s := range view.ApprovalSteps {
		role := s.RoleName
		if role == "" && s.RoleID != 0 {
			role = fmt.Sprintf("role#%d", s.RoleID)
		}
		qm := s.QuorumMode
		if qm == "" {
			qm = "any"
		}
		parts = append(parts, fmt.Sprintf("%s (%s, %s)", s.Name, role, qm))
	}
	return workflowStepsAudit{ApprovalChain: strings.Join(parts, " → ")}
}

func recordWorkflowAudit(c fiber.Ctx, processUID, baseDesc string, before, after any) {
	if audit.Default == nil {
		return
	}
	changes := audit.DropMetaKeys(audit.Diff(before, after))
	if len(changes) == 0 {
		return
	}
	entry := audit.AuditEntry(
		c, types.ModuleWorkflow, types.ActionUpdate, processUID, types.ProcessContent,
		audit.EnrichDescription(baseDesc, changes),
	)
	entry.Changes = changes
	_ = audit.Default.Record(c.Context(), nil, entry)
}

// MyTasks returns a paginated list of the authenticated user's pending tasks.
// Optional query: search (document no/summary).
func (h *Handler) MyTasks(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	uid := middleware.GetUserIDFromContext(c)
	single, many := parseDocContentTypes(c.Query("docContentType"))
	filter := engine.TaskFilter{
		Search:          strings.Trim(search.Search, "%"),
		DocContentType:  single,
		DocContentTypes: many,
		OrderBy:         search.OrderBy,
		SortDirection:   search.SortDirection,
	}
	if search.Export {
		all, err := collectMyTasks(h, c, uid, filter)
		if err != nil {
			return response.InternalServerError(c)
		}
		return exportMyTasks(c, all)
	}
	tasks, total, err := h.engine.MyTasks(c.Context(), uid, search.Page, search.PageSize, filter)
	if err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, response.BuildPagination(search, tasks, total))
}

const inboxPreviewSize = 8

// InboxPreview is a compact pending-task snapshot for the header bell.
type InboxPreview struct {
	Count int64             `json:"count"`
	Items []engine.TaskView `json:"items"`
}

// MyInbox returns the caller's pending-task count plus the newest few items.
// Intended for the header notification badge (polled); the full queue remains
// GET /tasks/mine.
func (h *Handler) MyInbox(c fiber.Ctx) error {
	uid := middleware.GetUserIDFromContext(c)
	tasks, total, err := h.engine.MyTasks(c.Context(), uid, 1, inboxPreviewSize, engine.TaskFilter{
		OrderBy:       "createdAt",
		SortDirection: "DESC",
	})
	if err != nil {
		return response.InternalServerError(c)
	}
	if tasks == nil {
		tasks = []engine.TaskView{}
	}
	return response.OkDetail(c, InboxPreview{Count: total, Items: tasks})
}

// MyDecisions returns a paginated list of the user's past agree or reject actions.
// Query: ?kind=approved|rejected (default approved); optional search.
func (h *Handler) MyDecisions(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	uid := middleware.GetUserIDFromContext(c)
	kind := c.Query("kind", "approved")
	var act types.EventAction
	switch kind {
	case "approved", "agree":
		act = types.EventAgree
	case "rejected", "reject":
		act = types.EventReject
	default:
		return response.BadRequest(c, "kind must be approved or rejected")
	}
	single, many := parseDocContentTypes(c.Query("docContentType"))
	filter := engine.TaskFilter{
		Search:          strings.Trim(search.Search, "%"),
		DocContentType:  single,
		DocContentTypes: many,
		OrderBy:         search.OrderBy,
		SortDirection:   search.SortDirection,
	}
	if search.Export {
		all, err := collectMyDecisions(h, c, uid, act, filter)
		if err != nil {
			return response.InternalServerError(c)
		}
		return exportMyDecisions(c, all)
	}
	rows, total, err := h.engine.MyDecisions(c.Context(), uid, act, search.Page, search.PageSize, filter)
	if err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, response.BuildPagination(search, rows, total))
}

// Act applies an approval action to an instance.
func (h *Handler) Act(c fiber.Ctx) error {
	instanceUID := c.Params("uid")
	var body actRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	action := types.TransitionAction(body.Action)
	inst, err := h.engine.Act(c.Context(), engine.ActParams{
		InstanceUID:    instanceUID,
		UserID:         middleware.GetUserIDFromContext(c),
		Action:         action,
		Comment:        body.Comment,
		IPAddress:      c.IP(),
		TotalRejection: body.TotalRejection,
		TargetNodeUID:  body.TargetNode,
	})
	if err != nil {
		return mapEngineError(c, err)
	}
	recordWorkflowDecision(c, inst, action, body.Comment, body.TotalRejection)
	return response.OkDetail(c, inst)
}

// ActMany applies the same approval action to several workflow instances.
func (h *Handler) ActMany(c fiber.Ctx) error {
	var body bulkActRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	action := types.TransitionAction(body.Action)

	type row struct {
		InstanceID string `json:"instanceId"`
		Error      string `json:"error,omitempty"`
		Instance   any    `json:"instance,omitempty"`
	}
	out := make([]row, 0, len(body.InstanceIDs))
	userID := middleware.GetUserIDFromContext(c)
	for _, uid := range body.InstanceIDs {
		inst, err := h.engine.Act(c.Context(), engine.ActParams{
			InstanceUID:    uid,
			UserID:         userID,
			Action:         action,
			Comment:        body.Comment,
			IPAddress:      c.IP(),
			TotalRejection: body.TotalRejection,
			TargetNodeUID:  body.TargetNode,
		})
		r := row{InstanceID: uid}
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Instance = inst
			recordWorkflowDecision(c, inst, action, body.Comment, body.TotalRejection)
		}
		out = append(out, r)
	}
	return response.OkDetail(c, out)
}

func recordWorkflowDecision(c fiber.Ctx, inst *models.ProcessInstance, action types.TransitionAction, comment string, totalRejection bool) {
	if audit.Default == nil || inst == nil {
		return
	}
	kind := strings.ToLower(types.ContentTypeLabel(inst.DocContentType))
	no := strings.TrimSpace(inst.No)
	if no == "" {
		no = inst.UID
	}
	auditAction := types.ActionUpdate
	verb := string(action)
	switch action {
	case types.ActAgree:
		auditAction = types.ActionApprove
		verb = "approval"
	case types.ActReject:
		auditAction = types.ActionReject
		if totalRejection {
			verb = "reject"
		} else {
			verb = "return"
		}
	case types.ActBack:
		verb = "return"
	}
	desc := fmt.Sprintf("%s for %s number %s", verb, kind, no)
	entry := audit.AuditEntry(c, types.ModuleWorkflow, auditAction, inst.UID, types.ProcessInstanceContent, desc)
	entry.Metadata = map[string]any{
		"action":         string(action),
		"comment":        comment,
		"totalRejection": totalRejection,
		"documentNo":     no,
		"kind":           kind,
	}
	_ = audit.Default.Record(c.Context(), nil, entry)
}

// GetInstance returns one approval instance with its process graph and trail.
func (h *Handler) GetInstance(c fiber.Ctx) error {
	view, err := h.engine.GetInstanceView(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	return response.OkDetail(c, view)
}

// History returns the approval trail for a process instance (initiator → current).
func (h *Handler) History(c fiber.Ctx) error {
	events, err := h.engine.Progress(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	return response.OkDetail(c, events)
}

// GetMySubstitute returns the signed-in user's approval delegation settings.
func (h *Handler) GetMySubstitute(c fiber.Ctx) error {
	view, err := h.engine.GetSubstitute(c.Context(), middleware.GetUserIDFromContext(c))
	if err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, view)
}

// SetMySubstitute configures or clears the signed-in user's approval substitute.
func (h *Handler) SetMySubstitute(c fiber.Ctx) error {
	var body substituteRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	p := engine.SetSubstituteParams{
		PrincipalUserID: middleware.GetUserIDFromContext(c),
		DelegateUID:     strings.TrimSpace(body.DelegateUserID),
		ProcessUID:      strings.TrimSpace(body.ProcessID),
		NodeUID:         strings.TrimSpace(body.NodeID),
	}
	if body.StartsAt != nil && strings.TrimSpace(*body.StartsAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.StartsAt))
		if err != nil {
			return response.BadRequest(c, "invalid startsAt (use RFC3339)")
		}
		p.StartsAt = &t
	}
	if body.EndsAt != nil && strings.TrimSpace(*body.EndsAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.EndsAt))
		if err != nil {
			return response.BadRequest(c, "invalid endsAt (use RFC3339)")
		}
		p.EndsAt = &t
	}
	view, err := h.engine.SetSubstitute(c.Context(), p)
	if err != nil {
		return mapEngineError(c, err)
	}
	if strings.TrimSpace(body.DelegateUserID) == "" {
		recordSubstituteAudit(c, types.ActionDelete, "cleared approval substitute(s)", map[string]any{
			"processId": strings.TrimSpace(body.ProcessID),
		})
		return response.Ok(c, "Delegations cleared", view)
	}
	recordSubstituteAudit(c, types.ActionUpdate, "configured approval substitute", map[string]any{
		"delegateUserId": strings.TrimSpace(body.DelegateUserID),
		"processId":      strings.TrimSpace(body.ProcessID),
		"nodeId":         strings.TrimSpace(body.NodeID),
	})
	return response.Ok(c, "Delegation saved", view)
}

// ClearMySubstitute removes one process-scoped substitute assignment.
func (h *Handler) ClearMySubstitute(c fiber.Ctx) error {
	assignmentUID := c.Params("uid")
	view, err := h.engine.ClearSubstitute(
		c.Context(),
		middleware.GetUserIDFromContext(c),
		assignmentUID,
	)
	if err != nil {
		return mapEngineError(c, err)
	}
	recordSubstituteAudit(c, types.ActionDelete, "removed approval substitute", map[string]any{
		"assignmentId": assignmentUID,
	})
	return response.Ok(c, "Delegation removed", view)
}

func recordSubstituteAudit(c fiber.Ctx, action types.Action, desc string, meta map[string]any) {
	if audit.Default == nil {
		return
	}
	entry := audit.AuditEntry(
		c, types.ModuleWorkflow, action, middleware.GetUserUIDFromContext(c),
		types.SubstituteContent, desc,
	)
	entry.Metadata = meta
	_ = audit.Default.Record(c.Context(), nil, entry)
}

func mapEngineError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, engine.ErrProcessNotFound):
		return response.NotFound(c, "workflow process not found")
	case errors.Is(err, engine.ErrInstanceNotFound):
		return response.NotFound(c, "workflow instance not found")
	case errors.Is(err, engine.ErrNoTask):
		return response.Forbidden(c, "you have no pending task on this document")
	case errors.Is(err, engine.ErrNotRunning):
		return response.Conflict(c, "workflow is not awaiting action")
	case errors.Is(err, engine.ErrNoTransition):
		return response.Conflict(c, "no transition available for this action")
	case errors.Is(err, engine.ErrNodeNotFound):
		return response.BadRequest(c, "target node not found")
	case errors.Is(err, engine.ErrInvalidTarget):
		return response.BadRequest(c, err.Error())
	case errors.Is(err, engine.ErrNodeInUse):
		return response.Conflict(c, err.Error())
	case errors.Is(err, engine.ErrInvalidDefinition):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = strings.TrimSpace(msg[i+2:])
		}
		return response.BadRequest(c, msg)
	case errors.Is(err, engine.ErrSubstituteSelf),
		errors.Is(err, engine.ErrSubstituteNotAllowed),
		errors.Is(err, engine.ErrSubstituteDelegate),
		errors.Is(err, engine.ErrSubstituteProcess):
		return response.BadRequest(c, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "endsAt") || strings.Contains(msg, "unknown approval") {
			return response.BadRequest(c, msg)
		}
		return response.InternalServerError(c)
	}
}

func parseDocContentType(raw string) types.ContentType {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 || n > 255 {
		return 0
	}
	return types.ContentType(n)
}

func parseDocContentTypes(raw string) (types.ContentType, []types.ContentType) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.Contains(raw, ",") {
		return parseDocContentType(raw), nil
	}
	var out []types.ContentType
	for _, part := range strings.Split(raw, ",") {
		if ct := parseDocContentType(part); ct != 0 {
			out = append(out, ct)
		}
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return 0, out
}
