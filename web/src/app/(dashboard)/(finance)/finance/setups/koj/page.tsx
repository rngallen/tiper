"use client";

import { FeeBatchList } from "../_components/FeeBatchList";

export default function KojFeesPage() {
  return (
    <FeeBatchList
      kind="koj"
      title="KOJ fees"
      subtitle="Create a batch header, then add product rates billed on KOJ receipts"
      path="/billing/koj/batches"
      href="/finance/setups/koj"
    />
  );
}
