package audit

import (
	"dfms/apps/auth/middleware"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
)

// AuditEntry builds an audit.Entry from the current HTTP request, including
// the authenticated user's display name when available.
func AuditEntry(c fiber.Ctx, module types.Module, action types.Action, recordID string, recordType types.ContentType, description string) Entry {
	claims := middleware.GetUserClaimsFromContext(c)
	entry := EntryFromRequest(claims.ISI, module, claims.Username, action, GetClientIP(c), c.Get("User-Agent"), c.RequestID(), recordID, recordType, description)
	return entry
}

// RecordHTTP writes a create/update/delete audit row for the current request.
// Pass the saved row as after on create, both snapshots on edit, and the
// deleted row as before on delete so Changes always carries field values.
func RecordHTTP(c fiber.Ctx, module types.Module, action types.Action, recordID string, recordType types.ContentType, description string, before, after any) {
	if Default == nil {
		return
	}
	entry := AuditEntry(c, module, action, recordID, recordType, description)
	AttachChanges(&entry, before, after)
	_ = Default.Record(c.Context(), nil, entry)
}
