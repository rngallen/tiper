package auth

import (
	"errors"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/permissions"
	"dfms/pkg/types"

	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
)

const (
	defaultSuperUserEmail    = "ngallen4@gmail.com"
	defaultSuperUserPassword = "Admin@2026"
	defaultAdminPhoneNumber  = "255765889960"

	roleAdmin       = "Admin"
	roleMD          = "Managing Director"
	roleCCS         = "Customer Care Supervisor"
	roleCSL         = "Stock Accountant"
	roleFCC         = "Finance Credit Controller"
	roleOpsMgr      = "Operation Manager"
	roleMOP         = "MOP Superintendent"
	roleBilling     = "Billing Officer"
	roleGantry      = "Gantry Supervisor"
	roleGantryClerk = "Gantry Dispatch Clerk"

	titleAdministrator = "Administrator"
)

// Default roles on migrate up — one set per process so a person can initiate
// or approve on some workflows and not others. Operator assignment is by role.
//
//	Admin
//	CFO, CEO — view-only across DFMS (not assigned on any workflow)
//	Customer Care, Stock Accountant, Finance, Managing Director, Billing,
//	Gantry, and operations roles used by the seeded TIPER processes
//
// Roles are not protected by name. Delete is allowed when unused (no UserRole,
// no NodeOperatorRole). Only the Admin user is seeded; other users are created
// in the UI and added to the matching initiator pool / operator role.

