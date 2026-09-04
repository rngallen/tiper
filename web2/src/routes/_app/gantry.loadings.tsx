import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/gantry/loadings")({
  component: () => (
    <RegisterPage
      title="Gantry loadings"
      columns={["Ticket", "OMC", "Product", "Volume", "Status"]}
      rows={[
        ["L-8841", "PUMA Energy", "AGO", "38,000 L", "Loading"],
        ["L-8842", "TotalEnergies", "PMS", "36,000 L", "Queued"],
      ]}
    />
  ),
});
