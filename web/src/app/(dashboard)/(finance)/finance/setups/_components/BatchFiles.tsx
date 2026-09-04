"use client";

import { DocumentFiles } from "@/components/DocumentFiles";

export function BatchFiles({ path, draft }: { path: string; draft: boolean }) {
  return (
    <DocumentFiles
      path={path}
      draft={draft}
      hint="Supporting files for this batch (rate sheets, EWURA notices, approvals)."
    />
  );
}
