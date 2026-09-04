import { createFileRoute } from "@tanstack/react-router";
import { Workspace } from "@/features/workspace/Workspace";
export const Route = createFileRoute("/_app/$")({ component: Workspace });
