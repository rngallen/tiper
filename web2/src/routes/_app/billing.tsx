import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/billing")({
  component: () => (
    <RegisterPage
      title="Billing runs"
      columns={["Invoice", "Customer", "Period", "Amount", "Sage"]}
      rows={[["INV-4412", "PUMA Energy TZ", "Aug 2026", "TZS 186.4m", "Unposted"]]}
    />
  ),
});
