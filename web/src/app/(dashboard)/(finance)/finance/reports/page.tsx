"use client";

import { PageHeader } from "@/components/ui";
import { ReportCatalog } from "@/components/ReportCatalog";
import { RequirePermission } from "@/components/RequirePermission";
import { PERMISSIONS } from "@/lib/permissions";
import { FINANCE_REPORT_GROUPS } from "@/lib/reportsCatalog";

export default function FinanceReportsPage() {
  return (
    <RequirePermission codes={[PERMISSIONS.reportsRead]} resource="reports">
      <div>
        <PageHeader
          title="Billing reports"
          subtitle="Billing runs, trade class, product, and exception reports"
        />
        <ReportCatalog groups={FINANCE_REPORT_GROUPS} />
      </div>
    </RequirePermission>
  );
}
