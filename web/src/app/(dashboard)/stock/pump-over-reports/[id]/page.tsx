"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import {
  DocumentMeta,
  ErpField,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { Button, Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import type { PumpOverReport } from "@/lib/pumpOver";

export default function PumpOverReportDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const doc = useAsync(() => api.get<PumpOverReport>(`/orders/pump-over-reports/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.pumpReportSubmit);
  const row = doc.data;

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`/orders/pump-over-reports/${id}/submit`, {});
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
        <ErrorBanner message={doc.error || "Pump-over report not found"} />
        <Link href="/stock/pump-over-reports" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  const draft = row.status === "draft" || row.status === "returned";

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Pump-over execution report"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void api.download("/reports/pump-over-report.pdf", { id })}>
              Print PDF
            </Button>
            <Link
              href="/stock/pump-over-reports"
              className="self-center text-sm font-medium text-slate-600 hover:text-slate-900"
            >
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Pump-over · Report">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Execution" columns={3}>
          <ErpField label="Date" value={formatDate(row.reportDate)} />
          <ErpField label="Request" value={row.request?.documentNumber || "—"} />
          <ErpField label="Customer" value={partyName(row.request?.customer)} />
          <ErpField label="Product" value={partyName(row.request?.product)} />
          <ErpField label="Delivered (m³)" value={formatQty(row.actualDelivered)} />
          <ErpField label="Received (m³)" value={formatQty(row.actualReceived)} />
          <ErpField label="Variance (m³)" value={formatQty(row.variance)} />
        </InquirySection>
        <div className="flex flex-wrap items-center gap-3">
          {canSubmit && draft ? (
            <Button onClick={() => void submit()} loading={busy}>
              Submit for approval
            </Button>
          ) : null}
          {row.request?.id ? (
            <Link href={`/stock/pump-over/${row.request.id}`} className="text-sm font-semibold text-brand-800">
              Open request {row.request.documentNumber}
            </Link>
          ) : null}
        </div>
      </Card>
      <DocumentAttachmentsCard
        path={`/orders/pump-over-reports/${id}`}
        draft={draft}
        caption="Meter tickets and execution evidence. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
