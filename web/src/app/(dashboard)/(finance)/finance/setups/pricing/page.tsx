"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { PricingRow } from "@/lib/lookups";

export default function PricingPage() {
  return (
    <CatalogSetup<PricingRow>
      tableId="finance-pricing"
      title="Pricing natures"
      subtitle="Tariff vs promotional. Promotional rows match promotional FSF models."
      path="/master/pricing-natures"
      createLabel="Add pricing nature"
      singular="pricing nature"
      extra={[
        {
          key: "isPromotional",
          short: "Promo",
          label: "Promotional (matches promotional FSF models)",
        },
      ]}
    />
  );
}
