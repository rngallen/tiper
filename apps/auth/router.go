// Package auth exposes the /api/v1/auth HTTP surface of TIPER DFMS:
// login with OTP-based MFA, PASETO access/refresh token issuance via HttpOnly
// cookies, self-service profile endpoints, and administration of users, roles,
// permissions and titles.
package auth

import (
	"dfms/apps/auth/middleware"
	"dfms/pkg/db"
	"dfms/pkg/permissions"

	"github.com/gofiber/fiber/v3"
)

// AuthRouter registers all /api/v1/auth routes on the application.
func AuthRouter(router fiber.Router) {
	authHandler := NewAuthHandler(db.Db)
	userHandler := NewUserHandler(db.Db)
	roleHandler := NewRoleHandler(db.Db)
	permissionHandler := NewPermissionHandler(db.Db)
	titleHandler := NewTitleHandler(db.Db)

	// Base group: /api/v1/auth
	authGroup := router.Group("/api/v1/auth")

	// ── Public ──
	authGroup.Get("/csrf", authHandler.csrfToken).Name("auth.csrf")
	authGroup.Post("/login", authHandler.login).Name("auth.login")
	authGroup.Post("/mfa/verify", middleware.OTPVerifyMiddleware(), authHandler.mfaVerify).Name("auth.mfa.verify")
	authGroup.Post("/mfa/resend", middleware.OTPVerifyMiddleware(), authHandler.mfaResend).Name("auth.mfa.resend")
	authGroup.Post("/refresh", middleware.PasetoRefreshMiddleware(), authHandler.refreshToken).Name("auth.refresh")

	// ── Authenticated ──
	authenticated := authGroup.Group("", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())

	// Session management
	authenticated.Post("/logout", authHandler.logout).Name("auth.logout")
	authenticated.Post("/session/touch", authHandler.touchSession).Name("auth.session.touch")

	// Self-service profile
	profile := authenticated.Group("/profile")
	profile.Get("", userHandler.me).Name("auth.profile.get")
	profile.Patch("", userHandler.updateMyProfile).Name("auth.profile.update")
	profile.Patch("/apperance-settings", userHandler.updateAppearanceSettings).Name("auth.profile.update.appearance")
	profile.Put("/table-prefs", userHandler.updateTablePrefs).Name("auth.profile.update.tablePrefs")
	profile.Put("/dashboard", userHandler.updateDashboard).Name("auth.profile.update.dashboard")
	authenticated.Post("/change-password", authHandler.changePassword).Name("auth.password.change")

	// Compact pickers — users.manage (assign roles) or workflow.manage (initiator pool).
	authenticated.Get("/roles/options",
		middleware.PermissionMiddleware(permissions.UsersRead, permissions.UsersManage, permissions.RolesRead, permissions.RolesManage),
		roleHandler.options).Name("auth.roles.options")
	authenticated.Get("/users/search",
		middleware.PermissionMiddleware(permissions.UsersManage, permissions.WorkflowManage, permissions.WorkflowTasks),
		userHandler.searchOptions).Name("auth.users.search")

	// ── Users administration ──
	users := authenticated.Group("/users")
	usersRead := middleware.PermissionMiddleware(permissions.UsersRead, permissions.UsersManage)
	usersWrite := middleware.PermissionMiddleware(permissions.UsersManage)
	users.Get("", usersRead, userHandler.list).Name("auth.users.list")
	users.Post("", usersWrite, userHandler.create).Name("auth.users.create")
	users.Patch("/:id<string>", usersWrite, userHandler.update).Name("auth.users.update")
	users.Delete("/:id<string>", usersWrite, userHandler.delete).Name("auth.users.delete")
	users.Post("/:id<string>/unlock", usersWrite, userHandler.unlock).Name("auth.users.unlock")
	users.Post("/:id<string>/reset-password", usersWrite, userHandler.resetPassword).Name("auth.users.resetPassword")
	users.Put("/:id<string>/roles", usersWrite, userHandler.replaceUserRoles).Name("auth.users.roles.replace")

	// ── Roles & permissions ──
	roles := authenticated.Group("/roles")
	rolesRead := middleware.PermissionMiddleware(permissions.RolesRead, permissions.RolesManage)
	rolesWrite := middleware.PermissionMiddleware(permissions.RolesManage)
	roles.Get("", rolesRead, roleHandler.list).Name("auth.roles.list")
	roles.Post("", rolesWrite, roleHandler.create).Name("auth.roles.create")
	roles.Get("/:id<int>", rolesRead, roleHandler.get).Name("auth.roles.get")
	roles.Patch("/:id<int>", rolesWrite, roleHandler.update).Name("auth.roles.update")
	roles.Delete("/:id<int>", rolesWrite, roleHandler.delete).Name("auth.roles.delete")
	roles.Put("/:id<int>/permissions", rolesWrite, roleHandler.replaceRolePermissions).Name("auth.roles.permissions.replace")

	perms := authenticated.Group("/permissions")
	perms.Get("", rolesRead, permissionHandler.list).Name("auth.permissions.list")

	// ── Titles ──
	titles := authenticated.Group("/titles")
	// Catalogue list is available to any signed-in user (e.g. profile title picker).
	titles.Get("", titleHandler.list).Name("auth.titles.list")
	titles.Post("", middleware.PermissionMiddleware(permissions.TitlesCreate, permissions.TitlesManage), titleHandler.create).Name("auth.titles.create")
	titles.Patch("/:id<string>", middleware.PermissionMiddleware(permissions.TitlesUpdate, permissions.TitlesManage), titleHandler.update).Name("auth.titles.update")
	titles.Delete("/:id<string>", middleware.PermissionMiddleware(permissions.TitlesDelete, permissions.TitlesManage), titleHandler.delete).Name("auth.titles.delete")
}
