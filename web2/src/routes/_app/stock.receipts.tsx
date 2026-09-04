import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/stock/receipts")({
  component: () => (
    <RegisterPage
      title="Receipts"
      columns={["Reference", "Source", "Product", "Nominated", "Status"]}
      rows={[
        ["RCV-091", "MT Kurasini", "AGO", "18,400 m³", "Discharging"],
        ["RCV-090", "MT Bahari Star", "PMS", "12,000 m³", "Inbound"],
      ]}
    />
  ),
});
