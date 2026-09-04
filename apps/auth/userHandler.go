package auth

import (
	"errors"
	"fmt"
	"strings"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/internal/notify"
	"dfms/pkg/audit"
	"dfms/pkg/constants"
	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/utils"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// UserHandler manages user administration endpoints.
type UserHandler struct{ db *gorm.DB }

// NewUserHandler constructs a UserHandler.
func NewUserHandler(db *gorm.DB) *UserHandler { return &UserHandler{db: db} }

// getUsers returns a paginated, searchable list of users.
func (h *UserHandler) list(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	allowedFields := map[string]string{
		"email":     "[User].Email",
		"firstname": "[User].FirstName",
		"lastname":  "[User].LastName",
		"phone":     "[Profile].PhoneNumber",
		"title":     "[Profile].Title",
		"active":    "[User].IsActive",
	}

	sortColumn, ok := allowedFields[search.OrderBy]
	if !ok {
		sortColumn = "[User].UID"
	}

	// Database context
	db := h.db.WithContext(c.Context())

	// LEFT JOIN Profile for sort-by-phone/title only. Preload loads Profile
	// (including AppearanceSetting JSON) in a separate query — association
	// Joins("Profile") would SELECT that nvarchar JSON into the User row and
	// break on MSSQL.
	query := db.Model(&models.User{}).
		Preload("Profile").
		Preload("Roles")
	if sortColumn == "[Profile].PhoneNumber" || sortColumn == "[Profile].Title" {
		query = query.Joins("LEFT JOIN [Profile] ON [Profile].UserID = [User].ID")
	}
	if search.HasSearch() {
		query = query.Where(
			"[User].FirstName LIKE ? OR [User].LastName LIKE ? OR [User].Email LIKE ?",
			search.Search, search.Search, search.Search,
		)
	}

	if search.IsActive != nil {
		query = query.Where("[User].IsActive = ?", *search.IsActive)
	}

	query = query.Order(sortColumn + " " + search.SortDirection + ", [User].ID ASC")

	if search.Export {
		return export.Query(c, query, "Users", "users",
			[]any{"ID", "First Name", "Last Name", "Email", "Phone Number", "Title", "Is Active"},
			func(user models.User) []any {
				return []any{
					user.UID,
					user.FirstName,
					user.LastName,
					user.Email,
					user.Profile.PhoneNumber,
					user.Profile.Title,
					user.IsActive,
				}
			},
		)
	}

	var items []models.User
	result, err := response.NewPaginator(c, query, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	attachUserCanDelete(items, middleware.GetUserIDFromContext(c))
	return response.OkDetail(c, result)
}

// searchOptions returns a compact user list for pickers (initiator pool, substitutes).
// Uses the shared search parser. Active, unlocked users only; the caller is omitted.
func (h *UserHandler) searchOptions(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	limit := search.PageSize
	if limit > 50 {
		limit = 50
	}
	db := h.db.WithContext(c.Context()).Model(&models.User{}).
		Select("UID", "Email", "FirstName", "LastName").
		Where("IsActive = 1 AND IsLocked = 0")
	// Always omit the caller (ULID and numeric PK). Pickers must not offer
	// the signed-in user as a substitute or pool member for themselves.
	if uid := middleware.GetUserUIDFromContext(c); uid != "" {
		db = db.Where("UID <> ?", uid)
	}
	if id := middleware.GetUserIDFromContext(c); id != 0 {
		db = db.Where("ID <> ?", id)
	}
	if search.HasSearch() {
		db = db.Where(
			"FirstName LIKE ? OR LastName LIKE ? OR Email LIKE ?",
			search.Search, search.Search, search.Search,
		)
	}
	var users []models.User
	if err := db.Order("Email ASC").Limit(limit).Find(&users).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	type opt struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	out := make([]opt, len(users))
	for i, u := range users {
		out[i] = opt{ID: u.UID, Email: u.Email, FirstName: u.FirstName, LastName: u.LastName}
	}
	return response.OkDetail(c, out)
}

// createUser provisions a new user with a temporary password.
func (h *UserHandler) create(c fiber.Ctx) error {
	var payload userCreateSchema

	// Parse and validate payload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	phone := strings.TrimSpace(payload.PhoneNumber)

	// Request context
	ctx := c.Context()

	// Database context
	db := h.db.WithContext(ctx)

	if err := ensurePhoneAvailable(db, phone, 0); err != nil {
		return response.Conflict(c, err.Error())
	}

	if err := ensureEmailAvailable(db, email); err != nil {
		return response.Conflict(c, err.Error())
	}

	tempPassword, err := utils.GenerateSecurePassword(12)
	if err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}

	user := models.User{
		Email:              email,
		FirstName:          payload.FirstName,
		LastName:           payload.LastName,
		Password:           tempPassword,
		IsActive:           true,
		MustChangePassword: true,
		Profile: models.Profile{
			PhoneNumber: phone,
			Title:       payload.Title,
			AppearanceSetting: map[string]any{
				"theme":        "light",
				"compactMode":  true,
				"largeText":    false,
				"sidebarState": true,
			},
		},
	}
	if err := user.EncryptPassword(); err != nil {
		return response.InternalServerError(c)
	}

	if err := linkProfileTitle(db, payload.Title); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := db.Create(&user).Error; err != nil {
		logs.Errorf("create user: %v", err)
		return response.InternalServerError(c)
	}

	audit.RecordHTTP(c, types.ModuleUser, types.ActionCreate, user.UID, types.UserContent, "created user "+user.Email, nil, user)

	notify.SendWelcomeCredentialsAsync(h.db, &user, tempPassword)

	return response.Created(c, fiber.Map{
		"user":     user,
		"notified": credentialsNotified(phone),
	})
}

// myProfile returns the current authenticated user with flattened permission
// codes and the super-user flag (which is omitted from User JSON elsewhere).
func (h *UserHandler) me(c fiber.Ctx) error {
	userID := middleware.GetUserIDFromContext(c)

	var user models.User
	err := h.db.WithContext(c.Context()).
		Preload("Roles.Permissions").
		Preload("Profile").
		First(&user, userID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "profile not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	seen := make(map[string]struct{})
	perms := make([]string, 0)
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			if _, ok := seen[p.Code]; ok {
				continue
			}
			seen[p.Code] = struct{}{}
			perms = append(perms, p.Code)
		}
	}

	sess := integrations.LiveSession()
	return response.OkDetail(c, fiber.Map{
		"user":                   user,
		"permissions":            perms,
		"isSuperUser":            user.IsSuperUser,
		"sessionIdleMinutes":     sess.IdleMinutes,
		"sessionIdleWarnSeconds": sess.WarnSeconds,
	})
}

