import { createFileRoute } from "@tanstack/react-router";
import { LoginScreen } from "@/features/auth/LoginScreen";
export const Route = createFileRoute("/login")({ component: LoginScreen });
