package workflow

import (
	"errors"

	"dfms/pkg/response"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ListAttachments returns files on the document behind this approval instance.
func (h *Handler) ListAttachments(c fiber.Ctx) error {
	rows, err := h.engine.ListInstanceAttachments(c.Context(), c.Params("uid"))
	if err != nil {
		return mapEngineError(c, err)
	}
	return response.OkDetail(c, rows)
}

// DownloadAttachment serves one file from the instance's source document.
func (h *Handler) DownloadAttachment(c fiber.Ctx) error {
	inst, row, err := h.engine.GetInstanceAttachment(c.Context(), c.Params("uid"), c.Params("aid"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "attachment not found")
		}
		return mapEngineError(c, err)
	}
	return attachment.DownloadAttachment(attachment.DownloadAttachmentRequest{
		Ctx: c, Db: h.engine.DB(), AttachmentID: row.ID,
		EntityID: inst.ObjectID, EntityType: inst.DocContentType,
	})
}
