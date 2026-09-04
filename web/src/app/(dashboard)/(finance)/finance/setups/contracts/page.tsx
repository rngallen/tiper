"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { CatalogRow } from "@/lib/lookups";

export default function ContractsPage() {
  return (
    <CatalogSetup<CatalogRow>
      tableId="finance-contracts"
      title="Contract types"
      subtitle="AD HOC, TOP, or SRT on the parcel — keys variable storage fees and MI-loss, not the tender."
      path="/master/contract-types"
      createLabel="Add contract type"
      singular="contract type"
      extra={[]}
    />
  );
}
