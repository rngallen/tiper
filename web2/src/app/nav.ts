export type NavItem = { label: string; to: string; section: string };

export const NAV: NavItem[] = [
  { section: "Command", label: "Operations board", to: "/dashboard" },
  { section: "Stock", label: "Tank farm", to: "/stock/tanks" },
  { section: "Stock", label: "Receipts", to: "/stock/receipts" },
  { section: "Gantry", label: "Loadings", to: "/gantry/loadings" },
  { section: "Terminal", label: "Pump-over", to: "/terminal/pumpover" },
  { section: "Commercial", label: "Billing", to: "/billing" },
  { section: "Compliance", label: "EWURA", to: "/ewura" },
  { section: "Control", label: "Users", to: "/access/users" },
  { section: "Control", label: "Roles", to: "/access/roles" },
];
