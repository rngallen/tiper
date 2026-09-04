"use client";

import { PageHeader } from "@/components/ui";
import { ReportCatalog } from "@/components/ReportCatalog";
import { RequirePermission } from "@/components/RequirePermission";
import { PERMISSIONS } from "@/lib/permissions";
import { REPORT_GROUPS } from "@/lib/reportsCatalog";

export default function ReportsPage() {
  return (
    <RequirePermission codes={[PERMISSIONS.reportsRead]} resource="reports">
      <div>
        <PageHeader
          title="Reports"
          subtitle="Find a report, set the period, preview the PDF, then download or export Excel"
        />
        <ReportCatalog groups={REPORT_GROUPS} />
      </div>
    </RequirePermission>
  );
}
