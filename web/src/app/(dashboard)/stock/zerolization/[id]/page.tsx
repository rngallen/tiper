"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { DocumentMeta, ErpField, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Button, Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { partyName } from "@/lib/ilr";

type Ref = { id: string; code?: string; name?: string };
type Zerol = {
  id: string;
  documentNumber: string;
  transferDate: string;
  quantity: string;
  status: string;
  fromVesselDate?: string;
  toVesselDate?: string;
  customer?: Ref;
  product?: Ref;
  fromVessel?: Ref;
  toVessel?: Ref;
};

export default function ZerolizationDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/ic/zerolization/${id}`;
  const doc = useAsync(() => api.get<Zerol>(path), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.zerolizationSubmit);
  const row = doc.data;
  const draft = row?.status === "draft" || row?.status === "returned";

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
        <ErrorBanner message={doc.error || "Zerolization not found"} />
        <Link href="/stock/zerolization" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Zerolization"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void api.download("/reports/zerolization-document.pdf", { id })}>
              Print PDF
            </Button>
            <Link href="/stock/zerolization" className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Terminal · Zerolization">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Transfer" columns={3}>
          <ErpField label="Date" value={formatDate(row.transferDate)} />
          <ErpField label="Customer" value={partyName(row.customer)} />
          <ErpField label="Product" value={partyName(row.product)} />
          <ErpField label="From vessel" value={`${partyName(row.fromVessel)} · ${formatDate(row.fromVesselDate)}`} />
          <ErpField label="To vessel" value={`${partyName(row.toVessel)} · ${formatDate(row.toVesselDate)}`} />
          <ErpField label="Quantity" value={formatQty(row.quantity)} />
        </InquirySection>
        {canSubmit && draft ? (
          <Button onClick={() => void submit()} loading={busy}>
            Submit for approval
          </Button>
        ) : null}
      </Card>
      <DocumentAttachmentsCard
        path={path}
        draft={Boolean(draft)}
        caption="Stock notes for this vessel consolidation. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
