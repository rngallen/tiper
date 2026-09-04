"use client";

import { PriceBatchEditor } from "../../_components/PriceBatchEditor";

export default function TbsBatchPage() {
  return (
    <PriceBatchEditor
      kind="tbs"
      path="/billing/tbs/batches"
      listHref="/finance/setups/tbs"
      title="TBS fees"
    />
  );
}
