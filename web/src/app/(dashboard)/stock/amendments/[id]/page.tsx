"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { DocumentMeta, ErpField, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Card, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";

type Amend = {
  id: string;
  documentNumber: string;
  kind: string;
  status: string;
  requestedQty: string;
  destination?: string;
  notes?: string;
  expirationDate?: string;
  truckPlate?: string;
  ilo?: { documentNumber?: string };
};

export default function AmendmentDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/orders/amendments/${id}`;
  const doc = useAsync(() => api.get<Amend>(path), [id]);
  const row = doc.data;
  const draft = row?.status === "draft" || row?.status === "returned";

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Amendment not found"} />
        <Link href="/stock/amendments" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Gantry amendment"
        actions={
          <Link href="/stock/amendments" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Gantry · Amendments">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Amendment" columns={3}>
          <ErpField label="Kind" value={row.kind} />
          <ErpField label="GLO" value={row.ilo?.documentNumber || "—"} />
          <ErpField label="Quantity" value={formatQty(row.requestedQty)} />
          <ErpField label="Expiry" value={formatDate(row.expirationDate)} />
          <ErpField label="Truck" value={row.truckPlate || "—"} />
          <ErpField label="Destination" value={row.destination || "—"} />
          <ErpField label="Notes" value={row.notes || "—"} />
        </InquirySection>
      </Card>
      <DocumentAttachmentsCard
        path={path}
        draft={Boolean(draft)}
        caption="Customer letters and supporting files. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