// updateAppearanceSettings saves theme / compact / large-text / sidebar prefs.
func (h *UserHandler) updateAppearanceSettings(c fiber.Ctx) error {
	var payload appearanceSettingSchema

	// Parse and validate payload
	if err := c.Bind().Body(&payload); err != nil {
		logs.Error(err)
		return response.BadRequestBind(c, err)
	}

	// Request context
	ctx := c.Context()

	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Database context
	db := h.db.WithContext(ctx)

	userId := middleware.GetUserIDFromContext(c)
	var profile models.Profile
	if err := db.Where("UserID = ?", userId).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "profile not found")
		}
		logs.Errorf("get user profile : %s", err)
		return response.InternalServerError(c)
	}

	beforeUI := appearanceUISnapshot(profile.AppearanceSetting)

	if profile.AppearanceSetting == nil {
		profile.AppearanceSetting = map[string]any{}
	}
	profile.AppearanceSetting["theme"] = payload.Theme
	profile.AppearanceSetting["compactMode"] = payload.CompactMode
	profile.AppearanceSetting["largeText"] = payload.LargeText
	profile.AppearanceSetting["sidebarState"] = payload.SidebarState

	afterUI := appearanceUISnapshot(profile.AppearanceSetting)
	changes := audit.DropMetaKeys(audit.Diff(
		struct {
			AppearanceSetting map[string]any `json:"appearanceSettings"`
		}{beforeUI},
		struct {
			AppearanceSetting map[string]any `json:"appearanceSettings"`
		}{afterUI},
	))

	if err := db.Save(&profile).Error; err != nil {
		logs.Errorf("update appearance settings : %s", err)
		return response.InternalServerError(c)
	}

	// Audit only when theme/layout toggles actually change (not table prefs /
	// timestamps, and not no-op saves).
	if audit.Default != nil && len(changes) > 0 {
		desc := "updated appearance settings"
		entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, middleware.GetUserUIDFromContext(c), types.UserContent, desc)
		entry.Description = audit.EnrichDescription(desc, changes)
		entry.Changes = changes
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	return response.OkMessage(c, audit.UpdateMessageFromChanges(changes))
}