// permissionCatalog maps each permission code to a human description. The
// module is derived from the code prefix. Broad {module}.{action} codes stay
// in the catalogue for Admin / emergency use; clerks get document codes.
var permissionCatalog = map[string]string{
	permissions.UsersRead:             "View users",
	permissions.UsersManage:           "Full user administration",
	permissions.RolesRead:             "View roles and permissions",
	permissions.RolesManage:           "Full role administration",
	permissions.TitlesRead:            "View titles",
	permissions.TitlesCreate:          "Create titles",
	permissions.TitlesUpdate:          "Update titles",
	permissions.TitlesDelete:          "Delete titles",
	permissions.TitlesManage:          "Full title administration",
	permissions.WorkflowRead:          "View workflow definitions and progress",
	permissions.WorkflowTasks:         "Act on workflow tasks",
	permissions.WorkflowManage:        "Manage workflow definitions",
	permissions.AuditRead:             "View audit trail",
	permissions.ReportsRead:           "View and export reports",
	permissions.SettingsRead:          "View system settings",
	permissions.SettingsManage:        "Manage system settings",
	permissions.JobsRun:               "Run scheduled background jobs on demand",
	permissions.MasterdataRead:        "View master data",
	permissions.MasterdataCreate:      "Create master data",
	permissions.MasterdataUpdate:      "Update master data",
	permissions.MasterdataDelete:      "Delete master data",
	permissions.MasterdataManage:      "Full master data administration",
	permissions.CustomersRead:         "View customers",
	permissions.CustomersCreate:       "Create customers",
	permissions.CustomersUpdate:       "Update customers",
	permissions.CustomersManage:       "Full customer administration (no delete — history is kept)",
	permissions.InventoryRead:         "View all stock documents and balances",
	permissions.InventoryCreate:       "Broad: create any stock document (prefer receipts / ITT / zerolization)",
	permissions.InventoryUpdate:       "Broad: update stock documents and operational captures (dips, events)",
	permissions.InventorySubmit:       "Broad: submit any stock document (prefer document codes)",
	permissions.InventoryManage:       "Full inventory administration",
	permissions.InventoryBalances:     "Read customer stock balances only",
	permissions.ReceiptsRead:          "View vessel receipts",
	permissions.ReceiptsCreate:        "Create vessel receipts",
	permissions.ReceiptsUpdate:        "Update draft vessel receipts",
	permissions.ReceiptsSubmit:        "Submit vessel receipts for approval",
	permissions.ReceiptsManage:        "Full vessel receipt administration",
	permissions.ITTRead:               "View in-tank transfers",
	permissions.ITTCreate:             "Create in-tank transfers",
	permissions.ITTUpdate:             "Update draft in-tank transfers",
	permissions.ITTSubmit:             "Submit in-tank transfers for approval",
	permissions.ITTManage:             "Full in-tank transfer administration",
	permissions.ZerolRead:             "View zerolization transfers",
	permissions.ZerolCreate:           "Create zerolization transfers",
	permissions.ZerolUpdate:           "Update draft zerolization",
	permissions.ZerolSubmit:           "Submit zerolization for approval",
	permissions.ZerolManage:           "Full zerolization administration",
	permissions.HoldRead:              "View financial hold releases",
	permissions.HoldCreate:            "Create financial hold releases",
	permissions.HoldUpdate:            "Update draft financial hold releases",
	permissions.HoldSubmit:            "Submit financial hold releases for approval",
	permissions.HoldManage:            "Full financial hold release administration",
	permissions.ChangeOfServiceRead:   "View change of service",
	permissions.ChangeOfServiceCreate: "Create change of service",
	permissions.ChangeOfServiceUpdate: "Update draft change of service",
	permissions.ChangeOfServiceSubmit: "Submit change of service for approval",
	permissions.ChangeOfServiceManage: "Full change of service administration",
	permissions.BillingRead:           "View billing runs",
	permissions.BillingCreate:         "Create billing documents",
	permissions.BillingUpdate:         "Update billing documents",
	permissions.BillingSubmit:         "Submit billing runs for approval",
	permissions.BillingRun:            "Run billing engines",
	permissions.BillingManage:         "Full billing administration",
	permissions.PricesRead:            "View fee and price batches",
	permissions.PricesCreate:          "Create fee and price batches",
	permissions.PricesUpdate:          "Update draft fee and price batches",
	permissions.PricesSubmit:          "Submit fee and price batches for approval",
	permissions.PricesManage:          "Full price administration (create, update, submit — no delete)",
	permissions.EwuraRead:             "View EWURA licenses",
	permissions.EwuraManage:           "Sync and manage EWURA licenses",
	permissions.SageRead:              "View Sage connection",
	permissions.SagePost:              "Post billing to Sage",
	permissions.SageManage:            "Manage Sage integration",
	permissions.OrdersRead:            "View all gantry and terminal documents",
	permissions.OrdersCreate:          "Broad: create any gantry/terminal document (prefer ILR / pump-over / …)",
	permissions.OrdersSubmit:          "Broad: submit any gantry/terminal document (prefer document codes)",
	permissions.OrdersManage:          "Full orders administration (all document types and gantry complete)",
	permissions.ILRRead:               "View internal loading requests and truck lines",
	permissions.ILRCreate:             "Create internal loading requests",
	permissions.ILRUpdate:             "Update draft internal loading requests",
	permissions.ILRSubmit:             "Submit internal loading requests for approval",
	permissions.ILRComplete:           "Post loaded quantity on gantry completion",
	permissions.ILRManage:             "Full internal loading administration",
	permissions.PumpOverRead:          "View pump-over requests",
	permissions.PumpOverCreate:        "Create pump-over requests",
	permissions.PumpOverUpdate:        "Update draft pump-over requests",
	permissions.PumpOverSubmit:        "Submit pump-over requests for approval",
	permissions.PumpOverManage:        "Full pump-over request administration",
	permissions.PumpReportRead:        "View pump-over execution reports",
	permissions.PumpReportCreate:      "Create pump-over execution reports",
	permissions.PumpReportUpdate:      "Update draft pump-over reports",
	permissions.PumpReportSubmit:      "Submit pump-over reports for approval",
	permissions.PumpReportManage:      "Full pump-over report administration",
	permissions.CompRead:              "View compartmentalization documents",
	permissions.CompCreate:            "Create compartmentalization documents",
	permissions.CompUpdate:            "Update draft compartmentalization",
	permissions.CompSubmit:            "Submit compartmentalization for gantry approval",
	permissions.CompManage:            "Full compartmentalization administration",
	permissions.AmendmentRead:         "View loading amendments",
	permissions.AmendmentCreate:       "Create loading amendments",
	permissions.AmendmentUpdate:       "Update draft loading amendments",
	permissions.AmendmentSubmit:       "Submit loading amendments for approval",
	permissions.AmendmentManage:       "Full loading amendment administration",
}

