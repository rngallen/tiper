package auth

import (
	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/audit"
	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// RoleHandler manages all HTTP operations for Role resources
type RoleHandler struct {
	db *gorm.DB
}

// NewRoleHandler creates a new handler with database dependency
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{db: db}
}

// getRoles returns a paginated, searchable, sortable list of roles
// Query params: ?page=1&pageSize=25&search=admin&orderBy=name&sort=desc
func (h *RoleHandler) list(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Secure column whitelisting — prevents SQL injection via ORDER BY
	allowedSort := map[string]string{
		"name":        "Name",
		"description": "Description",
		"category":    "Category",
	}
	sortColumn, ok := allowedSort[search.OrderBy]
	if !ok {
		sortColumn = "Name"
	}

	query := h.db.WithContext(c.Context()).Model(&models.Role{})
	if search.HasSearch() {
		query = query.Where("Name LIKE ? OR Description LIKE ?", search.Search, search.Search)
	}
	query = query.Order(stableOrder(sortColumn, search.SortDirection))

	if search.Export {
		return export.Query(c, query, "Roles", "roles",
			[]any{"ID", "Name", "Category", "Description"},
			func(role models.Role) []any {
				return []any{role.ID, role.Name, role.Category, role.Description}
			},
		)
	}

	var items []models.Role
	roles, err := response.NewPaginator(c, query, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := attachPermissionCounts(h.db.WithContext(c.Context()), items); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	if err := attachRoleCanDelete(h.db.WithContext(c.Context()), items); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	return response.OkDetail(c, roles)
}

// options returns a compact role catalogue for assignment pickers (Users admin).
func (h *RoleHandler) options(c fiber.Ctx) error {
	var roles []models.Role
	if err := h.db.WithContext(c.Context()).
		Select("ID", "Name", "Description", "Category").
		Order("Category ASC, Name ASC").
		Find(&roles).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	if err := attachPermissionCounts(h.db.WithContext(c.Context()), roles); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	type opt struct {
		ID              uint             `json:"id"`
		Name            string           `json:"name"`
		Description     string           `json:"description"`
		Category        types.RoleFamily `json:"category"`
		PermissionCount int              `json:"permissionCount"`
	}
	out := make([]opt, len(roles))
	for i, r := range roles {
		out[i] = opt{
			ID: r.ID, Name: r.Name, Description: r.Description,
			Category: r.Category, PermissionCount: r.PermissionCount,
		}
	}
	return response.OkDetail(c, out)
}

// getRole retrieves a single role with its permissions preloaded
func (h *RoleHandler) get(c fiber.Ctx) error {
	roleId := fiber.Params[uint](c, "id")

	var role models.Role
	if err := h.db.WithContext(c.Context()).Preload("Permissions").First(&role, roleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "role not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}
	role.PermissionCount = len(role.Permissions)
	one := []models.Role{role}
	if err := attachRoleCanDelete(h.db.WithContext(c.Context()), one); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	role.CanDelete = one[0].CanDelete
	return response.OkDetail(c, role)
}

// createRole creates a new role with normalized, uppercase name
func (h *RoleHandler) create(c fiber.Ctx) error {
	var payload rolesCreate

	// Parse and validate payload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, err.Error())
	}
	payload.Sanitize()
	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Normalize role name
	normalizedName := strings.ToUpper(strings.TrimSpace(payload.Name))

	// Database context
	db := h.db.WithContext(c.Context())

	// Prevent duplicate role names (case-insensitive)
	err := db.Where("Name = ?", normalizedName).First(&models.Role{}).Error
	if err == nil {
		return response.BadRequest(c, "Role with this name already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	// Create a new role
	role := models.Role{
		Name:        normalizedName,
		Description: payload.Description,
		Category:    types.NormalizeRoleFamily(payload.Category),
	}

	if err := db.Create(&role).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleRole, types.ActionCreate, role.UID,
		types.RoleContent, "role "+role.Name+" created", nil, role)

	return response.Created(c, role)
}

// updateRole updates only the description (name is immutable)
func (h *RoleHandler) update(c fiber.Ctx) error {
	roleId := fiber.Params[uint](c, "id")
	var payload rolesUpdate

	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}

	payload.Sanitize()
	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	// Database context
	db := h.db.WithContext(c.Context())

	var role models.Role
	if err := db.First(&role, roleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "role not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	before := role
	role.Description = payload.Description
	role.Category = types.NormalizeRoleFamily(payload.Category)

	if err := db.Model(&role).Updates(map[string]any{
		"Description": role.Description,
		"Category":    role.Category,
	}).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleRole, types.ActionUpdate, role.UID,
		types.RoleContent, "role "+role.Name+" updated", before, role)

	return response.OkMessage(c, audit.UpdateMessage(before, role))
}

// delete removes a role that is unused: not assigned to users and not an
// operator on any workflow step. Bootstrap (seeded) names are allowed.
func (h *RoleHandler) delete(c fiber.Ctx) error {
	roleId := fiber.Params[uint](c, "id")

	// Database context
	db := h.db.WithContext(c.Context())

	var role models.Role
	if err := db.First(&role, roleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "role not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	if reason, err := roleDeleteBlockReason(db, role); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	} else if reason != "" {
		return response.Conflict(c, reason)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("RoleID = ?", role.ID).Delete(&models.RolesPermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
	if err != nil {
		logs.Errorf("delete role: %v", err)
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleRole, types.ActionDelete, role.UID,
		types.RoleContent, "role "+role.Name+" deleted", role, nil)
	return response.Deleted(c)
}

// replaceRolePermissions atomically replaces all permissions for a role
func (h *RoleHandler) replaceRolePermissions(c fiber.Ctx) error {
	var payload replacePermissions

	// Parse and validate
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Database context
	db := h.db.WithContext(c.Context())

	var role models.Role
	roleId := fiber.Params[uint](c, "id")
	if err := db.Preload("Permissions").First(&role, "ID = ?", roleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "role not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	before := role.Permissions

	var permsList []models.Permission
	if len(payload.PermissionIDs) > 0 {
		if err := db.Where("ID IN ?", payload.PermissionIDs).Find(&permsList).Error; err != nil {
			logs.Error(err)
			return response.InternalServerError(c)
		}
		if len(permsList) != len(payload.PermissionIDs) {
			return response.BadRequest(c, "one or more permission IDs are invalid")
		}
	}

	if err := db.Model(&role).Association("Permissions").Replace(&permsList); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	middleware.InvalidatePermissionsForRole(c.Context(), db, role.ID)

	summary := audit.SummarizeSet("Permissions", permCodes(before), permCodes(permsList))
	if audit.Default != nil {
		desc := "role " + role.Name + " permissions updated"
		if summary != "" {
			desc = desc + " — " + summary
		}
		entry := audit.AuditEntry(c, types.ModuleRole, types.ActionUpdate, role.UID,
			types.RoleContent, desc)
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	return response.OkMessage(c, audit.UpdateMessageFromSummary(summary))
}

func permCodes(perms []models.Permission) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		if code := strings.TrimSpace(p.Code); code != "" {
			out = append(out, code)
		}
	}
	return out
}

// attachPermissionCounts sets Role.PermissionCount from RolePermission without
// preloading every permission row (list and assignment pickers).
func attachPermissionCounts(db *gorm.DB, roles []models.Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]uint, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	type row struct {
		RoleID uint
		C      int
	}
	var rows []row
	if err := db.Model(&models.RolesPermission{}).
		Select("RoleID, COUNT(*) AS C").
		Where("RoleID IN ?", ids).
		Group("RoleID").
		Scan(&rows).Error; err != nil {
		return err
	}
	byID := make(map[uint]int, len(rows))
	for _, r := range rows {
		byID[r.RoleID] = r.C
	}
	for i := range roles {
		roles[i].PermissionCount = byID[roles[i].ID]
	}
	return nil
}

func roleIDsFromQuery(db *gorm.DB, table, column string, ids []uint) (map[uint]struct{}, error) {
	out := make(map[uint]struct{})
	if len(ids) == 0 {
		return out, nil
	}
	var found []uint
	if err := db.Table(table).Where(column+" IN ?", ids).Distinct(column).Pluck(column, &found).Error; err != nil {
		return nil, err
	}
	for _, id := range found {
		out[id] = struct{}{}
	}
	return out, nil
}

// attachRoleCanDelete sets CanDelete when the role has no users and is not
// an operator on any workflow node. Seeded names are not special.
func attachRoleCanDelete(db *gorm.DB, roles []models.Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]uint, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	assigned, err := roleIDsFromQuery(db, "UserRole", "RoleID", ids)
	if err != nil {
		return err
	}
	onNode, err := roleIDsFromQuery(db, "NodeOperatorRole", "RoleID", ids)
	if err != nil {
		return err
	}
	for i := range roles {
		_, users := assigned[roles[i].ID]
		_, node := onNode[roles[i].ID]
		roles[i].CanDelete = !users && !node
	}
	return nil
}

func roleDeleteBlockReason(db *gorm.DB, role models.Role) (string, error) {
	var n int64
	if err := db.Model(&models.UserRole{}).Where("RoleID = ?", role.ID).Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return "role is assigned to users and cannot be deleted", nil
	}
	n = 0
	if err := db.Table("NodeOperatorRole").Where("RoleID = ?", role.ID).Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return "role is used on a workflow step and cannot be deleted", nil
	}
	return "", nil
}
