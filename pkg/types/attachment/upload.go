package attachment

import (
	"dfms/apps/models"
	"dfms/pkg/types"
	"dfms/utils"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// UploadAttachmentsRequest holds all the inputs needed for a generic attachment upload.
type UploadAttachmentsRequest struct {
	Ctx            fiber.Ctx
	Db             *gorm.DB
	DocumentNumber string
	// FolderKey, when set, is the per-document directory under YYYY/MM
	// (sanitised via EntityDirName). Empty uses DocumentNumber — the document
	// number or master code, not the entity ULID.
	FolderKey   string
	ContentType types.ContentType
	EntityID    uint
	UploadedBy  uint
	Attachments []*multipart.FileHeader
}

// UploadAttachments validates and persists each uploaded file, then creates a
// matching Attachment row for it. Files are written first; DB rows are inserted
// in one transaction. Any failure removes files saved in this request so the
// store does not keep orphans.
func UploadAttachments(req UploadAttachmentsRequest) error {
	folder := EntityDirName(stringsOr(req.FolderKey, req.DocumentNumber))
	saveDir, err := createEntityPath(req.ContentType, folder)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	docLabel := SanitizeLabel(req.DocumentNumber)
	if docLabel == "" {
		docLabel = "doc"
	}

	type savedFile struct {
		path string
		doc  models.Attachment
	}
	saved := make([]savedFile, 0, len(req.Attachments))
	cleanup := func() {
		for _, f := range saved {
			_ = os.Remove(f.path)
		}
	}

	for _, file := range req.Attachments {
		if file.Size > MaxFileSize() {
			cleanup()
			return fmt.Errorf("file %s exceeds max size", file.Filename)
		}

		name, ext := fileNameWithoutExt(file.Filename)
		if !isAllowedFileExtension(ext) {
			cleanup()
			return fmt.Errorf("file type not allowed: %s", file.Filename)
		}
		stem := SanitizeLabel(name)
		if stem == "" {
			stem = "file"
		}

		uid, err := utils.GetULID()
		if err != nil {
			cleanup()
			return err
		}

		storedName := fmt.Sprintf("%s_%s_%s%s", docLabel, stem, uid[:8], ext)
		fullPath := filepath.Join(saveDir, storedName)

		if err := req.Ctx.SaveFile(file, fullPath); err != nil {
			cleanup()
			return fmt.Errorf("failed to save file %s: %w", file.Filename, err)
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			_ = os.Remove(fullPath)
			cleanup()
			return fmt.Errorf("failed to stat file %s: %w", file.Filename, err)
		}
		if info.Size() > MaxFileSize() {
			_ = os.Remove(fullPath)
			cleanup()
			return fmt.Errorf("file %s exceeds max size", file.Filename)
		}

		mime := utils.GetMIME(ext)
		saved = append(saved, savedFile{
			path: fullPath,
			doc: models.Attachment{
				OriginalName: file.Filename,
				StoredName:   storedName,
				FilePath:     fullPath,
				EntityID:     req.EntityID,
				EntityType:   req.ContentType,
				Size:         info.Size(),
				Extension:    ext,
				UploadedByID: req.UploadedBy,
				ByteSize:     byteSize(info.Size()),
				Mime:         mime,
				CanPreview:   isPreviewableMime(mime, ext),
				IsActive:     true,
			},
		})
	}

	err = req.Db.Transaction(func(tx *gorm.DB) error {
		for i := range saved {
			if err := tx.Create(&saved[i].doc).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		cleanup()
		return err
	}
	return nil
}

func stringsOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
