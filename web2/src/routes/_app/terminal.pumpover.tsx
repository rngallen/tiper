import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/terminal/pumpover")({
  component: () => (
    <RegisterPage
      title="Pump-over"
      columns={["Order", "From", "To", "Product", "Status"]}
      rows={[["PO-220", "TK-04", "Customer tank", "AGO", "In progress"]]}
    />
  ),
});
