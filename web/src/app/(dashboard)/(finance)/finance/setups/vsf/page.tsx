"use client";

import { FeeBatchList } from "../_components/FeeBatchList";

export default function VcfFeesPage() {
  return (
    <FeeBatchList
      kind="vsf"
      title="Variable storage fees"
      subtitle="EWURA price and density per product — MI-loss is taken from an approved MI-loss batch"
      path="/billing/vsf/batches"
      href="/finance/setups/vsf"
      requireMiLoss
    />
  );
}
