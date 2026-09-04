"use client";

import { use, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import {
  DocumentMeta,
  ErpField,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { Button, Card, ErrorBanner, Field, PageHeader, StatusBadge } from "@/components/ui";
import { Modal } from "@/components/Modal";
import { QtyInput } from "@/components/QtyInput";
import { formatDate, formatQty, localISODate } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { fetchLookups } from "@/lib/lookups";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { ParcelTable } from "../../_components/ParcelTable";
import { ReceiptHeaderForm } from "../../_components/ReceiptHeaderForm";
import {
  firstHeaderError,
  receiptHeaderPayload,
  validateReceiptHeader,
  type ReceiptHeaderValues,
} from "../../_lib/header";
import { addQty } from "../../_lib/qty";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { creatorLabel } from "@/lib/creator";
import { useRouter } from "next/navigation";
import type { Receipt } from "../../_lib/types";

function isoDay(value?: string) {
  if (!value) return "";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return String(value).slice(0, 10);
  return localISODate(d);
}

function labelOf(row?: { code?: string; name?: string } | null) {
  const n = partyName(row);
  return n === "—" ? "" : n;
}

function valuesFromReceipt(row: Receipt): ReceiptHeaderValues {
  return {
    date: isoDay(row.date),
    vesselDate: isoDay(row.vesselDate),
    productId: row.product?.id || "",
    productLabel: labelOf(row.product),
    vesselId: row.vessel?.id || "",
    vesselLabel: labelOf(row.vessel),
    supplierId: row.supplier?.id || "",
    supplierLabel: labelOf(row.supplier),
    routeCode: row.routeCode || "",
    tenderCode: row.tenderCode || "",
    procurementMethodCode: row.procurementMethodCode || "",
    isProvision: Boolean(row.isProvision),
    density: row.density && Number(row.density) > 0 ? String(row.density) : "",
    notes: row.notes || "",
    usesTiperPipeline: Boolean(row.usesTiperPipeline),
  };
}

export default function EditReceiptPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const doc = useAsync(() => api.get<Receipt>(`/ic/receipts/${id}`), [id]);
  const catalogs = useAsync(fetchLookups, []);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tankQuantity, setTankQuantity] = useState("");
  const [lineLoss, setLineLoss] = useState("");
  const [form, setForm] = useState<ReceiptHeaderValues | null>(null);
  const [showFieldErrors, setShowFieldErrors] = useState(false);
  const [converting, setConverting] = useState(false);
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission(PERMISSIONS.receiptsUpdate);
  const canSubmit = hasPermission(PERMISSIONS.receiptsSubmit);
  const canConvert = hasPermission(PERMISSIONS.receiptsCreate);
  const row = doc.data;
  const draft = row?.status === "draft";
  const internal = row?.receiptType !== "external";
  const kind = internal ? "internal" : "external";
  const back = internal ? "/stock/receipts" : "/stock/receipts/external";

  useEffect(() => {
    if (row) setForm(valuesFromReceipt(row));
  }, [row]);

  const fieldErrors = useMemo(
    () => (form ? validateReceiptHeader(form, kind) : {}),
    [form, kind],
  );
  const headerIncomplete = Object.keys(fieldErrors).length > 0;
  const outturnL = (row?.details || []).reduce((s, d) => s + Number(d.quantity || 0), 0);
  const tankPreview = Number(tankQuantity || row?.tankQuantity || 0);
  const lossPreview = tankPreview - outturnL;

  async function saveHeader() {
    if (!form) return;
    setError("");
    setShowFieldErrors(true);
    if (headerIncomplete) {
      setError(firstHeaderError(fieldErrors));
      return;
    }
    setBusy(true);
    try {
      await api.patch(`/ic/receipts/${id}`, receiptHeaderPayload(form, kind));
      setMsg("Header saved");
      setShowFieldErrors(false);
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not save header"));
    } finally {
      setBusy(false);
    }
  }

  async function submit() {
    if (!form) return;
    setError("");
    setShowFieldErrors(true);
    if (headerIncomplete) {
      setError(firstHeaderError(fieldErrors));
      return;
    }
    setBusy(true);
    try {
      await api.patch(`/ic/receipts/${id}`, receiptHeaderPayload(form, kind));
      await api.post(`/ic/receipts/${id}/submit`, {});
      setMsg("Submitted for approval");
      setShowFieldErrors(false);
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    } finally {
      setBusy(false);
    }
  }

  async function printPdf() {
    await api.download("/reports/receipt-document.pdf", { id: row?.id || id });
  }

  async function convertToFinal() {
    setError("");
    setBusy(true);
    try {
      const created = await api.post<{ id: string }>(`/ic/receipts/${id}/convert-to-final`, {});
      setConverting(false);
      router.push(`/stock/receipts/${created.id}/edit`);
    } catch (e) {
      setError(formatApiError(e, "Could not convert to final"));
    } finally {
      setBusy(false);
    }
  }

  async function allocate() {
    setError("");
    setBusy(true);
    try {
      await api.post(`/ic/receipts/${id}/allocate-line-loss`, {
        tankQuantity: tankQuantity || row?.tankQuantity || "",
        lineLoss: lineLoss || row?.lineLoss || "",
      });
      setMsg("Line loss allocated to parcels");
      setTankQuantity("");
      setLineLoss("");
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not allocate line loss"));
    } finally {
      setBusy(false);
    }
  }

  if (doc.loading && !row) return <p className="text-sm text-slate-500">Loading…</p>;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Receipt not found"} />
        <Link href={back} className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={row.documentNumber}
        subtitle={internal ? "Internal vessel receipt" : "External vessel receipt"}
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void printPdf()}>
              Print PDF
            </Button>
            {internal && row.isProvision && row.status === "approved" && canConvert ? (
              <Button variant="secondary" onClick={() => setConverting(true)}>
                Convert to final
              </Button>
            ) : null}
            {draft ? (
              <>
                {canUpdate ? (
                  <Button variant="secondary" onClick={() => void saveHeader()} loading={busy}>
                    Save header
                  </Button>
                ) : null}
                {canSubmit ? (
                  <Button onClick={() => void submit()} loading={busy} disabled={headerIncomplete}>
                    Submit for approval
                  </Button>
                ) : null}
              </>
            ) : null}
            <Link href={back} className="self-center text-sm font-medium text-slate-600 hover:text-slate-900">
              Back
            </Link>
          </div>
        }
      />
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge status={row.status} />
        {internal ? (
          <StatusBadge status={row.isProvision ? "provision" : "final"} />
        ) : null}
        <span className="text-sm text-slate-500">{formatDate(row.date)}</span>
        <span className="text-sm text-slate-500">Created by {creatorLabel(row.createdBy)}</span>
      </div>
      {msg ? <p className="text-sm text-slate-700">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      {draft && headerIncomplete ? (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900">
          Header is incomplete — {firstHeaderError(fieldErrors).toLowerCase()} before submit.
        </p>
      ) : null}

      <Card className="space-y-4 p-5">
        <DocumentMeta crumb={internal ? "Stock · Receipts · Internal" : "Stock · Receipts · External"}>
          <StatusPill tone="navy">{row.status}</StatusPill>
          {internal ? (
            <StatusPill tone={row.isProvision ? "amber" : "slate"}>
              {row.isProvision ? "Provision survey" : "Final survey"}
            </StatusPill>
          ) : null}
        </DocumentMeta>
        {draft && form ? (
          <>
            <p className="text-xs text-slate-500">
              Fields marked <span className="font-semibold text-red-600">*</span> are required.
            </p>
            <ReceiptHeaderForm
              kind={kind}
              form={form}
              errors={showFieldErrors ? fieldErrors : {}}
              catalogs={catalogs.data ?? undefined}
              onChange={setForm}
            />
          </>
        ) : (
          <ReadOnlyHeader row={row} internal={internal} />
        )}
      </Card>

      {internal && row.isProvision && row.status === "approved" ? (
        <Card className="space-y-2 border-amber-200 bg-amber-50 p-5">
          <h2 className="text-sm font-semibold text-slate-900">Confirm survey (provision → final)</h2>
          <p className="text-sm text-slate-700">
            This receipt posted unconfirmed (provision) stock. After ullage is confirmed, convert it to a
            final receipt — that creates a new draft with the confirmed quantities. Submit and approve the
            final to drop provision stock and post confirmed stock.
          </p>
          {canConvert ? (
            <Button variant="secondary" onClick={() => setConverting(true)}>
              Convert to final
            </Button>
          ) : null}
        </Card>
      ) : null}

      {internal && draft ? (
        <Card className="space-y-4 p-5">
          <h2 className="text-xs font-bold uppercase tracking-wide text-brand-800">
            Line loss (tank vs outturn)
          </h2>
          <p className="text-sm text-slate-600">
            Outturn is the volume informed. Tank is what landed. Line loss = tank − outturn.
            Allocate to customers; proration parcels absorb the difference when present.
          </p>
          <div className="grid gap-4 sm:grid-cols-3">
            <Field label="Outturn (L)">
              <p className="py-2 text-sm font-semibold tabular-nums">{formatQty(String(outturnL))}</p>
            </Field>
            <Field label="Tank figure (L)">
              <QtyInput value={tankQuantity} onChange={setTankQuantity} placeholder={row.tankQuantity || "0"} />
            </Field>
            <Field label="Line loss (L)" hint="Leave blank to use tank − outturn when allocating">
              <QtyInput value={lineLoss} onChange={setLineLoss} placeholder={row.lineLoss || "0"} />
            </Field>
          </div>
          <p className="text-sm text-slate-600">
            Preview line loss {formatQty(String(lossPreview))} L
            {tankPreview ? ` from tank ${formatQty(String(tankPreview))} L` : ""}.
            Received = outturn + line loss
            {row.details?.[0]
              ? `. Example received ${formatQty(addQty(row.details[0].quantity, row.details[0].lineLoss))} L.`
              : "."}
          </p>
          <Button type="button" variant="secondary" loading={busy} onClick={() => void allocate()}>
            Allocate line loss to customers
          </Button>
        </Card>
      ) : null}

      <DocumentAttachmentsCard
        path={`/ic/receipts/${id}`}
        draft={draft || row.status === "returned"}
        caption="Survey reports, ullage reports, and other files. Drop on the left — the list stays on the right."
      />

      <ParcelTable row={row} catalogs={catalogs.data ?? undefined} onSaved={doc.reload} />

      {converting ? (
        <Modal
          open
          layer="nested"
          title="Convert provision to final"
          onClose={() => setConverting(false)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setConverting(false)}>
                Cancel
              </Button>
              <Button loading={busy} onClick={() => void convertToFinal()}>
                Create final receipt
              </Button>
            </>
          }
        >
          <p className="text-sm text-slate-700">
            Creates a new draft final receipt copied from {row.documentNumber}. You can adjust parcel
            quantities on the new document before submit. Approving the final reverses provision stock and
            posts confirmed stock.
          </p>
        </Modal>
      ) : null}
    </div>
  );
}

