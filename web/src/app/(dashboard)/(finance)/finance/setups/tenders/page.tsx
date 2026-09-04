"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { TenderRow } from "@/lib/lookups";

export default function TendersPage() {
  return (
    <CatalogSetup<TenderRow>
      tableId="finance-tenders"
      title="Tenders"
      subtitle="SRT vs non-SRT. Engine flags decide first-cycle billing — not the label."
      path="/master/tenders"
      createLabel="Add tender"
      singular="tender"
      extra={[
        { key: "isSingleReceiving", short: "SRT", label: "SRT class (reports)" },
        {
          key: "supplierPaysUnlessLoading",
          short: "Supplier pays",
          label: "Supplier pays unless gantry loading",
        },
      ]}
    />
  );
}
