"use client";

import { FeeBatchList } from "../_components/FeeBatchList";

export default function MiLossListPage() {
  return (
    <FeeBatchList
      kind="miloss"
      title="MI loss"
      subtitle="Contract MI-loss rates used by variable storage fees — uniqueness is product and contract together"
      path="/billing/miloss/batches"
      href="/finance/setups/miloss"
      withFx={false}
    />
  );
}
