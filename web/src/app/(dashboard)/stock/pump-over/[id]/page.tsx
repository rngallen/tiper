"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import {
  DocumentMeta,
  ErpField,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { stockStatusLabel } from "@/lib/status";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import type { PumpOverRequest } from "@/lib/pumpOver";

export default function PumpOverDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const doc = useAsync(() => api.get<PumpOverRequest>(`/orders/pump-over/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.pumpOverSubmit);
  const canReport = hasPermission(PERMISSIONS.pumpReportCreate);
  const row = doc.data;

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`/orders/pump-over/${id}/submit`, {});
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
        <ErrorBanner message={doc.error || "Pump-over request not found"} />
        <Link href="/stock/pump-over" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  const vessels = row.vessels || [];
  const draft = row.status === "draft" || row.status === "returned";

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Pump-over request"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void api.download("/reports/pump-over-document.pdf", { id })}>
              Print PDF
            </Button>
            <Link href="/stock/pump-over" className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Pump-over">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Request" columns={3}>
          <ErpField label="Date" value={formatDate(row.orderDate)} />
          <ErpField label="Customer" value={partyName(row.customer)} />
          <ErpField label="Product" value={partyName(row.product)} />
          <ErpField label="Receiving depot" value={partyName(row.depot)} />
          <ErpField label="Quantity (m³)" value={formatQty(row.quantity)} />
          <ErpField label="Customer order" value={row.customerOrderNumber || "—"} />
        </InquirySection>
        {vessels.length > 0 ? (
          <SectionTable
            title="Vessels"
            columns={[
              { label: "Vessel" },
              { label: "Date" },
              { label: "Status" },
              { label: "Qty (m³)", align: "right" },
            ]}
          >
            {vessels.map((v, i) => (
              <SectionRow key={`${v.vessel?.id || i}-${v.vesselDate}`}>
                <td className="font-semibold">{v.vessel?.code || v.vessel?.name || `Vessel ${i + 1}`}</td>
                <td>{formatDate(v.vesselDate)}</td>
                <td>{v.stockStatus ? stockStatusLabel(v.stockStatus) : "—"}</td>
                <td className="text-right tabular-nums">{formatQty(v.quantity)}</td>
              </SectionRow>
            ))}
          </SectionTable>
        ) : null}
        <div className="flex flex-wrap gap-2">
          {canSubmit && row.status === "draft" ? (
            <Button onClick={() => void submit()} loading={busy}>
              Submit for approval
            </Button>
          ) : null}
          {canReport && row.status === "approved" ? (
            <Button onClick={() => router.push(`/stock/pump-over-reports/new?request=${row.id}`)}>
              New execution report
            </Button>
          ) : null}
        </div>
      </Card>
      <DocumentAttachmentsCard
        path={`/orders/pump-over/${id}`}
        draft={draft}
        lockedCaption="From customer"
        caption="Nomination letters and pipeline instructions. Customer copies stay locked. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