function ReadOnlyHeader({ row, internal }: { row: Receipt; internal: boolean }) {
  return (
    <div className="space-y-4">
      <InquirySection title="Vessel and cargo" columns={3}>
        <ErpField label="Receipt date" value={formatDate(row.date)} />
        <ErpField label="Vessel date" value={formatDate(row.vesselDate)} />
        <ErpField label="Discharge route" value={row.routeCode || "—"} />
        <ErpField label="Vessel" value={partyName(row.vessel)} />
        <ErpField label="Product" value={partyName(row.product)} />
        {internal ? <ErpField label="Density (MT / m³)" value={row.density || "—"} /> : null}
        {!internal ? (
          <ErpField
            label="10-inch pipeline"
            value={row.usesTiperPipeline ? "Yes — KOJ fee" : "No"}
          />
        ) : null}
        <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
      </InquirySection>
      {internal ? (
        <InquirySection title="Commercial" columns={3}>
          <ErpField label="Supplier" value={partyName(row.supplier)} />
          <ErpField label="Survey" value={row.isProvision ? "Provision" : "Final"} />
          <ErpField label="Nature of tender" value={row.tenderCode || "—"} />
          <ErpField label="Method of tender" value={row.procurementMethodCode || "—"} />
        </InquirySection>
      ) : null}
      <InquirySection title="Notes" columns={1}>
        <ErpField label="Notes" value={row.notes || "—"} />
      </InquirySection>
    </div>
  );
}
