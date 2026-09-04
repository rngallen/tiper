import { createFileRoute } from "@tanstack/react-router";
import { DashboardScreen } from "@/features/ops/DashboardScreen";
export const Route = createFileRoute("/_app/dashboard")({ component: DashboardScreen });