// updateTablePrefs merges per-table column order/visibility into AppearanceSetting.tablePrefs.
func (h *UserHandler) updateTablePrefs(c fiber.Ctx) error {
	var payload tablePrefsPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}
	ctx := c.Context()
	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	db := h.db.WithContext(ctx)
	userID := middleware.GetUserIDFromContext(c)
	var profile models.Profile
	if err := db.Where("UserID = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "profile not found")
		}
		return response.InternalServerError(c)
	}

	if profile.AppearanceSetting == nil {
		profile.AppearanceSetting = map[string]any{}
	}
	raw, _ := profile.AppearanceSetting["tablePrefs"].(map[string]any)
	if raw == nil {
		raw = map[string]any{}
	}
	raw[payload.TableID] = map[string]any{
		"order":  payload.Order,
		"hidden": payload.Hidden,
	}
	profile.AppearanceSetting["tablePrefs"] = raw

	if err := db.Save(&profile).Error; err != nil {
		logs.Errorf("update table prefs: %s", err)
		return response.InternalServerError(c)
	}
	if audit.Default != nil {
		entry := audit.AuditEntry(
			c, types.ModuleProfile, types.ActionUpdate, middleware.GetUserUIDFromContext(c),
			types.UserProfileContent, "updated table columns for "+payload.TableID,
		)
		_ = audit.Default.Record(c.Context(), nil, entry)
	}
	return response.OkDetail(c, raw[payload.TableID])
}

// updateDashboard stores the signed-in user's chart layout on the profile.
func (h *UserHandler) updateDashboard(c fiber.Ctx) error {
	var payload dashboardPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}
	ctx := c.Context()
	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	for i := range payload.Widgets {
		if err := payload.Widgets[i].Validate(ctx); err != nil {
			return response.UnprocessableEntity(c, err)
		}
		if payload.Widgets[i].Span != 2 {
			payload.Widgets[i].Span = 1
		}
	}

	db := h.db.WithContext(ctx)
	userID := middleware.GetUserIDFromContext(c)
	var profile models.Profile
	if err := db.Where("UserID = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "profile not found")
		}
		return response.InternalServerError(c)
	}
	if profile.AppearanceSetting == nil {
		profile.AppearanceSetting = map[string]any{}
	}
	widgets := make([]map[string]any, 0, len(payload.Widgets))
	for _, w := range payload.Widgets {
		widgets = append(widgets, map[string]any{
			"id": w.ID, "type": w.Type, "title": w.Title, "source": w.Source, "span": w.Span,
		})
	}
	profile.AppearanceSetting["dashboard"] = widgets
	if err := db.Save(&profile).Error; err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, widgets)
}

// appearanceUISnapshot copies only the user-facing appearance toggles so audit
// ignores tablePrefs and other preference blobs stored in the same JSON column.
func appearanceUISnapshot(src map[string]any) map[string]any {
	out := map[string]any{
		"theme":        "light",
		"compactMode":  true,
		"largeText":    false,
		"sidebarState": true,
	}
	if src == nil {
		return out
	}
	if v, ok := src["theme"]; ok {
		out["theme"] = v
	}
	if v, ok := src["compactMode"]; ok {
		out["compactMode"] = v
	}
	if v, ok := src["largeText"]; ok {
		out["largeText"] = v
	}
	if v, ok := src["sidebarState"]; ok {
		out["sidebarState"] = v
	}
	return out
}

