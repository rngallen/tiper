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
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { creatorLabel } from "@/lib/creator";
import { partyName } from "@/lib/ilr";

type Ref = { id: string; code?: string; name?: string };
type Line = {
  id: string;
  quantity: string;
  cubicMeter?: string;
  metricTonne?: string;
  vesselDate?: string;
  customer?: Ref;
  product?: Ref;
  vessel?: Ref;
  stockStatus?: Ref;
};
type HoldRelease = {
  id: string;
  documentNumber: string;
  releaseDate: string;
  description?: string;
  notes?: string;
  status: string;
  createdBy?: { name?: string; email?: string };
  lines?: Line[];
};

export default function HoldReleaseDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const doc = useAsync(() => api.get<HoldRelease>(`/ic/hold-releases/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.holdSubmit);
  const row = doc.data;

  async function printPdf() {
    await api.download("/reports/hold-release-document.pdf", { id: row?.id || id });
  }

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`/ic/hold-releases/${id}/submit`, {});
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
        <ErrorBanner message={doc.error || "Hold release not found"} />
        <Link href="/stock/financial-hold" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  const draft = row.status === "draft" || row.status === "returned";
  const lines = row.lines || [];

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Financial hold release"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void printPdf()}>
              Print PDF
            </Button>
            {canSubmit && draft ? (
              <Button onClick={() => void submit()} loading={busy}>
                Submit for approval
              </Button>
            ) : null}
            <Link href="/stock/financial-hold" className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Terminal · Financial hold">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Header" columns={3}>
          <ErpField label="Date" value={formatDate(row.releaseDate)} />
          <ErpField label="Description" value={row.description || "—"} />
          <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
          <ErpField label="Notes" value={row.notes || "—"} />
        </InquirySection>
      </Card>
      <DocumentAttachmentsCard
        path={`/ic/hold-releases/${id}`}
        draft={draft}
        caption="Payment advice, invoices, or bank confirmation. Drop on the left — the list stays on the right."
      />
      <Card className="space-y-3 p-5">
        <SectionTable
          title="Parcels released"
          columns={[
            { label: "Customer" },
            { label: "Product" },
            { label: "Vessel" },
            { label: "Vessel date" },
            { label: "Status" },
            { label: "Litres", align: "right" },
            { label: "m³", align: "right" },
            { label: "MT", align: "right" },
          ]}
          empty="No parcels"
        >
          {lines.map((l) => (
            <SectionRow key={l.id}>
              <td className="font-semibold">{partyName(l.customer)}</td>
              <td>{partyName(l.product)}</td>
              <td>{partyName(l.vessel)}</td>
              <td>{formatDate(l.vesselDate)}</td>
              <td>{l.stockStatus?.name || "—"}</td>
              <td className="text-right tabular-nums">{formatQty(l.quantity)}</td>
              <td className="text-right tabular-nums">{formatQty(l.cubicMeter)}</td>
              <td className="text-right tabular-nums">{formatQty(l.metricTonne)}</td>
            </SectionRow>
          ))}
        </SectionTable>
      </Card>
    </div>
  );
}
