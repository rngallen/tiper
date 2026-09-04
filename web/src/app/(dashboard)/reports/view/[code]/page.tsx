"use client";

import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { PageHeader } from "@/components/ui";
import { ReportRunner } from "@/components/ReportRunner";
import { RequirePermission } from "@/components/RequirePermission";
import { PERMISSIONS } from "@/lib/permissions";
import type { ReportMeta } from "@/lib/types";

export default function ReportViewPage() {
  const params = useParams<{ code: string }>();
  const router = useRouter();
  const code = params.code;
  const registry = useAsync(() => api.get<ReportMeta[]>("/reports/registry"), []);
  const meta = (registry.data || []).find((r) => r.code === code);
  const title = meta?.name || code.replace(/-/g, " ");

  return (
    <RequirePermission codes={[PERMISSIONS.reportsRead]} resource="reports">
      <div>
        <PageHeader
          title={title}
          subtitle="Enter parameters, preview the PDF, then download or export Excel"
        />
        <ReportRunner
          href={`/reports/view/${code}`}
          title={title}
          description={meta?.description || ""}
          onClose={() => router.push("/reports")}
        />
      </div>
    </RequirePermission>
  );
}
