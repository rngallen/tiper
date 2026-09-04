import type { LucideIcon } from "lucide-react";
import { FileText } from "lucide-react";
import { RESOURCES } from "@/features/catalog/registry";

export type NavLeaf = {
  kind: "link";
  to: string;
  label: string;
  icon: LucideIcon;
  section: string;
  perms: string[];
};

export const NAV: NavLeaf[] = RESOURCES.map((r) => ({
  kind: "link" as const,
  to: r.path,
  label: r.title,
  icon: FileText,
  section: r.section,
  perms: r.perms,
}));

export const NAV_SECTIONS = [
  "Command",
  "Operations",
  "Reports",
  "Setups",
  "Compliance",
  "Commercial",
  "Access",
  "System",
] as const;
