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
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { partyName } from "@/lib/ilr";
import { ChangeOfServiceForm, type CosPayload } from "../_components/ChangeOfServiceForm";

type Ref = { id: string; code?: string; name?: string };
type Cos = {
  id: string;
  documentNumber: string;
  effectiveDate: string;
  vesselDate?: string;
  status: string;
  parcelId?: string;
  fromCollection?: string;
  toCollection?: string;
  notes?: string;
  customer?: Ref;
  vessel?: Ref;
  product?: Ref;
};

export default function ChangeOfServiceDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/change-of-service/${id}`;
  const doc = useAsync(() => api.get<Cos>(path), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState(false);
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.changeOfServiceSubmit);
  const canUpdate = hasPermission(PERMISSIONS.changeOfServiceUpdate);
  const row = doc.data;
  const draft = row?.status === "draft" || row?.status === "returned";

  async function submit() {
    setError("");
    setBusy(true);
    try {
      await api.post(`${path}/submit`, {});
      setMsg("Submitted for approval");
      setEditing(false);
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    } finally {
      setBusy(false);
    }
  }

  async function save(body: CosPayload) {
    setError("");
    setBusy(true);
    try {
      await api.put(path, body);
      setMsg("Saved");
      setEditing(false);
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not update change of service"));
    } finally {
      setBusy(false);
    }
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Change of service not found"} />
        <Link href="/billing/change-of-service" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle="Change of service — delivery method on one vessel parcel"
        actions={
          <div className="flex flex-wrap gap-2">
            {canUpdate && draft && !editing ? (
              <Button variant="secondary" onClick={() => setEditing(true)}>
                Edit
              </Button>
            ) : null}
            <Link href="/billing/change-of-service" className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Billing · Transactions · Change of service">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        {editing && draft ? (
          <ChangeOfServiceForm
            key={row.id}
            initial={{
              effectiveDate: row.effectiveDate,
              customerId: row.customer?.id,
              customerLabel: partyName(row.customer),
              parcelId: row.parcelId,
              vesselId: row.vessel?.id,
              vesselDate: row.vesselDate,
              fromCollection: row.fromCollection,
              toCollection: row.toCollection,
              notes: row.notes,
            }}
            saving={busy}
            submitLabel="Save"
            onCancel={() => setEditing(false)}
            onSubmit={(body) => void save(body)}
            onError={setError}
          />
        ) : (
          <>
            <InquirySection title="Parcel" columns={3}>
              <ErpField label="Effective" value={formatDate(row.effectiveDate)} />
              <ErpField label="Customer" value={partyName(row.customer)} />
              <ErpField label="Vessel" value={partyName(row.vessel)} />
              <ErpField label="Vessel date" value={formatDate(row.vesselDate)} />
              <ErpField label="Product" value={partyName(row.product)} />
              <ErpField label="Delivery" value={`${row.fromCollection || "—"} → ${row.toCollection || "—"}`} />
              <ErpField label="Notes" value={row.notes || "—"} />
            </InquirySection>
            {canSubmit && draft ? (
              <Button onClick={() => void submit()} loading={busy} disabled={!row.parcelId || !row.toCollection}>
                Submit for approval
              </Button>
            ) : null}
          </>
        )}
      </Card>
      <DocumentAttachmentsCard
        path={path}
        draft={Boolean(draft)}
        caption="Supporting letters or customer instruction. Drop on the left — the list stays on the right."
      />
    </div>
  );
}
