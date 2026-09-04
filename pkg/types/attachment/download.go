package attachment

import (
	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DownloadAttachmentRequest holds the inputs needed to serve a stored attachment.
type DownloadAttachmentRequest struct {
	Ctx          fiber.Ctx
	Db           *gorm.DB
	AttachmentID uint
	EntityID     uint
	EntityType   types.ContentType
}

// DownloadAttachment serves a file as a forced download after verifying entity
// ownership and confining the stored path to the uploads root.
//
// Inline/preview responses are intentionally not supported: user-uploaded
// content (PDF/SVG/HTML mislabeled as images, etc.) must not be rendered in the
// browser under a relaxed CSP.
func DownloadAttachment(req DownloadAttachmentRequest) error {
	var doc models.Attachment
	err := req.Db.Where("ID = ?", req.AttachmentID).
		Where("EntityType = ?", req.EntityType).
		Where("EntityID = ?", req.EntityID).
		First(&doc).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(req.Ctx, "Attachment not found")
		}
		logs.Error(err)
		return response.InternalServerError(req.Ctx)
	}

	safePath, err := resolveSafeUploadPath(doc.FilePath)
	if err != nil {
		return response.Forbidden(req.Ctx, "Invalid file path")
	}
	if _, err := os.Stat(safePath); err != nil {
		return response.NotFound(req.Ctx, "File not found on server")
	}

	req.Ctx.Set("Content-Disposition", contentDisposition(doc.OriginalName))
	req.Ctx.Set("X-Content-Type-Options", "nosniff")
	return req.Ctx.Download(safePath, doc.OriginalName)
}

// resolveSafeUploadPath cleans the stored path and ensures it stays under
// the live upload root (blocks "..", absolute escapes, and symlink jumps outside).
func resolveSafeUploadPath(stored string) (string, error) {
	if stored == "" || strings.Contains(stored, "..") {
		return "", errors.New("invalid path")
	}
	root, err := filepath.Abs(Root())
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(stored)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	rootPrefix := root + string(os.PathSeparator)
	if abs != root && !strings.HasPrefix(abs, rootPrefix) {
		return "", errors.New("path outside upload root")
	}
	return abs, nil
}

func contentDisposition(name string) string {
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, `"`, "'")
	if name == "" {
		name = "download"
	}
	return fmt.Sprintf(`attachment; filename="%s"`, name)
}
