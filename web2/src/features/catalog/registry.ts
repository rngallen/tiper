import { P } from "@/shared/config/permissions";
import { CT } from "@/shared/config/content-types";

export type FieldKind = "text" | "textarea" | "number" | "email" | "tel" | "date" | "checkbox";
export interface FieldDef { key: string; label: string; kind?: FieldKind; required?: boolean; placeholder?: string; }
export interface ColumnDef { key: string; label: string; kind?: "text" | "status" | "date" | "datetime" | "qty" | "money" | "bool" | "ref" | "code"; }
export type ResourceKind = "master" | "document" | "report" | "inbox" | "board" | "custom";
export interface ResourceDef {
  path: string; title: string; subtitle?: string; section: string; kind: ResourceKind; api: string;
  perms: string[]; createPerms?: string[]; updatePerms?: string[]; deletePerms?: string[];
  submitPath?: (id: string) => string; contentType?: number; columns: ColumnDef[]; fields?: FieldDef[];
  dated?: boolean; createLabel?: string; extraQuery?: Record<string, string | number | boolean>;
}

const m = (path: string, title: string, api: string, extra: Partial<ResourceDef> = {}): ResourceDef => ({
  path, title, section: "Setups", kind: "master", api, perms: [P.masterdata.read],
  createPerms: [P.masterdata.create], updatePerms: [P.masterdata.update], deletePerms: [P.masterdata.delete],
  columns: [{ key: "code", label: "Code", kind: "code" }, { key: "name", label: "Name" }, { key: "isActive", label: "Active", kind: "bool" }],
  fields: [{ key: "code", label: "Code", required: true }, { key: "name", label: "Name", required: true }, { key: "isActive", label: "Active", kind: "checkbox" }],
  createLabel: "New record", ...extra,
});
const d = (path: string, title: string, api: string, perms: string[], extra: Partial<ResourceDef> = {}): ResourceDef => ({
  path, title, section: "Operations", kind: "document", api, dated: true, perms,
  columns: [{ key: "documentNumber", label: "Document", kind: "code" }, { key: "date", label: "Date", kind: "date" }, { key: "status", label: "Status", kind: "status" }],
  ...extra,
});

