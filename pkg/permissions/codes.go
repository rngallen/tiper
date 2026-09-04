// Package permissions defines RBAC permission codes used by HTTP routes.
//
// Codes are {module}.{action} or {module}.{resource}.{action}.
// Hierarchy (see Satisfies):
//
//	exact code
//	{module}.{resource}.manage  — every action on that document
//	{module}.{action}           — same action across the module (legacy / admin)
//	{module}.manage             — every action in the module (Admin)
//
// Assign clerks document codes (orders.pumpreport.submit), not module-wide
// orders.submit / inventory.submit. Super-users bypass all checks.
package permissions

const (
	UsersRead   = "users.read"
	UsersManage = "users.manage"
)

const (
	RolesRead   = "roles.read"
	RolesManage = "roles.manage"
)

const (
	TitlesRead   = "titles.read"
	TitlesCreate = "titles.create"
	TitlesUpdate = "titles.update"
	TitlesDelete = "titles.delete"
	TitlesManage = "titles.manage"
)

const (
	WorkflowRead   = "workflow.read"
	WorkflowTasks  = "workflow.tasks"
	WorkflowManage = "workflow.manage"
)

const AuditRead = "audit.read"

const ReportsRead = "reports.read"

const (
	SettingsRead   = "settings.read"
	SettingsManage = "settings.manage"
)

const JobsRun = "jobs.run"

const (
	MasterdataRead   = "masterdata.read"
	MasterdataCreate = "masterdata.create"
	MasterdataUpdate = "masterdata.update"
	MasterdataDelete = "masterdata.delete"
	MasterdataManage = "masterdata.manage"
)

const (
	CustomersRead   = "customers.read"
	CustomersCreate = "customers.create"
	CustomersUpdate = "customers.update"
	CustomersManage = "customers.manage"
)

const (
	InventoryRead     = "inventory.read"
	InventoryCreate   = "inventory.create"
	InventoryUpdate   = "inventory.update"
	InventorySubmit   = "inventory.submit"
	InventoryManage   = "inventory.manage"
	InventoryBalances = "inventory.balances"
)

const (
	ReceiptsRead   = "inventory.receipts.read"
	ReceiptsCreate = "inventory.receipts.create"
	ReceiptsUpdate = "inventory.receipts.update"
	ReceiptsSubmit = "inventory.receipts.submit"
	ReceiptsManage = "inventory.receipts.manage"

	ITTRead   = "inventory.itt.read"
	ITTCreate = "inventory.itt.create"
	ITTUpdate = "inventory.itt.update"
	ITTSubmit = "inventory.itt.submit"
	ITTManage = "inventory.itt.manage"

	ZerolRead   = "inventory.zerolization.read"
	ZerolCreate = "inventory.zerolization.create"
	ZerolUpdate = "inventory.zerolization.update"
	ZerolSubmit = "inventory.zerolization.submit"
	ZerolManage = "inventory.zerolization.manage"

	HoldRead   = "inventory.hold.read"
	HoldCreate = "inventory.hold.create"
	HoldUpdate = "inventory.hold.update"
	HoldSubmit = "inventory.hold.submit"
	HoldManage = "inventory.hold.manage"
)

const (
	ChangeOfServiceRead   = "billing.cos.read"
	ChangeOfServiceCreate = "billing.cos.create"
	ChangeOfServiceUpdate = "billing.cos.update"
	ChangeOfServiceSubmit = "billing.cos.submit"
	ChangeOfServiceManage = "billing.cos.manage"
)

const (
	BillingRead   = "billing.read"
	BillingCreate = "billing.create"
	BillingUpdate = "billing.update"
	BillingSubmit = "billing.submit"
	BillingRun    = "billing.run"
	BillingManage = "billing.manage"
)

const (
	PricesRead   = "prices.read"
	PricesCreate = "prices.create"
	PricesUpdate = "prices.update"
	PricesSubmit = "prices.submit"
	PricesManage = "prices.manage"
)

const (
	EwuraRead   = "ewura.read"
	EwuraManage = "ewura.manage"
)

const (
	SageRead   = "sage.read"
	SagePost   = "sage.post"
	SageManage = "sage.manage"
)

const (
	OrdersRead   = "orders.read"
	OrdersCreate = "orders.create"
	OrdersSubmit = "orders.submit"
	OrdersManage = "orders.manage"
)

const (
	ILRRead     = "orders.ilr.read"
	ILRCreate   = "orders.ilr.create"
	ILRUpdate   = "orders.ilr.update"
	ILRSubmit   = "orders.ilr.submit"
	ILRComplete = "orders.ilr.complete"
	ILRManage   = "orders.ilr.manage"

	PumpOverRead   = "orders.pumpover.read"
	PumpOverCreate = "orders.pumpover.create"
	PumpOverUpdate = "orders.pumpover.update"
	PumpOverSubmit = "orders.pumpover.submit"
	PumpOverManage = "orders.pumpover.manage"

	PumpReportRead   = "orders.pumpreport.read"
	PumpReportCreate = "orders.pumpreport.create"
	PumpReportUpdate = "orders.pumpreport.update"
	PumpReportSubmit = "orders.pumpreport.submit"
	PumpReportManage = "orders.pumpreport.manage"

	CompRead   = "orders.compartmentalization.read"
	CompCreate = "orders.compartmentalization.create"
	CompUpdate = "orders.compartmentalization.update"
	CompSubmit = "orders.compartmentalization.submit"
	CompManage = "orders.compartmentalization.manage"

	AmendmentRead   = "orders.amendment.read"
	AmendmentCreate = "orders.amendment.create"
	AmendmentUpdate = "orders.amendment.update"
	AmendmentSubmit = "orders.amendment.submit"
	AmendmentManage = "orders.amendment.manage"
)
