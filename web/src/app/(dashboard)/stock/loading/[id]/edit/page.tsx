"use client";

import { use, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { DocumentMeta, ErpField, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { Button, Card, PageHeader } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { stockStatusLabel } from "@/lib/status";
import { LinesCard } from "../../_components/lines";
import { VesselsCard } from "../../_components/vessels";
import type { Approval, Charge, GLR, Party, StockPos } from "../../_components/types";

export default function EditLoadingPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data, loading, error, reload } = useAsync(() => api.get<GLR>(`/orders/loading/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canSubmit = hasPermission(PERMISSIONS.ilrSubmit);
  const draft = data?.status === "draft" || data?.status === "returned";
  const transit = Boolean(data?.stockStatus?.isTransit);

  useEffect(() => {
    if (data?.status === "draft") {
      void api.post(`/orders/loading/${id}/refresh-stock`, {}).then(reload).catch(() => undefined);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const products = useMemo(() => {
    const rows: Party[] = [];
    if (data?.product) rows.push(data.product);
    if (data?.byProduct) rows.push(data.byProduct);
    return rows;
  }, [data]);

  async function printPdf() {
    await api.download("/reports/glr-document.pdf", { id: data?.id || id });
  }

  async function submit() {
    setMsg("");
    try {
      await api.post(`/orders/loading/${id}/submit`, {});
      setMsg("Submitted for approval");
      reload();
    } catch (e) {
      setMsg(formatApiError(e, "Could not submit"));
    }
  }

  if (loading && !data) return <p className="text-sm text-slate-500">Loading…</p>;
  if (error || !data) return <p className="text-sm text-red-600">{error || "Not found"}</p>;

  return (
    <div className="space-y-6">
      <PageHeader
        title={data.documentNumber}
        subtitle="Internal loading request"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void printPdf()}>
              Print PDF
            </Button>
            {canSubmit && draft ? (
              <Button onClick={() => void submit()}>Submit for approval</Button>
            ) : null}
            <Link href="/stock/loading" className="self-center text-sm font-medium text-slate-600">
              Back
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-700">{msg}</p> : null}

      <HeaderCard row={data} />
      <ChargesCard charges={data.charges} />
      <DocumentAttachmentsCard
        path={`/orders/loading/${data.id}`}
        draft={draft}
        lockedCaption="From customer"
        caption="Nomination letters and supporting files. Customer copies stay locked. Drop on the left — the list stays on the right."
      />
      <VesselsCard
        uid={data.id}
        draft={draft}
        ilrDate={String(data.orderDate).slice(0, 10)}
        products={products}
        vessels={data.vessels || []}
        onSaved={reload}
      />
      <StockCard positions={data.stockPositions || []} />
      <LinesCard
        uid={data.id}
        draft={draft}
        transit={transit}
        ilrDate={String(data.orderDate).slice(0, 10)}
        expiryDays={data.iloExpiryDays || 14}
        products={products}
        lines={data.lines || []}
        onSaved={reload}
      />
      <ApprovalsCard rows={data.approvals || []} status={data.status} />
    </div>
  );
}

function HeaderCard({ row }: { row: GLR }) {
  return (
    <Card className="space-y-4 p-5">
      <DocumentMeta crumb="Stock · Loading · ILR">
        <StatusPill tone="navy">{row.status}</StatusPill>
      </DocumentMeta>
      <InquirySection title="Request" columns={3}>
        <ErpField label="Batch number" value={row.batchNumber || "—"} />
        <ErpField label="Order number" value={row.documentNumber} />
        <ErpField label="Request date" value={formatDate(row.orderDate)} />
        <ErpField label="Customer" value={row.customer?.name || row.customer?.code || "—"} />
        <ErpField label="EWURA licence" value={row.customer?.ewuraLicense || "—"} />
        <ErpField label="Product status" value={row.stockStatus ? stockStatusLabel(row.stockStatus) : "—"} />
        <ErpField
          label="Product"
          value={`${partyName(row.product)} · ${formatQty(row.quantity)} L`}
        />
        <ErpField
          label="By-product"
          value={
            row.byProduct ? `${partyName(row.byProduct)} · ${formatQty(row.byProductQuantity)} L` : "—"
          }
        />
        <ErpField label="Description" value={row.description || "—"} />
        <ErpField label="Valid contract" value={row.validContract ? "Yes" : "No"} />
        <ErpField label="Loading order" value={row.loadingOrderAvailable ? "Yes" : "No"} />
      </InquirySection>
    </Card>
  );
}

function ChargesCard({ charges }: { charges?: Charge[] }) {
  const rows = charges || [];
  return (
    <Card className="space-y-3 p-5">
      <h2 className="text-sm font-semibold text-slate-900">Customer outstanding / billing debts</h2>
      <div className="overflow-x-auto rounded-lg border border-slate-200">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs uppercase text-slate-500">
            <tr>
              <th className="px-3 py-2">Charge</th>
              <th className="px-3 py-2">Currency</th>
              <th className="px-3 py-2 text-right">Amount</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td className="px-3 py-6 text-center text-slate-500" colSpan={3}>
                  No outstanding charges on this request
                </td>
              </tr>
            ) : (
              rows.map((r, i) => (
                <tr key={`${r.charge}-${r.currencyCode}-${i}`} className="border-t border-slate-100">
                  <td className="px-3 py-2">{r.charge}</td>
                  <td className="px-3 py-2">{r.currencyCode}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{formatQty(r.amount)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

function StockCard({ positions }: { positions: StockPos[] }) {
  return (
    <Card className="space-y-3 p-5">
      <h2 className="text-sm font-semibold text-slate-900">Customer current stock position (Ltrs)</h2>
      <div className="overflow-x-auto rounded-lg border border-slate-200">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs uppercase text-slate-500">
            <tr>
              <th className="px-3 py-2">Product</th>
              <th className="px-3 py-2 text-right">Total balance</th>
              <th className="px-3 py-2 text-right">Volume under F.Hold</th>
              <th className="px-3 py-2 text-right">Free volume</th>
              <th className="px-3 py-2 text-right">Free after GLR</th>
            </tr>
          </thead>
          <tbody>
            {positions.length === 0 ? (
              <tr>
                <td className="px-3 py-6 text-center text-slate-500" colSpan={5}>
                  No stock snapshot
                </td>
              </tr>
            ) : (
              positions.map((p, i) => (
                <tr key={p.product?.id || i} className="border-t border-slate-100">
                  <td className="px-3 py-2">{partyName(p.product)}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{formatQty(p.totalBalance)}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{formatQty(p.holdQty)}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{formatQty(p.freeQty)}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{formatQty(p.finalQty)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

function ApprovalsCard({ rows, status }: { rows: Approval[]; status: string }) {
  return (
    <Card className="space-y-3 p-5">
      <h2 className="text-sm font-semibold text-slate-900">
        Approvals · <span className="font-normal capitalize">{status}</span>
      </h2>
      <table className="w-full text-sm">
        <thead className="text-left text-xs uppercase text-slate-500">
          <tr>
            <th className="py-1">Approved on</th>
            <th>Approved by</th>
            <th>Title</th>
            <th>Comment</th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td className="py-3 text-slate-500" colSpan={4}>
                Not yet submitted
              </td>
            </tr>
          ) : (
            rows.map((r, i) => (
              <tr key={i} className="border-t border-slate-100">
                <td className="py-1.5">{r.approvedOn}</td>
                <td>{r.approvedBy}</td>
                <td>{r.title || "—"}</td>
                <td>{r.comment || r.actName || "—"}</td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </Card>
  );
}
