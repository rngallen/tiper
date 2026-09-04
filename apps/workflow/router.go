package workflow

import (
	"dfms/apps/auth/middleware"
	engine "dfms/internal/workflow"
	"dfms/pkg/permissions"

	"github.com/gofiber/fiber/v3"
)

// WorkflowRouter registers /api/v1/workflow endpoints.
func WorkflowRouter(app *fiber.App, e *engine.Engine) {
	h := NewHandler(e)

	grp := app.Group("/api/v1/workflow", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())

	grp.Get("/processes", middleware.PermissionMiddleware(permissions.WorkflowManage, permissions.WorkflowRead), h.Processes).Name("workflow.processes")
	grp.Get("/processes/:uid<string>", middleware.PermissionMiddleware(permissions.WorkflowManage, permissions.WorkflowRead), h.GetProcess).Name("workflow.processes.get")
	grp.Patch("/processes/:uid<string>/amendment-mode", middleware.PermissionMiddleware(permissions.WorkflowManage), h.UpdateAmendmentMode).Name("workflow.processes.amendmentMode")
	grp.Put("/processes/:uid<string>/initiator-pool", middleware.PermissionMiddleware(permissions.WorkflowManage), h.ReplaceInitiatorPool).Name("workflow.processes.initiatorPool")
	grp.Put("/processes/:uid<string>/notifications", middleware.PermissionMiddleware(permissions.WorkflowManage), h.ReplaceNotifications).Name("workflow.processes.notifications")
	grp.Put("/processes/:uid<string>/steps", middleware.PermissionMiddleware(permissions.WorkflowManage), h.ReplaceApprovalSteps).Name("workflow.processes.steps")

	// A user's own task inbox and decisions — requires workflow.tasks (or manage).
	tasks := middleware.PermissionMiddleware(permissions.WorkflowTasks, permissions.WorkflowManage)
	grp.Get("/tasks/mine", tasks, h.MyTasks).Name("workflow.tasks.mine")
	grp.Get("/tasks/inbox", tasks, h.MyInbox).Name("workflow.tasks.inbox")
	grp.Get("/decisions/mine", tasks, h.MyDecisions).Name("workflow.decisions.mine")

	grp.Get("/substitute/me", tasks, h.GetMySubstitute).Name("workflow.substitute.get")
	grp.Put("/substitute/me", tasks, h.SetMySubstitute).Name("workflow.substitute.set")
	grp.Delete("/substitute/me/:uid<string>", tasks, h.ClearMySubstitute).Name("workflow.substitute.clear")

	grp.Post("/act/bulk", tasks, h.ActMany).Name("workflow.act.bulk")
	grp.Post("/instances/:uid<string>/act", tasks, h.Act).Name("workflow.act")
	see := middleware.PermissionMiddleware(permissions.WorkflowTasks, permissions.WorkflowManage, permissions.WorkflowRead)
	grp.Get("/instances/:uid<string>", see, h.GetInstance).Name("workflow.instance")
	grp.Get("/instances/:uid<string>/history", see, h.History).Name("workflow.history")
	grp.Get("/instances/:uid<string>/attachments", see, h.ListAttachments).Name("workflow.attachments")
	grp.Get("/instances/:uid<string>/attachments/:aid<string>", see, h.DownloadAttachment).Name("workflow.attachments.get")
}
