package attach

import (
	"errors"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Entity is the owning document for list / upload / download / delete.
type Entity = attachment.Entity

// CanMutateStatus is true for draft and returned (amend after soft reject).
func CanMutateStatus(status string) bool {
	return attachment.CanMutateStatus(status)
}

// Register mounts GET/POST /attachments and GET/DELETE /attachments/:aid.
func Register(g fiber.Router, prefix string, read, write fiber.Handler, db *gorm.DB, ct types.ContentType, load func(fiber.Ctx) (Entity, error)) {
	g.Get(prefix+"/attachments", read, func(c fiber.Ctx) error {
		return serveList(c, db, ct, load)
	})
	g.Post(prefix+"/attachments", write, func(c fiber.Ctx) error {
		return serveUpload(c, db, ct, load)
	})
	g.Get(prefix+"/attachments/:aid", read, func(c fiber.Ctx) error {
		return serveDownload(c, db, ct, load)
	})
	g.Delete(prefix+"/attachments/:aid", write, func(c fiber.Ctx) error {
		return serveDelete(c, db, ct, load)
	})
}

func serveList(c fiber.Ctx, db *gorm.DB, ct types.ContentType, load func(fiber.Ctx) (Entity, error)) error {
	e, err := loadEntity(c, load)
	if err != nil {
		return err
	}
	var rows []models.Attachment
	if err := db.WithContext(c.Context()).Where("EntityType = ? AND EntityID = ?", ct, e.ID).
		Order("CreatedAt ASC").Find(&rows).Error; err != nil {
		return err
	}
	return response.OkDetail(c, rows)
}

func serveUpload(c fiber.Ctx, db *gorm.DB, ct types.ContentType, load func(fiber.Ctx) (Entity, error)) error {
	e, err := loadEntity(c, load)
	if err != nil {
		return err
	}
	if !e.CanMutate {
		return response.Conflict(c, "attachments can only be added to a draft or returned document")
	}
	form, err := c.MultipartForm()
	if err != nil || form == nil || len(form.File["files"]) == 0 {
		return response.BadRequest(c, "files are required")
	}
	if err := attachment.UploadAttachments(attachment.UploadAttachmentsRequest{
		Ctx: c, Db: db, DocumentNumber: e.DocumentNumber,
		ContentType: ct, EntityID: e.ID,
		UploadedBy:  middleware.GetUserIDFromContext(c),
		Attachments: form.File["files"],
	}); err != nil {
		return response.BadRequest(c, err.Error())
	}
	var rows []models.Attachment
	if err := db.WithContext(c.Context()).Where("EntityType = ? AND EntityID = ?", ct, e.ID).
		Order("CreatedAt ASC").Find(&rows).Error; err != nil {
		return err
	}
	return response.Ok(c, "File uploaded", rows)
}

func serveDownload(c fiber.Ctx, db *gorm.DB, ct types.ContentType, load func(fiber.Ctx) (Entity, error)) error {
	e, err := loadEntity(c, load)
	if err != nil {
		return err
	}
	var row models.Attachment
	if err := db.Where("UID = ? AND EntityType = ? AND EntityID = ?", c.Params("aid"), ct, e.ID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "attachment not found")
		}
		return err
	}
	return attachment.DownloadAttachment(attachment.DownloadAttachmentRequest{
		Ctx: c, Db: db, AttachmentID: row.ID, EntityID: e.ID, EntityType: ct,
	})
}

func serveDelete(c fiber.Ctx, db *gorm.DB, ct types.ContentType, load func(fiber.Ctx) (Entity, error)) error {
	e, err := loadEntity(c, load)
	if err != nil {
		return err
	}
	if !e.CanMutate {
		return response.Conflict(c, "attachments can only be removed from a draft or returned document")
	}
	var row models.Attachment
	if err := db.Where("UID = ? AND EntityType = ? AND EntityID = ?", c.Params("aid"), ct, e.ID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "attachment not found")
		}
		return err
	}
	if row.CopiedFromID != nil {
		return response.UnprocessableEntity(c, errors.New("copied attachments cannot be removed"))
	}
	if err := db.Delete(&row).Error; err != nil {
		return err
	}
	return response.OkMessage(c, "Attachment removed")
}

func loadEntity(c fiber.Ctx, load func(fiber.Ctx) (Entity, error)) (Entity, error) {
	e, err := load(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Entity{}, response.NotFound(c, "document not found")
		}
		return Entity{}, err
	}
	return e, nil
}
