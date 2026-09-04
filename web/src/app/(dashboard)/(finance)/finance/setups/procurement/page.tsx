"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { CatalogRow } from "@/lib/lookups";

export default function ProcurementPage() {
  return (
    <CatalogSetup<CatalogRow>
      tableId="finance-procurement"
      title="Procurement methods"
      subtitle="BPS vs private. Used when matching first-cycle fee models."
      path="/master/procurement-methods"
      createLabel="Add procurement method"
      singular="procurement method"
      extra={[]}
    />
  );
}