// Update edits a user's name, status and profile (admin)
func (h *UserHandler) update(c fiber.Ctx) error {
	userId := c.Params("id")

	var payload userProfileUpdateSchema
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}

	// Request context
	ctx := c.Context()

	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Database context
	db := h.db.WithContext(ctx)

	var user models.User
	if err := db.Preload("Profile").Where("UID = ?", userId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	before := user

	phone := strings.TrimSpace(payload.PhoneNumber)
	if err := ensurePhoneAvailable(db, phone, user.ID); err != nil {
		return response.Conflict(c, err.Error())
	}

	// Update fields
	user.FirstName = payload.FirstName
	user.LastName = payload.LastName
	user.Profile.PhoneNumber = phone
	user.IsActive = payload.IsActive
	user.Profile.Title = payload.Title

	if err := linkProfileTitle(db, payload.Title); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := db.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		if response.IsDuplicate(err) {
			return response.Conflict(c, "a user with this phone number already exists")
		}
		logs.Errorf("update user : %s", err.Error())
		return response.InternalServerError(c)
	}

	if !payload.IsActive {
		middleware.InvalidateUserPermissions(user.UID)
	}

	desc := "user " + user.Email + " upated"
	changes := audit.DropAppearanceSettingKeys(audit.DropMetaKeys(audit.Diff(before, user)))
	entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, user.UID, types.UserContent, desc)
	if len(changes) > 0 {
		entry.Description = audit.EnrichDescription(desc, changes)
		entry.Changes = changes
	}
	if audit.Default != nil {
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	return response.OkMessage(c, audit.UpdateMessageFromChanges(changes))
}

// UpdateProfile allows authenticated user to update own profile
func (h *UserHandler) updateMyProfile(c fiber.Ctx) error {
	userID := middleware.GetUserIDFromContext(c)
	var payload userProfileUpdateSchema

	// Parse and validate payload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}

	// Request context
	ctx := c.Context()

	if err := payload.Validate(ctx); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Database context
	db := h.db.WithContext(ctx)

	var user models.User
	if err := db.Preload("Profile").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Errorf("get user : %s", err)
		return response.InternalServerError(c)
	}

	before := user

	phone := strings.TrimSpace(payload.PhoneNumber)
	if err := ensurePhoneAvailable(db, phone, user.ID); err != nil {
		return response.Conflict(c, err.Error())
	}

	// Update fields
	user.FirstName = payload.FirstName
	user.LastName = payload.LastName
	user.Profile.PhoneNumber = phone
	user.Profile.Title = payload.Title

	if err := linkProfileTitle(db, payload.Title); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := db.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		if response.IsDuplicate(err) {
			return response.Conflict(c, "a user with this phone number already exists")
		}
		logs.Errorf("failed to update user : %s", err)
		return response.InternalServerError(c)
	}

	changes := audit.DropAppearanceSettingKeys(audit.DropMetaKeys(audit.Diff(before, user)))
	if audit.Default != nil && len(changes) > 0 {
		desc := "user " + user.Email + " upated"
		entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, user.UID, types.UserContent, desc)
		entry.Description = audit.EnrichDescription(desc, changes)
		entry.Changes = changes
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	return response.OkMessage(c, audit.UpdateMessageFromChanges(changes))
}

// resetPassword issues a new temporary password and emails it to the user.
// Locked accounts must be unlocked first — password reset is not permitted
// while the account is locked.
func (h *UserHandler) resetPassword(c fiber.Ctx) error {
	// Database context
	db := h.db.WithContext(c.Context())

	var user models.User
	if err := db.Preload("Profile").First(&user, "UID = ?", c.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Errorf("failed to get user : %s", err)
		return response.InternalServerError(c)
	}

	if user.IsLocked {
		return response.BadRequest(c, "cannot reset password for a locked account; unlock the user first")
	}
	if !user.IsActive {
		return response.BadRequest(c, "cannot reset password for an inactive account")
	}

	tempPassword, err := utils.GenerateSecurePassword(12)
	if err != nil {
		logs.Errorf("failed to genereate password : %s", err)
		return response.InternalServerError(c)
	}

	oldHash := user.Password
	user.Password = tempPassword
	user.MustChangePassword = true
	if err := user.EncryptPassword(); err != nil {
		logs.Errorf("failed to encrypt user password : %s", err)
		return response.InternalServerError(c)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := models.RecordPasswordHash(tx, user.ID, oldHash); err != nil {
			return err
		}
		if err := tx.Model(&user).
			Select("Password", "MustChangePassword", "SessionVersion").
			Updates(map[string]any{
				"Password":           user.Password,
				"MustChangePassword": true,
				"SessionVersion":     gorm.Expr("SessionVersion + 1"),
			}).Error; err != nil {
			logs.Errorf("failed to update user : %s", err)
			return err
		}
		// Revoke every active session for this user.
		if err := tx.Where("Subject = ?", user.UID).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		middleware.InvalidateUserPermissions(user.UID)
		return nil
	})
	if err != nil {
		logs.Errorf("reset password for %s: %v", c.Params("id"), err)
		return response.InternalServerError(c)
	}
	user.SessionVersion++
	middleware.SetSessionVersion(user.UID, user.SessionVersion, middleware.SessionVersionTTL)

	notify.SendPasswordResetCredentialsAsync(h.db, &user, tempPassword)

	if audit.Default != nil {
		entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, user.UID, types.UserContent, "password reseted for "+user.Email)
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	phone := ""
	if user.Profile.PhoneNumber != "" {
		phone = user.Profile.PhoneNumber
	}
	return response.OkDetail(c, fiber.Map{
		"notified": credentialsNotified(phone),
		"message":  "password reset; temporary password sent by email and SMS when configured",
	})
}

