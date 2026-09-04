import { titleCase } from "@/lib/format";

const AREA_ORDER = [
  "Overview",
  "Approvals",
  "Stock",
  "Billing",
  "Access",
  "System",
  "Other",
] as const;

const AREA_BY_MODULE: Record<string, (typeof AREA_ORDER)[number]> = {
  reports: "Overview",
  audit: "Overview",
  workflow: "Approvals",
  masterdata: "Stock",
  customers: "Stock",
  inventory: "Stock",
  orders: "Stock",
  billing: "Billing",
  prices: "Billing",
  ewura: "Stock",
  sage: "Billing",
  users: "Access",
  roles: "Access",
  titles: "Access",
  settings: "System",
  jobs: "System",
};

const MODULE_LABEL: Record<string, string> = {
  users: "Users",
  roles: "Roles",
  titles: "Titles",
  workflow: "Workflows & inbox",
  masterdata: "Setups",
  customers: "Customers",
  inventory: "Inventory",
  orders: "Orders",
  billing: "Billing",
  prices: "Prices",
  ewura: "EWURA licenses",
  sage: "Sage",
  reports: "Reports",
  audit: "Audit",
  settings: "Settings",
  jobs: "Jobs",
};

export function permissionArea(module: string): string {
  return AREA_BY_MODULE[module] ?? "Other";
}

export function permissionModuleLabel(module: string): string {
  return MODULE_LABEL[module] ?? titleCase(module);
}

const RESOURCE_LABEL: Record<string, string> = {
  "orders._": "Broad (all order documents)",
  "orders.ilr": "Internal loading (ILR / ILO)",
  "orders.pumpover": "Pump-over requests",
  "orders.pumpreport": "Pump-over reports",
  "orders.compartmentalization": "Compartmentalization",
  "orders.amendment": "Loading amendments",
  "inventory._": "Broad (all stock documents)",
  "inventory.receipts": "Vessel receipts",
  "inventory.itt": "In-tank transfers",
  "inventory.zerolization": "Zerolization",
  "inventory.hold": "Financial hold",
  "billing._": "Broad (all billing documents)",
  "billing.cos": "Change of service",
};

/** Split a module's codes into document groups when 3-part codes are present. */
export function groupModuleResources<T extends { code: string }>(
  perms: T[],
): { key: string; label: string; perms: T[] }[] {
  const hasNested = perms.some((p) => p.code.split(".").length >= 3);
  if (!hasNested) {
    return [{ key: "_", label: "", perms }];
  }
  const map = new Map<string, T[]>();
  for (const p of perms) {
    const parts = p.code.split(".");
    const key = parts.length >= 3 ? `${parts[0]}.${parts[1]}` : `${parts[0]}._`;
    const arr = map.get(key) ?? [];
    arr.push(p);
    map.set(key, arr);
  }
  const rank = (k: string) => (k.endsWith("._") ? 0 : 1);
  return [...map.entries()]
    .sort((a, b) => {
      const d = rank(a[0]) - rank(b[0]);
      if (d !== 0) return d;
      return (RESOURCE_LABEL[a[0]] ?? a[0]).localeCompare(RESOURCE_LABEL[b[0]] ?? b[0]);
    })
    .map(([key, list]) => ({
      key,
      label: RESOURCE_LABEL[key] ?? key.split(".").slice(1).join(" "),
      perms: list,
    }));
}

export function groupPermissions<T extends { module?: string }>(
  perms: T[],
): { area: string; modules: [string, T[]][] }[] {
  const areas = new Map<string, Map<string, T[]>>();
  for (const p of perms) {
    const module = p.module || "other";
    const area = permissionArea(module);
    if (!areas.has(area)) areas.set(area, new Map());
    const mods = areas.get(area)!;
    const arr = mods.get(module) ?? [];
    arr.push(p);
    mods.set(module, arr);
  }
  const known = AREA_ORDER.filter((a) => areas.has(a));
  const extra = [...areas.keys()].filter((a) => !AREA_ORDER.includes(a as (typeof AREA_ORDER)[number]));
  return [...known, ...extra].map((area) => ({
    area,
    modules: [...areas.get(area)!.entries()].sort(([a], [b]) =>
      permissionModuleLabel(a).localeCompare(permissionModuleLabel(b)),
    ),
  }));
}
