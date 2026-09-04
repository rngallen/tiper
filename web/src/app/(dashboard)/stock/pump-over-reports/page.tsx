"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { Button, StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate, formatQty } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { creatorLabel } from "@/lib/creator";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import type { PumpOverReport } from "@/lib/pumpOver";

export default function PumpOverReportsPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.pumpReportCreate);
  const canSubmit = hasPermission(PERMISSIONS.pumpReportSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/orders/pump-over-reports/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<PumpOverReport>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/stock/pump-over-reports/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "reportDate",
        header: "Date",
        sortKey: "reportDate",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.reportDate),
        cell: (r) => formatDate(r.reportDate),
      },
      {
        id: "request",
        header: "Request",
        defaultVisible: true,
        exportValue: (r) => r.request?.documentNumber,
        cell: (r) => r.request?.documentNumber || "—",
      },
      {
        id: "customer",
        header: "Customer",
        defaultVisible: true,
        exportValue: (r) => r.request?.customer?.name || r.request?.customer?.code,
        cell: (r) => r.request?.customer?.name || r.request?.customer?.code || "—",
      },
      {
        id: "actualDelivered",
        header: "Delivered",
        sortKey: "actualDelivered",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.actualDelivered,
        cell: (r) => <span className="tabular-nums">{formatQty(r.actualDelivered)}</span>,
      },
      {
        id: "actualReceived",
        header: "Received",
        sortKey: "actualReceived",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.actualReceived,
        cell: (r) => <span className="tabular-nums">{formatQty(r.actualReceived)}</span>,
      },
      {
        id: "status",
        header: "Status",
        sortKey: "status",
        defaultVisible: true,
        exportValue: (r) => r.status,
        cell: (r) => <StatusBadge status={r.status} />,
      },
      {
        id: "createdBy",
        header: "Created by",
        defaultVisible: true,
        exportValue: (r) => creatorLabel(r.createdBy),
        cell: (r) => creatorLabel(r.createdBy),
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-2">
            <Link
              href={`/stock/pump-over-reports/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canSubmit && r.status === "draft" ? (
              <Button
                variant="secondary"
                className="!px-3 !py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  void submit(r.id);
                }}
              >
                Submit
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [canSubmit],
  );

  return (
    <EnterpriseTable
      tableId="orders-pump-over-reports"
      title="Pump-over reports"
      subtitle="Executed quantities after a pump-over request is approved — own status, own approval"
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/pump-over-reports/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New report
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Report status",
        allTitle: "All reports",
      }}
      remote={{ path: "/orders/pump-over-reports", dates: true, reloadKey: tick }}
      emptyMessage="No execution reports"
      columns={columns}
      searchPlaceholder="Report, request, customer"
      defaultOrderBy="reportDate"
      defaultSortDirection="DESC"
      exportName="pump_over_reports"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Pump-over · Report · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Execution">
            <ErpField label="Date" value={formatDate(r.reportDate)} />
            <ErpField label="Request" value={r.request?.documentNumber || "—"} />
            <ErpField label="Customer" value={partyName(r.request?.customer)} />
            <ErpField label="Delivered (m³)" value={formatQty(r.actualDelivered)} />
            <ErpField label="Received (m³)" value={formatQty(r.actualReceived)} />
          </InquirySection>
          <Link
            href={`/stock/pump-over-reports/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