// unlock clears a locked account so the user can sign in again after too many
// failed login attempts.
func (h *UserHandler) unlock(c fiber.Ctx) error {
	userId := c.Params("id")
	ctx := c.Context()

	// Database context
	db := h.db.WithContext(ctx)

	var user models.User
	if err := db.Where("UID = ?", userId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Errorf("failed to get user :%s : %s", userId, err.Error())
		return response.InternalServerError(c)
	}

	if !user.IsLocked {
		return response.BadRequest(c, "account is not locked")
	}

	// Reset lockout
	user.Attempts = 0
	user.IsLocked = false

	if err := db.Select("Attempts", "IsLocked").Save(&user).Error; err != nil {
		logs.Errorf("unlock user %s: %v", userId, err)
		return response.InternalServerError(c)
	}

	middleware.InvalidateUserPermissions(user.UID)
	if audit.Default != nil {
		entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, user.UID, types.UserContent, "unlocked user "+user.Email)
		_ = audit.Default.Record(ctx, nil, entry)
	}
	return response.OkMessage(c, "account unlocked")
}

// ReplaceRoles replaces all of a user's roles.
func (h *UserHandler) replaceUserRoles(c fiber.Ctx) error {
	var payload replaceRoles
	userId := c.Params("id")

	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequestBind(c, err)
	}

	if err := payload.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}

	// Database context
	db := h.db.WithContext(c.Context())

	var user models.User
	if err := db.Preload("Roles").Where("UID = ?", userId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Errorf("failed to get user :%s :%s", userId, err.Error())
		return response.InternalServerError(c)
	}
	beforeNames := roleNames(user.Roles)

	var roles []models.Role
	if len(payload.RoleIDs) > 0 {
		if err := db.Where("ID IN ?", payload.RoleIDs).Find(&roles).Error; err != nil {
			logs.Errorf("failed to get roles : %s", err)
			return response.InternalServerError(c)
		}
		if len(roles) != len(payload.RoleIDs) {
			return response.BadRequest(c, "oneor more role Ids are invalid")
		}
	}

	if len(roles) == 0 {
		if err := db.Where("UserID = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
			logs.Errorf("faield to get user roles : %s", err)
			return response.InternalServerError(c)
		}
	} else if err := db.Model(&user).Association("Roles").Replace(&roles); err != nil {
		logs.Errorf("failed to replace user :%s roles %v", user.UID, err)
		return response.InternalServerError(c)
	}

	afterNames := roleNames(roles)
	summary := audit.SummarizeSet("Roles", beforeNames, afterNames)
	if audit.Default != nil {
		desc := fmt.Sprintf("roles updated for user %s", user.Email)
		if summary != "" {
			desc = desc + " — " + summary
		}
		entry := audit.AuditEntry(c, types.ModuleUser, types.ActionUpdate, user.UID, types.UserContent, desc)
		entry.Metadata = map[string]any{"before": beforeNames, "after": afterNames}
		_ = audit.Default.Record(c.Context(), nil, entry)
	}

	middleware.InvalidateUserPermissions(user.UID)
	return response.OkMessage(c, audit.UpdateMessageFromSummary(summary))
}

