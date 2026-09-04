/** Stored Role.category — same codes as pkg/types.RoleFamily. */
export const ROLE_FAMILIES = [
  "system",
  "terminal",
  "gantry",
  "finance",
  "billing",
] as const;

export type RoleFamily = (typeof ROLE_FAMILIES)[number];

const LABELS: Record<RoleFamily, string> = {
  system: "System",
  terminal: "Terminal",
  gantry: "Gantry",
  finance: "Finance",
  billing: "Billing",
};

export function isRoleFamily(v: string | undefined): v is RoleFamily {
  return ROLE_FAMILIES.includes((v || "") as RoleFamily);
}

export function roleFamilyOf(category: string | undefined): RoleFamily {
  if (category === "stock" || category === "reception") return "terminal";
  return isRoleFamily(category) ? category : "system";
}

export function roleFamilyLabel(category: string | undefined): string {
  return LABELS[roleFamilyOf(category)];
}
