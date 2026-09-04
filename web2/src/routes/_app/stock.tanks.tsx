import { createFileRoute } from "@tanstack/react-router";
import { RegisterPage } from "@/app/pages";

export const Route = createFileRoute("/_app/stock/tanks")({
  component: () => (
    <RegisterPage
      title="Tank farm"
      columns={["Code", "Product", "Capacity", "Level", "Status"]}
      rows={[
        ["TK-01", "PMS", "12,000 m³", "78%", "Operational"],
        ["TK-04", "AGO", "15,000 m³", "86%", "Operational"],
        ["TK-05", "AGO", "15,000 m³", "22%", "Low"],
      ]}
    />
  ),
});
