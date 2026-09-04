import { createFileRoute } from "@tanstack/react-router";
import { OperationsBoard } from "@/app/pages";

export const Route = createFileRoute("/_app/dashboard")({
  component: OperationsBoard,
});
