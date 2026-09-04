/**
 * Permission codes mirror `pkg/permissions/codes.go`. Route middleware passes if
 * the user holds ANY listed code; `satisfies` replicates the server hierarchy so
 * navigation and buttons hide exactly what the API would refuse.
 */
export const P = {
  users: { read: "users.read", manage: "users.manage" },
  roles: { read: "roles.read", manage: "roles.manage" },
  titles: {
    read: "titles.read",
    create: "titles.create",
    update: "titles.update",
    delete: "titles.delete",
    manage: "titles.manage",
  },
  workflow: { read: "workflow.read", tasks: "workflow.tasks", manage: "workflow.manage" },
  audit: { read: "audit.read" },
  reports: { read: "reports.read" },
  settings: { read: "settings.read", manage: "settings.manage" },
  jobs: { run: "jobs.run" },
  masterdata: {
    read: "masterdata.read",
    create: "masterdata.create",
    update: "masterdata.update",
    delete: "masterdata.delete",
    manage: "masterdata.manage",
  },
  customers: {
    read: "customers.read",
    create: "customers.create",
    update: "customers.update",
    manage: "customers.manage",
  },
  inventory: {
    read: "inventory.read",
    create: "inventory.create",
    update: "inventory.update",
    submit: "inventory.submit",
    manage: "inventory.manage",
    balances: "inventory.balances",
    receipts: doc("inventory.receipts"),
    itt: doc("inventory.itt"),
    zerolization: doc("inventory.zerolization"),
    hold: doc("inventory.hold"),
  },
  billing: {
    read: "billing.read",
    create: "billing.create",
    update: "billing.update",
    submit: "billing.submit",
    run: "billing.run",
    manage: "billing.manage",
    cos: doc("billing.cos"),
  },
  prices: {
    read: "prices.read",
    create: "prices.create",
    update: "prices.update",
    submit: "prices.submit",
    manage: "prices.manage",
  },
  ewura: { read: "ewura.read", manage: "ewura.manage" },
  sage: { read: "sage.read", post: "sage.post", manage: "sage.manage" },
  orders: {
    read: "orders.read",
    create: "orders.create",
    submit: "orders.submit",
    manage: "orders.manage",
    ilr: { ...doc("orders.ilr"), complete: "orders.ilr.complete" },
    pumpover: doc("orders.pumpover"),
    pumpreport: doc("orders.pumpreport"),
    compartmentalization: doc("orders.compartmentalization"),
    amendment: doc("orders.amendment"),
  },
} as const;

function doc<M extends string>(m: M) {
  return {
    read: `${m}.read` as const,
    create: `${m}.create` as const,
    update: `${m}.update` as const,
    submit: `${m}.submit` as const,
    manage: `${m}.manage` as const,
  };
}

export type PermissionCode = string;

/**
 * Mirror of `permissions.Satisfies(held, required)`:
 * 1. exact match
 * 2. `m.a`   ← `m.manage`
 * 3. `m.r.a` ← `m.r.manage` | `m.a` | `m.manage`
 */
export function satisfies(held: ReadonlySet<string>, required: string): boolean {
  if (held.has(required)) return true;
  const parts = required.split(".");
  if (parts.length === 2) {
    return held.has(`${parts[0]}.manage`);
  }
  if (parts.length === 3) {
    const [m, r, a] = parts;
    return held.has(`${m}.${r}.manage`) || held.has(`${m}.${a}`) || held.has(`${m}.manage`);
  }
  return false;
}

/** ANY of `required` (the server's PermissionMiddleware semantics). */
export function canAny(
  held: ReadonlySet<string>,
  isSuperUser: boolean,
  required: readonly string[] | string | undefined,
): boolean {
  if (isSuperUser) return true;
  if (!required || (Array.isArray(required) && required.length === 0)) return true;
  const list = typeof required === "string" ? [required] : required;
  return list.some((code) => satisfies(held, code));
}
