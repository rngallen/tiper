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
import { formatDate } from "@/lib/format";
import { qty } from "@/lib/master";
import { creatorLabel } from "@/lib/creator";

type Run = {
  id: string;
  documentNumber: string;
  feeCode: string;
  status: string;
  amount: string;
  currencyCode: string;
  quantity?: string;
  rate?: string;
  billingSequence?: number;
  periodStart?: string;
  periodEnd?: string;
  chargeTo?: string;
  source?: string;
  exceptionReason?: string;
  createdBy?: { name?: string; email?: string };
};

export default function BillingPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  async function run(path: string, label: string) {
    setMsg("");
    try {
      await api.post(path, {});
      setMsg(label);
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Engine failed");
    }
  }

  const columns: EnterpriseColumn<Run>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/billing/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "feeCode",
        header: "Fee",
        sortKey: "feeCode",
        defaultVisible: true,
        exportValue: (r) => r.feeCode,
        cell: (r) => r.feeCode,
      },
      {
        id: "periodStart",
        header: "Period",
        sortKey: "periodStart",
        defaultVisible: false,
        exportValue: (r) => `${formatDate(r.periodStart)} – ${formatDate(r.periodEnd)}`,
        cell: (r) =>
          r.periodStart ? `${formatDate(r.periodStart)} – ${formatDate(r.periodEnd)}` : "—",
      },
      {
        id: "amount",
        header: "Amount",
        sortKey: "amount",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => `${r.currencyCode} ${r.amount}`,
        cell: (r) => (
          <span className="tabular-nums">
            {r.currencyCode} {qty(r.amount)}
          </span>
        ),
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
    ],
    [],
  );

  return (
    <EnterpriseTable
        tableId="billing-runs"
        title="Billing runs"
        subtitle="Prepare nth FSF, TBS, and VSF charges from approved prices (Billing › Setups)"
        headerExtra={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => run("/billing/engine/nth", "Nth FSF prepared")}>
              Run nth FSF
            </Button>
            <Button variant="secondary" onClick={() => run("/billing/engine/tbs", "TBS prepared")}>
              Run TBS
            </Button>
            <Button variant="secondary" onClick={() => run("/billing/engine/vsf", "VSF prepared")}>
              Run VSF
            </Button>
          </div>
        }
        lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
        pills={{
          options: DOC_STATUS_PILLS,
          of: (r) => r.status,
          guideTitle: "Billing status",
          allTitle: "All runs",
        }}
        remote={{ path: "/billing/runs", dates: true, reloadKey: tick }}
        emptyMessage="No billing runs"
        columns={columns}
        searchPlaceholder="Document, fee, status"
        rowSearch={(r) => `${r.documentNumber} ${r.feeCode} ${r.status} ${r.currencyCode} ${r.source || ""}`}
        defaultOrderBy="documentNumber"
        defaultSortDirection="DESC"
        dateOf={(r) => r.periodStart}
        exportName="billing_runs"
        inquiryTitle={(r) => r.documentNumber}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb="Billing · Runs · Inquiry"
              title={r.documentNumber}
              pills={<StatusPill tone="navy">{r.status}</StatusPill>}
            />
            <InquirySection title="Charge">
              <ErpField label="Fee" value={r.feeCode} />
              <ErpField label="Sequence" value={r.billingSequence ?? "—"} />
              <ErpField label="Period" value={`${formatDate(r.periodStart)} – ${formatDate(r.periodEnd)}`} />
              <ErpField label="Charge to" value={r.chargeTo || "—"} />
              <ErpField label="Quantity" value={qty(r.quantity)} />
              <ErpField label="Rate" value={qty(r.rate)} />
              <ErpField label="Amount" value={`${r.currencyCode} ${qty(r.amount)}`} />
              <ErpField label="Source" value={r.source || "—"} />
              <ErpField label="Exception" value={r.exceptionReason || "—"} />
            </InquirySection>
          </div>
        )}
    />
  );
}
