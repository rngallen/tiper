"use client";

import { FeeBatchList } from "../_components/FeeBatchList";

export default function FcfFeesPage() {
  return (
    <FeeBatchList
      kind="fsf"
      title="Fixed storage fees"
      subtitle="Fixed storage price list — latest approved list in force as of the receipt billing date"
      path="/billing/fsf/batches"
      href="/finance/setups/fsf"
    />
  );
}