// defaultTitles is the organisation job-title catalogue (not workflow roles).
var defaultTitles = []string{
	"Managing Director",
	"Customer Care Supervisor",
	"Stock Accountant",
	"Finance Credit Controller",
	"Operation Manager",
	"MOP Superintendent",
	"Billing Officer",
	"Gantry Supervisor",
	"Gantry Dispatch Clerk",
	titleAdministrator,
}

// LoadDefault seeds permissions, roles, the bootstrap Admin user, then
// job titles (titles require a CreatedBy user for the FK).
func LoadDefault(db *gorm.DB) error {
	if err := seedPermissions(db); err != nil {
		return err
	}
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedSuperUser(db); err != nil {
		return err
	}
	return seedTitles(db)
}

func seedPermissions(db *gorm.DB) error {
	var result *multierror.Error
	for code, desc := range permissionCatalog {
		module := code
		if i := strings.IndexByte(code, '.'); i > 0 {
			module = code[:i]
		}

		perm := models.Permission{Code: code, Description: desc, Module: module}
		var permission models.Permission
		if err := db.Where(models.Permission{Code: perm.Code}).
			Assign(models.Permission{Description: desc, Module: module}).
			FirstOrCreate(&permission).Error; err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

func seedRoles(db *gorm.DB) error {
	allPerms, err := loadAllPermissions(db)
	if err != nil {
		return err
	}

	catalogCodes := make([]string, 0, len(permissionCatalog))
	for code := range permissionCatalog {
		catalogCodes = append(catalogCodes, code)
	}
	if err := seedRoleWithPerms(db, roleAdmin, "Full system access", types.RoleSystem, filterPermissions(allPerms, catalogCodes...)); err != nil {
		return err
	}

	if err := seedRoleWithPerms(db, roleCCS, "Customer Care — receipts, ITT, ILR, pump-over requests, amendments", types.RoleTerminal,
		filterPermissions(allPerms, joinCodes(
			[]string{
				permissions.InventoryRead, permissions.CustomersRead, permissions.CustomersCreate, permissions.CustomersUpdate,
				permissions.MasterdataRead, permissions.MasterdataCreate, permissions.MasterdataUpdate,
				permissions.WorkflowTasks, permissions.WorkflowRead, permissions.ReportsRead,
			},
			docWrite(permissions.ReceiptsRead, permissions.ReceiptsCreate, permissions.ReceiptsUpdate, permissions.ReceiptsSubmit),
			docWrite(permissions.ITTRead, permissions.ITTCreate, permissions.ITTUpdate, permissions.ITTSubmit),
			docWrite(permissions.ILRRead, permissions.ILRCreate, permissions.ILRUpdate, permissions.ILRSubmit),
			docWrite(permissions.PumpOverRead, permissions.PumpOverCreate, permissions.PumpOverUpdate, permissions.PumpOverSubmit),
			docWrite(permissions.AmendmentRead, permissions.AmendmentCreate, permissions.AmendmentUpdate, permissions.AmendmentSubmit),
		)...)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleCSL, "CSL / stock accountant — receipts, ITT, zerolization", types.RoleTerminal,
		filterPermissions(allPerms, joinCodes(
			[]string{
				permissions.InventoryRead, permissions.InventoryBalances, permissions.CustomersRead,
				permissions.MasterdataRead, permissions.WorkflowTasks, permissions.WorkflowRead, permissions.ReportsRead,
			},
			docWrite(permissions.ReceiptsRead, permissions.ReceiptsCreate, permissions.ReceiptsUpdate, permissions.ReceiptsSubmit),
			docWrite(permissions.ITTRead, permissions.ITTCreate, permissions.ITTUpdate, permissions.ITTSubmit),
			docWrite(permissions.ZerolRead, permissions.ZerolCreate, permissions.ZerolUpdate, permissions.ZerolSubmit),
		)...)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleFCC, "Finance credit controller — review stock and billing; release financial hold", types.RoleFinance,
		filterPermissions(allPerms, joinCodes(
			[]string{
				permissions.InventoryRead, permissions.OrdersRead, permissions.BillingRead,
				permissions.PricesRead, permissions.CustomersRead, permissions.MasterdataRead,
				permissions.WorkflowTasks, permissions.WorkflowRead, permissions.ReportsRead,
			},
			docWrite(permissions.HoldRead, permissions.HoldCreate, permissions.HoldUpdate, permissions.HoldSubmit),
		)...)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleOpsMgr, "Operations manager — review pump-over reports and stock", types.RoleTerminal,
		filterPermissions(allPerms, permissions.InventoryRead, permissions.PumpReportRead, permissions.PumpOverRead,
			permissions.WorkflowTasks, permissions.WorkflowRead, permissions.ReportsRead)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleMOP, "MOP superintendent — pump-over execution reports (not requests)", types.RoleTerminal,
		filterPermissions(allPerms, joinCodes(
			[]string{
				permissions.ReceiptsRead, permissions.ReceiptsUpdate, permissions.PumpOverRead,
				permissions.WorkflowTasks, permissions.ReportsRead,
			},
			docWrite(permissions.PumpReportRead, permissions.PumpReportCreate, permissions.PumpReportUpdate, permissions.PumpReportSubmit),
		)...)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleBilling, "Billing officer — fee batches and billing runs (no stock edits)", types.RoleBilling,
		filterPermissions(allPerms, permissions.BillingRead, permissions.BillingCreate, permissions.BillingUpdate,
			permissions.BillingSubmit, permissions.BillingRun, permissions.PricesRead, permissions.PricesCreate,
			permissions.PricesUpdate, permissions.PricesSubmit, permissions.CustomersRead, permissions.SageRead,
			permissions.WorkflowTasks, permissions.ReportsRead,
			permissions.ChangeOfServiceRead, permissions.ChangeOfServiceCreate, permissions.ChangeOfServiceUpdate, permissions.ChangeOfServiceSubmit)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleGantry, "Gantry supervisor — complete loads; approve compartmentalization", types.RoleGantry,
		filterPermissions(allPerms, permissions.ILRRead, permissions.ILRComplete, permissions.CompRead,
			permissions.WorkflowTasks, permissions.ReportsRead)); err != nil {
		return err
	}
	if err := seedRoleWithPerms(db, roleGantryClerk, "Gantry dispatch clerk — compartmentalize trucks (not ILR create)", types.RoleGantry,
		filterPermissions(allPerms, joinCodes(
			[]string{permissions.ILRRead, permissions.WorkflowTasks, permissions.ReportsRead},
			docWrite(permissions.CompRead, permissions.CompCreate, permissions.CompUpdate, permissions.CompSubmit),
		)...)); err != nil {
		return err
	}
	return seedRoleWithPerms(db, roleMD, "Managing Director — final approver", types.RoleFinance,
		filterPermissions(allPerms, executiveReadCodes()...))
}

func docWrite(codes ...string) []string {
	return codes
}

func joinCodes(groups ...[]string) []string {
	var n int
	for _, g := range groups {
		n += len(g)
	}
	out := make([]string, 0, n)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// executiveReadCodes is every module's read permission. CFO/CEO are not
// workflow operators — they do not receive workflow.tasks.
func executiveReadCodes() []string {
	return []string{
		permissions.UsersRead,
		permissions.RolesRead,
		permissions.TitlesRead,
		permissions.MasterdataRead,
		permissions.CustomersRead,
		permissions.InventoryRead,
		permissions.OrdersRead,
		permissions.BillingRead,
		permissions.PricesRead,
		permissions.EwuraRead,
		permissions.WorkflowRead,
		permissions.AuditRead,
		permissions.ReportsRead,
		permissions.SettingsRead,
	}
}

func loadAllPermissions(db *gorm.DB) ([]models.Permission, error) {
	var perms []models.Permission
	if err := db.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func filterPermissions(all []models.Permission, codes ...string) []models.Permission {
	want := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		want[c] = struct{}{}
	}
	out := make([]models.Permission, 0, len(codes))
	for _, p := range all {
		if _, ok := want[p.Code]; ok {
			out = append(out, p)
		}
	}
	return out
}

func ensureRole(db *gorm.DB, name, description string) (models.Role, error) {
	var role models.Role
	err := db.Where("Name = ?", name).FirstOrCreate(&role, models.Role{
		Name:        name,
		Description: description,
	}).Error
	return role, err
}

func seedRoleWithPerms(db *gorm.DB, name, description string, family types.RoleFamily, perms []models.Permission) error {
	role, err := ensureRole(db, name, description)
	if err != nil {
		return err
	}
	if err := db.Model(&role).Updates(map[string]any{
		"Description": description,
		"Category":    types.NormalizeRoleFamily(string(family)),
	}).Error; err != nil {
		return err
	}
	return db.Model(&role).Association("Permissions").Replace(perms)
}

func seedSuperUser(db *gorm.DB) error {
	var adminRole models.Role
	if err := db.Where("Name = ?", roleAdmin).First(&adminRole).Error; err != nil {
		return err
	}

	var n int64
	if err := db.Model(&models.User{}).Where("Email = ?", defaultSuperUserEmail).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	user := models.User{
		Email:              defaultSuperUserEmail,
		FirstName:          "System",
		LastName:           "Administrator",
		Password:           defaultSuperUserPassword,
		IsActive:           true,
		IsSuperUser:        true,
		MustChangePassword: true,
		Profile: models.Profile{
			Title:       titleAdministrator,
			PhoneNumber: defaultAdminPhoneNumber,
			AppearanceSetting: map[string]any{
				"theme":        "light",
				"compactMode":  true,
				"largeText":    false,
				"sidebarState": false,
			},
		},
		Roles: []models.Role{adminRole},
	}
	if err := user.EncryptPassword(); err != nil {
		return err
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	logs.Warnf("Seeded bootstrap super-user %s with password %q — change it immediately after first login",
		defaultSuperUserEmail, defaultSuperUserPassword)
	return nil
}

// seedTitles ensures the organisation job-title catalogue exists.
// These are job titles (e.g. CFO), not workflow step names.
// Must run after at least one user exists (Title.CreatedBy FK).
func seedTitles(db *gorm.DB) error {
	creatorID, err := seedCreatorUserID(db)
	if err != nil {
		return err
	}

	for _, name := range defaultTitles {
		var title models.Title
		err := db.Where("Name = ?", name).First(&title).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&models.Title{Name: name, CreatedByID: creatorID}).Error; err != nil {
			return err
		}
	}

	return markTitleInUse(db, titleAdministrator)
}

func seedCreatorUserID(db *gorm.DB) (uint, error) {
	var user models.User
	if err := db.Where("Email = ?", defaultSuperUserEmail).Take(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

func markTitleInUse(db *gorm.DB, name string) error {
	var title models.Title
	if err := db.Where("Name = ?", name).First(&title).Error; err != nil {
		return err
	}
	return title.UpdateHasData(db)
}
