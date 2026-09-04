"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { DocumentMeta, ErpField, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Button, Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate } from "@/lib/format";
import { qty } from "@/lib/master";
import { feeEditable } from "@/lib/billing";

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
};

export default function BillingRunPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/runs/${id}`;
  const doc = useAsync(() => api.get<Run>(path), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const row = doc.data;
  const draft = feeEditable(row?.status);

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`${path}/submit`, {});
      setMsg("Submitted for approval");
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    } finally {
      setBusy(false);
    }
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Billing run not found"} />
        <Link href="/billing" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Billing run"
        actions={
          <Link href="/billing" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Billing · Runs">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Charge" columns={3}>
          <ErpField label="Fee" value={row.feeCode} />
          <ErpField label="Sequence" value={row.billingSequence ?? "—"} />
          <ErpField label="Period" value={`${formatDate(row.periodStart)} – ${formatDate(row.periodEnd)}`} />
          <ErpField label="Charge to" value={row.chargeTo || "—"} />
          <ErpField label="Quantity" value={qty(row.quantity)} />
          <ErpField label="Rate" value={qty(row.rate)} />
          <ErpField label="Amount" value={`${row.currencyCode} ${qty(row.amount)}`} />
          <ErpField label="Source" value={row.source || "—"} />
          <ErpField label="Exception" value={row.exceptionReason || "—"} />
        </InquirySection>
        {draft ? (
          <Button onClick={() => void submit()} loading={busy}>
            Submit for approval
          </Button>
        ) : null}
      </Card>
      <DocumentAttachmentsCard
        path={path}
        draft={draft}
        caption="Working papers and Sage backup. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