func roleNames(roles []models.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if name := strings.TrimSpace(r.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// delete hard-deletes a user who has never signed in (typo email). After
// first login, deactivate instead — workflow tasks and audit keep the row.
func (h *UserHandler) delete(c fiber.Ctx) error {
	uid := fiber.Params[string](c, "id")
	callerID := middleware.GetUserIDFromContext(c)
	db := h.db.WithContext(c.Context())

	var user models.User
	if err := db.First(&user, "UID = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	if user.ID == callerID {
		return response.Conflict(c, "cannot delete your own account")
	}
	if user.IsSuperUser {
		return response.Conflict(c, "super-user accounts cannot be deleted")
	}
	if user.LastLogin != nil {
		return response.Conflict(c, "this account has already signed in; deactivate it instead of deleting")
	}
	if reason, err := userDeleteBlockReason(db, user.ID); err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	} else if reason != "" {
		return response.Conflict(c, reason)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		id, uid := user.ID, user.UID
		if err := tx.Exec("DELETE FROM [PasswordHistory] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [UserRole] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [UserOTPChallenge] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [Profile] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [WorkflowInitiatorPoolUsers] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [WorkflowNotifyUsers] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [NodeOperatorUser] WHERE [UserID] = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [RefreshToken] WHERE [Subject] = ?", uid).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM [ApprovalSubstitute] WHERE [PrincipalUserID] = ? OR [DelegateUserID] = ?", id, id).Error; err != nil {
			return err
		}
		return tx.Exec("DELETE FROM [User] WHERE [ID] = ?", id).Error
	})
	if err != nil {
		if errors.Is(err, constants.ErrUserInUse) {
			return response.Conflict(c, "this account has already signed in; deactivate it instead of deleting")
		}
		logs.Error(err)
		return response.InternalServerError(c)
	}

	middleware.InvalidateUserPermissions(user.UID)
	audit.RecordHTTP(c, types.ModuleUser, types.ActionDelete, user.UID, types.UserContent, "deleted user "+user.Email, user, nil)
	return response.Deleted(c)
}

func attachUserCanDelete(users []models.User, callerID uint) {
	for i := range users {
		users[i].CanDelete = users[i].LastLogin == nil && !users[i].IsSuperUser && users[i].ID != callerID
	}
}

func userDeleteBlockReason(db *gorm.DB, userID uint) (string, error) {
	checks := []struct {
		model  any
		column string
		msg    string
	}{
		{&models.Receipt{}, "CreatedByID", "user has created receipts and cannot be deleted"},
		{&models.ProcessInstance{}, "CreatedByID", "user has workflow activity and cannot be deleted"},
		{&models.Title{}, "CreatedByID", "user created catalogue titles and cannot be deleted"},
		{&models.Task{}, "UserID", "user has workflow tasks and cannot be deleted"},
		{&models.Event{}, "UserID", "user has approval history and cannot be deleted"},
		{&models.Attachment{}, "UploadedByID", "user uploaded attachments and cannot be deleted"},
	}
	for _, c := range checks {
		var n int64
		if err := db.Model(c.model).Where(c.column+" = ?", userID).Count(&n).Error; err != nil {
			return "", err
		}
		if n > 0 {
			return c.msg, nil
		}
	}
	return "", nil
}

// linkProfileTitle ensures a non-empty profile title exists in the Title
// catalogue and marks it as referenced (HasData) so it cannot be hard-deleted.
func linkProfileTitle(db *gorm.DB, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	var title models.Title
	if err := db.Where("Name = ?", name).First(&title).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("job title %q is not in the catalogue; pick an organisation title (not a workflow role) or create one under Titles first", name)
		}
		return err
	}
	return title.UpdateHasData(db)
}

// ensurePhoneAvailable rejects non-empty phone numbers already used by another
// profile (OTP delivery requires a 1:1 phone → user mapping). Empty is allowed
// on many profiles. excludeUserID skips the caller's own row on update (0 = none).
func ensurePhoneAvailable(db *gorm.DB, phone string, excludeUserID uint) error {
	if phone == "" {
		return nil
	}
	q := db.Model(&models.Profile{}).Where("PhoneNumber = ?", phone)
	if excludeUserID != 0 {
		q = q.Where("UserID <> ?", excludeUserID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("a user with this phone number already exists")
	}
	return nil
}

func credentialsNotified(phone string) bool {
	if integrations.Default == nil {
		return false
	}
	if integrations.Default.Mail().Enabled {
		return true
	}
	return integrations.Default.SMS().Enabled && strings.TrimSpace(phone) != ""
}

// ensureEmailAvailable rejects non-empty email already used by another user
func ensureEmailAvailable(db *gorm.DB, email string) error {
	q := db.Model(&models.User{}).Where("Email = ?", email)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("a user with this email already exists")
	}
	return nil
}
