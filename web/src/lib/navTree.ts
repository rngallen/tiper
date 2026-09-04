import {
  LayoutDashboard,
  FileText,
  CreditCard,
  CheckSquare,
  Landmark,
  ScrollText,
  Settings,
  Users,
  Shield,
  BadgeCheck,
  KeyRound,
  GitBranch,
  BarChart3,
  Warehouse,
  Layers,
  Truck,
  Ship,
  Package,
  Building2,
  MapPin,
  CircleDot,
  UserRound,
  Droplets,
  Scale,
  BookOpen,
  Pencil,
  ArrowLeftRight,
  Fuel,
  Cylinder,
  Tags,
  Route,
  Handshake,
  FileSignature,
  BadgePercent,
  CalendarDays,
  type LucideIcon,
} from "lucide-react";
import { PERMISSIONS } from "@/lib/permissions";
import { APPROVAL_QUEUES } from "@/lib/approvals";
import { BILLING_MODULE, STOCK_MODULE, type WorkspaceArea } from "@/lib/erpNav";

export type NavLeaf = {
  kind: "link";
  href: string;
  label: string;
  icon: LucideIcon;
  perms: string[];
  section?: string;
  exact?: boolean;
};

export type NavGroup = {
  kind: "group";
  id: string;
  label: string;
  icon: LucideIcon;
  section?: string;
  children: NavNode[];
  defaultCollapsed?: boolean;
};

export type NavNode = NavLeaf | NavGroup;

const AREA_ICON: Record<string, LucideIcon> = {
  reception: Ship,
  gantry: Truck,
  "pump-over": Droplets,
  terminal: Landmark,
  reports: BarChart3,
  setups: Layers,
};

/** Distinct leaf icons — same idea as dfms-portal Maintenance / Transactions. */
const LEAF_ICON: Record<string, LucideIcon> = {
  "/stock/receipts": Ship,
  "/stock/receipts/external": Ship,
  "/stock/loading": Truck,
  "/stock/pump-over": Droplets,
  "/stock/pump-over-reports": FileText,
  "/stock/compartmentalization": Layers,
  "/stock/amendments": Pencil,
  "/stock/deliveries": Fuel,
  "/stock/itt": ArrowLeftRight,
  "/stock/zerolization": ArrowLeftRight,
  "/stock/financial-hold": Landmark,
  "/stock/balances": Scale,
  "/stock/movements": BookOpen,
  "/stock/reports": BarChart3,
  "/stock/setups/customers": Building2,
  "/stock/setups/suppliers": Landmark,
  "/stock/setups/products": Package,
  "/stock/setups/categories": Tags,
  "/stock/setups/tanks": Cylinder,
  "/stock/setups/vessels": Ship,
  "/stock/setups/depots": Warehouse,
  "/stock/setups/trucks": Truck,
  "/stock/setups/drivers": UserRound,
  "/stock/setups/transporters": Truck,
  "/stock/setups/destinations": MapPin,
  "/stock/setups/districts": MapPin,
  "/stock/setups/statuses": CircleDot,
  "/billing": CreditCard,
  "/billing/change-of-service": ArrowLeftRight,
  "/finance/reports": BarChart3,
  "/ewura": FileText,
  "/finance/setups/tenders": Tags,
  "/finance/setups/routes": Route,
  "/finance/setups/delivery": Truck,
  "/finance/setups/procurement": Handshake,
  "/finance/setups/contracts": FileSignature,
  "/finance/setups/pricing": BadgePercent,
  "/finance/setups/cycles": CalendarDays,
  "/finance/setups/miloss": BadgePercent,
};

function areaGroup(
  moduleId: string,
  area: WorkspaceArea,
  section: string,
  icon?: LucideIcon,
): NavGroup {
  const areaIcon = icon ?? AREA_ICON[area.id] ?? FileText;
  return {
    kind: "group",
    id: `${moduleId}-${area.id}`,
    label: area.label,
    icon: areaIcon,
    section,
    defaultCollapsed: area.id === "setups" || area.id === "reports",
    children: area.children.map((c) => ({
      kind: "link" as const,
      href: c.href,
      label: c.label,
      icon: LEAF_ICON[c.href] ?? areaIcon,
      perms: c.perms,
    })),
  };
}

