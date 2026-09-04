import { useLocation } from "@tanstack/react-router";
import { findResource } from "@/features/catalog/registry";
import { ResourceScreen } from "@/features/catalog/ResourceScreen";
import { DocumentScreen } from "@/features/catalog/DocumentScreen";
import { DashboardScreen } from "@/features/ops/DashboardScreen";
import { InboxScreen } from "@/features/approvals/InboxScreen";
import { ReportsScreen } from "@/features/reports/ReportsScreen";
import { SettingsScreen } from "@/features/settings/SettingsScreen";
import { EmptyState } from "@/shared/ui/empty-state";
import { useSession } from "@/features/auth/session";

export function Workspace() {
  const pathname = useLocation({ select: (l) => l.pathname });
  const can = useSession((s) => s.can);
  const match = findResource(pathname);
  if (!match) return <EmptyState title="Screen not found" description={`${pathname} is not in the console map.`} />;
  const { resource, id } = match;
  if (!can(...resource.perms)) return <EmptyState title="No access" description="Your role does not include this workspace." />;
  if (resource.kind === "board") return <DashboardScreen />;
  if (resource.kind === "inbox") return <InboxScreen />;
  if (resource.kind === "report") return <ReportsScreen resource={resource} />;
  if (resource.kind === "custom") return <SettingsScreen />;
  if (resource.kind === "document") return <DocumentScreen resource={resource} selectedId={id} />;
  return <ResourceScreen resource={resource} selectedId={id} />;
}
