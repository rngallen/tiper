package masterdata

import (
	"strings"

	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func (h handler) listStatuses(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.StockStatus{}).Preload("Parent"),
		search, "Name", "Code",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"name": "Name", "code": "Code",
	}, "Name", "Stock statuses", "stock-statuses",
		[]any{"Code", "Name", "Transit", "Local", "Mining", "Proration", "Active"},
		func(r models.StockStatus) []any {
			return []any{r.Code, r.Name, r.IsTransit, r.IsLocal, r.IsMining, r.IsProration, r.IsActive}
		},
	)
}

func applyStatusFlags(row *models.StockStatus, in statusRequest) {
	if in.IsTransit {
		row.IsTransit, row.IsLocal, row.IsMining, row.IsProration = true, false, false, false
		return
	}
	if in.IsProration {
		row.IsTransit, row.IsLocal, row.IsMining, row.IsProration = false, false, false, true
		return
	}
	if in.IsMining || in.IsLocal {
		row.IsTransit, row.IsLocal, row.IsMining, row.IsProration = false, true, in.IsMining, false
		return
	}
	t, loc, mine, pror := models.ClassifyStockStatus(in.Name)
	row.IsTransit, row.IsLocal, row.IsMining, row.IsProration = t, loc, mine, pror
}

func (h handler) createStatus(c fiber.Ctx) error {
	var in statusRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	code := in.Code
	if code == "" {
		code = models.SlugStatusCode(in.Name)
	}
	row := models.StockStatus{Code: code, Name: in.Name, IsActive: activeOrDefault(in.IsActive)}
	applyStatusFlags(&row, in)
	if pid := statusIDByUID(h.db, in.ParentID); pid > 0 {
		row.ParentID = &pid
	} else if row.IsTransit && !strings.EqualFold(row.Code, string(types.StockTransit)) {
		var root models.StockStatus
		if h.db.Where("Code = ?", types.StockTransit).First(&root).Error == nil {
			row.ParentID = &root.ID
		}
	}
	if err := h.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return writeErr(c, err, "a stock status with this code already exists")
	}
	recordAudit(c, types.ModuleInventory, types.ActionCreate, row.UID, types.StockStatusContent, "stock status "+row.Name+" created", nil, row)
	return response.Created(c, row)
}

func (h handler) updateStatus(c fiber.Ctx) error {
	var row models.StockStatus
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "stock status not found")
	}
	before := row
	var in statusRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	row.Name = in.Name
	applyStatusFlags(&row, in)
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	if in.ParentID == "" {
		row.ParentID = nil
	} else if pid := statusIDByUID(h.db, in.ParentID); pid > 0 && pid != row.ID {
		row.ParentID = &pid
	}
	if err := h.db.WithContext(c.Context()).Save(&row).Error; err != nil {
		return writeErr(c, err, "could not update stock status")
	}
	recordAudit(c, types.ModuleInventory, types.ActionUpdate, row.UID, types.StockStatusContent, "stock status "+row.Name+" updated", before, row)
	return okUpdate(c, row, before, row)
}

func statusIDByUID(db *gorm.DB, uid string) uint {
	return idByUID[models.StockStatus](db, uid)
}