export const NAV: NavNode[] = [
  {
    kind: "link",
    href: "/dashboard",
    label: "Dashboard",
    icon: LayoutDashboard,
    section: "Overview",
    perms: [PERMISSIONS.reportsRead],
  },
  {
    kind: "link",
    href: "/reports",
    label: "All reports",
    icon: BarChart3,
    section: "Overview",
    perms: [PERMISSIONS.reportsRead],
  },
  {
    kind: "link",
    href: "/audit",
    label: "Audit",
    icon: ScrollText,
    section: "Overview",
    perms: [PERMISSIONS.auditRead],
  },
  {
    kind: "group",
    id: "approvals",
    label: "Approvals",
    icon: CheckSquare,
    section: "Overview",
    children: APPROVAL_QUEUES.map((q) => ({
      kind: "link" as const,
      href: q.href,
      label: q.label,
      icon: q.icon,
      perms: q.perms,
    })),
  },
  {
    kind: "group",
    id: "stock",
    label: "Stock",
    icon: Warehouse,
    section: "Operations",
    children: STOCK_MODULE.areas.map((area) =>
      areaGroup("stock", area, "Operations"),
    ),
  },
  {
    kind: "group",
    id: "billing",
    label: "Billing",
    icon: CreditCard,
    section: "Operations",
    children: BILLING_MODULE.areas.map((area) =>
      areaGroup(
        "billing",
        area,
        "Operations",
        area.id === "setups" ? Landmark : area.id === "transactions" ? CreditCard : BarChart3,
      ),
    ),
  },
  {
    kind: "group",
    id: "access",
    label: "Access",
    icon: KeyRound,
    section: "Access",
    children: [
      {
        kind: "link",
        href: "/users",
        label: "Users",
        icon: Users,
        perms: [PERMISSIONS.usersRead, PERMISSIONS.usersManage],
      },
      {
        kind: "link",
        href: "/roles",
        label: "Roles",
        icon: Shield,
        perms: [PERMISSIONS.rolesRead, PERMISSIONS.rolesManage],
      },
      {
        kind: "link",
        href: "/titles",
        label: "Titles",
        icon: BadgeCheck,
        perms: [
          PERMISSIONS.titlesRead,
          PERMISSIONS.titlesCreate,
          PERMISSIONS.titlesUpdate,
          PERMISSIONS.titlesDelete,
          PERMISSIONS.titlesManage,
        ],
      },
      {
        kind: "link",
        href: "/workflows",
        label: "Workflows",
        icon: GitBranch,
        perms: [PERMISSIONS.workflowManage, PERMISSIONS.workflowRead],
      },
    ],
  },
  {
    kind: "link",
    href: "/settings",
    label: "Settings",
    icon: Settings,
    section: "System",
    perms: [
      PERMISSIONS.settingsRead,
      PERMISSIONS.settingsManage,
      PERMISSIONS.jobsRun,
    ],
  },
];

export function isActive(pathname: string, href: string, exact?: boolean) {
  if (exact) return pathname === href;
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function nodeActive(pathname: string, node: NavNode): boolean {
  if (node.kind === "link") return isActive(pathname, node.href, node.exact);
  return node.children.some((c) => nodeActive(pathname, c));
}

export function filterNav(
  nodes: NavNode[],
  has: (...codes: string[]) => boolean,
): NavNode[] {
  return nodes
    .map((node) => {
      if (node.kind === "link") {
        if (node.perms.length > 0 && !has(...node.perms)) return null;
        return node;
      }
      const children = filterNav(node.children, has);
      if (children.length === 0) return null;
      return { ...node, children };
    })
    .filter(Boolean) as NavNode[];
}
