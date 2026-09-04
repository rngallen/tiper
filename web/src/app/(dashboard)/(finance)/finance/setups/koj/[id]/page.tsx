"use client";

import { PriceBatchEditor } from "../../_components/PriceBatchEditor";

export default function KojBatchPage() {
  return (
    <PriceBatchEditor
      kind="koj"
      path="/billing/koj/batches"
      listHref="/finance/setups/koj"
      title="KOJ fees"
    />
  );
}
