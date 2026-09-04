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
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { partyName } from "@/lib/ilr";

type Ref = { id: string; code?: string; name?: string };
type ITT = {
  id: string;
  documentNumber: string;
  transferDate: string;
  vesselDate?: string;
  quantity: string;
  status: string;
  fromCustomer?: Ref;
  toCustomer?: Ref;
  product?: Ref;
  vessel?: Ref;
};

export default function ITTDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const doc = useAsync(() => api.get<ITT>(`/ic/itt/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.ittSubmit);
  const row = doc.data;

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`/ic/itt/${id}/submit`, {});
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
        <ErrorBanner message={doc.error || "In-tank transfer not found"} />
        <Link href="/stock/itt" className="text-sm font-medium text-brand-800">
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
        subtitle="In-tank transfer — stock posts on approval (sender out, receiver in); the original receipt line is not rewritten"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void api.download("/reports/itt-document.pdf", { id })}>
              Print PDF
            </Button>
            <Link href="/stock/itt" className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Terminal · ITT">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Transfer" columns={3}>
          <ErpField label="Date" value={formatDate(row.transferDate)} />
          <ErpField label="From" value={partyName(row.fromCustomer)} />
          <ErpField label="To" value={partyName(row.toCustomer)} />
          <ErpField label="Product" value={partyName(row.product)} />
          <ErpField label="Vessel" value={partyName(row.vessel)} />
          <ErpField label="Vessel date" value={formatDate(row.vesselDate)} />
          <ErpField label="Quantity (m³)" value={formatQty(row.quantity)} />
        </InquirySection>
        {canSubmit && draft ? (
          <Button onClick={() => void submit()} loading={busy}>
            Submit for approval
          </Button>
        ) : null}
      </Card>
      <DocumentAttachmentsCard
        path={`/ic/itt/${id}`}
        draft={draft}
        lockedCaption="From customer"
        caption="Customer letters and transfer instructions. Customer copies stay locked. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
