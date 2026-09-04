"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { DeliveryRow } from "@/lib/lookups";

export default function DeliveryPage() {
  return (
    <CatalogSetup<DeliveryRow>
      tableId="finance-delivery"
      title="Delivery methods"
      subtitle="How product leaves the terminal — pump-over or gantry loading. Cycle length is not stored here."
      path="/master/delivery-methods"
      createLabel="Add delivery method"
      singular="delivery method"
      extra={[
        {
          key: "isGantryLoading",
          short: "Gantry",
          label: "Gantry / one-stop loading",
        },
      ]}
    />
  );
}
