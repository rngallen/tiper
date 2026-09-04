/** Exact permission codes. Matching is not exact-only: see satisfiesPermission. */
export const PERMISSIONS = {
  usersRead: "users.read",
  usersManage: "users.manage",
  rolesRead: "roles.read",
  rolesManage: "roles.manage",
  titlesRead: "titles.read",
  titlesCreate: "titles.create",
  titlesUpdate: "titles.update",
  titlesDelete: "titles.delete",
  titlesManage: "titles.manage",
  workflowRead: "workflow.read",
  workflowTasks: "workflow.tasks",
  workflowManage: "workflow.manage",
  auditRead: "audit.read",
  reportsRead: "reports.read",
  settingsRead: "settings.read",
  settingsManage: "settings.manage",
  jobsRun: "jobs.run",
  masterdataRead: "masterdata.read",
  masterdataCreate: "masterdata.create",
  masterdataUpdate: "masterdata.update",
  masterdataDelete: "masterdata.delete",
  masterdataManage: "masterdata.manage",
  customersRead: "customers.read",
  customersCreate: "customers.create",
  customersUpdate: "customers.update",
  customersManage: "customers.manage",
  inventoryRead: "inventory.read",
  inventoryCreate: "inventory.create",
  inventoryUpdate: "inventory.update",
  inventorySubmit: "inventory.submit",
  inventoryManage: "inventory.manage",
  inventoryBalances: "inventory.balances",
  receiptsRead: "inventory.receipts.read",
  receiptsCreate: "inventory.receipts.create",
  receiptsUpdate: "inventory.receipts.update",
  receiptsSubmit: "inventory.receipts.submit",
  receiptsManage: "inventory.receipts.manage",
  ittRead: "inventory.itt.read",
  ittCreate: "inventory.itt.create",
  ittUpdate: "inventory.itt.update",
  ittSubmit: "inventory.itt.submit",
  ittManage: "inventory.itt.manage",
  zerolizationRead: "inventory.zerolization.read",
  zerolizationCreate: "inventory.zerolization.create",
  zerolizationUpdate: "inventory.zerolization.update",
  zerolizationSubmit: "inventory.zerolization.submit",
  zerolizationManage: "inventory.zerolization.manage",
  holdRead: "inventory.hold.read",
  holdCreate: "inventory.hold.create",
  holdUpdate: "inventory.hold.update",
  holdSubmit: "inventory.hold.submit",
  holdManage: "inventory.hold.manage",
  billingRead: "billing.read",
  billingCreate: "billing.create",
  billingUpdate: "billing.update",
  billingSubmit: "billing.submit",
  billingRun: "billing.run",
  billingManage: "billing.manage",
  changeOfServiceRead: "billing.cos.read",
  changeOfServiceCreate: "billing.cos.create",
  changeOfServiceUpdate: "billing.cos.update",
  changeOfServiceSubmit: "billing.cos.submit",
  changeOfServiceManage: "billing.cos.manage",
  pricesRead: "prices.read",
  pricesCreate: "prices.create",
  pricesUpdate: "prices.update",
  pricesSubmit: "prices.submit",
  pricesManage: "prices.manage",
  ewuraRead: "ewura.read",
  ewuraManage: "ewura.manage",
  sageRead: "sage.read",
  sageManage: "sage.manage",
  ordersRead: "orders.read",
  ordersCreate: "orders.create",
  ordersSubmit: "orders.submit",
  ordersManage: "orders.manage",
  ilrRead: "orders.ilr.read",
  ilrCreate: "orders.ilr.create",
  ilrUpdate: "orders.ilr.update",
  ilrSubmit: "orders.ilr.submit",
  ilrComplete: "orders.ilr.complete",
  ilrManage: "orders.ilr.manage",
  pumpOverRead: "orders.pumpover.read",
  pumpOverCreate: "orders.pumpover.create",
  pumpOverUpdate: "orders.pumpover.update",
  pumpOverSubmit: "orders.pumpover.submit",
  pumpOverManage: "orders.pumpover.manage",
  pumpReportRead: "orders.pumpreport.read",
  pumpReportCreate: "orders.pumpreport.create",
  pumpReportUpdate: "orders.pumpreport.update",
  pumpReportSubmit: "orders.pumpreport.submit",
  pumpReportManage: "orders.pumpreport.manage",
  compartmentalizationRead: "orders.compartmentalization.read",
  compartmentalizationCreate: "orders.compartmentalization.create",
  compartmentalizationUpdate: "orders.compartmentalization.update",
  compartmentalizationSubmit: "orders.compartmentalization.submit",
  compartmentalizationManage: "orders.compartmentalization.manage",
  amendmentRead: "orders.amendment.read",
  amendmentCreate: "orders.amendment.create",
  amendmentUpdate: "orders.amendment.update",
  amendmentSubmit: "orders.amendment.submit",
  amendmentManage: "orders.amendment.manage",
} as const;

/** Fee / price batch editors — manage wildcards are implied by satisfiesPermission. */
export const PRICE_WRITE_PERMS = [
  PERMISSIONS.billingCreate,
  PERMISSIONS.billingUpdate,
  PERMISSIONS.pricesCreate,
  PERMISSIONS.pricesUpdate,
];

export const PRICE_SUBMIT_PERMS = [
  PERMISSIONS.billingSubmit,
  PERMISSIONS.pricesSubmit,
];

/**
 * Whether held codes grant required. Mirrors pkg/permissions.Satisfies:
 * exact, {module}.{resource}.manage, {module}.{action}, {module}.manage.
 */
export function satisfiesPermission(held: string[], required: string): boolean {
  if (!required) return false;
  if (held.includes(required)) return true;
  const parts = required.split(".");
  if (parts.length === 2) {
    return held.includes(`${parts[0]}.manage`);
  }
  if (parts.length === 3) {
    const [module, resource, action] = parts;
    return (
      held.includes(`${module}.${resource}.manage`) ||
      held.includes(`${module}.${action}`) ||
      held.includes(`${module}.manage`)
    );
  }
  return false;
}

export function satisfiesAny(held: string[], ...required: string[]): boolean {
  return required.some((code) => satisfiesPermission(held, code));
}
