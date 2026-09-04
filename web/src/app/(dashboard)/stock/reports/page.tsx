"use client";

import { PageHeader } from "@/components/ui";
import { ReportCatalog } from "@/components/ReportCatalog";
import { RequirePermission } from "@/components/RequirePermission";
import { PERMISSIONS } from "@/lib/permissions";
import { STOCK_REPORT_GROUPS } from "@/lib/reportsCatalog";

export default function StockReportsPage() {
  return (
    <RequirePermission codes={[PERMISSIONS.reportsRead]} resource="reports">
      <div>
        <PageHeader
          title="Stock reports"
          subtitle="Stock, reception, and gantry reports — search, preview PDF, then export"
        />
        <ReportCatalog groups={STOCK_REPORT_GROUPS} />
      </div>
    </RequirePermission>
  );
}
