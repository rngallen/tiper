package workflow

import (
	"time"

	engine "dfms/internal/workflow"
	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

const exportPageSize = 500

func exportDocKind(ct types.ContentType) string {
	return types.ContentTypeLabel(ct)
}

func formatExportTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("02/01/2006 15:04")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func exportMyTasks(c fiber.Ctx, tasks []engine.TaskView) error {
	data := make([][]any, 0, len(tasks))
	for _, t := range tasks {
		data = append(data, []any{
			t.No,
			exportDocKind(t.DocContentType),
			t.Summary,
			t.NodeName,
			t.OnBehalfOfName,
			formatExportTime(t.CreatedAt),
		})
	}
	return export.Slice(c, "Approvals", "approval_inbox",
		[]any{
			"Document no", "Type", "Summary", "Step", "On behalf of", "Queued at",
		}, data)
}

func exportMyDecisions(c fiber.Ctx, rows []engine.DecisionView) error {
	data := make([][]any, 0, len(rows))
	for _, r := range rows {
		data = append(data, []any{
			r.No,
			exportDocKind(r.DocContentType),
			r.Summary,
			r.ActName,
			r.Comment,
			deref(r.FromNode),
			deref(r.ToNode),
			string(r.InstanceStatus),
			formatExportTime(r.CreatedAt),
		})
	}
	return export.Slice(c, "Approvals", "approval_decisions",
		[]any{
			"Document no", "Type", "Summary", "Your action", "Comment", "From", "To",
			"Workflow now", "When",
		}, data)
}

func collectMyTasks(h *Handler, c fiber.Ctx, uid uint, filter engine.TaskFilter) ([]engine.TaskView, error) {
	var all []engine.TaskView
	page := 1
	for {
		batch, total, err := h.engine.MyTasks(c.Context(), uid, page, exportPageSize, filter)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
		all = append(all, batch...)
		if int64(len(all)) >= total || len(batch) < exportPageSize {
			return all, nil
		}
		page++
	}
}

func collectMyDecisions(h *Handler, c fiber.Ctx, uid uint, act types.EventAction, filter engine.TaskFilter) ([]engine.DecisionView, error) {
	var all []engine.DecisionView
	page := 1
	for {
		batch, total, err := h.engine.MyDecisions(c.Context(), uid, act, page, exportPageSize, filter)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
		all = append(all, batch...)
		if int64(len(all)) >= total || len(batch) < exportPageSize {
			return all, nil
		}
		page++
	}
}
