"use client";

import { FeeBatchList } from "../_components/FeeBatchList";

export default function TbsFeesPage() {
  return (
    <FeeBatchList
      kind="tbs"
      title="TBS fees"
      subtitle="Create a batch header, then add product rates billed on daily gantry lifts"
      path="/billing/tbs/batches"
      href="/finance/setups/tbs"
    />
  );
}
