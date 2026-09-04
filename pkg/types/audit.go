package types

// Module represents the domain/entity being audited
type Module string

// Action represents the operation performed
type Action string

const (
	ModuleAuth      Module = "Auth"
	ModuleUser      Module = "User"
	ModuleProfile   Module = "Profile"
	ModuleRole      Module = "Role"
	ModuleTitle     Module = "Title"
	ModuleCustomer  Module = "Customer"
	ModuleInventory Module = "Inventory"
	ModuleBilling   Module = "Billing"
	ModuleEwura     Module = "Ewura"
	ModuleOrders    Module = "Orders"
	ModuleReports   Module = "Reports"
	ModuleWorkflow  Module = "Workflow"
	ModuleSettings  Module = "Settings"
	ModuleSystem    Module = "System"
)

// Audit action verbs.
const (
	ActionLogin        Action = "login"
	ActionRefreshToken Action = "refresh_token"
	ActionLogout       Action = "logout"
	ActionCreate       Action = "create"
	ActionUpdate       Action = "update"
	ActionDelete       Action = "delete"
	ActionApprove      Action = "approve"
	ActionReject       Action = "reject"
	ActionInitiate     Action = "initiate"
	ActionDispatch     Action = "dispatch"
	ActionSync         Action = "sync"
)