export const RESOURCES: ResourceDef[] = [
  { path: "/dashboard", title: "Operations board", section: "Command", kind: "board", api: "/reports/charts/kpis", perms: [P.reports.read], columns: [] },
  { path: "/approvals", title: "Approvals inbox", section: "Command", kind: "inbox", api: "/workflow/tasks/mine", perms: [P.workflow.tasks], columns: [] },
  { path: "/audit", title: "Audit trail", section: "Command", kind: "document", api: "/audit", dated: true, perms: [P.audit.read], columns: [
    { key: "createdAt", label: "When", kind: "datetime" }, { key: "actorEmail", label: "Actor" }, { key: "action", label: "Action", kind: "code" }, { key: "entity", label: "Entity" }, { key: "summary", label: "Summary" },
  ]},
  { path: "/reports", title: "Report catalog", section: "Command", kind: "report", api: "/reports/registry", perms: [P.reports.read], columns: [{ key: "title", label: "Report" }, { key: "group", label: "Group" }, { key: "description", label: "Description" }] },
  d("/stock/receipts", "Internal receipts", "/ic/receipts", [P.inventory.receipts.read], { extraQuery: { kind: "internal" }, contentType: CT.Receipt, submitPath: (id) => `/ic/receipts/${id}/submit`, columns: [
    { key: "documentNumber", label: "Receipt", kind: "code" }, { key: "date", label: "Date", kind: "date" }, { key: "vessel", label: "Vessel", kind: "ref" }, { key: "product", label: "Product", kind: "ref" }, { key: "tankQuantity", label: "Nominated", kind: "qty" }, { key: "status", label: "Status", kind: "status" },
  ]}),
  d("/stock/receipts/external", "External receipts", "/ic/receipts", [P.inventory.receipts.read], { extraQuery: { kind: "external" }, contentType: CT.Receipt, submitPath: (id) => `/ic/receipts/${id}/submit` }),
  d("/stock/loading", "Internal loading", "/orders/loading", [P.orders.ilr.read], { contentType: CT.GantryLoadingRequest, submitPath: (id) => `/orders/loading/${id}/submit`, columns: [
    { key: "documentNumber", label: "ILR", kind: "code" }, { key: "date", label: "Date", kind: "date" }, { key: "customer", label: "Customer", kind: "ref" }, { key: "truck", label: "Truck", kind: "ref" }, { key: "status", label: "Status", kind: "status" },
  ]}),
  d("/stock/compartmentalization", "Compartmentalization", "/orders/compartmentalizations", [P.orders.compartmentalization.read], { contentType: CT.Compartmentalization, submitPath: (id) => `/orders/compartmentalizations/${id}/submit` }),
  d("/stock/amendments", "Amendments", "/orders/amendments", [P.orders.amendment.read], { contentType: CT.OrderAmendment, submitPath: (id) => `/orders/amendments/${id}/submit` }),
  d("/stock/deliveries", "Gantry completion", "/orders/loading-lines", [P.orders.ilr.complete], { columns: [
    { key: "documentNumber", label: "Line", kind: "code" }, { key: "truck", label: "Truck", kind: "ref" }, { key: "product", label: "Product", kind: "ref" }, { key: "quantity", label: "Qty", kind: "qty" }, { key: "status", label: "Status", kind: "status" },
  ]}),
  d("/stock/pump-over", "Pump-over", "/orders/pump-over", [P.orders.pumpover.read], { contentType: CT.PumpOverRequest, submitPath: (id) => `/orders/pump-over/${id}/submit` }),
  d("/stock/pump-over-reports", "Pump-over reports", "/orders/pump-over-reports", [P.orders.pumpreport.read], { contentType: CT.PumpOverReport, submitPath: (id) => `/orders/pump-over-reports/${id}/submit` }),
  d("/stock/itt", "In-tank transfers", "/ic/itt", [P.inventory.itt.read], { contentType: CT.IttTransfer, submitPath: (id) => `/ic/itt/${id}/submit` }),
  d("/stock/zerolization", "Zerolization", "/ic/zerolization", [P.inventory.zerolization.read], { contentType: CT.Zerolization, submitPath: (id) => `/ic/zerolization/${id}/submit` }),
  d("/stock/financial-hold", "Financial hold", "/ic/hold-releases", [P.inventory.hold.read], { contentType: CT.FinancialHold, submitPath: (id) => `/ic/hold-releases/${id}/submit` }),
  { path: "/stock/balances", title: "Stock balances", section: "Reports", kind: "document", api: "/ic/balances", perms: [P.inventory.balances, P.inventory.read, P.reports.read], columns: [
    { key: "customer", label: "Customer", kind: "ref" }, { key: "product", label: "Product", kind: "ref" }, { key: "status", label: "Status", kind: "ref" }, { key: "quantity", label: "Book qty", kind: "qty" }, { key: "available", label: "Available", kind: "qty" },
  ]},
  { path: "/stock/movements", title: "Movements", section: "Reports", kind: "document", api: "/ic/movements", dated: true, perms: [P.inventory.read, P.reports.read], columns: [
    { key: "postedAt", label: "Posted", kind: "datetime" }, { key: "kind", label: "Kind", kind: "code" }, { key: "documentNumber", label: "Document", kind: "code" }, { key: "quantity", label: "Qty", kind: "qty" },
  ]},
  { path: "/stock/reports", title: "Stock reports", section: "Reports", kind: "report", api: "/reports/registry", extraQuery: { group: "Stock" }, perms: [P.reports.read], columns: [{ key: "title", label: "Report" }, { key: "description", label: "Description" }] },
  m("/stock/setups/customers", "Customers", "/master/customers", { perms: [P.customers.read], createPerms: [P.customers.create], updatePerms: [P.customers.update], columns: [
    { key: "code", label: "Code", kind: "code" }, { key: "name", label: "Name" }, { key: "tinNumber", label: "TIN" }, { key: "ewuraLicense", label: "EWURA" }, { key: "isActive", label: "Active", kind: "bool" },
  ], fields: [
    { key: "code", label: "Code", required: true }, { key: "name", label: "Name", required: true }, { key: "email", label: "Email", kind: "email" }, { key: "phone", label: "Phone", kind: "tel" }, { key: "tinNumber", label: "TIN" }, { key: "ewuraLicense", label: "EWURA license" }, { key: "isActive", label: "Active", kind: "checkbox" },
  ]}),
  { path: "/ewura", title: "EWURA licenses", section: "Compliance", kind: "document", api: "/ewura/licenses", perms: [P.ewura.read, P.ewura.manage], columns: [
    { key: "licenseNumber", label: "License", kind: "code" }, { key: "licensee", label: "Licensee" }, { key: "licenseClass", label: "Class" }, { key: "expiryDate", label: "Expires", kind: "date" }, { key: "isActive", label: "Active", kind: "bool" },
  ]},
  m("/stock/setups/suppliers", "Suppliers", "/master/suppliers"),
  m("/stock/setups/categories", "Product categories", "/master/categories"),
  m("/stock/setups/products", "Products", "/master/products", { columns: [
    { key: "code", label: "Code", kind: "code" }, { key: "name", label: "Name" }, { key: "unit", label: "UoM" }, { key: "isActive", label: "Active", kind: "bool" },
  ], fields: [
    { key: "code", label: "Code", required: true }, { key: "name", label: "Name", required: true }, { key: "unit", label: "Unit" }, { key: "isActive", label: "Active", kind: "checkbox" },
  ]}),
  m("/stock/setups/tanks", "Tanks", "/master/tanks", { columns: [
    { key: "code", label: "Tank", kind: "code" }, { key: "name", label: "Name" }, { key: "product", label: "Product", kind: "ref" }, { key: "maximumCapacity", label: "Capacity", kind: "qty" }, { key: "isActive", label: "Active", kind: "bool" },
  ], fields: [
    { key: "code", label: "Code", required: true }, { key: "name", label: "Name", required: true }, { key: "maximumCapacity", label: "Maximum capacity", kind: "number" }, { key: "deadStock", label: "Dead stock", kind: "number" }, { key: "isActive", label: "Active", kind: "checkbox" },
  ]}),
  m("/stock/setups/vessels", "Vessels", "/master/vessels"),
  m("/stock/setups/depots", "Depots", "/master/depots"),
  m("/stock/setups/trucks", "Trucks", "/master/trucks", { columns: [
    { key: "plateNumber", label: "Plate", kind: "code" }, { key: "trailer", label: "Trailer" }, { key: "vehicleType", label: "Type" }, { key: "isActive", label: "Active", kind: "bool" },
  ], fields: [
    { key: "plateNumber", label: "Plate number", required: true }, { key: "trailer", label: "Trailer" }, { key: "isActive", label: "Active", kind: "checkbox" },
  ]}),
  m("/stock/setups/drivers", "Drivers", "/master/drivers"),
  m("/stock/setups/transporters", "Haulers", "/master/transporters"),
  m("/stock/setups/destinations", "Destinations", "/master/destinations"),
  m("/stock/setups/districts", "Districts", "/master/districts"),
  m("/stock/setups/statuses", "Stock statuses", "/master/statuses"),
  d("/billing", "Billing runs", "/billing/runs", [P.billing.read], { section: "Commercial", contentType: CT.BillingRun, submitPath: (id) => `/billing/runs/${id}/submit` }),
  d("/billing/change-of-service", "Change of service", "/billing/change-of-service", [P.billing.read, P.billing.cos.read], { section: "Commercial", contentType: CT.ChangeOfService, submitPath: (id) => `/billing/change-of-service/${id}/submit` }),
  { path: "/finance/reports", title: "Billing reports", section: "Commercial", kind: "report", api: "/reports/registry", extraQuery: { group: "Billing" }, perms: [P.reports.read], columns: [{ key: "title", label: "Report" }, { key: "description", label: "Description" }] },
  d("/finance/setups/fsf", "Fixed storage fees", "/billing/fsf/batches", [P.billing.read, P.prices.read], { contentType: CT.BillingProfile, submitPath: (id) => `/billing/fsf/batches/${id}/submit` }),
  d("/finance/setups/vsf", "Variable storage fees", "/billing/vsf/batches", [P.billing.read, P.prices.read], { contentType: CT.VariableFeeBatch, submitPath: (id) => `/billing/vsf/batches/${id}/submit` }),
  d("/finance/setups/koj", "KOJ fees", "/billing/koj/batches", [P.billing.read, P.prices.read], { contentType: CT.KojFeeBatch, submitPath: (id) => `/billing/koj/batches/${id}/submit` }),
  d("/finance/setups/tbs", "TBS fees", "/billing/tbs/batches", [P.billing.read, P.prices.read], { contentType: CT.TbsFeeBatch, submitPath: (id) => `/billing/tbs/batches/${id}/submit` }),
  d("/finance/setups/fx", "Exchange rates", "/billing/fx", [P.billing.read, P.prices.read], { contentType: CT.ExchangeRate, submitPath: (id) => `/billing/fx/${id}/submit` }),
  d("/finance/setups/miloss", "MI loss", "/billing/miloss/batches", [P.billing.read, P.prices.read], { contentType: CT.MiLossBatch, submitPath: (id) => `/billing/miloss/batches/${id}/submit` }),
  m("/finance/setups/tenders", "Tenders", "/master/tenders", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/routes", "Discharge routes", "/master/routes", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/delivery", "Delivery methods", "/master/delivery-methods", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/procurement", "Procurement", "/master/procurement-methods", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/contracts", "Contract types", "/master/contract-types", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/pricing", "Pricing natures", "/master/pricing-natures", { perms: [P.masterdata.read, P.billing.read] }),
  m("/finance/setups/cycles", "Billing cycles", "/master/billing-cycles", { perms: [P.masterdata.read, P.billing.read] }),
  { path: "/users", title: "Users", section: "Access", kind: "master", api: "/auth/users", perms: [P.users.read, P.users.manage], createPerms: [P.users.manage], updatePerms: [P.users.manage], deletePerms: [P.users.manage], columns: [
    { key: "email", label: "Email" }, { key: "firstName", label: "First name" }, { key: "lastName", label: "Last name" }, { key: "isActive", label: "Active", kind: "bool" }, { key: "isLocked", label: "Locked", kind: "bool" }, { key: "lastLogin", label: "Last login", kind: "datetime" },
  ], fields: [
    { key: "email", label: "Email", kind: "email", required: true }, { key: "firstName", label: "First name", required: true }, { key: "lastName", label: "Last name", required: true }, { key: "isActive", label: "Active", kind: "checkbox" },
  ]},
  { path: "/roles", title: "Roles", section: "Access", kind: "master", api: "/auth/roles", perms: [P.roles.read, P.roles.manage], createPerms: [P.roles.manage], updatePerms: [P.roles.manage], columns: [
    { key: "name", label: "Name" }, { key: "description", label: "Description" }, { key: "category", label: "Category" },
  ], fields: [{ key: "name", label: "Name", required: true }, { key: "description", label: "Description", kind: "textarea" }]},
  { path: "/titles", title: "Titles", section: "Access", kind: "master", api: "/auth/titles", perms: [P.titles.read, P.titles.manage], createPerms: [P.titles.manage], updatePerms: [P.titles.manage], columns: [{ key: "name", label: "Name" }, { key: "description", label: "Description" }], fields: [{ key: "name", label: "Name", required: true }, { key: "description", label: "Description" }]},
  { path: "/workflows", title: "Workflows", section: "Access", kind: "document", api: "/workflow/processes", perms: [P.workflow.read, P.workflow.manage], columns: [{ key: "code", label: "Code", kind: "code" }, { key: "name", label: "Name" }, { key: "isActive", label: "Active", kind: "bool" }] },
  { path: "/settings", title: "Settings", section: "System", kind: "custom", api: "/auth/profile", perms: [P.settings.read, P.settings.manage, P.jobs.run], columns: [] },
];

const byPath = new Map(RESOURCES.map((r) => [r.path, r]));
export function findResource(pathname: string): { resource: ResourceDef; id?: string } | null {
  const clean = pathname.replace(/\/+$/, "") || "/";
  const exact = byPath.get(clean);
  if (exact) return { resource: exact };
  const parts = clean.split("/");
  if (parts.length < 3) return null;
  const parent = parts.slice(0, -1).join("/");
  const res = byPath.get(parent);
  if (res) return { resource: res, id: parts[parts.length - 1] };
  return null;
}
